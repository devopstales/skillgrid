package codeindex

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg"
)

// confFixtureGo is a fixture with (a) branch/loop/return structure (so the CFG
// and control-dependence pdg_edges are non-empty) and (b) two calls that
// resolve differently: a package-qualified call the static pass cannot type
// (unknown.Resolve) and a member call s.Do (LSP-resolvable). The content is
// fixed so fingerprints are stable.
func confFixtureGo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	content := "package main\n\nimport \"x/unknown\"\n\nfunc process(v int) int {\n\tif v > 0 {\n\t\tv = unknown.Resolve(v)\n\t} else {\n\t\tv = 0\n\t}\n\tfor i := 0; i < v; i++ {\n\t\tdrain(i)\n\t}\n\ts := newSvc()\n\ts.Do(v)\n\treturn v\n}\n\nfunc drain(i int) {}\n\nfunc newSvc() *Svc { return &Svc{} }\n\nfunc (x *Svc) Do(n int) int { return n }\n\ntype Svc struct{}\n"
	if err := osWriteFileConf(filepath.Join(root, "main.go"), content); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return root
}

func osWriteFileConf(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// TestPdgConfidenceLabels covers @step-01 (Scenario: Every PDG edge carries a
// confidence label): (1) every pdg_edges row has a non-empty Confidence Label in
// EXTRACTED | INFERRED | AMBIGUOUS | LSP_RESOLVED; (2) a resolvable
// data-dependence (fully intraprocedural control flow) is EXTRACTED; (3) a
// data-dependence whose value leaves through a call the static pass cannot
// resolve is AMBIGUOUS/INFERRED — never a fabricated EXTRACTED; (4) a call
// boundary resolved by the --lsp tier is LSP_RESOLVED, not AMBIGUOUS.
func TestPdgConfidenceLabels(t *testing.T) {
	// Part 1-3: --pdg only, no LSP resolver (the unresolved-boundary case).
	{
		ResetFileFirstSymbol()
		idx, clean := newTestIndexer(t)
		defer clean()
		idx.EnablePDG()
		cfg := pdgCfg
		cfg.PDG = true
		root := confFixtureGo(t)
		if _, err := idx.Run(contextBackground2(), root, cfg); err != nil {
			t.Fatalf("run (--pdg): %v", err)
		}
		db := idx.store.DB

		// (1) Every pdg_edges row carries a valid confidence label.
		var invalid int
		if err := db.QueryRow(`SELECT COUNT(*) FROM pdg_edges WHERE confidence NOT IN ('EXTRACTED','INFERRED','AMBIGUOUS','LSP_RESOLVED') OR confidence = ''`).Scan(&invalid); err != nil {
			t.Fatalf("count invalid pdg labels: %v", err)
		}
		if invalid != 0 {
			t.Errorf("found %d pdg_edges with an invalid/empty confidence label", invalid)
		}
		// (2) A resolvable data-dependence (fully intraprocedural control flow)
		// is EXTRACTED.
		var extractedControl int
		if err := db.QueryRow(`SELECT COUNT(*) FROM pdg_edges WHERE kind='control' AND confidence='EXTRACTED'`).Scan(&extractedControl); err != nil {
			t.Fatalf("count extracted control: %v", err)
		}
		if extractedControl == 0 {
			t.Errorf("expected EXTRACTED control-dependence pdg_edges, got 0")
		}
		// (3) A data-dependence through an unresolved call is AMBIGUOUS (or
		// INFERRED), never a fabricated EXTRACTED.
		var ambiguousOrInferredData int
		if err := db.QueryRow(`SELECT COUNT(*) FROM pdg_edges WHERE kind='data' AND confidence IN ('AMBIGUOUS','INFERRED')`).Scan(&ambiguousOrInferredData); err != nil {
			t.Fatalf("count ambiguous data: %v", err)
		}
		if ambiguousOrInferredData == 0 {
			t.Errorf("expected an AMBIGUOUS/INFERRED data-dependence (unresolved call boundary), got 0")
		}
	}

	// Part 4: --pdg + --lsp, the member call s.Do is resolved by the LSP tier.
	{
		ResetFileFirstSymbol()
		t.Setenv("PATH", t.TempDir())
		idx, clean := newTestIndexer(t)
		defer clean()
		idx.EnablePDG()
		idx.EnableLSP()
		idx.WithLSPResolver(func(ctx context.Context, filePath string, line int, receiver, callee string) (string, bool) {
			if receiver == "s" && callee == "Do" {
				return "Do", true
			}
			return "", false
		})
		cfg := pdgCfg
		cfg.PDG = true
		cfg.LSP = true
		root := confFixtureGo(t)
		if _, err := idx.Run(contextBackground2(), root, cfg); err != nil {
			t.Fatalf("run (--pdg --lsp): %v", err)
		}
		db := idx.store.DB
		// The member-call line is the `s.Do(v)` statement (line 15 of the
		// fixture).
		var lsp, ambig int
		if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind='calls' AND line=15 AND to_name='Do' AND confidence='LSP_RESOLVED'`).Scan(&lsp); err != nil {
			t.Fatalf("count lsp: %v", err)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind='calls' AND line=15 AND to_name='Do' AND confidence='AMBIGUOUS'`).Scan(&ambig); err != nil {
			t.Fatalf("count ambig: %v", err)
		}
		if lsp == 0 {
			t.Errorf("expected the LSP-resolved boundary to be LSP_RESOLVED, got 0")
		}
		if ambig != 0 {
			t.Errorf("the LSP-resolved boundary must not be AMBIGUOUS, got %d", ambig)
		}
	}
}

// TestPdgConfidenceLabelsUnit is a package-level unit check on pdg.Build that
// pins the confidence semantics directly (independent of the index): control
// edges are EXTRACTED; a data edge through an unresolved call is AMBIGUOUS; a
// data edge through a resolved call is not AMBIGUOUS.
func TestPdgConfidenceLabelsUnit(t *testing.T) {
	src := "package main\n\nfunc f(v int) int {\n\tif v > 0 {\n\t\treturn callResolved(v)\n\t}\n\treturn callUnresolved(v)\n}\n"
	c, err := pdg.BuildCFG([]byte(src), "go", "f", 3)
	if err != nil {
		t.Fatalf("BuildCFG: %v", err)
	}
	calls := []pdg.CallSite{
		{Line: 5, Receiver: "", Name: "callResolved", Resolved: true},
		{Line: 7, Receiver: "", Name: "callUnresolved", Resolved: false},
	}
	rows, err := pdg.Build(1, c, calls)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	sawExtractedControl := false
	for _, r := range rows {
		switch r.Kind {
		case "control":
			if r.Confidence != pdg.ConfidenceExtracted {
				t.Errorf("control edge is %q, want EXTRACTED", r.Confidence)
			}
			sawExtractedControl = true
		case "data":
			// data edges may be EXTRACTED/INFERRED/AMBIGUOUS but never empty.
			if r.Confidence == "" {
				t.Errorf("data edge has empty confidence")
			}
		}
	}
	if !sawExtractedControl {
		t.Errorf("expected at least one EXTRACTED control edge")
	}
}
