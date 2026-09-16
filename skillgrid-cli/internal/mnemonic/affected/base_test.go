package affected

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// gitFixtureRepo builds a hermetic temp git repo with a base commit on
// "main" and a feature branch "feat" with extra commits, so a merge-base
// diff has a real changed set. It never depends on the real repo's history.
func gitFixtureRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		// HERMES_HOME isolates the fixture from any ambient git global
		// config (incl. hooks/config that could color the blame output).
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+filepath.Join(root, "noconfig"), "GIT_CONFIG_SYSTEM="+filepath.Join(root, "noconfig"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(name, content string) {
		t.Helper()
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "fixture@example.com")
	run("config", "user.name", "Fixture")
	write("src/base.go", "package p\n\nfunc base() int {\n\treturn 1\n}\n")
	write("src/base_test.go", "package p\n\nfunc TestBase() {\n\t_ = base()\n}\n")
	run("add", "-A")
	run("commit", "-m", "base")
	run("checkout", "-b", "feat")
	// The branch changes base.go and adds mid.go (which base_test does not
	// touch) — the merge-base diff against main is exactly these files.
	write("src/base.go", "package p\n\nfunc base() int {\n\treturn 2\n}\n")
	write("src/mid.go", "package p\n\nfunc mid() int {\n\treturn base()\n}\n")
	run("add", "-A")
	run("commit", "-m", "feature")
	return root
}

// TestCodeAffectedBase covers @step-02 (Scenario: code_affected --base derives
// areas and owners): `--base main` derives the changed file set from
// `git merge-base <base> HEAD` + diff (no --stdin plumbing), walks the same
// import/tests_for traversal, groups the result into affected areas (per-hop
// detail collapsed under each area), and names git-history owners ("who to
// tag") for each touched area via `git blame` on the changed lines.
func TestCodeAffectedBase(t *testing.T) {
	st := fixtureGraph(t)
	root := gitFixtureRepo(t)

	res, err := AffectedBase(context.Background(), st.DB, root, "main", Options{})
	if err != nil {
		t.Fatalf("affected --base: %v", err)
	}
	// The merge-base diff (feat vs main) is src/base.go + src/mid.go.
	if !contains(res.Changed, "src/base.go") || !contains(res.Changed, "src/mid.go") {
		t.Fatalf("merge-base changed set = %v, want src/base.go + src/mid.go", res.Changed)
	}
	if contains(res.Changed, "src/base_test.go") {
		t.Errorf("the merge-base diff must NOT include the unchanged src/base_test.go, got %v", res.Changed)
	}
	if !contains(res.TestFiles, "src/base_test.go") {
		t.Fatalf("affected --base must surface the affected test file, got %v", res.TestFiles)
	}

	// Affected areas: per-hop detail collapsed under each area. base.go is
	// both changed and a hop source, so it forms an area; the hop detail is
	// aggregated under it.
	if len(res.Areas) == 0 {
		t.Fatalf("expected affected areas, got none")
	}
	var baseArea *Area
	for i := range res.Areas {
		if res.Areas[i].Name == "src/base.go" {
			baseArea = &res.Areas[i]
		}
	}
	if baseArea == nil {
		t.Fatalf("expected an area for changed file src/base.go, got %+v", res.Areas)
	}
	if !contains(baseArea.TestFiles, "src/base_test.go") {
		t.Errorf("the base.go area must list src/base_test.go, got %v", baseArea.TestFiles)
	}
	if baseArea.Hops == 0 {
		t.Errorf("the base.go area must carry collapsed hop detail, got none")
	}

	// Git-history owners ("who to tag"): git blame on the changed lines of
	// src/base.go names the fixture committer.
	if baseArea.Owners == nil {
		t.Fatalf("expected owners for the base.go area, got nil")
	}
	owner := strings.ToLower(baseArea.Owners[0])
	if !strings.Contains(owner, "fixture") && !strings.Contains(owner, "fixture@example.com") {
		t.Errorf("owner should be the git-history committer (fixture), got %v", baseArea.Owners)
	}
}

