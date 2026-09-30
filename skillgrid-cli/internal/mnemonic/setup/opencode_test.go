package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tidwall/gjson"
)

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
