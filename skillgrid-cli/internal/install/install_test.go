package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAvailableAgents(t *testing.T) {
	got := AvailableAgents()
	if len(got) != 3 {
		t.Fatalf("want 3 agents, got %d", len(got))
	}
	want := map[string]struct{ npm, bin string }{
		"opencode": {"opencode-ai", "opencode"},
		"kilo":     {"@kilocode/cli", "kilo"},
		"cursor":   {"", ""},
	}
	for _, a := range got {
		w := want[a.Key]
		if w.npm != a.NPM || w.bin != a.Bin {
			t.Errorf("agent %q npm=%q bin=%q, want npm=%q bin=%q", a.Key, a.NPM, a.Bin, w.npm, w.bin)
		}
	}
}

func TestNpmInstallGlobalArgs(t *testing.T) {
	cases := []struct {
		pkg  string
		want []string
	}{
		{"opencode-ai", []string{"install", "-g", "--allow-scripts=opencode-ai", "opencode-ai"}},
		{"agent-browser", []string{"install", "-g", "--allow-scripts=agent-browser", "agent-browser"}},
		{"@kilocode/cli", []string{"install", "-g", "@kilocode/cli"}},
	}
	for _, tc := range cases {
		got := npmInstallGlobalArgs(tc.pkg)
		if len(got) != len(tc.want) {
			t.Fatalf("%s: got %v, want %v", tc.pkg, got, tc.want)
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Fatalf("%s: got %v, want %v", tc.pkg, got, tc.want)
			}
		}
	}
}

