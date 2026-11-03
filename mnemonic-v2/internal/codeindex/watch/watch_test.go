package watch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/extract"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// openTestStore opens a real SQLite store for the watch package tests.
func openTestStore(t *testing.T) store.Store {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(context.Background(), dir, "p", store.SQLite, "")
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// [watch-debounce-burst-collapse] a burst of 5 edits within the debounce
// window collapses to exactly ONE onSync (deterministic — the DrainForTest
// seam drives the collapse; no flaky sleep, no dead second watcher).
func TestWatchBurstCollapse(t *testing.T) {
	const debounce = 20 * time.Millisecond
	var calls int
	var lastPaths []string
	done := make(chan struct{}, 1)
	w := NewWatcher(t.TempDir(), WatchConfig{Debounced: debounce},
		func(paths []string) error {
			calls++
			lastPaths = paths
			done <- struct{}{}
			return nil
		})
	// 5 events in the burst (5 distinct files).
	w.EmitForTest("a.go", "b.go", "c.go", "d.go", "e.go")
	// Drive the debounce + burst-collapse.
	w.DrainForTest()
	// exactly ONE onSync fired.
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("burst must collapse to 1 onSync (timed out)")
	}
	if calls != 1 {
		t.Fatalf("the burst of 5 must collapse to exactly 1 onSync, got %d", calls)
	}
	// the deduped path list carries all 5 (order-free).
	set := map[string]bool{}
	for _, p := range lastPaths {
		set[p] = true
	}
	if len(set) != 5 {
		t.Fatalf("the collapsed onSync must carry the deduped 5 paths, got %v", lastPaths)
	}
	// F2: the "no second call in the settle window" property. Wait a settle
	// window (3× the debounce) and assert no second delivery — the burst
	// collapsed to exactly one onSync and nothing follows. Deterministic: the
	// EmitForTest seam + the counter (no flaky real-timer timing; the counter
	// is the source of truth, the wait only bounds the settle window).
	settle := 3 * debounce
	select {
	case <-done:
		t.Fatalf("a second onSync fired within the settle window: the burst must collapse to exactly one call")
	case <-time.After(settle):
	}
	if w.ReindexCalled() != 1 {
		t.Fatalf("no second onSync should fire after the settle window, ReindexCalled=%d", w.ReindexCalled())
	}
}

// [watch-fingerprint-recheck] an mtime-only change (size+content unchanged,
// stamp unchanged) is a NO-OP: the metadata-only stat-walk re-check compares
// size+mtime against the files table + the extractor stamp, and an unchanged
// fingerprint does not trigger a re-index.
func TestWatchFingerprintRecheck(t *testing.T) {
	st := openTestStore(t)
	root := t.TempDir()
	ctx := context.Background()

	// Index a file (records size+mtime in the files table).
	p := filepath.Join(root, "a.go")
	os.WriteFile(p, []byte("package a\n"), 0o644)
	if _, err := codeindex.Index(ctx, root, st, codeindex.IndexOpts{}); err != nil {
		t.Fatalf("first index: %v", err)
	}
	// The file is now in the index.
	if _, err := st.FileByPath(ctx, "a.go"); err != nil {
		t.Fatalf("file must be indexed: %v", err)
	}

	// F1: an mtime-ONLY change — bump mtime (os.Chtimes) with size+content
	// unchanged, so the re-check sees a changed-mtime input. The spec's no-op
	// case is "mtime touched, content identical": the re-check compares
	// size+mtime against the files table + the stamp, and an unchanged
	// fingerprint (content identical) does NOT trigger a re-index. The
	// WatchConfig carries the Store (the re-check's files-table read) + the
	// current stamp.
	now := time.Now()
	if err := os.Chtimes(p, now, now.Add(time.Minute)); err != nil {
		t.Fatalf("bump mtime: %v", err)
	}
	w := NewWatcher(root, WatchConfig{
		Debounced: 10 * time.Millisecond,
		Stamp:     extract.Stamp(),
		Store:     st,
	}, nil)
	w.MarkPendingForTest("a.go") // a pending path whose mtime changed, content identical
	w.DrainForTest()
	if w.ReindexCalled() != 0 {
		t.Fatalf("an mtime-only (content identical) change must be a no-op, but ReindexCalled=%d", w.ReindexCalled())
	}
}

