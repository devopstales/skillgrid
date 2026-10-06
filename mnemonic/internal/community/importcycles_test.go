package community

import (
	"testing"
)

// TestFindImportCyclesNULLExternalTarget covers the external/stdlib import case:
// an import edge whose target_path matches no indexed file leaves f_to.path
// NULL via the LEFT JOIN. importFileEdges must scan that NULL without a
// "converting NULL to string" error and drop the edge, while still returning
// the real in-repo cycle.
func TestFindImportCyclesNULLExternalTarget(t *testing.T) {
	db := openStore(t)
	aID := seedFile(t, db, "a.go")
	bID := seedFile(t, db, "b.go")
	// from_id must be a valid symbol id (edges.from_id references symbols).
	aSym := seedSymbol(t, db, aID, "A", "func", 1)
	bSym := seedSymbol(t, db, bID, "B", "func", 1)
	// a.go -> b.go and b.go -> a.go: a real 2-file import cycle.
	if _, err := db.Exec(`INSERT INTO edges (kind, from_id, file_id, to_name, target_path, confidence, line)
		VALUES ('imports', ?, ?, 'b.go', 'b.go', 'EXTRACTED', 1),
		       ('imports', ?, ?, 'a.go', 'a.go', 'EXTRACTED', 2)`, aSym, aID, bSym, bID); err != nil {
		t.Fatalf("seed cycle: %v", err)
	}
	// a.go -> "context" (stdlib): no indexed file matches, so f_to.path is NULL.
	if _, err := db.Exec(`INSERT INTO edges (kind, from_id, file_id, to_name, target_path, confidence, line)
		VALUES ('imports', ?, ?, 'context', 'context', 'EXTRACTED', 3)`, aSym, aID); err != nil {
		t.Fatalf("seed external: %v", err)
	}

	cycles, err := FindImportCycles(db)
	if err != nil {
		t.Fatalf("FindImportCycles: %v (want nil — the NULL external target must not error)", err)
	}
	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle (external dropped), got %d: %+v", len(cycles), cycles)
	}
	if cycles[0].NodeCount != 2 || cycles[0].Cycle[0] != "a.go" {
		t.Errorf("cycle = %+v, want a 2-node loop starting at a.go", cycles[0])
	}
}

// TestDetectCyclesTwoNodeLoop covers the classic A→B→A import cycle: a single
// 2-file loop must be detected once, canonicalized to put the lexicographically
// smallest file first, and closed (last == first).
func TestDetectCyclesTwoNodeLoop(t *testing.T) {
	edges := []FileEdge{{From: "b.go", To: "a.go"}, {From: "a.go", To: "b.go"}}
	cycles, err := DetectCycles(edges)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d: %+v", len(cycles), cycles)
	}
	c := cycles[0]
	if c.NodeCount != 2 {
		t.Errorf("NodeCount = %d, want 2", c.NodeCount)
	}
	// Canonical: smallest file ("a.go") first, loop closed.
	if len(c.Cycle) != 3 || c.Cycle[0] != "a.go" || c.Cycle[1] != "b.go" || c.Cycle[2] != "a.go" {
		t.Errorf("cycle = %v, want [a.go b.go a.go]", c.Cycle)
	}
}

// TestDetectCyclesThreeNodeLoop covers A→B→C→A: a 3-file loop, canonicalized
// to start at the smallest file.
func TestDetectCyclesThreeNodeLoop(t *testing.T) {
	edges := []FileEdge{
		{From: "c.go", To: "a.go"},
		{From: "a.go", To: "b.go"},
		{From: "b.go", To: "c.go"},
	}
	cycles, err := DetectCycles(edges)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d: %+v", len(cycles), cycles)
	}
	c := cycles[0]
	if c.NodeCount != 3 {
		t.Errorf("NodeCount = %d, want 3", c.NodeCount)
	}
	if len(c.Cycle) != 4 || c.Cycle[0] != "a.go" || c.Cycle[3] != "a.go" {
		t.Errorf("cycle = %v, want [a.go ... a.go] (4 entries)", c.Cycle)
	}
}

// TestDetectCyclesAcyclicIsNone covers a DAG (no loop): zero cycles.
func TestDetectCyclesAcyclicIsNone(t *testing.T) {
	edges := []FileEdge{
		{From: "a.go", To: "b.go"},
		{From: "b.go", To: "c.go"},
		{From: "a.go", To: "c.go"},
	}
	cycles, err := DetectCycles(edges)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(cycles) != 0 {
		t.Errorf("expected 0 cycles, got %d: %+v", len(cycles), cycles)
	}
}

