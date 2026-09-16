package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// AKL importance scoring + recency decay (014, step 13).
//
// The score is a continuous, retrieval-weighted value that decays
// exponentially with the observation's age:
//
//	importance_score = retrieval_count * exp(-decay_rate * age_days)
//	recency_decay    = exp(-decay_rate * age_days)
//
// The decay rate (per day) is configurable via the mnemonic.importance.decay
// config key; the default is 0.05/day, so a 30-day-old observation with 100
// retrievals scores 100 * exp(-1.5) ≈ 22.3. A higher decay rate erodes old
// observations faster; a lower one retains importance longer.
//
// The score and its tier are stored as columns (migration 025) so they can
// be indexed/queried. They are populated on save and on an explicit
// RecomputeImportance pass; the query-time boost (step 13.3) reads the stored
// values at search time, where the age is bounded by the search result set.

// Maturity tiers, in ascending rank. Transitions are MONOTONIC: an
// observation never drops from a higher tier to a lower one.
const (
	TierFresh    = "fresh"
	TierMature   = "mature"
	TierArchival = "archival"
)

// maturityRank orders the tiers for the monotonic max.
var maturityRank = map[string]int{
	TierFresh:    0,
	TierMature:   1,
	TierArchival: 2,
}

// TierThresholds are the maturity-tier age thresholds in days (13.4). They
// are configurable via the mnemonic.importance.tier_thresholds config key.
type TierThresholds struct {
	// MatureAgeDays: age >= this AND retrievalCount > 0 → mature.
	MatureAgeDays int
	// ArchivalAgeDays: age >= this → archival (regardless of usage).
	ArchivalAgeDays int
	// UnusedArchivalDays: age > this AND retrievalCount == 0 → archival.
	UnusedArchivalDays int
}

// ImportanceConfig tunes AKL importance scoring (014, step 13). The service
// layer calls SetImportance from the mnemonic.importance config section; zero
// fields fall back to the defaults below, so a config without the section
// (or a malformed value in it) keeps the production behavior.
type ImportanceConfig struct {
	// DecayRate is the per-day exponential decay (default defaultImportanceDecay).
	DecayRate float64
	// Thresholds are the maturity-tier age cutoffs (default defaultTierThresholds).
	Thresholds TierThresholds
	// Now is the clock seam for tests (default time.Now).
	Now func() time.Time
}

// defaultImportanceDecay is the per-day decay rate (mnemonic.importance.decay).
const defaultImportanceDecay = 0.05

// defaultTierThresholds are the brief's tier cutoffs, in days.
var defaultTierThresholds = TierThresholds{
	MatureAgeDays:    7,
	ArchivalAgeDays:  30,
	UnusedArchivalDays: 14,
}

// defaultImportanceConfig returns the production defaults: the 0.05/day decay
// and the 7/30/14-day tier thresholds.
func defaultImportanceConfig() ImportanceConfig {
	return ImportanceConfig{
		DecayRate:  defaultImportanceDecay,
		Thresholds: defaultTierThresholds,
		Now:        time.Now,
	}
}

// ageDays returns the whole-day component of age (negative ages clamp to 0).
func ageDays(age time.Duration) float64 {
	d := age.Hours() / 24
	if d < 0 {
		return 0
	}
	return d
}

// ComputeImportanceScore is the AKL importance score (014 step 13.1):
//
//	retrieval_count * exp(-decay_rate * age_days)
//
// It increases with retrieval_count and decays exponentially with age. A zero
// or negative age is undiminished (age_days = 0 → exp(0) = 1); a negative
// decayRate (malformed config) is treated as 0.
func ComputeImportanceScore(retrievalCount int, age time.Duration, decayRate float64) float64 {
	if retrievalCount <= 0 {
		return 0
	}
	if decayRate < 0 {
		decayRate = 0
	}
	return float64(retrievalCount) * recencyFactor(age, decayRate)
}

