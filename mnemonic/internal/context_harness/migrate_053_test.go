package context_harness

import (
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func TestMigration053_TablesExist(t *testing.T) {
	st, err := store.Open(t.TempDir(), "ctxproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	for _, table := range []string{"tool_outputs", "tool_outputs_fts"} {
		var n int
		err := st.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&n)
		if err != nil {
			t.Fatalf("query %s: %v", table, err)
		}
		if n != 1 {
			t.Fatalf("table %s not created (count=%d)", table, n)
		}
	}
}

func TestMigration053_FTSInsertTrigger(t *testing.T) {
	st, err := store.Open(t.TempDir(), "ctxproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	_, err = st.DB.Exec(`INSERT INTO tool_outputs (id, session_id, project_id, tool_name, output, size_bytes, created_at)
		VALUES (1, 's1', 'ctxproj', 'bash', 'hello sandbox world', 19, '2026-10-06T00:00:00Z')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	var c int
	err = st.DB.QueryRow(`SELECT count(*) FROM tool_outputs_fts WHERE tool_outputs_fts MATCH 'sandbox'`).Scan(&c)
	if err != nil {
		t.Fatalf("fts match: %v", err)
	}
	if c != 1 {
		t.Fatalf("expected 1 fts hit, got %d", c)
	}
}

func TestMigration053_Idempotent(t *testing.T) {
	st, err := store.Open(t.TempDir(), "ctxproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	st.Close()
	st, err = store.Open(t.TempDir(), "ctxproj")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st.Close()
	var n int
	err = st.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='tool_outputs'").Scan(&n)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected exactly 1 tool_outputs table after reopen, got %d", n)
	}
}
