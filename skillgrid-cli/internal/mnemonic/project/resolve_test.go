package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// git helpers — we shell out to real git so the tests exercise the full code
// path including the binding round trip. Tests in this file assume a
// sufficiently recent git (any version that supports `--path-format=absolute`).

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	remoteCmd(t, dir, "init", "--quiet")
	remoteCmd(t, dir, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "--allow-empty", "-m", "init")
}

// remoteCmd runs a git subcommand with explicit directory scoping.
func remoteCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", full...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", full, err, out)
	}
}

func TestIdentityStableAcrossRemoteChange(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	remoteA := "git@github.com:acme/project-a.git"
	remoteB := "git@github.com:acme/project-b.git"

	// First resolve creates the binding seeded from remote A.
	remoteCmd(t, dir, "remote", "add", "origin", remoteA)
	if got, err := resolveInDir(dir); err != nil || got.ID != "project-a" || got.Source != SourceIdentity {
		t.Fatalf("first resolve: id=%q src=%q err=%v", got.ID, got.Source, err)
	}

	// Remote change must not alter the resolved identity.
	remoteCmd(t, dir, "remote", "set-url", "origin", remoteB)
	if got, err := resolveInDir(dir); err != nil || got.ID != "project-a" || got.Source != SourceIdentity {
		t.Fatalf("post-remote-change: id=%q src=%q err=%v (want project-a / identity)", got.ID, got.Source, err)
	}
}

func TestIdentityStableAcrossLocalRename(t *testing.T) {
	// Simulate a checkout move: bind once, then "move" the repo by copying the
	// worktree to a sibling path. The identity file lives in the git directory,
	// which travels with the repo, so both checkouts must agree.
	src := t.TempDir()
	initGitRepo(t, src)
	remoteCmd(t, src, "remote", "add", "origin", "git@github.com:acme/stable.git")
	if _, err := resolveInDir(src); err != nil {
		t.Fatalf("first resolve: %v", err)
	}

	dst := filepath.Join(filepath.Dir(src), "stable-move")
	if out, err := copyDir(src, dst); err != nil {
		t.Fatalf("cp: %v\n%s", err, out)
	}
	if got, err := resolveInDir(dst); err != nil || got.ID != "stable" {
		t.Fatalf("after-copy: id=%q err=%v (want stable)", got.ID, err)
	}
}

func resolveInDir(dir string) (Resolution, error) {
	orig, err := os.Getwd()
	if err != nil {
		return Resolution{}, err
	}
	if err := os.Chdir(dir); err != nil {
		return Resolution{}, err
	}
	defer os.Chdir(orig)
	res, err := ResolveDetailed(".")
	return res, err
}

func TestAmbiguityWithMultipleChildRepos(t *testing.T) {
	parent := t.TempDir()
	// Three child repos under parent.
	for _, name := range []string{"alpha", "beta", "gamma"} {
		dir := filepath.Join(parent, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		initGitRepo(t, dir)
	}
	// Run from parent (not inside any child).
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(parent)

	res, err := ResolveDetailed(".")
	if err == nil {
		t.Fatalf("expected ambiguity, got %s", res.ID)
	}
	if res.ID == "" {
		t.Fatalf("expected a fallback ID even on ambiguity")
	}
	if len(res.Available) != 3 {
		t.Fatalf("want 3 candidates, got %v", res.Available)
	}
	want := map[string]bool{"alpha": true, "beta": true, "gamma": true}
	for _, p := range res.Available {
		if !want[p] {
			t.Fatalf("unexpected candidate %q", p)
		}
	}
	if res.Source != SourceAmbig {
		t.Fatalf("source=%q want ambiguous", res.Source)
	}
}

func TestChildAutoPromotion(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "only")
	os.MkdirAll(child, 0o755)
	initGitRepo(t, child)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(parent)

	res, err := ResolveDetailed(".")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if res.ID != "only" || res.Source != SourceChild {
		t.Fatalf("id=%q src=%q warn=%q (want only / git-child)", res.ID, res.Source, res.Warning)
	}
	if res.Warning == "" {
		t.Fatalf("expected an auto-promote warning")
	}
}

func TestProcessOverrideWinsOverEverything(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	remoteCmd(t, dir, "remote", "add", "origin", "git@github.com:acme/real.git")

	t.Setenv(EnvProjectOverride, "pinned-bucket")
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	res, err := ResolveDetailed(".")
	if err != nil || res.ID != "pinned-bucket" || res.Source != SourceOverride {
		t.Fatalf("override: id=%q src=%q err=%v", res.ID, res.Source, err)
	}
}

func TestConfigBoundedToEnclosingRepo(t *testing.T) {
	// Parent has an ancestor config that must NOT win for an unrelated child.
	parent := t.TempDir()
	if err := os.MkdirAll(filepath.Join(parent, ".skillgrid"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, ".skillgrid", "config.json"), []byte(`{"project":"parent-claim"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Child is its own repo with no config (identity will win after first
	// resolve).
	child := filepath.Join(parent, "child-repo")
	os.MkdirAll(child, 0o755)
	initGitRepo(t, child)

	// Write a config in the child itself to prove scoping.
	if err := os.MkdirAll(filepath.Join(child, ".skillgrid"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, ".skillgrid", "config.json"), []byte(`{"project":"child-claim"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(child)

	res, err := ResolveDetailed(".")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if res.ID != "child-claim" || res.Source != SourceConfig {
		t.Fatalf("child resolve: id=%q src=%q (want child-claim / config)", res.ID, res.Source)
	}
}

// TestDirectoryHashFallbackForNonGitDir exercises the legacy branch when the
// path has no git repo anywhere up the tree.
func TestDirectoryHashFallbackForNonGitDir(t *testing.T) {
	dir := t.TempDir()
	// Move to a location that has no git repo and no .skillgrid config in the
	// way (TempDir paths usually don't).
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	res, err := ResolveDetailed(".")
	if err != nil {
		t.Fatalf("err=%v (non-git dir should never error)", err)
	}
	if res.Source != SourceFallback {
		t.Fatalf("source=%q want directory-hash", res.Source)
	}
	if res.ID == "" {
		t.Fatalf("fallback ID must be non-empty")
	}
}

// TestFallbackIDDeterministic guards the directory-hash contract: same abs
// path → same ID. The whole legacy fallback is this contract, so a regression
// would scatter existing stores again.
func TestFallbackIDDeterministic(t *testing.T) {
	dir := "/data/git/AI"
	first := fallbackProjectID(dir)
	again := fallbackProjectID(dir)
	if first != again {
		t.Fatalf("fallback id not deterministic: %q != %q", first, again)
	}
	if !strings.Contains(first, "-") {
		t.Fatalf("expected {base}-{hash} form, got %q", first)
	}
}

type copyErr struct{ out []byte }

func (e *copyErr) Error() string { return string(e.out) }

func copyDir(src, dst string) (string, error) {
	out, err := exec.Command("cp", "-a", src, dst).CombinedOutput()
	if err != nil {
		return string(out), &copyErr{out}
	}
	return string(out), nil
}
