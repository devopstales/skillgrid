package store

import (
	"context"
	"testing"
)

// poolSameHandle reports whether two Store handles wrap the same underlying
// database (the refcounted pool hands both Opens of the same project path the
// one physical *sql.DB).
func poolSameHandle(a, b Store) bool {
	return a.(*sqliteStore).db == b.(*sqliteStore).db
}

// TestPoolRefcountReleaseClosesOnZero asserts the pool's lifecycle contract
// with a different input shape than TestPoolRefcountSharesAndReleases: two
// Open calls for the same project, closed in the opposite order, must still
// leave the pool empty and must not close the DB out from under a live ref.
func TestPoolRefcountReleaseClosesOnZero(t *testing.T) {
	dir := t.TempDir()
	a, err := Open(context.Background(), dir, "proj-y", SQLite, "")
	if err != nil {
		t.Fatalf("open a: %v", err)
	}
	b, err := Open(context.Background(), dir, "proj-y", SQLite, "")
	if err != nil {
		t.Fatalf("open b: %v", err)
	}
	if !poolSameHandle(a, b) {
		t.Fatalf("expected shared handle for same project path")
	}
	// close in reverse acquisition order
	if err := b.Close(); err != nil {
		t.Fatalf("close b: %v", err)
	}
	if err := a.Close(); err != nil {
		t.Fatalf("close a: %v", err)
	}
	if poolCount() != 0 {
		t.Fatalf("pool must be empty after all refs released, got %d", poolCount())
	}
}
