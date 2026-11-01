package store

import (
	"strings"
	"testing"
)

// TestMigration059FTSStandalone verifies the 059 conversion: observations_fts
// is a standalone FTS5 table (no `content=` clause), the legacy AFTER triggers
// are dropped, and the index is rebuilt from observations_core. A standalone
// table does not reconcile against a content table, so the FTS5 'delete'
// command (unsupported by modernc.org/sqlite on standalone tables) is no longer
// needed — the index is maintained by DELETE + rebuild / UPDATE in place.
func TestMigration059FTSStandalone(t *testing.T) {
	st := openTestStore(t)

	// 1. The FTS table is standalone: its CREATE statement has no content=.
	var sqlText string
	if err := st.DB.QueryRow(
		`SELECT sql FROM sqlite_master WHERE name = 'observations_fts'`,
	).Scan(&sqlText); err != nil {
		t.Fatalf("fts sql: %v", err)
	}
	if strings.Contains(sqlText, "content=") {
		t.Errorf("observations_fts must be standalone (no content=), got: %s", sqlText)
	}
	if !strings.Contains(sqlText, "fts5") {
		t.Errorf("observations_fts must be an fts5 table, got: %s", sqlText)
	}

	// 2. The legacy AFTER triggers on the view are gone (dead on a view anyway;
	//    they would corrupt the standalone index on the 'delete' command).
	for _, trig := range []string{
		"observations_fts_insert", "observations_fts_update", "observations_fts_delete",
	} {
		var n int
		if err := st.DB.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE name = ? AND type = 'trigger'`, trig,
		).Scan(&n); err != nil {
			t.Fatalf("trigger %s: %v", trig, err)
		}
		if n != 0 {
			t.Errorf("legacy trigger %s must be dropped by 059", trig)
		}
	}

	// 3. The rebuild populated the index from observations_core: a seeded row
	//    is searchable via FTS MATCH (no content-table reconcile).
	now := "2026-01-01T00:00:00Z"
	if _, err := st.DB.Exec(
		`INSERT INTO sessions (id, project, directory, started_at, status) VALUES ('s1','test-project','/tmp',?, 'active')`, now,
	); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	// Seed through the compat VIEW: the INSTEAD OF INSERT trigger lands core +
	// the FTS row (restored after 059 made the table standalone).
	if _, err := st.DB.Exec(
		`INSERT INTO observations (session_id, type, title, content, project, scope, normalized_hash, revision_count, created_at, updated_at)
		 VALUES ('s1','decision','fts-standalone-probe','probe body terms','test-project','project','h-059',0,?,?)`, now, now,
	); err != nil {
		t.Fatalf("seed observation: %v", err)
	}
	var matched int
	if err := st.DB.QueryRow(
		`SELECT COUNT(*) FROM observations_fts WHERE observations_fts MATCH ?`, `"fts-standalone-probe"`,
	).Scan(&matched); err != nil {
		t.Fatalf("fts match: %v", err)
	}
	if matched != 1 {
		t.Fatalf("seeded row must be FTS-searchable after 059, matched %d", matched)
	}

	// 4. The index is integrity-clean (a standalone table's 'integrity-check'
	//    returns no rows on success; a corrupt index errors).
	if _, err := st.DB.Exec(
		`INSERT INTO observations_fts(observations_fts) VALUES ('integrity-check')`,
	); err != nil {
		t.Fatalf("fts integrity-check: %v", err)
	}
}

// TestMigration059FTSRebuildSurvivesUpdate exercises the owned-write FTS path
// (ftsReplaceObs: UPDATE in place) and the delete path (ftsDeleteObs: DELETE +
// rebuild) against the standalone table — the two operations that replaced the
// unsupported FTS5 'delete' command.
func TestMigration059FTSRebuildSurvivesUpdate(t *testing.T) {
	st := openTestStore(t)
	now := "2026-01-01T00:00:00Z"
	if _, err := st.DB.Exec(
		`INSERT INTO sessions (id, project, directory, started_at, status) VALUES ('s1','test-project','/tmp',?, 'active')`, now,
	); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	res, err := st.DB.Exec(
		`INSERT INTO observations_core (session_id, type, title, content, project, scope, normalized_hash, revision_count, created_at, updated_at)
		 VALUES ('s1','decision','update-probe','before update body','test-project','project','h-upd',0,?,?)`, now, now,
	)
	if err != nil {
		t.Fatalf("seed core: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last id: %v", err)
	}
	if _, err := st.DB.Exec(
		`INSERT INTO observations_fts(rowid, title, content, type, project) VALUES (?, 'update-probe', 'before update body', 'decision', 'test-project')`, id,
	); err != nil {
		t.Fatalf("seed fts: %v", err)
	}

	// Update-in-place (ftsReplaceObs): the old terms leave the index, the new
	// terms enter, and the row count stays 1.
	if _, err := st.DB.Exec(
		`UPDATE observations_fts SET content = ? WHERE rowid = ?`, "after update body", id,
	); err != nil {
		t.Fatalf("fts update: %v", err)
	}
	var before, after int
	st.DB.QueryRow(`SELECT COUNT(*) FROM observations_fts WHERE observations_fts MATCH ?`, "before").Scan(&before)
	st.DB.QueryRow(`SELECT COUNT(*) FROM observations_fts WHERE observations_fts MATCH ?`, "after").Scan(&after)
	if before != 0 {
		t.Errorf("old FTS terms must be gone after update-in-place, 'before' matched %d", before)
	}
	if after != 1 {
		t.Errorf("new FTS terms must be present after update-in-place, 'after' matched %d", after)
	}

	// Delete + rebuild (ftsDeleteObs): the row leaves the index entirely and
	// the rebuild leaves the index intact for other rows.
	if _, err := st.DB.Exec(`DELETE FROM observations_fts WHERE rowid = ?`, id); err != nil {
		t.Fatalf("fts delete row: %v", err)
	}
	if _, err := st.DB.Exec(`INSERT INTO observations_fts(observations_fts) VALUES ('rebuild')`); err != nil {
		t.Fatalf("fts rebuild: %v", err)
	}
	var gone int
	st.DB.QueryRow(`SELECT COUNT(*) FROM observations_fts WHERE observations_fts MATCH ?`, "after").Scan(&gone)
	if gone != 0 {
		t.Errorf("deleted row's FTS terms must be gone after delete+rebuild, 'after' matched %d", gone)
	}
	if _, err := st.DB.Exec(`INSERT INTO observations_fts(observations_fts) VALUES ('integrity-check')`); err != nil {
		t.Fatalf("post-delete integrity-check: %v", err)
	}
}
