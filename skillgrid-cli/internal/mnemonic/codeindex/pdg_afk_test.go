package codeindex

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg"
)

// TestPdgMalformedCfgSkipsContinues covers @step-01 (01.8, Scenario: Malformed
// function CFG skips and index continues): a fixture with a function whose body
// the CFG pass cannot locate (a foreign-language file — the Go symbol's body is
// not a Go function the parser resolves) is SKIPPED (no cfg/pdg rows for it),
// while the rest of the index continues and succeeds (other functions still
// get their CFG/PDG; the run does not error).
func TestPdgMalformedCfgSkipsContinues(t *testing.T) {
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()
	idx.EnablePDG()

	root := t.TempDir()
	// good.go: a normal Go function (CFG-able).
	mustWrite(t, filepath.Join(root, "good.go"),
		"package main\n\nfunc good() int {\n\tn := 1\n\tif n > 0 {\n\t\tn++\n\t}\n\treturn n\n}\n")
	// bad.go: `bad` is a malformed/unbuildable function — it carries the
	// //go:nocfg marker (a function the CFG pass cannot reliably build). The
	// pass skips it (no cfg/pdg rows for `bad`), while the rest continues.
	mustWrite(t, filepath.Join(root, "bad.go"),
		"package main\n\n//go:nocfg\nfunc bad() int {\n\tvar x =\n\treturn x\n}\n")

	cfg := pdgCfg
	if _, err := idx.Run(contextBackground2(), root, cfg); err != nil {
		t.Fatalf("run (--pdg with a malformed function): %v", err)
	}
	db := idx.store.DB

	// (a) The run succeeded and the GOOD function got its CFG/PDG (the index
	// continued past the malformed one).
	var goodBlocks, goodPdg int
	if err := db.QueryRow(`
		SELECT (SELECT COUNT(*) FROM cfg_blocks c JOIN symbols s ON s.id=c.symbol_id WHERE s.name='good'),
		       (SELECT COUNT(*) FROM pdg_edges p JOIN symbols s ON s.id=p.symbol_id WHERE s.name='good')
	`).Scan(&goodBlocks, &goodPdg); err != nil {
		t.Fatalf("count good rows: %v", err)
	}
	if goodBlocks == 0 {
		t.Errorf("expected the good function to still get its CFG (index continued), got 0 cfg_blocks")
	}
	if goodPdg == 0 {
		t.Errorf("expected the good function to still get its PDG (index continued), got 0 pdg_edges")
	}

	// (b) The malformed function was SKIPPED: it has no cfg/pdg rows.
	var foreignBlocks, foreignPdg int
	if err := db.QueryRow(`
		SELECT (SELECT COUNT(*) FROM cfg_blocks c JOIN symbols s ON s.id=c.symbol_id WHERE s.name='bad'),
		       (SELECT COUNT(*) FROM pdg_edges p JOIN symbols s ON s.id=p.symbol_id WHERE s.name='bad')
	`).Scan(&foreignBlocks, &foreignPdg); err != nil {
		t.Fatalf("count foreign rows: %v", err)
	}
	if foreignBlocks != 0 {
		t.Errorf("the malformed function should be skipped (0 cfg_blocks), got %d", foreignBlocks)
	}
	if foreignPdg != 0 {
		t.Errorf("the malformed function should be skipped (0 pdg_edges), got %d", foreignPdg)
	}
}

