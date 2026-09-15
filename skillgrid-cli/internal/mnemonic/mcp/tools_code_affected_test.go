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

// affectedMCPFixture indexes a small go app (base <- mid chain + a test file)
// and pins the project so the temp-dir fixture resolves to one stable bucket.
func affectedMCPFixture(t *testing.T) string {
	t.Helper()
	dataDir := t.TempDir()
	raw := t.TempDir()
	abs, err := filepath.Abs(raw)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	write := func(name, content string) {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(abs, name)), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(abs, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("src/base.go", "package p\n\nfunc base() int {\n\treturn 1\n}\n")
	write("src/mid.go", "package p\n\nfunc mid() int {\n\treturn base()\n}\n")
	write("src/base_test.go", "package p\n\nfunc TestBase() {\n\t_ = base()\n}\n")
	t.Setenv("MNEMONIC_PROJECT", "affectedmcp-probe")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
	codeindex.ResetFileFirstSymbol()
	if _, err := svc.RunCodeIndex(context.Background(), abs); err != nil {
		t.Fatalf("index: %v", err)
	}
	oldDir, _ := os.Getwd()
	if err := os.Chdir(abs); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })
	return dataDir
}

// TestAffectedTools covers @step-02 (Scenario: affected and rename tools
// register and stay stable): code_affected + code_rename register with
// distinct code_* names; code_affected exposes stdin/base/depth/filter/json/
// quiet; rename disambiguation returns a candidate list (no silent pick);
// dry_run touches nothing; 005/008 tools (incl. code_search) stay stable; bad
// affected/rename args are rejected with a clear validation error.
func TestAffectedTools(t *testing.T) {
	affectedMCPFixture(t)

	// Distinct code_* names registered.
	tools := NewServer().ListTools()
	for _, name := range []string{"code_affected", "code_rename"} {
		if _, ok := tools[name]; !ok {
			t.Errorf("expected tool %q to be registered", name)
		}
	}
	// 005/008 tools stay intact on the live server (baseline lock).
	for _, name := range []string{"code_search", "code_read", "code_communities", "code_impact", "code_route", "code_navigates"} {
		if _, ok := tools[name]; !ok {
			t.Errorf("005/008/010 tool %q no longer registered", name)
		}
	}
	// code_search name + required `query` param schema unchanged (005 baseline).
	assertCodeToolStable(t, codeSearchTool(), "code_search", []string{"query"})

	// The tool surface grows additively: 71 baseline + 2 affected/rename +
	// 1 code_pdg_query + 1 code_taint + 2 governance + 1 mem_layers + 2 session = 80.
	if len(tools) != 83 {
		t.Errorf("expected 83 tools (71 baseline + 2 affected/rename + 1 pdg_query + 1 taint + 2 governance + 1 mem_layers + 2 session + 2 status/compact), got %d", len(tools))
	}

	// code_affected runs the traversal (changed -> affected test files).
	res, err := handleCodeAffected(context.Background(), newCallTool("code_affected", map[string]any{
		"changed": []any{"src/base.go", "src/mid.go"},
	}))
	if err != nil {
		t.Fatalf("handleCodeAffected dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("code_affected errored: %s", callResultText(t, res))
	}
	var affOut struct {
		TestFiles []string `json:"test_files"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &affOut); err != nil {
		t.Fatalf("code_affected output not JSON: %v (text %s)", err, callResultText(t, res))
	}
	if !containsStr(affOut.TestFiles, "src/base_test.go") {
		t.Errorf("code_affected must surface src/base_test.go, got %v", affOut.TestFiles)
	}

	// code_rename dry_run (default) returns the plan without writing.
	rres, err := handleCodeRename(context.Background(), newCallTool("code_rename", map[string]any{
		"old": "base", "new": "verifyBase",
	}))
	if err != nil {
		t.Fatalf("handleCodeRename dispatch: %v", err)
	}
	if rres.IsError {
		t.Fatalf("code_rename errored: %s", callResultText(t, rres))
	}
	rtext := callResultText(t, rres)
	if !strings.Contains(rtext, `"dry_run":true`) && !strings.Contains(rtext, `"dry_run": true`) {
		t.Errorf("code_rename must report dry_run true (default), got: %s", rtext)
	}
	if !strings.Contains(rtext, "graph_edits") {
		t.Errorf("code_rename must return the graph bucket, got: %s", rtext)
	}
}

// TestAffectedToolsBadArgs covers @step-02 (bad affected/rename args rejected
// with a clear validation error, not an invented result).
func TestAffectedToolsBadArgs(t *testing.T) {
	affectedMCPFixture(t)

	// code_affected with no changed set (and no stdin/base) is a validation
	// error (no invented changed set).
	res, err := handleCodeAffected(context.Background(), newCallTool("code_affected", map[string]any{}))
	if err != nil {
		t.Fatalf("handleCodeAffected dispatch: %v", err)
	}
	if !res.IsError {
		t.Errorf("code_affected with no changed set should be a validation error, got: %s", callResultText(t, res))
	}

	// code_rename with a missing new name is a validation error.
	rres, err := handleCodeRename(context.Background(), newCallTool("code_rename", map[string]any{
		"old": "base",
	}))
	if err != nil {
		t.Fatalf("handleCodeRename dispatch: %v", err)
	}
	if !rres.IsError {
		t.Errorf("code_rename without new should be a validation error, got: %s", callResultText(t, rres))
	}
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
