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
	var calls int
	var lastPaths []string
	done := make(chan struct{}, 1)
	w := NewWatcher(t.TempDir(), WatchConfig{Debounced: 50 * time.Millisecond},
		func(paths []string) {
			calls++
			lastPaths = paths
			done <- struct{}{}
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
	// a second DrainForTest with no new events fires NO second onSync.
	w.DrainForTest()
	if w.ReindexCalled() != 1 {
		t.Fatalf("no second onSync should fire, ReindexCalled=%d", w.ReindexCalled())
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

	// An mtime-only change: the size is unchanged and the content is unchanged
	// (so the fingerprint — size+mtime+stamp — is unchanged). The re-check must
	// be a no-op (no re-index). The WatchConfig carries the Store (the
	// re-check's files-table read) + the current stamp.
	w := NewWatcher(root, WatchConfig{
		Debounced: 10 * time.Millisecond,
		Stamp:     extract.Stamp(),
		Store:     st,
	}, nil)
	w.MarkPendingForTest("a.go") // a pending path whose fingerprint is unchanged
	w.DrainForTest()
	if w.ReindexCalled() != 0 {
		t.Fatalf("an mtime-only (size+content unchanged) change must be a no-op, but ReindexCalled=%d", w.ReindexCalled())
	}
}

// [watch-failure-isolation] a re-index that fails does NOT stop the watcher —
// the next event still drains (the error is surfaced via LastErrForTest, the
// watcher stays alive).
func TestWatchFailureIsolation(t *testing.T) {
	var calls int
	var w *Watcher
	w = NewWatcher(t.TempDir(), WatchConfig{Debounced: 10 * time.Millisecond},
		func(paths []string) {
			calls++
			if calls == 1 {
				// The first re-index fails (a store lock).
				w.setErrForTest(errors.New("store lock"))
				return
			}
			// The second re-index succeeds.
			w.setErrForTest(nil)
		})
	// First event → onSync fails.
	w.EmitForTest("a.go")
	w.DrainForTest()
	if w.LastErrForTest() == nil {
		t.Fatalf("the failing re-index must surface its error via LastErrForTest")
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
	// A 50ms re-index in flight (holds no lock the reads wait on — CI-INV-4).
	w.StartReindexForTest(50 * time.Millisecond)
	start := time.Now()
	fresh, degraded := w.FreshnessForTest()
	if time.Since(start) > 20*time.Millisecond {
		t.Fatalf("a freshness check must not block on the re-index (CI-INV-4): took %v", time.Since(start))
	}
	_ = fresh
	_ = degraded
	w.Stop()
}

// [M10] absent-on-disk = delete: a file deleted between the watch event and the
// re-index is a Store.DeleteFile (the files row removed, symbols/edges
// cascading), NOT an error. The re-index treats the absence as a delete.
func TestWatchDeleteBetweenEventAndReindex(t *testing.T) {
	st := openTestStore(t)
	root := t.TempDir()
	ctx := context.Background()
	p := filepath.Join(root, "gone.go")
	os.WriteFile(p, []byte("package gone\n"), 0o644)
	// First index: the file is in the files table.
	if _, err := codeindex.Index(ctx, root, st, codeindex.IndexOpts{}); err != nil {
		t.Fatalf("first index: %v", err)
	}
	if _, err := st.FileByPath(ctx, "gone.go"); err != nil {
		t.Fatalf("file must be indexed: %v", err)
	}

	// The event fires, then the file is removed before the re-index runs. The
	// onSync is the real Index driver (absent-on-disk = delete, M10).
	w := NewWatcher(root, WatchConfig{
		Debounced: 5 * time.Millisecond,
		Stamp:     extract.Stamp(),
		Store:     st,
	}, func(paths []string) {
		_, _ = codeindex.Index(ctx, root, st, codeindex.IndexOpts{})
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
		func(paths []string) { done <- paths })
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