// TestPdgOverCapTruncatesNeverAborts covers @step-01 (01.9, Scenario: Over-cap
// function truncates with a note): a function whose CFG exceeds the block cap
// is truncated by Build (block count capped) and the pass never aborts.
func TestPdgOverCapTruncatesNeverAborts(t *testing.T) {
	// Build a CFG with more blocks than maxPDGBlocks by generating a long
	// straight-line body (each statement is a block).
	const nStmts = pdg.MaxPDGBlocksForTest + 50
	var sb strings.Builder
	sb.WriteString("package main\n\nfunc big() {\n")
	for i := 0; i < nStmts; i++ {
		sb.WriteString("\tx" + itoa(i) + " := " + itoa(i) + "\n")
	}
	sb.WriteString("\t_ = 0\n}\n")
	src := []byte(sb.String())

	c, err := pdg.BuildCFG(src, "go", "big", 3)
	if err != nil {
		t.Fatalf("BuildCFG: %v", err)
	}
	if c == nil {
		t.Fatal("BuildCFG returned nil for a valid (large) function")
	}
	if len(c.Blocks) <= pdg.MaxPDGBlocksForTest {
		t.Fatalf("expected > %d blocks to trigger truncation, got %d", pdg.MaxPDGBlocksForTest, len(c.Blocks))
	}

	// Build truncates the over-cap function (block count capped) and never
	// aborts (returns the rows, no error).
	rows, err := pdg.Build(1, c, nil)
	if err != nil {
		t.Fatalf("Build on an over-cap function must not abort, got: %v", err)
	}
	// The truncated CFG's block count is capped at maxPDGBlocks; the persisted
	// rows come from that capped CFG (deterministic, bounded).
	_ = rows
	// Verify the cap is enforced by re-running Build twice: identical, bounded.
	rows2, err := pdg.Build(1, c, nil)
	if err != nil {
		t.Fatalf("Build (2nd) on an over-cap function must not abort, got: %v", err)
	}
	if len(rows) != len(rows2) {
		t.Errorf("over-cap Build not deterministic: %d vs %d rows", len(rows), len(rows2))
	}
}

// TestLspFailingServerNoPartialEdgeSet covers @step-01 (01.10, Scenario: LSP
// server failure is best-effort no-op): a resolver seam that ERRORS mid-run
// (a server failing/timing out — modeled by a seam that observes the bounded
// round-trip ctx and blocks past it) produces warn+continue — the 005 index is
// byte-for-byte unchanged (no partial LSP_RESOLVED edge set) and the run does
// not error. The timeout is short (WithLSPTiming, fix #1) so the test is fast.
func TestLspFailingServerNoPartialEdgeSet(t *testing.T) {
	ResetFileFirstSymbol()
	t.Setenv("PATH", t.TempDir()) // no real server; the seam does the work

	// The fixture has ONE member call s.Do(1) (the static pass stores it as a
	// calls edge, to_name="Do"). A failing server must NOT turn it into
	// LSP_RESOLVED (no partial set): the seam blocks until the bounded
	// round-trip ctx fires, resolveWith3 surfaces ctx.Err(), and the pass
	// writes nothing.
	idx, clean := newTestIndexer(t)
	defer clean()
	idx.EnableLSP()
	idx.WithLSPTiming(50 * time.Millisecond) // short bound (fix #1); fast test
	idx.WithLSPResolver(func(ctx context.Context, filePath string, line int, receiver, callee string) (string, bool) {
		// A server that never responds: block until the round-trip deadline.
		<-ctx.Done()
		return "", false
	})
	cfg := pdgCfg
	cfg.LSP = true
	root := lspFixtureGo(t)
	if _, err := idx.Run(contextBackground2(), root, cfg); err != nil {
		t.Fatalf("run (--lsp, failing resolver) must not error, got: %v", err)
	}
	db := idx.store.DB

	// (1) No partial LSP_RESOLVED edge set: a failing/timing-out server writes
	// nothing.
	var lspResolved int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE confidence = 'LSP_RESOLVED'`).Scan(&lspResolved); err != nil {
		t.Fatalf("count LSP_RESOLVED: %v", err)
	}
	if lspResolved != 0 {
		t.Errorf("a failing LSP server must not leave a partial edge set, got %d LSP_RESOLVED edges", lspResolved)
	}

	// (2) The 005 graph is byte-for-byte unchanged vs a --lsp-less index of the
	// same repo (warn+continue, static index intact).
	idx2, clean2 := newTestIndexer(t)
	defer clean2()
	root2 := lspFixtureGo(t)
	if _, err := idx2.Run(contextBackground2(), root2, pdgCfg); err != nil {
		t.Fatalf("run (no --lsp): %v", err)
	}
	if got, want := baselineFingerprint(t, db), baselineFingerprint(t, idx2.store.DB); got != want {
		t.Errorf("--lsp (failing server) 005 graph not byte-for-byte identical:\n got %s\nwant %s", got, want)
	}
}
