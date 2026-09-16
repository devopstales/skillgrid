package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersonaTargetDir(t *testing.T) {
	home := "/home/user"
	cases := []struct {
		key     string
		wantDir string
		wantOK  bool
	}{
		{"opencode", filepath.Join(home, ".config", "opencode", "agents"), true},
		{"kilo", filepath.Join(home, ".config", "kilo", "agent"), true},
		{"cursor", filepath.Join(home, ".cursor", "agents"), true},
		{"unknown", "", false},
	}
	for _, c := range cases {
		dir, ok := personaTargetDir(home, c.key)
		if ok != c.wantOK || dir != c.wantDir {
			t.Errorf("personaTargetDir(%q) = (%q, %v), want (%q, %v)",
				c.key, dir, ok, c.wantDir, c.wantOK)
		}
	}
}

func TestCopyFlatFlattensTopLevel(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "explorer.md"), []byte("e"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "oracle.md"), []byte("o"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A subdirectory must be skipped (not recursed into).
	sub := filepath.Join(src, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "deep.md"), []byte("d"), 0o644); err != nil {
		t.Fatal(err)
	}

	dst := t.TempDir()
	if err := copyFlat(src, dst, false); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(dst, "explorer.md")); err != nil || string(b) != "e" {
		t.Errorf("explorer.md = %q, %v", b, err)
	}
	if b, err := os.ReadFile(filepath.Join(dst, "oracle.md")); err != nil || string(b) != "o" {
		t.Errorf("oracle.md = %q, %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(dst, "nested")); err == nil {
		t.Errorf("subdirectory should be skipped, but nested/ was copied")
	}
}

func TestCopyFlatDryRunWritesNothing(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "a.md"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	if err := copyFlat(src, dst, true); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(dst); len(entries) != 0 {
		t.Errorf("dry-run should write nothing, got %d entries", len(entries))
	}
}

func TestInstallPersonasMissingSourceIsNoop(t *testing.T) {
	home := t.TempDir()
	repoDir := t.TempDir() // no agents/ dir inside
	cfg := &Config{HomeDir: home, RepoDir: repoDir}
	if err := installPersonas(cfg, "opencode"); err != nil {
		t.Fatalf("missing source should be a no-op, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "agents")); err == nil {
		t.Errorf("no persona dir should be created when source is missing")
	}
}

func TestInstallPersonasCopiesFromSyncedRepo(t *testing.T) {
	home := t.TempDir()
	repoDir := t.TempDir()
	src := filepath.Join(repoDir, "agents", "opencode")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "explorer.md"), []byte("e"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{HomeDir: home, RepoDir: repoDir}
	if err := installPersonas(cfg, "opencode"); err != nil {
		t.Fatal(err)
	}
	got := filepath.Join(home, ".config", "opencode", "agents", "explorer.md")
	if b, err := os.ReadFile(got); err != nil || string(b) != "e" {
		t.Errorf("persona not installed at %s: %q, %v", got, b, err)
	}
}

func TestInstallPersonasUnknownAgentNoop(t *testing.T) {
	home := t.TempDir()
	repoDir := t.TempDir()
	cfg := &Config{HomeDir: home, RepoDir: repoDir}
	if err := installPersonas(cfg, "does-not-exist"); err != nil {
		t.Fatalf("unknown agent should be a no-op, got %v", err)
	}
}
