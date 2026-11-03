package service

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/project"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// skillgridIdentityFilename mirrors project.identityFilename so the regression
// test below can assert on the binding file without importing an unexported
// name. Keep in sync with internal/mnemonic/project/resolve.go.
const skillgridIdentityFilename = "skillgrid-mnemonic-identity.json"

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init", "--quiet")
	// Isolate the repo from ambient global git config (notably
	// core.hooksPath -> the skillgrid hook dir). A commit in this env would
	// otherwise run skillgrid's own pre-commit hook, which opens the project
	// and seeds the identity binding with the temp-dir basename BEFORE a
	// caller can add a remote origin — leaving a stale binding that breaks
	// identity resolution (see TestOpenForDirectorySeedsAliasFromLegacyID).
	empty := t.TempDir()
	runGit(t, dir, "config", "core.hooksPath", empty)
	runGit(t, dir, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "--allow-empty", "-m", "init")
}

// TestInitGitRepoDoesNotSeedIdentityBinding locks the hook-isolation regression:
// a bare init+commit must not leave a clone-private identity binding behind.
// Without core.hooksPath isolation, the ambient skillgrid pre-commit hook runs
// on the commit and writes the binding with the temp-dir basename, which then
// shadows the origin-derived identity for the whole repo.
func TestInitGitRepoDoesNotSeedIdentityBinding(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	if _, err := os.Stat(filepath.Join(dir, ".git", skillgridIdentityFilename)); !os.IsNotExist(err) {
		t.Fatalf("initGitRepo must not seed an identity binding; found %s (err=%v)", skillgridIdentityFilename, err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", full...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", full, err, out)
	}
}

// TestOpenForCWDRefusesAmbiguousParent guards interview D4: a multi-repo parent
// must never open/create a store under the directory-hash fallback id.
func TestOpenForCWDRefusesAmbiguousParent(t *testing.T) {
	parent := t.TempDir()
	for _, name := range []string{"alpha", "beta"} {
		child := filepath.Join(parent, name)
		if err := os.MkdirAll(child, 0o755); err != nil {
			t.Fatal(err)
		}
		initGitRepo(t, child)
	}

	dataDir := t.TempDir()
	svc := New(dataDir)

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(parent); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	h, cleanup, err := svc.OpenForCWD()
	if cleanup != nil {
		cleanup()
	}
	if err == nil {
		t.Fatalf("expected ambiguous abort, opened project %q", h.ProjectID())
	}
	if !errors.Is(err, project.ErrAmbiguousProject) {
		t.Fatalf("err=%v want errors.Is(..., ErrAmbiguousProject)", err)
	}

	entries, _ := os.ReadDir(dataDir)
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("no store file must be created under ambiguous parent; found %v", names)
	}
}

// TestOpenForDirectoryHonoursMNEMONIC_PROJECT recovers from ambiguity via override.
func TestOpenForDirectoryHonoursMNEMONIC_PROJECT(t *testing.T) {
	parent := t.TempDir()
	for _, name := range []string{"alpha", "beta"} {
		child := filepath.Join(parent, name)
		if err := os.MkdirAll(child, 0o755); err != nil {
			t.Fatal(err)
		}
		initGitRepo(t, child)
	}

	t.Setenv(project.EnvProjectOverride, "alpha")
	dataDir := t.TempDir()
	svc := New(dataDir)

	h, cleanup, err := svc.OpenForDirectory(parent)
	if err != nil {
		t.Fatalf("override should allow open: %v", err)
	}
	defer cleanup()
	if h.ProjectID() != "alpha" {
		t.Fatalf("project=%q want alpha", h.ProjectID())
	}
}

// TestOpenForDirectorySeedsAliasFromLegacyID covers silent SeedID→canonical
// merge on first identity bind (interview D6).
func TestOpenForDirectorySeedsAliasFromLegacyID(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	runGit(t, dir, "remote", "add", "origin", "git@github.com:acme/seed-alias.git")

	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	legacy := project.LegacyFallbackID(abs)
	dataDir := t.TempDir()

	legacyStore, err := store.Open(dataDir, legacy)
	if err != nil {
		t.Fatalf("seed legacy store: %v", err)
	}
	if err := legacyStore.Close(); err != nil {
		t.Fatal(err)
	}

	svc := New(dataDir)
	h, cleanup, err := svc.OpenForDirectory(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	if h.ProjectID() != "seed-alias" {
		t.Fatalf("project=%q want seed-alias", h.ProjectID())
	}

	var canonical string
	err = h.Store().DB.QueryRow(
		`SELECT canonical FROM project_aliases WHERE alias = ?`, legacy,
	).Scan(&canonical)
	if err != nil {
		t.Fatalf("alias row missing for %q: %v", legacy, err)
	}
	if canonical != "seed-alias" {
		t.Fatalf("canonical=%q want seed-alias", canonical)
	}
}
