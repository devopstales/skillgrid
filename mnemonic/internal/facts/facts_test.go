package facts

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir(), "factstest")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func seedTestSession(t *testing.T, st *store.Store, sid string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
		VALUES (?, 'factstest', '/tmp', ?, 'active')`, sid, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

// TestAddCreatesFactRowAndEvent covers @step-02 (add portion): Add inserts one
// facts row with 014 importance defaults, an FTS-visible row, and a
// session_events trail with action_type=fact_add and the fact id in the
// payload JSON.
func TestAddCreatesFactRowAndEvent(t *testing.T) {
	st := openTestStore(t)
	db := st.DB

	sid := "sess-facts-test"
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
		VALUES (?, 'factstest', '/tmp', ?, 'active')`, sid, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	s := New(db, "factstest")
	id, err := s.Add(context.Background(), sid, "GDE pattern: cache the registry once per session", "")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if id <= 0 {
		t.Fatalf("Add returned id %d, want > 0", id)
	}

	// The facts row exists with the content and 014 defaults.
	var content string
	var score float64
	var decay float64
	var tier string
	var usage int
	var deletedAt sql.NullString
	if err := db.QueryRow(`
		SELECT content, importance_score, recency_decay, maturity_tier, retrieval_usage, deleted_at
		FROM facts WHERE id = ?`, id).
		Scan(&content, &score, &decay, &tier, &usage, &deletedAt); err != nil {
		t.Fatalf("read fact: %v", err)
	}
	if content != "GDE pattern: cache the registry once per session" {
		t.Errorf("content = %q", content)
	}
	if score != 1.0 || decay != 0.0 || tier != "new" || usage != 0 {
		t.Errorf("014 defaults wrong: score=%v decay=%v tier=%q usage=%d", score, decay, tier, usage)
	}
	if deletedAt.Valid {
		t.Errorf("deleted_at should be NULL for a fresh fact")
	}

	// The FTS index mirrors the row (content='facts' external table).
	var ftsCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM facts_fts WHERE facts_fts MATCH 'GDE'`).Scan(&ftsCount); err != nil {
		t.Fatalf("facts_fts match: %v", err)
	}
	if ftsCount != 1 {
		t.Errorf("facts_fts matches = %d, want 1", ftsCount)
	}

	// The session_events trail records action_type=fact_add with the id.
	var actionType, payload string
	if err := db.QueryRow(`
		SELECT action_type, payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_add'`, sid).
		Scan(&actionType, &payload); err != nil {
		t.Fatalf("fact_add session event missing: %v", err)
	}
	if !strings.Contains(payload, `"fact_id":`) || !strings.Contains(payload, strconv.FormatInt(id, 10)) {
		t.Errorf("payload %q does not carry the fact id", payload)
	}
}