// recencyFactor is the exponential recency decay: exp(-decay_rate * age_days).
// It is the factor multiplying retrieval_count in ComputeImportanceScore and
// is also stored as the recency_decay column for provenance.
func recencyFactor(age time.Duration, decayRate float64) float64 {
	if decayRate < 0 {
		decayRate = 0
	}
	return math.Exp(-decayRate * ageDays(age))
}

// ComputeMaturityTier classifies an observation into a maturity tier using
// the default thresholds (014 step 13.2):
//
//	archival: age >= ArchivalAgeDays OR (retrievalCount == 0 AND age > UnusedArchivalDays)
//	mature:   age >= MatureAgeDays AND retrievalCount > 0
//	fresh:    everything else
func ComputeMaturityTier(age time.Duration, retrievalCount int) string {
	return ComputeMaturityTierWithThresholds(age, retrievalCount, defaultTierThresholds)
}

// ComputeMaturityTierWithThresholds is ComputeMaturityTier with explicit,
// configurable thresholds (014 step 13.4).
func ComputeMaturityTierWithThresholds(age time.Duration, retrievalCount int, th TierThresholds) string {
	days := ageDays(age)
	if days >= float64(th.ArchivalAgeDays) {
		return TierArchival
	}
	if retrievalCount == 0 && days > float64(th.UnusedArchivalDays) {
		return TierArchival
	}
	if days >= float64(th.MatureAgeDays) && retrievalCount > 0 {
		return TierMature
	}
	return TierFresh
}

// NextMaturityTier applies the monotonic rule (014 step 13.2): the effective
// tier is max(previousTier, computedTier) in the fresh < mature < archival
// ordering. Once an observation is mature it never goes back to fresh; once
// archival it never goes back to mature. An empty/unknown previous tier
// behaves like fresh (the lowest rank).
func NextMaturityTier(previous, computed string) string {
	prev := maturityRank[previous]
	comp := maturityRank[computed]
	if prev <= 0 {
		prev = maturityRank[TierFresh]
	}
	if comp <= 0 {
		comp = maturityRank[TierFresh]
	}
	switch {
	case comp >= prev:
		return computed
	default:
		return previous
	}
}

// SetImportance configures AKL importance scoring (014, step 13.4). The
// service layer calls it from the mnemonic.importance config section; zero
// fields fall back to the production defaults, so a zero-value call keeps
// exactly the default scoring.
func (s *Service) SetImportance(cfg ImportanceConfig) {
	if s == nil {
		return
	}
	def := defaultImportanceConfig()
	if cfg.DecayRate <= 0 {
		cfg.DecayRate = def.DecayRate
	}
	if cfg.Thresholds.MatureAgeDays <= 0 {
		cfg.Thresholds.MatureAgeDays = def.Thresholds.MatureAgeDays
	}
	if cfg.Thresholds.ArchivalAgeDays <= 0 {
		cfg.Thresholds.ArchivalAgeDays = def.Thresholds.ArchivalAgeDays
	}
	if cfg.Thresholds.UnusedArchivalDays <= 0 {
		cfg.Thresholds.UnusedArchivalDays = def.Thresholds.UnusedArchivalDays
	}
	if cfg.Now == nil {
		cfg.Now = def.Now
	}
	s.importanceCfg = cfg
}

// ImportanceConfig returns the active importance configuration (defaults when
// SetImportance was never called — a Service built directly in unit tests).
func (s *Service) ImportanceConfig() ImportanceConfig {
	if s == nil {
		return ImportanceConfig{}
	}
	if s.importanceCfg.DecayRate <= 0 {
		return defaultImportanceConfig()
	}
	return s.importanceCfg
}

