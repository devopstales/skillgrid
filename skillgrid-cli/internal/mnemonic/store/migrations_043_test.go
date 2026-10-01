package store

import (
	"testing"
	"time"
)

// TestMigration043LifecycleLog covers 043 (second-brain lifecycle audit):
// (a) observations gains archive_reason + archived_at; (b) lifecycle_log
// exists with an index; (c) status defaults to 'pending'; (d) migration
// recorded once, re-open idempotent.
func TestMigration043LifecycleLog(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "lifecycleproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	for _, col := range []string{"archive_reason", "archived_at"} {
		if !columnExists(t, st.DB, "observations", col) {
			t.Errorf("observations missing column %s", col)
		}
	}
	if !tableExists(t, st.DB, "lifecycle_log") {
		t.Fatalf("missing table lifecycle_log")
	}
	if countMigration(t, st.DB, "043_lifecycle_log.sql") != 1 {
		t.Fatalf("expected 043 migration recorded once")
	}

	// status defaults to 'pending' when omitted.
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT INTO lifecycle_log (project, action, created_at)
		VALUES ('lifecycleproj', 'archive', ?)`, now); err != nil {
		t.Fatalf("insert lifecycle row: %v", err)
	}
	var status string
	if err := st.DB.QueryRow(
		`SELECT status FROM lifecycle_log WHERE action='archive'`).Scan(&status); err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != "pending" {
		t.Errorf("status = %q, want pending", status)
	}

	// The index exists.
	var n int
	if err := st.DB.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_lifecycle_log_project'`).Scan(&n); err != nil {
		t.Fatalf("index: %v", err)
	}
	if n != 1 {
		t.Error("expected idx_lifecycle_log_project to exist")
	}

	// Re-open is idempotent: migration still recorded once, row intact.
	st.Close()
	st2, err := Open(dir, "lifecycleproj")
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer st2.Close()
	if countMigration(t, st2.DB, "043_lifecycle_log.sql") != 1 {
		t.Fatalf("043 applied more than once")
	}
	var rows int
	if err := st2.DB.QueryRow(`SELECT COUNT(*) FROM lifecycle_log`).Scan(&rows); err != nil {
		t.Fatalf("count lifecycle rows: %v", err)
	}
	if rows != 1 {
		t.Fatalf("lifecycle rows lost after re-open: got %d, want 1", rows)
	}
}
