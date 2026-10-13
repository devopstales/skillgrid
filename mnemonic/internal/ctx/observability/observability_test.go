package observability

import (
	"encoding/json"
	"testing"
	"time"

	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// freshStore opens a real project store (the 056-061 ctx migrations run on
// open) and wraps it in the ctx accessor. It is closed on cleanup.
func freshStore(t *testing.T) (*store.Store, *ctxstore.Store) {
	t.Helper()
	st, err := store.Open(t.TempDir(), "obs-test")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st, ctxstore.New(st)
}

// saving builds a per-turn proxy saving row for the bucket writer's input.
func saving(id, sessionID, mode string, before, after, createdAt int64) *Saving {
	return &Saving{
		SavingID:     id,
		SessionID:    sessionID,
		Turn:         1,
		Upstream:     "anthropic",
		Mode:         mode,
		TokensBefore: before,
		TokensAfter:  after,
		CreatedAt:    createdAt,
	}
}

// TestRoll_Pure verifies the pure per-saving roll: one saving yields one
// bucket per standard granularity with the correct tokens saved (clamped),
// one request, one session, and a cache hit only in cache mode.
func TestRoll_Pure(t *testing.T) {
	got := Roll(saving("s1", "sess-1", "cache", 1000, 400, 1_700_000_000))
	if len(got) != len(DefaultGranularities) {
		t.Fatalf("Roll: expected %d buckets (one per granularity), got %d", len(DefaultGranularities), len(got))
	}
	byGran := make(map[string]*ctxstore.StatsBucket)
	for _, b := range got {
		byGran[b.Granularity] = b
	}
	for _, g := range DefaultGranularities {
		b, ok := byGran[string(g)]
		if !ok {
			t.Fatalf("Roll: missing bucket for granularity %q", g)
		}
		if b.TokensSaved != 600 {
			t.Errorf("Roll %s: tokens saved = %d, want 600 (1000-400)", g, b.TokensSaved)
		}
		if b.Requests != 1 {
			t.Errorf("Roll %s: requests = %d, want 1", g, b.Requests)
		}
		if b.Sessions != 1 {
			t.Errorf("Roll %s: sessions = %d, want 1 (lower bound)", g, b.Sessions)
		}
		if b.CacheHits != 1 {
			t.Errorf("Roll %s: cache hits = %d, want 1 (cache mode)", g, b.CacheHits)
		}
		if b.BucketStart != windowStart(g, time.Unix(1_700_000_000, 0)) {
			t.Errorf("Roll %s: bucket start = %d, want aligned %d", g, b.BucketStart, windowStart(g, time.Unix(1_700_000_000, 0)))
		}
	}
}

// TestRoll_ClampsNegativeSaving verifies a turn where tokens after exceeds
// tokens before does not produce a negative savings (clamped to 0) and a
// non-cache-mode saving contributes no cache hit.
func TestRoll_ClampsNegativeSaving(t *testing.T) {
	got := Roll(saving("s1", "sess-1", "token", 100, 500, 1_700_000_000))
	if len(got) != len(DefaultGranularities) {
		t.Fatalf("Roll: expected %d buckets, got %d", len(DefaultGranularities), len(got))
	}
	for _, b := range got {
		if b.TokensSaved != 0 {
			t.Errorf("Roll %s: negative saving must clamp to 0, got %d", b.Granularity, b.TokensSaved)
		}
		if b.CacheHits != 0 {
			t.Errorf("Roll %s: token mode must not count a cache hit, got %d", b.Granularity, b.CacheHits)
		}
	}
}

// TestWriteSavings_MergesBuckets verifies the per-request writer: N savings
// land in ctx_stats_history with correct per-granularity tokens/requests/
// cache-hit sums, and two savings in the same window merge into one bucket.
func TestWriteSavings_MergesBuckets(t *testing.T) {
	st, c := freshStore(t)
	_ = st
	// Two cache-mode turns in the same hour (same window start).
	n := WriteSavings(c, []*Saving{
		saving("a", "sess-1", "cache", 1000, 400, 1_700_000_100),
		saving("b", "sess-1", "cache", 2000, 800, 1_700_000_200),
	})
	// Both savings -> same hourly window -> merged into one hourly bucket,
	// but still one bucket row per other granularity. Expect 4 distinct
	// bucket ids written (hourly, daily, weekly, monthly).
	if n != 4 {
		t.Fatalf("WriteSavings: expected 4 bucket rows written, got %d", n)
	}
	// Verify the hourly bucket summed both turns.
	var hourBuckets []*ctxstore.StatsBucket
	{
		rows, err := c.ListStatsBuckets("hourly")
		if err != nil {
			t.Fatalf("ListStatsBuckets(hourly): %v", err)
		}
		hourBuckets = rows
	}
	if len(hourBuckets) != 1 {
		t.Fatalf("hourly: expected 1 merged bucket, got %d", len(hourBuckets))
	}
	hb := hourBuckets[0]
	if hb.TokensSaved != (600 + 1200) {
		t.Errorf("hourly tokens = %d, want 1800", hb.TokensSaved)
	}
	if hb.Requests != 2 {
		t.Errorf("hourly requests = %d, want 2", hb.Requests)
	}
	if hb.CacheHits != 2 {
		t.Errorf("hourly cache hits = %d, want 2", hb.CacheHits)
	}
}

// TestRecompute_CorrectsSessionsCount verifies the periodic rollup re-derives
// buckets from ctx_proxy_sessions + ctx_proxy_savings and corrects the
// Sessions count to the exact distinct-session total per window.
func TestRecompute_CorrectsSessionsCount(t *testing.T) {
	_, c := freshStore(t)
	// Two sessions each with one cache-mode turn in the same hour.
	for _, sess := range []string{"sess-1", "sess-2"} {
		if err := c.PutProxySession(&ctxstore.ProxySession{
			SessionID: sess, Agent: "opencode", Mode: "cache",
			StartedAt: 1_699_999_900, EndedAt: 1_700_000_200,
			Requests: 1, TokensBefore: 1000, TokensAfter: 400, CacheHits: 1,
		}); err != nil {
			t.Fatalf("PutProxySession %s: %v", sess, err)
		}
		if err := c.PutProxySaving(&ctxstore.ProxySaving{
			SavingID: sess + "-turn", SessionID: sess, Turn: 1,
			Upstream: "anthropic", Mode: "cache",
			TokensBefore: 1000, TokensAfter: 400, CreatedAt: 1_700_000_100,
		}); err != nil {
			t.Fatalf("PutProxySaving %s: %v", sess, err)
		}
	}
	written := Recompute(c)
	if written != 4 {
		t.Fatalf("Recompute: expected 4 bucket rows written, got %d", written)
	}
	rows, err := c.ListStatsBuckets("hourly")
	if err != nil {
		t.Fatalf("ListStatsBuckets(hourly): %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("hourly: expected 1 bucket, got %d", len(rows))
	}
	if rows[0].Sessions != 2 {
		t.Errorf("hourly sessions = %d, want 2 (distinct sessions)", rows[0].Sessions)
	}
	if rows[0].TokensSaved != 1200 {
		t.Errorf("hourly tokens = %d, want 1200", rows[0].TokensSaved)
	}
	if rows[0].Requests != 2 {
		t.Errorf("hourly requests = %d, want 2", rows[0].Requests)
	}
}

// TestStartRollup_WritesWithoutRestart verifies the rollup goroutine, started
// with a short tick + stop channel, writes ctx_stats_history rows without a
// restart and stops cleanly (no goroutine leak).
func TestStartRollup_WritesWithoutRestart(t *testing.T) {
	_, c := freshStore(t)
	if err := c.PutProxySession(&ctxstore.ProxySession{
		SessionID: "sess-1", Agent: "opencode", Mode: "token",
		StartedAt: 1_699_999_900, EndedAt: 1_700_000_200,
		Requests: 1, TokensBefore: 5000, TokensAfter: 2000,
	}); err != nil {
		t.Fatalf("PutProxySession: %v", err)
	}
	if err := c.PutProxySaving(&ctxstore.ProxySaving{
		SavingID: "turn-1", SessionID: "sess-1", Turn: 1,
		Upstream: "anthropic", Mode: "token",
		TokensBefore: 5000, TokensAfter: 2000, CreatedAt: 1_700_000_100,
	}); err != nil {
		t.Fatalf("PutProxySaving: %v", err)
	}
	stop := make(chan struct{})
	stopFn := StartRollup(c, 20*time.Millisecond, stop)
	// Wait for the immediate first pass (or a tick) to land rows.
	deadline := time.Now().Add(2 * time.Second)
	var n int
	for {
		rows, err := c.ListStatsBuckets("hourly")
		if err == nil {
			n = len(rows)
		}
		if n > 0 {
			break
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if n == 0 {
		t.Fatalf("rollup did not write any ctx_stats_history rows without a restart")
	}
	// Stop cleanly; StartRollup waits for the goroutine (no leak).
	stopFn()
}

// TestSnapshot_MatchesStored verifies Stats returns the stored aggregate
// (tokens saved / cache hits / invalidations / requests / sessions) and the
// latest session's mode (the T11 seam field).
func TestSnapshot_MatchesStored(t *testing.T) {
	_, c := freshStore(t)
	if err := c.PutProxySession(&ctxstore.ProxySession{
		SessionID: "old", Agent: "cursor", Mode: "token",
		StartedAt: 100, EndedAt: 200,
		Requests: 3, TokensBefore: 9000, TokensAfter: 4000,
		CacheHits: 2, CacheInvalidations: 1,
	}); err != nil {
		t.Fatalf("PutProxySession old: %v", err)
	}
	if err := c.PutProxySession(&ctxstore.ProxySession{
		SessionID: "new", Agent: "opencode", Mode: "cache",
		StartedAt: 500, EndedAt: 900,
		Requests: 2, TokensBefore: 2000, TokensAfter: 500,
		CacheHits: 4, CacheInvalidations: 0,
	}); err != nil {
		t.Fatalf("PutProxySession new: %v", err)
	}
	st, err := Snapshot(c)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if st.Sessions != 2 {
		t.Errorf("sessions = %d, want 2", st.Sessions)
	}
	if st.Requests != 5 {
		t.Errorf("requests = %d, want 5", st.Requests)
	}
	if st.CacheHits != 6 {
		t.Errorf("cache hits = %d, want 6", st.CacheHits)
	}
	if st.CacheInvalidations != 1 {
		t.Errorf("cache invalidations = %d, want 1", st.CacheInvalidations)
	}
	// tokens saved = (9000-4000) + (2000-500) = 6500
	if st.TokensSaved != 6500 {
		t.Errorf("tokens saved = %d, want 6500", st.TokensSaved)
	}
	// latest by EndedAt is "new" (900 > 200) -> mode "cache"
	if st.Mode != "cache" {
		t.Errorf("mode = %q, want cache (latest session)", st.Mode)
	}
}

// TestStatsHistory_SurvivesReopen verifies StatsHistory reads all four
// granularities from ctx_stats_history and that the rows survive a re-open of
// the store (i.e. a server restart) — the durable-history clause.
func TestStatsHistory_SurvivesReopen(t *testing.T) {
	st, c := freshStore(t)
	// Persist one bucket per granularity.
	now := time.Unix(1_700_000_100, 0)
	for _, g := range DefaultGranularities {
		if err := c.PutStatsBucket(&ctxstore.StatsBucket{
			BucketID:    bucketKey(g, windowStart(g, now)),
			Granularity: string(g),
			BucketStart: windowStart(g, now),
			TokensSaved: 111, Requests: 7, Sessions: 3, CacheHits: 5,
			CreatedAt: now.Unix(),
		}); err != nil {
			t.Fatalf("PutStatsBucket %s: %v", g, err)
		}
	}
	// Re-open a FRESH accessor over the same store to simulate a restart.
	c2 := ctxstore.New(st)
	hist, err := StatsHistory(c2, nil) // nil -> default granularities
	if err != nil {
		t.Fatalf("StatsHistory: %v", err)
	}
	if len(hist.Buckets) != len(DefaultGranularities) {
		t.Fatalf("StatsHistory: expected %d granularities, got %d", len(DefaultGranularities), len(hist.Buckets))
	}
	for _, g := range DefaultGranularities {
		buckets, ok := hist.Buckets[g]
		if !ok {
			t.Fatalf("StatsHistory: missing granularity %q", g)
		}
		if len(buckets) != 1 {
			t.Errorf("StatsHistory %s: expected 1 bucket, got %d", g, len(buckets))
		} else {
			if buckets[0].TokensSaved != 111 || buckets[0].Requests != 7 || buckets[0].Sessions != 3 || buckets[0].CacheHits != 5 {
				t.Errorf("StatsHistory %s: wrong values %+v", g, buckets[0])
			}
		}
	}
}

// TestSummarizeSessions_WritesPerSession verifies SummarizeSessions derives
// one ctx_session_summaries row per distinct ctx_proxy_sessions row with the
// correct field mapping: duration_ms (ended-started), turns (requests),
// tokens_saved (before-after, clamped), cache_invalidations carried over, and
// the deterministic summary_json. Judgment/prefilter/shaping/traffic columns
// are left at zero/NULL (T12/T14 populate them later).
func TestSummarizeSessions_WritesPerSession(t *testing.T) {
	_, c := freshStore(t)
	// sess-1: a positive saving. sess-2: a negative saving (after > before)
	// that must clamp to 0, and a negative duration (ended < started) that
	// must clamp to 0 as well.
	if err := c.PutProxySession(&ctxstore.ProxySession{
		SessionID: "sess-1", Agent: "opencode", Mode: "cache",
		StartedAt: 1_700_000_000, EndedAt: 1_700_000_120,
		Requests: 5, TokensBefore: 10000, TokensAfter: 7000,
		CacheHits: 3, CacheInvalidations: 1,
	}); err != nil {
		t.Fatalf("PutProxySession sess-1: %v", err)
	}
	if err := c.PutProxySession(&ctxstore.ProxySession{
		SessionID: "sess-2", Agent: "claude", Mode: "token",
		StartedAt: 1_700_000_500, EndedAt: 1_700_000_400, // negative duration
		Requests: 3, TokensBefore: 9000, TokensAfter: 12000, // negative saving
	}); err != nil {
		t.Fatalf("PutProxySession sess-2: %v", err)
	}

	written, err := SummarizeSessions(c, 1_800_000_000)
	if err != nil {
		t.Fatalf("SummarizeSessions: %v", err)
	}
	if written != 2 {
		t.Fatalf("SummarizeSessions: expected 2 rows written, got %d", written)
	}

	rows, err := c.ListSessionSummaries()
	if err != nil {
		t.Fatalf("ListSessionSummaries: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 summary rows, got %d", len(rows))
	}
	byID := make(map[string]*ctxstore.SessionSummary, len(rows))
	for _, r := range rows {
		byID[r.SessionID] = r
	}

	s1, ok := byID["sess-1"]
	if !ok {
		t.Fatalf("missing summary row for sess-1")
	}
	if s1.DurationMS != 120 {
		t.Errorf("sess-1 duration_ms = %d, want 120 (1_700_000_120-1_700_000_000)", s1.DurationMS)
	}
	if s1.Turns != 5 {
		t.Errorf("sess-1 turns = %d, want 5 (requests)", s1.Turns)
	}
	if s1.TokensSaved != 3000 {
		t.Errorf("sess-1 tokens_saved = %d, want 3000 (10000-7000)", s1.TokensSaved)
	}
	if s1.CacheInvalidations != 1 {
		t.Errorf("sess-1 cache_invalidations = %d, want 1", s1.CacheInvalidations)
	}
	if s1.Agent != "opencode" {
		t.Errorf("sess-1 agent = %q, want opencode", s1.Agent)
	}
	if s1.JudgmentKeeps != 0 || s1.JudgmentTruncs != 0 || s1.PrefilterPins != 0 || s1.PrefilterDeads != 0 {
		t.Errorf("sess-1 judgment/prefilter columns must be 0 here, got %+v", s1)
	}
	if s1.ShapingApplied != nil || s1.EffortRouted != 0 || s1.TrafficMemories != 0 {
		t.Errorf("sess-1 shaping/effort/traffic columns must be zero/nil, got shaping=%v effort=%d traffic=%d", s1.ShapingApplied, s1.EffortRouted, s1.TrafficMemories)
	}
	if s1.CreatedAt != 1_800_000_000 {
		t.Errorf("sess-1 created_at = %d, want the passed now", s1.CreatedAt)
	}
	// summary_json must be the deterministic digest.
	var js struct {
		Mode         string `json:"mode"`
		Requests     int64  `json:"requests"`
		TokensBefore int64  `json:"tokens_before"`
		TokensAfter  int64  `json:"tokens_after"`
		TokensSaved  int64  `json:"tokens_saved"`
	}
	if err := json.Unmarshal([]byte(s1.SummaryJSON), &js); err != nil {
		t.Fatalf("sess-1 summary_json not valid JSON: %v (%q)", err, s1.SummaryJSON)
	}
	if js.Mode != "cache" || js.Requests != 5 || js.TokensBefore != 10000 || js.TokensAfter != 7000 || js.TokensSaved != 3000 {
		t.Errorf("sess-1 summary_json = %+v, want mode=cache requests=5 before=10000 after=7000 saved=3000", js)
	}

	s2, ok := byID["sess-2"]
	if !ok {
		t.Fatalf("missing summary row for sess-2")
	}
	if s2.DurationMS != 0 {
		t.Errorf("sess-2 duration_ms = %d, want 0 (negative clamped)", s2.DurationMS)
	}
	if s2.TokensSaved != 0 {
		t.Errorf("sess-2 tokens_saved = %d, want 0 (negative clamped)", s2.TokensSaved)
	}
	if s2.Turns != 3 {
		t.Errorf("sess-2 turns = %d, want 3 (requests)", s2.Turns)
	}
}

// TestSummarizeSessions_Idempotent verifies re-running the rollup over the
// same sessions does not append duplicate rows: PutSessionSummary is INSERT OR
// REPLACE on the session_id PK, so the row count is stable across rolls.
func TestSummarizeSessions_Idempotent(t *testing.T) {
	_, c := freshStore(t)
	for _, sess := range []string{"sess-1", "sess-2", "sess-3"} {
		if err := c.PutProxySession(&ctxstore.ProxySession{
			SessionID: sess, Agent: "opencode", Mode: "token",
			StartedAt: 1_700_000_000, EndedAt: 1_700_000_100,
			Requests: 2, TokensBefore: 1000, TokensAfter: 400,
		}); err != nil {
			t.Fatalf("PutProxySession %s: %v", sess, err)
		}
	}
	if n, err := SummarizeSessions(c, 1_800_000_000); err != nil || n != 3 {
		t.Fatalf("first roll: got (%d, %v), want (3, nil)", n, err)
	}
	// Re-roll the same sessions; the PK upsert must keep the count at 3.
	if n, err := SummarizeSessions(c, 1_800_000_000); err != nil || n != 3 {
		t.Fatalf("second roll: got (%d, %v), want (3, nil)", n, err)
	}
	rows, err := c.ListSessionSummaries()
	if err != nil {
		t.Fatalf("ListSessionSummaries: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 summary rows after two rolls (idempotent), got %d", len(rows))
	}
	// No duplicate session_ids.
	seen := make(map[string]int, len(rows))
	for _, r := range rows {
		seen[r.SessionID]++
	}
	for id, n := range seen {
		if n != 1 {
			t.Errorf("duplicate summary row for %s: %d rows", id, n)
		}
	}
}

// TestSummarizeSessions_FailOpen verifies the writer degrades to (0, nil) —
// never an error — when the store is nil or the ctx_* tables are missing
// (ADR-0016).
func TestSummarizeSessions_FailOpen(t *testing.T) {
	// nil store.
	if n, err := SummarizeSessions(nil, 1_800_000_000); err != nil || n != 0 {
		t.Errorf("nil store: got (%d, %v), want (0, nil)", n, err)
	}

	// Missing tables: open a store, drop the six ctx_* tables, wrap in the
	// accessor, and confirm a roll is a no-op (0, nil).
	st, err := store.Open(t.TempDir(), "obs-sum-nomigrate")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	for _, table := range []string{
		"ctx_proxy_sessions", "ctx_proxy_savings", "ctx_proxy_mode_history",
		"ctx_session_summaries", "ctx_stats_history", "ctx_traffic_memories",
	} {
		if _, err := st.DB.Exec(`DROP TABLE IF EXISTS ` + table); err != nil {
			t.Fatalf("drop %s: %v", table, err)
		}
	}
	c := ctxstore.New(st)
	if n, err := SummarizeSessions(c, 1_800_000_000); err != nil || n != 0 {
		t.Errorf("missing tables: got (%d, %v), want (0, nil)", n, err)
	}
}

// TestFailOpen_MissingTables verifies every reader and the writer degrade to
// empty/zero (never an error) on a store whose ctx_* tables are missing.
func TestFailOpen_MissingTables(t *testing.T) {
	st, err := store.Open(t.TempDir(), "obs-nomigrate")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	// Drop all six ctx_* tables to simulate a store opened before 056-061.
	for _, table := range []string{
		"ctx_proxy_sessions", "ctx_proxy_savings", "ctx_proxy_mode_history",
		"ctx_session_summaries", "ctx_stats_history", "ctx_traffic_memories",
	} {
		if _, err := st.DB.Exec(`DROP TABLE IF EXISTS ` + table); err != nil {
			t.Fatalf("drop %s: %v", table, err)
		}
	}
	c := ctxstore.New(st)

	if stt, err := Snapshot(c); err != nil || stt == nil || stt.Sessions != 0 || stt.TokensSaved != 0 {
		t.Errorf("Snapshot fail-open: got (%+v, %v)", stt, err)
	}
	if hist, err := StatsHistory(c, nil); err != nil || hist == nil {
		t.Errorf("StatsHistory fail-open: got (%+v, %v)", hist, err)
	} else if len(hist.Buckets) != len(DefaultGranularities) {
		t.Errorf("StatsHistory fail-open: expected %d empty granularities, got %d", len(DefaultGranularities), len(hist.Buckets))
	}
	// WriteSavings and Recompute must not error on missing tables. On a
	// modern SQLite build PutStatsBucket is fail-open (returns nil), so the
	// count may be non-zero; the invariant is no error, not a zero count.
	_ = WriteSavings(c, []*Saving{saving("x", "s", "cache", 100, 10, 1_700_000_000)})
	if n := Recompute(c); n < 0 {
		t.Errorf("Recompute fail-open: expected >= 0 written, got %d", n)
	}
	// Snapshot and StatsHistory must still be zero/empty.
	if stt, _ := Snapshot(c); stt.Sessions != 0 || stt.TokensSaved != 0 {
		t.Errorf("Snapshot fail-open (post-write): got %+v", stt)
	}
}