// effectiveImportance returns the active importance configuration, filling in
// any unset field with its default.
func (s *Service) effectiveImportance() ImportanceConfig {
	if s == nil {
		return defaultImportanceConfig()
	}
	cfg := s.importanceCfg
	def := defaultImportanceConfig()
	if cfg.DecayRate <= 0 {
		cfg.DecayRate = def.DecayRate
	}
	if cfg.Thresholds.MatureAgeDays <= 0 {
		cfg.Thresholds.MatureAgeDays = def.Thresholds.MatureAgeDays
	}
	if cfg.Thresholds.ArchivalAgeDays <= 0 {
		cfg.Thresholds.ArchivalAgeDays = def.Thresholds.ArchivalAgeDays
	}
	if cfg.Thresholds.UnusedArchivalDays <= 0 {
		cfg.Thresholds.UnusedArchivalDays = def.Thresholds.UnusedArchivalDays
	}
	if cfg.Now == nil {
		cfg.Now = def.Now
	}
	return cfg
}

// testRawImportanceMu guards TestRawImportance (written by tests before a
// save, read by stampImportance during that same save).
var testRawImportanceMu sync.Mutex

// stampImportance computes and persists the importance columns for one
// observation (014 step 13.1). It is called from Save() and from
// RecomputeImportance(). The maturity tier is MONOTONIC: the stored tier is
// max(previous, computed), so a re-evaluation never downgrades.
func (s *Service) stampImportance(ctx context.Context, id int64, retrievalUsage int, createdAt string) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return
	}
	cfg := s.effectiveImportance()
	created, err := parseObsTimestamp(createdAt)
	if err != nil || created.IsZero() {
		created = cfg.Now()
	}
	age := cfg.Now().Sub(created)
	score := ComputeImportanceScore(retrievalUsage, age, cfg.DecayRate)
	// Test-only seam (014 step 16): a non-negative TestRawImportance overrides
	// the computed score so federated-query tests can seed arbitrary per-
	// observation importance scores. Production leaves it at -1 (unset).
	testRawImportanceMu.Lock()
	raw := s.TestRawImportance
	testRawImportanceMu.Unlock()
	if raw >= 0 {
		score = raw
	}
	decay := recencyFactor(age, cfg.DecayRate)
	tier := ComputeMaturityTierWithThresholds(age, retrievalUsage, cfg.Thresholds)

	var prevTier sql.NullString
	if err := s.store.DB.QueryRowContext(ctx,
		`SELECT maturity_tier FROM observations WHERE id = ?`, id,
	).Scan(&prevTier); err != nil && !errors_IsNoRows(err) {
		return // best-effort: a tier lookup failure never blocks the save
	}
	stored := TierFresh
	if prevTier.Valid && prevTier.String != "" {
		stored = prevTier.String
	}
	stored = NextMaturityTier(stored, tier)

	if _, err := s.store.DB.ExecContext(ctx, `
		UPDATE observations SET importance_score = ?, recency_decay = ?, maturity_tier = ?
		WHERE id = ?`,
		score, decay, stored, id,
	); err != nil {
		// Best-effort: the importance columns are advisory (ranking +
		// prune support) and a write failure must never fail the save.
	}
}

