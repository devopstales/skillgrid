package home

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureLayoutAndConfig(t *testing.T) {
	root := t.TempDir()
	t.Setenv(EnvHome, root)
	got, err := Root()
	if err != nil || got != root {
		t.Fatalf("Root=%q err=%v", got, err)
	}
	p := Resolve(root)
	if err := EnsureLayout(p); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{p.ToolsDir, p.DepsDir, p.DepsBinDir, p.NpmDir, p.NpmBinDir, p.NpmCacheDir, p.LogsDir, p.SessionsDir, p.MemoriesDir} {
		if st, err := os.Stat(d); err != nil || !st.IsDir() {
			t.Fatalf("missing dir %s: %v", d, err)
		}
	}
	if p.DepsBinDir != filepath.Join(root, "dependencies", "bin") {
		t.Fatalf("DepsBinDir=%s", p.DepsBinDir)
	}
	if p.NpmCacheDir != filepath.Join(root, "npm", "cache") {
		t.Fatalf("NpmCacheDir=%s", p.NpmCacheDir)
	}
	cfg, err := LoadConfig(p.ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RepoURL == "" {
		t.Fatal("empty repo url")
	}
	st := State{Scope: "global", Agents: []string{"cursor"}, WrittenPaths: map[string][]string{"cursor": {"a"}}}
	if err := SaveState(p.StateFile, st); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadState(p.StateFile)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Agents[0] != "cursor" {
		t.Fatalf("%+v", loaded)
	}
	if err := AppendLog(p.LogsDir, "hello"); err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(p.LogsDir, "*.log"))
	if len(matches) == 0 {
		t.Fatal("no log")
	}
}
