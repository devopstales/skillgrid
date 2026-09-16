package codeindex

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestPublish covers 03.2 (Scenario: Reader sees old or new index never torn):
// the sidecar copy-and-swap publishes the new index atomically (a concurrent
// reader observes the old or the new, never a partial one); on a
// sidecar-unsupported filesystem the in-place fallback retries on pre-write
// failure and stops when it may have mutated the live index (keeping the old
// index live).
func TestPublish(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "index.sqlite")

	// The live index currently holds the "old" content.
	if err := os.WriteFile(live, []byte("old-index"), 0o644); err != nil {
		t.Fatalf("seed live: %v", err)
	}

	// Sidecar mode: Copy copies the live file, Build appends the new content,
	// and the atomic rename swaps it in.
	pub := &Publish{
		LivePath:       live,
		SupportSidecar: true,
		Copy: func(src, dst string) error {
			b, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			return os.WriteFile(dst, b, 0o644)
		},
		Build: func(sidecar string) error {
			// The sidecar now holds the full new index (old + new tail).
			return os.WriteFile(sidecar, []byte("new-index"), 0o644)
		},
	}
	if err := pub.Publish(); err != nil {
		t.Fatalf("sidecar publish: %v", err)
	}
	// The live index must now hold the new content, and no sidecar remains.
	got, err := os.ReadFile(live)
	if err != nil {
		t.Fatalf("read live after publish: %v", err)
	}
	if string(got) != "new-index" {
		t.Errorf("live index after sidecar publish = %q, want %q", got, "new-index")
	}
	if sidecarExists(live) {
		t.Errorf("sidecar %q remained after an atomic swap", PublishPath(live))
	}
}

// TestPublishInPlaceRetry covers the in-place fallback (sidecar unsupported):
// a pre-write failure is retried and eventually succeeds (the new index is
// live), while a mid-write failure stops with ErrMayHaveMutated (the old index
// stays live so a reader never sees a torn one).
func TestPublishInPlaceRetry(t *testing.T) {
	t.Run("pre-write retry succeeds", func(t *testing.T) {
		dir := t.TempDir()
		live := filepath.Join(dir, "index.sqlite")
		if err := os.WriteFile(live, []byte("old-index"), 0o644); err != nil {
			t.Fatalf("seed live: %v", err)
		}
		attempts := 0
		pub := &Publish{
			LivePath:         live,
			SupportSidecar:   false,
			MaxInPlaceRetries: 3,
			InPlace: func() error {
				attempts++
				if attempts == 1 {
					return NewPreWriteFailure(fmt.Errorf("disk full before write"))
				}
				return os.WriteFile(live, []byte("new-index"), 0o644)
			},
		}
		if err := pub.Publish(); err != nil {
			t.Fatalf("in-place publish after pre-write retry: %v", err)
		}
		if attempts != 2 {
			t.Errorf("expected 2 in-place attempts (1 pre-write fail + 1 success), got %d", attempts)
		}
		got, _ := os.ReadFile(live)
		if string(got) != "new-index" {
			t.Errorf("live index = %q, want %q", got, "new-index")
		}
	})

	t.Run("mid-write failure stops with MayHaveMutated", func(t *testing.T) {
		dir := t.TempDir()
		live := filepath.Join(dir, "index.sqlite")
		if err := os.WriteFile(live, []byte("old-index"), 0o644); err != nil {
			t.Fatalf("seed live: %v", err)
		}
		attempts := 0
		pub := &Publish{
			LivePath:         live,
			SupportSidecar:   false,
			MaxInPlaceRetries: 3,
			InPlace: func() error {
				attempts++
				// A mid-write failure: bytes may have landed, so it must NOT
				// be treated as pre-write (retry-safe).
				return fmt.Errorf("write interrupted mid-index")
			},
		}
		err := pub.Publish()
		if err == nil {
			t.Fatalf("expected an error from a mid-write in-place failure, got nil")
		}
		if !IsMayHaveMutated(err) {
			t.Errorf("expected ErrMayHaveMutated, got: %v", err)
		}
		// It must have stopped after the first (non-pre-write) attempt, NOT
		// retried.
		if attempts != 1 {
			t.Errorf("expected 1 in-place attempt (stop on mid-write), got %d", attempts)
		}
	})
}