// [watch-failure-isolation] a re-index that fails does NOT stop the watcher —
// the next event still drains (the error is surfaced via LastErrForTest, the
// watcher stays alive).
func TestWatchFailureIsolation(t *testing.T) {
	var calls int
	w := NewWatcher(t.TempDir(), WatchConfig{Debounced: 10 * time.Millisecond},
		func(paths []string) error {
			calls++
			if calls == 1 {
				// The first re-index fails (a store lock) — a REAL returning
				// error (the v1 OnSyncFunc signature shape), not a test-seam
				// injection.
				return errors.New("store lock")
			}
			// The second re-index succeeds.
			return nil
		})
	// First event → onSync returns an error.
	w.EmitForTest("a.go")
	w.DrainForTest()
	if w.LastErrForTest() == nil {
		t.Fatalf("the failing re-index must surface its real error via LastErrForTest")
	}
	if w.LastErrForTest().Error() != "store lock" {
		t.Fatalf("LastErrForTest must carry the REAL onSync error, got %v", w.LastErrForTest())
	}
	// The watcher is still alive (a failure must not stop it).
	if !w.Running() {
		t.Fatalf("a re-index failure must not stop the watcher")
	}
	// The next event still drains + fires onSync.
	w.EmitForTest("b.go")
	w.DrainForTest()
	if calls != 2 {
		t.Fatalf("the next event must still fire onSync (failure isolation), got calls=%d", calls)
	}
	if w.LastErrForTest() != nil {
		t.Fatalf("the second (succeeding) re-index must clear LastErr, got %v", w.LastErrForTest())
	}
}

// [watch-trigger-not-gate] CI-INV-4: a freshness check during a re-index
// returns immediately (it sees the last committed index + degraded:stale_index),
// never blocking on the in-flight re-index.
func TestWatchTriggerNotGate(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	// Seed the index so the freshness check has a committed index to return.
	root := t.TempDir()
	p := filepath.Join(root, "a.go")
	os.WriteFile(p, []byte("package a\n"), 0o644)
	if _, err := codeindex.Index(ctx, root, st, codeindex.IndexOpts{}); err != nil {
		t.Fatalf("first index: %v", err)
	}

	w := NewWatcher(root, WatchConfig{Debounced: 5 * time.Millisecond, Store: st}, nil)
	// A 200ms re-index in flight (holds no lock the reads wait on — CI-INV-4).
	// The window is long enough for the DURING check to land in-flight.
	w.StartReindexForTest(200 * time.Millisecond)
	// F4: DURING the in-flight re-index, the freshness check returns
	// immediately (no block) AND degraded == "stale_index" (the value, not
	// just the timing).
	start := time.Now()
	fresh, degraded := w.FreshnessForTest()
	if time.Since(start) > 20*time.Millisecond {
		t.Fatalf("a freshness check must not block on the re-index (CI-INV-4): took %v", time.Since(start))
	}
	if degraded != "stale_index" {
		t.Fatalf("a freshness check DURING a re-index must report degraded=stale_index, got %q", degraded)
	}
	if fresh != "" {
		t.Fatalf("during a re-index the check returns the in-flight signal, not a committed value, got fresh=%q", fresh)
	}
	// After the window passes, the check returns degraded == "" + a non-empty
	// last-committed value (the index seeded above).
	w.Stop()
	// The Stop() closes w.stop, which clears StartReindexForTest's sleeper.
	// Give the in-flight flag a moment to clear (the sleeper clears on stop).
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, d := w.FreshnessForTest(); d == "" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	_, degradedAfter := w.FreshnessForTest()
	if degradedAfter != "" {
		t.Fatalf("after the re-index window the freshness check must report degraded=\", got %q", degradedAfter)
	}
	// The last-committed value is non-empty (the index_freshness row the Index
	// driver wrote).
	committed, _ := w.FreshnessForTest()
	if committed == "" {
		t.Fatalf("after the re-index window the check must return the last committed index (non-empty last_indexed_at)")
	}
}

