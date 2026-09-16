package hybrid

import (
	"strings"
	"testing"
)

// TestConfidenceCategorical covers @step-01 (Scenario: Search response carries
// confidence action rerank reasons and fallbacks): the confidence is a strict
// categorical high|medium|low (never a float, never a fourth value).
func TestConfidenceCategorical(t *testing.T) {
	for _, level := range []string{
		ConfidenceHigh, ConfidenceMedium, ConfidenceLow,
	} {
		c := Confidence{Level: level}
		if !c.IsCategorical() {
			t.Errorf("level %q must be a valid categorical confidence", level)
		}
	}
	if (Confidence{Level: "bogus"}).IsCategorical() {
		t.Errorf("'bogus' is not a valid categorical confidence")
	}
}

// TestConfidenceActionMapping covers @step-01: each categorical confidence maps
// to an explicit agent action — high = read ranges + answer; medium = read +
// one confirming grep; low = use fallbacks.
func TestConfidenceActionMapping(t *testing.T) {
	high := ActionForConfidence(ConfidenceHigh)
	if !strings.Contains(strings.ToLower(high.Action), "answer") {
		t.Errorf("high action must instruct to answer, got %q", high.Action)
	}

	medium := ActionForConfidence(ConfidenceMedium)
	if !strings.Contains(strings.ToLower(medium.Action), "confirm") || !strings.Contains(strings.ToLower(medium.Action), "grep") {
		t.Errorf("medium action must instruct a confirming grep, got %q", medium.Action)
	}

	low := ActionForConfidence(ConfidenceLow)
	if !strings.Contains(strings.ToLower(low.Action), "fallback") {
		t.Errorf("low action must instruct to use fallbacks, got %q", low.Action)
	}
}

// TestLowConfidenceFallbacks covers @step-01: a low confidence attaches ready
// fallback suggestions — a concrete rg pattern, likely paths, and a "broaden
// query" hint.
func TestLowConfidenceFallbacks(t *testing.T) {
	c := Confidence{Level: ConfidenceLow}
	fb := c.Fallbacks("NewClient", []string{"internal/http/client.go", "internal/http/handler.go"})
	if len(fb) == 0 {
		t.Fatal("low confidence must attach fallback suggestions")
	}
	foundRG := false
	foundPaths := false
	foundBroaden := false
	for _, f := range fb {
		if strings.Contains(f, "rg ") || strings.Contains(f, "grep ") {
			foundRG = true
		}
		if strings.Contains(f, "internal/http/") {
			foundPaths = true
		}
		if strings.Contains(strings.ToLower(f), "broaden") {
			foundBroaden = true
		}
	}
	if !foundRG {
		t.Errorf("low fallbacks must include a ready rg/grep pattern, got %v", fb)
	}
	if !foundPaths {
		t.Errorf("low fallbacks must include likely paths, got %v", fb)
	}
	if !foundBroaden {
		t.Errorf("low fallbacks must suggest broadening the query, got %v", fb)
	}
}

// TestHighConfidenceNoFallbacks covers the contract: high/medium confidence do
// NOT attach fallback suggestions (they have an action instead).
func TestHighConfidenceNoFallbacks(t *testing.T) {
	if fb := (Confidence{Level: ConfidenceHigh}).Fallbacks("x", nil); len(fb) != 0 {
		t.Errorf("high confidence should carry no fallbacks, got %v", fb)
	}
	if fb := (Confidence{Level: ConfidenceMedium}).Fallbacks("x", nil); len(fb) != 0 {
		t.Errorf("medium confidence should carry no fallbacks, got %v", fb)
	}
}

// TestConfidenceFromScore covers @step-01: confidence is derived from the
// rerank signal — a strong exact-symbol hit is high, a weak/no-signal hit is
// low, and an in-between is medium.
func TestConfidenceFromScore(t *testing.T) {
	// A hit with the exact-symbol + definition boost is high.
	highFactors := ApplyRerankTable(RankHit{Path: "a.go", Symbol: "X", Kind: "function"}, RerankContext{Query: "X", Degree: 2})
	if conf := ConfidenceFromScore(0.9, highFactors); conf.Level != ConfidenceHigh {
		t.Errorf("strong exact-symbol hit should be high confidence, got %v", conf.Level)
	}

	// A hit with no factors at all (a weak match) is low.
	weakFactors := ApplyRerankTable(RankHit{Path: "vendor/lib/x.go"}, RerankContext{Query: "zzz"})
	if conf := ConfidenceFromScore(0.1, weakFactors); conf.Level != ConfidenceLow {
		t.Errorf("weak no-signal hit should be low confidence, got %v", conf.Level)
	}
}
