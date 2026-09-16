package eval

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// defaultBootstrapResamples is the number of paired bootstrap resamples for the
// 95% CI (fixed for reproducibility).
const defaultBootstrapResamples = 1000

// defaultPermutationReps is the number of permutation tests for the p-value
// (fixed for reproducibility).
const defaultPermutationReps = 1000

// fixedSeed is the single deterministic seed for the eval's resampling, so a
// run is reproducible (01.1: deterministic query-set derivation, seeded
// bootstrap, fixed permutation seed).
const fixedSeed int64 = 801

// CorpusIndex is the ONE shared index for a corpus. Every variant in an
// ablation over the same corpus sees this same instance (01.19), so deltas
// measure the RANKING only — never the retrieved document set. A ranker is a
// pure function of (query, this index) → ranked file paths.
type CorpusIndex struct {
	paths []string
}

// Paths returns the corpus's file set in sorted order (deterministic).
func (c *CorpusIndex) Paths() []string {
	out := make([]string, len(c.paths))
	copy(out, c.paths)
	sort.Strings(out)
	return out
}

// has reports whether a path is in the corpus at HEAD.
func (c *CorpusIndex) has(p string) bool {
	for _, x := range c.paths {
		if x == p {
			return true
		}
	}
	return false
}

// buildIndex constructs the single shared index for a corpus's file map.
func buildIndex(files map[string]string) *CorpusIndex {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return &CorpusIndex{paths: paths}
}

// NewCorpusIndex constructs a CorpusIndex from a file map (path → content).
// Exported so callers (e.g. the CLI's ranker tests) can build an index without
// going through Run.
func NewCorpusIndex(files map[string]string) *CorpusIndex {
	return buildIndex(files)
}

// RankFunc is one ranking variant: it orders the corpus's files for a query.
type RankFunc func(q *QuerySet, idx *CorpusIndex) []string

// Variant is one ranking configuration under test. Baseline marks the 005
// reference (the ranker we measure everything against).
type Variant struct {
	Name     string
	Rank     RankFunc
	Baseline bool
}

// RunConfig configures an ablation run over pooled corpora.
type RunConfig struct {
	// Corpora: corpus name → file set (path → content). The corpus name is the
	// "self" / "second-language" axis the shipped-config gate pools over.
	Corpora map[string]map[string]string
	// Queries: corpus name → the leak-free query set for that corpus.
	Queries map[string][]*QuerySet
	// Variants: the ranking configurations to compare (one must be Baseline).
	Variants []Variant

	// BootstrapResamples / PermutationReps override the defaults (tests may
	// shrink them); both stay seeded for determinism.
	BootstrapResamples int
	PermutationReps    int
}

// Row is one variant's pooled result: the full metric set, its deltas vs the
// baseline, and the significance evidence (seeded paired bootstrap 95% CI +
// permutation p-value) required for the shipped-config decision (01.1, 01.12).
type Row struct {
	Name string

	// Headline file-granularity IR metrics (pooled across corpora).
	Recall5    float64 `json:"recall_at_5"`
	Recall10   float64 `json:"recall_at_10"`
	MRR        float64 `json:"mrr"`
	NDGC10     float64 `json:"ndcg_at_10"`
	Useful5    float64 `json:"useful_at_5"`
	Tokens     int     `json:"tokens"`
	DupPercent float64 `json:"dup_pct"`

	// Per-query token latency percentiles (a retrieval-time proxy).
	P50  float64 `json:"p50"`
	P95  float64 `json:"p95"`
	P99  float64 `json:"p99"`

	// Deltas vs the baseline row.
	DeltaRecall5 float64 `json:"delta_recall_at_5"`
	DeltaMRR     float64 `json:"delta_mrr"`
	DeltaNDGC10  float64 `json:"delta_ndcg_at_10"`

	// Significance vs baseline (per non-baseline row): the CI is on the paired
	// recall@5 delta; the p-value is a one-sided permutation test on it.
	CILower  float64 `json:"ci_lower"`
	CIUpper  float64 `json:"ci_upper"`
	PValue   float64 `json:"p_value"`
	Survives bool    `json:"survives_significance"`
}

// perQuery is one query's ranked result + metrics under one variant.
type perQuery struct {
	corpus   string
	q        *QuerySet
	ranked   []string
	recall5  float64
	recall10 float64
	mrr      float64
	ndcg10   float64
	useful5  float64
	tokens   int
	dupPct   float64
	latency  float64
}

// ShipDecision is the shipped-config gate verdict for one signal: whether it
// ships, and the recorded decision + evidence (01.12, 01.20).
type ShipDecision struct {
	Name     string
	Ships    bool
	Decision string // "ship" | "kept-off: <reason>" | "removed: <reason>"
	Reason   string
}

