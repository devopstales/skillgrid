package store

import (
	"testing"
)

// TestMigration068TrafficMemoryStatus verifies the 068 migration on a fresh
// store: ctx_traffic_memories gains status (default candidate) + applied_at,
// a re-open does not re-run the migration (a duplicate ADD COLUMN would error
// if it did), and an insert picks up the column defaults.
func TestMigration068TrafficMemoryStatus(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir, "mig068")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('ctx_traffic_memories') WHERE name IN ('status','applied_at')`).Scan(&n); err != nil {
		t.Fatalf("probe columns: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected status + applied_at, got %d", n)
	}

	// Insert without the new columns: defaults must fill candidate/NULL.
	if _, err := st.DB.Exec(`
		INSERT INTO ctx_traffic_memories (memory_id, session_id, category, memory_text, confidence, created_at)
		VALUES ('mem-1', 'sess-1', 'preference', 'compact diffs', 0.5, 1000)`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	var status string
	var appliedAt *int64
	if err := st.DB.QueryRow(`SELECT status, applied_at FROM ctx_traffic_memories WHERE memory_id = 'mem-1'`).Scan(&status, &appliedAt); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if status != "candidate" {
		t.Fatalf("status = %q, want candidate", status)
	}
	if appliedAt != nil {
		t.Fatalf("applied_at = %d, want NULL", *appliedAt)
	}

	// Re-open the same store: 068 is recorded, so the ADD COLUMNs must not
	// re-fire (a duplicate-column error would fail the open).
	st2, err := Open(dataDir, "mig068")
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	st2.Close()
}
