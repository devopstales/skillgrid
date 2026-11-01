package memfs

import (
	"context"
	"strconv"
	"testing"
)

// TestListStaysCappedUnderScale pins the scale behavior of the code-index
// listing: a store with many indexed files must return at most the LIMIT cap
// (200) rows from `ls`, not the whole set. A regression that drops the LIMIT
// would materialize the entire files table.
func TestListStaysCappedUnderScale(t *testing.T) {
	fs, st := newCodeFSTestFixture(t, "code-scale-test")
	ctx := context.Background()
	db := st.DB

	// 250 files across 5 top-level dirs (src/, lib/, cmd/, pkg/, docs/).
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	for i := 0; i < 250; i++ {
		dir := []string{"src", "lib", "cmd", "pkg", "docs"}[i%5]
		if _, err := tx.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES (?, 1, 10, ?, '2026-01-01T00:00:00Z')`,
			dir+"/f"+strconv.Itoa(i)+".go", "h"+strconv.Itoa(i)); err != nil {
			tx.Rollback()
			t.Fatalf("insert file %d: %v", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Root listing is capped at LIMIT 200 file rows; with 250 files across 5
	// dirs the 5th dir can fall off the cap. Assert it returns at most the cap
	// and at least the dirs fully within the first 200 rows.
	res, err := fs.List(ctx, "")
	if err != nil {
		t.Fatalf("ls root at scale: %v", err)
	}
	if len(res) > 200 {
		t.Errorf("ls root at scale returned %d, want at most 200", len(res))
	}
	if len(res) < 4 {
		t.Errorf("ls root at scale returned %d top-level dirs, want >= 4", len(res))
	}

	// A directory with 50 files lists all 50 (under cap).
	res, err = fs.List(ctx, "src/")
	if err != nil {
		t.Fatalf("ls src/ at scale: %v", err)
	}
	if len(res) != 50 {
		t.Errorf("ls src/ at scale: expected 50 files, got %d", len(res))
	}

	// Find *.go is capped at 200 even though 250 files match.
	res, err = fs.Find(ctx, "*.go", "")
	if err != nil {
		t.Fatalf("find *.go at scale: %v", err)
	}
	if len(res) > 200 {
		t.Errorf("find *.go at scale returned %d rows, want at most 200 (LIMIT cap)", len(res))
	}
}
