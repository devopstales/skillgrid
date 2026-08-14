package plugins

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aiskillgrid/aiskillgrid/home"
)

func TestEnsurePluginEntry_CreatesAndDedupes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")
	if err := ensurePluginEntry(path, SuperpowersPluginSpec); err != nil {
		t.Fatal(err)
	}
	if err := ensurePluginEntry(path, SuperpowersPluginSpec); err != nil {
		t.Fatal(err)
	}
	// Different URL for same name should replace, not duplicate.
	alt := "superpowers@git+https://github.com/obra/superpowers.git#v5.0.0"
	if err := ensurePluginEntry(path, alt); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatal(err)
	}
	arr, ok := obj["plugin"].([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("plugin array = %#v", obj["plugin"])
	}
	if arr[0] != alt {
		t.Fatalf("got %v want %v", arr[0], alt)
	}
}

func TestInstallCursorPlugin_CopiesLocal(t *testing.T) {
	homeRoot := t.TempDir()
	checkout := t.TempDir()
	if err := os.MkdirAll(filepath.Join(checkout, ".cursor-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(checkout, ".cursor-plugin", "plugin.json"), []byte(`{"name":"superpowers"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(filepath.Join(checkout, ".git"), 0o755)
	_ = os.WriteFile(filepath.Join(checkout, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644)

	paths, warns, err := installCursorPlugin(homeRoot, checkout)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 {
		t.Fatalf("paths=%v", paths)
	}
	manifest := filepath.Join(homeRoot, ".cursor", "plugins", "local", "superpowers", ".cursor-plugin", "plugin.json")
	if _, err := os.Stat(manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(homeRoot, ".cursor", "plugins", "local", "superpowers", ".git")); !os.IsNotExist(err) {
		t.Fatalf("expected .git skipped, err=%v", err)
	}
	if len(warns) == 0 {
		t.Fatal("expected reload warning")
	}
}

func TestInstallOpenCodePlugin_ProjectScope(t *testing.T) {
	project := t.TempDir()
	opts := Options{
		Scope:      home.ScopeProject,
		ProjectDir: project,
		ConfigDir:  t.TempDir(),
		HomeRoot:   t.TempDir(),
	}
	paths, warns, err := installOpenCodePlugin(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("warns=%v", warns)
	}
	if len(paths) != 1 || paths[0] != filepath.Join(project, "opencode.json") {
		t.Fatalf("paths=%v", paths)
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatal(err)
	}
	arr := obj["plugin"].([]any)
	if len(arr) != 1 || arr[0] != SuperpowersPluginSpec {
		t.Fatalf("plugin=%v", arr)
	}
}

func TestPluginName(t *testing.T) {
	if got := pluginName(SuperpowersPluginSpec); got != "superpowers" {
		t.Fatalf("got %q", got)
	}
}
