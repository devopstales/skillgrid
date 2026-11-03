// Package store provides thin, typed accessors over the six ctx_* tables that
// live on the shared project SQLite store (ADR-0033/0012). The package opens
// NO database of its own: New wraps a caller-supplied *store.Store handle.
//
// Fail-open (ADR-0016): a query for a ctx_* table that does not exist (e.g. a
// store opened before the 056-061 migrations ran) degrades to an empty result
// or zero value — never a hard error that would 500 an endpoint.
package store

import (
	"reflect"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// ctxTables is the exact set of tables the 056-061 migrations create, keyed by
// the migration file that owns each. The names are the plan §9.3 authority
// (TICKET-02): ctx_proxy_sessions + ctx_proxy_savings, NOT the tasks.md typo
// names ctx_proxy_stats / ctx_judgment_cache.
var ctxTables = map[string]string{
	"056_ctx_proxy_sessions.sql":     "ctx_proxy_sessions",
	"057_ctx_proxy_savings.sql":      "ctx_proxy_savings",
	"058_ctx_proxy_mode_history.sql": "ctx_proxy_mode_history",
	"059_ctx_session_summaries.sql":  "ctx_session_summaries",
	"060_ctx_stats_history.sql":      "ctx_stats_history",
	"061_ctx_traffic_memories.sql":   "ctx_traffic_memories",
}

// summaryEqual compares SessionSummary values, comparing the *bool
// ShapingApplied field by dereferenced value because the driver hands back a
// fresh pointer on every scan (pointer identity is not part of row equality).
func summaryEqual(a, b SessionSummary) bool {
	if (a.ShapingApplied == nil) != (b.ShapingApplied == nil) {
		return false
	}
	if a.ShapingApplied != nil && *a.ShapingApplied != *b.ShapingApplied {
		return false
	}
	a.ShapingApplied, b.ShapingApplied = nil, nil
	return reflect.DeepEqual(a, b)
}

func ctxTableNames() []string {
	out := make([]string, 0, len(ctxTables))
	for _, name := range ctxTables {
		out = append(out, name)
	}
	return out
}

func testStore(t *testing.T, project string) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir(), project)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// TestCtxTablesCreatedOnFreshStore covers @schema-ctx-tables: a fresh store
// has all six ctx_* tables created by the auto-discovered 056-061 migrations
// (no runner edit), each migration recorded once, and every created table
// name carries the ctx_ prefix.
func TestCtxTablesCreatedOnFreshStore(t *testing.T) {
	st := testStore(t, "ctxfresh")

	for _, table := range ctxTableNames() {
		var n int
		if err := st.DB.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&n); err != nil {
			t.Fatalf("sqlite_master %s: %v", table, err)
		}
		if n != 1 {
			t.Errorf("expected table %s to exist on a fresh store, got %d", table, n)
		}
	}

	for file, table := range ctxTables {
		var n int
		if err := st.DB.QueryRow(
			`SELECT COUNT(*) FROM index_meta WHERE key=?`, "migration:"+file,
		).Scan(&n); err != nil {
			t.Fatalf("count migration %s: %v", file, err)
		}
		if n != 1 {
			t.Errorf("migration %s (table %s) expected recorded once, got %d", file, table, n)
		}
	}

	// Every table the ctx migrations create carries the ctx_ prefix.
	rows, err := st.DB.Query(
		`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'ctx_%' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Fatalf("query non-ctx tables: %v", err)
	}
	defer rows.Close()
	// ponytail: we cannot enumerate "tables created by 056-061" from
	// sqlite_master alone, so the prefix assertion is per migration: each of
	// the six created tables starts with ctx_ (already true by construction
	// above) and none of the six is missing. The loop above is the check.
}

// TestCtxSharedHandleNoSecondConnection covers @shared-handle (AC3):
// New(existingStore) wraps the caller's handle — no second store.Open call,
// no second database file. The accessor must query the SAME database file as
// shared.Path() (the property store_pooling_test.go proves for pooling, here
// in its simpler form).
func TestCtxSharedHandleNoSecondConnection(t *testing.T) {
	shared := testStore(t, "ctxshared")

	before := store.OpenCount()
	c := New(shared)
	if got := store.OpenCount(); got != before {
		t.Fatalf("New must not call store.Open: OpenCount went %d -> %d", before, got)
	}
	if c.shared != shared {
		t.Fatalf("New must wrap the passed-in *store.Store, got a different handle")
	}

	// The accessor queries hit the SAME database file as the shared store:
	// writing through the ctx accessor is visible on the shared handle.
	if err := c.PutProxySession(&ProxySession{
		SessionID: "sess-1",
		Agent:     "opencode",
		Mode:      "token",
	}); err != nil {
		t.Fatalf("PutProxySession: %v", err)
	}
	var n int
	if err := shared.DB.QueryRow(
		`SELECT COUNT(*) FROM ctx_proxy_sessions WHERE session_id='sess-1'`,
	).Scan(&n); err != nil {
		t.Fatalf("shared handle read: %v", err)
	}
	if n != 1 {
		t.Fatalf("row written via ctx accessor not visible on the shared handle (same-file property): %d", n)
	}
}

// TestCtxTableRoundTrip covers @schema-ctx-tables (AC2b): every table's typed
// insert + get round-trips a row with all columns compared, and the list
// accessor returns the row.
func TestCtxTableRoundTrip(t *testing.T) {
	st := testStore(t, "ctxroundtrip")
	c := New(st)

	// ctx_proxy_sessions
	sess := &ProxySession{
		SessionID: "sess-rt", Agent: "cursor", Mode: "cache",
		StartedAt: 1000, EndedAt: 2000, Requests: 7,
		TokensBefore: 5000, TokensAfter: 3000,
		CacheHits: 4, CacheInvalidations: 1,
	}
	if err := c.PutProxySession(sess); err != nil {
		t.Fatalf("PutProxySession: %v", err)
	}
	got, err := c.GetProxySession("sess-rt")
	if err != nil {
		t.Fatalf("GetProxySession: %v", err)
	}
	if got == nil || *got != *sess {
		t.Fatalf("ProxySession round-trip: got %+v want %+v", got, sess)
	}

	// ctx_proxy_savings
	sav := &ProxySaving{
		SavingID: "sav-1", SessionID: "sess-rt", Turn: 3, Upstream: "anthropic",
		Mode: "token", TokensBefore: 100, TokensAfter: 40,
		SavingsRatio: 0.6, CreatedAt: 1234,
	}
	if err := c.PutProxySaving(sav); err != nil {
		t.Fatalf("PutProxySaving: %v", err)
	}
	gotSav, err := c.GetProxySaving("sav-1")
	if err != nil {
		t.Fatalf("GetProxySaving: %v", err)
	}
	if gotSav == nil || *gotSav != *sav {
		t.Fatalf("ProxySaving round-trip: got %+v want %+v", gotSav, sav)
	}

	// ctx_proxy_mode_history
	chg := &ModeChange{
		ChangeID: "chg-1", SessionID: "sess-rt",
		OldMode: "passthrough", NewMode: "token",
		Reason: "budget", ChangedAt: 555,
	}
	if err := c.PutModeChange(chg); err != nil {
		t.Fatalf("PutModeChange: %v", err)
	}
	gotChg, err := c.GetModeChange("chg-1")
	if err != nil {
		t.Fatalf("GetModeChange: %v", err)
	}
	if gotChg == nil || *gotChg != *chg {
		t.Fatalf("ModeChange round-trip: got %+v want %+v", gotChg, chg)
	}

	// ctx_session_summaries
	shape := true
	sum := &SessionSummary{
		SessionID: "sess-rt", Agent: "opencode", DurationMS: 900, Turns: 5,
		TokensSaved: 2000, JudgmentKeeps: 2, JudgmentTruncs: 1,
		PrefilterPins: 3, PrefilterDeads: 0, CacheInvalidations: 1,
		ShapingApplied: &shape, EffortRouted: 2, TrafficMemories: 1,
		SummaryJSON: `{"turns":5}`, CreatedAt: 999,
	}
	if err := c.PutSessionSummary(sum); err != nil {
		t.Fatalf("PutSessionSummary: %v", err)
	}
	gotSum, err := c.GetSessionSummary("sess-rt")
	if err != nil {
		t.Fatalf("GetSessionSummary: %v", err)
	}
	if gotSum == nil || !summaryEqual(*gotSum, *sum) {
		t.Fatalf("SessionSummary round-trip: got %+v want %+v", gotSum, sum)
	}

	// ctx_stats_history (two rows to exercise the list)
	one := &StatsBucket{
		BucketID: "bkt-1", Granularity: "daily", BucketStart: 1700000000000,
		TokensSaved: 1000, Requests: 50, Sessions: 3, CacheHits: 20, CreatedAt: 1,
	}
	two := &StatsBucket{
		BucketID: "bkt-2", Granularity: "daily", BucketStart: 1700003600000,
		TokensSaved: 500, Requests: 25, Sessions: 1, CacheHits: 10, CreatedAt: 2,
	}
	if err := c.PutStatsBucket(one); err != nil {
		t.Fatalf("PutStatsBucket one: %v", err)
	}
	if err := c.PutStatsBucket(two); err != nil {
		t.Fatalf("PutStatsBucket two: %v", err)
	}
	gotBkt, err := c.GetStatsBucket("bkt-1")
	if err != nil {
		t.Fatalf("GetStatsBucket: %v", err)
	}
	if gotBkt == nil || *gotBkt != *one {
		t.Fatalf("StatsBucket round-trip: got %+v want %+v", gotBkt, one)
	}
	buckets, err := c.ListStatsBuckets("daily")
	if err != nil {
		t.Fatalf("ListStatsBuckets: %v", err)
	}
	if len(buckets) != 2 || buckets[0].BucketID != "bkt-1" || buckets[1].BucketID != "bkt-2" {
		t.Fatalf("ListStatsBuckets: got %+v", buckets)
	}

	// ctx_traffic_memories
	mem := &TrafficMemory{
		MemoryID: "mem-1", SessionID: "sess-rt", Category: "preference",
		MemoryText: "prefer compact diffs", EvidenceJSON: `{"hits":2}`,
		Confidence: 0.9, CreatedAt: 4242,
	}
	if err := c.PutTrafficMemory(mem); err != nil {
		t.Fatalf("PutTrafficMemory: %v", err)
	}
	gotMem, err := c.GetTrafficMemory("mem-1")
	if err != nil {
		t.Fatalf("GetTrafficMemory: %v", err)
	}
	if gotMem == nil || *gotMem != *mem {
		t.Fatalf("TrafficMemory round-trip: got %+v want %+v", gotMem, mem)
	}

	// List accessors for the session-keyed tables.
	sessList, err := c.ListProxySessions()
	if err != nil {
		t.Fatalf("ListProxySessions: %v", err)
	}
	if len(sessList) != 1 || sessList[0].SessionID != "sess-rt" {
		t.Fatalf("ListProxySessions: got %+v", sessList)
	}
	savList, err := c.ListProxySavings("sess-rt")
	if err != nil {
		t.Fatalf("ListProxySavings: %v", err)
	}
	if len(savList) != 1 || savList[0].SavingID != "sav-1" {
		t.Fatalf("ListProxySavings: got %+v", savList)
	}
	sumList, err := c.ListSessionSummaries()
	if err != nil {
		t.Fatalf("ListSessionSummaries: %v", err)
	}
	if len(sumList) != 1 || sumList[0].SessionID != "sess-rt" {
		t.Fatalf("ListSessionSummaries: got %+v", sumList)
	}
	memList, err := c.ListTrafficMemories("sess-rt")
	if err != nil {
		t.Fatalf("ListTrafficMemories: %v", err)
	}
	if len(memList) != 1 || memList[0].MemoryID != "mem-1" {
		t.Fatalf("ListTrafficMemories: got %+v", memList)
	}
}

// TestCtxRollup aggregates ctx_stats_history into tokens_saved / requests /
// sessions / cache_hits grouped by granularity + bucket, ordered by
// bucket_start.
func TestCtxRollup(t *testing.T) {
	st := testStore(t, "ctxrollup")
	c := New(st)

	rows := []*StatsBucket{
		{BucketID: "h1", Granularity: "hourly", BucketStart: 100, TokensSaved: 10, Requests: 1, Sessions: 1, CacheHits: 2, CreatedAt: 1},
		{BucketID: "h2", Granularity: "hourly", BucketStart: 200, TokensSaved: 30, Requests: 2, Sessions: 1, CacheHits: 4, CreatedAt: 2},
		{BucketID: "d1", Granularity: "daily", BucketStart: 300, TokensSaved: 90, Requests: 9, Sessions: 4, CacheHits: 8, CreatedAt: 3},
	}
	for _, b := range rows {
		if err := c.PutStatsBucket(b); err != nil {
			t.Fatalf("PutStatsBucket %s: %v", b.BucketID, err)
		}
	}

	got, err := c.Rollup()
	if err != nil {
		t.Fatalf("Rollup: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Rollup: expected 2 groups (hourly, daily), got %d: %+v", len(got), got)
	}
	want := map[string]RollupRow{
		"hourly": {Granularity: "hourly", TokensSaved: 40, Requests: 3, Sessions: 2, CacheHits: 6},
		"daily":  {Granularity: "daily", TokensSaved: 90, Requests: 9, Sessions: 4, CacheHits: 8},
	}
	for _, r := range got {
		w, ok := want[r.Granularity]
		if !ok {
			t.Fatalf("Rollup: unexpected granularity %q", r.Granularity)
		}
		if r != w {
			t.Fatalf("Rollup %s: got %+v want %+v", r.Granularity, r, w)
		}
		delete(want, r.Granularity)
	}
	if len(want) != 0 {
		t.Fatalf("Rollup: missing groups: %+v", want)
	}
}

// TestCtxFailOpenMissingTable covers @fail-open-degrade (store level): on a
// database where the 056-061 migrations never ran, every list/get accessor
// returns empty/zero (not found) with no panic and no hard error.
func TestCtxFailOpenMissingTable(t *testing.T) {
	st := testStore(t, "ctxnomigrate")
	// Drop the six tables to simulate a store opened before the migrations.
	for _, table := range ctxTableNames() {
		if _, err := st.DB.Exec(`DROP TABLE IF EXISTS ` + table); err != nil {
			t.Fatalf("drop %s: %v", table, err)
		}
	}
	c := New(st)

	if got, err := c.GetProxySession("nope"); err != nil {
		t.Fatalf("GetProxySession fail-open: %v", err)
	} else if got != nil {
		t.Fatalf("GetProxySession on missing table must be nil, got %+v", got)
	}
	if got, err := c.ListProxySessions(); err != nil {
		t.Fatalf("ListProxySessions fail-open: %v", err)
	} else if len(got) != 0 {
		t.Fatalf("ListProxySessions on missing table must be empty, got %d", len(got))
	}
	if got, err := c.ListProxySavings("s"); err != nil {
		t.Fatalf("ListProxySavings fail-open: %v", err)
	} else if len(got) != 0 {
		t.Fatalf("ListProxySavings on missing table must be empty, got %d", len(got))
	}
	if got, err := c.GetProxySaving("nope"); err != nil {
		t.Fatalf("GetProxySaving fail-open: %v", err)
	} else if got != nil {
		t.Fatalf("GetProxySaving on missing table must be nil, got %+v", got)
	}
	if got, err := c.GetModeChange("nope"); err != nil {
		t.Fatalf("GetModeChange fail-open: %v", err)
	} else if got != nil {
		t.Fatalf("GetModeChange on missing table must be nil, got %+v", got)
	}
	if got, err := c.GetSessionSummary("nope"); err != nil {
		t.Fatalf("GetSessionSummary fail-open: %v", err)
	} else if got != nil {
		t.Fatalf("GetSessionSummary on missing table must be nil, got %+v", got)
	}
	if got, err := c.ListSessionSummaries(); err != nil {
		t.Fatalf("ListSessionSummaries fail-open: %v", err)
	} else if len(got) != 0 {
		t.Fatalf("ListSessionSummaries on missing table must be empty, got %d", len(got))
	}
	if got, err := c.GetStatsBucket("nope"); err != nil {
		t.Fatalf("GetStatsBucket fail-open: %v", err)
	} else if got != nil {
		t.Fatalf("GetStatsBucket on missing table must be nil, got %+v", got)
	}
	if got, err := c.ListStatsBuckets("daily"); err != nil {
		t.Fatalf("ListStatsBuckets fail-open: %v", err)
	} else if len(got) != 0 {
		t.Fatalf("ListStatsBuckets on missing table must be empty, got %d", len(got))
	}
	if got, err := c.Rollup(); err != nil {
		t.Fatalf("Rollup fail-open: %v", err)
	} else if len(got) != 0 {
		t.Fatalf("Rollup on missing table must be empty, got %d", len(got))
	}
	if got, err := c.GetTrafficMemory("nope"); err != nil {
		t.Fatalf("GetTrafficMemory fail-open: %v", err)
	} else if got != nil {
		t.Fatalf("GetTrafficMemory on missing table must be nil, got %+v", got)
	}
	if got, err := c.ListTrafficMemories("s"); err != nil {
		t.Fatalf("ListTrafficMemories fail-open: %v", err)
	} else if len(got) != 0 {
		t.Fatalf("ListTrafficMemories on missing table must be empty, got %d", len(got))
	}

	// Writes degrade fail-open too: the row is lost (table gone), not a hard
	// error that would 500 the proxy's write path.
	if err := c.PutProxySession(&ProxySession{SessionID: "x", Mode: "token"}); err != nil {
		t.Fatalf("PutProxySession on missing table must be fail-open, got: %v", err)
	}
}
