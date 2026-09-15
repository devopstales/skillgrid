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

// unresolvedMCPFixture indexes a web app with one unserved handler (server.js)
// so the index records a dropped ref, and pins the project to a stable bucket.
func unresolvedMCPFixture(t *testing.T) string {
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
	write("server.js", "app.get('/mystery', unservedHandler);\n")
	write("config.d/indexing.yaml", "mnemonic:\n  include:\n    - '**/*.js'\n  exclude:\n    - '**/.git/**'\n")
	t.Setenv("MNEMONIC_PROJECT", "unresolvedmcp-probe")
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

// TestCodeUnresolvedRefsQuery covers 034 (Scenario: unresolved drop log is
// queryable): the tool returns the dropped refs with a total count header,
// filters by name substring, and reports "no unresolved refs" when empty.
func TestCodeUnresolvedRefsQuery(t *testing.T) {
	dataDir := unresolvedMCPFixture(t)
	_ = dataDir

	// The fixture's unserved handler was dropped at extraction.
	res, err := handleCodeUnresolvedRefs(context.Background(), newCallTool("code_unresolved_refs", nil))
	if err != nil {
		t.Fatalf("handleCodeUnresolvedRefs: %v", err)
	}
	if res.IsError {
		t.Fatalf("code_unresolved_refs errored: %s", callResultText(t, res))
	}
	var out struct {
		Total int `json:"total"`
		Refs  []struct {
			ReferenceName string `json:"reference_name"`
			ReferenceKind string `json:"reference_kind"`
			Line          int    `json:"line"`
			Candidates    int    `json:"candidates"`
			Status        string `json:"status"`
			Path          string `json:"path"`
		} `json:"refs"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("output not JSON: %v (text %s)", err, callResultText(t, res))
	}
	if out.Total < 1 || len(out.Refs) < 1 {
		t.Fatalf("expected the fixture's unserved handler to be logged, got total=%d refs=%d", out.Total, len(out.Refs))
	}
	found := false
	for _, r := range out.Refs {
		if r.ReferenceName == "unservedHandler" {
			found = true
			if r.ReferenceKind != "route_handler" {
				t.Errorf("kind = %q, want route_handler", r.ReferenceKind)
			}
			if r.Status != "failed" {
				t.Errorf("status = %q, want failed", r.Status)
			}
			if !strings.Contains(r.Path, "server.js") {
				t.Errorf("path = %q, want server.js", r.Path)
			}
		}
	}
	if !found {
		t.Errorf("expected unservedHandler in the unresolved refs, got %+v", out.Refs)
	}

	// Name substring filter: a non-matching name returns zero rows.
	res2, err := handleCodeUnresolvedRefs(context.Background(), newCallTool("code_unresolved_refs", map[string]any{"name": "zzz-no-match"}))
	if err != nil {
		t.Fatalf("filter query: %v", err)
	}
	if res2.IsError {
		t.Fatalf("filter query errored: %s", callResultText(t, res2))
	}
	var out2 struct {
		Total   int    `json:"total"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res2)), &out2); err != nil {
		t.Fatalf("filter output not JSON: %v (text %s)", err, callResultText(t, res2))
	}
	if out2.Total != 0 {
		t.Errorf("filtered total = %d, want 0", out2.Total)
	}
	if out2.Message != "no unresolved refs" {
		t.Errorf("empty filter result should report 'no unresolved refs', got %q", out2.Message)
	}
}

// TestCodeUnresolvedRefsToolRegistered covers 034 (tool registration): the
// query tool registers with a distinct name and is part of the narrow code_*
// menu (unlisted by default, re-enabled via CodeToolsEnvVar).
func TestCodeUnresolvedRefsToolRegistered(t *testing.T) {
	tools := NewServer().ListTools()
	if _, ok := tools["code_unresolved_refs"]; !ok {
		t.Fatalf("expected code_unresolved_refs to be registered")
	}

	// Hidden by default on the MCP surface.
	listed := listToolNames(t, NewServer())
	if listed["code_unresolved_refs"] {
		t.Errorf("code_unresolved_refs must be unlisted by default (narrow menu)")
	}
	// Re-enabled via the env var.
	t.Setenv(CodeToolsEnvVar, "all")
	listed2 := listToolNames(t, NewServer())
	if !listed2["code_unresolved_refs"] {
		t.Errorf("code_unresolved_refs must be re-enabled via %s=all", CodeToolsEnvVar)
	}
}
