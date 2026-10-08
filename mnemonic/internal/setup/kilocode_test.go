package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetupKilo_InstallsPlugins(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	home := t.TempDir()
	if err := SetupKiloCode(home, repoRoot, nil, nil, false); err != nil {
		t.Fatalf("SetupKiloCode: %v", err)
	}
	// The 5 native TS plugins are copied from plugins/opencode/ into kilo.
	for _, base := range nativePluginBases {
		dst := filepath.Join(home, ".config", "kilo", "plugin", base+".ts")
		if _, err := os.Stat(dst); err != nil {
			t.Fatalf("missing kilo plugin %s: %v", base, err)
		}
	}
	cfgData, err := os.ReadFile(AgentConfigPath(home, "kilo"))
	if err != nil {
		t.Fatalf("read kilo config: %v", err)
	}
	got := pluginEntries(cfgData)
	for _, base := range nativePluginBases {
		if !got["./plugin/"+base+".ts"] {
			t.Fatalf("kilo config missing plugin entry %s: %s", base, cfgData)
		}
	}
	// The retired hook file and checkpoint plugin are no longer installed.
	if _, err := os.Stat(filepath.Join(home, ".config", "kilo", "hook", "hooks.yaml")); err == nil {
		t.Fatal("kilo hooks.yaml should not be installed")
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "kilo", "plugin", "skillgrid-checkpoint.ts")); err == nil {
		t.Fatal("kilo skillgrid-checkpoint.ts should not be installed")
	}
	if got["opencode-yaml-hooks"] {
		t.Fatalf("kilo config should not register opencode-yaml-hooks: %s", cfgData)
	}

	// Dry-run must not write the plugins.
	homeDry := t.TempDir()
	cfgPath := AgentConfigPath(homeDry, "kilo")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SetupKiloCode(homeDry, repoRoot, nil, nil, true); err != nil {
		t.Fatalf("SetupKiloCode dry-run: %v", err)
	}
	for _, base := range nativePluginBases {
		if _, err := os.Stat(filepath.Join(homeDry, ".config", "kilo", "plugin", base+".ts")); err == nil {
			t.Fatalf("dry-run must not write kilo plugin %s", base)
		}
	}
}

func TestSetupKilo_DropsRetiredPlugins(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	home := t.TempDir()
	cfgPath := AgentConfigPath(home, "kilo")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	initial := `{"plugin":["/tmp/old/mnemonic.ts","opencode-command-hooks","~/.config/kilo/hook/hooks.yaml","opencode-yaml-hooks","./plugin/skillgrid-checkpoint.ts","keep-me"]}` + "\n"
	if err := os.WriteFile(cfgPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SetupKiloCode(home, repoRoot, nil, nil, false); err != nil {
		t.Fatalf("SetupKiloCode: %v", err)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	got := pluginEntries(data)
	for _, retired := range []string{"/tmp/old/mnemonic.ts", "opencode-command-hooks", "~/.config/kilo/hook/hooks.yaml", "opencode-yaml-hooks", "./plugin/skillgrid-checkpoint.ts"} {
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
