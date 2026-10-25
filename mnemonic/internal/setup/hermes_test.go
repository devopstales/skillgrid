package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// hermesMCPEntries is the canonical config.d/mcp.yaml server set (the same
// names the real repo ships); tests pass it instead of loading the file so
// they stay hermetic.
var hermesMCPEntries = []MCPServerConfig{
	{Name: "context7", Type: "remote", URL: "https://mcp.context7.com/mcp"},
	{Name: "deepwiki", Type: "remote", URL: "https://mcp.deepwiki.com/mcp"},
	{Name: "mnemonic", Type: "local", Command: []string{"mnemonic", "mcp"}},
	{Name: "playwright", Type: "local", Command: []string{"playwright-mcp"}},
	{Name: "agent-browser", Type: "local", Command: []string{"agent-browser", "mcp", "--tools", "all"}},
	{Name: "exa", Type: "remote", URL: "https://mcp.exa.ai/mcp"},
	{Name: "trivy", Type: "local", Command: []string{"trivy", "mcp"}},
	{Name: "codescene", Type: "local", Command: []string{"npx", "@codescene/codehealth-mcp"}},
	{Name: "backlog", Type: "local", Command: []string{"backlog", "mcp", "start"}},
}

// stubKanban stubs the hermes Kanban CLI for the duration of the test so
// the suite never shells out to a real (possibly first-run, slow) hermes.
// It returns a pointer to the recorded calls.
func stubKanban(t *testing.T) *[]string {
	t.Helper()
	var calls []string
	orig := KanbanExec
	KanbanExec = func(hermesHome string, args ...string) (string, error) {
		calls = append(calls, strings.Join(args, " "))
		return "ok", nil
	}
	t.Cleanup(func() { KanbanExec = orig })
	return &calls
}

