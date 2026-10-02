package install

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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

func TestShippedPersonaCatalog(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))

	names := []string{"debugger", "implementer", "researcher", "reviewer", "verifier"}
	readonly := map[string]bool{
		"implementer": false,
		"debugger":    false,
		"reviewer":    true,
		"researcher":  true,
		"verifier":    true,
	}

	for _, name := range names {
		cursorPath := filepath.Join(repoRoot, "agents", "cursor", name+".md")
		fm := readFrontmatter(t, cursorPath)
		if fm["name"] != name {
			t.Errorf("%s name = %v, want %s", cursorPath, fm["name"], name)
		}
		if desc, _ := fm["description"].(string); desc == "" {
			t.Errorf("%s description is empty", cursorPath)
		}
		if fm["model"] != "inherit" {
			t.Errorf("%s model = %v, want inherit", cursorPath, fm["model"])
		}
		if _, ok := fm["readonly"]; !ok {
			t.Errorf("%s missing readonly", cursorPath)
		} else if fm["readonly"] != readonly[name] {
			t.Errorf("%s readonly = %v, want %v", cursorPath, fm["readonly"], readonly[name])
		}

		opencodePath := filepath.Join(repoRoot, "agents", "opencode", name+".md")
		ofm := readFrontmatter(t, opencodePath)
		if _, ok := ofm["name"]; ok {
			t.Errorf("%s must not set name (filename is the agent name)", opencodePath)
		}
		if _, ok := ofm["model"]; ok {
			t.Errorf("%s must not pin a model", opencodePath)
		}
		if desc, _ := ofm["description"].(string); desc == "" {
			t.Errorf("%s description is empty", opencodePath)
		}
		if ofm["mode"] != "subagent" {
			t.Errorf("%s mode = %v, want subagent", opencodePath, ofm["mode"])
		}
		perm, _ := ofm["permission"].(map[string]any)
		wantEdit := "deny"
		if !readonly[name] {
			wantEdit = "allow"
		}
		if perm["edit"] != wantEdit {
			t.Errorf("%s permission.edit = %v, want %s", opencodePath, perm["edit"], wantEdit)
		}
	}
}

func readFrontmatter(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(data)
	if !strings.HasPrefix(s, "---\n") {
		t.Fatalf("%s: frontmatter must start with ---", path)
	}
	rest := strings.TrimPrefix(s, "---\n")
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		t.Fatalf("%s: frontmatter not closed", path)
	}
	var fm map[string]any
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		t.Fatalf("%s: parse frontmatter: %v", path, err)
	}
	return fm
}

func TestInstallPersonasUnknownAgentNoop(t *testing.T) {
	home := t.TempDir()
	repoDir := t.TempDir()
	cfg := &Config{HomeDir: home, RepoDir: repoDir}
	if err := installPersonas(cfg, "does-not-exist"); err != nil {
		t.Fatalf("unknown agent should be a no-op, got %v", err)
	}
}
