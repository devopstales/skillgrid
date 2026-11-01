package project_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/project"
)

// TestProjectDetectionSelectors covers the detection chain per selector
// (BDD [project-detection-selectors]).
func TestProjectDetectionSelectors(t *testing.T) {
	det := project.Detector{EnvVar: "MNEMONIC_PROJECT"}
	git := func(dir string) (string, error) { return filepath.Join(dir, "repo"), nil }
	got, err := det.Resolve("/x/repo", "explicit", git)
	if err != nil {
		t.Fatalf("Resolve(explicit): %v", err)
	}
	if got != "explicit" {
		t.Fatalf("explicit must win: %q", got)
	}
	t.Setenv("MNEMONIC_PROJECT", "from-env")
	got, err = det.Resolve("/x/repo", "", git)
	if err != nil {
		t.Fatalf("Resolve(env): %v", err)
	}
	if got != "from-env" {
		t.Fatalf("env must win over detection: %q", got)
	}
	t.Setenv("MNEMONIC_PROJECT", "")
	got, err = det.Resolve("/x/repo", "", git)
	if err != nil {
		t.Fatalf("Resolve(git): %v", err)
	}
	if got != "repo" {
		t.Fatalf("git binding → basename: %q", got)
	}
	noGit := func(string) (string, error) { return "", os.ErrNotExist }
	got, err = det.Resolve("/x/plain-dir", "", noGit)
	if err != nil {
		t.Fatalf("Resolve(basename): %v", err)
	}
	if got != "plain-dir" {
		t.Fatalf("basename fallback: %q", got)
	}
}

// TestProjectGitBindingFallback is the TRIANGULATE case: the git toplevel is
// an ancestor of cwd, so the basename of the toplevel (not of cwd) must be
// used — the Engram hostbind semantics.
func TestProjectGitBindingFallback(t *testing.T) {
	det := project.Detector{EnvVar: "MNEMONIC_PROJECT"}
	t.Setenv("MNEMONIC_PROJECT", "")
	git := func(string) (string, error) { return "/home/u/myrepo", nil }
	got, err := det.Resolve("/home/u/myrepo/src/deep", "", git)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "myrepo" {
		t.Fatalf("ancestor git binding must resolve to the toplevel basename: %q", got)
	}
}

// TestProjectCanonicalize pins the schema_<sanitized_project> input (Task
// 18): lowercase, path separator → '-', trailing separator stripped.
func TestProjectCanonicalize(t *testing.T) {
	cases := map[string]string{
		"Repo":        "repo",
		"Repo/":       "repo",
		"Repo/Sub":    "repo-sub",
		"Repo/Sub/":   "repo-sub",
		"repo--x":     "repo--x", // already-canonical: untouched
		"MiXeD/CaSe/": "mixed-case",
		"a/b/c/d/":    "a-b-c-d",
	}
	for in, want := range cases {
		if got := project.Canonicalize(in); got != want {
			t.Fatalf("Canonicalize(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestProjectResolveUsageError pins the CLI rule (spec 005-transport-cli §3):
// an explicit project combined with --all is a usage error, and Resolve with
// an unresolvable cwd is an error (there is no fallback to "guess").
func TestProjectResolveUsageError(t *testing.T) {
	det := project.Detector{EnvVar: "MNEMONIC_PROJECT"}
	t.Setenv("MNEMONIC_PROJECT", "")
	noGit := func(string) (string, error) { return "", os.ErrNotExist }
	// explicit without --all resolves normally
	if got, err := det.ResolveWithAll("/x", "explicit", false, noGit); err != nil || got != "explicit" {
		t.Fatalf("explicit without --all must resolve: %q, %v", got, err)
	}
	if _, err := det.ResolveWithAll("/x", "explicit", true, noGit); err == nil {
		t.Fatal("explicit project + --all must be a usage error")
	}
	// --all without an explicit project resolves to the "every project" marker
	// (empty string), not an error — the handler turns it into fan-out.
	if got, err := det.ResolveWithAll("/x", "", true, func(string) (string, error) { return "/x/repo", nil }); err != nil || got != "" {
		t.Fatalf("--all without explicit must resolve to the all-marker: %q, %v", got, err)
	}
	if _, err := det.Resolve("", "", noGit); err == nil {
		t.Fatal("empty cwd must error")
	}
	if _, err := det.Resolve("/", "", noGit); err == nil {
		t.Fatal("root cwd without a git binding must error (no basename to fall back on)")
	}
	if got, err := det.Resolve("/x", "", func(string) (string, error) { return "", errors.New("git gone") }); err != nil || got != "x" {
		t.Fatalf("git error must fall through to basename(cwd): %q, %v", got, err)
	}
}
