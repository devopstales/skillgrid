package memory

import (
	"context"

	"slices"
	"testing"
	"time"
)

// newImproveFixture opens a store and service, enabling the opt-in improve()
// feedback loop (014 step 08) with a known config, and a session for saves.
func newImproveFixture(t *testing.T, project string) *ownerFixture {
	t.Helper()
	fx := newOwnerFixture(t, project)
	fx.svc.SetImprove(ImproveConfig{
		Enabled:     true,
		Threshold:   5,
		MaxUsage:    50,
		BoostRate:   0.1,
		DecayRate:   0.25,
		// Cooldown: 1ns → the default 1-minute fallback does not kick in, so
		// consecutive searches in a test are never blocked by the cooldown.
		Cooldown:    time.Nanosecond,
		Now:         func() time.Time { return time.Now().UTC() },
	})
	return fx
}

// seedImproveObs inserts an observation directly with a controlled
// retrieval_usage and created_at (the FTS trigger keeps search in sync).
// createdOffset is how far in the past created_at sits relative to now.
func seedImproveObs(t *testing.T, fx *ownerFixture, title, body, owner string, usage int, createdOffset time.Duration) int64 {
	t.Helper()
	now := time.Now().UTC()
	fmtS := func(d time.Time) string { return d.UTC().Format(time.RFC3339) }
	created := fmtS(now.Add(-createdOffset))
	updated := fmtS(now)
	expires := fmtS(now.Add(7 * 24 * time.Hour))
	res, err := fx.st.DB.Exec(`
		INSERT INTO observations (
			session_id, type, title, content, project, scope,
			normalized_hash, created_at, updated_at, source,
			owner, visibility, status, retrieval_usage, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		fx.sessionID, "decision", title, body, fx.svc.ProjectID(), "",
		"", created, updated, "manual", owner, "private", "active", usage, expires,
	)
	if err != nil {
		t.Fatalf("seed observation %q: %v", title, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("seed last insert id %q: %v", title, err)
	}
	// Stamp the importance columns like Save() would (014 step 13), so the
	// query-time re-rank reads the same stored values production sees.
	fx.svc.stampImportance(context.Background(), id, usage, created)
	return id
}

// TestImproveBoostsHighUsageObservations covers @step-08 (Scenario:
// high-usage-observations-are-boosted): with improve() enabled, an
// observation with retrieval_usage=100 outranks one with retrieval_usage=10,
// which outranks one with retrieval_usage=0 (below threshold).
func TestImproveBoostsHighUsageObservations(t *testing.T) {
	fx := newImproveFixture(t, "improve-boost")
	ctx := context.Background()

	// The 0-usage observation matches the most query tokens (title token +
	// its distinctive "shared" body token), so the raw SQL bm25 order is
	// [0-usage, 100-usage, 10-usage] — the opposite of the boosted order.
	seedImproveObs(t, fx, "improvealpha highusage asset", "improvealpha filler body one", fx.ownerA, 100, 1*time.Hour)
	seedImproveObs(t, fx, "improvebeta midusage asset", "improvebeta filler body two", fx.ownerA, 10, 1*time.Hour)
	seedImproveObs(t, fx, "improveceta lowusage asset", "improveceta shared filler body three", fx.ownerA, 0, 1*time.Hour)

	// Capture the raw SQL ranking first (improve off): this is the
	// pre-improve order the boost must correct.
	rawCfg := fx.svc.ImproveConfig()
	fx.svc.SetImprove(ImproveConfig{Enabled: false})
	raw, err := fx.svc.SearchOwnerScoped(ctx, fx.ownerA, "agent",
		"improvealpha improvebeta improveceta shared", "any", "", 10)
	if err != nil {
		t.Fatalf("search (raw): %v", err)
	}
	fx.svc.SetImprove(rawCfg)
	if len(raw) != 3 {
		t.Fatalf("expected 3 hits, got %d", len(raw))
	}
	rawUsage := make([]int, len(raw))
	for i, h := range raw {
		rawUsage[i] = h.RetrievalUsage
	}
	if slices.Equal(rawUsage, []int{100, 10, 0}) {
		t.Fatalf("test setup broken: raw SQL order is already usage-desc (%v); "+
			"pick tokens/usage so the pre-improve order differs from the boosted order", rawUsage)
	}

	// Now with improve() enabled the same search must be re-ranked by
	// retrieval usage: 100 > 10 > 0.
	hits, err := fx.svc.SearchOwnerScoped(ctx, fx.ownerA, "agent",
		"improvealpha improvebeta improveceta shared", "any", "", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 3 {
		t.Fatalf("expected 3 hits, got %d", len(hits))
	}
	gotOrder := make([]int, len(hits))
	for i, h := range hits {
		gotOrder[i] = h.RetrievalUsage
	}
	// The raw-capture search above bumped each hit by one (governance
	// retrieval-usage), so the boosted values are +1: 101 > 11 > 1.
	want := []int{101, 11, 1}
	if !slices.Equal(gotOrder, want) {
		t.Fatalf("improve() must re-rank by retrieval usage desc: raw SQL order %v, got %v, want %v",
			rawUsage, gotOrder, want)
	}
}

// TestImproveDecaysNeverAccessed covers @step-08 (Scenario:
// never-accessed-observations-decay): an observation that has never been
// retrieved (retrieval_usage=0) and is older than the TTL decays in rank —
// it must rank below an accessed observation, and the decay is proportional
// to age (the older of two never-accessed observations ranks last).
//
// The SQL ORDER BY for these three documents is deterministic (bm25 ranks by
// relevance, not by age or insertion order), so the raw order is always
// [accessed, oldNever, olderNever]. improve() must keep accessed first (it is
// boosted by usage and never decays) and confirm the age-proportional ordering
// of the two never-accessed docs.
func TestImproveDecaysNeverAccessed(t *testing.T) {
	fx := newImproveFixture(t, "improve-decay")
	ctx := context.Background()

	ttl := fx.svc.EffectiveTTL()
	// never-accessed, age = 8 TTL (> TTL) — decays by (8-1)/1 * DecayRate.
	oldNever := seedImproveObs(t, fx, "decayold never accessed", "decayold filler body one", fx.ownerA, 0, 20*ttl)
	// never-accessed, age = 10 TTL (> TTL) — decays by (10-1)/1 * DecayRate.
	// Its body repeats "decayolder" so it matches the most query tokens and
	// leads the raw SQL bm25 ranking — the decay must then flip it behind
	// the 8-TTL doc (proportional-to-age proof).
	olderNever := seedImproveObs(t, fx, "decayolder never accessed", "decayolder decayolder decayolder filler body two", fx.ownerA, 0, 30*ttl)
	// accessed (usage 50 > 0 → never decays, boosted), fresh (age < TTL → no decay).
	// Usage 50 (the MaxUsage cap) gives a large boost so the accessed doc
	// decisively outranks the never-accessed ones regardless of base rank.
	accessed := seedImproveObs(t, fx, "decayfresh accessed asset", "decayfresh filler body three", fx.ownerA, 50, 1*time.Hour)

	// Fetch all three by id (no search, so no usage bumps) and verify the
	// SQL ranking would put the accessed doc first (bm25 is deterministic
	// here: each doc matches exactly one query token, so the order is by
	// document structure, not insertion order). We build the raw order from
	// a single SearchOwnerScoped with improve disabled, then re-enable and
	// call improve() directly on that slice — isolating the decay from any
	// usage bump.
	// Build the raw slice via a direct SQL query (no usage bump) so the
	// never-accessed docs stay at retrieval_usage=0 for the decay check.
	rawCfg := fx.svc.ImproveConfig()
	raw, err := rawSQLSearch(ctx, fx, "decayold decayolder decayfresh")
	if err != nil {
		t.Fatalf("search (raw): %v", err)
	}
	_ = rawCfg
	if len(raw) != 3 {
		t.Fatalf("expected 3 hits, got %d", len(raw))
	}
	rawOrder := ids(raw)
	t.Logf("raw SQL order: %v (accessed=%d oldNever=%d olderNever=%d)", rawOrder, accessed, oldNever, olderNever)

	// improve() on the raw slice must rank the accessed doc first (boosted,
	// no decay) and the 8-TTL never-accessed doc before the 10-TTL one
	// (decay proportional to age: (10-1)/1 > (8-1)/1 → 10-TTL decays more).
	reranked := fx.svc.improve(ctx, raw)
	got := ids(reranked)
	if got[0] != accessed {
		t.Fatalf("accessed observation must rank first (no decay for used obs): raw %v, got %v", rawOrder, got)
	}
	// The 10-TTL never-accessed doc — which led the raw SQL ranking by
	// matching more query tokens — must now rank BELOW the 8-TTL one: decay
	// is proportional to age ((10-1)/1 * DecayRate > (8-1)/1 * DecayRate),
	// and the extra decay outweighs its 1-position base-rank lead.
	if got[1] != oldNever || got[2] != olderNever {
		t.Fatalf("decay must be proportional to age (8-TTL before 10-TTL): raw %v, got %v, want [%d %d %d]",
			rawOrder, got, accessed, oldNever, olderNever)
	}
}

// ids returns the observation IDs in slice order.
func ids(obs []Observation) []int64 {
	out := make([]int64, len(obs))
	for i, o := range obs {
		out[i] = o.ID
	}
	return out
}

// TestImproveDisabledNoRegression covers @step-08 (Scenarios:
// improve-is-opt-in / does-not-regress-when-disabled): with the
// mnemonic.improve config absent (Enabled=false, the default), mem_search
// returns the exact pre-improve SQL ordering — no boost, no decay — and the
// config loader defaults the section off. The loop is also gated by a
// cooldown: consecutive searches within the window are not re-ranked again.
func TestImproveDisabledNoRegression(t *testing.T) {
	fx := newOwnerFixture(t, "improve-disabled")
	ctx := context.Background()

	// No SetImprove call → the zero-value cfg is disabled (the opt-in
	// default, matching a config file without the mnemonic.improve section).
	cfg := fx.svc.ImproveConfig()
	if cfg.Enabled {
		t.Fatalf("improve() must be disabled by default (opt-in), got Enabled=true")
	}

	// Mixed usage: a hot doc (100), a mid doc (10), a cold fresh doc (0).
	// The cold doc matches the most query tokens so the SQL ranking is
	// [cold, hot, mid] — the OPPOSITE of the boosted order [hot, mid, cold].
	seedImproveObs(t, fx, "disablalpha cold fresh", "disablalpha disablalpha disablalpha filler body one", fx.ownerA, 0, 1*time.Hour)
	seedImproveObs(t, fx, "disablbeta hot used", "disablbeta filler body two", fx.ownerA, 100, 1*time.Hour)
	seedImproveObs(t, fx, "disablgamma mid used", "disablgamma filler body three", fx.ownerA, 10, 1*time.Hour)

	// Disabled: the search must return the raw SQL order, untouched.
	hits, err := fx.svc.SearchOwnerScoped(ctx, fx.ownerA, "agent",
		"disablalpha disablbeta disablgamma", "any", "", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 3 {
		t.Fatalf("expected 3 hits, got %d", len(hits))
	}
	usageOrder := make([]int, len(hits))
	for i, h := range hits {
		usageOrder[i] = h.RetrievalUsage
	}
	// The SQL ranking (cold first by token match) must be preserved verbatim:
	// no boost/decay may reorder it while disabled.
	wantSQL := []int{0, 100, 10}
	if !slices.Equal(usageOrder, wantSQL) {
		t.Fatalf("disabled improve() must keep the pre-improve SQL order: got %v, want %v", usageOrder, wantSQL)
	}

	// Re-enabling the loop (the config opt-in) must change the order — proof
	// the disabled path skipped the re-rank entirely.
	fx.svc.SetImprove(ImproveConfig{
		Enabled:     true,
		Threshold:   5,
		MaxUsage:    50,
		BoostRate:   0.1,
		DecayRate:   0.25,
		Cooldown:    time.Nanosecond,
		Now:         func() time.Time { return time.Now().UTC() },
	})
	enabled, err := fx.svc.SearchOwnerScoped(ctx, fx.ownerA, "agent",
		"disablalpha disablbeta disablgamma", "any", "", 10)
	if err != nil {
		t.Fatalf("search (enabled): %v", err)
	}
	if slices.Equal(ids(hits), ids(enabled)) {
		t.Fatalf("enabled improve() must re-rank differently from the disabled SQL order")
	}
	if len(enabled) != 3 || enabled[0].RetrievalUsage != 101 {
		t.Fatalf("enabled improve() must boost the high-usage doc first, got %+v", usageOrders(enabled))
	}

	// Cooldown: a fresh service (clean usage counters) with a long cooldown —
	// the first search re-ranks (boosted), the second (within the window)
	// must NOT re-rank again and returns the raw SQL order.
	fx2 := newOwnerFixture(t, "improve-cooldown")
	seedImproveObs(t, fx2, "disablalpha cold fresh", "disablalpha disablalpha disablalpha filler body one", fx2.ownerA, 0, 1*time.Hour)
	seedImproveObs(t, fx2, "disablbeta hot used", "disablbeta filler body two", fx2.ownerA, 100, 1*time.Hour)
	seedImproveObs(t, fx2, "disablgamma mid used", "disablgamma filler body three", fx2.ownerA, 10, 1*time.Hour)
	fx2.svc.SetImprove(ImproveConfig{
		Enabled:     true,
		Threshold:   5,
		MaxUsage:    50,
		BoostRate:   0.1,
		DecayRate:   0.25,
		Cooldown:    time.Hour,
		Now:         func() time.Time { return time.Now().UTC() },
	})
	first, err := fx2.svc.SearchOwnerScoped(ctx, fx2.ownerA, "agent",
		"disablalpha disablbeta disablgamma", "any", "", 10)
	if err != nil {
		t.Fatalf("search (cooldown 1st): %v", err)
	}
	second, err := fx2.svc.SearchOwnerScoped(ctx, fx2.ownerA, "agent",
		"disablalpha disablbeta disablgamma", "any", "", 10)
	if err != nil {
		t.Fatalf("search (cooldown 2nd): %v", err)
	}
	// The second search is inside the cooldown window → returned as the raw
	// SQL order (cold doc first by token match), so it differs from the
	// boosted first search (hot doc first).
	if slices.Equal(ids(first), ids(second)) {
		t.Fatalf("second search within the cooldown must not re-rank (got identical order %v)", ids(first))
	}
	// The first search is boosted (hot doc first). Its values are the
	// pre-bump usage (the bump happens after the re-rank): 100 > 10 > 0.
	if first[0].RetrievalUsage != 100 {
		t.Fatalf("first search must be boosted (hot doc first), got %+v", usageOrders(first))
	}
	// The second search is within the cooldown → raw SQL order (cold doc
	// first by token match), values bumped once by the first search: cold=1.
	if second[0].RetrievalUsage != 1 {
		t.Fatalf("second search within cooldown must be the raw SQL order (cold doc first), got %+v", usageOrders(second))
	}
}

// usageOrders returns the retrieval-usage values in slice order (for error
// messages).
func usageOrders(obs []Observation) []int {
	out := make([]int, len(obs))
	for i, o := range obs {
		out[i] = o.RetrievalUsage
	}
	return out
}

// rawSQLSearch runs the same FTS query as SearchOwnerScoped but WITHOUT the
// retrieval-usage bump, so tests can capture the pre-improve SQL ranking with
// usage values untouched (the decay check needs usage=0 to be meaningful).
func rawSQLSearch(ctx context.Context, fx *ownerFixture, query string) ([]Observation, error) {
	db := fx.st.DB
	ftsQuery := buildFTSQuery(query, "any")
	rows, err := db.QueryContext(ctx, `
		SELECT o.id, o.session_id, o.type, o.title, o.content, o.project, o.scope,
		       o.topic_key, o.source, o.normalized_hash, o.revision_count, o.prompt_id, o.created_at, o.updated_at,
		       COALESCE(o.pinned, 0), COALESCE(o.duplicate_count, 0), o.last_seen_at, o.expires_at, o.tool_name,
		       o.owner, COALESCE(o.visibility, 'private'), COALESCE(o.status, 'active'), COALESCE(o.retrieval_usage, 0),
		       o.importance_score, o.recency_decay, o.maturity_tier
		FROM observations o
		INNER JOIN observations_fts ON observations_fts.rowid = o.id
		WHERE observations_fts MATCH ? AND o.deleted_at IS NULL AND o.project = ?
 		  AND (o.expires_at IS NULL OR o.expires_at = '' OR strftime('%s', o.expires_at) > strftime('%s', 'now'))
		ORDER BY COALESCE(o.pinned, 0) DESC, bm25(observations_fts)
		LIMIT 10`,
		ftsQuery, fx.svc.ProjectID(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObservations(rows)
}
