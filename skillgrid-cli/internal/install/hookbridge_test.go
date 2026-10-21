package install

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/tidwall/gjson"
)

func readSettings(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func settingsPathFor(t *testing.T, home string) string {
	t.Helper()
	p, err := piAgentSettingsPath(home)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func assertPiHooksBlock(t *testing.T, data []byte, hooksDir string) {
	t.Helper()
	hooks := gjson.GetBytes(data, "hooks")
	if !hooks.IsObject() {
		t.Fatalf("hooks is not an object: %s", data)
	}
	for _, event := range []string{"SessionStart", "Stop", "PostToolUse", "PreToolUse"} {
		if !hooks.Get(event).IsArray() {
			t.Fatalf("hooks.%s missing or not an array: %s", event, data)
		}
	}
	expect := func(key, want string) {
		if got := hooks.Get(key).String(); got != want {
			t.Fatalf("hooks.%s = %q, want %q", key, got, want)
		}
	}
	expect("SessionStart.0.hooks.0.command", "node "+filepath.Join(hooksDir, "session-start.js"))
	expect("Stop.0.hooks.0.command", "node "+filepath.Join(hooksDir, "gate-stop.js"))
	expect("Stop.0.hooks.1.command", "node "+filepath.Join(hooksDir, "stop-tests.js"))
	expect("PostToolUse.0.hooks.0.command", "node "+filepath.Join(hooksDir, "tool-call-capture.js"))
	if got := hooks.Get("PreToolUse.0.matcher").String(); got != "bash" {
		t.Fatalf("PreToolUse matcher = %q, want bash", got)
	}
	expect("PreToolUse.0.hooks.0.if", "Bash(git *)")
	expect("PreToolUse.0.hooks.0.command", "node "+filepath.Join(hooksDir, "ticketing-guard.js"))
	for i := 0; i < 4; i++ {
		expectKey := "type"
		if g := hooks.Get("SessionStart.0.hooks.0." + expectKey).String(); g != "command" {
			t.Fatalf("SessionStart hook type = %q, want command", g)
		}
	}
}

func TestInstallPiHookBridgeDryRunWritesNothing(t *testing.T) {
	home := t.TempDir()
	cfg := Config{HomeDir: home, Agents: []string{"skill-pi"}, DryRun: true}
	if err := installPiHookBridge(&cfg); err != nil {
		t.Fatalf("installPiHookBridge dry-run: %v", err)
	}
	if _, err := os.Stat(settingsPathFor(t, home)); err == nil {
		t.Fatal("dry-run must not create settings.json")
	}
}

func TestInstallPiHookBridgeRealInstall(t *testing.T) {
	home := t.TempDir()
	settings := settingsPathFor(t, home)
	if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
		t.Fatal(err)
	}
	preExisting := `{"defaultProvider":"x"}`
	if err := os.WriteFile(settings, []byte(preExisting), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{HomeDir: home, Agents: []string{"skill-pi"}}
	if err := installPiHookBridge(&cfg); err != nil {
		t.Fatalf("installPiHookBridge: %v", err)
	}
	data := readSettings(t, settings)
	assertPiHooksBlock(t, data, filepath.Join(home, ".skillgrid", "hooks"))
	if got := gjson.GetBytes(data, "defaultProvider").String(); got != "x" {
		t.Fatalf("pre-existing key lost: defaultProvider = %q", got)
	}
}

func TestInstallPiHookBridgeIdempotent(t *testing.T) {
	home := t.TempDir()
	cfg := Config{HomeDir: home, Agents: []string{"skill-pi"}}
	if err := installPiHookBridge(&cfg); err != nil {
		t.Fatalf("first run: %v", err)
	}
	first := readSettings(t, settingsPathFor(t, home))
	if err := installPiHookBridge(&cfg); err != nil {
		t.Fatalf("second run: %v", err)
	}
	second := readSettings(t, settingsPathFor(t, home))
	if !bytes.Equal(first, second) {
		t.Fatalf("re-run changed bytes:\nfirst:  %s\nsecond: %s", first, second)
	}
}

func TestInstallPiHookBridgeReplacesExistingHooksKey(t *testing.T) {
	home := t.TempDir()
	settings := settingsPathFor(t, home)
	if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte(`{"defaultProvider":"x","hooks":{"Custom":[{"hooks":[{"type":"command","command":"custom.js"}]}]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{HomeDir: home, Agents: []string{"skill-pi"}}
	if err := installPiHookBridge(&cfg); err != nil {
		t.Fatalf("installPiHookBridge: %v", err)
	}
	data := readSettings(t, settings)
	assertPiHooksBlock(t, data, filepath.Join(home, ".skillgrid", "hooks"))
	if gjson.GetBytes(data, "hooks.Custom").Exists() {
		t.Fatalf("foreign hooks key survived: %s", data)
	}
	if got := gjson.GetBytes(data, "defaultProvider").String(); got != "x" {
		t.Fatalf("pre-existing key lost: defaultProvider = %q", got)
	}
}

func TestInstallPiHookBridgeHonorsAgentDirEnv(t *testing.T) {
	home := t.TempDir()
	custom := filepath.Join(home, "custom-agent-dir")
	t.Setenv("SKILL_PI_AGENT_DIR", custom)
	cfg := Config{HomeDir: home, Agents: []string{"skill-pi"}}
	if err := installPiHookBridge(&cfg); err != nil {
		t.Fatalf("installPiHookBridge: %v", err)
	}
	data := readSettings(t, filepath.Join(custom, "settings.json"))
	assertPiHooksBlock(t, data, filepath.Join(home, ".skillgrid", "hooks"))
}