// hmcServers parses the mcp_servers block of a Hermes config.yaml into a
// name -> raw map for assertions.
func hmcServers(t *testing.T, cfgPath string) map[string]map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read hermes config: %v", err)
	}
	var doc struct {
		MCPServers map[string]map[string]interface{} `yaml:"mcp_servers"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse hermes config: %v", err)
	}
	return doc.MCPServers
}

// TestSetupHermes_MCPConfig is the acceptance scenario "happy path — MCP
// server is registered in Hermes config" + "edge — all MCP servers from
// config.d/mcp.yaml are registered": every server is written, the mnemonic
// entry pins MNEMONIC_MCP_EXCLUDE=mem, a re-run is idempotent, and a
// pre-existing user config is preserved (plus backed up).
func TestSetupHermes_MCPConfig(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	home := t.TempDir()
	stubKanban(t)
	// Pre-existing user config: an unrelated key + an existing mcp_servers
	// entry that must survive the upsert.
	cfgPath := filepath.Join(home, ".hermes", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	initial := "model: gpt-5\nmcp_servers:\n  user-server:\n    url: https://user.example/mcp\n"
	if err := os.WriteFile(cfgPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SetupHermes(home, repoRoot, hermesMCPEntries, false); err != nil {
		t.Fatalf("SetupHermes: %v", err)
	}

	servers := hmcServers(t, cfgPath)
	for _, e := range hermesMCPEntries {
		s, ok := servers[e.Name]
		if !ok {
			t.Fatalf("mcp_servers missing %q: %v", e.Name, servers)
		}
		if e.Type == "remote" {
			if s["url"] != e.URL {
				t.Fatalf("mcp_servers.%s.url = %v, want %s", e.Name, s["url"], e.URL)
			}
		} else {
			if s["command"] != e.Command[0] {
				t.Fatalf("mcp_servers.%s.command = %v, want %s", e.Name, s["command"], e.Command[0])
			}
		}
	}
	// The mnemonic entry carries the exclude env var (the provider handles
	// mem_* natively, so the MCP server must not re-expose them).
	if env, _ := servers["mnemonic"]["env"].(map[string]interface{}); env == nil || env[ExcludeEnvVar] != "mem" {
		t.Fatalf("mnemonic env = %v, want %s=mem", servers["mnemonic"]["env"], ExcludeEnvVar)
	}
	// The unrelated user config survives.
	data, _ := os.ReadFile(cfgPath)
	if !strings.Contains(string(data), "model: gpt-5") {
		t.Fatalf("user config lost: %s", data)
	}
	if _, ok := servers["user-server"]; !ok {
		t.Fatalf("pre-existing mcp_servers entry lost: %v", servers)
	}
	// A pre-write backup of the original config exists.
	bakDir := filepath.Join(home, ".skillgrid", "backup", "hermes")
	baks, err := os.ReadDir(bakDir)
	if err != nil || len(baks) == 0 {
		t.Fatalf("no hermes config backup in %s: %v", bakDir, err)
	}

	// Idempotency: a second run rewrites the same block, no duplication.
	if err := SetupHermes(home, repoRoot, hermesMCPEntries, false); err != nil {
		t.Fatalf("SetupHermes (re-run): %v", err)
	}
	servers = hmcServers(t, cfgPath)
	if len(servers) != len(hermesMCPEntries)+1 { // +1 for user-server
		t.Fatalf("re-run changed the server set: %v", servers)
	}
}

// TestSetupHermes_MalformedConfigYAML is the acceptance scenario "failure —
// malformed config.yaml triggers backup and error": the installer reports
// the YAML error, backs the original up, and does not overwrite it.
func TestSetupHermes_MalformedConfigYAML(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	home := t.TempDir()
	stubKanban(t)
	cfgPath := filepath.Join(home, ".hermes", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	malformed := "model: gpt-5\n  bad: [unclosed\n"
	if err := os.WriteFile(cfgPath, []byte(malformed), 0o644); err != nil {
		t.Fatal(err)
	}

	err := SetupHermes(home, repoRoot, hermesMCPEntries, false)
	if err == nil {
		t.Fatal("SetupHermes on malformed config: want error, got nil")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Fatalf("error should report the YAML parse failure: %v", err)
	}
	// The malformed file is left untouched.
	data, _ := os.ReadFile(cfgPath)
	if string(data) != malformed {
		t.Fatalf("malformed config was overwritten: %s", data)
	}
	// A backup of the original exists (the pre-write backup ran before the
	// parse attempt).
	bakDir := filepath.Join(home, ".skillgrid", "backup", "hermes")
	baks, _ := os.ReadDir(bakDir)
	if len(baks) == 0 {
		t.Fatalf("no backup of the malformed config in %s", bakDir)
	}
}

// TestSetupHermes_PluginCopy is the acceptance scenario "happy path — full
// install configures all layers" (provider layer): plugins/hermes/ is
// copied to $HERMES_HOME/plugins/mnemonic/ with __pycache__ excluded.
func TestSetupHermes_PluginCopy(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	home := t.TempDir()
	stubKanban(t)
	if err := SetupHermes(home, repoRoot, nil, false); err != nil {
		t.Fatalf("SetupHermes: %v", err)
	}
	dst := filepath.Join(home, ".hermes", "plugins", "mnemonic")
	for _, name := range []string{"__init__.py", "provider.py", "plugin.yaml"} {
		p := filepath.Join(dst, name)
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("missing plugin file %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dst, "__pycache__")); err == nil {
		t.Fatal("__pycache__ must not be copied")
	}
}

// TestSetupHermes_SoulMD is the acceptance scenarios "edge — protocol
// fallback in SOUL.md when provider is not active" + "failure — SOUL.md
// block is replaced on re-install": the marker block is written between the
// delimiters, surrounding content is preserved, and a re-run replaces the
// block in place instead of duplicating it.
func TestSetupHermes_SoulMD(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	home := t.TempDir()
	stubKanban(t)
	if err := SetupHermes(home, repoRoot, nil, false); err != nil {
		t.Fatalf("SetupHermes: %v", err)
	}
	soulPath := filepath.Join(home, ".hermes", "SOUL.md")
	soul, err := os.ReadFile(soulPath)
	if err != nil {
		t.Fatalf("read SOUL.md: %v", err)
	}
	if !strings.Contains(string(soul), hermesBeginMarker) || !strings.Contains(string(soul), hermesEndMarker) {
		t.Fatalf("SOUL.md missing markers: %s", soul)
	}
	if strings.Count(string(soul), hermesBeginMarker) != 1 || strings.Count(string(soul), hermesEndMarker) != 1 {
		t.Fatalf("SOUL.md has duplicate markers: %s", soul)
	}
	b := strings.Index(string(soul), hermesBeginMarker)
	e := strings.Index(string(soul), hermesEndMarker)
	if b < 0 || e < b {
		t.Fatalf("SOUL.md markers out of order: %s", soul)
	}
	// The protocol body sits between the markers.
	body := string(soul)[b+len(hermesBeginMarker) : e]
	if strings.TrimSpace(body) == "" {
		t.Fatalf("SOUL.md protocol block is empty: %s", soul)
	}

	// Surrounding content is preserved.
	home2 := t.TempDir()
	soulPath2 := filepath.Join(home2, ".hermes", "SOUL.md")
	if err := os.MkdirAll(filepath.Dir(soulPath2), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(soulPath2, []byte("# My Agent\n\nBe kind.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SetupHermes(home2, repoRoot, nil, false); err != nil {
		t.Fatalf("SetupHermes: %v", err)
	}
	updated, _ := os.ReadFile(soulPath2)
	if !strings.Contains(string(updated), "# My Agent") || !strings.Contains(string(updated), "Be kind.") {
		t.Fatalf("SOUL.md content not preserved: %s", updated)
	}

	// Re-install replaces the block in place (no duplication).
	if err := SetupHermes(home2, repoRoot, nil, false); err != nil {
		t.Fatalf("SetupHermes (re-run): %v", err)
	}
	again, _ := os.ReadFile(soulPath2)
	if strings.Count(string(again), hermesBeginMarker) != 1 {
		t.Fatalf("re-run duplicated the SOUL.md block: %s", again)
	}
}

// TestSetupHermes_HermesHome is the HERMES_HOME relocation contract: the
// env var relocates every artifact (config.yaml, SOUL.md, plugins, skills)
// away from the default ~/.hermes.
func TestSetupHermes_HermesHome(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	home := t.TempDir()
	stubKanban(t)
	custom := t.TempDir()
	t.Setenv(HermesHomeEnv, custom)

	if err := SetupHermes(home, repoRoot, hermesMCPEntries, false); err != nil {
		t.Fatalf("SetupHermes: %v", err)
	}
	// Everything lives under the custom home.
	if _, err := os.Stat(filepath.Join(custom, "config.yaml")); err != nil {
		t.Fatalf("config.yaml not under $%s: %v", HermesHomeEnv, err)
	}
	if _, err := os.Stat(filepath.Join(custom, "SOUL.md")); err != nil {
		t.Fatalf("SOUL.md not under $%s: %v", HermesHomeEnv, err)
	}
	if _, err := os.Stat(filepath.Join(custom, "plugins", "mnemonic", "provider.py")); err != nil {
		t.Fatalf("plugin not under $%s: %v", HermesHomeEnv, err)
	}
	// The default ~/.hermes was NOT touched (home is a temp dir).
	if _, err := os.Stat(filepath.Join(home, ".hermes")); err == nil {
		t.Fatal("default ~/.hermes was written despite $HERMES_HOME")
	}
}

// TestSetupHermes_SkillsAndTicketing checks the skills deployment (skill
// files land under $HERMES_HOME/skills/) and the ticketing config update
// (ticketing.type -> hermes-kanban on a scratch repo, unrelated keys
// preserved, disabled ticketing left alone, idempotent on re-run).
func TestSetupHermes_SkillsAndTicketing(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	home := t.TempDir()
	stubKanban(t)
	if err := SetupHermes(home, repoRoot, nil, false); err != nil {
		t.Fatalf("SetupHermes: %v", err)
	}
	skills := filepath.Join(home, ".hermes", "skills")
	entries, err := os.ReadDir(skills)
	if err != nil || len(entries) == 0 {
		t.Fatalf("skills not copied to %s: %v (entries=%d)", skills, err, len(entries))
	}

	// Ticketing config on a scratch repo (the real repo's .skillgrid is not
	// rewritten by tests).
	scratch := t.TempDir()
	if err := os.MkdirAll(filepath.Join(scratch, ".skillgrid"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(scratch, ".skillgrid", "config.yaml")
	initial := "schema: skillgrid/v1\nproject: demo\nticketing:\n  enabled: true\n  type: backlogmd\n  backlogmd:\n    dir: .backlog\n"
	if err := os.WriteFile(cfg, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SetupHermes(home, scratch, nil, false); err != nil {
		t.Fatalf("SetupHermes(scratch): %v", err)
	}
	merged, _ := os.ReadFile(cfg)
	s := string(merged)
	if !strings.Contains(s, "type: hermes-kanban") {
		t.Fatalf("ticketing.type not updated: %s", s)
	}
	if !strings.Contains(s, "board: demo") || !strings.Contains(s, "assignee: default") {
		t.Fatalf("kanban board/assignee missing: %s", s)
	}
	// Unrelated keys survive the merge.
	if !strings.Contains(s, "schema: skillgrid/v1") {
		t.Fatalf("config lost other keys: %s", s)
	}
	// Re-run is idempotent.
	if err := SetupHermes(home, scratch, nil, false); err != nil {
		t.Fatalf("SetupHermes(scratch re-run): %v", err)
	}
	again, _ := os.ReadFile(cfg)
	if strings.Count(string(again), "type: hermes-kanban") != 1 {
		t.Fatalf("re-run duplicated ticketing.type: %s", again)
	}
	// Disabled ticketing is left untouched.
	disabled := t.TempDir()
	if err := os.MkdirAll(filepath.Join(disabled, ".skillgrid"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfgD := filepath.Join(disabled, ".skillgrid", "config.yaml")
	disabledCfg := "project: demo\nticketing:\n  enabled: false\n  type: backlogmd\n"
	if err := os.WriteFile(cfgD, []byte(disabledCfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SetupHermes(home, disabled, nil, false); err != nil {
		t.Fatalf("SetupHermes(disabled): %v", err)
	}
	after, _ := os.ReadFile(cfgD)
	if string(after) != disabledCfg {
		t.Fatalf("disabled ticketing was rewritten: %s", after)
	}
}

// TestSetupHermes_KanbanBoard is the acceptance scenarios "edge — Kanban
// board is created if it does not exist" + "failure — Kanban board creation
// failure is non-fatal": the `hermes kanban boards create <slug>` call is
// made with the right slug (the config `project:` name), a failing CLI is a
// warning (install completes), and a dry-run only prints the command.
func TestSetupHermes_KanbanBoard(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	scratch := t.TempDir()
	if err := os.MkdirAll(filepath.Join(scratch, ".skillgrid"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scratch, ".skillgrid", "config.yaml"),
		[]byte("project: kanban-test-proj\nticketing:\n  enabled: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	kanbanHome := t.TempDir()
	t.Setenv(HermesHomeEnv, kanbanHome)

	// Success path: the CLI is invoked with the project slug.
	var gotArgs []string
	var gotHome string
	orig := KanbanExec
	KanbanExec = func(hermesHome string, args ...string) (string, error) {
		gotHome = hermesHome
		gotArgs = args
		return "board created", nil
	}
	if err := SetupHermes(home, scratch, nil, false); err != nil {
		t.Fatalf("SetupHermes (kanban ok): %v", err)
	}
	KanbanExec = orig
	if len(gotArgs) == 0 || gotArgs[0] != "kanban" || gotArgs[1] != "boards" ||
		gotArgs[2] != "create" || gotArgs[3] != "kanban-test-proj" {
		t.Fatalf("kanban create args = %v, want kanban boards create kanban-test-proj", gotArgs)
	}
	// HERMES_HOME is passed per-call to the CLI (no process-env mutation).
	if gotHome != kanbanHome {
		t.Fatalf("kanban CLI HERMES_HOME = %q, want %q", gotHome, kanbanHome)
	}

	// Failure path: a failing CLI is non-fatal (the install completes).
	KanbanExec = func(hermesHome string, args ...string) (string, error) {
		return "", fmt.Errorf("exit status 1")
	}
	home2 := t.TempDir()
	hermesHome2 := t.TempDir()
	t.Setenv(HermesHomeEnv, hermesHome2)
	if err := SetupHermes(home2, scratch, hermesMCPEntries, false); err != nil {
		t.Fatalf("kanban failure must be non-fatal: %v", err)
	}
	KanbanExec = orig
	// The rest of the install still completed (HERMES_HOME was set to a
	// temp dir for this run, so the config lives directly under it).
	if _, err := os.Stat(filepath.Join(hermesHome2, "config.yaml")); err != nil {
		t.Fatalf("MCP config missing after kanban failure: %v", err)
	}

	// Dry-run path: the CLI is never invoked, only printed.
	called := false
	KanbanExec = func(hermesHome string, args ...string) (string, error) {
		called = true
		return "", nil
	}
	if err := SetupHermes(t.TempDir(), scratch, nil, true); err != nil {
		t.Fatalf("SetupHermes dry-run: %v", err)
	}
	KanbanExec = orig
	if called {
		t.Fatal("dry-run must not invoke the kanban CLI")
	}
}

// TestSetupHermes_DryRun is the acceptance scenario "edge — dry-run prints
// planned changes without writing": nothing lands on disk (plugins, config,
// SOUL.md, skills) and the kanban create is only printed.
func TestSetupHermes_DryRun(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	home := t.TempDir()
	kanbanHome := t.TempDir()
	t.Setenv(HermesHomeEnv, kanbanHome)
	stubKanban(t)
	if err := SetupHermes(home, repoRoot, hermesMCPEntries, true); err != nil {
		t.Fatalf("SetupHermes dry-run: %v", err)
	}
	// $HERMES_HOME must be empty: no config, no SOUL.md, no plugins, no
	// skills, no kanban state.
	if entries, _ := os.ReadDir(kanbanHome); len(entries) > 0 {
		t.Fatalf("dry-run wrote to $%s: %v", HermesHomeEnv, entries)
	}
	// No backup dir (no pre-existing config to back up).
	if entries, _ := os.ReadDir(filepath.Join(home, ".skillgrid", "backup", "hermes")); len(entries) > 0 {
		t.Fatalf("dry-run wrote a backup: %v", entries)
	}
}

// TestSetupHermes_PluginSourceMissingWarns is the QA-fix scenario: when the
// repo has no plugins/hermes/ source dir, copyDir must not fail the install
// (missing source stays a no-op) but it must warn instead of silently
// skipping.
func TestSetupHermes_PluginSourceMissingWarns(t *testing.T) {
	// Scratch repo: everything SetupHermes needs EXCEPT plugins/hermes/.
	// The memory protocol is the one file writeHermesSoul refuses to run
	// without.
	repoRoot := t.TempDir()
	protocolPath := filepath.Join(repoRoot, "plugins", "opencode", "memory-protocol.md")
	if err := os.MkdirAll(filepath.Dir(protocolPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(protocolPath, []byte("# Memory protocol\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "plugins", "hermes")); !os.IsNotExist(err) {
		t.Fatalf("scratch repo unexpectedly has plugins/hermes: %v", err)
	}

	home := t.TempDir()
	stubKanban(t)

	if err := SetupHermes(home, repoRoot, nil, false); err != nil {
		t.Fatalf("missing plugin source must be non-fatal: %v", err)
	}
	// The install continued past the missing source: SOUL.md was written.
	if _, err := os.Stat(filepath.Join(home, ".hermes", "SOUL.md")); err != nil {
		t.Fatalf("install did not continue after the missing-source warn: %v", err)
	}
	// No plugin dir was created (nothing was copied).
	if _, err := os.Stat(filepath.Join(home, ".hermes", "plugins", "mnemonic")); !os.IsNotExist(err) {
		t.Fatalf("plugin dir should not exist when the source is missing: %v", err)
	}
	// The warning was logged: the source path named in the message.
	msg := copyDirWarnText(t, filepath.Join(repoRoot, "plugins", "hermes"))
	if msg == "" {
		t.Fatal("missing plugin source did not produce a warning")
	}
}

// copyDirWarnText runs copyDir against a missing src and returns the line of
// stderr output that names the source, so the warning assertion is exact
// rather than a substring of unrelated log noise.
func copyDirWarnText(t *testing.T, src string) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	err = copyDir(src, t.TempDir(), false)
	w.Close()
	os.Stderr = old
	if err != nil {
		t.Fatalf("copyDir(missing src) must not error: %v", err)
	}
	var out []byte
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		out = append(out, buf[:n]...)
		if err != nil {
			break
		}
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "warn") && strings.Contains(line, src) {
			return line
		}
	}
	t.Logf("stderr was: %q", string(out))
	return ""
}

// TestSetupHermes_ReadOnlySkillsDir is the acceptance scenario "failure —
// read-only skills directory does not fail the install": when
// $HERMES_HOME/skills is read-only, the skills copy fails but the install
// continues (plugin, MCP config, and SOUL.md are already in place).
func TestSetupHermes_ReadOnlySkillsDir(t *testing.T) {
	// Scratch repo with a memory protocol (writeHermesSoul needs it).
	repoRoot := t.TempDir()
	protocolPath := filepath.Join(repoRoot, "plugins", "opencode", "memory-protocol.md")
	if err := os.MkdirAll(filepath.Dir(protocolPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(protocolPath, []byte("# Memory protocol\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A skills source so copyDir has something to copy.
	if err := os.MkdirAll(filepath.Join(repoRoot, ".agents", "skills", "test-skill"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".agents", "skills", "test-skill", "SKILL.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	hermesHome := t.TempDir()
	t.Setenv(HermesHomeEnv, hermesHome)
	stubKanban(t)

	// Pre-create the skills directory and make it read-only BEFORE the
	// install. copyDir creates subdirs via MkdirAll, but if the target
	// skills/ dir already exists and is read-only, MkdirAll for
	// skills/test-skill/ will fail with a permission error.
	if err := os.MkdirAll(filepath.Join(hermesHome, "skills"), 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Chmod(filepath.Join(hermesHome, "skills"), 0o755)
	})

	// The install must NOT return an error (skills copy is non-fatal).
	if err := SetupHermes(home, repoRoot, hermesMCPEntries, false); err != nil {
		t.Fatalf("read-only skills dir must be non-fatal: %v", err)
	}

	// MCP config was written (the install continued past the skills copy).
	cfgPath := filepath.Join(hermesHome, "config.yaml")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("MCP config missing after read-only skills: %v", err)
	}
	// SOUL.md was written (the install continued).
	soulPath := filepath.Join(hermesHome, "SOUL.md")
	if _, err := os.Stat(soulPath); err != nil {
		t.Fatalf("SOUL.md missing after read-only skills: %v", err)
	}
}

// TestKanbanTicketPublishRoundTrip is the integration QA-fix scenario: a
// parent task and a child task created with --parent must round-trip through
// the real hermes Kanban CLI — list shows both, show <child> exposes the
// parent link. Gated on the hermes binary; everything runs against a scratch
// HERMES_HOME so the real one is never touched.
func TestKanbanTicketPublishRoundTrip(t *testing.T) {
	if _, err := exec.LookPath("hermes"); err != nil {
		t.Skip("hermes binary not on PATH")
	}
	if testing.Short() {
		t.Skip("skipping hermes integration test in -short mode")
	}
	// Scratch data root with a pre-warmed uv cache: a FRESH HERMES_HOME
	// makes hermes run `uv sync` for the runtime venv on the first kanban
	// call — minutes of dependency prep, and a hard failure on machines
	// without a working OpenSSL (the cryptography sdist then compiles
	// from source). Symlinking the real HERMES_HOME uv cache makes the
	// first call fast (cache hit) and keeps the scratch root hermetic for
	// kanban.db and friends.
	hermesHome := t.TempDir()
	if defaultHome, err := os.UserHomeDir(); err == nil {
		cacheSrc := filepath.Join(defaultHome, ".hermes", "cache", "uv")
		if _, err := os.Stat(cacheSrc); err == nil {
			if err := os.MkdirAll(filepath.Join(hermesHome, "cache"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(cacheSrc, filepath.Join(hermesHome, "cache", "uv")); err != nil {
				t.Fatal(err)
			}
		}
	}

	run := func(t *testing.T, args ...string) (string, error) {
		t.Helper()
		// The first call also covers hermes's one-time runtime prep for the
		// scratch data root; every later call is a plain kanban operation.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "hermes", args...)
		cmd.Env = append(os.Environ(), HermesHomeEnv+"="+hermesHome)
		devNull, err := os.OpenFile(os.DevNull, os.O_RDONLY, 0)
		if err == nil {
			cmd.Stdin = devNull
			defer devNull.Close()
		}
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	board := "qa-test-board"
	if out, err := run(t, "kanban", "boards", "create", board); err != nil {
		t.Fatalf("boards create: %v: %s", err, out)
	}

	// Parent task.
	out, err := run(t, "kanban", "--board", board, "create", "QA parent task", "--workspace", "scratch")
	if err != nil {
		t.Fatalf("create parent: %v: %s", err, out)
	}
	parentID := kanbanTaskID(t, out)

	// Child task, linked at create time.
	out, err = run(t, "kanban", "--board", board, "create", "QA child task", "--workspace", "scratch", "--parent", parentID)
	if err != nil {
		t.Fatalf("create child: %v: %s", err, out)
	}
	childID := kanbanTaskID(t, out)

	// list shows both tasks.
	list, err := run(t, "kanban", "--board", board, "list", "--json")
	if err != nil {
		t.Fatalf("list: %v: %s", err, list)
	}
	var tasks []map[string]interface{}
	if err := json.Unmarshal([]byte(list), &tasks); err != nil {
		t.Fatalf("parse list JSON: %v: %s", err, list)
	}
	ids := map[string]bool{}
	for _, task := range tasks {
		if id, _ := task["id"].(string); id != "" {
			ids[id] = true
		}
	}
	if !ids[parentID] || !ids[childID] {
		t.Fatalf("list missing parent/child: ids=%v want %s and %s", ids, parentID, childID)
	}

	// show <child> exposes the parent link.
	show, err := run(t, "kanban", "--board", board, "show", childID, "--json")
	if err != nil {
		t.Fatalf("show child: %v: %s", err, show)
	}
	if !strings.Contains(show, parentID) {
		t.Fatalf("show <child> does not reference the parent id %s: %s", parentID, show)
	}

	// Cleanup: archive the test tasks (the board itself is in the scratch
	// HERMES_HOME temp dir).
	for _, id := range []string{childID, parentID} {
		if out, err := run(t, "kanban", "--board", board, "archive", id); err != nil {
			t.Logf("archive %s (cleanup): %v: %s", id, err, out)
		}
	}
}

// kanbanTaskID extracts the created task id from `hermes kanban create`
// output (JSON object/array with an id field, or a `t<digits>` token).
func kanbanTaskID(t *testing.T, out string) string {
	t.Helper()
	if strings.HasPrefix(strings.TrimSpace(out), "{") || strings.HasPrefix(strings.TrimSpace(out), "[") {
		var one map[string]interface{}
		var many []map[string]interface{}
		unmarshalErrs := []error{json.Unmarshal([]byte(out), &one), json.Unmarshal([]byte(out), &many)}
		if unmarshalErrs[0] == nil {
			many = []map[string]interface{}{one}
		}
		for _, m := range many {
			if id, ok := m["id"].(string); ok && id != "" {
				return id
			}
		}
	}
	// Text form: "Created t_c6ce5dc3  (ready, assignee=-)" — hermes task
	// ids are t_ followed by hex.
	if m := regexp.MustCompile(`t_[0-9a-f]+`).FindString(out); m != "" {
		return m
	}
	t.Fatalf("cannot find task id in create output: %s", out)
	return ""
}

// TestMergeTicketingType exercises the surgical text merge directly: the
// type line is replaced, kanban keys are inserted, unrelated keys survive,
// and an already-installed config round-trips unchanged.
func TestMergeTicketingType(t *testing.T) {
	doc := "schema: skillgrid/v1\n# keep me\nticketing:\n  enabled: true\n  type: backlogmd\n  backlogmd:\n    dir: .backlog\n"
	out := mergeTicketingType(doc, "hermes-kanban", "myproj")
	if !strings.Contains(out, "# keep me") {
		t.Fatalf("comment lost: %s", out)
	}
	if !strings.Contains(out, "type: hermes-kanban") {
		t.Fatalf("type not replaced: %s", out)
	}
	if !strings.Contains(out, "board: myproj") || !strings.Contains(out, "assignee: default") {
		t.Fatalf("kanban keys missing: %s", out)
	}
	if strings.Contains(out, "type: backlogmd") {
		t.Fatalf("old type still present: %s", out)
	}

	already := mergeTicketingType(out, "hermes-kanban", "myproj")
	if already != out {
		t.Fatalf("installed config is not a fixed point:\n%s\n---\n%s", out, already)
	}

	// No ticketing block at all: a minimal one is appended.
	bare := "project: p\n"
	appended := mergeTicketingType(bare, "hermes-kanban", "p")
	if !strings.Contains(appended, "ticketing:") || !strings.Contains(appended, "type: hermes-kanban") {
		t.Fatalf("no ticketing block appended: %s", appended)
	}
}

// TestMergeTicketingType_AssigneePreserved is the whole-branch review fix:
// a user-set kanban.assignee (non-empty, non-default) must survive the
// install instead of being silently reset to default; an empty or
// default-valued assignee still becomes `default`.
func TestMergeTicketingType_AssigneePreserved(t *testing.T) {
	// User-set profile: kept.
	user := "ticketing:\n  enabled: true\n  type: backlogmd\n  kanban:\n    assignee: alice\n"
	out := mergeTicketingType(user, "hermes-kanban", "p")
	if !strings.Contains(out, "assignee: alice") {
		t.Fatalf("user-set assignee not preserved: %s", out)
	}
	if strings.Contains(out, "assignee: default") {
		t.Fatalf("user-set assignee overwritten: %s", out)
	}

	// Existing default: stays default (fixed point).
	def := "ticketing:\n  enabled: true\n  kanban:\n    assignee: default\n"
	outDef := mergeTicketingType(def, "hermes-kanban", "p")
	if !strings.Contains(outDef, "assignee: default") {
		t.Fatalf("default assignee changed: %s", outDef)
	}

	// Empty assignee: becomes default.
	empty := "ticketing:\n  enabled: true\n  kanban:\n    assignee: \"\"\n"
	outEmpty := mergeTicketingType(empty, "hermes-kanban", "p")
	if !strings.Contains(outEmpty, "assignee: default") {
		t.Fatalf("empty assignee not normalized to default: %s", outEmpty)
	}
}
