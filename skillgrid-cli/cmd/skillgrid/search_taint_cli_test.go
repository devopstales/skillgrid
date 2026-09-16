package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// taintCLIFixture writes a go project with a known source->sink flow, indexes
// it with the --pdg pass through the shared service, and returns the service +
// root (so the test can drive runSearchTaint with a stable project).
func taintCLIFixture(t *testing.T, pdg bool) (*service.Service, string) {
	t.Helper()
	dataDir := t.TempDir()
	svc := service.New(dataDir)
	root := t.TempDir()
	write := func(name, content string) {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("src/main.go", "package p\n\nfunc handler(r *Req) {\n\tp := GetRequestParam(r, \"q\")\n\tu := Transform(p)\n\tsqlExec(u)\n}\n\nfunc GetRequestParam(r *Req, k string) string { return r.Path }\n\nfunc Transform(s string) string { return s }\n\nfunc sqlExec(s string) {}\n")
	if _, err := svc.RunCodeIndexPDG(context.Background(), root, pdg, false); err != nil {
		t.Fatalf("index --pdg=%v: %v", pdg, err)
	}
	return svc, root
}

// captureSearchTaint runs runSearchTaint with the service injected, capturing
// stdout+stderr to a temp file.
func captureSearchTaint(t *testing.T, svc *service.Service, root string, args []string) string {
	t.Helper()
	oldSvc := cliService
	cliService = svc
	defer func() { cliService = oldSvc }()
	f, err := os.CreateTemp(t.TempDir(), "taint-out-*")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	oldStdout, oldStderr := os.Stdout, os.Stderr
	os.Stdout = f
	os.Stderr = f
	runSearchTaint("test", args)
	os.Stdout, os.Stderr = oldStdout, oldStderr
	_ = f.Close()
	b, _ := os.ReadFile(f.Name())
	return string(b)
}

// TestSearchTaintCLIParity covers @step-02 (02.11, Scenario: CLI taint search
// and pdg flag parity): `skillgrid search taint` returns the source->sink taint
// findings from a --pdg index (parity with the `index --pdg` pass the fixture
// used), with --symbol/--file filters and --json. An unknown symbol is
// not-found (no fabricated findings).
func TestSearchTaintCLIParity(t *testing.T) {
	t.Setenv("MNEMONIC_PROJECT", "cli-taint")
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", t.TempDir())
	svc, root := taintCLIFixture(t, true)
	openSearchTaintService = func() (*service.Service, string, error) {
		return svc, "cli-taint", nil
	}
	t.Cleanup(func() { openSearchTaintService = openGraphService })

	// A --pdg index returns the known source->sink finding.
	out := captureSearchTaint(t, svc, root, []string{"search", "taint"})
	if strings.Contains(out, "not found") {
		t.Fatalf("expected taint findings on a --pdg index, got not-found: %s", out)
	}
	if !strings.Contains(out, "finding") {
		t.Fatalf("expected a findings count, got: %s", out)
	}
	// JSON form returns the findings array.
	outJSON := captureSearchTaint(t, svc, root, []string{"search", "taint", "--json"})
	if !strings.Contains(outJSON, "\"findings\"") {
		t.Errorf("expected a findings array in JSON, got: %s", outJSON)
	}
	if !strings.Contains(outJSON, "GetRequestParam") {
		t.Errorf("expected the GetRequestParam source in JSON, got: %s", outJSON)
	}
	// A --symbol filter narrows the findings.
	outSym := captureSearchTaint(t, svc, root, []string{"search", "taint", "--symbol", "handler"})
	if strings.Contains(outSym, "not found") {
		t.Errorf("expected the handler symbol to have taint findings, got: %s", outSym)
	}
	// An unknown symbol is not-found.
	outUnknown := captureSearchTaint(t, svc, root, []string{"search", "taint", "--symbol", "does_not_exist"})
	if !strings.Contains(outUnknown, "not found") {
		t.Errorf("expected not-found for an unknown symbol, got: %s", outUnknown)
	}
}

// TestSearchTaintCLINotIndexed covers @step-02 (02.11): a non---pdg index makes
// `skillgrid search taint` print a clear "run index --pdg" message, not an
// error and not a fabricated result.
func TestSearchTaintCLINotIndexed(t *testing.T) {
	t.Setenv("MNEMONIC_PROJECT", "cli-taint-notindexed")
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", t.TempDir())
	svc, root := taintCLIFixture(t, false)
	openSearchTaintService = func() (*service.Service, string, error) {
		return svc, "cli-taint-notindexed", nil
	}
	t.Cleanup(func() { openSearchTaintService = openGraphService })

	out := captureSearchTaint(t, svc, root, []string{"search", "taint"})
	if !strings.Contains(out, "index --pdg") {
		t.Errorf("expected a 'run index --pdg' message on a non---pdg index, got: %s", out)
	}
	// A non---pdg index should not report a findings count (the run-pdg hint
	// is not a findings result).
	if strings.Contains(out, "taint finding(s)") {
		t.Errorf("a non---pdg index should not report a findings count, got: %s", out)
	}
}