// [M10] absent-on-disk = delete: a file deleted between the watch event and the
// re-index is a Store.DeleteFile (the files row removed, symbols/edges
// cascading), NOT an error. The re-index treats the absence as a delete.
func TestWatchDeleteBetweenEventAndReindex(t *testing.T) {
	st := openTestStore(t)
	root := t.TempDir()
	ctx := context.Background()
	p := filepath.Join(root, "gone.go")
	// A Go file with a real symbol so the cascade has something to remove
	// (F5: assert the symbol is gone too, not just the files row).
	os.WriteFile(p, []byte("package gone\n\nfunc Gone() {}\n"), 0o644)
	// First index: the file + its symbol are in the index.
	if _, err := codeindex.Index(ctx, root, st, codeindex.IndexOpts{}); err != nil {
		t.Fatalf("first index: %v", err)
	}
	if _, err := st.FileByPath(ctx, "gone.go"); err != nil {
		t.Fatalf("file must be indexed: %v", err)
	}
	// F5: capture the known symbol before deletion (the cascade target).
	syms, err := st.SymbolByName(ctx, "Gone")
	if err != nil || len(syms) == 0 {
		t.Fatalf("the real extractor must index the Gone symbol for the cascade assertion: %+v, %v", syms, err)
	}
	goneSymID := syms[0].ID

	// The event fires, then the file is removed before the re-index runs. The
	// onSync is the real Index driver (absent-on-disk = delete, M10).
	w := NewWatcher(root, WatchConfig{
		Debounced: 5 * time.Millisecond,
		Stamp:     extract.Stamp(),
		Store:     st,
	}, func(paths []string) error {
		_, err := codeindex.Index(ctx, root, st, codeindex.IndexOpts{})
		return err
	})
	w.EmitForTest("gone.go")
	// The file vanishes between the event and the drain.
	if err := os.Remove(p); err != nil {
		t.Fatalf("remove file: %v", err)
	}
	w.DrainForTest()
	// The files row is gone (absent-on-disk = delete, M10 — the re-index walks
	// the root, the file is absent, and the driver's absent-on-disk = delete
	// removes it + cascades symbols/edges/chunks).
	if _, err := st.FileByPath(ctx, "gone.go"); err == nil {
		t.Fatalf("absent-on-disk must DELETE the files row (M10, cascading symbols/edges)")
	}
	// F5: the cascade is clean — the symbol is gone too (the FK ON DELETE
	// CASCADE removed it with the files row). A delete that left orphan
	// symbols would fail here.
	if _, err := st.SymbolByID(ctx, goneSymID); err == nil {
		t.Fatalf("the FK ON DELETE CASCADE must remove the symbol with the files row (cascade clean), but symbol %d still exists", goneSymID)
	}
	// NO error surfaced: the re-index treated the absence as a delete.
	if w.LastErrForTest() != nil {
		t.Fatalf("a deleted file must not be a re-index error (M10): %v", w.LastErrForTest())
	}
}

// The ONE real-fsnotify integration test: a real temp-dir write fires the
// watcher → the onSync drains. Bounded timeout + t.Cleanup(Stop).
func TestWatchRealFsnotifyIntegration(t *testing.T) {
	root := t.TempDir()
	done := make(chan []string, 1)
	w := NewWatcher(root, WatchConfig{Debounced: 50 * time.Millisecond},
		func(paths []string) error {
			done <- paths
			return nil
		})
	if err := w.Start(context.Background()); err != nil {
		t.Fatalf("start real fsnotify watcher: %v", err)
	}
	t.Cleanup(w.Stop)
	// A real file write under the watched root.
	time.Sleep(50 * time.Millisecond) // let the fsnotify registration settle
	if err := os.WriteFile(filepath.Join(root, "live.go"), []byte("package live\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	select {
	case paths := <-done:
		if len(paths) == 0 {
			t.Fatalf("real fsnotify event must produce a non-empty path list: %v", paths)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("real fsnotify event did not fire the onSync within 2s (bounded)")
	}
}
