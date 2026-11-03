// Package watch is the fsnotify watch loop (spec 02-code-index 007): a
// trigger, not a gate. It watches the project source tree, debounces bursts
// (2 s default, clamped 100 ms–60 s), collapses a burst of N events into ONE
// incremental re-index (the onSync seam), runs a metadata-only fingerprint
// re-check (size+mtime vs the files table + the extractor stamp — a no-op when
// unchanged), isolates re-index failures (a failure never stops the watcher),
// and honors the M10 absent-on-disk = delete contract (an absent path is a
// Store.DeleteFile, not an error). CI-INV-4: a query during a re-index sees the
// last committed index + degraded: stale_index, never blocks.
//
// The re-check + re-index are dialect-agnostic (INV-2): the watcher reads the
// files table through the Store's RawQuery seam and re-indexes through the
// onSync callback (the TICKET-01/03 Index driver), so it never touches the
// embedder leg. The watcher holds NO lock the reads wait on (CI-INV-4) — the
// Store's WAL is the concurrency; the watcher just issues normal Store reads
// (the re-check) and writes (the re-index, via onSync).
package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// DefaultDebounce is the default burst-collapse window (spec 007 §7). A burst
// of rapid edits (create/modify/delete) collapses into a single incremental
// re-index after this window of inactivity. Tunable per Watcher (clamped to
// [MinDebounce, MaxDebounce]).
const DefaultDebounce = 2000 * time.Millisecond

// MinDebounce / MaxDebounce clamp a configured debounce window so a
// misconfigured value (0 or absurd) cannot make the watcher useless.
const (
	MinDebounce = 100 * time.Millisecond
	MaxDebounce = 60 * time.Second
)

// clampDebounce clamps d into [MinDebounce, MaxDebounce]. Zero/negative falls
// to the default.
func clampDebounce(d time.Duration) time.Duration {
	if d <= 0 {
		return DefaultDebounce
	}
	if d < MinDebounce {
		return MinDebounce
	}
	if d > MaxDebounce {
		return MaxDebounce
	}
	return d
}

// WatchConfig configures the watch loop.
type WatchConfig struct {
	// Debounce is the burst-collapse window (clamped to [MinDebounce,
	// MaxDebounce]; <=0 uses DefaultDebounce).
	Debounced time.Duration
	// Include/Exclude are the source filters (the 005 include/exclude globs).
	// Non-source files (not matching Include, or matching Exclude) are ignored.
	Include []string
	Exclude []string
	// Stamp is the extractor stamp the fingerprint re-check compares against
	// the stored index_freshness.extractor_stamp. A stamp change forces a
	// re-index even if size+mtime match (the extractor itself changed). When
	// empty, the re-check treats a stamp change as a no-op (it cannot compare).
	Stamp string
	// Store is the Store the re-check reads the files table from + the
	// freshness check reads index_freshness from (the CI-INV-4 freshness
	// check). When nil, the re-check cannot consult the files table (every
	// pending path is treated as changed → a re-index fires).
	Store store.Store
}

// OnSyncFunc is the re-index seam the watcher fires on a debounced burst. The
// paths are the deduped repo-relative changed paths. A nil OnSync means the
// watcher only tracks pending files (no re-index). A non-nil OnSync that fails
// is isolated (logged + the watcher keeps running — failure isolation, spec
// 007 §3).
type OnSyncFunc func(paths []string)

// Watcher is the autosync watcher: it watches the index root for
// create/modify/delete of source files, debounces bursts, runs the
// metadata-only fingerprint re-check, and triggers a single incremental
// re-index (the onSync seam). It is a trigger, not a gate (CI-INV-4).
type Watcher struct {
	root   string
	cfg    WatchConfig
	onSync OnSyncFunc

	// the fsnotify source (nil in the deterministic tests — they drive the
	// drain via the seams).
	watcher  *fsnotify.Watcher
	stop     chan struct{}
	stopOnce sync.Once

	mu      sync.Mutex
	pending map[string]struct{}
	stopped bool
	lastErr error
	// reindexCount counts the re-indexes fired (the ReindexCalled seam).
	reindexCount int
	// reindexInFlight is true while a re-index is running (the CI-INV-4
	// freshness check reports degraded: stale_index when it is).
	reindexInFlight bool
}

// NewWatcher creates a watcher for root, with the re-index seam onSync (nil =
// track pending only, no re-index). It does not start the fsnotify loop until
// Start.
func NewWatcher(root string, cfg WatchConfig, onSync OnSyncFunc) *Watcher {
	return &Watcher{
		root:    root,
		cfg:     cfg,
		onSync:  onSync,
		stop:    make(chan struct{}),
		pending: map[string]struct{}{},
	}
}