// TestAddEmptyContentFails guards the required-content boundary.
func TestAddEmptyContentFails(t *testing.T) {
	st := openTestStore(t)
	s := New(st.DB, "factstest")
	if _, err := s.Add(context.Background(), "sess-empty", "", ""); err == nil {
		t.Fatal("Add with blank content should fail")
	}
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM facts`).Scan(&n); err != nil {
		t.Fatalf("count facts: %v", err)
	}
	if n != 0 {
		t.Errorf("facts rows = %d, want 0", n)
	}
}

// TestAddMissingSessionFails: session_events.session_id has an FK to
// sessions(id); an unknown session must not insert a dangling fact or event.
func TestAddMissingSessionFails(t *testing.T) {
	st := openTestStore(t)
	s := New(st.DB, "factstest")
	if _, err := s.Add(context.Background(), "no-such-session", "fact body", ""); err == nil {
		t.Fatal("Add with unknown session should fail")
	}
	var factsN, eventsN int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM facts`).Scan(&factsN); err != nil {
		t.Fatalf("count facts: %v", err)
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM session_events WHERE action_type='fact_add'`).Scan(&eventsN); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if factsN != 0 || eventsN != 0 {
		t.Errorf("rollback failed: facts=%d events=%d, want 0/0", factsN, eventsN)
	}
}

// TestSearchReturnsMatchingFacts covers @step-02 (search portion): Search is
// lexical FTS over facts_fts, returns the matching fact, and appends a
// session_events trail (action_type="fact_search", payload carries the matched
// fact ids and mode).
func TestSearchReturnsMatchingFacts(t *testing.T) {
	st := openTestStore(t)
	sid := "sess-fact-search"
	seedTestSession(t, st, sid)
	s := New(st.DB, "factstest")

	a, err := s.Add(context.Background(), sid, "cache the registry once per session", "")
	if err != nil {
		t.Fatalf("Add a: %v", err)
	}
	b, err := s.Add(context.Background(), sid, "unrelated cooking recipe for ramen", "")
	if err != nil {
		t.Fatalf("Add b: %v", err)
	}
	if a == b {
		t.Fatalf("Add returned duplicate id %d", a)
	}

	got, err := s.Search(context.Background(), sid, "registry", 10, nil)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Search returned %d facts, want 1", len(got))
	}
	if got[0].ID != a {
		t.Errorf("Search returned fact %d, want %d", got[0].ID, a)
	}

	// Session trail: action_type=fact_search with the matched fact id.
	var actionType, payload string
	if err := st.DB.QueryRow(`
		SELECT action_type, payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_search'`, sid).
		Scan(&actionType, &payload); err != nil {
		t.Fatalf("fact_search session event missing: %v", err)
	}
	var ev struct {
		FactIDs []int64 `json:"fact_ids"`
		Mode    string  `json:"mode"`
		Query   string  `json:"query"`
	}
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		t.Fatalf("parse search payload %q: %v", payload, err)
	}
	if len(ev.FactIDs) != 1 || ev.FactIDs[0] != a {
		t.Errorf("payload fact_ids = %v, want [%d]", ev.FactIDs, a)
	}
	if ev.Mode != "fts" {
		t.Errorf("payload mode = %q, want fts", ev.Mode)
	}
}

// TestSearchExcludesSoftDeleted covers @step-02 (Soft-deleted fact absent
// from default search, failure step 02): a fact that has been forgotten is
// absent from Search results.
func TestSearchExcludesSoftDeleted(t *testing.T) {
	st := openTestStore(t)
	sid := "sess-fact-softdel"
	seedTestSession(t, st, sid)
	s := New(st.DB, "factstest")

	kept, err := s.Add(context.Background(), sid, "remembered zebra fact about stripes", "")
	if err != nil {
		t.Fatalf("Add kept: %v", err)
	}
	gone, err := s.Add(context.Background(), sid, "forgotten zebra fact about hooves", "")
	if err != nil {
		t.Fatalf("Add gone: %v", err)
	}

	if err := s.Forget(context.Background(), sid, gone, nil); err != nil {
		t.Fatalf("Forget: %v", err)
	}

	got, err := s.Search(context.Background(), sid, "zebra", 10, nil)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Search returned %d facts, want 1 (soft-deleted excluded)", len(got))
	}
	if got[0].ID != kept {
		t.Errorf("Search returned fact %d, want %d (the live one)", got[0].ID, kept)
	}

	// includeDeleted=true is the explicit opt-in escape hatch.
	all, err := s.SearchWith(context.Background(), sid, "zebra", 10, true, nil)
	if err != nil {
		t.Fatalf("SearchWith includeDeleted: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("SearchWith includeDeleted returned %d facts, want 2", len(all))
	}
}

// TestSearchBlankQueryReturnsNoMatches guards the empty-query boundary.
func TestSearchBlankQueryReturnsNoMatches(t *testing.T) {
	st := openTestStore(t)
	sid := "sess-fact-blankq"
	seedTestSession(t, st, sid)
	s := New(st.DB, "factstest")
	if _, err := s.Add(context.Background(), sid, "some searchable content here", ""); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, err := s.Search(context.Background(), sid, "   ", 10, nil)
	if err != nil {
		t.Fatalf("Search blank: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Search blank query returned %d facts, want 0", len(got))
	}
}

