package codeindex

import (
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg"
)

// TestTaintSourceToSink covers @step-02 (02.2, Scenario: Source to sink taint
// path found): a fixture with a known source (request param) that flows to a
// known sink (SQL exec) through a resolvable data-dependence chain. (1) the
// solver returns a finding with the source kind, sink kind, and the hop-by-hop
// path; (2) the finding is persisted in taint_findings.
func TestTaintSourceToSink(t *testing.T) {
	// (1) Unit: the solver returns a finding with source/sink + hop-by-hop path.
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
	if len(findings) == 0 {
		t.Fatalf("expected a source->sink taint finding, got none")
	}
	// The known flow: GetRequestParam (request_param source) -> sqlExec
	// (sql_exec sink).
	var found bool
	for _, f := range findings {
		if f.SourceName != "GetRequestParam" || f.SinkName != "sqlExec" {
			continue
		}
		found = true
		// The finding carries the SOURCE KIND and SINK KIND (the
		// configurable-set class it matched).
		if f.SourceKind != "request_param" {
			t.Errorf("expected source kind request_param, got %q", f.SourceKind)
		}
		if f.SinkKind != "sql_exec" {
			t.Errorf("expected sink kind sql_exec, got %q", f.SinkKind)
		}
		// The path is hop-by-hop (source -> Transform -> sink), non-empty.
		if len(f.Path) < 2 {
			t.Errorf("expected a non-empty hop-by-hop path, got %d hops", len(f.Path))
		}
		// The source hop is the first; the sink hop is the last.
		if f.Path[0].Name != "GetRequestParam" {
			t.Errorf("expected the first hop to be the source, got %q", f.Path[0].Name)
		}
		last := f.Path[len(f.Path)-1]
		if last.Name != "sqlExec" {
			t.Errorf("expected the last hop to be the sink, got %q", last.Name)
		}
		// The path is complete (reaches the sink, not truncated).
		if f.StopsAt != "" {
			t.Errorf("expected a complete source->sink path, got truncated: %s", f.StopsAt)
		}
		// Every hop carries a confidence label.
		for i, h := range f.Path {
			if h.Confidence == "" {
				t.Errorf("hop %d carries an empty confidence label", i)
			}
		}
	}
	if !found {
		t.Errorf("expected a GetRequestParam -> sqlExec finding, got: %v", findings)
	}

	// (2) Persisted: a --pdg index of the same fixture writes taint_findings.
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()
	idx = idx.EnablePDG()
	root := writeTaintFixtureDir(t)
	if _, err := idx.Run(contextBackground2(), root, pdgCfg); err != nil {
		t.Fatalf("run (--pdg): %v", err)
	}
	db := idx.store.DB
	var persisted int
	if err := db.QueryRow(`SELECT COUNT(*) FROM taint_findings WHERE source_name LIKE '%GetRequestParam%' AND sink_name LIKE '%sqlExec%'`).Scan(&persisted); err != nil {
		t.Fatalf("count persisted taint: %v", err)
	}
	if persisted == 0 {
		t.Errorf("expected the source->sink finding to be persisted in taint_findings, got 0")
	}
	// The persisted row carries the path label (confidence) and a note.
	var conf, note string
	if err := db.QueryRow(`SELECT confidence, note FROM taint_findings WHERE source_name LIKE '%GetRequestParam%' AND sink_name LIKE '%sqlExec%' LIMIT 1`).Scan(&conf, &note); err != nil {
		t.Fatalf("scan persisted taint: %v", err)
	}
	if conf == "" {
		t.Errorf("persisted taint finding has an empty confidence/path-label")
	}
	_ = note
}
