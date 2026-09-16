package codeindex

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg"
)

// TestCfgPdgBuild covers @step-01 (Scenario: Opt-in index builds per-function
// CFG and PDG): indexing the fixture WITH --pdg populates cfg_blocks +
// cfg_edges per function (basic blocks from branch/loop/return structure) and
// pdg_edges for both control- and data-dependence; a non---pdg index of the
// same repo leaves those tables empty.
func TestCfgPdgBuild(t *testing.T) {
	ResetFileFirstSymbol()

	// With --pdg: the tables populate.
	{
		idx, clean := newTestIndexer(t)
		defer clean()
		idx.EnablePDG()
		root := writePdgFixture(t)
		if _, err := idx.Run(contextBackground2(), root, pdgCfg); err != nil {
			t.Fatalf("run (--pdg): %v", err)
		}
		db := idx.store.DB
		var blocks, edges, pdg int
		if err := db.QueryRow(`SELECT COUNT(*) FROM cfg_blocks`).Scan(&blocks); err != nil {
			t.Fatalf("count cfg_blocks: %v", err)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM cfg_edges`).Scan(&edges); err != nil {
			t.Fatalf("count cfg_edges: %v", err)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM pdg_edges`).Scan(&pdg); err != nil {
			t.Fatalf("count pdg_edges: %v", err)
		}
		if blocks == 0 {
			t.Errorf("--pdg: expected cfg_blocks per function, got 0")
		}
		if edges == 0 {
			t.Errorf("--pdg: expected cfg_edges, got 0")
		}
		if pdg == 0 {
			t.Errorf("--pdg: expected pdg_edges (control + data), got 0")
		}
		var ctrl, data int
		if err := db.QueryRow(`SELECT COUNT(*) FROM pdg_edges WHERE kind = 'control'`).Scan(&ctrl); err != nil {
			t.Fatalf("count control: %v", err)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM pdg_edges WHERE kind = 'data'`).Scan(&data); err != nil {
			t.Fatalf("count data: %v", err)
		}
		if ctrl == 0 {
			t.Errorf("--pdg: expected control-dependence pdg_edges, got 0")
		}
		if data == 0 {
			t.Errorf("--pdg: expected data-dependence pdg_edges, got 0")
		}
	}

	// Without --pdg: the tables are empty.
	{
		idx, clean := newTestIndexer(t)
		defer clean()
		root := writePdgFixture(t)
		if _, err := idx.Run(contextBackground2(), root, pdgCfg); err != nil {
			t.Fatalf("run (no --pdg): %v", err)
		}
		db := idx.store.DB
		for _, table := range []string{"cfg_blocks", "cfg_edges", "pdg_edges"} {
			var n int
			if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
				t.Fatalf("count %s: %v", table, err)
			}
			if n != 0 {
				t.Errorf("non---pdg: table %s has %d rows, want 0", table, n)
			}
		}
	}
}

// contextBackground2 is a tiny helper for this test file.
func contextBackground2() context.Context { return context.Background() }

// TestCfgBuildDeterministic is a unit check that BuildCFG produces a non-empty,
// structured CFG (branch + loop + return) for the main function, and that a
// repeated build is byte-for-byte reproducible.
func TestCfgBuildDeterministic(t *testing.T) {
	src := []byte("package main\n\nfunc main() {\n\tn := add(1, 2)\n\tif n > 3 {\n\t\twarn(n)\n\t} else {\n\t\tok()\n\t}\n\tfor i := 0; i < n; i++ {\n\t\tlog(i)\n\t}\n\tswitch n {\n\tcase 1:\n\t\ta()\n\tdefault:\n\t\tb()\n\t}\n\treturn\n}\n\nfunc add(a, b int) int {\n\treturn a + b\n}\n\nfunc warn(n int) {\n\tm := n * 2\n\t_ = m\n}\n\nfunc ok() {}\n\nfunc log(i int) {}\n\nfunc a() {}\n\nfunc b() {}\n")
	c, err := pdg.BuildCFG(src, "go", "main", 3)
	if err != nil {
		t.Fatalf("BuildCFG: %v", err)
	}
	if c == nil {
		t.Fatal("BuildCFG returned nil for a valid function")
	}
	// The main function has branch (if/switch) + loop (for) + return structure.
	hasBranch, hasLoop, hasReturn := false, false, false
	for _, b := range c.Blocks {
		switch b.Kind {
		case "branch":
			hasBranch = true
		case "loop":
			hasLoop = true
		case "return":
			hasReturn = true
		}
	}
	if !hasBranch {
		t.Errorf("expected a branch block (if/switch), got none")
	}
	if !hasLoop {
		t.Errorf("expected a loop block (for), got none")
	}
	if !hasReturn {
		t.Errorf("expected a return block, got none")
	}
	// Determinism: a second build is byte-for-byte identical.
	c2, err := pdg.BuildCFG(src, "go", "main", 3)
	if err != nil {
		t.Fatalf("BuildCFG (2nd): %v", err)
	}
	if pdg.DumpCFG(c) != pdg.DumpCFG(c2) {
		t.Errorf("CFG not deterministic:\n%s\nvs\n%s", pdg.DumpCFG(c), pdg.DumpCFG(c2))
	}
}
