package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAvailableAgents(t *testing.T) {
	got := AvailableAgents()
	if len(got) != 3 {
		t.Fatalf("want 3 agents, got %d", len(got))
	}
	want := map[string]string{
		"opencode": "opencode-ai",
		"kilo":     "@kilocode/cli",
		"cursor":   "",
	}
	for _, a := range got {
		if want[a.Key] != a.NPM {
			t.Errorf("agent %q npm=%q, want %q", a.Key, a.NPM, want[a.Key])
		}
	}
}

func TestGlobalTools(t *testing.T) {
	got := GlobalTools()
	if len(got) != 2 {
		t.Fatalf("want 2 global tools, got %d", len(got))
	}
	want := map[string]string{
		"skills":   "skills",
		"openspec": "@fission-ai/openspec@latest",
	}
	for _, tl := range got {
		if want[tl.Name] != tl.NPM {
			t.Errorf("tool %q npm=%q, want %q", tl.Name, tl.NPM, want[tl.Name])
		}
	}
}

func TestEnsureHomeStruct(t *testing.T) {
	home := t.TempDir()
	cfg := Config{
		HomeDir:   home,
		RepoHome:  home + "/.skillgrid",
		RepoDir:   home + "/.skillgrid/repos/skillgrid",
		AgentsDir: home + "/.agents",
	}
	if err := ensureHomeStruct(&cfg); err != nil {
		t.Fatalf("ensureHomeStruct: %v", err)
	}
	for _, d := range []string{cfg.RepoHome, cfg.RepoDir} {
		if info, err := os.Stat(d); err != nil || !info.IsDir() {
			t.Errorf("missing expected dir %s", d)
		}
	}
}

func TestCopyAll(t *testing.T) {
	src := t.TempDir()
	sub := filepath.Join(src, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("beta"), 0o644); err != nil {
		t.Fatal(err)
	}

	dst := t.TempDir()
	if err := copyAll(src, dst); err != nil {
		t.Fatalf("copyAll: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(dst, "a.txt")); err != nil || string(b) != "alpha" {
		t.Errorf("a.txt = %q", b)
	}
	if b, err := os.ReadFile(filepath.Join(dst, "sub", "b.txt")); err != nil || string(b) != "beta" {
		t.Errorf("b.txt = %q", b)
	}
}

func TestVerboseOut(t *testing.T) {
	// No panic is the success criterion; suppressed when flags are off.
	cfg := Config{}
	VerboseOut(&cfg, "hidden unless verbose or dry-run")
}
