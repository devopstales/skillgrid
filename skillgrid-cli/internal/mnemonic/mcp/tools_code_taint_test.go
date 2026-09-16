package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// taintMCPFixture indexes a Go repo with a known source->sink flow (a request
// param -> SQL exec) through the shared service with --pdg, and returns the
// dataDir + root for handler calls.
func taintMCPFixture(t *testing.T, pdg bool) (dataDir, root string) {
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
	// A request param (GetRequestParam) flows to a SQL exec (sqlExec).
	write("main.go", "package main\n\nfunc handler(r *Req) {\n\tp := GetRequestParam(r, \"q\")\n\tu := Transform(p)\n\tsqlExec(u)\n}\n\nfunc GetRequestParam(r *Req, k string) string { return r.Path }\n\nfunc Transform(s string) string { return s }\n\nfunc sqlExec(s string) {}\n")
	t.Setenv("MNEMONIC_PROJECT", "taint-probe")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
	codeindex.ResetFileFirstSymbol()
	if pdg {
		if _, err := svc.RunCodeIndexPDG(context.Background(), root, true, false); err != nil {
			t.Fatalf("index --pdg: %v", err)
		}
	} else {
		if _, err := svc.RunCodeIndex(context.Background(), root); err != nil {
			t.Fatalf("index: %v", err)
		}
	}
	return dataDir, root
}

// taintOut mirrors the JSON the code_taint handler returns.
type taintOut struct {
	Findings   []struct {
		Symbol   string `json:"symbol"`
		Source   int    `json:"source_line"`
		SourceNm string `json:"source_name"`
		Sink     int    `json:"sink_line"`
		SinkNm   string `json:"sink_name"`
		Label    string `json:"confidence"`
		StopsAt  string `json:"stops_at"`
	} `json:"findings"`
	PdgEnabled bool   `json:"pdg_enabled"`
	Count      int    `json:"count"`
	Message    string `json:"message"`
}

func runTaint(t *testing.T, params map[string]any) *taintOut {
	t.Helper()
	res, err := handleCodeTaint(context.Background(), newCallTool("code_taint", params))
	if err != nil {
		t.Fatalf("handleCodeTaint: %v", err)
	}
	var out taintOut
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("parse code_taint result %q: %v", callResultText(t, res), err)
	}
	return &out
}

// TestTaintTool covers @step-02 (02.7, Scenario: code_taint registered and bad
// args fail): (1) code_taint is registered with a distinct name + optional
// --symbol/--file/--json params; (2) the full set of 005/008/010 code_* tool
// names + required params is unchanged; (3) code_taint on a --pdg index returns
// the known finding; (4) a non---pdg index returns a clear "run --pdg" message
// (not an error); (5) bad/missing args are handled (not fabricated findings).
func TestTaintTool(t *testing.T) {
	_, _ = taintMCPFixture(t, true)

	// (1) Registration + distinct name + optional params.
	tool := codeTaintTool()
	assertCodeToolStable(t, tool, "code_taint", nil) // all params optional
	tools := NewServer().ListTools()
	if _, ok := tools["code_taint"]; !ok {
		t.Errorf("code_taint not registered")
	}
	// (2) 005/008/010 tools stay intact (baseline lock).
	for _, name := range []string{"code_search", "code_read", "code_communities", "code_impact", "code_route", "code_affected", "code_pdg_query"} {
		if _, ok := tools[name]; !ok {
			t.Errorf("005/008/010 tool %q no longer registered", name)
		}
	}
	assertCodeToolStable(t, codeSearchTool(), "code_search", []string{"query"})
	assertCodeToolStable(t, codePdgQueryTool(), "code_pdg_query", []string{"symbol", "statement"})

	// (3) A --pdg index returns the known source->sink finding.
	out := runTaint(t, map[string]any{})
	if !out.PdgEnabled {
		t.Fatalf("expected pdg_enabled=true on a --pdg index")
	}
	if out.Count == 0 {
		t.Fatalf("expected taint findings on a --pdg index, got 0: %s", out.Message)
	}
	var found bool
	for _, f := range out.Findings {
		if strings.Contains(f.SourceNm, "GetRequestParam") && strings.Contains(f.SinkNm, "sqlExec") {
			found = true
			if f.Label == "" {
				t.Errorf("finding carries an empty confidence label")
			}
		}
	}
	if !found {
		t.Errorf("expected a GetRequestParam -> sqlExec finding, got: %+v", out.Findings)
	}

	// (5) A --symbol filter narrows the findings (a known symbol returns its
	// findings; an unknown symbol is not-found, no fabricated findings).
	outSym := runTaint(t, map[string]any{"symbol": "handler"})
	if outSym.Count == 0 {
		t.Errorf("expected the handler symbol to have taint findings, got 0: %s", outSym.Message)
	}
	outUnknown := runTaint(t, map[string]any{"symbol": "does_not_exist"})
	if outUnknown.Count != 0 {
		t.Errorf("an unknown symbol should have no fabricated findings, got %d", outUnknown.Count)
	}

	// (02.8) A --file filter narrows to findings in that file; an unknown file
	// is not-found. The fixture's finding is in main.go.
	outFile := runTaint(t, map[string]any{"file": "main.go"})
	if outFile.Count == 0 {
		t.Errorf("expected main.go to have taint findings, got 0: %s", outFile.Message)
	}
	outFileUnknown := runTaint(t, map[string]any{"file": "no_such_file.go"})
	if outFileUnknown.Count != 0 {
		t.Errorf("an unknown file should have no fabricated findings, got %d", outFileUnknown.Count)
	}

	// (02.8) --json returns the machine-readable findings array (CI).
	res, err := handleCodeTaint(context.Background(), newCallTool("code_taint", map[string]any{"json": true}))
	if err != nil {
		t.Fatalf("handleCodeTaint (--json): %v", err)
	}
	text := callResultText(t, res)
	var jsonOut struct {
		Findings []json.RawMessage `json:"findings"`
		Count    int               `json:"count"`
	}
	if err := json.Unmarshal([]byte(text), &jsonOut); err != nil {
		t.Fatalf("parse --json result %q: %v", text, err)
	}
	if jsonOut.Count == 0 {
		t.Errorf("--json: expected findings, got count 0: %s", text)
	}
}

// TestTaintToolNotIndexed covers @step-02 (02.9, Scenario: Non-pdg taint query
// returns run-pdg hint): code_taint against a non---pdg index returns a clear
// "run index --pdg" message and an empty result, NOT an error.
func TestTaintToolNotIndexed(t *testing.T) {
	_, _ = taintMCPFixture(t, false)
	res, err := handleCodeTaint(context.Background(), newCallTool("code_taint", map[string]any{}))
	if err != nil {
		t.Fatalf("handleCodeTaint returned a Go error on a non---pdg index (want a message, not an error): %v", err)
	}
	out := runTaint(t, map[string]any{})
	if out.PdgEnabled {
		t.Errorf("expected pdg_enabled=false on a non---pdg index")
	}
	if out.Count != 0 {
		t.Errorf("a non---pdg index should return no findings, got %d", out.Count)
	}
	if !strings.Contains(strings.ToLower(out.Message), "index --pdg") {
		t.Errorf("expected a clear 'run index --pdg' message, got %q", out.Message)
	}
	// The result text is not an error (IsError false).
	if res.IsError {
		t.Errorf("a non---pdg index should not be an error, got IsError=true: %s", callResultText(t, res))
	}
}