// TestCodeAffectedBaseNoChanges covers @step-02 (--base on a branch with no
// diff against the base ref): an empty merge-base diff is an empty result with
// a clear message, not an error.
func TestCodeAffectedBaseNoChanges(t *testing.T) {
	st := fixtureGraph(t)
	root := gitFixtureRepo(t)
	// HEAD == feat; diff against feat is empty.
	res, err := AffectedBase(context.Background(), st.DB, root, "feat", Options{})
	if err != nil {
		t.Fatalf("affected --base (empty diff): %v", err)
	}
	if len(res.Changed) != 0 {
		t.Errorf("empty merge-base diff must report no changed files, got %v", res.Changed)
	}
	if res.Message == "" {
		t.Errorf("empty merge-base diff must carry a clear message, got empty")
	}
}

// TestCodeAffectedBaseBadRef covers @step-02 (--base with an unknown ref is a
// clear error, not an invented changed set).
func TestCodeAffectedBaseBadRef(t *testing.T) {
	st := fixtureGraph(t)
	root := gitFixtureRepo(t)
	if _, err := AffectedBase(context.Background(), st.DB, root, "no-such-ref", Options{}); err == nil {
		t.Errorf("an unknown --base ref must be a clear error")
	}
}

// TestParseBlameAuthor covers @step-02 (review fix): the author name is
// everything between the metadata open paren and the " <10-digit epoch>"
// timestamp — author names containing digits ("Dev2") must NOT be truncated
// at their first digit run, and the timestamp must not leak into the name.
func TestParseBlameAuthor(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{
			line: "3b729a55 (Dev2 2026-09-10 10:21:08 +0200 1) one1",
			want: "Dev2",
		},
		{
			line: "3b729a55 (Dev2 1789028468 1) one1",
			want: "Dev2",
		},
		{
			line: "3b729a55 (Fixture 1789028468 2) two2",
			want: "Fixture",
		},
		{
			line: "3b729a55 (John Smith Jr 1789028468 2) two2",
			want: "John Smith Jr",
		},
		{
			line: "3b729a55 (Fixture 2025-01-02 03:04:05 -0700 42) x",
			want: "Fixture",
		},
		{
			line: "3b729a55 (no trailing metadata) ",
			want: "no trailing metadata",
		},
		{
			line: "no parens at all",
			want: "",
		},
	}
	for _, c := range cases {
		if got := parseBlameAuthor(c.line); got != c.want {
			t.Errorf("parseBlameAuthor(%q) = %q, want %q", c.line, got, c.want)
		}
	}
}

// TestCodeAffectedBaseDigitAuthor covers @step-02 (review fix): a git author
// whose name contains a digit ("Dev2") survives `git blame` parsing end to
// end (owner is "Dev2", not "Dev").
func TestCodeAffectedBaseDigitAuthor(t *testing.T) {
	st := fixtureGraph(t)
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL="+filepath.Join(root, "noconfig"),
			"GIT_CONFIG_SYSTEM="+filepath.Join(root, "noconfig"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-b", "main")
	run("config", "user.name", "Dev2")
	run("config", "user.email", "dev2@example.com")
	commit := func(files map[string]string, msg string) {
		t.Helper()
		for name, content := range files {
			p := filepath.Join(root, name)
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", name, err)
			}
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				t.Fatalf("write %s: %v", name, err)
			}
		}
		run("add", "-A")
		run("commit", "-m", msg)
	}
	commit(map[string]string{
		"src/base.go":     "package p\n\nfunc base() int {\n\treturn 1\n}\n",
		"src/base_test.go": "package p\n\nfunc TestBase() {\n\t_ = base()\n}\n",
	}, "base")
	run("checkout", "-b", "feat")
	commit(map[string]string{
		"src/base.go": "package p\n\nfunc base() int {\n\treturn 2\n}\n",
	}, "feature")

	res, err := AffectedBase(context.Background(), st.DB, root, "main", Options{})
	if err != nil {
		t.Fatalf("affected --base (digit author): %v", err)
	}
	var baseArea *Area
	for i := range res.Areas {
		if res.Areas[i].Name == "src/base.go" {
			baseArea = &res.Areas[i]
		}
	}
	if baseArea == nil {
		t.Fatalf("expected an area for changed file src/base.go, got %+v", res.Areas)
	}
	if !contains(baseArea.Owners, "Dev2") {
		t.Fatalf("owner must be parsed as \"Dev2\" (digit in name preserved), got %v", baseArea.Owners)
	}
	if contains(baseArea.Owners, "Dev") {
		t.Errorf("owner must not be truncated to \"Dev\", got %v", baseArea.Owners)
	}
}

var _ = sort.Strings
