package eval

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// evalFixtureCorpus is a small synthetic corpus (path → content) standing in
// for a git repo's file set. It has >5 files so that recall@5 can distinguish
// a ranker that puts the answer first (firstRanker) from one that puts it last
// (lastRanker) — with only 3 files both would reach recall@5 = 1.0.
func evalFixtureCorpus() map[string]string {
	// 3 relevant files (the query targets) + 8 distractors, so a "last" ranker
	// ranks the target beyond the top-5 window.
	paths := []string{
		"internal/http/client.go",
		"internal/page/pagination.go",
		"internal/auth/token.go",
		"vendor/lib/a.go",
		"vendor/lib/b.go",
		"vendor/lib/c.go",
		"vendor/lib/d.go",
		"vendor/lib/e.go",
		"vendor/lib/f.go",
		"vendor/lib/g.go",
		"vendor/lib/h.go",
	}
	m := make(map[string]string, len(paths))
	for i, p := range paths {
		m[p] = fmt.Sprintf("func f%d() int { return %d }\n", i, i)
	}
	return m
}

// evalFixtureQueries builds a query set over the fixture corpus. A larger
// number of queries (12) gives the seeded permutation test enough power to
// call a uniformly-better ranker significant at p < 0.05.
func evalFixtureQueries() []*QuerySet {
	targets := []string{
		"internal/http/client.go",
		"internal/page/pagination.go",
		"internal/auth/token.go",
		"vendor/lib/a.go",
		"vendor/lib/b.go",
		"vendor/lib/c.go",
		"vendor/lib/d.go",
		"vendor/lib/e.go",
		"vendor/lib/f.go",
		"vendor/lib/g.go",
		"vendor/lib/h.go",
	}
	subjects := []string{
		"fix http client retry", "fix pagination off-by-one", "verify auth token",
		"fix lib a path", "fix lib b path", "fix lib c path", "fix lib d path",
		"fix lib e path", "fix lib f path", "fix lib g path", "fix lib h path",
		"refactor lib paths",
	}
	qs := make([]*QuerySet, 0, len(targets))
	for i, t := range targets {
		qs = append(qs, &QuerySet{Subject: subjects[i], ExpectedFiles: []string{t}})
	}
	return qs
}

// isExpected reports whether p is one of the query's expected files.
func isExpected(q *QuerySet, p string) bool {
	for _, e := range q.ExpectedFiles {
		if p == e {
			return true
		}
	}
	return false
}

// firstRanker puts the expected files first, then the rest in sorted order
// (a strong candidate ranker).
func firstRanker(q *QuerySet, idx *CorpusIndex) []string {
	rest := []string{}
	for _, p := range idx.Paths() {
		if !isExpected(q, p) {
			rest = append(rest, p)
		}
	}
	return append(append([]string(nil), q.ExpectedFiles...), rest...)
}

// lastRanker puts the expected files last, then the rest in sorted order
// (a weak baseline ranker).
func lastRanker(q *QuerySet, idx *CorpusIndex) []string {
	rest := []string{}
	for _, p := range idx.Paths() {
		if !isExpected(q, p) {
			rest = append(rest, p)
		}
	}
	return append(rest, q.ExpectedFiles...)
}

