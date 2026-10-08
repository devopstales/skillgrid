package setup

import (
	"os"
	"path/filepath"
	"testing"
)

// S19: FindRepoRoot returns the repo root when the new plugin files exist.
// A subdirectory must resolve to the root via the upward walk.
func TestFindRepoRoot_FindsRootViaNewPlugins(t *testing.T) {
	repoRoot := t.TempDir()
	// Only the new opencode plugin marker needs to be present for the walk
	// to recognize a skillgrid repo root.
	if err := os.MkdirAll(filepath.Join(repoRoot, "plugins", "opencode"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, base := range nativePluginBases {
		if err := os.WriteFile(filepath.Join(repoRoot, "plugins", "opencode", base+".ts"), []byte("// marker\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Pin HOME so the user-home synced-repo escape cannot mask the walk.
	home := t.TempDir()
	t.Setenv("HOME", home)

	sub := filepath.Join(repoRoot, "mnemonic", "internal", "setup")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got := FindRepoRoot(sub)
	if got == "" {
		t.Fatal("FindRepoRoot returned empty; want the repo root")
	}
	if got != filepath.Clean(repoRoot) {
		t.Fatalf("FindRepoRoot = %q, want %q", got, filepath.Clean(repoRoot))
	}
}

// S20: FindRepoRoot fails (returns "") when none of the new plugin files
// exist anywhere up the tree and the user-home synced-repo fallback is absent.
func TestFindRepoRoot_FailsWhenNoNewPluginFiles(t *testing.T) {
	empty := t.TempDir() // contains none of the 5 opencode plugins
	home := t.TempDir()  // no ~/.skillgrid/repos/skillgit marker
	t.Setenv("HOME", home)

	// From a subdirectory of the empty tree: the walk climbs to the volume
	// root without a match, and the home fallback finds no synced repo.
	sub := filepath.Join(empty, "a", "b", "c")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := FindRepoRoot(sub); got != "" {
		t.Fatalf("FindRepoRoot = %q, want empty (no new plugin files present)", got)
	}
}
