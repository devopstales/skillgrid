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
		".agents/opencode/skill.md":       "# agent",
		"docs/guide.md":                   "# guide",
		"git-hooks/pre-commit.js":         "# shim",
		"hooks/checkpoint-state.js":       "# impl",
		"hooks/opencode-session-start.sh": "#!/bin/sh\nexit 0\n",
		// Not whitelisted — must NOT be mirrored.
		"plugins/opencode/mnemonic.ts": "// plugin",
		"scripts/build.sh":             "# build",
		"README.md":                    "# readme",
		// Skipped at any depth.
		".git/HEAD":                 "ref: refs/heads/main",
		"node_modules/foo/index.js": "// dep",
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
		"hooks/opencode-session-start.sh",
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

// --- router bootstrap distribution (T6, 2026-11-17-skillgrid-host-plugins) ---

func writeBootstrapFixture(t *testing.T, repo, content string) {
	t.Helper()
	p := filepath.Join(repo, ".agents", "skills", "_shared", "agent-config", "router-bootstrap.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSyncRouterBootstrap(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	writeBootstrapFixture(t, repo, "router v1")
	cfg := &Config{RepoDir: repo, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := syncRouterBootstrap(cfg); err != nil {
		t.Fatalf("syncRouterBootstrap: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(home, ".skillgrid", "router-bootstrap.md"))
	if err != nil {
		t.Fatalf("bootstrap not installed: %v", err)
	}
	if string(b) != "router v1" {
		t.Errorf("bootstrap content = %q, want %q", b, "router v1")
	}
}

func TestSyncRouterBootstrapDryRunWritesNothing(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	writeBootstrapFixture(t, repo, "router v1")
	cfg := &Config{RepoDir: repo, RepoHome: filepath.Join(home, ".skillgrid"), DryRun: true}
	if err := syncRouterBootstrap(cfg); err != nil {
		t.Fatalf("syncRouterBootstrap dry-run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".skillgrid")); err == nil {
		t.Errorf("dry-run must not create ~/.skillgrid")
	}
}

func TestSyncRouterBootstrapIdempotentRerun(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	writeBootstrapFixture(t, repo, "router v1")
	cfg := &Config{RepoDir: repo, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := syncRouterBootstrap(cfg); err != nil {
		t.Fatalf("first syncRouterBootstrap: %v", err)
	}
	dst := filepath.Join(home, ".skillgrid", "router-bootstrap.md")
	fi1, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	// Unchanged re-run: the installed copy must not be rewritten
	// (byte-identical, stable mtime).
	if err := syncRouterBootstrap(cfg); err != nil {
		t.Fatalf("second syncRouterBootstrap: %v", err)
	}
	fi2, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !fi1.ModTime().Equal(fi2.ModTime()) {
		t.Errorf("unchanged re-run rewrote the installed copy (mtime %s -> %s)", fi1.ModTime(), fi2.ModTime())
	}
	if b, err := os.ReadFile(dst); err != nil || string(b) != "router v1" {
		t.Errorf("re-run changed content: %q", b)
	}
}

func TestSyncRouterBootstrapOverwritesChangedAsset(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	writeBootstrapFixture(t, repo, "router v1")
	cfg := &Config{RepoDir: repo, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := syncRouterBootstrap(cfg); err != nil {
		t.Fatalf("syncRouterBootstrap: %v", err)
	}
	dst := filepath.Join(home, ".skillgrid", "router-bootstrap.md")
	// Edited clone asset: the installed copy must follow the new content.
	writeBootstrapFixture(t, repo, "router v2")
	if err := syncRouterBootstrap(cfg); err != nil {
		t.Fatalf("syncRouterBootstrap after edit: %v", err)
	}
	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "router v2" {
		t.Errorf("edited asset not picked up: %q", b)
	}
}

func TestSyncRouterBootstrapMissingAsset(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	cfg := &Config{RepoDir: repo, RepoHome: filepath.Join(home, ".skillgrid")}
	// Older checkouts ship no bootstrap asset: skip, don't fail.
	if err := syncRouterBootstrap(cfg); err != nil {
		t.Fatalf("syncRouterBootstrap should skip a missing asset, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".skillgrid", "router-bootstrap.md")); err == nil {
		t.Error("no bootstrap file may be written when the asset is missing")
	}
}
