package store

import (
	"database/sql"
	"testing"
)

// TestStoreOpenAppliesGovernanceSchema covers @step-01 (additive 017 migration):
// a store that opens cleanly has the governance columns, the 017 migration is
// recorded once, and the 005 observations table + FTS remain intact
// (additive, not a rewrite).
func TestStoreOpenAppliesGovernanceSchema(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "govproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	// Governance columns present on the existing observations table.
	for _, col := range []string{"owner", "status", "visibility", "retrieval_usage"} {
		var n int
		if err := st.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('observations') WHERE name=?`, col).Scan(&n); err != nil {
			t.Fatalf("pragma %s: %v", col, err)
		}
		if n != 1 {
			t.Fatalf("governance column %s not present after 017 (n=%d)", col, n)
		}
	}

	// New tables present.
	for _, name := range []string{"observation_versions", "acl_grants"} {
		if !tableExists(t, st.DB, name) {
			t.Fatalf("expected 017 table %s after open", name)
		}
	}
	if countMigration(t, st.DB, "017_layered_memory_governance.sql") != 1 {
		t.Fatalf("expected 017 migration recorded once")
	}

	// 005 tables intact (additive — never rewritten).
	for _, name := range []string{"observations", "observations_fts", "sessions"} {
		if !tableExists(t, st.DB, name) {
			t.Fatalf("005 table %s missing after 017", name)
		}
	}

	// Defaults are private-by-default + active status.
	id := seedObservation(t, st, "gov-default")
	var owner sql.NullString
	var status, visibility string
	var usage int
	if err := st.DB.QueryRow(`
		SELECT owner, status, visibility, retrieval_usage
		FROM observations WHERE id = ?`, id).Scan(&owner, &status, &visibility, &usage); err != nil {
		t.Fatalf("read governance defaults: %v", err)
	}
	if status != "active" {
		t.Errorf("expected default status 'active', got %q", status)
	}
	if visibility != "private" {
		t.Errorf("expected default visibility 'private', got %q", visibility)
	}
	if usage != 0 {
		t.Errorf("expected default retrieval_usage 0, got %d", usage)
	}
	_ = owner
}

// TestGovernanceSchemaUpgradeFromLegacy covers @step-01 edge: an older store
// (pre-017, no governance columns) upgrades additively and keeps its rows.
func TestGovernanceSchemaUpgradeFromLegacy(t *testing.T) {
	dir := t.TempDir()
	db := rawSQLDB(t, dir, "legacygov")
	// A pre-017 store: 005 observations without the governance columns.
	if _, err := db.Exec(`
		CREATE TABLE sessions (
		    id TEXT PRIMARY KEY, project TEXT NOT NULL, directory TEXT NOT NULL,
		    started_at TEXT NOT NULL, ended_at TEXT, summary TEXT,
		    status TEXT NOT NULL DEFAULT 'active');
		CREATE TABLE observations (
		    id INTEGER PRIMARY KEY AUTOINCREMENT,
		    session_id TEXT NOT NULL REFERENCES sessions(id),
		    type TEXT NOT NULL, title TEXT NOT NULL, content TEXT NOT NULL,
		    project TEXT, scope TEXT, topic_key TEXT, normalized_hash TEXT,
		    revision_count INTEGER NOT NULL DEFAULT 0,
		    created_at TEXT NOT NULL, updated_at TEXT NOT NULL, deleted_at TEXT);
	`); err != nil {
		t.Fatalf("legacy schema: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1','legacygov','/tmp','2026-01-01T00:00:00Z','active')`); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO observations (session_id, type, title, content, project, created_at, updated_at)
		VALUES ('s1','decision','keep-me','body','legacygov','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("seed obs: %v", err)
	}
	db.Close()

	st, err := Open(dir, "legacygov")
	if err != nil {
		t.Fatalf("open legacy: %v", err)
	}
	defer st.Close()
	for _, col := range []string{"owner", "status", "visibility", "retrieval_usage"} {
		if !tableExists(t, st.DB, "observations") {
			t.Fatalf("observations missing after upgrade")
		}
		var n int
		if err := st.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('observations') WHERE name=?`, col).Scan(&n); err != nil {
			t.Fatalf("pragma %s: %v", col, err)
		}
		if n != 1 {
			t.Fatalf("column %s not present after legacy upgrade", col)
		}
	}
	// Existing row intact + defaulted to private/active.
	var status, visibility, title string
	if err := st.DB.QueryRow(`
		SELECT status, visibility, title FROM observations WHERE title='keep-me'`).Scan(&status, &visibility, &title); err != nil {
		t.Fatalf("read upgraded row: %v", err)
	}
	if title != "keep-me" {
		t.Fatalf("row rewritten: title=%q", title)
	}
	if status != "active" || visibility != "private" {
		t.Errorf("upgraded row should default to active/private, got %s/%s", status, visibility)
	}
}
