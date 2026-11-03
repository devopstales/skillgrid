package codeindex

import (
	"context"
	"testing"
)

// TestTaintReproducible covers @step-02 (02.6, Scenario: Repeated taint runs
// are reproducible): indexing the same fixture repo with --pdg twice (fresh
// store each time, same source/sink config) yields a byte-for-byte identical
// set of taint_findings (source/sink/kind/path-label/note, ordering-
// independent) — no nondeterminism leaks into findings.
func TestTaintReproducible(t *testing.T) {
	run := func() string {
		ResetFileFirstSymbol()
		idx, clean := newTestIndexer(t)
		defer clean()
		idx = idx.EnablePDG()
		cfg := pdgCfg
		cfg.PDG = true
		root := writeTaintFixtureDir(t)
		if _, err := idx.Run(contextBackground2(), root, cfg); err != nil {
			t.Fatalf("run (--pdg): %v", err)
		}
		return taintFingerprint(t, idx.store.DB)
	}
	// A --lsp --pdg run too (the LSP_RESOLVED boundary must also be stable).
	runLsp := func() string {
		ResetFileFirstSymbol()
		t.Setenv("PATH", t.TempDir())
		idx, clean := newTestIndexer(t)
		defer clean()
		idx = idx.EnableLSP().EnablePDG()
		idx.WithLSPResolver(func(ctx context.Context, filePath string, line int, receiver, callee string) (string, bool) {
			if receiver == "s" && callee == "Do" {
				return "Do", true
			}
			return "", false
		})
		cfg := pdgCfg
		cfg.PDG = true
		cfg.LSP = true
		root := writeLspTaintFixtureDir(t)
		if _, err := idx.Run(contextBackground2(), root, cfg); err != nil {
			t.Fatalf("run (--lsp --pdg): %v", err)
		}
		return taintFingerprint(t, idx.store.DB)
	}

	// (1) Two --pdg runs of the same fixture are byte-for-byte identical.
	first := run()
	second := run()
	if first != second {
		t.Errorf("--pdg taint findings not reproducible across fresh stores:\n got %s\nwant %s", first, second)
	}
	if first == "" {
		t.Errorf("expected non-empty taint findings to compare, got empty fingerprint")
	}

	// (2) Two --lsp --pdg runs are byte-for-byte identical (the LSP boundary is
	// deterministic too).
	firstLsp := runLsp()
	secondLsp := runLsp()
	if firstLsp != secondLsp {
		t.Errorf("--lsp --pdg taint findings not reproducible across fresh stores:\n got %s\nwant %s", firstLsp, secondLsp)
	}
}