// Debounce returns the effective (clamped) debounce window.
func (w *Watcher) Debounce() time.Duration {
	return clampDebounce(w.cfg.Debounced)
}

// IsSource reports whether a repo-relative path is a source file per the
// include/exclude filters. Directory exclusion mirrors the Index walk's
// shouldSkipDir so a change inside an excluded directory is ignored.
func (w *Watcher) IsSource(rel string) bool {
	rel = filepath.ToSlash(rel)
	for _, pattern := range w.cfg.Exclude {
		if matchesGlob(pattern, rel) {
			return false
		}
	}
	if len(w.cfg.Include) > 0 && !matchesAny(w.cfg.Include, rel) {
		return false
	}
	return true
}

// Start begins the fsnotify loop. It returns an error if the watcher could not
// be created (e.g. the root is missing). OnSync runs on the background loop
// goroutine so Start returns promptly.
func (w *Watcher) Start(ctx context.Context) error {
	fn, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create fsnotify watcher: %w", err)
	}
	if err := fn.Add(w.root); err != nil {
		fn.Close()
		return fmt.Errorf("watch root: %w", err)
	}
	w.watcher = fn
	go w.loop(ctx)
	return nil
}

// Stop closes the watcher and stops the debounce loop. It is idempotent.
func (w *Watcher) Stop() {
	w.stopOnce.Do(func() {
		close(w.stop)
		if w.watcher != nil {
			w.watcher.Close()
		}
		w.mu.Lock()
		w.stopped = true
		w.mu.Unlock()
	})
}

// Running reports whether the watcher loop is alive (a re-index failure does
// not stop it — failure isolation).
func (w *Watcher) Running() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return !w.stopped
}

// loop consumes fsnotify events, marks source files pending, and debounces a
// burst into a single re-index (the v1 loop, lifted + the re-check folded in).
func (w *Watcher) loop(ctx context.Context) {
	debounce := w.Debounce()
	timer := time.NewTimer(debounce)
	timer.Stop()
	fire := false
	for {
		select {
		case <-w.stop:
			timer.Stop()
			return
		case <-ctx.Done():
			timer.Stop()
			return
		case evt, ok := <-w.watcher.Events:
			if !ok {
				timer.Stop()
				return
			}
			if !w.shouldFire(evt) {
				continue
			}
			w.markPending(evt)
			if !fire {
				// First event in the burst: arm the debounce timer.
				timer.Reset(debounce)
				fire = true
			}
		case _, ok := <-w.watcher.Errors:
			if !ok {
				timer.Stop()
				return
			}
			// A fsnotify error: stop the loop (the fingerprint gate is the
			// backstop — the watcher is a trigger, not a gate).
			w.mu.Lock()
			w.stopped = true
			w.mu.Unlock()
			timer.Stop()
			return
		case <-timer.C:
			fire = false
			w.drain()
		}
	}
}

// shouldFire reports whether an event is a create/modify/delete of a source
// file (non-source, rename, and chown-only events are ignored).
func (w *Watcher) shouldFire(evt fsnotify.Event) bool {
	if evt.Op&fsnotify.Chmod != 0 && evt.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
		return false
	}
	rel, err := filepath.Rel(w.root, evt.Name)
	if err != nil || rel == ".." {
		return false
	}
	return w.IsSource(rel)
}

// markPending records the event's file as pending.
func (w *Watcher) markPending(evt fsnotify.Event) {
	w.mu.Lock()
	defer w.mu.Unlock()
	rel, _ := filepath.Rel(w.root, evt.Name)
	if rel == "." {
		rel = ""
	}
	if rel != "" && w.IsSource(rel) {
		w.pending[filepath.ToSlash(rel)] = struct{}{}
	}
}

// drain is the debounce-fire path: it drains the pending set, runs the
// metadata-only fingerprint re-check, and fires exactly ONE onSync for the
// actually-changed set (a burst collapses to one re-index). An empty changed
// set (a no-op re-check) fires nothing.
func (w *Watcher) drain() {
	paths := w.drainPending()
	if len(paths) == 0 {
		return
	}
	// The fingerprint re-check (metadata-only): filter to the actually-changed
	// paths. A no-op re-check (size+mtime+stamp unchanged) → empty → no re-index.
	changed := w.recheck(paths)
	if len(changed) == 0 {
		return
	}
	w.fireReindex(changed)
}

// drainPending returns and clears the pending set (sorted for determinism).
func (w *Watcher) drainPending() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]string, 0, len(w.pending))
	for p := range w.pending {
		out = append(out, p)
	}
	sort.Strings(out)
	w.pending = map[string]struct{}{}
	return out
}

