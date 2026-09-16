package codeindex

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// DefaultDebounce is the default burst-collapse window. A burst of rapid edits
// (create/modify/delete) collapses into a single incremental re-index after
// this window of inactivity. Tunable per Watcher (clamped to [MinDebounce,
// MaxDebounce]).
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

// WatchConfig configures an autosync Watcher.
type WatchConfig struct {
	// Debounce is the burst-collapse window (clamped to [MinDebounce,
	// MaxDebounce]; <=0 uses DefaultDebounce).
	Debounced time.Duration
	// Include/Exclude are the 005 source filters (include/exclude + the
	// implicit .gitignore handling in Scan). Non-source files (not matching
	// Include, or matching Exclude) are ignored.
	Include []string
	Exclude []string
	// OnSync is called (under the writer lock) when a debounced burst fires.
	// The rel paths are the files observed (absolute paths under root). A nil
	// OnSync means the watcher only tracks pending files (no re-index).
	OnSync func(ctx context.Context, files []string) error
}

// Watcher is the autosync watcher (the comfort layer): it watches the index
// root for create/modify/delete of source files, debounces bursts, and runs a
// single incremental re-index under the writer lock. It is the comfort layer —
// the pull-at-query fingerprint gate (fingerprint.go) is the correctness
// backstop that works with the watcher off.
type Watcher struct {
	root     string
	cfg      WatchConfig
	watcher  *fsnotify.Watcher
	stop     chan struct{}
	stopOnce sync.Once

	mu       sync.Mutex
	pending  map[string]struct{}
	stopped  bool
	wasWrite bool
}

// NewWatcher creates an autosync watcher for root. It does not start the
// fsnotify loop until Start.
func NewWatcher(root string, cfg WatchConfig) *Watcher {
	return &Watcher{
		root:    root,
		cfg:     cfg,
		stop:    make(chan struct{}),
		pending: map[string]struct{}{},
	}
}

// Debounce returns the effective (clamped) debounce window.
func (w *Watcher) Debounce() time.Duration {
	return clampDebounce(w.cfg.Debounced)
}

// IsSource reports whether a repo-relative path is a source file per the 005
// include/exclude filters. Directory exclusion mirrors Scan's shouldSkipDir so
// a change inside an excluded directory is ignored.
func (w *Watcher) IsSource(rel string) bool {
	rel = filepath.ToSlash(rel)
	for _, pattern := range w.cfg.Exclude {
		if shouldSkipDir(rel, []string{pattern}) || matchesGlob(pattern, rel) {
			return false
		}
	}
	if len(w.cfg.Include) > 0 && !matchesAny(w.cfg.Include, rel) {
		return false
	}
	return true
}

// Start begins the fsnotify loop. It returns an error if the watcher could not
// be created (e.g. the root is missing). OnSync runs on a background goroutine
// so Start returns promptly.
func (w *Watcher) Start(ctx context.Context) error {
	fn, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create fsnotify watcher: %w", err)
	}
	if err := fn.Add(w.root); err != nil {
		fn.Close()
		return fmt.Errorf("watch root %s: %w", w.root, err)
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

// loop consumes fsnotify events, marks source files pending, and debounces a
// burst into a single OnSync call.
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
			// Watcher error: stop (the fingerprint gate is the backstop).
			w.mu.Lock()
			w.stopped = true
			w.mu.Unlock()
			timer.Stop()
			return
		case <-timer.C:
			fire = false
			paths := w.drainPending()
			if len(paths) == 0 {
				continue
			}
			if w.cfg.OnSync != nil {
				_ = w.cfg.OnSync(ctx, paths)
			}
		}
	}
}

// shouldFire reports whether an event is a create/modify/delete of a source
// file (non-source, rename, and chown events are ignored).
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

// markPending records the event's file as pending (for the staleness banner)
// and notes whether the burst contains a write.
func (w *Watcher) markPending(evt fsnotify.Event) {
	w.mu.Lock()
	defer w.mu.Unlock()
	rel, _ := filepath.Rel(w.root, evt.Name)
	if rel == "." {
		rel = ""
	}
	w.pending[filepath.ToSlash(rel)] = struct{}{}
	if evt.Op&fsnotify.Write != 0 {
		w.wasWrite = true
	}
}