func TestNormalizeNPMPackage(t *testing.T) {
	cases := map[string]string{
		"@upstash/context7-mcp":      "@upstash/context7-mcp",
		"vercel-labs/agent-browser":  "github:vercel-labs/agent-browser",
		"github:foo/bar":             "github:foo/bar",
		"@playwright/mcp@latest":     "@playwright/mcp@latest",
	}
	for in, want := range cases {
		if got := normalizeNPMPackage(in); got != want {
			t.Errorf("normalizeNPMPackage(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestGlobalTools(t *testing.T) {
	got := GlobalTools()
	if len(got) != 3 {
		t.Fatalf("want 3 global tools, got %d", len(got))
	}
	want := map[string]string{
		"skills":     "skills",
		"cucumber":   "@cucumber/cucumber",
		"backlog.md": "backlog.md",
	}
	for _, tl := range got {
		if want[tl.Name] != tl.NPM {
			t.Errorf("tool %q npm=%q, want %q", tl.Name, tl.NPM, want[tl.Name])
		}
	}
}

func TestEnsureHomeStruct(t *testing.T) {
	home := t.TempDir()
	cfg := Config{
		HomeDir:   home,
		RepoHome:  home + "/.skillgrid",
		RepoDir:   home + "/.skillgrid/repos/skillgrid",
		AgentsDir: home + "/.agents",
	}
	if err := ensureHomeStruct(&cfg); err != nil {
		t.Fatalf("ensureHomeStruct: %v", err)
	}
	for _, d := range []string{cfg.RepoHome, cfg.RepoDir} {
		if info, err := os.Stat(d); err != nil || !info.IsDir() {
			t.Errorf("missing expected dir %s", d)
		}
	}
}

func TestCopyAll(t *testing.T) {
	src := t.TempDir()
	sub := filepath.Join(src, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("beta"), 0o644); err != nil {
		t.Fatal(err)
	}

	dst := t.TempDir()
	if err := copyAll(src, dst); err != nil {
		t.Fatalf("copyAll: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(dst, "a.txt")); err != nil || string(b) != "alpha" {
		t.Errorf("a.txt = %q", b)
	}
	if b, err := os.ReadFile(filepath.Join(dst, "sub", "b.txt")); err != nil || string(b) != "beta" {
		t.Errorf("b.txt = %q", b)
	}
}

func TestVerboseOut(t *testing.T) {
	// No panic is the success criterion; suppressed when flags are off.
	cfg := Config{}
	VerboseOut(&cfg, "hidden unless verbose or dry-run")
}

func TestSetupAgentsUnknown(t *testing.T) {
	cfg := Config{Agents: []string{"unknown"}}
	if err := setupAgents(&cfg); err == nil {
		t.Fatal("expected error for unknown agent")
	}
}

func TestSetupAgentsIntegration(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()

	files := []string{
		"plugins/opencode/mnemonic.ts",
		"plugins/kilo/mnemonic.ts",
		"plugins/opencode/memory-protocol.md",
		"plugins/kilo/memory-protocol.md",
		"plugins/cursor/mnemonic.mdc",
		"plugins/opencode/skillgrid-logo.tsx",
		"plugins/kilo/skillgrid-logo.tsx",
		"config.d/mcp.yaml",
	}
	for _, f := range files {
		p := filepath.Join(repo, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("mock content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mcpYAML := `servers:
  skillgrid-mnemonic:
    type: local
    command:
      - skillgrid
      - mcp
  context7:
    type: remote
    url: https://mcp.context7.com/mcp
`
	if err := os.WriteFile(filepath.Join(repo, "config.d", "mcp.yaml"), []byte(mcpYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)

	cfg := Config{
		HomeDir:   home,
		RepoDir:   repo,
		AgentsDir: filepath.Join(home, ".agents"),
		Agents:    []string{"opencode", "kilo", "cursor"},
	}

	if err := setupAgents(&cfg); err != nil {
		t.Fatalf("setupAgents: %v", err)
	}

	for _, p := range []string{
		filepath.Join(home, ".config", "opencode", "opencode.jsonc"),
		filepath.Join(home, ".config", "kilo", "kilo.jsonc"),
		filepath.Join(home, ".cursor", "mcp.json"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("missing expected config %s: %v", p, err)
		}
	}

	opencodeCfg, err := os.ReadFile(filepath.Join(home, ".config", "opencode", "opencode.jsonc"))
	if err != nil {
		t.Fatalf("read opencode config: %v", err)
	}
	if !strings.Contains(string(opencodeCfg), "skillgrid-mnemonic") {
		t.Errorf("opencode config missing skillgrid-mnemonic MCP: %s", opencodeCfg)
	}
	if !strings.Contains(string(opencodeCfg), "context7") {
		t.Errorf("opencode config missing context7 MCP: %s", opencodeCfg)
	}

	backupBase := filepath.Join(home, ".skillgrid", "backup")
	for _, agent := range []string{"opencode", "kilo", "cursor"} {
		agentDir := filepath.Join(backupBase, agent)
		entries, err := os.ReadDir(agentDir)
		if err != nil {
			t.Errorf("missing backup dir %s: %v", agentDir, err)
			continue
		}
		if len(entries) == 0 {
			t.Errorf("no backups in %s", agentDir)
		}
	}
}

func TestLoadMCPConfig(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `servers:
  skillgrid-mnemonic:
    type: local
    command:
      - skillgrid
      - mcp
  context7:
    type: remote
    url: https://mcp.context7.com/mcp
`
	if err := os.WriteFile(filepath.Join(repo, "config.d", "mcp.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := LoadMCPConfig(repo)
	if err != nil {
		t.Fatalf("LoadMCPConfig: %v", err)
	}
	if len(entries.Servers) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries.Servers))
	}
	found := map[string]bool{}
	for name, e := range entries.Servers {
		found[name] = true
		if name == "skillgrid-mnemonic" {
			if e.Type != "local" || len(e.Command) != 2 || e.Command[0] != "skillgrid" {
				t.Errorf("unexpected skillgrid-mnemonic entry: %+v", e)
			}
		}
		if name == "context7" {
			if e.Type != "remote" || e.URL != "https://mcp.context7.com/mcp" {
				t.Errorf("unexpected context7 entry: %+v", e)
			}
		}
	}
	if !found["skillgrid-mnemonic"] || !found["context7"] {
		t.Errorf("missing expected servers: %+v", found)
	}
}

func TestInstallMCPServers(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `agents:
  - "@kilocode/cli"
mcp:
  - "@upstash/context7-mcp"
  - "@playwright/mcp@latest"
`
	if err := os.WriteFile(filepath.Join(repo, "config.d", "tools.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		RepoDir: repo,
		DryRun:  true,
	}
	if err := installMCPServers(&cfg); err != nil {
		t.Fatalf("installMCPServers: %v", err)
	}
}
