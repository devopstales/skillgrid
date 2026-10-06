package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// pdgCLIFixture writes a go project with branch/loop/return structure, indexes
// it with the --pdg pass through the shared service, and returns the service +
// root (so the test can drive runSearchPdg with a stable project).
func pdgCLIFixture(t *testing.T, pdg bool) (*service.Service, string) {
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
	write("src/main.go", "package p\n\nfunc main() {\n\tn := add(1, 2)\n\tif n > 3 {\n\t\twarn(n)\n\t} else {\n\t\tok()\n\t}\n\tfor i := 0; i < n; i++ {\n\t\tlog(i)\n\t}\n}\n\nfunc add(a, b int) int { return a + b }\n\nfunc warn(n int) { _ = n * 2 }\n\nfunc ok() {}\n\nfunc log(i int) {}\n")
	if _, err := svc.RunCodeIndexPDG(context.Background(), root, pdg, false); err != nil {
		t.Fatalf("index --pdg=%v: %v", pdg, err)
	}
	return svc, root
}

// captureSearchPdg runs runSearchPdg with the service injected (via
// cliService + openSearchPdgService), capturing stdout+stderr to a temp file
// (simpler and more reliable than a pipe, which raced with the goroutine).
func captureSearchPdg(t *testing.T, svc *service.Service, root string, args []string) string {
	t.Helper()
	oldSvc := cliService
	cliService = svc
	defer func() { cliService = oldSvc }()
	f, err := os.CreateTemp(t.TempDir(), "pdg-out-*")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	oldStdout, oldStderr := os.Stdout, os.Stderr
	os.Stdout = f
	os.Stderr = f
	runSearchPdg("test", args)
	os.Stdout, os.Stderr = oldStdout, oldStderr
	_ = f.Close()
	b, _ := os.ReadFile(f.Name())
	return string(b)
}

// TestSearchPdgCLIParity covers @step-01 (01.15): `skillgrid search pdg`
// returns the statement's control/data dependents from a --pdg index (parity
// with the `index --pdg` pass the fixture used). An unknown symbol is
// not-found (no fabricated dependences).
func TestSearchPdgCLIParity(t *testing.T) {
	t.Setenv("MNEMONIC_PROJECT", "cli-pdg")
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", t.TempDir())
	svc, root := pdgCLIFixture(t, true)
	openSearchPdgService = func() (*service.Service, string, error) {
		return svc, "cli-pdg", nil
	}
	t.Cleanup(func() { openSearchPdgService = openGraphService })

	// A branch header (line 5 of src/main.go, the `if n > 3` statement) has
	// control dependents.
	out := captureSearchPdg(t, svc, root, []string{"search", "pdg", "main", "5"})
	if strings.Contains(out, "not found") {
		t.Fatalf("expected main@5 (branch) to be found, got not-found: %s", out)
	}
	if !strings.Contains(out, "dependence") {
		t.Fatalf("expected a dependences count, got: %s", out)
	}
	// JSON form returns the dependents array.
	outJSON := captureSearchPdg(t, svc, root, []string{"search", "pdg", "main", "5", "--json"})
	if !strings.Contains(outJSON, "\"found\": true") {
		t.Errorf("expected found=true in JSON, got: %s", outJSON)
	}
	if !strings.Contains(outJSON, "\"dependents\"") {
		t.Errorf("expected a dependents array in JSON, got: %s", outJSON)
	}

	// (01.14) Unknown symbol -> not-found, no fabricated dependences.
	outUnknown := captureSearchPdg(t, svc, root, []string{"search", "pdg", "does_not_exist", "5"})
	if !strings.Contains(outUnknown, "not found") {
		t.Errorf("expected not-found for an unknown symbol, got: %s", outUnknown)
	}
}

// TestSearchPdgCLINotIndexed covers @step-01 (01.15): a non---pdg index makes
// `skillgrid search pdg` print a clear "run index --pdg" message, not an error
// and not a fabricated result.
func TestSearchPdgCLINotIndexed(t *testing.T) {
	t.Setenv("MNEMONIC_PROJECT", "cli-pdg-notindexed")
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", t.TempDir())
	svc, root := pdgCLIFixture(t, false)
	openSearchPdgService = func() (*service.Service, string, error) {
		return svc, "cli-pdg-notindexed", nil
	}
	t.Cleanup(func() { openSearchPdgService = openGraphService })

	out := captureSearchPdg(t, svc, root, []string{"search", "pdg", "main", "5"})
	if !strings.Contains(out, "index --pdg") {
		t.Errorf("expected a 'run index --pdg' message on a non---pdg index, got: %s", out)
	}
	if strings.Contains(out, "dependence") {
		t.Errorf("a non---pdg index should not report dependences, got: %s", out)
	}
}
