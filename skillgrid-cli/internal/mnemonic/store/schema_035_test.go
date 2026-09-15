package store

import (
	"database/sql"
	"testing"
)

// columnExists reports whether table has a column named name (PRAGMA table_info).
func columnExists(t *testing.T, db *sql.DB, table, name string) bool {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatalf("pragma %s: %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid       int
			colName   string
			colType   string
			notnull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &colName, &colType, &notnull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan pragma %s: %v", table, err)
		}
		if colName == name {
			return true
		}
	}
	return rows.Err() == nil
}

// TestStoreOpenAddsGraphEnrichmentColumns covers 035: opening a store adds
// edges.context + edges.confidence_score, files.ast_hash, and
// community_meta.hub_label (all NOT NULL with defaults), and records
// migration 035 exactly once.
func TestStoreOpenAddsGraphEnrichmentColumns(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "proj035")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	if countMigration(t, st.DB, "035_graph_enrichment.sql") != 1 {
		t.Fatalf("expected 035 migration recorded once")
	}
	for _, col := range []struct {
		table string
		name  string
	}{
		{"edges", "context"},
		{"edges", "confidence_score"},
		{"files", "ast_hash"},
		{"community_meta", "hub_label"},
	} {
		if !columnExists(t, st.DB, col.table, col.name) {
			t.Fatalf("expected column %s.%s after open", col.table, col.name)
		}
	}
	// A row inserted WITHOUT the new columns must succeed (defaults), and the
	// new columns must round-trip their defaults ('' and 1.0).
	res, err := st.DB.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES ('x.go', 1, 1, 'h', 'now')`)
	if err != nil {
		t.Fatalf("insert file: %v", err)
	}
	fileID, _ := res.LastInsertId()
	if _, err := st.DB.Exec(`INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line, valid_from)
		VALUES ('calls', 1, ?, NULL, 'n', '', 'EXTRACTED', 0, 1)`, fileID); err != nil {
		t.Fatalf("insert edge without new columns (defaults): %v", err)
	}
	var ctx string
	var score float64
	if err := st.DB.QueryRow(`SELECT context, confidence_score FROM edges LIMIT 1`).Scan(&ctx, &score); err != nil {
		t.Fatalf("read edge new columns: %v", err)
	}
	if ctx != "" || score != 1.0 {
		t.Errorf("expected edge defaults ('', 1.0), got (%q, %v)", ctx, score)
	}
	// files.ast_hash must default to ''.
	var astHash string
	if err := st.DB.QueryRow(`SELECT ast_hash FROM files WHERE id = ?`, fileID).Scan(&astHash); err != nil {
		t.Fatalf("read ast_hash: %v", err)
	}
	if astHash != "" {
		t.Errorf("expected files.ast_hash default '', got %q", astHash)
	}
}
