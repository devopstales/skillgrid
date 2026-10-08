package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tidwall/gjson"
)

func TestSetupOpenCode_InstallsPlugins(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	home := t.TempDir()
	if err := SetupOpenCode(home, repoRoot, nil, nil, false); err != nil {
		t.Fatalf("SetupOpenCode: %v", err)
	}
	// The 5 native TS plugins are copied into the harness plugin dir.
	for _, base := range nativePluginBases {
		dst := filepath.Join(home, ".config", "opencode", "plugin", base+".ts")
		if _, err := os.Stat(dst); err != nil {
			t.Fatalf("missing plugin %s: %v", base, err)
		}
	}
	cfgData, err := os.ReadFile(AgentConfigPath(home, "opencode"))
	if err != nil {
		t.Fatalf("read opencode config: %v", err)
	}
	plugins := pluginEntries(cfgData)
	for _, base := range nativePluginBases {
		if !plugins["./plugin/"+base+".ts"] {
			t.Fatalf("opencode config missing plugin entry %s: %s", base, cfgData)
		}
	}
	// The retired hook file and checkpoint plugin are no longer installed.
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "hook", "hooks.yaml")); err == nil {
		t.Fatal("hooks.yaml should not be installed")
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "plugin", "skillgrid-checkpoint.ts")); err == nil {
		t.Fatal("skillgrid-checkpoint.ts should not be installed")
	}
	if plugins["opencode-yaml-hooks"] {
		t.Fatalf("opencode config should not register opencode-yaml-hooks: %s", cfgData)
	}

	// Dry-run must not write the plugins.
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
	for _, base := range nativePluginBases {
		if _, err := os.Stat(filepath.Join(homeDry, ".config", "opencode", "plugin", base+".ts")); err == nil {
			t.Fatalf("dry-run must not write plugin %s", base)
		}
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
	initial := `{"plugin":["opencode-command-hooks","./plugins/mnemonic.ts","opencode-yaml-hooks","./plugin/skillgrid-checkpoint.ts","keep-me"]}` + "\n"
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
	got := pluginEntries(data)
	for _, retired := range []string{"opencode-command-hooks", "./plugins/mnemonic.ts", "opencode-yaml-hooks", "./plugin/skillgrid-checkpoint.ts"} {
		if got[retired] {
			t.Fatalf("retired plugin %q still registered: %s", retired, data)
		}
	}
	want := map[string]bool{"keep-me": true}
	for _, base := range nativePluginBases {
		want["./plugin/"+base+".ts"] = true
	}
	for entry := range want {
		if !got[entry] {
			t.Fatalf("missing plugin %q: %s", entry, data)
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
	// Only the 3 shared workers are installed now.
	for _, name := range openCodeHookScripts {
		p := filepath.Join(home, ".skillgrid", "hooks", name)
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("missing shared worker %s: %v", name, err)
		}
	}
	// The retired shell adapters are gone.
	for _, name := range []string{"opencode-session-start.sh", "opencode-session-end.sh", "opencode-policy.sh", "opencode-tool-capture.sh"} {
		p := filepath.Join(home, ".skillgrid", "hooks", name)
		if _, err := os.Stat(p); err == nil {
			t.Fatalf("retired shell script %s should not be installed", name)
		}
	}
	// No hooks.yaml for opencode anymore.
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "hook", "hooks.yaml")); err == nil {
		t.Fatal("opencode hooks.yaml should not be installed")
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

// pluginEntries parses the "plugin" array of a raw harness config into a
// set for assertion. A missing or empty array yields an empty set.
func pluginEntries(data []byte) map[string]bool {
	set := map[string]bool{}
	for _, p := range gjson.Parse(string(data)).Get("plugin").Array() {
		set[p.String()] = true
	}
	return set
}
