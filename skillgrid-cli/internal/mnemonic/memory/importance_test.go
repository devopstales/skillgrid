package memory

import (
	"context"
	"math"
	"slices"
	"testing"
	"time"
)

// TestImportanceScoreComputation covers 13.1: the AKL importance score is
// retrieval_count * exp(-decay_rate * age_days) — it increases with
// retrieval_count, decays exponentially with age, and honors the formula
// exactly (the 30-day / 100-retrieval / 0.05-decay case is the documented
// 22.3 anchor).
func TestImportanceScoreComputation(t *testing.T) {
	const decay = 0.05

	// Formula anchor: 100 retrievals, 30 days old, decay 0.05 → ≈22.3.
	got := ComputeImportanceScore(100, 30*24*time.Hour, decay)
	want := 100 * math.Exp(-decay*30)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("formula: 100 retrievals @ 30d with decay 0.05 = %v, want %v", got, want)
	}
	if got < 22 || got > 23 {
		t.Fatalf("sanity: 100 retrievals @ 30d should be ≈22.3, got %v", got)
	}

	// Increases with retrieval_count at fixed age (0/5/50/100).
	counts := []int{0, 5, 50, 100}
	age := 7 * 24 * time.Hour
	scores := make([]float64, len(counts))
	for i, c := range counts {
		scores[i] = ComputeImportanceScore(c, age, decay)
	}
	if scores[0] != 0 {
		t.Fatalf("0 retrievals must score exactly 0, got %v", scores[0])
	}
	for i := 1; i < len(counts); i++ {
		if !(scores[i] > scores[i-1]) {
			t.Fatalf("importance must increase with retrieval_count: %v", scores)
		}
	}

	// Exponential decay with age at fixed retrieval_count (1d/7d/30d).
	ages := []time.Duration{24 * time.Hour, 7 * 24 * time.Hour, 30 * 24 * time.Hour}
	aged := make([]float64, len(ages))
	for i, a := range ages {
		aged[i] = ComputeImportanceScore(50, a, decay)
	}
	for i := 1; i < len(aged); i++ {
		if !(aged[i] < aged[i-1]) {
			t.Fatalf("older observations must score lower: %v", aged)
		}
	}
	// The decay is exponential: the 1d→7d and 7d→30d ratios differ
	// (a linear decay would make both ratios equal).
	if math.Abs(aged[0]/aged[1]-aged[1]/aged[2]) < 1e-6 {
		t.Fatalf("decay must be exponential, not linear: %v", aged)
	}

	// A zero age is undiminished (age_days = 0 → exp(0) = 1).
	if got := ComputeImportanceScore(42, 0, decay); math.Abs(got-42) > 1e-9 {
		t.Fatalf("zero age must be undiminished: got %v, want 42", got)
	}
	// A negative age (clock skew) must not amplify the score past the
	// undiminished value.
	if got := ComputeImportanceScore(42, -time.Hour, decay); got > 42 {
		t.Fatalf("negative age must not amplify: got %v", got)
	}
}

