package store

import (
	"testing"
)

func TestFKProbe(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "fk-probe")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	var on int
	if err := st.DB.QueryRow(`PRAGMA foreign_keys`).Scan(&on); err != nil {
		t.Fatalf("pragma: %v", err)
	}
	t.Logf("foreign_keys = %d", on)
	if _, err := st.DB.Exec(`INSERT INTO observations (session_id, type, title, content, project, scope, created_at, updated_at) VALUES ('nope','decision','t','c','fk-probe','project','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`); err != nil {
		t.Logf("insert with missing session: %v", err)
	} else {
		t.Log("insert with missing session SUCCEEDED (FKs off on this connection)")
	}
}
