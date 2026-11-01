package eval

import (
	"math"
	"sort"
)

// CIs is a 95% confidence interval [lower, upper] for a statistic.
type CIs struct {
	Lower float64
	Upper float64
}

// recallAt is the fraction of expected files present in the top-k ranked list.
// Expected files are the ground-truth files; ranked is the candidate file list
// (file granularity: a path repeated is still one file).
func recallAt(ranked, expected []string, k int) float64 {
	if len(expected) == 0 {
		return 0
	}
	if k <= 0 {
		k = len(ranked)
	}
	if k > len(ranked) {
		k = len(ranked)
	}
	top := make(map[string]bool, k)
	for i := 0; i < k; i++ {
		top[ranked[i]] = true
	}
	hit := 0
	for _, e := range expected {
		if top[e] {
			hit++
		}
	}
	return float64(hit) / float64(len(expected))
}

// mrrOf is the reciprocal rank of the first expected file in the ranked list.
func mrrOf(ranked, expected []string) float64 {
	want := make(map[string]bool, len(expected))
	for _, e := range expected {
		want[e] = true
	}
	for i, r := range ranked {
		if want[r] {
			return 1.0 / float64(i+1)
		}
	}
	return 0
}

// ndcgAt is the normalized discounted cumulative gain at k for binary
// relevance (an expected file is relevant).
func ndcgAt(ranked, expected []string, k int) float64 {
	want := make(map[string]bool, len(expected))
	for _, e := range expected {
		want[e] = true
	}
	if k <= 0 {
		k = len(ranked)
	}
	if k > len(ranked) {
		k = len(ranked)
	}
	dcg := 0.0
	for i := 0; i < k; i++ {
		if want[ranked[i]] {
			dcg += 1.0 / math.Log2(float64(i+2))
		}
	}
	if dcg == 0 {
		return 0
	}
	// Ideal DCG: all relevant docs in the top ranks (as many as we have).
	ideal := 0.0
	rel := len(expected)
	if rel > k {
		rel = k
	}
	for i := 0; i < rel; i++ {
		ideal += 1.0 / math.Log2(float64(i+2))
	}
	if ideal == 0 {
		return 0
	}
	return dcg / ideal
}

// usefulAtBudget counts the fraction of expected files that appear in the
// top-budget ranked list (a budgeted "how much of the answer made the cut").
func usefulAtBudget(ranked, expected []string, budget int) float64 {
	if len(expected) == 0 {
		return 0
	}
	if budget <= 0 {
		budget = len(ranked)
	}
	if budget > len(ranked) {
		budget = len(ranked)
	}
	top := make(map[string]bool, budget)
	for i := 0; i < budget; i++ {
		top[ranked[i]] = true
	}
	hit := 0
	for _, e := range expected {
		if top[e] {
			hit++
		}
	}
	return float64(hit) / float64(len(expected))
}

// tokenCount is a cheap token estimate for the returned file set (words across
// the distinct files). Deterministic and monotonic in content size.
func tokenCount(ranked []string) int {
	seen := make(map[string]bool)
	n := 0
	for _, r := range ranked {
		if seen[r] {
			continue
		}
		seen[r] = true
		n += len(splitWords(r))
	}
	return n
}

