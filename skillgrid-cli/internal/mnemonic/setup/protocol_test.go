package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProtocolMarkdownFromRepoPrefersShared(t *testing.T) {
	repo := t.TempDir()
	shared := filepath.Join(repo, "plugins", "_shared", "memory-protocol.md")
	if err := os.MkdirAll(filepath.Dir(shared), 0o755); err != nil {
		t.Fatal(err)
	}
	const body = "# Mnemonic Memory Protocol\n\nfrom-shared\n"
	if err := os.WriteFile(shared, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	got := ProtocolMarkdownFromRepo(repo)
	if !strings.Contains(got, "from-shared") {
		t.Fatalf("ProtocolMarkdownFromRepo = %q, want plugins/_shared/memory-protocol.md", got)
	}
}

func TestCursorPluginLayout(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", ".."))
	required := []string{
		".cursor-plugin/plugin.json",
		".cursor-plugin/marketplace.json",
		"hooks/hooks-cursor.json",
		"hooks/cursor-session-start.sh",
		"hooks/cursor-session-end.sh",
		"hooks/cursor-tool-capture.sh",
		"hooks/cursor-policy.sh",
		"hooks/tool-call-capture.js",
		"rules/mnemonic.mdc",
		"scripts/sync-mnemonic-rule.sh",
		"plugins/_shared/memory-protocol.md",
		"plugins/cursor/mcp.json",
		"agents/cursor/implementer.md",
		"agents/cursor/reviewer.md",
		"agents/cursor/researcher.md",
		"agents/cursor/verifier.md",
		"agents/cursor/debugger.md",
	}
	for _, rel := range required {
		path := filepath.Join(repoRoot, rel)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
	// The Cursor plugin manifest stays at the repo root. plugins/cursor holds
	// only the mnemonic MCP config.
	if _, err := os.Stat(filepath.Join(repoRoot, "plugins", "cursor", ".cursor-plugin")); err == nil {
		t.Error("plugins/cursor/.cursor-plugin must not exist (Cursor plugin lives at repo root)")
	}

	pluginRaw, err := os.ReadFile(filepath.Join(repoRoot, ".cursor-plugin", "plugin.json"))
	if err != nil {
		t.Fatalf("read plugin.json: %v", err)
	}
	var pluginDoc struct {
		Agents     string `json:"agents"`
		Skills     string `json:"skills"`
		MCPServers string `json:"mcpServers"`
	}
	if err := json.Unmarshal(pluginRaw, &pluginDoc); err != nil {
		t.Fatalf("parse plugin.json: %v", err)
	}
	if pluginDoc.Agents != "./agents/cursor/" {
		t.Errorf("plugin.json agents = %q, want ./agents/cursor/", pluginDoc.Agents)
	}
	if pluginDoc.Skills != "./.agents/skills/**/SKILL.md" {
		t.Errorf("plugin.json skills = %q, want ./.agents/skills/**/SKILL.md", pluginDoc.Skills)
	}
	if pluginDoc.MCPServers != "./plugins/cursor/mcp.json" {
		t.Errorf("plugin.json mcpServers = %q, want ./plugins/cursor/mcp.json", pluginDoc.MCPServers)
	}
	var skillCount int
	skillsRoot := filepath.Join(repoRoot, ".agents", "skills")
	walkErr := filepath.WalkDir(skillsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == "SKILL.md" {
			skillCount++
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk skills: %v", walkErr)
	}
	if skillCount == 0 {
		t.Error("plugin skills glob matches no SKILL.md files")
	}

	mcpRaw, err := os.ReadFile(filepath.Join(repoRoot, "plugins", "cursor", "mcp.json"))
	if err != nil {
		t.Fatalf("read plugins/cursor/mcp.json: %v", err)
	}
	var mcpDoc struct {
		MCPServers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(mcpRaw, &mcpDoc); err != nil {
		t.Fatalf("parse mcp.json: %v", err)
	}
	mnemonic, ok := mcpDoc.MCPServers["mnemonic"]
	if !ok {
		t.Fatal("mcp.json missing mcpServers.mnemonic")
	}
	if mnemonic.Command != "skillgrid" {
		t.Errorf("mnemonic command = %q, want skillgrid", mnemonic.Command)
	}
	if len(mnemonic.Args) != 1 || mnemonic.Args[0] != "mcp" {
		t.Errorf("mnemonic args = %v, want [mcp]", mnemonic.Args)
	}
}