// TestMaturityTierTransitions covers 13.2: the maturity tier moves
// fresh → mature → archival as age + retrieval accumulate, and transitions
// are MONOTONIC — an observation can never be downgraded from a higher tier.
func TestMaturityTierTransitions(t *testing.T) {
	// Pure function transitions (thresholds per the brief):
	//   fresh:    age < 7d
	//   mature:   age >= 7d AND retrievalCount > 0
	//   archival: age >= 30d OR (retrievalCount == 0 AND age > 14d)
	cases := []struct {
		age    time.Duration
		usage  int
		want   string
	}{
		{0, 0, "fresh"},
		{0, 10, "fresh"}, // age < 7d → fresh regardless of usage
		{6 * 24 * time.Hour, 10, "fresh"},
		{7 * 24 * time.Hour, 10, "mature"},
		{7 * 24 * time.Hour, 1, "mature"},
		{20 * 24 * time.Hour, 3, "mature"},
		{15 * 24 * time.Hour, 0, "archival"}, // unused > 14d → archival
		{14 * 24 * time.Hour, 0, "fresh"},    // 14d unused is NOT yet archival
		{30 * 24 * time.Hour, 0, "archival"},
		{30 * 24 * time.Hour, 100, "archival"}, // age >= 30d dominates
		{45 * 24 * time.Hour, 5, "archival"},
	}
	for _, c := range cases {
		if got := ComputeMaturityTier(c.age, c.usage); got != c.want {
			t.Errorf("ComputeMaturityTier(age=%v, usage=%d) = %q, want %q", c.age, c.usage, got, c.want)
		}
	}

	// Monotonic: once higher, never lower (stored = max(previous, computed)).
	if got := ComputeMaturityTier(20*24*time.Hour, 1); got != "mature" {
		t.Fatalf("setup: 20d + 1 usage must be mature, got %q", got)
	}
	if got := NextMaturityTier("mature", ComputeMaturityTier(2*24*time.Hour, 100)); got != "mature" {
		t.Fatalf("monotonic: a mature observation re-evaluated at 2d/100 must stay mature, got %q", got)
	}
	if got := NextMaturityTier("archival", ComputeMaturityTier(1*24*time.Hour, 10)); got != "archival" {
		t.Fatalf("monotonic: an archival observation must never downgrade, got %q", got)
	}
	if got := NextMaturityTier("fresh", ComputeMaturityTier(8*24*time.Hour, 3)); got != "mature" {
		t.Fatalf("monotonic: fresh can upgrade, got %q", got)
	}

	// Persistence: Save() stamps the tier, and a later re-evaluation only
	// moves it UP.
	fx := newOwnerFixture(t, "tier-obs")
	ctx := context.Background()
	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID,
		Type:      "decision",
		Title:     "tier transition asset",
		Content:   "body for the tier test",
		Owner:     fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	obs, err := fx.svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if obs.MaturityTier != "fresh" {
		t.Fatalf("a freshly saved observation must be tier fresh, got %q", obs.MaturityTier)
	}
	if !obs.ImportanceScore.Valid {
		t.Fatalf("importance_score must be populated on save")
	}
	if obs.ImportanceScore.Float64 != 0 {
		t.Fatalf("a 0-usage observation must score 0, got %v", obs.ImportanceScore.Float64)
	}
	if obs.RecencyDecay.Float64 == 0 {
		t.Fatalf("recency_decay must be populated on save (exp(0)=1), got NULL")
	}

	// Age it to 7d with 10 retrievals → recompute moves it to mature.
	now := time.Now().UTC()
	mustExec(t, fx, `UPDATE observations SET retrieval_usage = 10,
		created_at = ? WHERE id = ?`,
		now.Add(-7*24*time.Hour).Format(time.RFC3339), id)
	fx.svc.RecomputeImportance(ctx)
	obs, err = fx.svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get after aging: %v", err)
	}
	if obs.MaturityTier != "mature" {
		t.Fatalf("7d + 10 retrievals must be mature, got %q", obs.MaturityTier)
	}

	// Age it to 30d → archival (and usage dropped to 0 — a recompute must
	// still never downgrade, it can only upgrade).
	mustExec(t, fx, `UPDATE observations SET retrieval_usage = 0,
		created_at = ? WHERE id = ?`,
		now.Add(-30*24*time.Hour).Format(time.RFC3339), id)
	fx.svc.RecomputeImportance(ctx)
	obs, err = fx.svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get after 30d: %v", err)
	}
	if obs.MaturityTier != "archival" {
		t.Fatalf("30d must be archival, got %q", obs.MaturityTier)
	}

	// 30d + 0 usage with no stored tier (NULL) → a recompute alone takes it
	// straight to archival.
	id2, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID,
		Type:      "decision",
		Title:     "tier archival direct",
		Content:   "body for the archival-direct test",
		Owner:     fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save2: %v", err)
	}
	mustExec(t, fx, `UPDATE observations SET maturity_tier = NULL,
		created_at = ? WHERE id = ?`,
		now.Add(-30*24*time.Hour).Format(time.RFC3339), id2)
	fx.svc.RecomputeImportance(ctx)
	obs2, err := fx.svc.Get(ctx, id2)
	if err != nil {
		t.Fatalf("get2: %v", err)
	}
	if obs2.MaturityTier != "archival" {
		t.Fatalf("NULL tier + 30d must recompute to archival, got %q", obs2.MaturityTier)
	}

	// Ordering sanity used by the monotonic max: fresh < mature < archival.
	if !(maturityRank["fresh"] < maturityRank["mature"]) || !(maturityRank["mature"] < maturityRank["archival"]) {
		t.Fatalf("tier ordering must be fresh < mature < archival, got %v", maturityRank)
	}
}

