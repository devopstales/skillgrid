package graph

import (
	"context"
	"database/sql"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// impactFixture builds a store with a small graph: target `base` with a direct
// dependent `mid` (depth 1) and a deeper dependent `top` (depth 2), plus a
// duplicate-named symbol `base` in a second file for disambiguation.
func impactFixture(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	ctx := context.Background()
	db := st.DB

	seed := `
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES
			('/a/base.go', 1, 1, 'a', 'now'),
			('/b/base.go', 2, 2, 'b', 'now'),
			('/c/mid.go', 3, 3, 'c', 'now'),
			('/d/top.go', 4, 4, 'd', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'base', 'base', 'function', 'go', 'func base()', 1, 5, 'h', 'uid-base-a' FROM files f WHERE f.path='/a/base.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'base', 'base', 'method', 'go', 'func (x) base()', 1, 5, 'h', 'uid-base-b' FROM files f WHERE f.path='/b/base.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'mid', 'mid', 'function', 'go', 'func mid()', 1, 5, 'h', 'uid-mid' FROM files f WHERE f.path='/c/mid.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'top', 'top', 'function', 'go', 'func top()', 1, 5, 'h', 'uid-top' FROM files f WHERE f.path='/d/top.go';
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'calls',
			(SELECT id FROM symbols WHERE uid='uid-mid'),
			(SELECT file_id FROM symbols WHERE uid='uid-mid'),
			NULL, 'base',
			'EXTRACTED', 10;
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'calls',
			(SELECT id FROM symbols WHERE uid='uid-top'),
			(SELECT file_id FROM symbols WHERE uid='uid-top'),
			NULL, 'mid',
			'INFERRED', 20;
	`
	if _, err := db.ExecContext(ctx, seed); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return st
}

// TestCodeImpactRiskTiers covers @step-03 (code_impact tiers risk): direct
// dependents are WILL BREAK, deeper dependents LIKELY AFFECTED, every edge
// confidence-tagged, and minConfidence filters low-confidence hops.
func TestCodeImpactRiskTiers(t *testing.T) {
	st := impactFixture(t)
	ctx := context.Background()
	target := Symbol{ID: mustID(ctx, st.DB, "uid-base-a"), Name: "base"}

	res, err := Impact(ctx, st.DB, target, ImpactOptions{})
	if err != nil {
		t.Fatalf("impact: %v", err)
	}
	if len(res.WillBreak) != 1 {
		t.Fatalf("expected 1 will-break (mid), got %d", len(res.WillBreak))
	}
	if res.WillBreak[0].Symbol.Name != "mid" {
		t.Errorf("will-break should be mid, got %s", res.WillBreak[0].Symbol.Name)
	}
	if res.WillBreak[0].Risk != RiskWillBreak {
		t.Errorf("depth-1 risk = %q, want WILL BREAK", res.WillBreak[0].Risk)
	}
	if res.WillBreak[0].Confidence != ConfidenceExtracted {
		t.Errorf("edge must be confidence-tagged, got %q", res.WillBreak[0].Confidence)
	}
	if len(res.Likely) != 1 {
		t.Fatalf("expected 1 likely-affected (top), got %d", len(res.Likely))
	}
	if res.Likely[0].Symbol.Name != "top" {
		t.Errorf("likely-affected should be top, got %s", res.Likely[0].Symbol.Name)
	}
	if res.Likely[0].Risk != RiskLikelyAffected {
		t.Errorf("depth-2 risk = %q, want LIKELY AFFECTED", res.Likely[0].Risk)
	}
	if res.Likely[0].Confidence != ConfidenceInferred {
		t.Errorf("deeper edge must be confidence-tagged, got %q", res.Likely[0].Confidence)
	}

	// minConfidence=EXTRACTED filters the INFERRED hop (top), keeping mid.
	res2, err := Impact(ctx, st.DB, target, ImpactOptions{MinConfidence: ConfidenceExtracted})
	if err != nil {
		t.Fatalf("impact: %v", err)
	}
	if len(res2.WillBreak) != 1 || res2.WillBreak[0].Symbol.Name != "mid" {
		t.Errorf("minConfidence=EXTRACTED should keep mid, got %+v", res2.WillBreak)
	}
	if len(res2.Likely) != 0 {
		t.Errorf("minConfidence=EXTRACTED should exclude the INFERRED top hop, got %+v", res2.Likely)
	}
	if res2.Excluded == 0 {
		t.Errorf("expected the excluded low-confidence hop to be counted, got 0")
	}
}

// TestCodeImpactDisambiguates covers @step-03 (a name matching several symbols
// returns a ranked candidate list, never a silent pick; narrowable by file/uid
// /kind).
func TestCodeImpactDisambiguates(t *testing.T) {
	st := impactFixture(t)
	ctx := context.Background()
	db := st.DB

	res, err := Resolve(ctx, db, "base", ResolveFilter{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !res.Ambiguous {
		t.Fatalf("expected ambiguous resolution for 'base', got %+v", res)
	}
	if len(res.Matches) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(res.Matches))
	}
	ranked, err := RankCandidates(ctx, db, res.Matches)
	if err != nil {
		t.Fatalf("rank: %v", err)
	}
	if len(ranked) != 2 {
		t.Fatalf("expected 2 ranked candidates, got %d", len(ranked))
	}
	// The more-connected base (has a dependent) ranks first.
	if ranked[0].UID != "uid-base-a" {
		t.Errorf("most-connected candidate should rank first, got %s", ranked[0].UID)
	}

	// Narrow by file.
	resFile, err := Resolve(ctx, db, "base", ResolveFilter{File: "/b/base.go"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resFile.Ambiguous || resFile.Target.UID != "uid-base-b" {
		t.Errorf("file narrow should pick base-b, got %+v", resFile)
	}
	// Narrow by kind.
	resKind, err := Resolve(ctx, db, "base", ResolveFilter{Kind: "method"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resKind.Ambiguous || resKind.Target.UID != "uid-base-b" {
		t.Errorf("kind narrow should pick base-b, got %+v", resKind)
	}
	// Narrow by uid.
	resUID, err := Resolve(ctx, db, "base", ResolveFilter{UID: "uid-base-a"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resUID.Ambiguous || resUID.Target.UID != "uid-base-a" {
		t.Errorf("uid narrow should pick base-a, got %+v", resUID)
	}
	// An unsatisfied filter yields not-found (no silent pick).
	resNone, err := Resolve(ctx, db, "base", ResolveFilter{File: "/nope/base.go"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !resNone.NotFound {
		t.Errorf("unsatisfied filter should be not-found, got %+v", resNone)
	}
}

// TestImpactTraversesRouteReferencesEdges covers the step-01 blast-radius
// reconciliation: a RESOLVED route->handler `references` edge (step-01 kind,
// to_id set, to_name = handler) is traversed by the reverse impact walk, so a
// handler's blast radius includes its serving route. A DROPPED (ambiguous)
// reference has no stored edge, so it must NOT appear (drop-not-guess).
func TestImpactTraversesRouteReferencesEdges(t *testing.T) {
	st := impactFixture(t)
	ctx := context.Background()
	db := st.DB

	// Add a route node (kind=route) that references the `base-a` handler by a
	// RESOLVED references edge (to_id set, to_name = 'base', EXTRACTED).
	_, err := db.ExecContext(ctx, `
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('/urls.py', 5, 5, 'u', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'GET /base/', 'GET /base/', 'route', 'python', 'path("/base/")', 1, 5, 'h', 'uid-route'
		FROM files f WHERE f.path = '/urls.py';
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'references',
			(SELECT id FROM symbols WHERE uid = 'uid-route'),
			(SELECT file_id FROM symbols WHERE uid = 'uid-route'),
			(SELECT id FROM symbols WHERE uid = 'uid-base-a'),
			'base',
			'EXTRACTED', 30;
	`)
	if err != nil {
		t.Fatalf("seed route: %v", err)
	}

	// Blast radius of the handler `base` (uid-base-a) must now include its
	// serving route (the reverse references edge traversed).
	target := Symbol{ID: mustID(ctx, db, "uid-base-a"), Name: "base"}
	res, err := Impact(ctx, db, target, ImpactOptions{})
	if err != nil {
		t.Fatalf("impact: %v", err)
	}
	var routeIncluded bool
	for _, e := range append(res.WillBreak, res.Likely...) {
		if e.Symbol.UID == "uid-route" {
			routeIncluded = true
			if e.Kind != "references" {
				t.Errorf("route hop kind = %q, want references", e.Kind)
			}
		}
	}
	if !routeIncluded {
		t.Errorf("expected the serving route to be in the handler's blast radius (references edge traversed), got %+v", res)
	}

	// A DROPPED reference (unresolvable) is never stored, so it must not
	// inflate the radius. Seed a second route node with NO references edge to
	// `base` (it was dropped by drop-not-guess) and assert it is absent.
	_, err = db.ExecContext(ctx, `
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('/urls2.py', 6, 6, 'u2', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'GET /mystery/', 'GET /mystery/', 'route', 'python', 'path("/mystery/")', 1, 5, 'h', 'uid-route-dropped'
		FROM files f WHERE f.path = '/urls2.py';
	`)
	if err != nil {
		t.Fatalf("seed dropped route: %v", err)
	}
	res2, err := Impact(ctx, db, target, ImpactOptions{})
	if err != nil {
		t.Fatalf("impact: %v", err)
	}
	for _, e := range append(res2.WillBreak, res2.Likely...) {
		if e.Symbol.UID == "uid-route-dropped" {
			t.Errorf("a dropped (unresolvable) reference must not appear in the blast radius, got %+v", e)
		}
	}
}

// mustID resolves a symbol row id by UID (test helper).
func mustID(ctx context.Context, db *sql.DB, uid string) int64 {
	var id int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM symbols WHERE uid = ?`, uid).Scan(&id); err != nil {
		return 0
	}
	return id
}
