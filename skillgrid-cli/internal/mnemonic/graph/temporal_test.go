package graph

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// graphTestDB opens a raw *sql.DB over a temp SQLite file, applies the REAL
// embedded migrations (so the 023 valid_from/valid_to columns exist) and
// allows several open connections. fetchEdges re-enters the same pool from a
// nested per-edge query (loadSymbolByID), and the store's single-connection
// pool (SetMaxOpenConns(1)) deadlocks on that, so the fixture keeps its own
// pool instead of going through store.Open.
func graphTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "graph.sqlite"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(8)
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=10000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			t.Fatalf("pragma %s: %v", pragma, err)
		}
	}
	if err := store.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// temporalSeed plants one file, two symbols, and one calls edge per (kind,
// line, validFrom, validTo) — all endpoints by-id, so the temporal filter is
// the only thing that can hide an edge.
func temporalSeed(t *testing.T, db *sql.DB, edges ...edgeSpec) (fromID int64) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('/t/temporal.go', 1, 1, 't', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'fromSym', 'fromSym', 'function', 'go', 'func fromSym()', 1, 5, 'h', 'uid-from'
		FROM files f WHERE f.path = '/t/temporal.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'toSym', 'toSym', 'function', 'go', 'func toSym()', 6, 10, 'h', 'uid-to'
		FROM files f WHERE f.path = '/t/temporal.go';
	`); err != nil {
		t.Fatalf("seed symbols: %v", err)
	}
	for _, e := range edges {
		var vt any
		if e.validTo != 0 {
			vt = e.validTo
		}
		if _, err := db.ExecContext(ctx, `
			INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line, valid_from, valid_to)
			SELECT ?, s.id, f.id, (SELECT id FROM symbols WHERE uid = 'uid-to'), NULL, 'toSym', 'EXTRACTED', ?, ?, ?
			FROM symbols s, files f WHERE s.uid = 'uid-from' AND f.path = '/t/temporal.go'`,
			e.kind, e.line, e.validFrom, vt); err != nil {
			t.Fatalf("seed edge line %d: %v", e.line, err)
		}
	}
	if err := db.QueryRowContext(ctx, `SELECT id FROM symbols WHERE uid = 'uid-from'`).Scan(&fromID); err != nil {
		t.Fatalf("lookup from symbol: %v", err)
	}
	return fromID
}

// edgeSpec is one seed edge: a distinct (kind, line) keeps the rows outside
// the unique constraint even when only the temporal bounds differ.
type edgeSpec struct {
	kind      string
	line      int
	validFrom int64
	validTo   int64
}

// TestFetchEdgesHidesExpiredEdges is 10.2 fix (review F1) [RED] — the
// canonical traversal (fetchEdges, behind Neighbors/Explain/Path) applies the
// temporal filter: an active edge is returned, an expired edge (valid_to in
// the past) and a pending edge (valid_from in the future) are hidden.
func TestFetchEdgesHidesExpiredEdges(t *testing.T) {
	db := graphTestDB(t)
	now := time.Now().Unix()
	fromID := temporalSeed(t, db,
		edgeSpec{kind: "calls", line: 10, validFrom: now - 10000, validTo: 0},    // active
		edgeSpec{kind: "calls", line: 20, validFrom: now - 10000, validTo: now - 100}, // expired
		edgeSpec{kind: "calls", line: 30, validFrom: now + 10000, validTo: 0},    // pending
	)
	from := Symbol{ID: fromID, Name: "fromSym"}
	ctx := context.Background()

	edges, err := fetchEdges(ctx, db, from)
	if err != nil {
		t.Fatalf("fetchEdges: %v", err)
	}
	byLine := map[int]string{}
	for _, e := range edges {
		byLine[e.Line] = e.Kind
	}
	if len(byLine) != 1 {
		t.Fatalf("expected exactly 1 active edge, got %d: %v", len(byLine), byLine)
	}
	if _, ok := byLine[10]; !ok {
		t.Fatalf("active edge (line 10) missing from traversal: %v", byLine)
	}
	if _, ok := byLine[20]; ok {
		t.Fatalf("expired edge (line 20) leaked into the traversal")
	}
	if _, ok := byLine[30]; ok {
		t.Fatalf("pending edge (line 30) leaked into the traversal")
	}

	// The same filter must hold through the public Neighbors API (the code_*
	// tools' path).
	nb, err := Neighbors(ctx, db, from, ViewCallees)
	if err != nil {
		t.Fatalf("Neighbors: %v", err)
	}
	if len(nb) != 1 || nb[0].Line != 10 {
		t.Fatalf("Neighbors must return only the active edge, got %+v", nb)
	}
}

// TestBackfilledEdgesVisibleInTraversal guards the migration backfill on the
// traversal path: an edge with valid_from = 0 (pre-migration) passes
// valid_from <= now and stays in the traversal.
func TestBackfilledEdgesVisibleInTraversal(t *testing.T) {
	db := graphTestDB(t)
	fromID := temporalSeed(t, db,
		edgeSpec{kind: "calls", line: 10, validFrom: 0, validTo: 0}, // backfilled
	)
	ctx := context.Background()

	edges, err := fetchEdges(ctx, db, Symbol{ID: fromID, Name: "fromSym"})
	if err != nil {
		t.Fatalf("fetchEdges: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("backfilled (valid_from=0) edge must be in the traversal, got %d edges", len(edges))
	}
}

// TestPromotedSessionEdgesCarryValidFrom is the F2 guard for the step-09
// session-promotion edge path: a 'promotes' edge inserted WITHOUT an explicit
// valid_from (the old INSERT shape) must still carry the migration default 0
// and remain visible in the traversal — i.e. wiring valid_from into the
// promotion INSERT does not retroactively hide any promoted edge.
func TestPromotedSessionEdgesCarryValidFrom(t *testing.T) {
	db := graphTestDB(t)
	ctx := context.Background()

	// A session-like symbol + a pseudo-file, mirroring step 09's shape.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('session:abc', 1, 1, 's', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, '[session abc] Session summary', '[session abc] Session summary', 'session', 'session', 'summary', 1, 1, 'h', 'session:abc'
		FROM files f WHERE f.path = 'session:abc';
	`); err != nil {
		t.Fatalf("seed session symbol: %v", err)
	}
	var nodeID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM symbols WHERE uid = 'session:abc'`).Scan(&nodeID); err != nil {
		t.Fatalf("lookup node: %v", err)
	}
	// The old promotion INSERT (no valid_from column): must still work and
	// default to 0.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line)
		VALUES ('promotes', ?, (SELECT id FROM files WHERE path = 'session:abc'), NULL, 42, 'obs:42', 'EXTRACTED', 0)`,
		nodeID,
	); err != nil {
		t.Fatalf("insert promotion edge: %v", err)
	}
	var vf int64
	if err := db.QueryRowContext(ctx, `SELECT valid_from FROM edges WHERE kind = 'promotes'`).Scan(&vf); err != nil {
		t.Fatalf("read valid_from: %v", err)
	}
	if vf != 0 {
		t.Fatalf("insert without valid_from must default to 0, got %d", vf)
	}
	// And the edge must be traversable (backfilled rows stay active).
	edges, err := fetchEdges(ctx, db, Symbol{ID: nodeID, Name: "session"})
	if err != nil {
		t.Fatalf("fetchEdges: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("promotion edge must be visible in the traversal, got %d edges", len(edges))
	}
}
