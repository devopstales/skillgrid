package codeindex

import (
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg"
)

// validTaintLabel reports whether a confidence label is one of the four
// allowed taint labels.
func validTaintLabel(l string) bool {
	switch l {
	case pdg.ConfidenceExtracted, pdg.ConfidenceInferred, pdg.ConfidenceAmbiguous, pdg.ConfidenceLSPResolved:
		return true
	}
	return false
}

// TestTaintConfidenceLabels covers @step-02 (02.5, Scenario: Every taint hop
// carries a confidence label): (1) every hop in every taint_findings path has a
// non-empty Confidence Label in EXTRACTED | INFERRED | AMBIGUOUS |
// LSP_RESOLVED; (2) a fixture where every hop is a resolved data-dependence →
// the path is EXTRACTED; (3) a fixture where at least one hop is an unresolved
// boundary → the path is INFERRED or AMBIGUOUS, never EXTRACTED; (4) a fixture
// where a hop is resolved only via LSP_RESOLVED → that hop is LSP_RESOLVED and
// the path is not EXTRACTED.
func TestTaintConfidenceLabels(t *testing.T) {
	// (1) Every hop in every finding carries a valid, non-empty label.
	cases := map[string]struct {
		src   string
		calls []pdg.CallSite
	}{
		"resolved-chain": {
			src: "package main\n\nfunc handler(r *Req) {\n\tp := GetRequestParam(r, \"q\")\n\tu := Transform(p)\n\tsqlExec(u)\n}\n\nfunc GetRequestParam(r *Req, k string) string { return r.Path }\n\nfunc Transform(s string) string { return s }\n\nfunc sqlExec(s string) {}\n",
			calls: []pdg.CallSite{
				{Line: 4, Name: "GetRequestParam", Resolved: true},
				{Line: 5, Name: "Transform", Resolved: true},
				{Line: 6, Name: "sqlExec", Resolved: true},
			},
		},
		"unresolved-boundary": {
			src: "package main\n\nfunc handler() {\n\tv := GetEnv(\"K\")\n\tf(v)\n\tWriteFile(\"out\", v)\n}\n\nfunc GetEnv(k string) string { return k }\n\nfunc f(v string) {}\n\nfunc WriteFile(p, d string) {}\n",
			calls: []pdg.CallSite{
				{Line: 4, Name: "GetEnv", Resolved: true},
				{Line: 5, Name: "f", Resolved: false},
				{Line: 6, Name: "WriteFile", Resolved: true},
			},
		},
		"lsp-boundary": {
			src: "package main\n\nfunc handler() {\n\tv := GetEnv(\"K\")\n\tf(v)\n\tWriteFile(\"out\", v)\n}\n\nfunc GetEnv(k string) string { return k }\n\nfunc f(v string) {}\n\nfunc WriteFile(p, d string) {}\n",
			calls: []pdg.CallSite{
				{Line: 4, Name: "GetEnv", Resolved: true},
				{Line: 5, Name: "f", Resolved: true, LSPResolved: true},
				{Line: 6, Name: "WriteFile", Resolved: true},
			},
		},
	}
	for name, tc := range cases {
		c, err := pdg.BuildCFG([]byte(tc.src), "go", "handler", 3)
		if err != nil {
			t.Fatalf("%s: BuildCFG: %v", name, err)
		}
		rows, err := pdg.Build(1, c, tc.calls)
		if err != nil {
			t.Fatalf("%s: Build: %v", name, err)
		}
		findings := pdg.Taint(taintInput(1, rows, tc.calls), pdg.DefaultTaintConfig())
		for _, f := range findings {
			// (1) every hop has a valid, non-empty label.
			for i, h := range f.Path {
				if h.Confidence == "" {
					t.Errorf("%s: hop %d has an empty confidence label", name, i)
				}
				if !validTaintLabel(h.Confidence) {
					t.Errorf("%s: hop %d has an invalid confidence label %q", name, i, h.Confidence)
				}
			}
			// The path label is also a valid, non-empty label.
			if f.PathLabel == "" || !validTaintLabel(f.PathLabel) {
				t.Errorf("%s: path label %q is not a valid taint label", name, f.PathLabel)
			}
		}
	}

	// (2) All hops resolved data-dependences → path is EXTRACTED.
	{
		src := []byte("package main\n\nfunc handler(r *Req) {\n\tp := GetRequestParam(r, \"q\")\n\tu := Transform(p)\n\tsqlExec(u)\n}\n\nfunc GetRequestParam(r *Req, k string) string { return r.Path }\n\nfunc Transform(s string) string { return s }\n\nfunc sqlExec(s string) {}\n")
		c, err := pdg.BuildCFG(src, "go", "handler", 3)
		if err != nil {
			t.Fatalf("BuildCFG: %v", err)
		}
		calls := []pdg.CallSite{
			{Line: 4, Name: "GetRequestParam", Resolved: true},
			{Line: 5, Name: "Transform", Resolved: true},
			{Line: 6, Name: "sqlExec", Resolved: true},
		}
		rows, err := pdg.Build(1, c, calls)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		findings := pdg.Taint(taintInput(1, rows, calls), pdg.DefaultTaintConfig())
		var found bool
		for _, f := range findings {
			if f.SourceName != "GetRequestParam" || f.SinkName != "sqlExec" {
				continue
			}
			found = true
			if f.PathLabel != pdg.ConfidenceExtracted {
				t.Errorf("an all-resolved path should be EXTRACTED, got %q", f.PathLabel)
			}
			for _, h := range f.Path {
				if h.Confidence != pdg.ConfidenceExtracted {
					t.Errorf("an all-resolved path should have only EXTRACTED hops, got %q at line %d", h.Confidence, h.Line)
				}
			}
		}
		if !found {
			t.Errorf("expected an all-resolved GetRequestParam -> sqlExec finding, got: %v", findings)
		}
	}

	// (3) An unresolved boundary → path is INFERRED or AMBIGUOUS, never
	// EXTRACTED.
	{
		src := []byte("package main\n\nfunc handler() {\n\tv := GetEnv(\"K\")\n\tf(v)\n\tWriteFile(\"out\", v)\n}\n\nfunc GetEnv(k string) string { return k }\n\nfunc f(v string) {}\n\nfunc WriteFile(p, d string) {}\n")
		c, err := pdg.BuildCFG(src, "go", "handler", 3)
		if err != nil {
			t.Fatalf("BuildCFG: %v", err)
		}
		calls := []pdg.CallSite{
			{Line: 4, Name: "GetEnv", Resolved: true},
			{Line: 5, Name: "f", Resolved: false},
			{Line: 6, Name: "WriteFile", Resolved: true},
		}
		rows, err := pdg.Build(1, c, calls)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		findings := pdg.Taint(taintInput(1, rows, calls), pdg.DefaultTaintConfig())
		var found bool
		for _, f := range findings {
			if f.SourceName != "GetEnv" {
				continue
			}
			found = true
			if f.PathLabel == pdg.ConfidenceExtracted {
				t.Errorf("a path with an unresolved boundary must not be EXTRACTED, got EXTRACTED")
			}
			if f.PathLabel != pdg.ConfidenceAmbiguous && f.PathLabel != pdg.ConfidenceInferred {
				t.Errorf("a path with an unresolved boundary should be AMBIGUOUS or INFERRED, got %q", f.PathLabel)
			}
		}
		if !found {
			t.Errorf("expected a GetEnv finding through the unresolved boundary, got: %v", findings)
		}
	}

	// (4) A hop resolved only via LSP_RESOLVED → that hop is LSP_RESOLVED and
	// the path is not EXTRACTED.
	{
		src := []byte("package main\n\nfunc handler() {\n\tv := GetEnv(\"K\")\n\tf(v)\n\tWriteFile(\"out\", v)\n}\n\nfunc GetEnv(k string) string { return k }\n\nfunc f(v string) {}\n\nfunc WriteFile(p, d string) {}\n")
		c, err := pdg.BuildCFG(src, "go", "handler", 3)
		if err != nil {
			t.Fatalf("BuildCFG: %v", err)
		}
		calls := []pdg.CallSite{
			{Line: 4, Name: "GetEnv", Resolved: true},
			{Line: 5, Name: "f", Resolved: true, LSPResolved: true},
			{Line: 6, Name: "WriteFile", Resolved: true},
		}
		rows, err := pdg.Build(1, c, calls)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		findings := pdg.Taint(taintInput(1, rows, calls), pdg.DefaultTaintConfig())
		var found bool
		for _, f := range findings {
			if f.SinkName != "WriteFile" {
				continue
			}
			found = true
			sawLsp := false
			for _, h := range f.Path {
				if h.Name == "f" && h.Confidence == pdg.ConfidenceLSPResolved {
					sawLsp = true
				}
			}
			if !sawLsp {
				t.Errorf("expected the f hop to be LSP_RESOLVED, got: %v", f.Path)
			}
			if f.PathLabel == pdg.ConfidenceExtracted {
				t.Errorf("a path with an LSP_RESOLVED hop must not be EXTRACTED, got EXTRACTED")
			}
		}
		if !found {
			t.Errorf("expected a WriteFile finding through the LSP boundary, got: %v", findings)
		}
	}
}