// mustExec is a tiny Exec helper for the tier persistence checks.
func mustExec(t *testing.T, fx *ownerFixture, query string, args ...any) {
	t.Helper()
	if _, err := fx.st.DB.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

// TestImportanceScoreQueryRanking covers 13.3: with the improve() opt-in
// enabled, mem_search (SearchOwnerScoped) re-ranks by importance_score — a
// multiplicative boost on the base relevance — so the high-importance
// observation ranks first. With improve() disabled the raw SQL order is
// byte-identical (no boost).
func TestImportanceScoreQueryRanking(t *testing.T) {
	fx := newImproveFixture(t, "imp-rank")
	ctx := context.Background()

	// Three observations sharing the "improvranks" token. The LOW one repeats
	// the token so it leads the raw SQL bm25 ranking — the importance boost
	// must flip the order.
	//
	//   impalpha: usage 50, 2d  → score 50*exp(-0.1) ≈ 45.2 (high)
	//   impbeta:  usage 10, 2d  → score 10*exp(-0.1) ≈ 9.05 (medium)
	//   improvgamma: usage 1, 2d → score 1*exp(-0.1)  ≈ 0.905 (low)
	high := seedImproveObs(t, fx, "improvranks alpha asset", "improvranks filler body one", fx.ownerA, 50, 2*24*time.Hour)
	mid := seedImproveObs(t, fx, "improvranks beta asset", "improvranks filler body two", fx.ownerA, 10, 2*24*time.Hour)
	low := seedImproveObs(t, fx, "improvranks gamma asset", "improvranks improvranks improvranks filler body three", fx.ownerA, 1, 2*24*time.Hour)

	// The stored columns must match the formula (the re-rank reads them).
	want := map[int64]float64{
		high: 50 * math.Exp(-0.05*2),
		mid:  10 * math.Exp(-0.05*2),
		low:  1 * math.Exp(-0.05*2),
	}
	for _, id := range []int64{high, mid, low} {
		obs, err := fx.svc.Get(ctx, id)
		if err != nil {
			t.Fatalf("get %d: %v", id, err)
		}
		if !obs.ImportanceScore.Valid || math.Abs(obs.ImportanceScore.Float64-want[id]) > 1e-4 {
			t.Fatalf("obs %d: stored importance = %+v, want %v (populated on save)", id, obs.ImportanceScore, want[id])
		}
		if obs.MaturityTier != "fresh" {
			t.Fatalf("obs %d: 2d must be tier fresh, got %q", id, obs.MaturityTier)
		}
	}

	// Raw SQL order (improve off): the low doc leads (3 token matches).
	fx.svc.SetImprove(ImproveConfig{Enabled: false})
	raw, err := fx.svc.SearchOwnerScoped(ctx, fx.ownerA, "agent", "improvranks", "any", "", 10)
	if err != nil {
		t.Fatalf("search (raw): %v", err)
	}
	rawCfg := fx.svc.ImproveConfig()
	fx.svc.SetImprove(ImproveConfig{
		Enabled: true, Cooldown: time.Nanosecond, Now: func() time.Time { return time.Now().UTC() },
	})
	if len(raw) != 3 {
		t.Fatalf("expected 3 hits, got %d", len(raw))
	}
	if ids(raw)[0] != low {
		t.Fatalf("setup: raw SQL order must be led by the low-importance doc, got %v", ids(raw))
	}

	// Enabled: the importance re-rank must put high first, and the order must
	// follow the stored scores (45.2 > 9.05 > 0.905).
	hits, err := fx.svc.SearchOwnerScoped(ctx, fx.ownerA, "agent", "improvranks", "any", "", 10)
	if err != nil {
		t.Fatalf("search (boosted): %v", err)
	}
	if len(hits) != 3 {
		t.Fatalf("expected 3 hits, got %d", len(hits))
	}
	got := ids(hits)
	wantOrder := []int64{high, mid, low}
	if !slices.Equal(got, wantOrder) {
		t.Fatalf("importance boost must re-rank high→mid→low: raw %v, got %v, want %v", ids(raw), got, wantOrder)
	}
	if got[0] == ids(raw)[0] {
		t.Fatalf("boosted order must differ from the raw SQL order (low doc must not lead)")
	}

	// Disabled again → the raw SQL order is restored verbatim (byte-identical
	// default search, no boost).
	fx.svc.SetImprove(ImproveConfig{Enabled: false})
	plain, err := fx.svc.SearchOwnerScoped(ctx, fx.ownerA, "agent", "improvranks", "any", "", 10)
	if err != nil {
		t.Fatalf("search (disabled): %v", err)
	}
	if !slices.Equal(ids(plain), ids(raw)) {
		t.Fatalf("disabled improve() must keep the raw SQL order: raw %v, got %v", ids(raw), ids(plain))
	}
	_ = rawCfg
}

// TestImportanceConfigurableDecay covers 13.4: the decay rate is
// configurable (a higher decay erodes old observations' importance faster,
// a lower decay retains it longer), the default is the documented 0.05/day,
// and tier thresholds follow the same pattern.
func TestImportanceConfigurableDecay(t *testing.T) {
	age := 30 * 24 * time.Hour
	count := 100

	// Default decay rate is 0.05 per day (the documented 22.3 anchor).
	def := defaultImportanceConfig()
	if def.DecayRate != 0.05 {
		t.Fatalf("default decay rate must be 0.05/day, got %v", def.DecayRate)
	}
	if got := ComputeImportanceScore(count, age, def.DecayRate); got < 22 || got > 23 {
		t.Fatalf("default decay: 100 retrievals @ 30d should be ≈22.3, got %v", got)
	}

	// High decay (0.2/day): older observations lose importance faster.
	highDecay := ComputeImportanceScore(count, age, 0.2)
	if !(highDecay < ComputeImportanceScore(count, age, 0.05)) {
		t.Fatalf("high decay must erode old scores faster: high=%v default=%v",
			highDecay, ComputeImportanceScore(count, age, 0.05))
	}

	// Low decay (0.01/day): older observations retain importance longer.
	lowDecay := ComputeImportanceScore(count, age, 0.01)
	if !(lowDecay > ComputeImportanceScore(count, age, 0.05)) {
		t.Fatalf("low decay must retain old scores longer: low=%v default=%v",
			lowDecay, ComputeImportanceScore(count, age, 0.05))
	}

	// The ranking flips as the decay tightens: at 0.2/day the 10-day-old doc
	// is worth more than the 30-day-old one even though it was retrieved 10x
	// less often.
	recent := ComputeImportanceScore(10, 10*24*time.Hour, 0.2)
	old := ComputeImportanceScore(100, 30*24*time.Hour, 0.2)
	if !(recent > old) {
		t.Fatalf("under high decay the recent doc must outrank the old one: recent=%v old=%v", recent, old)
	}

	// Tier thresholds follow the same pattern: the defaults are the brief's
	// values and the pure computation honors a custom set.
	if def.Thresholds.MatureAgeDays != 7 || def.Thresholds.ArchivalAgeDays != 30 || def.Thresholds.UnusedArchivalDays != 14 {
		t.Fatalf("default tier thresholds must be 7/30/14 days, got %+v", def.Thresholds)
	}
	custom := TierThresholds{MatureAgeDays: 7, ArchivalAgeDays: 30, UnusedArchivalDays: 14}
	if got := ComputeMaturityTierWithThresholds(3*24*time.Hour, 50, custom); got != "fresh" {
		t.Fatalf("custom thresholds: 3d must stay fresh, got %q", got)
	}
	if got := ComputeMaturityTierWithThresholds(8*24*time.Hour, 5, custom); got != "mature" {
		t.Fatalf("custom thresholds: 8d + usage must be mature, got %q", got)
	}
	if got := ComputeMaturityTierWithThresholds(15*24*time.Hour, 0, custom); got != "archival" {
		t.Fatalf("custom thresholds: 15d unused must be archival, got %q", got)
	}

	// SetImportance wiring: an explicit config is honored, and a zero-value
	// call reverts to the defaults (the malformed/absent-config fallback).
	fx := newOwnerFixture(t, "imp-cfg")
	fx.svc.SetImportance(ImportanceConfig{})
	if got := fx.svc.ImportanceConfig(); got.DecayRate != 0.05 {
		t.Fatalf("SetImportance(zero) must apply the 0.05/day default, got %v", got.DecayRate)
	}
	fx.svc.SetImportance(ImportanceConfig{DecayRate: 0.2,
		Thresholds: TierThresholds{MatureAgeDays: 3, ArchivalAgeDays: 21, UnusedArchivalDays: 10}})
	got := fx.svc.ImportanceConfig()
	if got.DecayRate != 0.2 || got.Thresholds.MatureAgeDays != 3 ||
		got.Thresholds.ArchivalAgeDays != 21 || got.Thresholds.UnusedArchivalDays != 10 {
		t.Fatalf("SetImportance must honor the explicit config, got %+v", got)
	}
	// A negative decay (malformed) falls back to the default.
	fx.svc.SetImportance(ImportanceConfig{DecayRate: -1})
	if got := fx.svc.ImportanceConfig(); got.DecayRate != 0.05 {
		t.Fatalf("negative decay must fall back to the 0.05/day default, got %v", got.DecayRate)
	}
}
