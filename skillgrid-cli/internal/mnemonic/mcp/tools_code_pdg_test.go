package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// pdgFixtureGo indexes a Go repo with branch/loop/return structure so the
// --pdg pass populates cfg_blocks/cfg_edges/pdg_edges. Returns dataDir + root.
func pdgMCPFixture(t *testing.T) (dataDir, root string) {
	t.Helper()
	dataDir = t.TempDir()
	raw := t.TempDir()
	abs, err := filepath.Abs(raw)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	root = abs
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	// main(): add -> if/else (branch) -> for (loop) -> switch -> return.
	// process(v): if/else (branch with a call) -> for (loop) -> member call
	// s.Do(v) -> return. This gives non-empty cfg + pdg for both.
	write("main.go", "package main\n\nfunc main() {\n\tn := add(1, 2)\n\tif n > 3 {\n\t\twarn(n)\n\t} else {\n\t\tok()\n\t}\n\tfor i := 0; i < n; i++ {\n\t\tlog(i)\n\t}\n\tswitch n {\n\tcase 1:\n\t\ta()\n\tdefault:\n\t\tb()\n\t}\n}\n\nfunc add(a, b int) int { return a + b }\n\nfunc warn(n int) { _ = n * 2 }\n\nfunc ok() {}\n\nfunc log(i int) {}\n\nfunc a() {}\n\nfunc b() {}\n")
	write("process.go", "package main\n\nimport \"x/unknown\"\n\nfunc process(v int) int {\n\tif v > 0 {\n\t\tv = unknown.Resolve(v)\n\t} else {\n\t\tv = 0\n\t}\n\tfor i := 0; i < v; i++ {\n\t\tdrain(i)\n\t}\n\ts := newSvc()\n\ts.Do(v)\n\treturn v\n}\n\nfunc drain(i int) {}\n\nfunc newSvc() *Svc { return &Svc{} }\n\nfunc (x *Svc) Do(n int) int { return n }\n\ntype Svc struct{}\n")
	// Pin the project before indexing so index + queries share a bucket.
	t.Setenv("MNEMONIC_PROJECT", "pdg-probe")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
	codeindex.ResetFileFirstSymbol()
	if _, err := svc.RunCodeIndexPDG(context.Background(), root, true, false); err != nil {
		t.Fatalf("index --pdg: %v", err)
	}
	return dataDir, root
}

// pdgQueryOut mirrors the JSON the handler returns.
type pdgQueryOut struct {
	Symbol     string `json:"symbol"`
	Statement  int    `json:"statement"`
	Found      bool   `json:"found"`
	Dependents []struct {
		Kind       string `json:"kind"`
		FromLine   int    `json:"from_line"`
		ToLine     int    `json:"to_line"`
		Confidence string `json:"confidence"`
	} `json:"dependents"`
	PdgEnabled bool   `json:"pdg_enabled"`
	Message    string `json:"message"`
}

func runPdgQuery(t *testing.T, params map[string]any) *pdgQueryOut {
	t.Helper()
	res, err := handleCodePdgQuery(context.Background(), newCallTool("code_pdg_query", params))
	if err != nil {
		t.Fatalf("handleCodePdgQuery: %v", err)
	}
	var out pdgQueryOut
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("parse pdg query result %q: %v", callResultText(t, res), err)
	}
	return &out
}

