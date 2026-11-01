package codeindex

import (
	"path/filepath"
	"testing"
)

// TestWriterLock covers 03.6 (Scenario: Second writer exits with a
// writer-lock error): the first writer acquires the single-writer lock; a
// second writer exits with a clear writer-lock error pointing at the live
// writer. Both the in-process lock (the watcher + fingerprint gate within one
// process) and the cross-process O_EXCL file lock are covered.
func TestWriterLock(t *testing.T) {
	// In-process writer lock.
	l := NewWriterLock()
	if err := l.Acquire("writer-a"); err != nil {
		t.Fatalf("first writer acquire: %v", err)
	}
	if l.Holder() != "writer-a" {
		t.Fatalf("holder = %q, want writer-a", l.Holder())
	}
	// A second writer must be rejected with a clear lock error pointing at the
	// live writer.
	err := l.Acquire("writer-b")
	if err == nil {
		t.Fatalf("expected a lock error for the second writer, got nil")
	}
	if !IsLockError(err) {
		t.Fatalf("expected a *LockError, got %v", err)
	}
	if l.Holder() != "writer-a" {
		t.Fatalf("holder must remain writer-a after the rejected second writer, got %q", l.Holder())
	}
	// The error message points at the live writer.
	if got := err.Error(); got == "" || !containsStr(got, "writer-a") {
		t.Fatalf("lock error must name the live writer, got: %q", got)
	}
	// Release → the second writer can now acquire.
	l.Release()
	if err := l.Acquire("writer-b"); err != nil {
		t.Fatalf("second writer acquire after release: %v", err)
	}
	l.Release()

	// Cross-process O_EXCL file lock.
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "writer.lock")
	release, err := AcquireFileLock(lockPath, "serve-1")
	if err != nil {
		t.Fatalf("first file lock acquire: %v", err)
	}
	// A second process must get a lock error pointing at the live holder.
	_, err = AcquireFileLock(lockPath, "serve-2")
	if err == nil {
		t.Fatalf("expected a file lock error for the second writer, got nil")
	}
	if !IsLockError(err) {
		t.Fatalf("expected a *LockError, got %v", err)
	}
	if got := err.Error(); !containsStr(got, "serve-1") {
		t.Fatalf("file lock error must name the live holder serve-1, got: %q", got)
	}
	release()
	// After release the lock is acquirable again.
	if release2, err := AcquireFileLock(lockPath, "serve-3"); err != nil {
		t.Fatalf("file lock acquire after release: %v", err)
	} else {
		release2()
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
