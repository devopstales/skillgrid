package codeindex

import (
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg"
)

// boundaryFixtureDir writes a fixture where a source (env var) reaches an
// unresolved cross-file boundary (s.Do in a second file) — no LSP_RESOLVED
// edge, so the path truncates at the boundary.
func boundaryFixtureDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, root+"/main.go",
		"package main\n\nfunc handler() {\n\ts := NewSvc()\n\tv := GetEnv(\"K\")\n\ts.Do(v)\n}\n\nfunc GetEnv(k string) string { return k }\n")
	mustWrite(t, root+"/svc.go",
		"package main\n\ntype Svc struct{}\n\nfunc NewSvc() *Svc { return &Svc{} }\n\nfunc (x *Svc) Do(n string) {}\n")
	return root
}

// TestTaintBoundaryNotFabricated covers @step-02 (02.4, Scenario: Taint path
// stops at an unresolved boundary): (1) a fixture where a source flows to a
// sink only through an unresolved call boundary (no LSP_RESOLVED edge) is
// reported with the path truncated at the boundary, the boundary hop marked
// AMBIGUOUS, and a "stops at <boundary>" note; (2) a fixture where a source
// has no path to any sink produces NO finding (never fabricated); (3) no
// finding is ever reported with an empty hop list.
func TestTaintBoundaryNotFabricated(t *testing.T) {
	// (1) Unresolved boundary: truncated + AMBIGUOUS + stops-at note.
	{
		// Unit: a source flows through an unresolved boundary (no LSP edge).
		src := []byte("package main\n\nfunc handler() {\n\tv := GetEnv(\"K\")\n\tf(v)\n}\n\nfunc GetEnv(k string) string { return k }\n\nfunc f(v string) {}\n")
		c, err := pdg.BuildCFG(src, "go", "handler", 3)
		if err != nil {
			t.Fatalf("BuildCFG: %v", err)
		}
		// f is unresolved (no LSP_RESOLVED edge; the static pass left it
		// name-only — to_id NULL).
		calls := []pdg.CallSite{
			{Line: 4, Name: "GetEnv", Resolved: true},
			{Line: 5, Name: "f", Resolved: false},
		}
		rows, err := pdg.Build(1, c, calls)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		findings := pdg.Taint(taintInput(1, rows, calls), pdg.DefaultTaintConfig())
		if len(findings) == 0 {
			t.Fatalf("expected a truncated finding at the unresolved boundary, got none")
		}
		var found bool
		for _, f := range findings {
			if f.SourceName != "GetEnv" {
				continue
			}
			found = true
			// The path is truncated at the boundary (stops-at note).
			if f.StopsAt == "" {
				t.Errorf("expected a stops-at note on the truncated finding, got none")
			}
			if f.StopsAt != "stops at f" {
				t.Errorf("expected 'stops at f', got %q", f.StopsAt)
			}
			// The boundary hop is AMBIGUOUS.
			last := f.Path[len(f.Path)-1]
			if last.Confidence != pdg.ConfidenceAmbiguous {
				t.Errorf("the boundary hop is %q, want AMBIGUOUS", last.Confidence)
			}
			// The path is NOT EXTRACTED (it has an AMBIGUOUS hop).
			if f.PathLabel == pdg.ConfidenceExtracted {
				t.Errorf("a truncated path must not be EXTRACTED, got EXTRACTED")
			}
		}
		if !found {
			t.Errorf("expected a GetEnv -> boundary finding, got: %v", findings)
		}
	}

	// (2) No path to any sink: NO finding (never fabricated).
	{
		src := []byte("package main\n\nfunc handler() {\n\tv := GetEnv(\"K\")\n\tlog(v)\n}\n\nfunc GetEnv(k string) string { return k }\n\nfunc log(v string) {}\n")
		c, err := pdg.BuildCFG(src, "go", "handler", 3)
		if err != nil {
			t.Fatalf("BuildCFG: %v", err)
		}
		// log is a resolved call that is NOT a sink (no configured sink class
		// matches it), and there is no other sink.
		calls := []pdg.CallSite{
			{Line: 4, Name: "GetEnv", Resolved: true},
			{Line: 5, Name: "log", Resolved: true},
		}
		rows, err := pdg.Build(1, c, calls)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		findings := pdg.Taint(taintInput(1, rows, calls), pdg.DefaultTaintConfig())
		// GetEnv is a source; log is not a sink and is resolved (no
		// truncation). So there is no source->sink path and no truncated
		// boundary — no finding.
		for _, f := range findings {
			t.Errorf("expected no finding (no path to a sink), got: src=%s sink=%s label=%s stops=%q", f.SourceName, f.SinkName, f.PathLabel, f.StopsAt)
		}
	}

	// (3) No finding with an empty hop list (across all fixtures).
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
		for _, f := range findings {
			if len(f.Path) == 0 {
				t.Errorf("a finding with an empty hop list is fabricated: %+v", f)
			}
		}
	}

	// (1 cont.) Integration: a --pdg index of an unresolved cross-file boundary
	// persists a truncated finding (AMBIGUOUS + stops-at note), not a complete
	// source->sink.
	{
		ResetFileFirstSymbol()
		idx, clean := newTestIndexer(t)
		defer clean()
		idx = idx.EnablePDG()
		cfg := pdgCfg
		cfg.PDG = true
		root := boundaryFixtureDir(t)
		if _, err := idx.Run(contextBackground2(), root, cfg); err != nil {
			t.Fatalf("run (--pdg): %v", err)
		}
		db := idx.store.DB
		// No LSP_RESOLVED edge (no --lsp).
		var lsp int
		if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE confidence='LSP_RESOLVED'`).Scan(&lsp); err != nil {
			t.Fatalf("count LSP_RESOLVED: %v", err)
		}
		if lsp != 0 {
			t.Fatalf("expected 0 LSP_RESOLVED edges (no --lsp), got %d", lsp)
		}
		// A truncated finding (stops at the boundary) is persisted; the
		// boundary hop is AMBIGUOUS (the persisted path label).
		var truncated, complete int
		if err := db.QueryRow(`SELECT COUNT(*) FROM taint_findings WHERE note LIKE 'stops at%'`).Scan(&truncated); err != nil {
			t.Fatalf("count truncated: %v", err)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM taint_findings WHERE confidence='EXTRACTED' AND (note='' OR note NOT LIKE 'stops at%')`).Scan(&complete); err != nil {
			t.Fatalf("count complete: %v", err)
		}
		if truncated == 0 {
			t.Errorf("--pdg (unresolved boundary): expected a truncated finding (stops-at note), got 0")
		}
		if complete != 0 {
			t.Errorf("--pdg (unresolved boundary): expected no complete EXTRACTED source->sink, got %d", complete)
		}
		// No persisted finding has an empty path (the solver never emits one).
		var empty int
		if err := db.QueryRow(`SELECT COUNT(*) FROM taint_findings WHERE source_name = '' AND sink_name = '' AND note = ''`).Scan(&empty); err != nil {
			t.Fatalf("count empty: %v", err)
		}
		if empty != 0 {
			t.Errorf("found %d findings with no source/sink/note (fabricated), want 0", empty)
		}
	}
}
