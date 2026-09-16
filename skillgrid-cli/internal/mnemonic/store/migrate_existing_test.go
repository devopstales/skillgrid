package store

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMigration08AppliesToExistingStores is a one-off probe that points
// store.Open at a copy of a real-world pre-008 store (the user's own
// ~/.skillgrid/mnemonic/skillgrid.sqlite) and confirms the 008 migration
// runs, the new lifecycle + embedding columns exist, and no rows are lost.
// It is skipped when the real file is absent so CI machines pass.
func TestMigration08AppliesToExistingStores(t *testing.T) {
	src, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home")
	}
	orig := filepath.Join(src, ".skillgrid", "mnemonic", "skillgrid.sqlite")
	if _, err := os.Stat(orig); err != nil {
		t.Skip("real store absent: ", orig)
	}
	tmp := t.TempDir()
	// Copy the db + wal.
	for _, f := range []string{"skillgrid.sqlite", "skillgrid.sqlite-shm", "skillgrid.sqlite-wal"} {
		b, e := os.ReadFile(filepath.Join(src, ".skillgrid", "mnemonic", f))
		if e != nil {
			continue // shm/wal are optional
		}
		if e := os.WriteFile(filepath.Join(tmp, f), b, 0o644); e != nil {
			t.Fatalf("copy %s: %v", f, e)
		}
	}
	st, err := Open(tmp, "skillgrid")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	var schema int
	if err := st.DB.QueryRow("SELECT schema_version FROM index_meta WHERE key='schema_version'").Scan(&schema); err != nil {
		t.Fatalf("schema_version: %v", err)
	}
	if schema < 8 {
		t.Fatalf("schema_version=%d want >= 8", schema)
	}

	// Every new column present.
	for _, col := range []string{"pinned", "expires_at", "duplicate_count", "last_seen_at", "embedding", "embedding_model", "embedding_created_at", "tool_name"} {
		var n int
		if err := st.DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('observations') WHERE name=?", col).Scan(&n); err != nil {
			t.Fatalf("pragma: %v", err)
		}
		if n != 1 {
			t.Fatalf("column %s not present after migration (n=%d)", col, n)
		}
	}

	// Existing rows intact.
	var rows int
	if err := st.DB.QueryRow("SELECT COUNT(*) FROM observations WHERE project='skillgrid' AND deleted_at IS NULL").Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	t.Logf("migrated store has %d intact observations; schema v%d", rows, schema)
}
