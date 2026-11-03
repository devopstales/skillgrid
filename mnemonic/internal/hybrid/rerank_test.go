package hybrid

import (
	"testing"
)

// TestRerankFactorsCarryRationale covers @step-01 (Scenario: Search response
// carries confidence action rerank reasons and fallbacks): each hit carries a
// named, capped boost/penalty factor with a written rationale. The factor set
// is exactly the explainable table (exact-symbol, definition-kind, path-match,
// degree bounded, source-over-prose, documentation, generated/vendor,
// test-on-non-test) and every applied factor has a non-empty rationale.
func TestRerankFactorsCarryRationale(t *testing.T) {
	h := RankHit{
		Path:      "internal/http/client.go",
		StartLine: 10,
		EndLine:   20,
		Symbol:    "NewClient",
		Kind:      "function",
	}
	factors := ApplyRerankTable(h, RerankContext{
		Query:      "NewClient",
		IsTestFile: false,
		Degree:     3,
	})
	// Every applied factor must have a written rationale (explainable).
	for _, f := range factors {
		if f.Rationale == "" {
			t.Errorf("factor %q has no rationale", f.Name)
		}
	}
	// The expected named factors exist in the table (the contract, not a
	// specific application).
	for _, name := range []string{
		"exact-symbol", "definition-kind", "path-match", "degree",
		"source-over-prose", "documentation", "generated-vendor", "test-on-non-test",
	} {
		if _, ok := RerankTable[name]; !ok {
			t.Errorf("RerankTable is missing the %q factor", name)
		}
	}
	// exact-symbol must be applied (query == symbol) as a boost.
	if !hasFactor(factors, "exact-symbol") {
		t.Errorf("exact-symbol boost not applied for query==symbol, got %v", factors)
	}
}

// TestRerankTableIsBounded covers @step-01: the degree factor is bounded — a
// very high degree contributes a capped amount, never an unbounded one.
func TestRerankTableIsBounded(t *testing.T) {
	lo := ApplyRerankTable(RankHit{Path: "a.go", Symbol: "X", Kind: "function"}, RerankContext{Query: "X", Degree: 1})
	hi := ApplyRerankTable(RankHit{Path: "a.go", Symbol: "X", Kind: "function"}, RerankContext{Query: "X", Degree: 1000})
	dLo := sumFactorDelta(lo)
	dHi := sumFactorDelta(hi)
	// The high-degree boost must be strictly bounded: it grows, but the total
	// factor delta for the huge degree must be capped (not equal to the degree).
	if dHi >= 100 {
		t.Errorf("degree boost not bounded: delta for degree 1000 = %v", dHi)
	}
	if dHi <= dLo {
		t.Errorf("higher degree should boost more (lo=%v hi=%v)", dLo, dHi)
	}
}

// TestRerankPenaltiesApplied covers @step-01: documentation and generated/vendor
// paths are penalized, and a test-on-non-test hit is penalized.
func TestRerankPenaltiesApplied(t *testing.T) {
	// A docs file is penalized.
	doc := ApplyRerankTable(RankHit{Path: "docs/README.md", Kind: "prose"}, RerankContext{Query: "foo"})
	if !hasFactor(doc, "documentation") {
		t.Errorf("documentation penalty not applied for a docs file: %v", doc)
	}
	// A vendored/generated file is penalized.
	vend := ApplyRerankTable(RankHit{Path: "vendor/lib/x.go"}, RerankContext{Query: "foo"})
	if !hasFactor(vend, "generated-vendor") {
		t.Errorf("generated-vendor penalty not applied for a vendor file: %v", vend)
	}
	// A test file hit for a non-test query is penalized.
	testHit := ApplyRerankTable(RankHit{Path: "a_test.go"}, RerankContext{Query: "foo", IsTestFile: true})
	if !hasFactor(testHit, "test-on-non-test") {
		t.Errorf("test-on-non-test penalty not applied: %v", testHit)
	}
}

// TestRerankAgreement covers @step-01 (Scenario: file-level RRF agreement):
// fusion candidates keyed by (path, line-bucket); two retrievers finding the
// SAME file at DIFFERENT locators yield ONE strong file-level candidate under a
// fixed agreement weight.
func TestRerankAgreement(t *testing.T) {
	// Two retrievers (fts + semantic) both find internal/http/client.go but at
	// different line locators WITHIN the same line-bucket (20-wide: 10 and 19
	// are both bucket 0).
	candidates := []FusionCandidate{
		{Path: "internal/http/client.go", Line: 10, Source: "fts"},
		{Path: "internal/http/client.go", Line: 19, Source: "semantic"},
		{Path: "other/file.go", Line: 5, Source: "fts"},
	}
	agreed := FileLevelAgreement(candidates, RRFK)
	// The file found by two retrievers must collapse to ONE candidate with an
	// agreement boost; the other file stays single.
	got := map[string]int{}
	for _, a := range agreed {
		got[a.Path]++
	}
	if got["internal/http/client.go"] != 1 {
		t.Errorf("two retrievers on the same file must yield one candidate, got %d", got["internal/http/client.go"])
	}
	if got["other/file.go"] != 1 {
		t.Errorf("single-retriever file must yield one candidate, got %d", got["other/file.go"])
	}
	// The agreed candidate must carry a higher score than the single one and
	// record the agreement.
	agreedScore, singleScore := 0.0, 0.0
	for _, a := range agreed {
		if a.Path == "internal/http/client.go" {
			agreedScore = a.Score
		}
		if a.Path == "other/file.go" {
			singleScore = a.Score
		}
	}
	if agreedScore <= singleScore {
		t.Errorf("agreement must boost the shared-file candidate: agreed=%v single=%v", agreedScore, singleScore)
	}
}

// TestRerankDifferentBucketsStaySeparate covers the (path, line-bucket) keying:
// two hits on the same file but in DIFFERENT line-buckets stay distinct
// candidates (agreement is per-bucket, not per-file).
func TestRerankDifferentBucketsStaySeparate(t *testing.T) {
	candidates := []FusionCandidate{
		{Path: "a.go", Line: 10, Source: "fts"},   // bucket 0 (10/20)
		{Path: "a.go", Line: 50, Source: "fts"},   // bucket 2 (50/20)
	}
	agreed := FileLevelAgreement(candidates, RRFK)
	if len(agreed) != 2 {
		t.Errorf("two different line-buckets on the same file must stay separate, got %d", len(agreed))
	}
}

func hasFactor(fs []RerankFactor, name string) bool {
	for _, f := range fs {
		if f.Name == name {
			return true
		}
	}
	return false
}

func sumFactorDelta(fs []RerankFactor) float64 {
	var s float64
	for _, f := range fs {
		s += f.Delta
	}
	return s
}