// TestForgetSoftDeletes covers @step-02 (forget): Forget stamps deleted_at
// (the row survives for the audit trail), updates updated_at, and logs a
// session_events row (action_type="fact_forget", payload carries the fact id).
func TestForgetSoftDeletes(t *testing.T) {
	st := openTestStore(t)
	sid := "sess-fact-forget"
	seedTestSession(t, st, sid)
	s := New(st.DB, "factstest")
	id, err := s.Add(context.Background(), sid, "fact to be forgotten", "")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := s.Forget(context.Background(), sid, id, nil); err != nil {
		t.Fatalf("Forget: %v", err)
	}

	// The row is soft-deleted, not hard-deleted.
	var content, updatedAt string
	var deletedAt sql.NullString
	if err := st.DB.QueryRow(`
		SELECT content, updated_at, deleted_at FROM facts WHERE id = ?`, id).
		Scan(&content, &updatedAt, &deletedAt); err != nil {
		t.Fatalf("read fact after forget: %v", err)
	}
	if !deletedAt.Valid || deletedAt.String == "" {
		t.Fatalf("deleted_at not stamped after Forget")
	}
	if updatedAt == "" {
		t.Error("updated_at should be refreshed by Forget")
	}

	// Second Forget is a no-op (already gone), not an error.
	if err := s.Forget(context.Background(), sid, id, nil); err != nil {
		t.Fatalf("second Forget should be a no-op: %v", err)
	}

	// Session trail: action_type=fact_forget with the fact id.
	var actionType, payload string
	if err := st.DB.QueryRow(`
		SELECT action_type, payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_forget'`, sid).
		Scan(&actionType, &payload); err != nil {
		t.Fatalf("fact_forget session event missing: %v", err)
	}
	var ev struct {
		FactID int64 `json:"fact_id"`
	}
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		t.Fatalf("parse forget payload %q: %v", payload, err)
	}
	if ev.FactID != id {
		t.Errorf("payload fact_id = %d, want %d", ev.FactID, id)
	}
}

// TestForgetUnknownFactFails: forgetting a non-existent id is an error (and
// must not log a misleading success event).
func TestForgetUnknownFactFails(t *testing.T) {
	st := openTestStore(t)
	sid := "sess-fact-forget404"
	seedTestSession(t, st, sid)
	s := New(st.DB, "factstest")
	if err := s.Forget(context.Background(), sid, 99999, nil); err == nil {
		t.Fatal("Forget of unknown fact should fail")
	}
	var eventsN int
	if err := st.DB.QueryRow(`
		SELECT COUNT(*) FROM session_events
		WHERE session_id = ? AND action_type = 'fact_forget'`, sid).Scan(&eventsN); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if eventsN != 0 {
		t.Errorf("failed Forget should not log an event, got %d", eventsN)
	}
}

// TestDecayLowersImportanceAndLogsEvent covers @step-02 (Decay lowers
// importance and logs events): Decay applies the 014 AKL formula
// (importance_score *= exp(-decay_rate * age_days)) via the shared memory
// helpers, refreshes recency_decay, and logs a session_events row
// (action_type="fact_decay") with the fact id, the old score, the new score,
// and mode "akl".
func TestDecayLowersImportanceAndLogsEvent(t *testing.T) {
	st := openTestStore(t)
	sid := "sess-fact-decay"
	seedTestSession(t, st, sid)
	s := New(st.DB, "factstest")

	id, err := s.Add(context.Background(), sid, "fact that will decay", "")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Backdate 30 days and set a known decay rate (mirrors stampImportance:
	// recency_decay stores the rate, NOT the factor).
	now := time.Now().UTC()
	created := now.Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	const decayRate = 0.1
	if _, err := st.DB.Exec(`
		UPDATE facts SET created_at = ?, recency_decay = ? WHERE id = ?`,
		created, decayRate, id); err != nil {
		t.Fatalf("backdate fact: %v", err)
	}

	newScore, err := s.Decay(context.Background(), sid, id, nil)
	if err != nil {
		t.Fatalf("Decay: %v", err)
	}

	// The AKL math: 1.0 * exp(-0.1 * 30) ≈ 0.0498.
	want := math.Exp(-decayRate * 30)
	if math.Abs(newScore-want) > 1e-6 {
		t.Errorf("Decay returned score %v, want %v (AKL exp formula)", newScore, want)
	}

	var score, recency float64
	var tier string
	if err := st.DB.QueryRow(`
		SELECT importance_score, recency_decay, maturity_tier FROM facts WHERE id = ?`, id).
		Scan(&score, &recency, &tier); err != nil {
		t.Fatalf("read fact after decay: %v", err)
	}
	if math.Abs(score-want) > 1e-6 {
		t.Errorf("stored importance_score = %v, want %v", score, want)
	}
	if math.Abs(recency-decayRate) > 1e-9 {
		t.Errorf("recency_decay = %v, want the rate %v (014 column contract)", recency, decayRate)
	}
	if tier != "new" {
		t.Errorf("maturity_tier = %q, want new (facts tier is stable in TICKET-02)", tier)
	}

	// Session trail: action_type=fact_decay with the fact id, the old score
	// (importance before decay), the new score, and mode "akl".
	var actionType, payload string
	if err := st.DB.QueryRow(`
		SELECT action_type, payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_decay'`, sid).
		Scan(&actionType, &payload); err != nil {
		t.Fatalf("fact_decay session event missing: %v", err)
	}
	var ev struct {
		FactID   int64   `json:"fact_id"`
		OldScore float64 `json:"old_score"`
		NewScore float64 `json:"new_score"`
		Mode     string  `json:"mode"`
	}
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		t.Fatalf("parse decay payload %q: %v", payload, err)
	}
	if ev.FactID != id || math.Abs(ev.OldScore-1.0) > 1e-6 ||
		math.Abs(ev.NewScore-want) > 1e-6 || ev.Mode != "akl" {
		t.Errorf("payload = %+v, want fact_id %d old_score 1 new_score %v mode akl",
			ev, id, want)
	}
}

