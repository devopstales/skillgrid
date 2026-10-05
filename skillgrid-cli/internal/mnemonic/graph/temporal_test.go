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

// TestFetchEdgesSingleConnection is the RED guard for the fetchEdges deadlock:
// the production store pool is single-connection (store.Open ->
// SetMaxOpenConns(1)), and fetchEdges must not re-enter that pool with a
// per-edge query while its outer edge rows are still open. The old shape
// (loadSymbolByID inside `for rows.Next()`) deadlocks here —
// "all goroutines are asleep" — because the open *sql.Rows holds the only
// connection. This test uses store.Open (MaxOpenConns(1)) on purpose so the
// regression is caught at the real pool size, not masked by graphTestDB's
// MaxOpenConns(8). It guards every code_* neighbor/explain/path caller, which
// all route through fetchEdges.
func TestFetchEdgesSingleConnection(t *testing.T) {
	st, err := store.Open(t.TempDir(), "sc")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	ctx := context.Background()
	db := st.DB

	// Two files, two symbols, one active by-id edge and one active name-only
	// edge — both endpoints exercise the per-edge loadSymbolByID /
	// loadSymbolsByName nested queries while the outer rows are open.
	seed := `
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES
			('/x/from.go', 1, 1, 'x', 'now'),
			('/y/to.go', 2, 2, 'y', 'now'),
			('/z/amb.go', 3, 3, 'z', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'scFrom', 'scFrom', 'function', 'go', 'func scFrom()', 1, 5, 'h', 'uid-sc-from' FROM files f WHERE f.path='/x/from.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'scTo', 'scTo', 'function', 'go', 'func scTo()', 6, 10, 'h', 'uid-sc-to' FROM files f WHERE f.path='/y/to.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'scAmb', 'scAmb', 'function', 'go', 'func scAmb()', 1, 5, 'h', 'uid-sc-amb' FROM files f WHERE f.path='/z/amb.go';
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line, valid_from)
		SELECT 'calls',
			(SELECT id FROM symbols WHERE uid='uid-sc-from'),
			(SELECT file_id FROM symbols WHERE uid='uid-sc-from'),
			(SELECT id FROM symbols WHERE uid='uid-sc-to'),
			'scTo', 'EXTRACTED', 10, 0;
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line, valid_from)
		SELECT 'calls',
			(SELECT id FROM symbols WHERE uid='uid-sc-from'),
			(SELECT file_id FROM symbols WHERE uid='uid-sc-from'),
			NULL, 'scAmb', 'EXTRACTED', 20, 0;
	`
	if _, err := db.ExecContext(ctx, seed); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var fromID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM symbols WHERE uid='uid-sc-from'`).Scan(&fromID); err != nil {
		t.Fatalf("lookup from: %v", err)
	}

	edges, err := fetchEdges(ctx, db, Symbol{ID: fromID, Name: "scFrom"})
	if err != nil {
		t.Fatalf("fetchEdges (MaxOpenConns=1): %v", err)
	}
	// One by-id edge (scTo) + one name-only edge (scAmb, single candidate) = 2.
	if len(edges) != 2 {
		t.Fatalf("expected 2 resolved edges, got %d: %+v", len(edges), edges)
	}
}

// TestFetchEdgesDanglingFromID is the RED guard for dangling from_id: the live
// kubedash store has hundreds of edges whose from_id references a symbol that
// was later deleted (re-index or partial delete), so a by-id endpoint can be
// unresolvable. fetchEdges must not abort the whole traversal with
// "sql: no rows in result set" — an unresolvable by-id from keeps its raw
// endpoint (no fabricated From), matching how a missing by-id to is already
// swallowed (kept as a name-only stop) rather than treated as fatal.
func TestFetchEdgesDanglingFromID(t *testing.T) {
	st, err := store.Open(t.TempDir(), "df")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	ctx := context.Background()
	db := st.DB

	seed := `
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES
			('/d/from.go', 1, 1, 'd', 'now'),
			('/e/to.go', 2, 2, 'e', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'dfFrom', 'dfFrom', 'function', 'go', 'func dfFrom()', 1, 5, 'h', 'uid-df-from' FROM files f WHERE f.path='/d/from.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'dfTo', 'dfTo', 'function', 'go', 'func dfTo()', 6, 10, 'h', 'uid-df-to' FROM files f WHERE f.path='/e/to.go';
		-- Edge A: healthy, from_id = the real dfFrom symbol, to dfTo by id.
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line, valid_from)
		SELECT 'calls',
			(SELECT id FROM symbols WHERE uid='uid-df-from'),
			(SELECT file_id FROM symbols WHERE uid='uid-df-from'),
			(SELECT id FROM symbols WHERE uid='uid-df-to'),
			'dfTo', 'EXTRACTED', 10, 0;
		-- Edge B: dangling from_id = 99999 (no such symbol), to dfTo by id.
		-- The traversal must survive this and not lose edge A.
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line, valid_from)
		SELECT 'calls', 99999,
			(SELECT file_id FROM symbols WHERE uid='uid-df-from'),
			(SELECT id FROM symbols WHERE uid='uid-df-to'),
			'dfTo', 'EXTRACTED', 20, 0;
	`
	if _, err := db.ExecContext(ctx, seed); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var fromID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM symbols WHERE uid='uid-df-from'`).Scan(&fromID); err != nil {
		t.Fatalf("lookup from: %v", err)
	}

	// The symbol's own edges are fetched via (from_id = ? OR to_id = ?), so it
	// only returns edge A (the healthy one) — but fetchEdges must still not
	// error even though edge B in the same table has a dangling from_id.
	edges, err := fetchEdges(ctx, db, Symbol{ID: fromID, Name: "dfFrom"})
	if err != nil {
		t.Fatalf("fetchEdges with a dangling from_id in the table: %v", err)
	}
	if len(edges) != 1 || edges[0].Line != 10 {
		t.Fatalf("expected the 1 healthy edge, got %+v", edges)
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