// TestDetectCyclesSelfImport covers a single-file self-import (A→A): a
// 1-node cycle.
func TestDetectCyclesSelfImport(t *testing.T) {
	edges := []FileEdge{{From: "a.go", To: "a.go"}}
	cycles, err := DetectCycles(edges)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(cycles) != 1 {
		t.Fatalf("expected 1 self-cycle, got %d: %+v", len(cycles), cycles)
	}
	if cycles[0].NodeCount != 1 {
		t.Errorf("self NodeCount = %d, want 1", cycles[0].NodeCount)
	}
}

// TestDetectCyclesTwoDistinctLoops covers two disjoint cycles (A↔B and C↔D):
// both must be found, and the result ordered deterministically (by node_count
// then lexicographic first file).
func TestDetectCyclesTwoDistinctLoops(t *testing.T) {
	edges := []FileEdge{
		{From: "a.go", To: "b.go"},
		{From: "b.go", To: "a.go"},
		{From: "c.go", To: "d.go"},
		{From: "d.go", To: "c.go"},
	}
	cycles, err := DetectCycles(edges)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(cycles) != 2 {
		t.Fatalf("expected 2 cycles, got %d: %+v", len(cycles), cycles)
	}
	// Both are 2-node; ordered by first file (a.go before c.go).
	if cycles[0].Cycle[0] != "a.go" {
		t.Errorf("first cycle starts %q, want a.go", cycles[0].Cycle[0])
	}
	if cycles[1].Cycle[0] != "c.go" {
		t.Errorf("second cycle starts %q, want c.go", cycles[1].Cycle[0])
	}
}

// TestDetectCyclesReachesSameLoopOnce covers a cycle reached from two different
// entry points (X→A and Y→A, with A↔B): the A↔B loop must be reported exactly
// once (canonical de-duplication), not once per entry point.
func TestDetectCyclesReachesSameLoopOnce(t *testing.T) {
	edges := []FileEdge{
		{From: "x.go", To: "a.go"},
		{From: "y.go", To: "a.go"},
		{From: "a.go", To: "b.go"},
		{From: "b.go", To: "a.go"},
	}
	cycles, err := DetectCycles(edges)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(cycles) != 1 {
		t.Fatalf("expected exactly 1 cycle (de-duplicated), got %d: %+v", len(cycles), cycles)
	}
	if cycles[0].Cycle[0] != "a.go" {
		t.Errorf("cycle starts %q, want a.go", cycles[0].Cycle[0])
	}
}

// TestCohesionOf covers the cohesion math: density internal/C(n,2), the
// degenerate <2 case (pointer to 0, NOT nil), and the fully-connected clique.
func TestCohesionOf(t *testing.T) {
	// Singleton: 0.0 (not nil).
	if c := cohesionOf(1, 0); c == nil || *c != 0 {
		t.Errorf("cohesionOf(1,0) = %v, want ptr to 0", c)
	}
	// Two members, one internal edge: 1/1 = 1.0 (a pair is a clique).
	if c := cohesionOf(2, 1); c == nil || *c != 1.0 {
		t.Errorf("cohesionOf(2,1) = %v, want 1.0", c)
	}
	// Four members, two internal edges: 2/6 ≈ 0.333.
	if c := cohesionOf(4, 2); c == nil || *c != 2.0/6.0 {
		t.Errorf("cohesionOf(4,2) = %v, want %v", c, 2.0/6.0)
	}
	// Four members, no internal edges: 0.0.
	if c := cohesionOf(4, 0); c == nil || *c != 0 {
		t.Errorf("cohesionOf(4,0) = %v, want 0.0", c)
	}
}

// TestWriteImportCyclesTargetState covers that WriteImportCycles replaces the
// table (stale cycles pruned, new set written, node_count + cycle persisted).
func TestWriteImportCyclesTargetState(t *testing.T) {
	db := openStore(t)
	c := []ImportCycle{{Cycle: []string{"a.go", "b.go", "a.go"}, NodeCount: 2}}
	if err := WriteImportCycles(db, c); err != nil {
		t.Fatalf("write: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM import_cycles`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row, got %d", count)
	}
	var cycle string
	var nc int
	if err := db.QueryRow(`SELECT cycle, node_count FROM import_cycles`).Scan(&cycle, &nc); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if cycle != "a.go,b.go,a.go" || nc != 2 {
		t.Errorf("row = (%q, %d), want (a.go,b.go,a.go, 2)", cycle, nc)
	}
	// Target-state: a second write with a different set replaces the first.
	if err := WriteImportCycles(db, nil); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM import_cycles`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected table emptied by target-state rewrite, got %d", count)
	}
}
