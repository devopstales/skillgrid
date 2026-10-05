package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestSetupKilo_CheckpointPlugin(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	home := t.TempDir()
	if err := SetupKiloCode(home, repoRoot, nil, nil, false); err != nil {
		t.Fatalf("SetupKiloCode: %v", err)
	}
	dst := filepath.Join(home, ".config", "kilo", "plugin", "skillgrid-checkpoint.ts")
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read checkpoint plugin: %v", err)
	}
	if !strings.Contains(string(data), checkpointPluginMarker) {
		t.Fatalf("checkpoint plugin missing marker %q", checkpointPluginMarker)
	}
	cfgData, err := os.ReadFile(AgentConfigPath(home, "kilo"))
	if err != nil {
		t.Fatalf("read kilo config: %v", err)
	}
	if !strings.Contains(string(cfgData), "./plugin/skillgrid-checkpoint.ts") {
		t.Fatalf("kilo config missing checkpoint plugin entry: %s", cfgData)
	}

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
	if _, err := os.Stat(filepath.Join(homeDry, ".config", "kilo", "plugin", "skillgrid-checkpoint.ts")); err == nil {
		t.Fatal("dry-run must not write checkpoint plugin")
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
	initial := `{"plugin":["/tmp/old/mnemonic.ts","opencode-command-hooks","~/.config/kilo/hook/hooks.yaml","keep-me"]}` + "\n"
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
	got := map[string]bool{}
	for _, p := range gjson.Parse(string(data)).Get("plugin").Array() {
		got[p.String()] = true
	}
	for _, retired := range []string{"/tmp/old/mnemonic.ts", "opencode-command-hooks", "~/.config/kilo/hook/hooks.yaml"} {
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