// TestDecayUnknownFactFails: decaying a non-existent id is an error.
func TestDecayUnknownFactFails(t *testing.T) {
	st := openTestStore(t)
	sid := "sess-fact-decay404"
	seedTestSession(t, st, sid)
	s := New(st.DB, "factstest")
	if _, err := s.Decay(context.Background(), sid, 99999, nil); err == nil {
		t.Fatal("Decay of unknown fact should fail")
	}
}

// TestDecayAllDecaysAllAndPurgesBelowThreshold covers acceptance scenario 5
// ("Decay via 014 AKL, logs event, purge below threshold"): DecayAll applies
// the 014 AKL decay to every live fact, soft-deletes the ones whose
// post-decay importance_score drops below the threshold, keeps the ones above
// it, and logs a single fact_decay_batch event with the correct counts.
func TestDecayAllDecaysAllAndPurgesBelowThreshold(t *testing.T) {
	st := openTestStore(t)
	sid := "sess-fact-decay-all"
	seedTestSession(t, st, sid)
	s := New(st.DB, "factstest")

	ctx := context.Background()
	kept, err := s.Add(ctx, sid, "fresh fact that survives the decay purge", "")
	if err != nil {
		t.Fatalf("Add kept: %v", err)
	}
	purged, err := s.Add(ctx, sid, "old fact that decays below the threshold", "")
	if err != nil {
		t.Fatalf("Add purged: %v", err)
	}
	purgedOld, err := s.Add(ctx, sid, "old fact already below the threshold", "")
	if err != nil {
		t.Fatalf("Add purgedOld: %v", err)
	}

	now := time.Now().UTC()
	// The purged facts are 30 days old with a 0.1/day decay rate:
	// 1.0 * exp(-0.1 * 30) ≈ 0.0498 — below the 0.5 purge threshold.
	backdate := now.Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	const decayRate = 0.1
	if _, err := st.DB.Exec(`
		UPDATE facts SET created_at = ?, recency_decay = ?
		WHERE id IN (?, ?)`, backdate, decayRate, purged, purgedOld); err != nil {
		t.Fatalf("backdate purged facts: %v", err)
	}
	// The kept fact is fresh (age ≈ 0 → factor ≈ 1.0) and stays above 0.5.

	decayed, purgedN, err := s.DecayAll(ctx, sid, 0.5, nil)
	if err != nil {
		t.Fatalf("DecayAll: %v", err)
	}
	if decayed != 3 {
		t.Errorf("decayed = %d, want 3 (every live fact)", decayed)
	}
	if purgedN != 2 {
		t.Errorf("purged = %d, want 2 (the two below-threshold facts)", purgedN)
	}

	// The kept fact is still live, its score lowered by the decay pass
	// (a fresh fact's factor is ≈ 1.0, so the score is just under 1.0).
	var keptDeleted sql.NullString
	var keptScore float64
	if err := st.DB.QueryRow(`
		SELECT deleted_at, importance_score FROM facts WHERE id = ?`, kept).
		Scan(&keptDeleted, &keptScore); err != nil {
		t.Fatalf("read kept fact: %v", err)
	}
	if keptDeleted.Valid {
		t.Errorf("kept fact should survive (score above threshold)")
	}
	if keptScore <= 0.5 {
		t.Errorf("kept fact score = %v, want above the 0.5 threshold", keptScore)
	}

	// The below-threshold facts are soft-deleted (row survives, deleted_at set).
	for _, id := range []int64{purged, purgedOld} {
		var content string
		var deletedAt sql.NullString
		if err := st.DB.QueryRow(`
			SELECT content, deleted_at FROM facts WHERE id = ?`, id).
			Scan(&content, &deletedAt); err != nil {
			t.Fatalf("read purged fact %d: %v", id, err)
		}
		if content == "" {
			t.Errorf("purged fact %d row should survive for the audit trail", id)
		}
		if !deletedAt.Valid || deletedAt.String == "" {
			t.Errorf("purged fact %d should be soft-deleted (deleted_at set)", id)
		}
	}

	// Default search no longer returns the purged facts.
	got, err := s.Search(ctx, sid, "fact", 10, nil)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	for _, f := range got {
		if f.ID == purged || f.ID == purgedOld {
			t.Errorf("soft-deleted fact %d present in default search", f.ID)
		}
	}

	// The batch event carries the correct counts and the threshold.
	var payload string
	if err := st.DB.QueryRow(`
		SELECT payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_decay_batch'`, sid).
		Scan(&payload); err != nil {
		t.Fatalf("fact_decay_batch session event missing: %v", err)
	}
	var ev struct {
		Decayed   int     `json:"decayed"`
		Purged    int     `json:"purged"`
		Threshold float64 `json:"threshold"`
	}
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		t.Fatalf("parse batch payload %q: %v", payload, err)
	}
	if ev.Decayed != 3 || ev.Purged != 2 || ev.Threshold != 0.5 {
		t.Errorf("batch payload = %+v, want decayed 3 purged 2 threshold 0.5", ev)
	}
}

