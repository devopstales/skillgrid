package store

import (
	"context"
	"strings"
	"testing"
)

// [fts-stays-in-sync] the FTS index must track every observation mutation:
// the insert trigger indexes a new row, and a title update must remove the old
// FTS row before the new one is indexed (the BEFORE UPDATE delete trigger).
func TestFTSStaysInSync(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	res, err := st.AddObservation(ctx, AddObservationParams{
		SessionID: "s", Type: "decision", Title: "Use trigram search",
		Content: "trigram recall beats word splitting for C++",
		Project: "proj-a", Scope: "project",
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	r, err := st.Search(ctx, SearchQuery{Query: "trigram", Project: "proj-a", Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(r.Observations) != 1 || r.Observations[0].ID != res.ID {
		t.Fatalf("fts insert trigger broken: %+v", r)
	}
	// update the title — the BEFORE UPDATE trigger must DELETE the old FTS row
	// before the AFTER UPDATE trigger indexes the new one
	title := "Bigger trigrams"
	if err := st.UpdateObservationForProject(ctx, res.ID, "proj-a", UpdateParams{Title: &title}); err != nil {
		t.Fatalf("update: %v", err)
	}
	r2, err := st.Search(ctx, SearchQuery{Query: "Bigger", Project: "proj-a", Limit: 10})
	if err != nil {
		t.Fatalf("search bigger: %v", err)
	}
	if len(r2.Observations) != 1 || r2.Observations[0].ID != res.ID {
		t.Fatalf("fts update trigger broken: %+v", r2)
	}
	r3, err := st.Search(ctx, SearchQuery{Query: "Use", Project: "proj-a", Limit: 10})
	if err != nil {
		t.Fatalf("search stale: %v", err)
	}
	if len(r3.Observations) != 0 {
		t.Fatalf("stale FTS row after update: %+v", r3)
	}
}

// [short-term-like-fallback] a term below the trigram minimum (< 3 runes) must
// still match via the bounded LIKE fallback over the FTS columns, and the
// fallback must be project-scoped (no cross-project leakage).
func TestShortTermFallsBackToLIKE(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	if _, err := st.AddObservation(ctx, AddObservationParams{
		SessionID: "s", Type: "discovery", Title: "Go",
		Content: "Go 1.26 is the floor for the v2 module",
		Project: "proj-a", Scope: "project",
	}); err != nil {
		t.Fatalf("add: %v", err)
	}
	r, err := st.Search(ctx, SearchQuery{Query: "go", Project: "proj-a", Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(r.Observations) != 1 || !strings.Contains(r.Observations[0].Title, "Go") {
		t.Fatalf("short-term LIKE fallback broken (trigram needs >=3 runes): %+v", r)
	}
	// project-isolation: a LIKE fallback must not leak rows from other projects.
	if _, err := st.AddObservation(ctx, AddObservationParams{
		SessionID: "s", Type: "discovery", Title: "Go compiler notes",
		Content: "Go compiler performance notes",
		Project: "proj-b", Scope: "project",
	}); err != nil {
		t.Fatalf("add proj-b: %v", err)
	}
	// searching proj-a for "go" must not return the proj-b row
	ra, err := st.Search(ctx, SearchQuery{Query: "go", Project: "proj-a", Limit: 10})
	if err != nil {
		t.Fatalf("search proj-a: %v", err)
	}
	for _, obs := range ra.Observations {
		if obs.Project != nil && *obs.Project == "proj-b" {
			t.Fatalf("LIKE fallback leaked proj-b row into proj-a search: %+v", obs)
		}
	}
	// searching proj-b for "go" must find exactly the proj-b row
	rb, err := st.Search(ctx, SearchQuery{Query: "go", Project: "proj-b", Limit: 10})
	if err != nil {
		t.Fatalf("search proj-b: %v", err)
	}
	if len(rb.Observations) != 1 {
		t.Fatalf("LIKE fallback proj-b search expected 1 row, got %+v", rb.Observations)
	}
}

// [search-ranking-recency-stability] the composite rank is
// bm25 × (1 + 0.10·pinned + 0.06·1/(1+age_days/30) + 0.04·(rev+dups)/(rev+dups+4)).
// Two otherwise-identical rows differ ONLY in last_seen_at / revision_count —
// the fresher, more stable row must rank first.
func TestSearchRankingRecencyAndStability(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	raw, ok := st.(*sqliteStore)
	if !ok {
		t.Fatalf("expected *sqliteStore, got %T", st)
	}
	seed := func() int64 {
		res, err := st.AddObservation(ctx, AddObservationParams{
			SessionID: "s", Type: "decision", Title: "identical title",
			Content: "identical content body",
			Project: "proj-a", Scope: "project",
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		return res.ID
	}
	a := seed()
	b := seed()
	// b is fresher (30 min in the future vs. "now") and more stable
	// (3 revisions); a is 30 days stale. The ranking formula measures age
	// with JULIANDAY('now'), so parseable absolute timestamps are required
	// (a relative term like "30 days ago" would read as NULL and make the
	// boost 0, masking the formula under test).
	if err := raw.TouchForTest(ctx, b, "2026-11-01 15:00:00", 3); err != nil {
		t.Fatalf("touch b: %v", err)
	}
	if err := raw.TouchForTest(ctx, a, "2026-10-02 15:00:00", 0); err != nil {
		t.Fatalf("touch a: %v", err)
	}
	r, err := st.Search(ctx, SearchQuery{Query: "identical", Project: "proj-a", Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(r.Observations) != 2 {
		t.Fatalf("expected both rows, got %+v", r.Observations)
	}
	if r.Observations[0].ID != b {
		t.Fatalf("recency/stability boosts must order b first: %+v", r.Observations)
	}
}

// [search-ranking-boosts] the pinned term of the composite rank: a pinned row
// must outrank an otherwise-identical unpinned row. The two rows are seeded
// back-to-back and then given the SAME last_seen_at via TouchForTest, so the
// recency term is equal for both and the ONLY ranking delta is the 0.10·pinned
// boost. (Without the timestamp equalization the pin's own updated_at bump
// would let the recency term carry the order, masking the pinned regression.)
func TestSearchRankingPinnedBoost(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	raw, ok := st.(*sqliteStore)
	if !ok {
		t.Fatalf("expected *sqliteStore, got %T", st)
	}
	seed := func() int64 {
		res, err := st.AddObservation(ctx, AddObservationParams{
			SessionID: "s", Type: "decision", Title: "pinned boost check",
			Content: "identical body for the pinned term",
			Project: "proj-a", Scope: "project",
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		return res.ID
	}
	unpinned := seed()
	pinned := seed()
	// pin the second row via the UpdateParams.Pinned path.
	p := true
	if err := st.UpdateObservationForProject(ctx, pinned, "proj-a", UpdateParams{Pinned: &p}); err != nil {
		t.Fatalf("pin: %v", err)
	}
	// equalize last_seen_at so the recency term is identical for both rows.
	for _, id := range []int64{unpinned, pinned} {
		if err := raw.TouchForTest(ctx, id, "2026-11-01 15:00:00", 0); err != nil {
			t.Fatalf("touch %d: %v", id, err)
		}
	}
	r, err := st.Search(ctx, SearchQuery{Query: "pinned boost", Project: "proj-a", Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(r.Observations) != 2 {
		t.Fatalf("expected both rows, got %+v", r.Observations)
	}
	if r.Observations[0].ID != pinned {
		t.Fatalf("pinned boost must order the pinned row first: %+v", r.Observations)
	}
}