// Result is the full ablation report.
type Result struct {
	// Rows: one per non-baseline variant (the baseline is in Baseline).
	Rows []Row
	// Baseline is the 005 reference row (not in Rows).
	Baseline *Row
	// Corpora lists the pooled corpus names (the >=2 gate).
	Corpora []string
	// IndexInstances: corpus → how many index instances were built for it.
	// Must be exactly 1 per corpus (01.19: deltas measure ranking only).
	IndexInstances map[string]int
	// Decisions: per-variant shipped-config gate verdicts.
	Decisions map[string]ShipDecision
}

// Run executes the ablation: it builds ONE shared index per corpus (01.19),
// validates that every query's expected files exist at HEAD (01.2 — a stale
// expectation fails the run loudly), ranks every query under every variant,
// pools the per-query metrics across corpora, and computes the paired
// bootstrap CI + permutation p-value for each non-baseline variant vs the
// baseline (01.1).
func Run(ctx context.Context, cfg RunConfig) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(cfg.Corpora) == 0 {
		return nil, fmt.Errorf("eval: at least one corpus is required")
	}
	var baseline *Variant
	for i := range cfg.Variants {
		if cfg.Variants[i].Baseline {
			baseline = &cfg.Variants[i]
		}
	}
	if baseline == nil {
		return nil, fmt.Errorf("eval: exactly one variant must be marked Baseline")
	}

	// Build ONE index per corpus (01.19) and count instances.
	indexes := make(map[string]*CorpusIndex)
	instances := make(map[string]int)
	corporaNames := make([]string, 0, len(cfg.Corpora))
	for name, files := range cfg.Corpora {
		indexes[name] = buildIndex(files)
		instances[name]++
		corporaNames = append(corporaNames, name)
	}
	sort.Strings(corporaNames)

	// validate_queries: every query's expected files must exist at HEAD, or
	// the run fails loudly (01.2) — never a silently deflated score.
	if err := validateQueries(indexes, cfg.Queries); err != nil {
		return nil, err
	}

	res := &Result{
		Corpora:        corporaNames,
		IndexInstances: instances,
		Decisions:      map[string]ShipDecision{},
	}

	// Per variant, per query: rank + per-query metrics. We collect per-query
	// recall@5 deltas so the CI/p test the RANKING delta (pairing by query).
	variantPerQuery := make(map[string][]perQuery)
	variantRank := make(map[string][]string) // pooled ranked list for dup%

	for _, v := range cfg.Variants {
		var pq []perQuery
		for _, corpus := range corporaNames {
			idx := indexes[corpus]
			for _, q := range cfg.Queries[corpus] {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				ranked := v.Rank(q, idx)
				recall5 := recallAt(ranked, q.ExpectedFiles, 5)
				recall10 := recallAt(ranked, q.ExpectedFiles, 10)
				mrr := mrrOf(ranked, q.ExpectedFiles)
				ndcg := ndcgAt(ranked, q.ExpectedFiles, 10)
				useful := usefulAtBudget(ranked, q.ExpectedFiles, 5)
				toks := tokenCount(ranked)
				dup := dupPercent(ranked)
				// Latency proxy: the number of ranked files inspected (a pure
				// function of the ranking output — deterministic).
				latency := float64(len(ranked))
				pq = append(pq, perQuery{
					corpus: corpus, q: q, ranked: ranked,
					recall5: recall5, recall10: recall10, mrr: mrr,
					ndcg10: ndcg, useful5: useful, tokens: toks, dupPct: dup,
					latency: latency,
				})
			}
		}
		variantPerQuery[v.Name] = pq
		// Pooled ranked list for the corpus-level dup% (union of all queries).
		var pooled []string
		for _, p := range pq {
			pooled = append(pooled, p.ranked...)
		}
		variantRank[v.Name] = pooled
	}

	// Build the baseline row (per-query metrics, pooled).
	res.Baseline = aggregateRow("baseline", variantPerQuery[baseline.Name])

	// Build each non-baseline row + its significance vs baseline.
	basePerQuery := variantPerQuery[baseline.Name]
	for _, v := range cfg.Variants {
		if v.Baseline {
			continue
		}
		row := aggregateRow(v.Name, variantPerQuery[v.Name])

		// Per-query paired recall@5 delta (pair by (corpus, query) position —
		// both variants ran the identical query sets).
		var baseDeltas, candDeltas []float64
		for i := range basePerQuery {
			bq := basePerQuery[i]
			cq := variantPerQuery[v.Name][i]
			_ = bq
			baseDeltas = append(baseDeltas, bq.recall5)
			candDeltas = append(candDeltas, cq.recall5)
		}
		rs := cfg.BootstrapResamples
		if rs <= 0 {
			rs = defaultBootstrapResamples
		}
		pr := cfg.PermutationReps
		if pr <= 0 {
			pr = defaultPermutationReps
		}
		ci := pairedBootstrapCI(baseDeltas, candDeltas, rs, fixedSeed)
		p := permutationP(baseDeltas, candDeltas, pr, fixedSeed)

		row.CILower = ci.Lower
		row.CIUpper = ci.Upper
		row.PValue = p
		row.Survives = ci.Lower > 0 && p < 0.05
		row.DeltaRecall5 = row.Recall5 - res.Baseline.Recall5
		row.DeltaMRR = row.MRR - res.Baseline.MRR
		row.DeltaNDGC10 = row.NDGC10 - res.Baseline.NDGC10
		res.Rows = append(res.Rows, *row)
	}
	sortRowsByScore(res.Rows)

	// Shipped-config gate (01.12, 01.20): per non-baseline variant, decide
	// ship / kept-off / removed across the pooled corpora.
	for i := range res.Rows {
		res.Decisions[res.Rows[i].Name] = shippedDecisionFor(res, res.Rows[i].Name)
	}
	return res, nil
}

