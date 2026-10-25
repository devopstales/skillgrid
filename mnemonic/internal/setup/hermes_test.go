package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	KanbanExec = func(args ...string) (string, error) {
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
	t.Setenv(HermesHomeEnv, t.TempDir())

	// Success path: the CLI is invoked with the project slug.
	var gotArgs []string
	orig := KanbanExec
	KanbanExec = func(args ...string) (string, error) {
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

	// Failure path: a failing CLI is non-fatal (the install completes).
	KanbanExec = func(args ...string) (string, error) {
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
	KanbanExec = func(args ...string) (string, error) {
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