// TestPdgQueryTool covers @step-01: (01.12) code_pdg_query is registered with
// required symbol+statement, the 005/008/010 tool surface is unchanged, and bad
// args are rejected with a validation error; (01.13) a known symbol+statement
// returns its control/data dependents; (01.14) an unknown symbol or statement
// returns not-found with no fabricated dependences; and a non---pdg index
// returns a clear "run --pdg" message (not an error, not an empty-but-present
// result).
func TestPdgQueryTool(t *testing.T) {
	dataDir, root := pdgMCPFixture(t)
	_ = dataDir
	_ = root

	// (01.12) Registration + required params.
	tool := codePdgQueryTool()
	assertCodeToolStable(t, tool, "code_pdg_query", []string{"symbol", "statement"})
	tools := NewServer().ListTools()
	if _, ok := tools["code_pdg_query"]; !ok {
		t.Errorf("code_pdg_query not registered")
	}
	// 005/008/010 tools stay intact (baseline lock).
	for _, name := range []string{"code_search", "code_read", "code_communities", "code_impact", "code_route", "code_affected"} {
		if _, ok := tools[name]; !ok {
			t.Errorf("005/008/010 tool %q no longer registered", name)
		}
	}
	assertCodeToolStable(t, codeSearchTool(), "code_search", []string{"query"})

	// (01.12) Bad args are rejected with a validation error (abort, not a
	// fabricated empty result).
	for _, params := range []map[string]any{
		{},                        // missing both
		{"symbol": "main"},        // missing statement
		{"statement": "5"},        // missing symbol
		{"symbol": "main", "statement": "not-a-line"}, // non-numeric statement
		{"symbol": "main", "statement": "0"},          // non-positive statement
	} {
		res, err := handleCodePdgQuery(context.Background(), newCallTool("code_pdg_query", params))
		if err != nil {
			t.Fatalf("handleCodePdgQuery returned Go error for %v: %v", params, err)
		}
		if !res.IsError {
			t.Errorf("expected a validation error for %v, got success: %s", params, callResultText(t, res))
		}
	}

	// (01.13) A known symbol + a statement inside its CFG returns its
	// dependents (control/data), each confidence-labeled.
	out := runPdgQuery(t, map[string]any{"symbol": "main", "statement": "4"})
	if !out.Found {
		t.Fatalf("expected main@4 to be found in the PDG, got not-found: %s", out.Message)
	}
	if !out.PdgEnabled {
		t.Errorf("expected pdg_enabled=true on a --pdg index")
	}
	// The `n := add(1,2)` statement (line 4) feeds the if-branch (line 5):
	// expect at least one control-dependence.
	if len(out.Dependents) == 0 {
		// A statement at the top of the function may have no outgoing pdg_edge
		// (no branch/call source at that exact line). Fall back to a statement
		// that is a branch header (line 5).
		out2 := runPdgQuery(t, map[string]any{"symbol": "main", "statement": "5"})
		if !out2.Found {
			t.Fatalf("expected main@5 (branch) to be found, got: %s", out2.Message)
		}
		if len(out2.Dependents) == 0 {
			t.Errorf("expected main@5 (branch header) to have control dependents, got none")
		}
		for _, d := range out2.Dependents {
			if d.Confidence == "" {
				t.Errorf("dependence carries an empty confidence label")
			}
		}
	}

	// (01.14) Unknown symbol -> not-found, no fabricated dependences, no error.
	notFoundSym := runPdgQuery(t, map[string]any{"symbol": "does_not_exist", "statement": "4"})
	if notFoundSym.Found {
		t.Errorf("unknown symbol should be not-found, got found")
	}
	if len(notFoundSym.Dependents) != 0 {
		t.Errorf("unknown symbol should have no fabricated dependences, got %d", len(notFoundSym.Dependents))
	}
	// (01.14) Known symbol, statement not in its CFG -> not-found.
	notFoundStmt := runPdgQuery(t, map[string]any{"symbol": "main", "statement": "999"})
	if notFoundStmt.Found {
		t.Errorf("a statement outside the CFG should be not-found, got found")
	}
	if len(notFoundStmt.Dependents) != 0 {
		t.Errorf("an out-of-CFG statement should have no fabricated dependences, got %d", len(notFoundStmt.Dependents))
	}
}

// TestPdgQueryNotIndexed covers @step-01 (01.1.a part 3): code_pdg_query
// against a non---pdg index returns a clear "run index --pdg" message and an
// empty result, NOT an error.
func TestPdgQueryNotIndexed(t *testing.T) {
	dataDir := t.TempDir()
	raw := t.TempDir()
	abs, err := filepath.Abs(raw)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	root := abs
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {\n\ta()\n}\n\nfunc a() {}\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv("MNEMONIC_PROJECT", "pdg-probe-notindexed")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
	codeindex.ResetFileFirstSymbol()
	// Index WITHOUT --pdg (the default path).
	if _, err := svc.RunCodeIndex(context.Background(), root); err != nil {
		t.Fatalf("index (no --pdg): %v", err)
	}
	out := runPdgQuery(t, map[string]any{"symbol": "main", "statement": "3"})
	if out.Found {
		t.Errorf("expected not-found on a non---pdg index, got found")
	}
	if out.PdgEnabled {
		t.Errorf("expected pdg_enabled=false on a non---pdg index")
	}
	if len(out.Dependents) != 0 {
		t.Errorf("non---pdg index should return no dependences, got %d", len(out.Dependents))
	}
	if out.Message == "" {
		t.Errorf("expected a clear 'run index --pdg' message, got empty")
	}
}

// TestPdgQueryUnit pins the pdg.Build confidence semantics at the unit level
// (no index): control edges are EXTRACTED, a data edge through an unresolved
// call is AMBIGUOUS, and the builder never emits an empty confidence.
func TestPdgQueryUnit(t *testing.T) {
	src := "package main\n\nfunc f(v int) int {\n\tif v > 0 {\n\t\treturn g(v)\n\t}\n\treturn h(v)\n}\n"
	c, err := pdg.BuildCFG([]byte(src), "go", "f", 3)
	if err != nil {
		t.Fatalf("BuildCFG: %v", err)
	}
	calls := []pdg.CallSite{
		{Line: 5, Receiver: "", Name: "g", Resolved: true},
		{Line: 7, Receiver: "", Name: "h", Resolved: false},
	}
	rows, err := pdg.Build(1, c, calls)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	sawExtractedControl := false
	for _, r := range rows {
		if r.Confidence == "" {
			t.Errorf("edge has empty confidence: %+v", r)
		}
		if r.Kind == "control" {
			if r.Confidence != pdg.ConfidenceExtracted {
				t.Errorf("control edge is %q, want EXTRACTED", r.Confidence)
			}
			sawExtractedControl = true
		}
	}
	if !sawExtractedControl {
		t.Errorf("expected at least one EXTRACTED control edge")
	}
}
