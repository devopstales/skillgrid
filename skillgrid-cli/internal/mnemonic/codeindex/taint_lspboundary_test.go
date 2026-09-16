package codeindex

import (
	"context"
	"database/sql"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg"
)

// lspTaintFixtureGo is a Go repo where a source (env var read) reaches a sink
// (file write) ONLY through a member-call boundary (s.Do) that the static pass
// cannot resolve (the Do method is in another file) but the --lsp tier does
// (an LSP_RESOLVED edge). Without --lsp the boundary is unresolved (the path
// truncates); with --lsp the path continues through it.
func writeLspTaintFixtureDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, root+"/main.go",
		"package main\n\nfunc handler() {\n\ts := NewSvc()\n\tv := GetEnv(\"K\")\n\ts.Do(v)\n\tWriteFile(\"out.txt\", v)\n}\n\nfunc GetEnv(k string) string { return k }\n\nfunc WriteFile(p, d string) {}\n")
	// The Do method + Svc type live in a SECOND file, so the static pass
	// (single-file) cannot resolve s.Do — only the --lsp tier can.
	mustWrite(t, root+"/svc.go",
		"package main\n\ntype Svc struct{}\n\nfunc NewSvc() *Svc { return &Svc{} }\n\nfunc (x *Svc) Do(n string) {}\n")
	return root
}

// TestTaintLspResolvedBoundary covers @step-02 (02.3, Scenario: Taint path
// continues through an LSP_RESOLVED boundary): (1) the same fixture indexed
// WITH --lsp --pdg continues through the s.Do boundary to the sink and reports
// a complete source->sink finding (not truncated), with the boundary hop
// labeled LSP_RESOLVED (not AMBIGUOUS); (2) the same fixture indexed WITHOUT
// --lsp (no LSP_RESOLVED edge) still truncates at the boundary with an
// AMBIGUOUS/"stops at" note — proving the LSP layer is a feeder, not a
// dependency.
func TestTaintLspResolvedBoundary(t *testing.T) {
	// (1) --lsp --pdg: the path continues through the s.Do boundary.
	{
		ResetFileFirstSymbol()
		t.Setenv("PATH", t.TempDir()) // no real server; the seam does the work
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
		cfg.LSP = true
		cfg.PDG = true
		root := writeLspTaintFixtureDir(t)
		if _, err := idx.Run(contextBackground2(), root, cfg); err != nil {
			t.Fatalf("run (--lsp --pdg): %v", err)
		}
		db := idx.store.DB
		// An LSP_RESOLVED edge exists for the s.Do boundary.
		var lsp int
		if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind='calls' AND to_name='Do' AND confidence='LSP_RESOLVED'`).Scan(&lsp); err != nil {
			t.Fatalf("count LSP_RESOLVED: %v", err)
		}
		if lsp == 0 {
			t.Fatalf("expected an LSP_RESOLVED edge for s.Do, got 0")
		}
		// A complete (non-truncated) source->sink finding exists.
		var complete int
		if err := db.QueryRow(`SELECT COUNT(*) FROM taint_findings WHERE sink_name LIKE '%WriteFile%' AND (note = '' OR note NOT LIKE 'stops at%')`).Scan(&complete); err != nil {
			t.Fatalf("count complete taint: %v", err)
		}
		if complete == 0 {
			t.Errorf("--lsp: expected a complete source->sink finding (path continues through the LSP boundary), got 0")
		}
		// The boundary hop is LSP_RESOLVED, not AMBIGUOUS (unit-level: the
		// persisted path label is LSP_RESOLVED, not AMBIGUOUS).
		var labels []string
		rows, err := db.Query(`SELECT confidence FROM taint_findings WHERE sink_name LIKE '%WriteFile%'`)
		if err != nil {
			t.Fatalf("query taint labels: %v", err)
		}
		for rows.Next() {
			var l string
			rows.Scan(&l)
			labels = append(labels, l)
		}
		rows.Close()
		for _, l := range labels {
			if l == pdg.ConfidenceAmbiguous {
				t.Errorf("--lsp: the LSP-resolved path must not be AMBIGUOUS, got a finding labeled AMBIGUOUS")
			}
		}
		if !hasLabel(labels, pdg.ConfidenceLSPResolved) {
			t.Errorf("--lsp: expected a finding whose path is LSP_RESOLVED (the boundary hop), got labels: %v", labels)
		}
	}

	// (2) --pdg only (no --lsp): the same fixture truncates at the boundary.
	{
		ResetFileFirstSymbol()
		idx, clean := newTestIndexer(t)
		defer clean()
		idx = idx.EnablePDG()
		cfg := pdgCfg
		cfg.PDG = true
		root := writeLspTaintFixtureDir(t)
		if _, err := idx.Run(contextBackground2(), root, cfg); err != nil {
			t.Fatalf("run (--pdg, no --lsp): %v", err)
		}
		db := idx.store.DB
		// No LSP_RESOLVED edge (the static pass left s.Do unresolved).
		var lsp int
		if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind='calls' AND to_name='Do' AND confidence='LSP_RESOLVED'`).Scan(&lsp); err != nil {
			t.Fatalf("count LSP_RESOLVED: %v", err)
		}
		if lsp != 0 {
			t.Errorf("no --lsp: expected 0 LSP_RESOLVED edges, got %d", lsp)
		}
		// No complete finding (the path cannot cross the unresolved boundary).
		var complete int
		if err := db.QueryRow(`SELECT COUNT(*) FROM taint_findings WHERE sink_name LIKE '%WriteFile%' AND (note = '' OR note NOT LIKE 'stops at%')`).Scan(&complete); err != nil {
			t.Fatalf("count complete taint: %v", err)
		}
		if complete != 0 {
			t.Errorf("no --lsp: the unresolved boundary must truncate the path (no complete finding), got %d", complete)
		}
		// A truncated finding (stops at the boundary) exists: the source
		// (GetEnv, an env var) reaches the unresolved s.Do boundary, so the
		// path is reported truncated there (AMBIGUOUS + stops-at note).
		var truncated int
		if err := db.QueryRow(`SELECT COUNT(*) FROM taint_findings WHERE source_name LIKE '%GetEnv%' AND note LIKE 'stops at%'`).Scan(&truncated); err != nil {
			t.Fatalf("count truncated taint: %v", err)
		}
		if truncated == 0 {
			t.Errorf("no --lsp: expected a truncated finding (stops at the boundary), got 0")
		}
	}
}

