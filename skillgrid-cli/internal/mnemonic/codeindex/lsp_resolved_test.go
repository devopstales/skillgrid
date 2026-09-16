package codeindex

import (
	"context"
	"testing"
)

// TestLspResolvedEdges covers @step-01 (Scenario: LSP index adds
// LSP_RESOLVED member-call edges): with a deterministic fake server (a
// hermetic per-call resolver, no gopls on PATH), indexing a fixture with --lsp
// whose member call the static pass cannot type yields (1) an LSP_RESOLVED
// edge in 005's edges table for that member call, (2) each carrying the
// LSP_RESOLVED confidence label (the fourth value), and (3) the static edges
// for calls the static pass already resolved are not duplicated/downgraded.
func TestLspResolvedEdges(t *testing.T) {
	ResetFileFirstSymbol()
	// Isolate PATH so the test does not depend on a real server; the resolver
	// seam (below) does the work.
	t.Setenv("PATH", t.TempDir())

	// The fixture has a member call `s.Do(1)` the static pass stores as a
	// calls edge to_name="Do" (receiver-bound, not typed). The fake resolver
	// resolves it to the `Do` method symbol.
	idx, clean := newTestIndexer(t)
	defer clean()
	idx.EnableLSP()
	idx.WithLSPResolver(func(ctx context.Context, filePath string, line int, receiver, callee string) (string, bool) {
		if receiver == "s" && callee == "Do" {
			return "Do", true // resolved to the Do method symbol
		}
		return "", false
	})
	cfg := pdgCfg
	cfg.LSP = true
	root := lspFixtureGo(t)
	if _, err := idx.Run(contextBackground2(), root, cfg); err != nil {
		t.Fatalf("run (--lsp + fake resolver): %v", err)
	}
	db := idx.store.DB

	// (1) An LSP_RESOLVED edge exists for the member call.
	var lspResolved int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE confidence = 'LSP_RESOLVED'`).Scan(&lspResolved); err != nil {
		t.Fatalf("count LSP_RESOLVED: %v", err)
	}
	if lspResolved == 0 {
		t.Fatalf("expected LSP_RESOLVED edges for the member call, got 0")
	}

	// (2) Every edge carries a valid confidence label (the fourth value
	// LSP_RESOLVED is accepted).
	var nonLSP int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE confidence NOT IN ('EXTRACTED','INFERRED','AMBIGUOUS','LSP_RESOLVED')`).Scan(&nonLSP); err != nil {
		t.Fatalf("count non-confidence: %v", err)
	}
	if nonLSP != 0 {
		t.Errorf("found %d edges with an invalid confidence label", nonLSP)
	}

	// (3) Static edges the static pass already resolved are not duplicated:
	// the number of calls edges to a name the static pass resolved (newSvc) is
	// unchanged — LSP only adds edges for the member call.
	var newSvcCalls int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind='calls' AND to_name='newSvc'`).Scan(&newSvcCalls); err != nil {
		t.Fatalf("count newSvc calls: %v", err)
	}
	if newSvcCalls != 1 {
		t.Errorf("expected exactly 1 static call edge to newSvc (not duplicated), got %d", newSvcCalls)
	}
}

// TestLspResolvedBoundaryNotAmbiguous asserts that a call boundary resolved by
// the LSP tier is LSP_RESOLVED, not re-marked AMBIGUOUS (01.7 / 01.4 part 4).
func TestLspResolvedBoundaryNotAmbiguous(t *testing.T) {
	ResetFileFirstSymbol()
	t.Setenv("PATH", t.TempDir())
	idx, clean := newTestIndexer(t)
	defer clean()
	idx.EnableLSP()
	idx.WithLSPResolver(func(ctx context.Context, filePath string, line int, receiver, callee string) (string, bool) {
		if receiver == "s" && callee == "Do" {
			return "Do", true
		}
		return "", false
	})
	cfg := pdgCfg
	cfg.LSP = true
	root := lspFixtureGo(t)
	if _, err := idx.Run(contextBackground2(), root, cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	db := idx.store.DB
	var lsp, ambig int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind='calls' AND line=7 AND confidence='LSP_RESOLVED'`).Scan(&lsp); err != nil {
		t.Fatalf("count lsp: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind='calls' AND line=7 AND to_name='Do' AND confidence='AMBIGUOUS'`).Scan(&ambig); err != nil {
		t.Fatalf("count ambig: %v", err)
	}
	if lsp == 0 {
		t.Errorf("expected an LSP_RESOLVED edge at the member-call line, got 0")
	}
	if ambig != 0 {
		t.Errorf("the LSP-resolved boundary must not be AMBIGUOUS, got %d AMBIGUOUS edges", ambig)
	}
}
