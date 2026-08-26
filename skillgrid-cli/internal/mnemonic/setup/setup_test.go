package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	jsonc "github.com/tidwall/gjson"
)

func testHomeAndRepo(t *testing.T) (home, repoRoot string) {
	t.Helper()
	home = t.TempDir()
	repoRoot = testRepoRoot(t)

	// Seed minimal opencode config for bridge tests.
	opencodeDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(filepath.Join(opencodeDir, "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(opencodeDir, "shared"), 0o755); err != nil {
		t.Fatal(err)
	}
	pluginSrc := filepath.Join(repoRoot, pluginRelPath)
	pluginData, err := os.ReadFile(pluginSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(opencodeDir, "plugins", "mnemonic.ts"), pluginData, 0o644); err != nil {
		t.Fatal(err)
	}
	httpSrc := filepath.Join(repoRoot, httpClientRelPath)
	httpData, err := os.ReadFile(httpSrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(opencodeDir, "shared", "http-client.ts"), httpData, 0o644); err != nil {
		t.Fatal(err)
	}
	return home, repoRoot
}

func TestSetupOpenCodeIdempotent(t *testing.T) {
	home, repoRoot := testHomeAndRepo(t)

	if err := SetupOpenCode(home, repoRoot, false); err != nil {
		t.Fatalf("first setup: %v", err)
	}
	if err := SetupOpenCode(home, repoRoot, false); err != nil {
		t.Fatalf("second setup: %v", err)
	}

	cfgPath := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	mcpPath := "mcp." + mcpServerName
	if !jsonc.Get(content, mcpPath).Exists() {
		t.Fatalf("missing %s in config", mcpPath)
	}
	if strings.Count(content, `"skillgrid-mnemonic"`) != 1 {
		t.Errorf("expected one skillgrid-mnemonic entry, got %d", strings.Count(content, `"skillgrid-mnemonic"`))
	}

	pluginPath := tildePath(home, filepath.Join(home, ".config", "opencode", "plugins", "mnemonic.ts"))
	plugins := jsonc.Get(content, "plugin").Array()
	count := 0
	for _, p := range plugins {
		if p.String() == pluginPath {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected plugin path once, found %d (path=%q)", count, pluginPath)
	}

	if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "plugins", "mnemonic.ts")); err != nil {
		t.Errorf("plugin file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "shared", "http-client.ts")); err != nil {
		t.Errorf("http-client missing: %v", err)
	}
}

func TestSetupKiloCodeIdempotent(t *testing.T) {
	home, repoRoot := testHomeAndRepo(t)

	if err := SetupKiloCode(home, repoRoot, false); err != nil {
		t.Fatalf("first setup: %v", err)
	}
	if err := SetupKiloCode(home, repoRoot, false); err != nil {
		t.Fatalf("second setup: %v", err)
	}

	cfgPath := filepath.Join(home, ".config", "kilo", "kilo.jsonc")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if strings.Count(content, `"skillgrid-mnemonic"`) != 1 {
		t.Errorf("expected one mcp entry, got %d", strings.Count(content, `"skillgrid-mnemonic"`))
	}

	agentsPath := filepath.Join(home, ".config", "kilo", "AGENTS.md")
	agents, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	agentText := string(agents)
	if strings.Count(agentText, kiloBeginMarker) != 1 {
		t.Errorf("expected one begin marker, got %d", strings.Count(agentText, kiloBeginMarker))
	}
	if strings.Count(agentText, kiloEndMarker) != 1 {
		t.Errorf("expected one end marker, got %d", strings.Count(agentText, kiloEndMarker))
	}
	if !strings.Contains(agentText, "mem_save") {
		t.Error("AGENTS.md missing protocol content")
	}

	if _, err := os.Stat(filepath.Join(home, ".config", "kilo", "plugins", "mnemonic.ts")); err != nil {
		t.Errorf("bridged plugin missing: %v", err)
	}
}

func TestSetupCursorIdempotent(t *testing.T) {
	home, _ := testHomeAndRepo(t)
	repoRoot := testRepoRoot(t)

	if err := SetupCursor(home, false); err != nil {
		t.Fatalf("first setup: %v", err)
	}
	if err := SetupCursor(home, false); err != nil {
		t.Fatalf("second setup: %v", err)
	}

	mcpPath := filepath.Join(home, ".cursor", "mcp.json")
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if strings.Count(content, `"skillgrid-mnemonic"`) != 1 {
		t.Errorf("expected one mcpServers entry, got %d", strings.Count(content, `"skillgrid-mnemonic"`))
	}

	rulePath := filepath.Join(home, ".cursor", "rules", "mnemonic.mdc")
	rule, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatal(err)
	}
	ruleText := string(rule)
	if strings.Contains(ruleText, "{{MEMORY_PROTOCOL}}") {
		t.Error("rule still contains template placeholder")
	}
	if !strings.Contains(ruleText, "mem_save") {
		t.Error("rule missing protocol content")
	}
	if !strings.Contains(ruleText, "alwaysApply: true") {
		t.Error("rule missing frontmatter")
	}

	_ = repoRoot
}

func TestRunSetupUnknownAgent(t *testing.T) {
	if err := RunSetup("unknown", "", false); err == nil {
		t.Fatal("expected error for unknown agent")
	}
}
