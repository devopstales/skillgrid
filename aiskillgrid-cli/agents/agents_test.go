package agents

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aiskillgrid/aiskillgrid/mcpmerge"
)

func setupPack(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	skill := filepath.Join(root, "packs", "skills", "aiskillgrid-stub")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("# stub\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rules := filepath.Join(root, "packs", "rules")
	if err := os.MkdirAll(rules, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rules, "no-ai-commit-coauthors.mdc"), []byte("---\nalwaysApply: true\n---\n# no ai\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mcpDir := filepath.Join(root, "packs", "mcp")
	if err := os.MkdirAll(mcpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mcpDir, "servers.json"), []byte(`{"mcpServers":{"aiskillgrid-demo":{"command":"true"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCursorProjectInstall(t *testing.T) {
	pack := setupPack(t)
	project := t.TempDir()
	userHome := t.TempDir()
	cfg := t.TempDir()
	ctx := Context{
		Scope:      ScopeProject,
		ProjectDir: project,
		HomeRoot:   userHome,
		ConfigDir:  cfg,
		PackRoot:   pack,
	}
	res, err := Cursor{}.Install(ctx)
	if err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(project, ".cursor", "skills", "aiskillgrid-stub", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatal(err)
	}
	rulePath := filepath.Join(project, ".cursor", "rules", "no-ai-commit-coauthors.mdc")
	if _, err := os.Stat(rulePath); err != nil {
		t.Fatal(err)
	}
	mcpPath := filepath.Join(project, ".cursor", "mcp.json")
	obj, err := mcpmerge.LoadOrEmpty(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	servers := obj["mcpServers"].(map[string]any)
	if _, ok := servers["aiskillgrid-demo"]; !ok {
		t.Fatalf("missing mcp entry: %#v written=%v", obj, res.Written)
	}
}

func TestKiloGlobalInstall(t *testing.T) {
	pack := setupPack(t)
	userHome := t.TempDir()
	cfg := t.TempDir()
	ctx := Context{
		Scope:     ScopeGlobal,
		HomeRoot:  userHome,
		ConfigDir: cfg,
		PackRoot:  pack,
	}
	if _, err := (Kilo{}).Install(ctx); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(userHome, ".kilo", "skills", "aiskillgrid-stub", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatal(err)
	}
	rulePath := filepath.Join(userHome, ".kilo", "rules", "no-ai-commit-coauthors.mdc")
	if _, err := os.Stat(rulePath); err != nil {
		t.Fatal(err)
	}
	mcpPath := filepath.Join(cfg, "kilo", "kilo.jsonc")
	obj, err := mcpmerge.LoadOrEmpty(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := obj["mcp"].(map[string]any)["aiskillgrid-demo"]; !ok {
		t.Fatalf("missing kilo mcp: %#v", obj)
	}
}

func TestAllAgentsInstall(t *testing.T) {
	pack := setupPack(t)
	project := t.TempDir()
	userHome := t.TempDir()
	cfg := t.TempDir()
	ctx := Context{
		Scope:      ScopeProject,
		ProjectDir: project,
		HomeRoot:   userHome,
		ConfigDir:  cfg,
		PackRoot:   pack,
	}
	for _, a := range All() {
		if _, err := a.Install(ctx); err != nil {
			t.Fatalf("%s: %v", a.Name(), err)
		}
	}
	// Copilot gets .instructions.md; others keep .mdc
	paths := []string{
		filepath.Join(project, ".cursor", "rules", "no-ai-commit-coauthors.mdc"),
		filepath.Join(project, ".kilo", "rules", "no-ai-commit-coauthors.mdc"),
		filepath.Join(project, ".opencode", "rules", "no-ai-commit-coauthors.mdc"),
		filepath.Join(project, ".github", "instructions", "no-ai-commit-coauthors.instructions.md"),
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("missing rule at %s: %v", p, err)
		}
	}
}

func TestResolvedMCPInjection(t *testing.T) {
	// Test that when ResolvedMCP is set, mergeMCPFile uses injected servers instead of pack file
	project := t.TempDir()
	userHome := t.TempDir()
	cfg := t.TempDir()
	ctx := Context{
		Scope:      ScopeProject,
		ProjectDir: project,
		HomeRoot:   userHome,
		ConfigDir:  cfg,
		PackRoot:   "", // no pack needed
		ResolvedMCP: map[string]any{
			"aiskillgrid-engram": map[string]any{
				"command": "/tmp/engram",
				"args":    []any{"mcp"},
			},
		},
	}
	res, err := Cursor{}.Install(ctx)
	if err != nil {
		t.Fatal(err)
	}
	mcpPath := filepath.Join(project, ".cursor", "mcp.json")
	obj, err := mcpmerge.LoadOrEmpty(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	servers := obj["mcpServers"].(map[string]any)
	engram, ok := servers["aiskillgrid-engram"]
	if !ok {
		t.Fatalf("missing injected mcp entry: %#v written=%v", obj, res.Written)
	}
	engramMap := engram.(map[string]any)
	if engramMap["command"] != "/tmp/engram" {
		t.Fatalf("wrong command: got %v", engramMap["command"])
	}
}
