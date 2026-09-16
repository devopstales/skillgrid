package codeindex

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCatchUp covers 03.4 (Scenario: Connect-time catch-up absorbs offline
// edits): edits made while no watcher was running (a git pull, another editor)
// are reconciled by a (size, mtime) + content-hash check on (re)connect; only
// genuinely changed files are re-indexed. The reconciliation reuses the
// fingerprint stat-walk (the (size, mtime) layer) + the indexer's single
// content-hash guard.
func TestCatchUp(t *testing.T) {
	st, _, clean := openStoreFor(t)
	defer clean()
	idx := New(st)
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.go"), "package a\n\nvar A = 1\n")
	mustWrite(t, filepath.Join(root, "b.go"), "package b\n\nvar B = 2\n")
	cfg := testCfg

	// Initial index (both files).
	if _, err := idx.Run(context.Background(), root, cfg); err != nil {
		t.Fatalf("initial index: %v", err)
	}
	if err := StoreFingerprint(st.DB, root, cfg, ExtractorStamp()); err != nil {
		t.Fatalf("store fingerprint: %v", err)
	}

	// Offline edit: the watcher is OFF, so the index is not updated. We touch
	// b.go (size + mtime + content change) and leave a.go untouched.
	if err := os.WriteFile(filepath.Join(root, "b.go"), []byte("package b\n\nvar B = 20\n"), 0o644); err != nil {
		t.Fatalf("offline edit b.go: %v", err)
	}
	// Bump b.go's mtime to be clearly newer than the index.
	future := time.Now().Add(time.Second)
	_ = os.Chtimes(filepath.Join(root, "b.go"), future, future)

	// Connect-time reconciliation: the fingerprint stat-walk detects exactly
	// b.go as drifted (a.go's size + mtime are unchanged).
	changed, err := FingerprintDriftDB(st.DB, root, cfg, ExtractorStamp())
	if err != nil {
		t.Fatalf("fingerprint drift: %v", err)
	}
	if _, ok := changed["b.go"]; !ok {
		t.Errorf("expected b.go to be detected as drifted (offline edit), got %v", changed)
	}
	if _, ok := changed["a.go"]; ok {
		t.Errorf("a.go was NOT edited offline; it should not be in the drift set: %v", changed)
	}

	// Reconcile: only the drifted file is re-indexed (the indexer's content
	// hash guard skips the unchanged file).
	stats, err := idx.Run(context.Background(), root, cfg)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if stats.FilesIndexed != 1 {
		t.Errorf("expected 1 file re-indexed (b.go only), got %d", stats.FilesIndexed)
	}
	if stats.FilesSkipped != 1 {
		t.Errorf("expected 1 file skipped (a.go unchanged), got %d", stats.FilesSkipped)
	}
	// The reconciled content is now in the index (b.go's new value).
	var count int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM chunks WHERE file_id = (SELECT id FROM files WHERE path='b.go')`).Scan(&count); err != nil {
		t.Fatalf("count b.go chunks: %v", err)
	}
	if count == 0 {
		t.Errorf("expected b.go to have chunks after reconciliation")
	}
}

// TestWatchFailure covers 03.8 (Scenario: Reindex failure keeps the old index
// live): a watcher re-index that fails after it may have mutated the live index
// stops the in-place watcher and surfaces the failure; copy-and-swap mode keeps
// the old index live.
func TestWatchFailure(t *testing.T) {
	// Copy-and-swap: a build failure leaves the live index untouched (the old
	// index stays live).
	dir := t.TempDir()
	live := filepath.Join(dir, "index.sqlite")
	if err := os.WriteFile(live, []byte("old-index"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	pub := &Publish{
		LivePath:       live,
		SupportSidecar: true,
		Copy: func(src, dst string) error { return os.WriteFile(dst, []byte("x"), 0o644) },
		Build: func(_ string) error        { return os.ErrNotExist },
	}
	if err := pub.Publish(); err == nil {
		t.Fatalf("expected a build failure to surface")
	}
	got, _ := os.ReadFile(live)
	if string(got) != "old-index" {
		t.Errorf("copy-and-swap build failure must keep the old index live; got %q", got)
	}

	// In-place: a failure that may have mutated the live index stops the
	// watcher and surfaces ErrMayHaveMutated (the old index is kept).
	attempts := 0
	ip := &Publish{
		LivePath:         live,
		SupportSidecar:   false,
		MaxInPlaceRetries: 3,
		InPlace: func() error {
			attempts++
			return os.ErrPermission // mid-write (not pre-write) → stop
		},
	}
	err := ip.Publish()
	if err == nil || !IsMayHaveMutated(err) {
		t.Errorf("expected ErrMayHaveMutated, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("in-place failure must stop after the first attempt, got %d", attempts)
	}
}
