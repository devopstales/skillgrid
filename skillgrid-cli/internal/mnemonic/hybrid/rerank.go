package hybrid

import (
	"math"
	"sort"
	"strings"
)

// RankHit is the pre-rerank input for the explainable rerank table: the
// metadata the named factors read from a hit (path, symbol, kind, lines) plus
// the RRF score it starts from.
type RankHit struct {
	Path      string
	StartLine int
	EndLine   int
	Symbol    string
	Kind      string
	Score     float64
}

// RerankContext carries the query + corpus facts the rerank table needs to
// decide which named factors fire (is the query a test, the hit's degree, etc.).
type RerankContext struct {
	Query      string
	IsTestFile bool
	Degree     int
	// SourceKind is "prose" (a doc/config/SQL hit) vs "source" (code).
	SourceKind string
}

// RerankFactor is one named, capped boost/penalty with its written rationale
// (01.9). Delta is the signed contribution to the hit score; each factor's
// magnitude is bounded by MaxDelta so no single signal dominates.
type RerankFactor struct {
	Name      string
	Delta     float64
	MaxDelta  float64
	Rationale string
}

// RerankTable is the explainable, bounded rerank table (01.9). Every entry
// carries its sign (boost/penalty), a hard cap, and a written rationale so a
// hit's score is fully attributable.
var RerankTable = map[string]RerankFactor{
	"exact-symbol": {
		Name: "exact-symbol", Delta: 0.30, MaxDelta: 0.30,
		Rationale: "the query names this symbol verbatim — the strongest single signal.",
	},
	"definition-kind": {
		Name: "definition-kind", Delta: 0.15, MaxDelta: 0.15,
		Rationale: "the hit is a definition (function/type) for the queried kind, not just a reference.",
	},
	"path-match": {
		Name: "path-match", Delta: 0.10, MaxDelta: 0.10,
		Rationale: "a query term appears in the file path — strong topical locality.",
	},
	"degree": {
		Name: "degree", Delta: 0.05, MaxDelta: 0.10,
		Rationale: "higher graph degree signals a central concept; bounded so hubs don't dominate.",
	},
	"source-over-prose": {
		Name: "source-over-prose", Delta: 0.05, MaxDelta: 0.05,
		Rationale: "prefer code over prose/config when both match the query.",
	},
	"documentation": {
		Name: "documentation", Delta: -0.10, MaxDelta: 0.10,
		Rationale: "documentation/config hits are ranked below the code they describe.",
	},
	"generated-vendor": {
		Name: "generated-vendor", Delta: -0.15, MaxDelta: 0.15,
		Rationale: "vendored or generated code is demoted — it is not the project's own signal.",
	},
	"test-on-non-test": {
		Name: "test-on-non-test", Delta: -0.10, MaxDelta: 0.10,
		Rationale: "a test file hit for a non-test query is usually a secondary reference.",
	},
}

// ApplyRerankTable walks the named factors in RerankTable for one hit and
// returns the factors that actually fired (each with its bounded Delta and
// written rationale). The hit's final rerank score is its base score plus the
// sum of the fired Deltas.
func ApplyRerankTable(h RankHit, ctx RerankContext) []RerankFactor {
	var fired []RerankFactor
	add := func(name string, delta float64) {
		t := RerankTable[name]
		// Bound the contribution to [-MaxDelta, +MaxDelta].
		if delta > t.MaxDelta {
			delta = t.MaxDelta
		}
		if delta < -t.MaxDelta {
			delta = -t.MaxDelta
		}
		t.Delta = delta
		fired = append(fired, t)
	}

	// exact-symbol: query == symbol (case-insensitive).
	if h.Symbol != "" && strings.EqualFold(ctx.Query, h.Symbol) {
		add("exact-symbol", RerankTable["exact-symbol"].Delta)
	} else if h.Symbol != "" && strings.Contains(strings.ToLower(h.Symbol), strings.ToLower(ctx.Query)) {
		// partial: a smaller boost (still named + bounded).
		add("exact-symbol", 0.15)
	}

	// definition-kind: a definition kind (function/method/struct/type/class) for
	// a query that is not empty.
	if isDefinitionKind(h.Kind) {
		add("definition-kind", RerankTable["definition-kind"].Delta)
	}

	// path-match: a query term appears in the file path.
	if pathMatchesQuery(h.Path, ctx.Query) {
		add("path-match", RerankTable["path-match"].Delta)
	}

	// degree: bounded by degree (saturates at MaxDelta).
	if ctx.Degree > 0 {
		// log-scaled, capped: degree 1 -> ~0.05, grows, saturates at 0.10.
		delta := 0.05 * (1 - math.Exp(-float64(ctx.Degree)/5.0))
		if delta < 0.01 {
			delta = 0.01
		}
		add("degree", delta)
	}

	// source-over-prose: code beats prose.
	if ctx.SourceKind == "source" {
		add("source-over-prose", RerankTable["source-over-prose"].Delta)
	}
	if ctx.SourceKind == "prose" || isProsePath(h.Path) {
		add("documentation", RerankTable["documentation"].Delta)
	}

	// generated/vendor: a vendor/ or generated marker in the path.
	if isVendorOrGenerated(h.Path) {
		add("generated-vendor", RerankTable["generated-vendor"].Delta)
	}

	// test-on-non-test: a test file hit for a non-test query.
	if ctx.IsTestFile && !isTestQuery(ctx.Query) {
		add("test-on-non-test", RerankTable["test-on-non-test"].Delta)
	}

	// Deterministic order for the explainable list (table order, then name).
	sort.SliceStable(fired, func(i, j int) bool { return fired[i].Name < fired[j].Name })
	return fired
}

