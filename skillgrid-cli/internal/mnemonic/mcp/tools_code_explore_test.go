package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// exploreFixture indexes a small multi-file python project whose symbols have
// cross-file resolved edges, then injects the service so handlers open the
// same store.
func exploreFixture(t *testing.T) (dataDir, root string) {
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
	write("a.py", "def alpha(a, b):\n    return a + b\n")
	write("b.py", "import a\n\ndef beta():\n    return a.alpha(1, 2)\n")
	if err := os.MkdirAll(filepath.Join(root, "config.d"), 0o755); err != nil {
		t.Fatalf("mkdir config.d: %v", err)
	}
	write(filepath.Join("config.d", "indexing.yaml"),
		"mnemonic:\n  include:\n    - \"**/*.py\"\n  exclude:\n    - \"**/.git/**\"\n")
	// Pin the project BEFORE indexing so both the index and later queries land
	// in the same stable bucket (the temp dir is nested in the skillgit repo).
	t.Setenv("MNEMONIC_PROJECT", "explore-probe")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
	codeindex.ResetFileFirstSymbol()
	if _, err := svc.RunCodeIndex(context.Background(), root); err != nil {
		t.Fatalf("index: %v", err)
	}
	return dataDir, root
}

// TestExploreToolSurface covers @step-03 (code_explore is the primary tool and
// menu tools re-enable): the composite code_explore is registered and
// documented as the primary tool; the narrow graph/orientation menu tools are
// unlisted by default but re-enable via config; code_status returns a
// per-language fair-coverage field. Threat: Mnemonic tool surface.
func TestExploreToolSurface(t *testing.T) {
	dataDir, root := exploreFixture(t)
	_ = root
	// Pin the project so the temp-dir fixture (inside the skillgrid git repo)
	// resolves to one stable bucket for both the index and the query.
	t.Setenv("MNEMONIC_PROJECT", "explore-probe")

	// The composite tool is registered and its description documents it as the
	// primary tool.
	tool := codeExploreTool()
	if tool.Name != "code_explore" {
		t.Fatalf("explore tool name = %q, want code_explore", tool.Name)
	}
	if !strings.Contains(strings.ToLower(tool.Description), "primary") {
		t.Errorf("code_explore description must document it as the primary tool, got: %s", tool.Description)
	}

	// initialize guidance: answer structural questions directly, don't re-grep.
	guidance := exploreInitializeGuidance()
	low := strings.ToLower(guidance)
	if !strings.Contains(low, "primary") || !strings.Contains(low, "grep") {
		t.Errorf("initialize guidance must steer to the primary tool and say not to re-grep, got: %s", guidance)
	}

	// Default surface (real tools/list handler, filters applied): composite
	// listed, menu tools unlisted.
	s := NewServer()
	listed := listToolNames(t, s)
	if !listed["code_explore"] {
		t.Errorf("code_explore must be listed on the default MCP surface")
	}
	for _, name := range menuCodeTools {
		if listed[name] {
			t.Errorf("menu tool %s must be unlisted by default", name)
		}
	}
	// The four stable chunk tools stay listed.
	for _, name := range []string{"code_status", "code_index", "code_search", "code_read"} {
		if !listed[name] {
			t.Errorf("stable tool %s must stay listed", name)
		}
	}

	// Re-enable via config: the env var adds the menu tools back to the
	// surface (like CODEGRAPH_MCP_TOOLS).
	t.Setenv(CodeToolsEnvVar, "all")
	s2 := NewServer()
	listed2 := listToolNames(t, s2)
	for _, name := range menuCodeTools {
		if !listed2[name] {
			t.Errorf("menu tool %s must be re-enabled via %s=all", name, CodeToolsEnvVar)
		}
	}
	if !listed2["code_explore"] {
		t.Errorf("code_explore must stay listed when menu tools are enabled")
	}

	// code_status returns the per-language fair-coverage field (additive;
	// existing fields unchanged).
	oldDir, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	SetService(service.New(dataDir))
	req := mcplib.CallToolRequest{}
	res, err := handleCodeStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("handleCodeStatus: %v", err)
	}
	text := callResultText(t, res)
	if !strings.Contains(text, "fair_coverage") {
		t.Errorf("code_status must return a per-language fair_coverage field, got: %s", text)
	}
	for _, field := range []string{"file_count", "chunk_count", "stale", "unresolved_refs"} {
		if !strings.Contains(text, field) {
			t.Errorf("code_status lost an existing field %s: %s", field, text)
		}
	}
}
