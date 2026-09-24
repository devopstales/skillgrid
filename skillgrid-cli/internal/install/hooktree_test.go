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

func TestSyncMirrorDirs(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	writeMirrorFixture(t, repo, map[string]string{
		// Whitelisted — should be mirrored.
		".agents/opencode/skill.md":    "# agent",
		"docs/guide.md":                "# guide",
		"git-hooks/pre-commit.js":      "# shim",
		"hooks/checkpoint-state.js":    "# impl",
		// Not whitelisted — must NOT be mirrored.
		"plugins/opencode/mnemonic.ts": "// plugin",
		"scripts/build.sh":             "# build",
		"README.md":                    "# readme",
		// Skipped at any depth.
		".git/HEAD":                    "ref: refs/heads/main",
		"node_modules/foo/index.js":    "// dep",
	})

	cfg := &Config{RepoDir: repo, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := syncMirrorDirs(cfg); err != nil {
		t.Fatalf("syncMirrorDirs: %v", err)
	}
	for _, want := range []string{
		".agents/opencode/skill.md",
		"docs/guide.md",
		"git-hooks/pre-commit.js",
		"hooks/checkpoint-state.js",
	} {
		if b, err := os.ReadFile(filepath.Join(home, ".skillgrid", want)); err != nil {
			t.Errorf("mirrored %s missing: %v", want, err)
		} else if len(b) == 0 {
			t.Errorf("mirrored %s empty", want)
		}
	}
	for _, absent := range []string{
		"plugins", "scripts", "README.md",
		".git", "node_modules",
	} {
		if _, err := os.Stat(filepath.Join(home, ".skillgrid", absent)); err == nil {
			t.Errorf("%s must not be mirrored", absent)
		}
	}
}

func TestSyncMirrorDirsSkipsMissingSource(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	writeMirrorFixture(t, repo, map[string]string{
		"docs/guide.md": "# guide",
	})

	cfg := &Config{RepoDir: repo, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := syncMirrorDirs(cfg); err != nil {
		t.Fatalf("syncMirrorDirs: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".skillgrid", "docs", "guide.md")); err != nil {
		t.Errorf("docs/guide.md should be mirrored: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".skillgrid", ".agents")); err == nil {
		t.Errorf(".agents/ should not be created when source is missing")
	}
	if _, err := os.Stat(filepath.Join(home, ".skillgrid", "git-hooks")); err == nil {
		t.Errorf("git-hooks/ should not be created when source is missing")
	}
}

func TestSyncMirrorDirsDryRunWritesNothing(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	writeMirrorFixture(t, repo, map[string]string{
		"docs/guide.md": "# guide",
	})
	cfg := &Config{RepoDir: repo, RepoHome: filepath.Join(home, ".skillgrid"), DryRun: true}
	if err := syncMirrorDirs(cfg); err != nil {
		t.Fatalf("syncMirrorDirs dry-run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".skillgrid")); err == nil {
		t.Errorf("dry-run must not create ~/.skillgrid")
	}
}

func TestSyncMirrorDirsReplacesStaleDestination(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	writeMirrorFixture(t, repo, map[string]string{
		"docs/guide.md": "new",
	})
	stale := filepath.Join(home, ".skillgrid", "docs", "stale.md")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{RepoDir: repo, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := syncMirrorDirs(cfg); err != nil {
		t.Fatalf("syncMirrorDirs: %v", err)
	}
	if _, err := os.Stat(stale); err == nil {
		t.Errorf("stale file should be replaced on re-mirror")
	}
	if _, err := os.Stat(filepath.Join(home, ".skillgrid", "docs", "guide.md")); err != nil {
		t.Errorf("guide.md should be mirrored: %v", err)
	}
}

func TestCopyAsset(t *testing.T) {
	repo := t.TempDir()
	dst := filepath.Join(t.TempDir(), "out", "asset.tsx")
	logoPath := filepath.Join(repo, "plugins", "opencode", "skillgrid-logo.tsx")
	if err := os.MkdirAll(filepath.Dir(logoPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logoPath, []byte("// logo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyAsset(repo, "plugins/opencode/skillgrid-logo.tsx", dst, false); err != nil {
		t.Fatalf("copyAsset: %v", err)
	}
	if b, err := os.ReadFile(dst); err != nil || string(b) != "// logo" {
		t.Errorf("copyAsset: got %q, %v", b, err)
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