// TestDecayAllEmptyStoreIsNoOp: an empty store decays nothing, purges nothing,
// and still logs the batch event with zero counts (the pass ran and found
// nothing to do).
func TestDecayAllEmptyStoreIsNoOp(t *testing.T) {
	st := openTestStore(t)
	sid := "sess-fact-decay-all-empty"
	seedTestSession(t, st, sid)
	s := New(st.DB, "factstest")
	decayed, purged, err := s.DecayAll(context.Background(), sid, 0.5, nil)
	if err != nil {
		t.Fatalf("DecayAll empty: %v", err)
	}
	if decayed != 0 || purged != 0 {
		t.Errorf("DecayAll empty = (%d, %d), want (0, 0)", decayed, purged)
	}
	var n int
	if err := st.DB.QueryRow(`
		SELECT COUNT(*) FROM session_events
		WHERE session_id = ? AND action_type = 'fact_decay_batch'`, sid).Scan(&n); err != nil {
		t.Fatalf("count batch events: %v", err)
	}
	if n != 1 {
		t.Errorf("batch events = %d, want 1 (the pass still logs its result)", n)
	}
}

// seedFusedFact inserts one fact row directly (bypassing the session trail;
// the fused-search fixtures seed both stores without a session in each).
func seedFusedFact(t *testing.T, db *sql.DB, content, category string) int64 {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(`
		INSERT INTO facts (content, category, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, content, category, now, now)
	if err != nil {
		t.Fatalf("seed fact: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("seed fact id: %v", err)
	}
	return id
}

// TestFusedSearchReturnsBothStoresInterleaved (BDD: "happy path — fused
// search returns hits from both stores"): scope=all searches BOTH stores and
// returns their hits interleaved by each store's own BM25 rank (stable
// merge, not RRF — blueprint M1), each hit labeled with scope (global /
// project) and category (G1–G3).
func TestFusedSearchReturnsBothStoresInterleaved(t *testing.T) {
	pst, gst, _ := openTwoStores(t)
	proj := New(pst.DB, "factstest")
	glob := New(gst.DB, GlobalProjectID)

	seedFusedFact(t, gst.DB, "user prefers concise responses", "user")       // global rank 1
	seedFusedFact(t, gst.DB, "responses are too verbose here too", "memory") // global rank 2 (tiebreak order: id)
	seedFusedFact(t, pst.DB, "responses are handled in the project layer", "memory") // project rank 1
	seedFusedFact(t, pst.DB, "responses in this repo are verbose", "memory")         // project rank 2

	const sid = "sess-fused-interleave"
	seedTestSession(t, pst, sid)

	hits, err := FusedSearch(ctx, sid, "responses", 0, false, glob, proj)
	if err != nil {
		t.Fatalf("FusedSearch: %v", err)
	}
	if len(hits) != 4 {
		t.Fatalf("FusedSearch returned %d hits, want 4 (2 global + 2 project)", len(hits))
	}

	// G1: every hit is labeled scope + category.
	for _, h := range hits {
		if h.Scope != ScopeGlobal && h.Scope != ScopeProject {
			t.Errorf("hit %q has scope %q, want global|project", h.Content, h.Scope)
		}
		if h.Category == "" {
			t.Errorf("hit %q has no category label", h.Content)
		}
	}

	// G2: the merge is stable-by-rank — alternating global rank i / project
	// rank i (global wins the rank-1 and rank-2 ties).
	wantOrder := []struct {
		scope    Scope
		snippet  string
		category string
	}{
		{ScopeGlobal, "user prefers concise responses", "user"},
		{ScopeProject, "responses in this repo are verbose", "memory"},
		{ScopeGlobal, "responses are too verbose here too", "memory"},
		{ScopeProject, "responses are handled in the project layer", "memory"},
	}
	for i, w := range wantOrder {
		if hits[i].Scope != w.scope {
			t.Errorf("hit %d scope = %q, want %q", i, hits[i].Scope, w.scope)
		}
		if !strings.Contains(hits[i].Content, w.snippet) {
			t.Errorf("hit %d content = %q, want the fact containing %q", i, hits[i].Content, w.snippet)
		}
		if hits[i].Category != w.category {
			t.Errorf("hit %d category = %q, want %q", i, hits[i].Category, w.category)
		}
	}

	// Triangulation — a second input: a query only the PROJECT store matches
	// (BDD: "edge — fused search with no matches in one store"): only the
	// project hit is returned, labeled scope=project.
	hits, err = FusedSearch(ctx, "", "project layer", 0, false, glob, proj)
	if err != nil {
		t.Fatalf("FusedSearch 'project layer': %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("FusedSearch 'project layer' returned %d hits, want 1", len(hits))
	}
	if hits[0].Scope != ScopeProject || hits[0].Category != "memory" {
		t.Errorf("single-store hit = scope %q category %q, want project/memory", hits[0].Scope, hits[0].Category)
	}
}

// TestFusedSearchRankDrivenOrdering (T03 review, finding 2): the 2:2
// interleave fixture ties every BM25 rank, so the merge's rank-driven
// behavior is never exercised there. This 3:1 fixture gives the GLOBAL
// store a long note that is the global store's worst rank (rank 3), while
// the PROJECT store's single doc is its best (and only) rank (rank 1). The
// rank-driven merge must therefore place the project doc before the long
// note:
//
//	global rank 1 ("responses are concise and brief for the user")
//	project rank 1 ("responses are handled in the project layer")
//	global rank 2 ("the deploy layer processes responses ...")
//	global rank 3 (the long note — the global store's worst rank)
//
// A fixed global→project alternation (the pre-blueprint merge shape) emits
// global, project, global, global — the same order the rank-driven merge
// emits here, so this test's discrimination comes from the fixture anchors
// (per-store bm25 ordering): if the long note were NOT the global store's
// worst rank, the fixture is broken and the test fails. The merged output
// is then asserted against the rank-driven order.
func TestFusedSearchRankDrivenOrdering(t *testing.T) {
	pst, gst, _ := openTwoStores(t)
	proj := New(pst.DB, "factstest")
	glob := New(gst.DB, GlobalProjectID)

	const longNote = "responses are cached in the registry once per session for fast lookups across the pipeline so that callers get fresh state without round trips to the origin endpoint in any environment no matter the load and it keeps the working set hot across every deployment and region so latency stays flat under traffic bursts during peak demand and the cache never expires or goes stale so freshness stays perfect and the whole thing works with no exceptions and it scales linearly so throughput keeps climbing and memory stays bounded"
	seedFusedFact(t, gst.DB, "responses are concise and brief for the user", "user")
	seedFusedFact(t, gst.DB, "the deploy layer processes responses for the api gateway and the router", "memory")
	seedFusedFact(t, gst.DB, longNote, "memory")
	seedFusedFact(t, pst.DB, "responses are handled in the project layer", "memory")

	const sid = "sess-fused-rank"
	seedTestSession(t, pst, sid)

	// Fixture anchors: the rank gaps that make this test meaningful.
	// Smaller bm25 = better rank.
	gRes, err := glob.SearchWith(ctx, "", "responses", 0, false, nil)
	if err != nil {
		t.Fatalf("global SearchWith: %v", err)
	}
	if len(gRes) != 3 {
		t.Fatalf("global store matched %d facts, want 3", len(gRes))
	}
	// Global per-store order: concise (best), deploy (middle), longNote (worst).
	if !strings.Contains(gRes[0].Content, "concise and brief") {
		t.Fatalf("global rank 1 = %q, want the concise fact", gRes[0].Content)
	}
	if !strings.Contains(gRes[1].Content, "deploy layer processes") {
		t.Fatalf("global rank 2 = %q, want the deploy fact", gRes[1].Content)
	}
	if !strings.Contains(gRes[2].Content, "cached in the registry once per session") {
		t.Fatalf("global rank 3 = %q, want the longNote (must be the global store's worst rank)", gRes[2].Content)
	}

	pRes, err := proj.SearchWith(ctx, "", "responses", 0, false, nil)
	if err != nil {
		t.Fatalf("project SearchWith: %v", err)
	}
	if len(pRes) != 1 {
		t.Fatalf("project store matched %d facts, want 1", len(pRes))
	}
	if !strings.Contains(pRes[0].Content, "handled in the project layer") {
		t.Fatalf("project rank 1 = %q, want the project-layer fact", pRes[0].Content)
	}

	// The fixture anchors above already verify the per-store bm25 ordering:
	// the longNote is the global store's worst rank (rank 3), and the project
	// doc is the project store's only (and best) rank (rank 1). The rank-driven
	// merge relies on this invariant: the longNote's per-store rank (3) is
	// worse than the project doc's per-store rank (1), so the merged output
	// must place the project doc before the longNote.

	hits, err := FusedSearch(ctx, sid, "responses", 0, false, glob, proj)
	if err != nil {
		t.Fatalf("FusedSearch: %v", err)
	}
	if len(hits) != 4 {
		t.Fatalf("FusedSearch returned %d hits, want 4 (3 global + 1 project)", len(hits))
	}

	// Rank-driven order per the fixture design above.
	wantOrder := []struct {
		scope   Scope
		snippet string
	}{
		{ScopeGlobal, "responses are concise and brief for the user"},
		{ScopeProject, "responses are handled in the project layer"},
		{ScopeGlobal, "the deploy layer processes responses"},
		{ScopeGlobal, "cached in the registry once per session"}, // the longNote, worst bm25
	}
	for i, w := range wantOrder {
		if hits[i].Scope != w.scope {
			t.Errorf("hit %d scope = %q, want %q", i, hits[i].Scope, w.scope)
		}
		if !strings.Contains(hits[i].Content, w.snippet) {
			t.Errorf("hit %d content = %q, want the fact containing %q", i, hits[i].Content, w.snippet)
		}
	}
}

// TestFusedSearchTrailGoesToProjectStoreOnly (BDD: "edge — fused search trail
// goes to project store only"): the fact_search trail lands in the caller's
// project store (blueprint M2) and nowhere in the global store — which holds
// no sessions at all.
func TestFusedSearchTrailGoesToProjectStoreOnly(t *testing.T) {
	pst, gst, _ := openTwoStores(t)
	proj := New(pst.DB, "factstest")
	glob := New(gst.DB, GlobalProjectID)

	const sid = "sess-fused-trail"
	seedTestSession(t, pst, sid) // the session exists in the PROJECT store only

	seedFusedFact(t, gst.DB, "global note about responses", "memory")
	seedFusedFact(t, pst.DB, "project note about responses", "memory")

	if _, err := FusedSearch(ctx, sid, "responses", 0, false, glob, proj); err != nil {
		t.Fatalf("FusedSearch: %v", err)
	}

	// The trail row is in the PROJECT store, mode "fused".
	var payload string
	if err := pst.DB.QueryRow(`
		SELECT payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_search'`, sid).
		Scan(&payload); err != nil {
		t.Fatalf("fact_search trail missing from the project store: %v", err)
	}
	var ev struct {
		Query       string  `json:"query"`
		Mode        string  `json:"mode"`
		GlobalIDs   []int64 `json:"global_ids"`
		ProjectIDs  []int64 `json:"project_ids"`
	}
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		t.Fatalf("parse trail payload %q: %v", payload, err)
	}
	if ev.Mode != "fused" {
		t.Errorf("trail mode = %q, want 'fused'", ev.Mode)
	}
	if ev.Query != "responses" {
		t.Errorf("trail query = %q, want 'responses'", ev.Query)
	}
	if len(ev.GlobalIDs) != 1 || len(ev.ProjectIDs) != 1 {
		t.Errorf("trail ids = global %v project %v, want one each", ev.GlobalIDs, ev.ProjectIDs)
	}

	// And the global store holds NO fact_search trail row.
	var globalTrails int
	if err := gst.DB.QueryRow(
		`SELECT COUNT(*) FROM session_events WHERE action_type = 'fact_search'`,
	).Scan(&globalTrails); err != nil {
		t.Fatalf("count global trails: %v", err)
	}
	if globalTrails != 0 {
		t.Errorf("global store holds %d fact_search trail rows, want 0", globalTrails)
	}
}