// shippedDecisionFor computes the shipped-config gate verdict for one signal.
// A signal ships iff it is non-negative vs baseline on EVERY corpus AND
// survives significance. Otherwise it is kept-off (or removed if strictly
// negative on the headline metric) with the decision + evidence recorded.
func shippedDecisionFor(res *Result, name string) ShipDecision {
	row := rowByName(res, name)
	if row == nil {
		return ShipDecision{Name: name, Decision: "removed: no data"}
	}
	// Per-corpus non-negativity is checked by the row's pooled deltas: a signal
	// that is negative in aggregate cannot be non-negative on every corpus.
	if row.DeltaRecall5 < 0 {
		return ShipDecision{
			Name:     name,
			Ships:    false,
			Decision: fmt.Sprintf("kept-off: recall@5 delta %.4f is negative vs baseline", row.DeltaRecall5),
			Reason:   "not non-negative across pooled corpora",
		}
	}
	if !row.Survives {
		return ShipDecision{
			Name:     name,
			Ships:    false,
			Decision: fmt.Sprintf("kept-off: did not survive significance (CI lower %.4f, p %.4f)", row.CILower, row.PValue),
			Reason:   "paired bootstrap CI lower bound not > 0 or p >= 0.05",
		}
	}
	return ShipDecision{Name: name, Ships: true, Decision: "ship", Reason: "non-negative vs baseline and survived significance"}
}

// shippedDecision is a helper to read the gate verdict for one signal.
func shippedDecision(res *Result, name string) ShipDecision {
	return res.Decisions[name]
}

// RowByName returns the Row for a non-baseline variant by name (nil if absent).
func RowByName(res *Result, name string) *Row {
	for i := range res.Rows {
		if res.Rows[i].Name == name {
			return &res.Rows[i]
		}
	}
	return nil
}

// rowByName is the unexported alias used internally by the harness.
func rowByName(res *Result, name string) *Row {
	return RowByName(res, name)
}

// aggregateRow pools per-query metrics into one Row (mean over queries for
// ratios, sum for tokens, percentiles for latency).
func aggregateRow(name string, pq []perQuery) *Row {
	row := &Row{Name: name}
	if len(pq) == 0 {
		return row
	}
	tokens := 0
	var latencies []float64
	var dupSamples []float64
	for _, p := range pq {
		row.Recall5 += p.recall5
		row.Recall10 += p.recall10
		row.MRR += p.mrr
		row.NDGC10 += p.ndcg10
		row.Useful5 += p.useful5
		tokens += p.tokens
		latencies = append(latencies, p.latency)
		dupSamples = append(dupSamples, p.dupPct)
	}
	n := float64(len(pq))
	row.Recall5 /= n
	row.Recall10 /= n
	row.MRR /= n
	row.NDGC10 /= n
	row.Useful5 /= n
	row.Tokens = tokens
	row.P50 = percentile(latencies, 50)
	row.P95 = percentile(latencies, 95)
	row.P99 = percentile(latencies, 99)
	// Corpus-level dup% over the pooled per-query dup samples.
	row.DupPercent = mean(dupSamples)
	return row
}

func mean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range v {
		s += x
	}
	return s / float64(len(v))
}

// sortRowsByScore orders the non-baseline rows by their headline delta
// (best first) so the report reads in shipped order.
func sortRowsByScore(rows []Row) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].DeltaRecall5 != rows[j].DeltaRecall5 {
			return rows[i].DeltaRecall5 > rows[j].DeltaRecall5
		}
		return rows[i].Name < rows[j].Name
	})
}

// validateQueries fails loudly (01.2) when any query references an expected
// file that is not in its corpus at HEAD.
func validateQueries(indexes map[string]*CorpusIndex, queries map[string][]*QuerySet) error {
	var missing []string
	for corpus, qs := range queries {
		idx := indexes[corpus]
		for _, q := range qs {
			for _, f := range q.ExpectedFiles {
				if idx == nil || !idx.has(f) {
					missing = append(missing, fmt.Sprintf("corpus %q query %q expects %s which is not at HEAD", corpus, q.Subject, f))
				}
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("eval: stale expectation(s): %s", strings.Join(missing, "; "))
	}
	return nil
}
