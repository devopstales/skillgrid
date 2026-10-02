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