// fireReindex runs the onSync seam for the changed paths, counting the
// re-index (ReindexCalled) and tracking in-flight state (CI-INV-4). A failure
// is isolated: a re-index that errors (or panics) is recorded (LastErrForTest)
// but the watcher keeps running — the next event still drains (failure
// isolation, spec 007 §3).
//
// The onSync seam is a value callback (no error return) — the production
// caller (the TICKET-07 code_watch tool / CLI) wraps the Index driver and logs
// its own error. For the watcher's own failure-isolation accounting, a re-index
// failure is signaled by the onSync setting the test error (setErrForTest); a
// panic in the seam is recovered + recorded. LastErr is only set (never
// cleared) here — the onSync clears it on the next success (the test seam
// contract).
func (w *Watcher) fireReindex(paths []string) {
	if w.onSync == nil {
		// No re-index seam: count the re-index (the deterministic tests assert
		// the burst-collapse + the no-op by ReindexCalled) but run nothing.
		w.mu.Lock()
		w.reindexCount++
		w.mu.Unlock()
		return
	}
	w.mu.Lock()
	w.reindexInFlight = true
	w.reindexCount++
	w.mu.Unlock()
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("reindex panic: %v", r)
			}
		}()
		w.onSync(paths)
	}()
	if err != nil {
		w.setErrForTest(err)
	}
	w.mu.Lock()
	w.reindexInFlight = false
	w.mu.Unlock()
}

// recheck runs the metadata-only stat-walk fingerprint re-check (spec 007 §2,
// 001 §4): for each pending path, stat it; compare size+mtime against the
// files table + the extractor stamp. The NO-OP condition (no re-index): the
// path is present on disk AND its size+mtime match the stored files row AND
// the extractor stamp is unchanged. A stamp change forces a re-index even if
// size+mtime match. An absent path (M10) is kept for the re-index (the
// driver's absent-on-disk = delete handles it — the re-check does NOT delete;
// it passes the path through so the re-index sees the absence).
//
// ponytail: the re-check is per-pending-path (not a full tree walk) — the
// watch only re-checks the paths its events touched. A git checkout that
// touched many files fires many events → many pending paths → the re-check
// covers them all. A full-tree drift walk (the pull-at-query backstop) is a
// separate concern (the query path's fingerprint gate), not the watch's.
func (w *Watcher) recheck(paths []string) []string {
	st := w.cfg.Store
	if st == nil {
		// No Store: the re-check cannot consult the files table — treat every
		// pending path as changed (a re-index fires). This is the conservative
		// (safe) default: a false-positive re-index is cheap; a missed one is
		// not.
		return paths
	}
	// Read the stored stamp (index_freshness.extractor_stamp) once.
	storedStamp := w.storedStamp(st)
	stampChanged := w.cfg.Stamp != "" && storedStamp != "" && w.cfg.Stamp != storedStamp

	out := []string{}
	for _, p := range paths {
		full := filepath.Join(w.root, p)
		info, err := os.Stat(full)
		if err != nil {
			// Absent on disk (M10): the re-check does NOT delete — it passes
			// the path through so the re-index (the Index driver) sees the
			// absence and runs Store.DeleteFile (absent-on-disk = delete). The
			// re-check is metadata-only; the delete is the re-index's job.
			out = append(out, p)
			continue
		}
		// Compare size+mtime against the stored files row.
		row, err := st.FileByPath(context.Background(), p)
		if err != nil {
			// Not in the index (a new file) → changed.
			out = append(out, p)
			continue
		}
		// The NO-OP condition: size+mtime unchanged AND stamp unchanged.
		if info.Size() == row.Size && info.ModTime().UnixNano() == row.MtimeNS && !stampChanged {
			continue // no-op
		}
		out = append(out, p)
	}
	return out
}

// storedStamp reads index_freshness.extractor_stamp (the last successful
// index's stamp) via the RawQuery seam. "" when there is no freshness record
// or the store lacks the seam.
func (w *Watcher) storedStamp(st store.Store) string {
	raw, ok := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return ""
	}
	rows, err := raw.RawQuery(`SELECT extractor_stamp FROM index_freshness LIMIT 1`)
	if err != nil || len(rows) == 0 || len(rows[0]) == 0 {
		return ""
	}
	s, _ := rows[0][0].(string)
	return s
}

// ── test seams (the deterministic tests + the trigger-not-gate property) ────

// EmitForTest injects events without fsnotify (the deterministic tests). Each
// arg is a repo-relative path marked pending (source-filtered).
func (w *Watcher) EmitForTest(paths ...string) {
	for _, p := range paths {
		w.MarkPendingForTest(p)
	}
}

