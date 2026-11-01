package eval

import (
	"testing"
)

// TestQuerySetNoiseDrops covers @step-01 (Scenario: Evaluation harness drops
// noise commits from the query set): the git-history-derived query generator
// keeps only genuine code changes and drops merges, reverts, releases,
// version-bumps, formatting-only commits, changelog-like commits, and
// benchmark-touching commits.
func TestQuerySetNoiseDrops(t *testing.T) {
	cases := []struct {
		subject   string
		kept      bool
		files     []string
		why       string
	}{
		{"fix: correct retry backoff in http client", true, []string{"internal/http/client.go"}, "genuine code change kept"},
		{"Merge branch 'feat/foo' into main", false, []string{"internal/http/client.go"}, "merge dropped"},
		{"Merge pull request #42 from acme/patch", false, []string{"a.go"}, "merge PR dropped"},
		{"revert: 'feat: add flaky retry'", false, []string{"a.go"}, "revert dropped"},
		{"revert flaky retry", false, []string{"a.go"}, "revert dropped (lowercase subject)"},
		{"Reverted the flaky retry", false, []string{"a.go"}, "reverted (past) dropped"},
		{"This reverts commit abc1234", false, []string{"a.go"}, "this reverts dropped"},
		{"fix: avoid reverting state on cancel", true, []string{"internal/http/client.go"}, "subject that merely MENTIONS reverting is kept (regex is start-anchored)"},
		{"fix: revert is not idempotent", true, []string{"internal/http/client.go"}, "mid-subject 'revert' is kept (not start-anchored)"},
		{"chore: bump version to 1.2.3", false, []string{"go.mod"}, "version bump dropped"},
		{"chore(deps): bump libx from 1.0 to 2.0", false, []string{"go.mod"}, "dep bump dropped"},
		{"release 1.2.3", false, []string{"a.go"}, "release dropped"},
		{"v1.2.3", false, []string{"a.go"}, "version tag dropped"},
		{"chore: gofmt", false, []string{"a.go"}, "formatting dropped"},
		{"style: fix formatting", false, []string{"a.go"}, "formatting (style) dropped"},
		{"docs: update CHANGELOG for 1.2.3", false, []string{"CHANGELOG.md"}, "changelog-like dropped"},
		{"Update CHANGELOG.md", false, []string{"CHANGELOG.md"}, "changelog file dropped"},
		{"chore: tidy go.sum", false, []string{"go.sum"}, "go.sum bump dropped"},
		{"feat: benchmark http client", false, []string{"bench_test.go"}, "benchmark-touching dropped"},
		{"perf: tune allocator in bench", false, []string{"bench.go"}, "benchmark file dropped"},
		{"fix: off-by-one in pagination", true, []string{"internal/page.go"}, "plain fix kept"},
	}
	for i, c := range cases {
		q, err := BuildQuerySet(c.subject, c.files)
		if err != nil {
			t.Fatalf("case %d (%q) BuildQuerySet: %v", i, c.subject, err)
		}
		got := q != nil
		if got != c.kept {
			t.Errorf("case %d (%q): kept=%v want %v — %s", i, c.subject, got, c.kept, c.why)
		}
	}
}

// TestCorpusExcludes covers @step-01 (Scenario: Evaluation corpus excludes the
// benchmark scaffolding): the eval harness itself (the benchmark scaffolding)
// is excluded from the graded corpus, so a file under the eval directory is
// dropped from the query set's expected files.
func TestCorpusExcludes(t *testing.T) {
	cases := []struct {
		path string
		in   bool
	}{
		{"internal/mnemonic/eval/genqueries.go", false},
		{"internal/mnemonic/eval/metrics.go", false},
		{"internal/mnemonic/eval/harness.go", false},
		{"skillgrid-cli/internal/mnemonic/eval/metrics_test.go", false},
		{"internal/mnemonic/hybrid/rank.go", true},
		{"internal/http/client.go", true},
		{".skillgrid/sdd/008-mnemonic-community-knowledge-graph/eval/fixture.go", false},
	}
	for _, c := range cases {
		got := !ExcludedFromCorpus(c.path)
		if got != c.in {
			t.Errorf("path %q: in-corpus=%v want %v", c.path, got, c.in)
		}
	}
}

// TestGenQueriesLeakFree covers @step-01 (Scenario: Evaluation harness derives
// a leak-free query set): a commit's expected files never include the file the
// query subject is drawn from in a way that leaks the answer — specifically,
// the generator keeps the full changed-file set minus corpus excludes, and an
// empty/noise set yields a nil query.
func TestGenQueriesLeakFree(t *testing.T) {
	// A genuine change that touches an eval file + a real file: the eval file
	// is dropped from the expected set (the scaffolding is not a signal), the
	// real file is kept.
	q, err := BuildQuerySet("fix: real change", []string{
		"internal/mnemonic/eval/metrics.go",
		"internal/http/client.go",
	})
	if err != nil {
		t.Fatalf("BuildQuerySet: %v", err)
	}
	if q == nil {
		t.Fatal("expected a non-nil query")
	}
	for _, f := range q.ExpectedFiles {
		if f == "internal/mnemonic/eval/metrics.go" {
			t.Errorf("eval scaffolding leaked into expected files: %v", q.ExpectedFiles)
		}
	}
	found := false
	for _, f := range q.ExpectedFiles {
		if f == "internal/http/client.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("real changed file missing from expected files: %v", q.ExpectedFiles)
	}

	// A change touching ONLY eval scaffolding yields no query (noise).
	nq, err := BuildQuerySet("fix: eval only", []string{"internal/mnemonic/eval/metrics.go"})
	if err != nil {
		t.Fatalf("BuildQuerySet eval-only: %v", err)
	}
	if nq != nil {
		t.Errorf("eval-only change should yield a nil query, got %+v", nq)
	}
}
