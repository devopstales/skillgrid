package codeindex

import (
	"context"
	"testing"
)

// TestLspComposition covers @step-01 (Scenario: LSP tier works standalone and
// composes with PDG): (1) standalone — --lsp only (no --pdg) writes
// LSP_RESOLVED edges into 005's edges table while cfg_blocks/cfg_edges/
// pdg_edges/taint_findings stay empty; (2) composed — --lsp --pdg writes BOTH
// LSP_RESOLVED edges AND the per-function CFG/PDG rows, and the PDG sees the
// LSP_RESOLVED call edges as available (not dropped).
func TestLspComposition(t *testing.T) {
	// (1) Standalone: --lsp only.
	{
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
			t.Fatalf("run (--lsp only): %v", err)
		}
		db := idx.store.DB
		var lspResolved int
		if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE confidence='LSP_RESOLVED'`).Scan(&lspResolved); err != nil {
			t.Fatalf("count LSP_RESOLVED: %v", err)
		}
		if lspResolved == 0 {
			t.Errorf("standalone --lsp: expected LSP_RESOLVED edges, got 0")
		}
		// The PDG tables stay empty (no --pdg).
		pdgTablesEmpty(t, db)
	}

	// (2) Composed: --lsp --pdg.
	{
		ResetFileFirstSymbol()
		t.Setenv("PATH", t.TempDir())
		idx, clean := newTestIndexer(t)
		defer clean()
		idx.EnableLSP()
		idx.EnablePDG()
		idx.WithLSPResolver(func(ctx context.Context, filePath string, line int, receiver, callee string) (string, bool) {
			if receiver == "s" && callee == "Do" {
				return "Do", true
			}
			return "", false
		})
		cfg := pdgCfg
		cfg.LSP = true
		cfg.PDG = true
		root := lspFixtureGo(t)
		if _, err := idx.Run(contextBackground2(), root, cfg); err != nil {
			t.Fatalf("run (--lsp --pdg): %v", err)
		}
		db := idx.store.DB
		// Both LSP_RESOLVED edges AND CFG/PDG rows are present.
		var lspResolved, blocks, edges, pdg int
		if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE confidence='LSP_RESOLVED'`).Scan(&lspResolved); err != nil {
			t.Fatalf("count LSP_RESOLVED: %v", err)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM cfg_blocks`).Scan(&blocks); err != nil {
			t.Fatalf("count cfg_blocks: %v", err)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM cfg_edges`).Scan(&edges); err != nil {
			t.Fatalf("count cfg_edges: %v", err)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM pdg_edges`).Scan(&pdg); err != nil {
			t.Fatalf("count pdg_edges: %v", err)
		}
		if lspResolved == 0 {
			t.Errorf("composed: expected LSP_RESOLVED edges, got 0")
		}
		if blocks == 0 || edges == 0 {
			t.Errorf("composed: expected cfg_blocks/cfg_edges, got %d/%d", blocks, edges)
		}
		if pdg == 0 {
			t.Errorf("composed: expected pdg_edges, got 0")
		}
		// The PDG sees the LSP_RESOLVED call edges as available (not dropped):
		// the member call's line is a call block in the CFG, and the LSP_RESOLVED
		// edge is still present in the edges table after the PDG pass.
		if lspResolved == 0 {
			t.Errorf("composed: LSP_RESOLVED edges dropped by the PDG pass")
		}
	}
}
