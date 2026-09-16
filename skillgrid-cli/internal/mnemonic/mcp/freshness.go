package mcp

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/search"
)

// freshness is the response-path wiring for the auto-sync watcher +
// pull-at-query fingerprint gate + reader auto-reopen. It is package-level so
// every code_* tool handler funnels through the same staleness banner /
// auto-reopen / fingerprint-gate logic. It is a no-op (zero overhead) until
// InitFreshness is called with a live index path.
type freshness struct {
	mu        sync.Mutex
	tracker   *codeindex.ReopenTracker
	lock      *codeindex.WriterLock
	watcher   *codeindex.Watcher
	root      string
	cfg       codeindex.Config
	svc       *service.Service
	stamp     string
	pendFn    func() map[string]struct{}
}

var fresh = &freshness{lock: codeindex.NewWriterLock()}

// InitFreshness wires the response path for project: it records the live index
// path (for auto-reopen) and the index root + config (for the fingerprint
// gate's structural re-index). reopenWindow <= 0 uses codeindex's ~5s default.
func InitFreshness(svc *service.Service, projectID, root string, cfg codeindex.Config, reopenWindow time.Duration) {
	fresh.mu.Lock()
	defer fresh.mu.Unlock()
	if projectID == "" {
		return
	}
	fresh.svc = svc
	fresh.root = root
	fresh.cfg = cfg
	fresh.stamp = codeindex.ExtractorStamp()
	fresh.tracker = codeindex.NewReopenTracker(liveIndexPath(svc, projectID), reopenWindow)
}

// liveIndexPath resolves the project's SQLite index path (the file readers
// open): <dataDir>/<projectID>.sqlite.
func liveIndexPath(_ *service.Service, projectID string) string {
	dataDir := dataDirOrHome()
	return filepath.Join(dataDir, projectID+".sqlite")
}

func dataDirOrHome() string {
	if v := os.Getenv("SKILLGRID_MNEMONIC_DATA_DIR"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".skillgrid", "mnemonic")
}

// preQuery runs the reader auto-reopen + pull-at-query fingerprint gate before
// a code_* query answers. It returns a staleness banner (a referenced pending
// file) and a footer (un-referenced pending files). It never touches the
// embedder leg (the gate is structural-only).
func (f *freshness) preQuery(ctx context.Context, referencedPaths []string) (banner, footer string) {
	f.mu.Lock()
	tracker := f.tracker
	lock := f.lock
	root := f.root
	cfg := f.cfg
	svc := f.svc
	stamp := f.stamp
	f.mu.Unlock()

	// 1) Auto-reopen: if a newer index was published within the window, the
	// tracker fires its reopen hook (no restart). No-op when the tracker is nil.
	if tracker != nil {
		_ = tracker.Check()
	}

	// 2) Pull-at-query fingerprint gate: a ~3ms (size, mtime) stat-walk against
	// the last index's fingerprint; on drift a structural-only incremental
	// re-index runs (under the writer lock) BEFORE the answer. Never the
	// embedder leg. No-op when freshness is not initialized.
	drained := map[string]struct{}{}
	if tracker != nil && svc != nil && root != "" {
		if err := lock.Acquire("query-gate"); err == nil {
			if changed, _ := codeindex.FingerprintDrift(root, cfg, stamp); len(changed) > 0 {
				// Structural-only re-index (no embedder). Advisory: a failure
				// keeps the old index live.
				_, _ = svc.ReindexStructural(ctx, root, cfg)
				for p := range changed {
					drained[p] = struct{}{}
				}
			}
			lock.Release()
		}
	}

	// 3) Staleness banner: a pending file the response references gets a
	// banner; un-referenced pending files surface as a footer. Files drained by
	// the gate (re-indexed) are no longer pending.
	pending := f.pendingPaths()
	for p := range drained {
		delete(pending, p)
	}
	refs := map[string]struct{}{}
	for _, p := range referencedPaths {
		refs[filepath.ToSlash(p)] = struct{}{}
	}
	for p := range pending {
		if _, ok := refs[p]; ok {
			banner = "⚠️ " + p + " is pending sync — Read it directly"
			break
		}
	}
	if len(pending) > 0 {
		footer = stalenessFooter(pending)
	}
	return banner, footer
}

// pendingPaths returns the current pending (unsynced) set. A custom pendFn
// (test hook) wins; otherwise the watcher's pending set is used. When the
// watcher is off (nil), the set is empty (the fingerprint gate is the backstop).
func (f *freshness) pendingPaths() map[string]struct{} {
	f.mu.Lock()
	w := f.watcher
	pendFn := f.pendFn
	f.mu.Unlock()
	if pendFn != nil {
		return pendFn()
	}
	if w == nil {
		return map[string]struct{}{}
	}
	out := map[string]struct{}{}
	for _, p := range w.Pending() {
		out[filepath.ToSlash(p)] = struct{}{}
	}
	return out
}

// SetPendingSource overrides the pending-file source (test hook; nil restores
// the watcher).
func SetPendingSource(fn func() map[string]struct{}) {
	fresh.mu.Lock()
	fresh.pendFn = fn
	fresh.mu.Unlock()
}

// AttachWatcher attaches the autosync watcher to the response path (so pending
// files feed the staleness banner). nil detaches.
func AttachWatcher(w *codeindex.Watcher) {
	fresh.mu.Lock()
	fresh.watcher = w
	fresh.mu.Unlock()
}

// SetReopenHook installs the auto-reopen hook on the current tracker (nil
// tracker is a no-op). Test helper to observe reopens.
func SetReopenHook(fn func(livePath string)) {
	fresh.mu.Lock()
	defer fresh.mu.Unlock()
	if fresh.tracker != nil {
		fresh.tracker.OnReopen = fn
	}
}

// applyFreshness prepends the staleness banner and appends the footer to a
// code_* tool response. It runs the pull-at-query fingerprint gate (structural
// only, embedder-free) + reader auto-reopen before answering, so the response
// reflects the just-made edit even with the watcher off. When no pending file
// is referenced and no footer is due, res is returned unchanged.
func applyFreshness(res *mcplib.CallToolResult, referencedPaths []string) *mcplib.CallToolResult {
	banner, footer := fresh.preQuery(context.Background(), referencedPaths)
	if banner == "" && footer == "" {
		return res
	}
	var prefix, suffix string
	if banner != "" {
		prefix = banner + "\n"
	}
	if footer != "" {
		suffix = "\n" + footer
	}
	for i, c := range res.Content {
		if tc, ok := c.(mcplib.TextContent); ok {
			res.Content[i] = mcplib.TextContent{Text: prefix + tc.Text + suffix}
		}
	}
	return res
}

// hitPaths extracts the referenced file paths from a code_search hit list (for
// the staleness banner: a hit that points at a still-pending file is named).
func hitPaths(hits []search.CodeHit) []string {
	out := make([]string, 0, len(hits))
	for _, h := range hits {
		out = append(out, h.Path)
	}
	return out
}

// stalenessFooter formats the un-referenced pending files as a small footer.
func stalenessFooter(pending map[string]struct{}) string {
	if len(pending) == 0 {
		return ""
	}
	paths := make([]string, 0, len(pending))
	for p := range pending {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return "⚠️ pending sync (not yet indexed): " + strings.Join(paths, ", ")
}