// drainPending returns and clears the pending set (sorted-free, caller
// tolerates order) and resets the wasWrite flag.
func (w *Watcher) drainPending() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]string, 0, len(w.pending))
	for p := range w.pending {
		out = append(out, p)
	}
	w.pending = map[string]struct{}{}
	w.wasWrite = false
	return out
}

// MarkPending records a file as pending (unsynced) without an fsnotify event.
// The staleness banner reads the pending set; the serve wiring calls this for
// files edited since the last sync. It is a no-op for non-source files.
func (w *Watcher) MarkPending(rel string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	rel = filepath.ToSlash(rel)
	if w.IsSource(rel) {
		w.pending[rel] = struct{}{}
	}
}

// ClearPending clears the pending set (called after a sync completes).
func (w *Watcher) ClearPending() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pending = map[string]struct{}{}
	w.wasWrite = false
}

// Pending returns the current set of pending (unsynced) source files.
func (w *Watcher) Pending() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]string, 0, len(w.pending))
	for p := range w.pending {
		out = append(out, p)
	}
	return out
}

// Stopped reports whether the watcher loop has stopped (e.g. a watcher error).
func (w *Watcher) Stopped() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.stopped
}

// NoWatchEnv is the env var that disables the watcher (manual index only).
const NoWatchEnv = "SKILLGRID_NO_WATCH"

// WatchDisabled reports whether the watcher is disabled via SKILLGRID_NO_WATCH.
// A disabled watcher means manual index: the agent must run `skillgrid index`
// (or code_index) to refresh, and the fingerprint gate is the backstop.
func WatchDisabled() bool {
	v := os.Getenv(NoWatchEnv)
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	return err == nil && b
}

// LockError is returned when a second writer tries to acquire the
// single-writer lock already held by the live writer.
type LockError struct {
	Holder string
}

func (e *LockError) Error() string {
	return fmt.Sprintf("writer lock: another writer is live (%s); this process is read-only until it exits", e.Holder)
}

// IsLockError reports whether err (or its cause chain) is a writer-lock error.
func IsLockError(err error) bool {
	for err != nil {
		if _, ok := err.(*LockError); ok {
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

// WriterLock is an in-process single-writer lock shared by the watcher and the
// pull-at-query fingerprint gate's re-index: a second writer must exit with a
// clear lock error pointing at the live writer. In the multi-process case the
// lock is backed by an O_EXCL file under the data dir (see AcquireFileLock);
// this in-process type guards the watcher + fingerprint gate within one
// process.
type WriterLock struct {
	mu     sync.Mutex
	holder string
}

// NewWriterLock returns a fresh in-process writer lock.
func NewWriterLock() *WriterLock {
	return &WriterLock{}
}

// Acquire takes the lock, naming the holder. A second caller receives a
// *LockError pointing at the live holder.
func (l *WriterLock) Acquire(holder string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.holder != "" {
		return &LockError{Holder: l.holder}
	}
	l.holder = holder
	return nil
}

// Release frees the lock.
func (l *WriterLock) Release() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.holder = ""
}

// Holder returns the current holder ("" when free).
func (l *WriterLock) Holder() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.holder
}

// AcquireFileLock acquires a cross-process writer lock via an O_EXCL file at
// path. On success it returns a Release func. If another process holds the
// lock, it returns a *LockError naming the live holder (read from the file).
func AcquireFileLock(path, holder string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err == nil {
		_, _ = f.WriteString(holder)
		_ = f.Close()
		return func() { _ = os.Remove(path) }, nil
	}
	if !os.IsExist(err) {
		return nil, err
	}
	// Another writer holds it: read the live holder.
	body, _ := os.ReadFile(path)
	return nil, &LockError{Holder: strings.TrimSpace(string(body))}
}
