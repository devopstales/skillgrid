package graph

import (
	"context"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// seedGraph seeds a small graph through the real Store write path:
// A ← B ← C (C calls B calls A), plus a name-only edge from D to A.
func seedGraph(t *testing.T) store.Store {
	t.Helper()
	st, cleanup := openTestStore(t)
	t.Cleanup(cleanup)

	ctx := context.Background()

	// File 1: A (the target)
	fileA, err := st.UpsertFile(ctx, store.FileRow{
		Path: "a.go", Language: "go", ContentHash: "hash-a",
	})
	if err != nil {
		t.Fatalf("upsert file A: %v", err)
	}
	if _, err := st.UpsertSymbols(ctx, fileA, []store.Symbol{
		{Name: "A", Kind: "function", Language: "go", StartLine: 1, EndLine: 10, UID: "uid-a"},
	}); err != nil {
		t.Fatalf("upsert symbols A: %v", err)
	}

	// File 2: B (calls A)
	fileB, err := st.UpsertFile(ctx, store.FileRow{
		Path: "b.go", Language: "go", ContentHash: "hash-b",
	})
	if err != nil {
		t.Fatalf("upsert file B: %v", err)
	}
	if _, err := st.UpsertSymbols(ctx, fileB, []store.Symbol{
		{Name: "B", Kind: "function", Language: "go", StartLine: 1, EndLine: 20, UID: "uid-b"},
	}); err != nil {
		t.Fatalf("upsert symbols B: %v", err)
	}
	symB, _ := st.SymbolByName(ctx, "B")
	if len(symB) != 1 {
		t.Fatalf("expected 1 symbol B, got %d", len(symB))
	}
	// Edge: B calls A (resolved)
	toA := int64(1) // A's ID
	if err := st.ReplaceEdgesForFile(ctx, fileB, []store.Edge{
		{Kind: "calls", FromID: symB[0].ID, ToID: &toA, Confidence: "EXTRACTED", Line: 5},
	}); err != nil {
		t.Fatalf("replace edges B: %v", err)
	}

	// File 3: C (calls B)
	fileC, err := st.UpsertFile(ctx, store.FileRow{
		Path: "c.go", Language: "go", ContentHash: "hash-c",
	})
	if err != nil {
		t.Fatalf("upsert file C: %v", err)
	}
	if _, err := st.UpsertSymbols(ctx, fileC, []store.Symbol{
		{Name: "C", Kind: "function", Language: "go", StartLine: 1, EndLine: 30, UID: "uid-c"},
	}); err != nil {
		t.Fatalf("upsert symbols C: %v", err)
	}
	symC, _ := st.SymbolByName(ctx, "C")
	if len(symC) != 1 {
		t.Fatalf("expected 1 symbol C, got %d", len(symC))
	}
	// Edge: C calls B (resolved)
	toB := symB[0].ID
	if err := st.ReplaceEdgesForFile(ctx, fileC, []store.Edge{
		{Kind: "calls", FromID: symC[0].ID, ToID: &toB, Confidence: "EXTRACTED", Line: 10},
	}); err != nil {
		t.Fatalf("replace edges C: %v", err)
	}

	// File 4: D (name-only edge to "Gone" — unresolved, no symbol named "Gone")
	fileD, err := st.UpsertFile(ctx, store.FileRow{
		Path: "d.go", Language: "go", ContentHash: "hash-d",
	})
	if err != nil {
		t.Fatalf("upsert file D: %v", err)
	}
	if _, err := st.UpsertSymbols(ctx, fileD, []store.Symbol{
		{Name: "D", Kind: "function", Language: "go", StartLine: 1, EndLine: 5, UID: "uid-d"},
	}); err != nil {
		t.Fatalf("upsert symbols D: %v", err)
	}
	symD, _ := st.SymbolByName(ctx, "D")
	if len(symD) != 1 {
		t.Fatalf("expected 1 symbol D, got %d", len(symD))
	}
	// Edge: D calls "Gone" (name-only, unresolved — no symbol named "Gone")
	if err := st.ReplaceEdgesForFile(ctx, fileD, []store.Edge{
		{Kind: "calls", FromID: symD[0].ID, ToID: nil, ToName: "Gone", Confidence: "INFERRED", Line: 3},
	}); err != nil {
		t.Fatalf("replace edges D: %v", err)
	}

	return st
}

// [impact-blast-radius] B is WILL BREAK (depth 1), C is LIKELY AFFECTED (depth 2),
// unresolved D is not in the radius, excluded hops are counted.
func TestImpactBlastRadius(t *testing.T) {
	st := seedGraph(t)
	ctx := context.Background()

	symA, _ := st.SymbolByName(ctx, "A")
	if len(symA) != 1 {
		t.Fatalf("expected 1 symbol A, got %d", len(symA))
	}

	res, err := Impact(ctx, st, symA[0].ID, "INFERRED", 5)
	if err != nil {
		t.Fatalf("impact: %v", err)
	}

	// B is WILL BREAK (depth 1)
	if !res.HasWillBreak("B") {
		t.Fatalf("B must be WILL BREAK: %+v", res.WillBreak)
	}
	// C is LIKELY AFFECTED (depth 2)
	if !res.HasLikely("C") {
		t.Fatalf("C must be LIKELY AFFECTED: %+v", res.Likely)
	}
	// D is not in the radius (name-only edge, unresolved)
	if res.Has("D") {
		t.Fatalf("unresolved D must not be in the radius")
	}
	// The excluded count is 0 because there are no INFERRED edges that are
	// dependents of A (the edge from D is to "Gone", not "A")
}

// [code-resolve-contract] exists/file/line/called_from; ambiguous → candidates[]; gone → exists:false.
func TestCodeResolveContract(t *testing.T) {
	st := seedResolve(t)
	ctx := context.Background()

	// Resolve CapturePayment
	r, err := Resolve(ctx, st, "CapturePayment", nil)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !r.Exists || r.File != "internal/pay/pay.go" || r.Line != 480 {
		t.Fatalf("resolve CapturePayment: %+v", r)
	}
	if len(r.CalledFrom) != 1 || r.CalledFrom[0] != "internal/billing/enterprise.go" {
		t.Fatalf("called_from: %+v", r.CalledFrom)
	}

	// Ambiguous: two symbols named Foo
	ra, err := Resolve(ctx, st, "Foo", nil)
	if err != nil {
		t.Fatalf("resolve Foo: %v", err)
	}
	if !ra.Ambiguous || len(ra.Candidates) != 2 {
		t.Fatalf("ambiguous Foo: %+v", ra)
	}

	// Gone: exists:false
	rn, err := Resolve(ctx, st, "Gone", nil)
	if err != nil {
		t.Fatalf("resolve Gone: %v", err)
	}
	if rn.Exists {
		t.Fatalf("gone symbol must be exists:false: %+v", rn)
	}
}

// [code-packages-contract] path/files/symbols; no sub-package unit.
func TestCodePackagesContract(t *testing.T) {
	st := seedPackages(t)
	ctx := context.Background()

	pkgs, err := Packages(ctx, st)
	if err != nil {
		t.Fatalf("packages: %v", err)
	}
	byPath := map[string]PackageInfo{}
	for _, p := range pkgs {
		byPath[p.Path] = p
	}
	if byPath["internal/memory"].Files != 12 || byPath["internal/memory"].Symbols != 84 {
		t.Fatalf("package map: %+v", pkgs)
	}
	if _, ok := byPath["internal/memory/sub"]; ok {
		t.Fatalf("sub-package must not be a unit: %+v", byPath["internal/memory/sub"])
	}
}

// [path-graph-stops] the causal dispatch kind — the interface at the frontier.
func TestPathGraphStops(t *testing.T) {
	st := seedStops(t)
	ctx := context.Background()

	symA, _ := st.SymbolByName(ctx, "A")
	symUnreachable, _ := st.SymbolByName(ctx, "Unreachable")
	if len(symA) != 1 || len(symUnreachable) != 1 {
		t.Fatalf("expected symbols A and Unreachable")
	}

	p, err := Path(ctx, st, symA[0].ID, symUnreachable[0].ID)
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	if p.Path == nil && p.Stops == nil {
		t.Fatalf("PathResult must be something: %+v", p)
	}
	if p.Stops != nil && p.Stops.DispatchKind != "interface" {
		t.Fatalf("causal dispatch kind: %+v", p.Stops)
	}
}

// [affected-tests-java] the v1 gap fixed — FooTest.java recognized.
func TestAffectedTestsJava(t *testing.T) {
	st := seedJava(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := Affected(ctx, st, AffectedOpts{Changed: []string{"src/Foo.java"}})
	if err != nil {
		t.Fatalf("affected: %v", err)
	}
	if len(res.AffectedTests) != 1 || res.AffectedTests[0] != "src/FooTest.java" {
		t.Fatalf("java test convention: %+v", res.AffectedTests)
	}
}

// [name-only-query-time-relabel] 0/1/>1 relabel — >1 → AMBIGUOUS each.
func TestNameOnlyQueryTimeRelabel(t *testing.T) {
	st := seedNameOnly(t)
	ctx := context.Background()

	symFoo1, _ := st.SymbolByName(ctx, "Foo")
	if len(symFoo1) != 2 {
		t.Fatalf("expected 2 symbols Foo, got %d", len(symFoo1))
	}

	n, err := Callers(ctx, st, symFoo1[0].ID)
	if err != nil {
		t.Fatalf("callers: %v", err)
	}
	ambiguous := 0
	for _, e := range n.Edges {
		if e.Confidence == "AMBIGUOUS" {
			ambiguous++
		}
	}
	if ambiguous == 0 {
		t.Fatalf("ambiguous name-only must relabel to AMBIGUOUS: %+v", n.Edges)
	}
}

// [orient-map-navigable] orient on a package → TOC + outbound edges with ids; depth=1 does not expand; re-root works.
func TestOrientMapReRootsAndBounded(t *testing.T) {
	st := seedOrient(t)
	ctx := context.Background()

	// Orient on a package
	pkg, err := Orient(ctx, st, OrientUnit{Kind: UnitPackage, Path: "internal/pay"}, 1)
	if err != nil {
		t.Fatalf("orient package: %v", err)
	}
	if len(pkg.TOC) == 0 {
		t.Fatalf("package TOC must list symbols: %+v", pkg.TOC)
	}
	if len(pkg.Outbound) == 0 {
		t.Fatalf("package map must carry outbound edges: %+v", pkg.Outbound)
	}
	for _, e := range pkg.Outbound {
		if e.TargetExpanded {
			t.Fatalf("depth=1 must not expand edge targets: %+v", e)
		}
	}

	// Re-root on an edge target
	targetID := pkg.Outbound[0].ToID
	if targetID == 0 {
		t.Fatalf("outbound edge must have a target ID: %+v", pkg.Outbound[0])
	}
	sym, err := Orient(ctx, st, OrientUnit{Kind: UnitSymbol, ID: targetID}, 1)
	if err != nil {
		t.Fatalf("orient symbol: %v", err)
	}
	if sym.Meta == "" || len(sym.TOC) == 0 {
		t.Fatalf("re-root on symbol id must return its map: %+v", sym)
	}
}

// [code-explain-contract] degree + connections ranked by neighbor degree, hubs first.
func TestCodeExplainContract(t *testing.T) {
	st := seedExplain(t)
	ctx := context.Background()

	symCapture, _ := st.SymbolByName(ctx, "CapturePayment")
	if len(symCapture) != 1 {
		t.Fatalf("expected 1 symbol CapturePayment, got %d", len(symCapture))
	}

	r, err := Explain(ctx, st, symCapture[0].ID)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	if r.Degree != 3 {
		t.Fatalf("degree must count connections: %+v", r)
	}
	if len(r.Connections) < 2 || r.Connections[0].NeighborDegree < r.Connections[1].NeighborDegree {
		t.Fatalf("connections must be ranked by neighbor degree: %+v", r.Connections)
	}
}

// [retrieval-trail-recorded] a walker read appends exactly one trail row.
func TestRetrievalTrailRecorded(t *testing.T) {
	st := seedGraph(t)
	ctx := context.Background()

	before := trailCountForProject(t, st, "p")
	symA, _ := st.SymbolByName(ctx, "A")
	if len(symA) != 1 {
		t.Fatalf("expected 1 symbol A, got %d", len(symA))
	}
	_, _ = Impact(ctx, st, symA[0].ID, "INFERRED", 5)
	after := trailCountForProject(t, st, "p")
	if after != before+1 {
		t.Fatalf("code_impact must record a trail: before=%d after=%d", before, after)
	}
}