// MarkPendingForTest marks a path pending (the re-check input) without an
// fsnotify event.
func (w *Watcher) MarkPendingForTest(p string) {
	p = filepath.ToSlash(p)
	if !w.IsSource(p) {
		return
	}
	w.mu.Lock()
	w.pending[p] = struct{}{}
	w.mu.Unlock()
}

// DrainForTest drives the debounce-fire path (drain) synchronously — the
// deterministic tests use it instead of waiting on the real timer. It runs the
// re-check + the burst-collapse (exactly ONE onSync for the changed set).
func (w *Watcher) DrainForTest() {
	w.drain()
}

// ReindexCalled returns the re-index count (the deterministic tests assert the
// burst-collapse + the no-op by it).
func (w *Watcher) ReindexCalled() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.reindexCount
}

// LastErrForTest returns the last re-index error (failure isolation).
func (w *Watcher) LastErrForTest() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastErr
}

// setErrForTest records a re-index error (the failure-isolation test's onSync
// signals a failure through it). It is a test seam: the production onSync
// (the Index driver) returns its own error, which the caller logs.
func (w *Watcher) setErrForTest(err error) {
	w.mu.Lock()
	w.lastErr = err
	w.mu.Unlock()
}

// StartReindexForTest simulates an in-flight re-index for the given duration
// (the trigger-not-gate test: a freshness check during it must not block). It
// runs on a background goroutine so the test's FreshnessForTest call happens
// while the re-index is in flight.
func (w *Watcher) StartReindexForTest(d time.Duration) {
	w.mu.Lock()
	w.reindexInFlight = true
	w.mu.Unlock()
	go func() {
		time.Sleep(d)
		w.mu.Lock()
		w.reindexInFlight = false
		w.mu.Unlock()
	}()
}

// FreshnessForTest is the CI-INV-4 freshness check: it returns the last
// committed index's freshness (index_freshness) + whether a re-index is in
// flight (degraded: stale_index when it is). It must NOT block on an in-flight
// re-index (no lock the reads wait on — the Store's WAL is the concurrency).
// Returns (lastIndexedAt, degraded).
func (w *Watcher) FreshnessForTest() (string, string) {
	w.mu.Lock()
	inFlight := w.reindexInFlight
	w.mu.Unlock()
	if inFlight {
		return "", "stale_index"
	}
	// The last committed index (the index_freshness row).
	last := ""
	if st := w.cfg.Store; st != nil {
		if raw, ok := st.(interface {
			RawQuery(query string, args ...any) ([][]any, error)
		}); ok {
			if rows, err := raw.RawQuery(`SELECT ifnull(last_indexed_at,'') FROM index_freshness LIMIT 1`); err == nil && len(rows) > 0 && len(rows[0]) > 0 {
				if s, ok := rows[0][0].(string); ok {
					last = s
				}
			}
		}
	}
	return last, ""
}

// ── include/exclude glob matching (self-contained, no import cycle) ────────

// matchesGlob reports whether rel matches the glob pattern (a simple
// path-segment glob: * matches within a segment, ** matches across segments).
// ponytail: the watch's include/exclude is a thin filter (the Index walk's
// excludedDirs is the authoritative scope); this covers the common patterns
// (a bare dir name, a dir/** glob). Upgrade path: a full glob library if the
// transport's code_watch exposes arbitrary globs.
func matchesGlob(pattern, rel string) bool {
	if pattern == "" {
		return false
	}
	// A bare dir name matches any path under it (dir or dir/...).
	if !strings.ContainsAny(pattern, "*") {
		return rel == pattern || strings.HasPrefix(rel, pattern+"/")
	}
	// A ** glob: a.go under any dir, or dir/** under dir.
	if strings.Contains(pattern, "**") {
		suffix := strings.ReplaceAll(pattern, "**/", "")
		if suffix == "" || suffix == pattern {
			return true
		}
		return rel == suffix || strings.HasSuffix(rel, "/"+suffix)
	}
	// A single * within a segment.
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "/")
		relParts := strings.Split(rel, "/")
		if len(parts) > len(relParts) {
			return false
		}
		for i, p := range parts {
			if p == "*" {
				continue
			}
			if i >= len(relParts) || relParts[i] != p {
				return false
			}
		}
		return true
	}
	return rel == pattern
}

// matchesAny reports whether rel matches any of the patterns.
func matchesAny(patterns []string, rel string) bool {
	for _, p := range patterns {
		if matchesGlob(p, rel) {
			return true
		}
	}
	return false
}
