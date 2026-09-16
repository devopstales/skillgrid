package codeindex

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestWatcher covers 03.1 (Scenario: Saving a source file triggers a debounced
// re-index): a burst of rapid source edits collapses into ONE OnSync call after
// the (injected) debounce window; only the changed surface is re-indexed;
// non-source files are ignored. Time-based behavior is injectable — the test
// uses a 50ms window instead of the 2000ms default so it does not wait.
func TestWatcher(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "main.go")
	mustWrite(t, src, "package main\n")
	nsrc := filepath.Join(root, "notes.txt")
	mustWrite(t, nsrc, "hello\n")

	var calls int32
	var synced []string
	var mu sync.Mutex
	w := NewWatcher(root, WatchConfig{
		Debounced: 50 * time.Millisecond, // injected short window
		Include:   []string{"**/*.go"},
		Exclude:   []string{"**/node_modules/**", "**/.git/**"},
		OnSync: func(ctx context.Context, files []string) error {
			atomic.AddInt32(&calls, 1)
			mu.Lock()
			synced = append(synced, files...)
			mu.Unlock()
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := w.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer w.Stop()

	// Give the fsnotify watcher a beat to attach to the directory.
	time.Sleep(30 * time.Millisecond)

	// A burst of rapid edits to the source file within the debounce window.
	for i := 0; i < 3; i++ {
		if err := os.WriteFile(src, []byte("package main\n\nfunc f"+itoa(i)+"() {}\n"), 0o644); err != nil {
			t.Fatalf("write burst %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	// A non-source edit must NOT trigger a sync (it is ignored).
	if err := os.WriteFile(nsrc, []byte("hello world\n"), 0o644); err != nil {
		t.Fatalf("write non-source: %v", err)
	}

	// Wait for the debounce window to elapse and the single burst to fire.
	deadline := time.Now().Add(600 * time.Millisecond)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&calls) >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected exactly 1 OnSync call (burst collapsed), got %d", got)
	}
	mu.Lock()
	got := append([]string(nil), synced...)
	mu.Unlock()
	if len(got) == 0 {
		t.Fatalf("OnSync received no files")
	}
	foundSrc := false
	for _, p := range got {
		if p == "main.go" {
			foundSrc = true
		}
		if p == "notes.txt" {
			t.Errorf("non-source file %q was synced (should be ignored)", p)
		}
	}
	if !foundSrc {
		t.Errorf("source file main.go not in synced set: %v", got)
	}
}

// TestWatchDisabled covers 03.9 (Scenario: Watcher disabled means manual
// index): with SKILLGRID_NO_WATCH=1 the watcher is disabled (manual index).
func TestWatchDisabled(t *testing.T) {
	t.Setenv(NoWatchEnv, "1")
	if !WatchDisabled() {
		t.Errorf("expected WatchDisabled()=true with %s=1", NoWatchEnv)
	}
	t.Setenv(NoWatchEnv, "0")
	if WatchDisabled() {
		t.Errorf("expected WatchDisabled()=false with %s=0", NoWatchEnv)
	}
	t.Setenv(NoWatchEnv, "")
	if WatchDisabled() {
		t.Errorf("expected WatchDisabled()=false when unset")
	}
}