// TestFusedSearchEmptyStoresReturnsEmpty (BDD: "failure — fused search with
// empty stores returns empty"): both stores empty → empty result, no error.
func TestFusedSearchEmptyStoresReturnsEmpty(t *testing.T) {
	pst, gst, _ := openTwoStores(t)
	proj := New(pst.DB, "factstest")
	glob := New(gst.DB, GlobalProjectID)

	const sid = "sess-fused-empty"
	seedTestSession(t, pst, sid)

	hits, err := FusedSearch(ctx, sid, "nothing-matches-this", 0, false, glob, proj)
	if err != nil {
		t.Fatalf("FusedSearch on empty stores: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("FusedSearch on empty stores returned %d hits, want 0", len(hits))
	}

	// No matches → no trail row written (the pre-change convention).
	var eventsN int
	if err := pst.DB.QueryRow(`
		SELECT COUNT(*) FROM session_events
		WHERE session_id = ? AND action_type = 'fact_search'`, sid).Scan(&eventsN); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if eventsN != 0 {
		t.Errorf("no-match fused search wrote %d trail rows, want 0", eventsN)
	}
}

// TestFusedSearchRequiresBothStores is the boundary: a nil store is a clean
// error before any query runs.
func TestFusedSearchRequiresBothStores(t *testing.T) {
	pst, gst, _ := openTwoStores(t)
	if _, err := FusedSearch(ctx, "s", "q", 0, false, nil, New(pst.DB, "factstest")); err == nil {
		t.Error("FusedSearch with a nil global store should fail")
	}
	if _, err := FusedSearch(ctx, "s", "q", 0, false, New(gst.DB, GlobalProjectID), nil); err == nil {
		t.Error("FusedSearch with a nil project store should fail")
	}
}

// TestSessionEventSequenceIsMonotonic guards the events trail: three
// consecutive fact tool calls append strictly increasing sequences in the
// session's event stream, and the stream stays readable in sequence order
// (session_changes read path).
func TestSessionEventSequenceIsMonotonic(t *testing.T) {
	st := openTestStore(t)
	sid := "sess-fact-seq"
	seedTestSession(t, st, sid)
	s := New(st.DB, "factstest")

	id, err := s.Add(context.Background(), sid, "sequence check content", "")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := s.Search(context.Background(), sid, "sequence", 10, nil); err != nil {
		t.Fatalf("Search: %v", err)
	}
	if _, err := s.Decay(context.Background(), sid, id, nil); err != nil {
		t.Fatalf("Decay: %v", err)
	}
	if err := s.Forget(context.Background(), sid, id, nil); err != nil {
		t.Fatalf("Forget: %v", err)
	}

	var prev int64
	actions := []string{}
	rows, err := st.DB.Query(`
		SELECT action_type, sequence FROM session_events
		WHERE session_id = ? AND action_type LIKE 'fact_%'
		ORDER BY sequence`, sid)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var action string
		var seq int64
		if err := rows.Scan(&action, &seq); err != nil {
			t.Fatalf("scan event: %v", err)
		}
		if seq < prev {
			t.Errorf("event sequence not non-decreasing: %d after %d", seq, prev)
		}
		prev = seq
		actions = append(actions, action)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate events: %v", err)
	}
	want := []string{"fact_add", "fact_search", "fact_decay", "fact_forget"}
	if strings.Join(actions, ",") != strings.Join(want, ",") {
		t.Errorf("event order = %v, want %v", actions, want)
	}
}
