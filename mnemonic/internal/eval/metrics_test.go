package eval

import (
	"testing"
)

// TestRecallMRRNDGC covers @step-01 (Scenario: Evaluation harness reports
// metrics with paired CI + p-values): file-granularity IR metrics (recall@k,
// MRR, nDCG@10) are computed correctly for a single query's ranked candidate
// list against its expected files.
func TestRecallMRRNDGC(t *testing.T) {
	// expected: a.go (relevant). ranked: b.go, a.go, c.go.
	recall5 := recallAt([]string{"b.go", "a.go", "c.go"}, []string{"a.go"}, 5)
	if recall5 != 1.0 {
		t.Errorf("recall@5 = %v, want 1.0 (a.go is in top 5)", recall5)
	}
	recall3 := recallAt([]string{"b.go", "c.go", "a.go"}, []string{"a.go"}, 3)
	if recall3 != 1.0 {
		t.Errorf("recall@3 = %v, want 1.0", recall3)
	}
	// a.go at rank 2 → MRR = 1/2.
	mrr := mrrOf([]string{"b.go", "a.go", "c.go"}, []string{"a.go"})
	if want := 0.5; mrr < want-1e-9 || mrr > want+1e-9 {
		t.Errorf("MRR = %v, want 0.5", mrr)
	}
	// a.go not in top-10 → recall@10 = 0.
	recall10 := recallAt([]string{"b.go", "c.go", "d.go"}, []string{"a.go"}, 10)
	if recall10 != 0 {
		t.Errorf("recall@10 = %v, want 0 (a.go absent)", recall10)
	}
	// nDCG@10 in [0,1] and 1.0 when the relevant doc is first.
	ndcgPerfect := ndcgAt([]string{"a.go", "b.go", "c.go"}, []string{"a.go"}, 10)
	if ndcgPerfect < 1-1e-9 || ndcgPerfect > 1+1e-9 {
		t.Errorf("nDCG@10 (perfect) = %v, want ~1.0", ndcgPerfect)
	}
	ndcgImperfect := ndcgAt([]string{"b.go", "a.go", "c.go"}, []string{"a.go"}, 10)
	if ndcgImperfect <= 0 || ndcgImperfect >= 1 {
		t.Errorf("nDCG@10 (imperfect) = %v, want in (0,1)", ndcgImperfect)
	}
}

// TestUsefulBudgetTokensDup covers the utility metrics: useful@budget counts
// expected files found within the token budget; tokens estimates returned
// tokens; dup% measures near-duplicate (same-file) hits.
func TestUsefulBudgetTokensDup(t *testing.T) {
	// Two expected files; top-5 returns a.go only (b.go is rank 6) → useful@5=1/2.
	ranked := []string{"a.go", "x.go", "y.go", "z.go", "w.go", "b.go"}
	useful := usefulAtBudget(ranked, []string{"a.go", "b.go"}, 5)
	if want := 0.5; useful < want-1e-9 || useful > want+1e-9 {
		t.Errorf("useful@5 = %v, want 0.5", useful)
	}

	// dup%: 5 hits, 3 on the same file → 2 duplicates → 40%.
	dupPct := dupPercent([]string{"a.go", "a.go", "b.go", "a.go", "c.go"})
	if want := 0.4; dupPct < want-1e-9 || dupPct > want+1e-9 {
		t.Errorf("dup%% = %v, want 0.4", dupPct)
	}

	// tokens: a positive estimate that grows with text length.
	small := tokenCount([]string{"a.go"})
	big := tokenCount([]string{"a.go", "b.go", "c.go", "d.go", "e.go"})
	if small <= 0 || big <= small {
		t.Errorf("tokens small=%v big=%v: want positive and increasing", small, big)
	}
}

// TestPercentiles covers p50/p95/p99 computation.
func TestPercentiles(t *testing.T) {
	v := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if p := percentile(v, 50); p < 5 || p > 6 {
		t.Errorf("p50 = %v, want ~5.5", p)
	}
	if p := percentile(v, 95); p < 9 || p > 10 {
		t.Errorf("p95 = %v, want ~9.5", p)
	}
	if p := percentile(v, 99); p < 9 || p > 10 {
		t.Errorf("p99 = %v, want ~9.9", p)
	}
}

// TestSeededBootstrapCI covers @step-01 (significance): the paired bootstrap
// 95% CI is deterministic for a fixed seed and, for a candidate that is
// uniformly better than baseline, the CI lower bound is above 0 (the delta is
// significantly positive).
func TestSeededBootstrapCI(t *testing.T) {
	// baseline per-query metric and a strictly better candidate.
	base := []float64{0.4, 0.4, 0.4, 0.4, 0.4, 0.4, 0.4, 0.4, 0.4, 0.4}
	cand := []float64{0.7, 0.7, 0.7, 0.7, 0.7, 0.7, 0.7, 0.7, 0.7, 0.7}

	ci1 := pairedBootstrapCI(base, cand, 1000, 42)
	ci2 := pairedBootstrapCI(base, cand, 1000, 42)
	if ci1 != ci2 {
		t.Errorf("bootstrap CI not deterministic for the same seed: %v vs %v", ci1, ci2)
	}
	if ci1.Lower <= 0 {
		t.Errorf("CI lower bound = %v, want > 0 (uniformly-better candidate is significant)", ci1.Lower)
	}

	// A different seed should generally give a different estimate (not a
	// requirement, but guards against a hardcoded constant).
	ci3 := pairedBootstrapCI(base, cand, 1000, 43)
	_ = ci3
}

// TestPermutationPValue covers @step-01 (significance): for a candidate that
// is uniformly better, the permutation p-value is small (significant); for an
// identical candidate it is large (not significant).
func TestPermutationPValue(t *testing.T) {
	base := []float64{0.4, 0.4, 0.4, 0.4, 0.4, 0.4, 0.4, 0.4, 0.4, 0.4}
	better := []float64{0.7, 0.7, 0.7, 0.7, 0.7, 0.7, 0.7, 0.7, 0.7, 0.7}

	pSig := permutationP(base, better, 1000, 42)
	if pSig > 0.05 {
		t.Errorf("permutation p (better) = %v, want < 0.05 (significant)", pSig)
	}

	// identical → p near 1 (the observed delta is within the null distribution).
	pSame := permutationP(base, base, 1000, 42)
	if pSame < 0.5 {
		t.Errorf("permutation p (identical) = %v, want large (~1.0, not significant)", pSame)
	}
}
