package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

const checkpointPluginMarker = "export const SkillgridCheckpoint"

func TestSetupOpenCode_CheckpointPlugin(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	home := t.TempDir()
	if err := SetupOpenCode(home, repoRoot, nil, nil, false); err != nil {
		t.Fatalf("SetupOpenCode: %v", err)
	}
	dst := filepath.Join(home, ".config", "opencode", "plugin", "skillgrid-checkpoint.ts")
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read checkpoint plugin: %v", err)
	}
	if !strings.Contains(string(data), checkpointPluginMarker) {
		t.Fatalf("checkpoint plugin missing marker %q", checkpointPluginMarker)
	}
	cfgData, err := os.ReadFile(AgentConfigPath(home, "opencode"))
	if err != nil {
		t.Fatalf("read opencode config: %v", err)
	}
	if !strings.Contains(string(cfgData), "./plugin/skillgrid-checkpoint.ts") {
		t.Fatalf("opencode config missing checkpoint plugin entry: %s", cfgData)
	}

	homeDry := t.TempDir()
	cfgPath := AgentConfigPath(homeDry, "opencode")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SetupOpenCode(homeDry, repoRoot, nil, nil, true); err != nil {
		t.Fatalf("SetupOpenCode dry-run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(homeDry, ".config", "opencode", "plugin", "skillgrid-checkpoint.ts")); err == nil {
		t.Fatal("dry-run must not write checkpoint plugin")
	}
	if _, err := os.Stat(filepath.Join(homeDry, ".skillgrid", "hooks", "opencode-session-start.sh")); err == nil {
		t.Fatal("dry-run must not write opencode hook scripts")
	}
}

func TestSetupOpenCode_DropsRetiredPlugins(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	home := t.TempDir()
	cfgPath := AgentConfigPath(home, "opencode")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	initial := `{"plugin":["opencode-command-hooks","./plugins/mnemonic.ts","keep-me"]}` + "\n"
	if err := os.WriteFile(cfgPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SetupOpenCode(home, repoRoot, nil, nil, false); err != nil {
		t.Fatalf("SetupOpenCode: %v", err)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, p := range gjson.Parse(string(data)).Get("plugin").Array() {
		got[p.String()] = true
	}
	for _, retired := range []string{"opencode-command-hooks", "./plugins/mnemonic.ts"} {
		if got[retired] {
			t.Fatalf("retired plugin %q still registered: %s", retired, data)
		}
	}
	for _, keep := range []string{"keep-me", "opencode-yaml-hooks", "./plugin/skillgrid-checkpoint.ts"} {
		if !got[keep] {
			t.Fatalf("missing plugin %q: %s", keep, data)
		}
	}
}

func TestSetupOpenCode_InstallsHookScripts(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	home := t.TempDir()
	if err := SetupOpenCode(home, repoRoot, nil, nil, false); err != nil {
		t.Fatalf("SetupOpenCode: %v", err)
	}
	scripts := []string{
		"opencode-session-start.sh",
		"opencode-session-end.sh",
		"opencode-policy.sh",
		"opencode-tool-capture.sh",
	}
	for _, name := range scripts {
		p := filepath.Join(home, ".skillgrid", "hooks", name)
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
		if fi.Mode()&0o111 == 0 {
			t.Fatalf("%s is not executable", name)
		}
	}
	cfg, err := os.ReadFile(filepath.Join(home, ".config", "opencode", "hook", "hooks.yaml"))
	if err != nil {
		t.Fatalf("read hooks.yaml: %v", err)
	}
	text := string(cfg)
	for _, name := range scripts {
		if !strings.Contains(text, name) {
			t.Fatalf("hooks.yaml missing %s:\n%s", name, text)
		}
	}
	if strings.Contains(text, "node ") || strings.Contains(text, "node -e") {
		t.Fatalf("hooks.yaml still inlines hook logic:\n%s", text)
	}
}

// TestUpsertPrivateToolsEnv is the installer-writes-private-tools-env seam:
// the opencode installer writes SKILLGRID_MNEMONIC_PRIVATE_TOOLS into the
// harness env map from the config allowlist, keeping the Go config and the
// plugin env in sync, and leaves the config untouched when the list is empty.
func TestUpsertPrivateToolsEnv(t *testing.T) {
	home := t.TempDir()
	cfgPath := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Non-empty allowlist → the env var is written (comma-joined).
	if err := upsertPrivateToolsEnv(cfgPath, []string{"mem_save", "mem_save_prompt"}, false); err != nil {
		t.Fatalf("upsertPrivateToolsEnv: %v", err)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	got := gjson.Parse(string(data)).Get("env.SKILLGRID_MNEMONIC_PRIVATE_TOOLS").String()
	if got != "mem_save,mem_save_prompt" {
		t.Fatalf("env.SKILLGRID_MNEMONIC_PRIVATE_TOOLS = %q, want mem_save,mem_save_prompt", got)
	}

	// Empty allowlist → no env key is created.
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := upsertPrivateToolsEnv(cfgPath, nil, false); err != nil {
		t.Fatalf("upsertPrivateToolsEnv(nil): %v", err)
	}
	data, err = os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if gjson.Parse(string(data)).Get("env").Exists() {
		t.Fatalf("empty allowlist must not create an env key, got %s", data)
	}
}
