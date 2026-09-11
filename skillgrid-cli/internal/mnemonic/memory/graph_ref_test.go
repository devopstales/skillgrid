package memory

import (
	"context"
	"database/sql"
	"testing"
)

// graphRefOf reads the raw graph_ref column for an observation (NULL-safe).
func graphRefOf(t *testing.T, db *sql.DB, obsID int64) sql.NullInt64 {
	t.Helper()
	var ref sql.NullInt64
	if err := db.QueryRow(`SELECT graph_ref FROM observations WHERE id = ?`, obsID).Scan(&ref); err != nil {
		t.Fatalf("scan graph_ref for %d: %v", obsID, err)
	}
	return ref
}

// orphanGraphRefs returns the graph_ref values that do not resolve to a live
// codeindex symbol — the orphan set CheckCrossLinkIntegrity must report.
func orphanGraphRefs(t *testing.T, db *sql.DB) []int64 {
	t.Helper()
	rows, err := db.Query(`
		SELECT o.graph_ref
		FROM observations o
		WHERE o.graph_ref IS NOT NULL
		  AND NOT EXISTS (SELECT 1 FROM symbols s WHERE s.id = o.graph_ref)`)
	if err != nil {
		t.Fatalf("orphan query: %v", err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var ref int64
		if err := rows.Scan(&ref); err != nil {
			t.Fatalf("scan orphan: %v", err)
		}
		out = append(out, ref)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}

// seedSymbolFile inserts a files row plus one symbol row, returning the
// symbol id. Used by the graph_ref tests so they do not run the full indexer.
func seedSymbolFile(t *testing.T, db *sql.DB, path, name, uid string) int64 {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES (?, 1, 10, ?, '2026-01-01T00:00:00Z')`, path, "h-"+uid); err != nil {
		t.Fatalf("insert file %s: %v", path, err)
	}
	var fileID int64
	if err := db.QueryRow(`SELECT id FROM files WHERE path = ?`, path).Scan(&fileID); err != nil {
		t.Fatalf("lookup file %s: %v", path, err)
	}
	if _, err := db.Exec(`
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		VALUES (?, ?, ?, 'function', 'go', 'func '+?+'()', 1, 2, ?, ?)`,
		fileID, name, name, name, "ch-"+uid, uid); err != nil {
		t.Fatalf("insert symbol %s: %v", name, err)
	}
	var symbolID int64
	if err := db.QueryRow(`SELECT id FROM symbols WHERE uid = ?`, uid).Scan(&symbolID); err != nil {
		t.Fatalf("lookup symbol %s: %v", uid, err)
	}
	return symbolID
}

// TestGraphRefDefaultsToNull covers @step-07: observations carry an optional
// graph_ref column linking them to a codeindex symbol. A save whose source
// file has no indexed symbol leaves graph_ref NULL (best-effort, never an
// error); a save for a file with an indexed symbol sets graph_ref to that
// symbol id; no broken references remain.
func TestGraphRefDefaultsToNull(t *testing.T) {
	fx := newFixture(t, "graphref")
	ctx := context.Background()
	db := fx.st.DB

	// A source file with no indexed symbols.
	if _, err := db.Exec(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('/tmp/orphan.go', 1, 10, 'h-orphan', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("insert orphan file: %v", err)
	}
	// A source file with an indexed symbol.
	symbolID := seedSymbolFile(t, db, "/tmp/linked.go", "linkedFunc", "uid-linked-1")

	// Save for the symbol-less file: graph_ref must be NULL.
	obsNoSym, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID,
		Type:      "decision",
		Title:     "orphan source note",
		Content:   "observation from a file with no indexed symbol",
		Source:    "/tmp/orphan.go",
	})
	if err != nil {
		t.Fatalf("save orphan: %v", err)
	}
	if ref := graphRefOf(t, db, obsNoSym); ref.Valid {
		t.Fatalf("graph_ref should be NULL for a source file with no symbol, got %d", ref.Int64)
	}

	// Save for the file with a symbol: graph_ref must be that symbol id.
	obsWithSym, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID,
		Type:      "decision",
		Title:     "linked source note",
		Content:   "observation from a file with an indexed symbol",
		Source:    "/tmp/linked.go",
	})
	if err != nil {
		t.Fatalf("save linked: %v", err)
	}
	ref := graphRefOf(t, db, obsWithSym)
	if !ref.Valid || ref.Int64 != symbolID {
		t.Fatalf("graph_ref=%v want %d", ref, symbolID)
	}

	// No broken references: every non-NULL graph_ref resolves to a symbol.
	if orphans := orphanGraphRefs(t, db); len(orphans) != 0 {
		t.Fatalf("expected no broken graph_ref references, got %v", orphans)
	}
}

// TestTripleStoreCrossLinkQuery covers @step-07: the opt-in cross-link query
// traverses observation -> symbol (graph_ref) -> related observations through
// a single SQL JOIN. Observations that share a symbol are surfaced; the input
// observation is never returned for itself.
func TestTripleStoreCrossLinkQuery(t *testing.T) {
	fx := newFixture(t, "crosklink")
	ctx := context.Background()
	db := fx.st.DB

	symbolID := seedSymbolFile(t, db, "/tmp/shared.go", "sharedFunc", "uid-shared-1")

	obsA, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID,
		Type:      "decision",
		Title:     "cross-link note A",
		Content:   "first observation bound to the shared symbol",
		Source:    "/tmp/shared.go",
	})
	if err != nil {
		t.Fatalf("save A: %v", err)
	}
	obsB, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID,
		Type:      "discovery",
		Title:     "cross-link note B",
		Content:   "second observation bound to the same shared symbol",
		Source:    "/tmp/shared.go",
	})
	if err != nil {
		t.Fatalf("save B: %v", err)
	}

	// Sanity: both observations carry the symbol as graph_ref.
	if ref := graphRefOf(t, db, obsA); !ref.Valid || ref.Int64 != symbolID {
		t.Fatalf("obs A graph_ref=%v want %d", ref, symbolID)
	}
	if ref := graphRefOf(t, db, obsB); !ref.Valid || ref.Int64 != symbolID {
		t.Fatalf("obs B graph_ref=%v want %d", ref, symbolID)
	}

	// Cross-link from A must return B (and only B) via the SQL JOIN path.
	related, err := fx.svc.CrossLinkQuery(ctx, obsA)
	if err != nil {
		t.Fatalf("cross link: %v", err)
	}
	if len(related) != 1 || related[0].ID != obsB {
		t.Fatalf("cross-link from A returned %v, want only B (%d)", related, obsB)
	}
}

// TestTripleStoreOrphanDetection covers @step-07: CheckCrossLinkIntegrity
// reports observations whose graph_ref points at a symbol that no longer
// exists, and lets valid references pass.
func TestTripleStoreOrphanDetection(t *testing.T) {
	fx := newFixture(t, "orphancheck")
	ctx := context.Background()
	db := fx.st.DB

	symbolID := seedSymbolFile(t, db, "/tmp/orphan.go", "orphanFunc", "uid-orphan-1")

	// A valid reference (graph_ref resolves to a live symbol).
	obsValid, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID,
		Type:      "decision",
		Title:     "valid ref note",
		Content:   "observation bound to a live symbol",
		Source:    "/tmp/orphan.go",
	})
	if err != nil {
		t.Fatalf("save valid: %v", err)
	}

	// An orphan reference: point graph_ref at a symbol id that does not exist.
	_, err = db.Exec(`UPDATE observations SET graph_ref = 987654 WHERE id = ?`, obsValid)
	if err != nil {
		t.Fatalf("set orphan graph_ref: %v", err)
	}
	// And a second observation still holding the valid reference.
	obsGood, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID,
		Type:      "decision",
		Title:     "good ref note",
		Content:   "observation bound to the same live symbol",
		Source:    "/tmp/orphan.go",
	})
	if err != nil {
		t.Fatalf("save good: %v", err)
	}

	orphans, err := fx.svc.CheckCrossLinkIntegrity(ctx)
	if err != nil {
		t.Fatalf("integrity check: %v", err)
	}
	if len(orphans) != 1 || orphans[0].ID != obsValid || orphans[0].GraphRef != 987654 {
		t.Fatalf("orphans=%+v want exactly the 987654 reference (obs %d)", orphans, obsValid)
	}
	for _, o := range orphans {
		if o.ID == obsGood {
			t.Fatalf("valid reference (obs %d, symbol %d) reported as orphan", obsGood, symbolID)
		}
	}
}