func hasLabel(labels []string, want string) bool {
	for _, l := range labels {
		if l == want {
			return true
		}
	}
	return false
}

// TestTaintLspResolvedBoundaryUnit pins the per-hop LSP_RESOLVED label at the
// unit level: a data edge into a call site resolved only by the --lsp tier is
// labeled LSP_RESOLVED (a resolved call boundary), not AMBIGUOUS, and the path
// is NOT EXTRACTED (it is a resolved boundary, not a resolved data-dependence).
func TestTaintLspResolvedBoundaryUnit(t *testing.T) {
	src := []byte("package main\n\nfunc handler() {\n\tv := GetEnv(\"K\")\n\tf(v)\n\tWriteFile(\"out.txt\", v)\n}\n\nfunc GetEnv(k string) string { return k }\n\nfunc f(v string) {}\n\nfunc WriteFile(p, d string) {}\n")
	c, err := pdg.BuildCFG(src, "go", "handler", 3)
	if err != nil {
		t.Fatalf("BuildCFG: %v", err)
	}
	// f is LSP-resolved (a resolved call boundary); the sink WriteFile is
	// statically resolved.
	calls := []pdg.CallSite{
		{Line: 4, Name: "GetEnv", Resolved: true},
		{Line: 5, Name: "f", LSPResolved: true, Resolved: true},
		{Line: 6, Name: "WriteFile", Resolved: true},
	}
	rows, err := pdg.Build(1, c, calls)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	findings := pdg.Taint(taintInput(1, rows, calls), pdg.DefaultTaintConfig())
	if len(findings) == 0 {
		t.Fatalf("expected a source->sink taint finding through the LSP boundary, got none")
	}
	var checked bool
	for _, f := range findings {
		if f.SinkName != "WriteFile" {
			continue
		}
		checked = true
		// The boundary hop (into f) is LSP_RESOLVED, not AMBIGUOUS.
		sawLsp := false
		for _, h := range f.Path {
			if h.Name == "f" {
				if h.Confidence != pdg.ConfidenceLSPResolved {
					t.Errorf("the LSP-resolved boundary hop is %q, want LSP_RESOLVED", h.Confidence)
				}
				sawLsp = true
			}
			if h.Confidence == pdg.ConfidenceAmbiguous {
				t.Errorf("the LSP-resolved path must not carry an AMBIGUOUS hop, got one at line %d", h.Line)
			}
		}
		if !sawLsp {
			t.Errorf("expected the path to carry an LSP_RESOLVED boundary hop, got: %v", f.Path)
		}
		// The path is NOT EXTRACTED (a resolved boundary is not a resolved
		// data-dependence).
		if f.PathLabel == pdg.ConfidenceExtracted {
			t.Errorf("a path with an LSP_RESOLVED hop must not be EXTRACTED, got EXTRACTED")
		}
		// The path is complete (continues through the boundary to the sink).
		if f.StopsAt != "" {
			t.Errorf("the LSP-resolved boundary must not truncate the path, got: %s", f.StopsAt)
		}
	}
	if !checked {
		t.Errorf("expected a finding ending at the WriteFile sink, got: %v", findings)
	}
}

var _ = sql.ErrNoRows