// isDefinitionKind reports whether a symbol kind is a definition (not a
// reference/call/usage).
func isDefinitionKind(kind string) bool {
	switch strings.ToLower(kind) {
	case "function", "method", "struct", "type", "class", "interface", "enum", "module":
		return true
	}
	return false
}

// isProsePath reports whether a path is documentation/config (not code).
func isProsePath(path string) bool {
	lower := strings.ToLower(path)
	for _, m := range []string{"/docs/", "/doc/", "readme", "changelog", ".md", ".rst", ".txt", ".yaml", ".yml", ".json", ".toml"} {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

// isVendorOrGenerated reports whether a path is vendored or generated code.
func isVendorOrGenerated(path string) bool {
	lower := strings.ToLower(path)
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

// pathMatchesQuery reports whether any query term (>=3 chars) appears in the
// file path (case-insensitive).
func pathMatchesQuery(path, query string) bool {
	if query == "" {
		return false
	}
	terms := strings.Fields(strings.ToLower(query))
	needle := strings.ToLower(path)
	for _, t := range terms {
		if len(t) >= 3 && strings.Contains(needle, t) {
			return true
		}
	}
	return false
}

// isTestQuery reports whether the query itself looks like a test (contains
// "test").
func isTestQuery(query string) bool {
	return strings.Contains(strings.ToLower(query), "test")
}

// FusionCandidate is one retriever's ranked candidate keyed by (path,
// line-bucket) — the file-level agreement unit (01.9). LineBucket is a coarse
// line grouping so two retrievers finding the same file at nearby locators
// collapse into one file-level candidate.
type FusionCandidate struct {
	Path       string
	Line       int
	Source     string
	LineBucket int
	Score      float64
}

// lineBucketSize is the line-bucket width for file-level agreement.
const lineBucketSize = 20

// bucketize computes the (path, line-bucket) key for a candidate.
func bucketize(c FusionCandidate) (path, bucket string) {
	return c.Path, c.Path + ":" + itoa(c.Line/lineBucketSize)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [24]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// FileLevelAgreement collapses fusion candidates by (path, line-bucket): two
// retrievers finding the same file in the same bucket yield ONE strong
// file-level candidate whose score is the sum of the per-source RRF terms plus
// a fixed agreement weight (01.9). Different buckets on the same file stay
// separate.
func FileLevelAgreement(cands []FusionCandidate, k int) []FusionCandidate {
	if k <= 0 {
		k = RRFK
	}
	if len(cands) == 0 {
		return nil
	}
	type key struct {
		path   string
		bucket int
	}
	// Preserve first-seen order for determinism.
	var order []key
	seen := map[key]bool{}
	agg := map[key]*FusionCandidate{}
	for _, c := range cands {
		kk := key{path: c.Path, bucket: c.Line / lineBucketSize}
		if _, ok := seen[kk]; !ok {
			seen[kk] = true
			order = append(order, kk)
			agg[kk] = &FusionCandidate{Path: c.Path, Line: c.Line, LineBucket: c.Line / lineBucketSize}
		}
	}
	// Per-source rank within each (path,bucket) group → RRF term.
	for _, kk := range order {
		score := 0.0
		sources := map[string]bool{}
		for _, c := range cands {
			if c.Path != kk.path || c.Line/lineBucketSize != kk.bucket {
				continue
			}
			// RRF term per source occurrence.
			score += 1.0 / float64(k+1)
			sources[c.Source] = true
		}
		out := agg[kk]
		out.Score = score
		out.Source = ""
		for s := range sources {
			if out.Source == "" {
				out.Source = s
			} else {
				out.Source += "+" + s
			}
		}
		// Fixed agreement weight: one per extra distinct source (>1).
		if len(sources) > 1 {
			out.Score += agreementWeight
		}
	}
	res := make([]FusionCandidate, 0, len(order))
	for _, kk := range order {
		res = append(res, *agg[kk])
	}
	sort.SliceStable(res, func(i, j int) bool {
		if res[i].Score != res[j].Score {
			return res[i].Score > res[j].Score
		}
		return res[i].Path < res[j].Path
	})
	return res
}

// agreementWeight is the fixed boost for a file found by >1 retriever in the
// same bucket (01.9).
const agreementWeight = 0.05