// RecomputeImportance re-evaluates the importance columns for every live
// observation in this project (014 step 13.1, the periodic-recompute path).
// The per-save stamping keeps fresh rows current; this pass catches rows
// whose age crossed a tier threshold without a save. It is idempotent and
// monotonic (tiers only move up).
func (s *Service) RecomputeImportance(ctx context.Context) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return fmt.Errorf("memory service not initialized")
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT id, created_at, COALESCE(retrieval_usage, 0)
		FROM observations
		WHERE project = ? AND deleted_at IS NULL`,
		s.projectID,
	)
	if err != nil {
		return fmt.Errorf("recompute importance: %w", err)
	}
	defer rows.Close()
	type row struct {
		id    int64
		at    string
		usage int
	}
	var pending []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.at, &r.usage); err != nil {
			return fmt.Errorf("recompute importance scan: %w", err)
		}
		pending = append(pending, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("recompute importance iterate: %w", err)
	}
	for _, r := range pending {
		s.stampImportance(ctx, r.id, r.usage, r.at)
	}
	return nil
}

// observedImportance returns the importance score used by the query-time
// re-rank (014 step 13.3): the stored importance_score when it is populated,
// otherwise a compute-on-the-fly fallback from retrieval_usage + created_at
// (pre-025 rows or a stamp failure).
func (s *Service) observedImportance(o Observation, now time.Time) float64 {
	if o.ImportanceScore.Valid && o.ImportanceScore.Float64 > 0 {
		return o.ImportanceScore.Float64
	}
	age := time.Duration(0)
	if created, err := parseObsTimestamp(o.CreatedAt); err == nil && !created.IsZero() {
		age = now.Sub(created)
	}
	return ComputeImportanceScore(o.RetrievalUsage, age, s.effectiveImportance().DecayRate)
}

// improve re-ranks an already-SQL-ranked search result set by retrieval
// usage (014 step 08, self-improvement feedback loop). Step 13.3 replaced it
// in SearchOwnerScoped with importanceRerank (importance_score instead of
// binary retrieval_usage); it is kept for direct callers and the step-08
// tests. It is the in-memory re-rank the brief requires: the SQL ORDER BY
// (pinned, bm25) is untouched, and when improve is disabled (the default) it
// returns the input unchanged, so default search is byte-identical to
// pre-improve.
//
// A nil/empty input is returned as-is. The cooldown (per service) prevents
// excessive re-weighting on consecutive searches: within the cooldown window
// the input is returned unchanged.

// importanceRerank applies the AKL importance boost to an already-SQL-ranked
// search result set (014 step 13.3). It is the in-memory re-rank the brief
// requires: the SQL ORDER BY (pinned, bm25) is untouched, and when the
// improve() opt-in is disabled it returns the input unchanged, so default
// search is byte-identical to pre-step-13.
//
// The boost is multiplicative on the base relevance score. Base relevance is
// the position-based rank of the SQL result set (0 for the top hit, +1 per
// position) plus 1, so every document has a positive base:
//
//	base     = 1 + position
//	factor   = importance_score / max_importance   (normalized to (0, 1])
//	score    = base * factor
//
// max_importance is the maximum score in the result set (when no score is
// positive the boost is a no-op — nothing to normalize). A stable partition
// by score descending keeps the SQL rank as the tie-breaker, so equal-importance
// documents preserve their SQL order. The re-rank reuses the improve()
// cooldown anchor (same opt-in surface: mnemonic.improve.enabled).
func (s *Service) importanceRerank(_ context.Context, results []Observation) []Observation {
	if s == nil || len(results) == 0 || !s.improveCfg.Enabled {
		return results
	}
	now := s.improveCfg.Now()
	if !s.improveLast.IsZero() && now.Sub(s.improveLast) < s.improveCfg.Cooldown {
		return results
	}
	s.improveLast = now
	scores := make([]float64, len(results))
	maxScore := 0.0
	for i, o := range results {
		sc := s.observedImportance(o, now)
		if sc > maxScore {
			maxScore = sc
		}
		scores[i] = sc
	}
	if maxScore <= 0 {
		return results
	}
	weighted := make([]float64, len(results))
	for i, sc := range scores {
		weighted[i] = (1 + float64(i)) * (sc / maxScore)
	}
	idx := make([]int, len(weighted))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return weighted[idx[a]] > weighted[idx[b]] })
	out := make([]Observation, len(weighted))
	for i, j := range idx {
		out[i] = results[j]
	}
	return out
}

// errors_IsNoRows wraps errors.Is for the single stampImportance call site.
func errors_IsNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// setImportanceColumns assigns the importance columns of an Observation from
// the three trailing values returned by a 26-column observation SELECT.
func (o *Observation) setImportanceColumns(score, decay sql.NullFloat64, tier sql.NullString) {
	o.ImportanceScore = score
	o.RecencyDecay = decay
	if tier.Valid {
		o.MaturityTier = tier.String
	}
}
