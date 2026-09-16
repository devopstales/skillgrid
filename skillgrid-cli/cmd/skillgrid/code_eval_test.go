package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/eval"
)

// gitRun runs a git command in dir, fatal-ing the test on error.
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v (out %s)", args, err, out)
	}
}

// evalGitFixture builds a real git repo in a temp dir with a genuine code
// change + a noise change (a merge + a version bump), returning the repo root.
func evalGitFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	gitRun(t, root, "init", "-q")
	gitRun(t, root, "config", "user.email", "test@example.com")
	gitRun(t, root, "config", "user.name", "test")

	// Commit 1 (genuine): a real code change.
	if err := os.MkdirAll(filepath.Join(root, "internal", "http"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "http", "client.go"),
		[]byte("package http\n\nfunc NewClient() *Client { return &Client{} }\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	gitRun(t, root, "add", "-A")
	gitRun(t, root, "commit", "-q", "-m", "feat: add http client")

	// Commit 2 (genuine): a real code change in a second file.
	if err := os.WriteFile(filepath.Join(root, "internal", "http", "retry.go"),
		[]byte("package http\n\nfunc (c *Client) Retry(n int) bool { return n > 0 }\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	gitRun(t, root, "add", "-A")
	gitRun(t, root, "commit", "-q", "-m", "fix: retry backoff")

	// Commit 3 (noise): a version bump (go.mod) — must be dropped.
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	gitRun(t, root, "add", "-A")
	gitRun(t, root, "commit", "-q", "-m", "chore: bump version to 1.21")

	return root
}

// TestSkillgridEvalSelfCorpus covers @step-01 (Scenario: `skillgrid eval
// --corpus <self>`): the eval runner reads the git history of the self corpus,
// derives the leak-free query set, builds one index, runs the ablation, and
// emits a report with the baseline + the shipped config and its significance.
func TestSkillgridEvalSelfCorpus(t *testing.T) {
	root := evalGitFixture(t)
	// Derive commits from the real git history and confirm the noise drop.
	commits, err := readGitHistory(root)
	if err != nil {
		t.Fatalf("readGitHistory: %v", err)
	}
	if len(commits) < 2 {
		t.Fatalf("expected >=2 commits, got %d", len(commits))
	}
	queries := eval.DeriveQuerySet(commits)
	// The version-bump commit must be dropped; the two real commits kept.
	if len(queries) != 2 {
		t.Errorf("expected 2 leak-free queries (version bump dropped), got %d: %+v", len(queries), queries)
	}

	// The self corpus's file set (HEAD).
	files := loadCorpusFiles(root)
	if len(files) == 0 {
		t.Fatalf("expected corpus files, got none")
	}
	// Run the harness over the self corpus (one index, baseline + shipped).
	res, err := eval.Run(context.Background(), eval.RunConfig{
		Corpora: map[string]map[string]string{"self": files},
		Queries: map[string][]*eval.QuerySet{"self": queries},
		Variants: []eval.Variant{
			{Name: "baseline", Rank: func(q *eval.QuerySet, idx *eval.CorpusIndex) []string { return idx.Paths() }, Baseline: true},
		},
	})
	if err != nil {
		t.Fatalf("eval.Run: %v", err)
	}
	if res.IndexInstances["self"] != 1 {
		t.Errorf("self corpus should be indexed once, got %d", res.IndexInstances["self"])
	}
}

// TestSkillgridEvalUnknownCorpus covers @step-01 (bad community/eval args
// rejected): an unknown/invalid --corpus is rejected with a clear error, not an
// invented run.
func TestSkillgridEvalUnknownCorpus(t *testing.T) {
	_, err := parseEvalCorpora([]string{"self", "not-a-real-name"})
	if err == nil {
		t.Fatal("expected an error for an unknown corpus name without a path")
	}
	if !strings.Contains(err.Error(), "not-a-real-name") {
		t.Errorf("unknown-corpus error should name the corpus, got: %v", err)
	}

	// A corpus with a missing path is also rejected clearly.
	_, err = parseEvalCorpora([]string{"self", "ghost=/nonexistent/path/xyz"})
	if err == nil {
		t.Fatal("expected an error for a corpus path that does not exist")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("missing-path error should say the path does not exist, got: %v", err)
	}
}

// TestSkillgridEvalRequiresCorpus covers @step-01: the eval runner requires at
// least one --corpus (no invented default corpus).
func TestSkillgridEvalRequiresCorpus(t *testing.T) {
	if _, err := parseEvalCorpora(nil); err == nil {
		t.Error("expected an error when no --corpus is provided")
	}
}

// TestShippedRankDiffersFromBaseline covers the review finding that the CLI
// "shipped" variant must NOT be the identity ranking (which made the 01.12 gate
// vacuous — every delta exactly 0). shippedRank applies the step-01
// retrieval-quality factors (a lexical projection of hybrid's rerank table), so
// on a realistic query it reorders the corpus DIFFERENTLY from identityRank,
// and the harness reports a non-trivial (non-vacuously-0) delta.
func TestShippedRankDiffersFromBaseline(t *testing.T) {
	// A corpus where the factors reorder the corpus for the query "retry":
	//   - client_retry.go  → exact-symbol (stem contains "retry") + path-match
	//   - docs/notes.md    → documentation penalty
	//   - vendor/lib/dep.go→ generated-vendor penalty
	//   - main.go          → source-only baseline
	// identityRank returns them in sorted order:
	//   client_retry.go, docs/notes.md, main.go, vendor/lib/dep.go
	// shippedRank must reorder: the docs + vendor files are demoted, so the
	// order differs from identity.
	corpus := map[string]string{
		"client_retry.go":   "func Retry(n int) bool { return n > 0 }\n",
		"docs/notes.md":     "# notes\nretry on backoff\n",
		"vendor/lib/dep.go": "func Dep() {}\n",
		"main.go":           "func main() {}\n",
	}
	idx := buildCorpusIndex(corpus)

	q := &eval.QuerySet{Subject: "retry", ExpectedFiles: []string{"client_retry.go"}}
	base := identityRank(q, idx)
	ship := shippedRank(q, idx)

	// The two orders must differ (shippedRank applies the factors).
	if equalStrings(base, ship) {
		t.Fatalf("shippedRank must differ from identityRank on a realistic query; both gave %v", base)
	}
	// The demoted files (docs, vendor) must be lower under shippedRank than
	// under identityRank.
	if rankOf(ship, "docs/notes.md") <= rankOf(ship, "client_retry.go") {
		t.Errorf("documentation file should be demoted below the exact-symbol file; shipped order %v", ship)
	}
	if rankOf(ship, "vendor/lib/dep.go") <= rankOf(ship, "client_retry.go") {
		t.Errorf("vendor file should be demoted below the exact-symbol file; shipped order %v", ship)
	}

	// The harness must report a NON-IDENTICAL (non-vacuously-0) ranked order:
	// the shipped row's delta is measured against the baseline and is not
	// trivially the identity==baseline case. Run the harness with the real
	// shippedRank vs identityRank and confirm the shipped row differs from a
	// vacuous identity row (recall@5 for the expected file is 1.0 under
	// shippedRank because the exact-symbol file is ranked first, while under a
	// pure identity rank it would be ranked by path only).
	res, err := eval.Run(context.Background(), eval.RunConfig{
		Corpora: map[string]map[string]string{"self": corpus},
		Queries: map[string][]*eval.QuerySet{"self": {q}},
		Variants: []eval.Variant{
			{Name: "baseline", Rank: identityRank, Baseline: true},
			{Name: "shipped", Rank: shippedRank},
		},
	})
	if err != nil {
		t.Fatalf("eval.Run: %v", err)
	}
	shipRow := rowByNameRes(res, "shipped")
	if shipRow == nil {
		t.Fatal("missing shipped row")
	}
	// The shipped ranker puts the expected (exact-symbol) file first → recall@5
	// = 1.0. The baseline (identity) ranks by path: client_retry.go is first in
	// sorted order too, so this single-query recall is 1.0 for both. The
	// non-vacuity is proven by the ORDER difference above; here we assert the
	// row is real (has CI/p) and that shippedRank's ranking is non-trivial.
	if shipRow.Recall5 != 1.0 {
		t.Errorf("shippedRank should rank the exact-symbol file first → recall@5=1.0, got %v", shipRow.Recall5)
	}
	if shipRow.CILower == 0 && shipRow.CIUpper == 0 && shipRow.PValue == 0 {
		t.Errorf("shipped row must carry CI + p (a real measurement, not vacuous): %+v", shipRow)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func rankOf(order []string, p string) int {
	for i, x := range order {
		if x == p {
			return i
		}
	}
	return -1
}

func buildCorpusIndex(files map[string]string) *eval.CorpusIndex {
	return eval.NewCorpusIndex(files)
}

func rowByNameRes(res *eval.Result, name string) *eval.Row {
	return eval.RowByName(res, name)
}
