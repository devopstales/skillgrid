package memory

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

// parseObsTimestamp parses an observation created_at/updated_at value, which
// the codebase persists as RFC3339 (UTC, "Z"). Falls back to the SQLite
// "YYYY-MM-DD HH:MM:SS" shape some legacy rows use (treated as UTC).
func parseObsTimestamp(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty timestamp")
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02 15:04:05", s)
}

// improveDefaultCooldown is the default minimum interval between two
// re-rankings of the same search result set (014 step 08). A zero or
// negative configured cooldown falls back to this, so consecutive searches
// cannot churn the ranking more often than once per minute.
const improveDefaultCooldown = time.Minute

// ImproveConfig tunes the self-improvement feedback loop (014 step 08). It is
// OPT-IN: Enabled defaults to false so the default search behavior is
// byte-identical to the pre-improve SQL ordering. The rates are configurable
// via the mnemonic.improve config section (mnemonic/config).
//
// Formulas (applied in-memory after the SQL has ranked the results):
//
//	boost = min(retrieval_usage, MaxUsage) * BoostRate   (when retrieval_usage > Threshold)
//	decay = max(0, age - TTL) / TTL * DecayRate          (when retrieval_usage == 0)
//	score = baseRankScore + boost - decay
//
// baseRankScore is the position-based rank of the SQL result set (0 for the
// top hit, +1 per position). The re-ranked order is deterministic and is the
// order SearchOwnerScoped returns when improve is enabled.
type ImproveConfig struct {
	// Enabled gates the whole loop (config mnemonic.improve.enabled,
	// default false → improve() is a no-op and search output is unchanged).
	Enabled bool
	// Threshold is the minimum retrieval_usage that earns a boost (default 5).
	Threshold int
	// MaxUsage caps the boost input so one hot observation cannot outrank
	// everything unbounded (default 50).
	MaxUsage int
	// BoostRate multiplies the capped usage into the boost score (default 0.1).
	BoostRate float64
	// DecayRate multiplies the TTL-excess fraction into the decay score
	// (default 0.25).
	DecayRate float64
	// Cooldown is the minimum interval between re-rankings on the same
	// service (default improveDefaultCooldown when zero). A sub-millisecond
	// value (e.g. time.Nanosecond) effectively disables the gating for a
	// process — used by tests that issue back-to-back searches.
	Cooldown time.Duration
	// Now is the clock seam for tests (default time.Now). The age of an
	// observation is now - created_at.
	Now func() time.Time
}

// defaultImproveConfig returns the production defaults: disabled (opt-in),
// with the documented rates so an operator enabling the loop gets sane values
// out of the box.
func defaultImproveConfig() ImproveConfig {
	return ImproveConfig{
		Threshold:   5,
		MaxUsage:    50,
		BoostRate:   0.1,
		DecayRate:   0.25,
		Cooldown:    improveDefaultCooldown,
		Now:         time.Now,
	}
}

// SetImprove configures the opt-in feedback loop (014 step 08). The service
// layer calls it from the mnemonic.improve config key; a zero-value call
// reverts every field to the production defaults (still disabled, since the
// config section is absent by default).
func (s *Service) SetImprove(cfg ImproveConfig) {
	if s == nil {
		return
	}
	def := defaultImproveConfig()
	if cfg.Threshold <= 0 {
		cfg.Threshold = def.Threshold
	}
	if cfg.MaxUsage <= 0 {
		cfg.MaxUsage = def.MaxUsage
	}
	if cfg.BoostRate <= 0 {
		cfg.BoostRate = def.BoostRate
	}
	if cfg.DecayRate <= 0 {
		cfg.DecayRate = def.DecayRate
	}
	if cfg.Cooldown <= 0 {
		cfg.Cooldown = def.Cooldown
	}
	if cfg.Now == nil {
		cfg.Now = def.Now
	}
	s.improveCfg = cfg
	s.improveLast = time.Time{}
}

// ImproveConfig returns the active feedback-loop configuration (the zero
// value is the disabled default). Tests use it to snapshot and restore the
// config around raw-order captures.
func (s *Service) ImproveConfig() ImproveConfig {
	if s == nil {
		return ImproveConfig{}
	}
	return s.improveCfg
}

// improve re-ranks an already-SQL-ranked search result set by retrieval
// usage (014 step 08, self-improvement feedback loop). It is the
// in-memory re-rank the brief requires: the SQL ORDER BY (pinned, bm25) is
// untouched, and when improve is disabled (the default) it returns the input
// unchanged, so default search is byte-identical to pre-improve.
//
// A nil/empty input is returned as-is. The cooldown (per service) prevents
// excessive re-weighting on consecutive searches: within the cooldown window
// the input is returned unchanged.
func (s *Service) improve(_ context.Context, results []Observation) []Observation {
	if s == nil || len(results) == 0 || !s.improveCfg.Enabled {
		return results
	}
	now := s.improveCfg.Now()
	if !s.improveLast.IsZero() && now.Sub(s.improveLast) < s.improveCfg.Cooldown {
		return results
	}
	s.improveLast = now
	ttl := s.effectiveTTL()
	scores := make([]float64, len(results))
	for i, o := range results {
		scores[i] = float64(i)
		scores[i] += s.improveBoost(o.RetrievalUsage)
		scores[i] -= s.improveDecay(o.RetrievalUsage, o.CreatedAt, now, ttl)
	}
	// Stable partition by score descending; equal scores keep the SQL rank.
	idx := make([]int, len(results))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return scores[idx[a]] > scores[idx[b]] })
	out := make([]Observation, len(results))
	for i, j := range idx {
		out[i] = results[j]
	}
	return out
}

// improveBoost is the retrieval-usage boost: capped linear in usage, only
// past the threshold. boost = min(retrieval_usage, MaxUsage) * BoostRate.
func (s *Service) improveBoost(usage int) float64 {
	if s == nil || !s.improveCfg.Enabled || usage <= s.improveCfg.Threshold {
		return 0
	}
	c := usage
	if c > s.improveCfg.MaxUsage {
		c = s.improveCfg.MaxUsage
	}
	return float64(c) * s.improveCfg.BoostRate
}

// improveDecay is the never-accessed decay: proportional to how far the
// observation's age exceeds the TTL.
// decay = max(0, age - TTL) / TTL * DecayRate, applied only when
// retrieval_usage == 0. An observation created within the TTL (or ever
// retrieved) never decays.
func (s *Service) improveDecay(usage int, createdAt string, now time.Time, ttl time.Duration) float64 {
	if s == nil || !s.improveCfg.Enabled || usage != 0 || ttl <= 0 {
		return 0
	}
	created, err := parseObsTimestamp(createdAt)
	if err != nil || created.IsZero() {
		return 0
	}
	age := now.Sub(created)
	excess := age - ttl
	if excess <= 0 {
		return 0
	}
	return float64(excess) / float64(ttl) * s.improveCfg.DecayRate
}
