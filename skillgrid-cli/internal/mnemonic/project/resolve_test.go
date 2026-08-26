package project_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"skillgrid-cli/internal/mnemonic/project"
)

func initGitRepo(t *testing.T, dir, origin string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	if origin != "" {
		runGit(t, dir, "remote", "add", "origin", origin)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "init")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

func TestResolveFromGitRemote(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir, "https://github.com/devopstales/skillgrid.git")

	got, err := project.Resolve(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := "github.com/devopstales/skillgrid"
	if got != want {
		t.Fatalf("Resolve() = %q, want %q", got, want)
	}

	// SSH form should normalize to the same id.
	dir2 := t.TempDir()
	initGitRepo(t, dir2, "git@github.com:devopstales/skillgrid.git")
	got2, err := project.Resolve(dir2)
	if err != nil {
		t.Fatal(err)
	}
	if got2 != want {
		t.Fatalf("Resolve() ssh = %q, want %q", got2, want)
	}
}

func TestResolveFromConfig(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir, "https://github.com/devopstales/skillgrid.git")

	cfgDir := filepath.Join(dir, ".skillgrid")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(`{"project":"my-custom-project"}`), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := project.Resolve(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != "my-custom-project" {
		t.Fatalf("Resolve() = %q, want %q", got, "my-custom-project")
	}
}

func TestResolveFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := project.Resolve(dir)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Base(dir)
	if len(got) <= len(base)+1 {
		t.Fatalf("Resolve() = %q, want %q-<hash>", got, base)
	}
	if got[:len(base)+1] != base+"-" {
		t.Fatalf("Resolve() = %q, want prefix %q-", got, base)
	}
	hashPart := got[len(base)+1:]
	if len(hashPart) != 8 {
		t.Fatalf("hash suffix = %q (len %d), want 8 hex chars", hashPart, len(hashPart))
	}
	for _, c := range hashPart {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Fatalf("hash suffix %q is not lowercase hex", hashPart)
		}
	}

	got2, err := project.Resolve(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got2 != got {
		t.Fatalf("Resolve() not stable: %q vs %q", got, got2)
	}
}
