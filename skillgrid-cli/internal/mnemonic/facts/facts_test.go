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

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
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
	id, err := s.Add(context.Background(), sid, "GDE pattern: cache the registry once per session")
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
	if _, err := s.Add(context.Background(), "sess-empty", ""); err == nil {
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
	if _, err := s.Add(context.Background(), "no-such-session", "fact body"); err == nil {
		t.Fatal("Add with unknown session should fail (FK)")
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

	a, err := s.Add(context.Background(), sid, "cache the registry once per session")
	if err != nil {
		t.Fatalf("Add a: %v", err)
	}
	b, err := s.Add(context.Background(), sid, "unrelated cooking recipe for ramen")
	if err != nil {
		t.Fatalf("Add b: %v", err)
	}
	if a == b {
		t.Fatalf("Add returned duplicate id %d", a)
	}

	got, err := s.Search(context.Background(), sid, "registry", 10)
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

	kept, err := s.Add(context.Background(), sid, "remembered zebra fact about stripes")
	if err != nil {
		t.Fatalf("Add kept: %v", err)
	}
	gone, err := s.Add(context.Background(), sid, "forgotten zebra fact about hooves")
	if err != nil {
		t.Fatalf("Add gone: %v", err)
	}

	if err := s.Forget(context.Background(), sid, gone); err != nil {
		t.Fatalf("Forget: %v", err)
	}

	got, err := s.Search(context.Background(), sid, "zebra", 10)
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
	all, err := s.SearchWith(context.Background(), sid, "zebra", 10, true)
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
	if _, err := s.Add(context.Background(), sid, "some searchable content here"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, err := s.Search(context.Background(), sid, "   ", 10)
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
	id, err := s.Add(context.Background(), sid, "fact to be forgotten")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := s.Forget(context.Background(), sid, id); err != nil {
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
	if err := s.Forget(context.Background(), sid, id); err != nil {
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
	if err := s.Forget(context.Background(), sid, 99999); err == nil {
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

	id, err := s.Add(context.Background(), sid, "fact that will decay")
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

	newScore, err := s.Decay(context.Background(), sid, id)
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
	if _, err := s.Decay(context.Background(), sid, 99999); err == nil {
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
	kept, err := s.Add(ctx, sid, "fresh fact that survives the decay purge")
	if err != nil {
		t.Fatalf("Add kept: %v", err)
	}
	purged, err := s.Add(ctx, sid, "old fact that decays below the threshold")
	if err != nil {
		t.Fatalf("Add purged: %v", err)
	}
	purgedOld, err := s.Add(ctx, sid, "old fact already below the threshold")
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

	decayed, purgedN, err := s.DecayAll(ctx, sid, 0.5)
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
	got, err := s.Search(ctx, sid, "fact", 10)
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
	decayed, purged, err := s.DecayAll(context.Background(), sid, 0.5)
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

// TestSessionEventSequenceIsMonotonic guards the events trail: three
// consecutive fact tool calls append strictly increasing sequences in the
// session's event stream, and the stream stays readable in sequence order
// (session_changes read path).
func TestSessionEventSequenceIsMonotonic(t *testing.T) {
	st := openTestStore(t)
	sid := "sess-fact-seq"
	seedTestSession(t, st, sid)
	s := New(st.DB, "factstest")

	id, err := s.Add(context.Background(), sid, "sequence check content")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := s.Search(context.Background(), sid, "sequence", 10); err != nil {
		t.Fatalf("Search: %v", err)
	}
	if _, err := s.Decay(context.Background(), sid, id); err != nil {
		t.Fatalf("Decay: %v", err)
	}
	if err := s.Forget(context.Background(), sid, id); err != nil {
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
