package codeindex

import (
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg"
)

// TestTaintConfigurableSets covers @step-02 (02.10, Scenario: Source and sink
// sets are configurable): the solver's source/sink sets are deterministic and
// configurable — (1) the default set finds the built-in flows; (2) an
// overridden source/sink set changes what the solver matches (a finding under
// one config is not a finding under another); (3) an empty set finds nothing
// (never a fabricated match).
func TestTaintConfigurableSets(t *testing.T) {
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
	in := taintInput(1, rows, calls)

	// (1) The default set finds the built-in GetRequestParam -> sqlExec flow.
	defFindings := pdg.Taint(in, pdg.DefaultTaintConfig())
	var defFound bool
	for _, f := range defFindings {
		if f.SourceName == "GetRequestParam" && f.SinkName == "sqlExec" {
			defFound = true
		}
	}
	if !defFound {
		t.Errorf("default set: expected a GetRequestParam -> sqlExec finding, got: %v", defFindings)
	}

	// (2) An overridden source/sink set changes what matches: a config whose
	// sources/sinks do NOT include request_param/sql_exec finds the flow NOT
	// (the same data, different config).
	other := pdg.TaintConfig{
		Sources: []pdg.SourceKind{
			{Name: "env_var", Description: "env", MatchSource: func(name, note string) bool { return name == "GetEnv" }},
		},
		Sinks: []pdg.SinkKind{
			{Name: "file_write", Description: "write", MatchSink: func(name, note string) bool { return name == "WriteFile" }},
		},
	}
	otherFindings := pdg.Taint(in, other)
	for _, f := range otherFindings {
		if f.SourceName == "GetRequestParam" && f.SinkName == "sqlExec" {
			t.Errorf("overridden set: GetRequestParam -> sqlExec should not match (not in the config), got: %v", f)
		}
	}

	// (3) An empty set finds nothing (never a fabricated match).
	empty := pdg.TaintConfig{Sources: []pdg.SourceKind{}, Sinks: []pdg.SinkKind{}}
	emptyFindings := pdg.Taint(in, empty)
	if len(emptyFindings) != 0 {
		t.Errorf("empty set: expected no findings, got %d", len(emptyFindings))
	}

	// Determinism: the default config is stable across calls (same input).
	a := pdg.Taint(in, pdg.DefaultTaintConfig())
	b := pdg.Taint(in, pdg.DefaultTaintConfig())
	if len(a) != len(b) {
		t.Errorf("default config not deterministic: %d vs %d findings", len(a), len(b))
	}
}
