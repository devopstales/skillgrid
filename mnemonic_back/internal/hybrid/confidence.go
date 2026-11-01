package hybrid

import (
	"fmt"
	"strings"
)

// Categorical confidence levels (01.10). The contract is strictly categorical —
// a search response carries exactly one of these, never a float.
const (
	ConfidenceHigh   = "high"
	ConfidenceMedium = "medium"
	ConfidenceLow    = "low"
)

// Confidence is the categorical retrieval-confidence contract. Level is the
// strict high|medium|low value; the derived fields (Action, Fallbacks) turn it
// into an explicit agent action.
type Confidence struct {
	Level      string   `json:"level"`
	Action     string   `json:"action,omitempty"`
	Suggested  []string `json:"fallbacks,omitempty"`
}

// IsCategorical reports whether Level is one of the three valid values.
func (c Confidence) IsCategorical() bool {
	switch c.Level {
	case ConfidenceHigh, ConfidenceMedium, ConfidenceLow:
		return true
	}
	return false
}

// Fallbacks returns the ready fallback suggestions for a low confidence (01.10):
// a concrete rg pattern, the likely paths the hit pointed at, and a "broaden
// query" hint. High/medium return no fallbacks (they have an action instead).
func (c Confidence) Fallbacks(query string, likelyPaths []string) []string {
	if c.Level != ConfidenceLow {
		return nil
	}
	var out []string
	if query != "" {
		out = append(out, fmt.Sprintf("rg %q (exact symbol/identifier match)", query))
	}
	if len(likelyPaths) > 0 {
		// Dedup + cap so the suggestion stays actionable.
		seen := map[string]bool{}
		for _, p := range likelyPaths {
			if seen[p] {
				continue
			}
			seen[p] = true
			out = append(out, "likely path: "+p)
			if len(out) >= 5 {
				break
			}
		}
	}
	out = append(out, "broaden query: drop qualifiers, search a stem or the file path")
	return out
}

// AgentAction is the explicit action a confidence level maps to (01.10).
type AgentAction struct {
	Level  string `json:"level"`
	Action string `json:"action"`
}

// ActionForConfidence maps a categorical confidence to its explicit agent
// action: high = read the narrowed ranges and answer; medium = read + one
// confirming grep; low = use the fallbacks.
func ActionForConfidence(level string) AgentAction {
	switch level {
	case ConfidenceHigh:
		return AgentAction{Level: level, Action: "read the narrowed line ranges and answer directly"}
	case ConfidenceMedium:
		return AgentAction{Level: level, Action: "read the hit, then run one confirming grep before answering"}
	case ConfidenceLow:
		return AgentAction{Level: level, Action: "use the fallback suggestions (rg pattern / likely paths / broaden query)"}
	default:
		return AgentAction{Level: ConfidenceLow, Action: "confidence invalid — treat as low; use the fallback suggestions"}
	}
}

// WithActionAndFallbacks returns the Confidence with its action + fallbacks
// populated from the level (01.10).
func (c Confidence) WithActionAndFallbacks(query string, likelyPaths []string) Confidence {
	c.Action = ActionForConfidence(c.Level).Action
	c.Suggested = c.Fallbacks(query, likelyPaths)
	return c
}

// mediumScoreThreshold: a raw score at/above this (with no named boost) is a
// cautious medium; below is low.
const mediumScoreThreshold = 0.15

// hasStrongBoost reports whether the fired factors include a strong positive
// signal (exact-symbol or definition-kind at full strength).
func hasStrongBoost(factors []RerankFactor) bool {
	for _, f := range factors {
		if (f.Name == "exact-symbol" || f.Name == "definition-kind") && f.Delta >= 0.15 {
			return true
		}
	}
	return false
}

// hasAnyPenalty reports whether a demoting factor fired (documentation,
// generated/vendor, test-on-non-test) — a penalty caps confidence at medium
// even for a decent score.
func hasAnyPenalty(factors []RerankFactor) bool {
	for _, f := range factors {
		if f.Delta < 0 {
			return true
		}
	}
	return false
}

// ConfidenceFromScore derives the categorical confidence from the fired
// rerank factors + the hit's rerank score (01.10). The factors are the
// primary signal (a raw FTS bm25 score is not comparable across corpora): a
// strong exact-symbol/definition hit with no penalty is high; a hit with any
// boost but weaker, or a penalized hit, is medium; a hit with no signal is low.
func ConfidenceFromScore(baseScore float64, factors []RerankFactor) Confidence {
	switch {
	case hasStrongBoost(factors) && !hasAnyPenalty(factors):
		return Confidence{Level: ConfidenceHigh}
	case hasAnyBoost(factors):
		return Confidence{Level: ConfidenceMedium}
	case baseScore >= mediumScoreThreshold:
		// No named boost fired but the raw score is still above the weak
		// floor — a cautious medium.
		return Confidence{Level: ConfidenceMedium}
	default:
		return Confidence{Level: ConfidenceLow}
	}
}

// hasAnyBoost reports whether any positive (boosting) factor fired.
func hasAnyBoost(factors []RerankFactor) bool {
	for _, f := range factors {
		if f.Delta > 0 {
			return true
		}
	}
	return false
}

// likelyPaths extracts the distinct file paths from a set of hits (for the
// low-confidence fallback suggestions).
func likelyPaths(hits []Hit) []string {
	seen := map[string]bool{}
	var out []string
	for _, h := range hits {
		if !seen[h.Path] {
			seen[h.Path] = true
			out = append(out, h.Path)
		}
	}
	return out
}

// RerankDelta is the signed total of a hit's fired rerank factors (the
// explainable delta on top of the RRF score).
func RerankDelta(factors []RerankFactor) float64 {
	var s float64
	for _, f := range factors {
		s += f.Delta
	}
	return s
}

// RerankReasons renders the fired factors as a human-readable explainable
// string ("name: rationale (±delta)"), one per line, in deterministic order.
func RerankReasons(factors []RerankFactor) string {
	if len(factors) == 0 {
		return ""
	}
	var b strings.Builder
	for i, f := range factors {
		if i > 0 {
			b.WriteString("; ")
		}
		sign := "-"
		if f.Delta >= 0 {
			sign = "+"
		}
		fmt.Fprintf(&b, "%s %s%.3f (%s)", f.Name, sign, f.Delta, f.Rationale)
	}
	return b.String()
}
