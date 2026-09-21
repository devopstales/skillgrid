package install

import (
	"os"
	"path/filepath"
	"testing"
)

func writeMirrorFixture(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSyncFullTree(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	writeMirrorFixture(t, repo, map[string]string{
		"hooks/checkpoint-state.sh":    "# impl",
		"git-hooks/pre-commit":         "# shim",
		"plugins/opencode/mnemonic.ts": "// plugin",
		"README.md":                    "# readme",
		"docs/guide.md":                "# guide",
		// Worktree state: never mirrored.
		".git/HEAD":                    "ref: refs/heads/main",
		".skillgrid/specs/x/brief.md":  "# spec",
		"dist/skillgrid":               "binary",
		"node_modules/foo/index.js":    "// dep",
		"sub/nested/node_modules/b.js": "// nested dep",
	})

	cfg := &Config{RepoDir: repo, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := syncFullTree(cfg); err != nil {
		t.Fatalf("syncFullTree: %v", err)
	}
	for _, want := range []string{
		"hooks/checkpoint-state.sh",
		"git-hooks/pre-commit",
		"plugins/opencode/mnemonic.ts",
		"README.md",
		"docs/guide.md",
	} {
		if b, err := os.ReadFile(filepath.Join(home, ".skillgrid", want)); err != nil {
			t.Errorf("mirrored %s missing: %v", want, err)
		} else if len(b) == 0 {
			t.Errorf("mirrored %s empty", want)
		}
	}
	for _, absent := range []string{
		".git", ".skillgrid", "dist", "node_modules", "sub/nested/node_modules",
	} {
		if _, err := os.Stat(filepath.Join(home, ".skillgrid", absent)); err == nil {
			t.Errorf("%s must not be mirrored", absent)
		}
	}
	// The nested non-excluded sibling survives the node_modules skip.
	if _, err := os.Stat(filepath.Join(home, ".skillgrid", "sub")); err != nil {
		t.Errorf("sub/ should be mirrored: %v", err)
	}
}

func TestSyncFullTreeProtectsOperationalDirs(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	writeMirrorFixture(t, repo, map[string]string{
		"hooks/a.sh": "# impl",
	})
	protected := map[string]string{
		"mnemonic/data.sqlite": "live db",
		"repos/old/file.txt":   "checkout",
		"backup/x":             "backup",
		"bin/skillgrid":        "binary",
		"tmp/y":                "tmp",
		"logs/z":               "log",
		"config.d/custom.yaml": "user config",
	}
	for rel, content := range protected {
		p := filepath.Join(home, ".skillgrid", rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := &Config{RepoDir: repo, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := syncFullTree(cfg); err != nil {
		t.Fatalf("syncFullTree: %v", err)
	}
	for rel, content := range protected {
		if b, err := os.ReadFile(filepath.Join(home, ".skillgrid", rel)); err != nil || string(b) != content {
			t.Errorf("protected %s disturbed: %q, %v", rel, b, err)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".skillgrid", "hooks", "a.sh")); err != nil {
		t.Errorf("hooks/a.sh should be mirrored: %v", err)
	}
}

func TestSyncFullTreeDryRunWritesNothing(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	writeMirrorFixture(t, repo, map[string]string{
		"hooks/a.sh": "# impl",
	})
	cfg := &Config{RepoDir: repo, RepoHome: filepath.Join(home, ".skillgrid"), DryRun: true}
	if err := syncFullTree(cfg); err != nil {
		t.Fatalf("syncFullTree dry-run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".skillgrid")); err == nil {
		t.Errorf("dry-run must not create ~/.skillgrid")
	}
}

func TestSyncFullTreeReplacesStaleDestination(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	writeMirrorFixture(t, repo, map[string]string{
		"hooks/a.sh": "new",
	})
	stale := filepath.Join(home, ".skillgrid", "hooks", "stale.sh")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{RepoDir: repo, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := syncFullTree(cfg); err != nil {
		t.Fatalf("syncFullTree: %v", err)
	}
	if _, err := os.Stat(stale); err == nil {
		t.Errorf("stale file should be replaced on re-mirror")
	}
}

func TestPlanMirror(t *testing.T) {
	repo := t.TempDir()
	writeMirrorFixture(t, repo, map[string]string{
		"hooks/a.sh":       "# impl",
		".skillgrid/x":     "# state",
		"mnemonic/dummy":   "# would-be operational",
	})
	ops, err := planMirror(repo)
	if err != nil {
		t.Fatalf("planMirror: %v", err)
	}
	got := map[string]string{}
	for _, op := range ops {
		got[op.name] = op.action
	}
	if got["hooks"] != "copy" {
		t.Errorf("hooks: got %q, want copy", got["hooks"])
	}
	if got[".skillgrid"] != "skip-source" {
		t.Errorf(".skillgrid: got %q, want skip-source", got[".skillgrid"])
	}
	if got["mnemonic"] != "protect-dst" {
		t.Errorf("mnemonic: got %q, want protect-dst", got["mnemonic"])
	}
}

func TestResolveAssetPath(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()

	// No staged copy → repo fallback.
	got := resolveAssetPath(home, repo, "plugins/opencode/mnemonic.ts")
	if got != filepath.Join(repo, "plugins/opencode/mnemonic.ts") {
		t.Errorf("fallback: got %q", got)
	}

	// Staged copy present → staged wins.
	staged := filepath.Join(home, ".skillgrid", "plugins", "opencode", "mnemonic.ts")
	if err := os.MkdirAll(filepath.Dir(staged), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staged, []byte("// staged"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := resolveAssetPath(home, repo, "plugins/opencode/mnemonic.ts"); got != staged {
		t.Errorf("staged-first: got %q, want %q", got, staged)
	}

	// Non-plugin relatives never consult the staged tree.
	if got := resolveAssetPath(home, repo, "config.d/mcp.yaml"); got != filepath.Join(repo, "config.d/mcp.yaml") {
		t.Errorf("non-plugin: got %q", got)
	}
}

func TestWireGitHooksDryRun(t *testing.T) {
	home := t.TempDir()
	staged := filepath.Join(home, ".skillgrid", "git-hooks")
	if err := os.MkdirAll(staged, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{RepoHome: filepath.Join(home, ".skillgrid"), DryRun: true}
	if err := wireGitHooks(cfg); err != nil {
		t.Errorf("wireGitHooks dry-run: %v", err)
	}
}

func TestWireGitHooksMissingTarget(t *testing.T) {
	home := t.TempDir()
	cfg := &Config{RepoHome: filepath.Join(home, ".skillgrid")}
	if err := wireGitHooks(cfg); err == nil {
		t.Errorf("wireGitHooks should fail when the staged git-hooks dir is absent")
	}
}