func splitWords(s string) []string {
	var out []string
	start := -1
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '/' || c == '.' || c == '-' || c == '_' {
			if start >= 0 {
				out = append(out, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		out = append(out, s[start:])
	}
	return out
}

// dupPercent is the fraction of ranked entries that are near-duplicates — the
// same file appearing more than once. 5 hits, 3 on one file → 2 dups → 0.4.
func dupPercent(ranked []string) float64 {
	if len(ranked) == 0 {
		return 0
	}
	counts := make(map[string]int)
	for _, r := range ranked {
		counts[r]++
	}
	dups := 0
	for _, c := range counts {
		dups += c - 1
	}
	return float64(dups) / float64(len(ranked))
}

// percentile is the linearly-interpolated p-th percentile of v. v need not be
// sorted.
func percentile(v []float64, p float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	if p <= 0 {
		return s[0]
	}
	if p >= 100 {
		return s[len(s)-1]
	}
	rank := (p / 100) * float64(len(s)-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	if lo == hi {
		return s[lo]
	}
	frac := rank - float64(lo)
	return s[lo]*(1-frac) + s[hi]*frac
}

// pairedBootstrapCI is a seeded, paired bootstrap 95% CI over the per-query
// delta (candidate - baseline). It resamples the paired indices together (the
// pairing is what makes it a "paired" statistic) so the CI reflects the
// ranking delta, not an aggregate. Deterministic for a fixed seed.
func pairedBootstrapCI(baseline, candidate []float64, resamples int, seed int64) CIs {
	n := len(baseline)
	if n == 0 {
		return CIs{}
	}
	if len(candidate) != n {
		// Mismatched pair counts: trim to the shorter (defensive, should not
		// happen from the harness which pairs by query).
		m := n
		if len(candidate) < m {
			m = len(candidate)
		}
		baseline = baseline[:m]
		candidate = candidate[:m]
		n = m
	}
	if resamples <= 0 {
		resamples = 1000
	}
	rng := newRNG(seed)
	var deltas []float64
	for s := 0; s < resamples; s++ {
		sum := 0.0
		for i := 0; i < n; i++ {
			idx := int(rng.next() % uint64(n))
			sum += candidate[idx] - baseline[idx]
		}
		deltas = append(deltas, sum/float64(n))
	}
	obs := 0.0
	for i := 0; i < n; i++ {
		obs += candidate[i] - baseline[i]
	}
	obs /= float64(n)
	// Percentile CI over the resampled deltas, shifted so the interval is
	// centered on the observed delta (bias-corrected).
	shift := obs - percentile(deltas, 50)
	return CIs{
		Lower: percentile(deltas, 2.5) + shift,
		Upper: percentile(deltas, 97.5) + shift,
	}
}

// permutationP is a seeded permutation-test p-value for the paired delta.
// Under the null (the candidate adds nothing) we randomly swap which arm is
// "candidate" vs "baseline" for each query and count how often the resulting
// mean delta is at least as extreme as the observed (one-sided, positive
// delta). p near 1 means not significant; p small means significant.
func permutationP(baseline, candidate []float64, reps int, seed int64) float64 {
	n := len(baseline)
	if n == 0 {
		return 1
	}
	if len(candidate) != n {
		m := n
		if len(candidate) < m {
			m = len(candidate)
		}
		baseline = baseline[:m]
		candidate = candidate[:m]
		n = m
	}
	if reps <= 0 {
		reps = 1000
	}
	obs := 0.0
	for i := 0; i < n; i++ {
		obs += candidate[i] - baseline[i]
	}
	obs /= float64(n)
	rng := newRNG(seed)
	extreme := 0
	for r := 0; r < reps; r++ {
		sum := 0.0
		for i := 0; i < n; i++ {
			// Random sign flip per query: the null has no systematic direction.
			if rng.next()&1 == 0 {
				sum += candidate[i] - baseline[i]
			} else {
				sum += baseline[i] - candidate[i]
			}
		}
		perm := sum / float64(n)
		if perm >= obs {
			extreme++
		}
	}
	return float64(extreme+1) / float64(reps+1)
}

// rng is a small deterministic PRNG (PCG-like xorshift) so the bootstrap and
// permutation tests are reproducible for a fixed seed without importing a
// non-pure-Go dependency.
type rng struct {
	state uint64
}

func newRNG(seed int64) *rng {
	if seed == 0 {
		seed = 1
	}
	// SplitMix64 to seed the state from an arbitrary 64-bit value.
	s := uint64(seed)
	s += 0x9e3779b97f4a7c15
	s = (s ^ (s >> 30)) * 0xbf58476d1ce4e5b9
	s = (s ^ (s >> 27)) * 0x94d049bb133111eb
	s ^= s >> 31
	if s == 0 {
		s = 1
	}
	return &rng{state: s}
}

// next is a PCG-XSH-RR 64-bit output step.
func (r *rng) next() uint64 {
	prev := r.state
	r.state = prev*6364136223846793005 + 1442695040888963407
	x := (prev >> 18) ^ prev
	x = (x ^ (x >> 27)) * 0x5851f42d4c957f2d
	return x ^ (x >> 31)
}
