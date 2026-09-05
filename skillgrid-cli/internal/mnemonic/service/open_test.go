package service

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/project"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init", "--quiet")
	runGit(t, dir, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "--allow-empty", "-m", "init")
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
