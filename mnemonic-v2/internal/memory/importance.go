package memory

import "time"

// importance.go — the on-the-fly importance score (spec 002-schema §2.2.3,
// v1 dream.go:382-400). The formula is the source of truth; the stored
// observations.importance column is a cache, computed at pipeline step 6d
// from the row's own state.

// Importance scores an observation from its named sources:
//
//	Importance(retrievalUsage, ageDays, typ)
//	    = clamp(0.6*min(usage,50)/50 + 0.37*recency + 0.2*typeDurability, 0, 1)
//
// recency is a GATED age penalty (not a raw freshness boost): age ≤ 30 days
// keeps full recency 1.0; past the grace window it decays to 0 at 60 days.
// typeDurability = 0.9 for {decision, architecture, convention, config},
// else 0.4 (v1 dream.go:62-64). retrieval_usage is the named source of the
// usage term — the observations.retrieval_usage column, incremented by
// Store.BumpRetrievalUsage in the lookup path (Task 11) and read at
// pipeline step 6 — never a free parameter.
func Importance(retrievalUsage, ageDays int, typ string) float64 {
	usageNorm := 0.0
	if retrievalUsage > 0 {
		usageNorm = float64(retrievalUsage) / 50.0
		if usageNorm > 1.0 {
			usageNorm = 1.0
		}
	}
	recency := 1.0
	if ageDays > 30 {
		r := 1.0 - float64(ageDays)/30.0
		if r < 0 {
			r = 0
		}
		recency = r
	}
	typeDurability := typeDurabilityDefault
	if durableTypes[typ] {
		typeDurability = typeDurabilityDurable
	}
	s := 0.6*usageNorm + 0.37*recency + 0.2*typeDurability
	if s < 0 {
		s = 0
	}
	if s > 1 {
		s = 1
	}
	return s
}

const (
	typeDurabilityDurable = 0.9
	typeDurabilityDefault = 0.4
)

// durableTypes is the spec 002-schema §2.2.3 durable vocabulary.
var durableTypes = map[string]bool{
	"decision":     true,
	"architecture": true,
	"convention":   true,
	"config":       true,
}

// ageDays is the whole-day age of a store timestamp ("2006-01-02 15:04:05")
// from now, floored at 0. A zero/unparseable stamp is age 0 (a fresh row).
// The recompute path (lookup.go) derives the row's age from the rank-age
// anchor through this helper, so the importance formula's recency term uses
// the same anchor the search boost does (last_seen_at || updated_at ||
// created_at, spec 005 §3).
func ageDays(stamp string) int {
	t, err := time.Parse("2006-01-02 15:04:05", stamp)
	if err != nil || t.IsZero() {
		return 0
	}
	days := int(time.Since(t).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
