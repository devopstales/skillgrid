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
		"@upstash/context7-mcp":     "@upstash/context7-mcp",
		"vercel-labs/agent-browser": "github:vercel-labs/agent-browser",
		"github:foo/bar":            "github:foo/bar",
		"@playwright/mcp@latest":    "@playwright/mcp@latest",
	}
	for in, want := range cases {
		if got := normalizeNPMPackage(in); got != want {
			t.Errorf("normalizeNPMPackage(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestGlobalTools(t *testing.T) {
	got := GlobalTools()
	if len(got) != 4 {
		t.Fatalf("want 4 global tools, got %d", len(got))
	}
	want := map[string][2]string{
		"skills":     {"skills", "skills"},
		"cucumber":   {"@cucumber/cucumber", "cucumber-js"},
		"backlog.md": {"backlog.md", "backlog"},
		"jscpd":      {"jscpd", "jscpd"},
	}
	for _, tl := range got {
		w, ok := want[tl.Name]
		if !ok {
			t.Errorf("unexpected tool %q", tl.Name)
			continue
		}
		if tl.NPM != w[0] {
			t.Errorf("tool %q npm=%q, want %q", tl.Name, tl.NPM, w[0])
		}
		if tl.Bin != w[1] {
			t.Errorf("tool %q bin=%q, want %q", tl.Name, tl.Bin, w[1])
		}
	}
}

func TestSecurityTools(t *testing.T) {
	got := SecurityTools()
	if len(got) != 4 {
		t.Fatalf("want 4 security tools, got %d", len(got))
	}
	want := map[string]struct {
		manager string
		bin     string
		args    []string
	}{
		"wapiti3": {
			manager: "uv",
			bin:     "wapiti3",
			args:    []string{"tool", "install", "wapiti3"},
		},
		"akca": {
			manager: "go",
			bin:     "akca",
			args:    []string{"install", "github.com/akha-security/akca/engine/cmd/akca@latest"},
		},
		"nuclei": {
			manager: "go",
			bin:     "nuclei",
			args:    []string{"install", "-v", "github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest"},
		},
		"trivy": {
			manager: "go",
			bin:     "trivy",
			args:    []string{"install", "github.com/aquasecurity/trivy/cmd/trivy@latest"},
		},
	}
	for _, s := range got {
		w, ok := want[s.Name]
		if !ok {
			t.Errorf("unexpected security tool %q", s.Name)
			continue
		}
		if s.Manager != w.manager {
			t.Errorf("tool %q manager=%q, want %q", s.Name, s.Manager, w.manager)
		}
		if s.Bin != w.bin {
			t.Errorf("tool %q bin=%q, want %q", s.Name, s.Bin, w.bin)
		}
		if len(s.InstallArgs) != len(w.args) {
			t.Errorf("tool %q args=%v, want %v", s.Name, s.InstallArgs, w.args)
			continue
		}
		for i := range w.args {
			if s.InstallArgs[i] != w.args[i] {
				t.Errorf("tool %q args[%d]=%q, want %q", s.Name, i, s.InstallArgs[i], w.args[i])
			}
		}
	}
}

func TestInstallSecurityToolsDryRun(t *testing.T) {
	// Dry-run exercises the full loop (skip-if-present, manager-present check,
	// and the dry-run branch) without invoking go/uv.
	cfg := Config{DryRun: true}
	if err := installSecurityTools(&cfg); err != nil {
		t.Fatalf("installSecurityTools (dry-run): %v", err)
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

// TestMemoryAndPersonasBoundary proves the split: the memory writer registers
// MCP, and the installer distributes personas, both from the same Run-shaped
// sequence, without either package reaching into the other's concern.
func TestMemoryAndPersonasBoundary(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()

	// Memory assets the setup writers need.
	for _, f := range []string{
		"plugins/opencode/hooks.yaml",
		"plugins/kilo/hooks.yaml",
		"plugins/opencode/memory-protocol.md",
		"plugins/kilo/memory-protocol.md",
		"plugins/opencode/skillgrid-checkpoint.ts",
		"plugins/kilo/skillgrid-checkpoint.ts",
		"plugins/_shared/memory-protocol.md",
		"rules/mnemonic.mdc",
		".cursor-plugin/plugin.json",
		"plugins/opencode/skillgrid-logo.tsx",
		"plugins/kilo/skillgrid-logo.tsx",
		"hooks/opencode-session-start.sh",
		"hooks/opencode-session-end.sh",
		"hooks/opencode-policy.sh",
		"hooks/opencode-tool-capture.sh",
		"hooks/tool-call-capture.js",
		"hooks/stop-tests.js",
		"hooks/gate-stop.js",
	} {
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
`
	if err := os.MkdirAll(filepath.Join(repo, "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "config.d", "mcp.yaml"), []byte(mcpYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// Persona source in the synced repo.
	personaSrc := filepath.Join(repo, "agents", "opencode")
	if err := os.MkdirAll(personaSrc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(personaSrc, "explorer.md"), []byte("persona"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)
	cfg := Config{
		HomeDir: home,
		RepoDir: repo,
		Agents:  []string{"opencode"},
	}

	// Memory: setupAgents (MCP + plugin + protocol).
	if err := setupAgents(&cfg); err != nil {
		t.Fatalf("setupAgents: %v", err)
	}
	// Harness: backup + config + personas (the installer's lane).
	if err := backupAgentConfigs(&cfg); err != nil {
		t.Fatalf("backupAgentConfigs: %v", err)
	}
	if err := installAgentConfig(&cfg, "opencode"); err != nil {
		t.Fatalf("installAgentConfig: %v", err)
	}
	if err := installPersonas(&cfg, "opencode"); err != nil {
		t.Fatalf("installPersonas: %v", err)
	}

	// Memory side: MCP registered.
	opencodeCfg, err := os.ReadFile(filepath.Join(home, ".config", "opencode", "opencode.jsonc"))
	if err != nil {
		t.Fatalf("read opencode config: %v", err)
	}
	if !strings.Contains(string(opencodeCfg), "skillgrid-mnemonic") {
		t.Errorf("memory setup did not register MCP: %s", opencodeCfg)
	}

	// Harness side: persona flattened into the global agent dir.
	got := filepath.Join(home, ".config", "opencode", "agents", "explorer.md")
	if b, err := os.ReadFile(got); err != nil || string(b) != "persona" {
		t.Errorf("persona not installed at %s: %q, %v", got, b, err)
	}
	// It must NOT be nested under a redundant opencode/ segment.
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "agents", "opencode", "explorer.md")); err == nil {
		t.Errorf("persona should be flattened, not nested under agents/opencode/")
	}
}

func TestSetupAgentsIntegration(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()

	files := []string{
		"plugins/opencode/hooks.yaml",
		"plugins/kilo/hooks.yaml",
		"plugins/opencode/memory-protocol.md",
		"plugins/kilo/memory-protocol.md",
		"plugins/opencode/skillgrid-checkpoint.ts",
		"plugins/kilo/skillgrid-checkpoint.ts",
		"plugins/_shared/memory-protocol.md",
		"rules/mnemonic.mdc",
		".cursor-plugin/plugin.json",
		"plugins/opencode/skillgrid-logo.tsx",
		"plugins/kilo/skillgrid-logo.tsx",
		"hooks/opencode-session-start.sh",
		"hooks/opencode-session-end.sh",
		"hooks/opencode-policy.sh",
		"hooks/opencode-tool-capture.sh",
		"hooks/tool-call-capture.js",
		"hooks/stop-tests.js",
		"hooks/gate-stop.js",
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

func installRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		script := filepath.Join(dir, "hooks", "opencode-session-start.sh")
		yaml := filepath.Join(dir, "plugins", "opencode", "hooks.yaml")
		if _, err := os.Stat(script); err == nil {
			if _, err := os.Stat(yaml); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("skillgrid repo root not found")
	return ""
}

// TestInstallCreatesOpenCodeHookLayout is the installer's agent-config step
// (setupAgents): OpenCode and Kilo configs, hook yaml, checkpoint plugin, and
// the scripts under ~/.skillgrid/hooks/.
func TestInstallCreatesOpenCodeHookLayout(t *testing.T) {
	repo := installRepoRoot(t)
	home := t.TempDir()
	cfg := Config{
		HomeDir: home,
		RepoDir: repo,
		Agents:  []string{"opencode", "kilo"},
	}
	if err := setupAgents(&cfg); err != nil {
		t.Fatalf("setupAgents: %v", err)
	}

	scripts := []string{
		"opencode-session-start.sh",
		"opencode-session-end.sh",
		"opencode-policy.sh",
		"opencode-tool-capture.sh",
		"tool-call-capture.js",
		"stop-tests.js",
		"gate-stop.js",
	}
	for _, name := range scripts {
		dst := filepath.Join(home, ".skillgrid", "hooks", name)
		fi, err := os.Stat(dst)
		if err != nil {
			t.Fatalf("missing %s: %v", dst, err)
		}
		if strings.HasSuffix(name, ".sh") && fi.Mode()&0o111 == 0 {
			t.Fatalf("%s is not executable", dst)
		}
		want, err := os.ReadFile(filepath.Join(repo, "hooks", name))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(dst)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Fatalf("%s installed content differs from the repo", name)
		}
	}

	opencodeYAML, err := os.ReadFile(filepath.Join(home, ".config", "opencode", "hook", "hooks.yaml"))
	if err != nil {
		t.Fatalf("opencode hooks.yaml: %v", err)
	}
	kiloYAML, err := os.ReadFile(filepath.Join(home, ".config", "kilo", "hook", "hooks.yaml"))
	if err != nil {
		t.Fatalf("kilo hooks.yaml: %v", err)
	}
	for _, name := range []string{
		"opencode-session-start.sh",
		"opencode-session-end.sh",
		"opencode-policy.sh",
		"opencode-tool-capture.sh",
	} {
		if !strings.Contains(string(opencodeYAML), name) || !strings.Contains(string(kiloYAML), name) {
			t.Fatalf("%s missing from installed hooks.yaml", name)
		}
	}
	if !strings.Contains(string(kiloYAML), "SKILLGRID_AGENT=kilo") {
		t.Fatalf("kilo hooks.yaml must set SKILLGRID_AGENT=kilo:\n%s", kiloYAML)
	}

	for _, rel := range []string{
		".config/opencode/plugin/skillgrid-checkpoint.ts",
		".config/kilo/plugin/skillgrid-checkpoint.ts",
	} {
		b, err := os.ReadFile(filepath.Join(home, rel))
		if err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
		if !strings.Contains(string(b), "SkillgridCheckpoint") {
			t.Fatalf("%s is not the checkpoint plugin", rel)
		}
	}

	for _, rel := range []string{
		".config/opencode/opencode.jsonc",
		".config/kilo/kilo.jsonc",
	} {
		b, err := os.ReadFile(filepath.Join(home, rel))
		if err != nil {
			t.Fatal(err)
		}
		text := string(b)
		if !strings.Contains(text, "opencode-yaml-hooks") || !strings.Contains(text, "./plugin/skillgrid-checkpoint.ts") {
			t.Fatalf("%s missing current plugins:\n%s", rel, text)
		}
		if strings.Contains(text, "opencode-command-hooks") || strings.Contains(text, "mnemonic.ts") {
			t.Fatalf("%s still registers a retired plugin:\n%s", rel, text)
		}
	}
}