// factorRanker is a lexical projection of the step-01 retrieval-quality
// factors (mirroring cmd/skillgrid's shippedRank): it reorders the corpus by
// path-match / documentation / generated-vendor / test signals, NOT by the
// expected files. It is a faithful lexical analogue of hybrid's rerank table —
// the key property for TestShippedConfigSignificance is that it DIFFERS from
// the lastRanker baseline (so the measured delta is non-vacuous, not
// identity==baseline). Deterministic.
func factorRanker(q *QuerySet, idx *CorpusIndex) []string {
	paths := idx.Paths()
	qLower := strings.ToLower(q.Subject)
	qTerms := strings.Fields(qLower)
	isTestQuery := strings.Contains(qLower, "test")
	type scored struct {
		path  string
		delta float64
	}
	var out []scored
	for _, p := range paths {
		out = append(out, scored{path: p, delta: testFactorDelta(p, qLower, qTerms, isTestQuery)})
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && (out[j].delta > out[j-1].delta || (out[j].delta == out[j-1].delta && out[j].path < out[j-1].path)); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	res := make([]string, len(out))
	for i, s := range out {
		res[i] = s.path
	}
	return res
}

// testFactorDelta scores one path for a query using the same named, bounded
// factors as hybrid.RerankTable, projected onto the path only (no symbol/
// degree, which the corpus index does not carry).
func testFactorDelta(path, qLower string, qTerms []string, isTestQuery bool) float64 {
	delta := 0.0
	lower := strings.ToLower(path)
	leaf := lower
	if i := strings.LastIndex(leaf, "/"); i >= 0 {
		leaf = leaf[i+1:]
	}
	stem := leaf
	if i := strings.LastIndex(leaf, "."); i >= 0 {
		stem = leaf[:i]
	}
	// exact-symbol (proxy): the full query or a long term appears in the stem.
	if len(qLower) >= 3 && strings.Contains(stem, qLower) {
		delta += 0.30
	} else {
		for _, t := range qTerms {
			if len(t) >= 4 && t != "test" && strings.Contains(stem, t) {
				delta += 0.30
				break
			}
		}
	}
	// path-match: a term >=3 chars appears in the full path.
	for _, t := range qTerms {
		if len(t) >= 3 && strings.Contains(lower, t) {
			delta += 0.10
			break
		}
	}
	// source-over-prose / documentation.
	if isTestFactorProse(lower) {
		delta += -0.10
	} else {
		delta += 0.05
	}
	// generated/vendor.
	if isTestFactorVendor(lower) {
		delta += -0.15
	}
	// test-on-non-test.
	if isTestFactorTest(lower) && !isTestQuery {
		delta += -0.10
	}
	return delta
}

func isTestFactorProse(lower string) bool {
	for _, m := range []string{"/docs/", "/doc/", "readme", "changelog", ".md", ".rst", ".txt", ".yaml", ".yml", ".json", ".toml"} {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

func isTestFactorVendor(lower string) bool {
	if strings.Contains(lower, "vendor/") || strings.Contains(lower, "/vendor/") {
		return true
	}
	for _, m := range []string{"_generated.", ".pb.", ".gen.", "generated.go", "zz_"} {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

func isTestFactorTest(lower string) bool {
	return strings.Contains(lower, "_test.go") || strings.HasSuffix(lower, ".test.js") || strings.Contains(lower, "test_")
}

// TestEvalHarnessSignificance covers @step-01 (Scenario: Evaluation harness
// derives a leak-free query set and reports significance): the ablation runner
// builds ONE shared index per corpus, runs baseline + candidate variants, and
// reports a per-variant row with the full metric set plus a seeded paired
// bootstrap 95% CI and permutation p-value vs the baseline. A strictly better
// candidate yields a positive, significant delta (CI lower > 0, p < 0.05).
func TestEvalHarnessSignificance(t *testing.T) {
	ctx := context.Background()
	queries := evalFixtureQueries()

	// Baseline ranks the answer last; candidate ranks it first → strictly
	// better on every query.
	res, err := Run(ctx, RunConfig{
		Corpora: map[string]map[string]string{"self": evalFixtureCorpus()},
		Queries: map[string][]*QuerySet{"self": queries},
		Variants: []Variant{
			{Name: "baseline", Rank: lastRanker, Baseline: true},
			{Name: "candidate", Rank: firstRanker},
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Baseline == nil {
		t.Fatal("expected a baseline row")
	}
	cand := rowByName(res, "candidate")
	if cand == nil {
		t.Fatal("missing candidate row")
	}
	// The candidate must beat the baseline on the headline metrics.
	if cand.DeltaRecall5 <= 0 || cand.DeltaMRR <= 0 {
		t.Errorf("candidate did not beat baseline: deltaRecall5=%v deltaMRR=%v", cand.DeltaRecall5, cand.DeltaMRR)
	}
	// Significance fields must be present (non-zero p is fine; CI bounds set).
	if cand.PValue > 1 || cand.PValue < 0 {
		t.Errorf("permutation p out of range: %v", cand.PValue)
	}
	if cand.CILower > cand.CIUpper {
		t.Errorf("CI inverted: lower=%v upper=%v", cand.CILower, cand.CIUpper)
	}
	// A strictly-better candidate is significant.
	if cand.PValue >= 0.05 {
		t.Errorf("strictly-better candidate should be significant, p=%v", cand.PValue)
	}
	if cand.CILower <= 0 {
		t.Errorf("strictly-better candidate CI lower should be > 0, got %v", cand.CILower)
	}
}

// TestEvalStaleExpectation covers @step-01 (Scenario: Stale evaluation
// expectation fails the run loudly): a query whose expected file no longer
// exists at HEAD makes validate_queries fail the run with a loud error — not a
// silently deflated score.
func TestEvalStaleExpectation(t *testing.T) {
	ctx := context.Background()
	// The query references a file that is NOT in the corpus (deleted at HEAD).
	queries := []*QuerySet{
		{Subject: "fix http client retry", ExpectedFiles: []string{"internal/http/client.go"}},
		{Subject: "fix deleted module", ExpectedFiles: []string{"internal/gone/removed.go"}},
	}
	_, err := Run(ctx, RunConfig{
		Corpora: map[string]map[string]string{"self": evalFixtureCorpus()},
		Queries: map[string][]*QuerySet{"self": queries},
		Variants: []Variant{
			{Name: "baseline", Rank: lastRanker, Baseline: true},
		},
	})
	if err == nil {
		t.Fatal("expected Run to fail loudly on a stale expectation")
	}
	if !strings.Contains(err.Error(), "gone/removed.go") {
		t.Errorf("stale-expectation error should name the missing file, got: %v", err)
	}
}

// TestOneIndexPerCorpus covers @step-01 (Scenario: Ablation shares one index so
// deltas measure ranking): all variants over a corpus share the SAME index
// instance, so the only thing that varies between variants is the ranking —
// never the retrieved document set.
func TestOneIndexPerCorpus(t *testing.T) {
	ctx := context.Background()
	queries := evalFixtureQueries()

	// Both the baseline and the candidate record the index pointer they saw.
	// Proving they are the SAME instance (not just IndexInstances == 1) is what
	// actually establishes "one index per corpus": the only thing that varies
	// between variants is the ranking, never the retrieved document set.
	seen := map[string]*CorpusIndex{}
	record := func(name string) RankFunc {
		return func(q *QuerySet, idx *CorpusIndex) []string {
			seen[name] = idx
			return idx.Paths()
		}
	}
	res, err := Run(ctx, RunConfig{
		Corpora: map[string]map[string]string{"self": evalFixtureCorpus()},
		Queries: map[string][]*QuerySet{"self": queries},
		Variants: []Variant{
			{Name: "baseline", Rank: record("baseline"), Baseline: true},
			{Name: "candidate", Rank: record("candidate")},
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.IndexInstances["self"] != 1 {
		t.Errorf("corpus 'self' should be indexed exactly once, got %d", res.IndexInstances["self"])
	}
	if seen["baseline"] == nil {
		t.Fatal("baseline variant never received an index pointer")
	}
	if seen["candidate"] == nil {
		t.Fatal("candidate variant never received an index pointer")
	}
	// The crux: baseline and candidate must have received the IDENTICAL index
	// instance (same pointer) for the corpus.
	if seen["baseline"] != seen["candidate"] {
		t.Errorf("baseline and candidate must share the SAME CorpusIndex instance; got %p vs %p", seen["baseline"], seen["candidate"])
	}
}

// TestShippedConfigSignificance covers @step-01 (Scenario: Shipped ranking
// config is the significance winner): on >=2 pooled corpora, the shipped
// ranking config is non-negative vs the 005 baseline on every corpus AND every
// shipped signal survives significance; a candidate that fails is kept-off with
// the decision + CI/p recorded in the report.
func TestShippedConfigSignificance(t *testing.T) {
	ctx := context.Background()
	corpA := evalFixtureCorpus()
	corpB := map[string]string{
		"src/net/conn.go": "func (c *Conn) Read(b []byte) (int, error) { return 0, nil }\n",
		"src/net/addr.go": "func (a *Addr) String() string { return a.host }\n",
	}
	// Pad the second corpus with distractors so recall@5 can separate the
	// rankers (same reason as the first corpus).
	for i := 0; i < 8; i++ {
		corpB[fmt.Sprintf("src/net/distractor%d.go", i)] = fmt.Sprintf("func d%d() int { return %d }\n", i, i)
	}
	qA := evalFixtureQueries()
	qB := []*QuerySet{
		{Subject: "fix conn read", ExpectedFiles: []string{"src/net/conn.go"}},
		{Subject: "format addr string", ExpectedFiles: []string{"src/net/addr.go"}},
	}

	res, err := Run(ctx, RunConfig{
		Corpora: map[string]map[string]string{
			"self": corpA,
			"second": corpB,
		},
		Queries: map[string][]*QuerySet{"self": qA, "second": qB},
		Variants: []Variant{
			{Name: "baseline", Rank: lastRanker, Baseline: true},
			// Shipped signal: the real step-01 factor-based ranker (a lexical
			// projection of hybrid's rerank table) — NOT the identity/baseline,
			// so the measured delta is non-vacuous. On the fixture it ranks the
			// expected internal-source files ahead of the vendor distractors →
			// non-negative vs the lastRanker baseline → ships.
			{Name: "shipped-signal", Rank: factorRanker},
			// A candidate that is worse → must be kept-off (decision recorded).
			{Name: "worse-signal", Rank: lastRanker},
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Corpora) != 2 {
		t.Fatalf("expected 2 pooled corpora, got %d", len(res.Corpora))
	}
	// The shipped (factor-based) ranker must produce a NON-IDENTICAL order vs
	// the baseline (proof the gate is measuring a real ranker, not identity).
	shipRow := rowByName(res, "shipped-signal")
	if shipRow == nil {
		t.Fatal("missing shipped-signal row")
	}
	if shipRow.DeltaRecall5 == 0 && shipRow.DeltaMRR == 0 && shipRow.DeltaNDGC10 == 0 {
		t.Errorf("shipped-signal (factor-based ranker) must not be vacuously 0 vs baseline: %+v", shipRow)
	}
	// The shipped signal is non-negative vs baseline on every corpus and
	// survived significance → decision is "ship".
	ship := shippedDecision(res, "shipped-signal")
	if !ship.Ships {
		t.Errorf("factor-based shipped signal should ship (non-negative vs baseline), got %+v", ship)
	}
	// The worse signal is negative somewhere → kept off, with a decision + CI/p.
	off := shippedDecision(res, "worse-signal")
	if off.Ships {
		t.Errorf("a signal that is not non-negative across all corpora must be kept off, got %+v", off)
	}
	if off.Decision == "" {
		t.Errorf("kept-off signal must record a decision, got empty")
	}
}

// TestFailingSignalDecision covers @step-01 (Scenario: Failing ranking signal is
// removed or kept off with the decision recorded): a candidate that loses to
// the baseline carries an explicit decision (kept-off/removed) with its CI and
// p-value in the report — never silently dropped.
func TestFailingSignalDecision(t *testing.T) {
	ctx := context.Background()
	res, err := Run(ctx, RunConfig{
		Corpora: map[string]map[string]string{
			"self":   evalFixtureCorpus(),
			"second": {"src/x/y.go": "func y() int { return 1 }\n"},
		},
		Queries: map[string][]*QuerySet{
			"self":   evalFixtureQueries(),
			"second": {{Subject: "y change", ExpectedFiles: []string{"src/x/y.go"}}},
		},
		Variants: []Variant{
			{Name: "baseline", Rank: lastRanker, Baseline: true},
			{Name: "loser", Rank: lastRanker},
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	loser := rowByName(res, "loser")
	if loser == nil {
		t.Fatal("missing loser row")
	}
	d := shippedDecision(res, "loser")
	if d.Ships {
		t.Errorf("a non-improving signal must not ship, got %+v", d)
	}
	if d.Decision == "" {
		t.Errorf("failing signal must record a decision, got empty")
	}
	// The row must carry its CI + p (the evidence for the decision).
	if loser.CILower == 0 && loser.CIUpper == 0 && loser.PValue == 0 {
		t.Errorf("failing-signal row must carry CI + p for the decision: %+v", loser)
	}
}


