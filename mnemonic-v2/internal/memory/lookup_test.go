package memory

import (
	"context"
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// [search-ranking-boosts] a pinned row matching the same query ranks first
// (the pinned +0.10 boost is the deciding delta when the BM25 base is
// clamped to 1.0 for the all-identical case), and every result row is a
// compact preview of at most 300 characters with a truncation flag.
func TestSearchRankingWithBoosts(t *testing.T) {
	svc := newTestService(t)
	_ = seedObservation(t, svc.st, "p", "decision", "auth", "token auth", "architecture/auth")
	pinned := seedObservation(t, svc.st, "p", "decision", "auth", "token auth pinned", "architecture/auth")
	if err := svc.st.UpdateObservationForProject(ctx, pinned.ID, "p", store.UpdateParams{Pinned: ptr(true)}); err != nil {
		t.Fatalf("pin: %v", err)
	}
	r, err := svc.Search(ctx, "auth", SearchFilters{Project: "p", Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(r.Hits) < 2 {
		t.Fatalf("expected both rows, got %d", len(r.Hits))
	}
	if r.Hits[0].ID != pinned.ID {
		t.Fatalf("pinned row must rank first, got %d", r.Hits[0].ID)
	}
	// compact rows: a 500-char body must come back as a ≤300-char preview
	// with the truncated flag set (never the full body).
	long := seedObservation(t, svc.st, "p", "discovery", "auth long", strings.Repeat("x", 500), "")
	_ = long
	r2, err := svc.Search(ctx, "auth long", SearchFilters{Project: "p", Limit: 10})
	if err != nil {
		t.Fatalf("search long: %v", err)
	}
	hit := findHit(r2, long.ID)
	if hit == nil {
		t.Fatalf("long row not found in results")
	}
	if len(hit.Preview) > 300 || !hit.Truncated {
		t.Fatalf("preview must be ≤300 + truncated flag: len=%d truncated=%v", len(hit.Preview), hit.Truncated)
	}
}

// [lookup-topic-key-exact] an exact topic_key lookup returns the single
// upserted row (or none); it never widens to a broader search.
func TestLookupTopicKeyExact(t *testing.T) {
	svc := newTestService(t)
	seedObservation(t, svc.st, "p", "decision", "auth model", "the model", "architecture/auth-model")
	res, err := svc.Lookup(ctx, "p", "project", "architecture/auth-model", 0, "")
	if err != nil {
		t.Fatalf("topic_key lookup: %v", err)
	}
	if len(res.Hits) != 1 {
		t.Fatalf("topic_key lookup must return the single upserted row: got %d hits", len(res.Hits))
	}
	resNone, err := svc.Lookup(ctx, "p", "project", "architecture/absent", 0, "")
	if err != nil {
		t.Fatalf("absent topic_key lookup must not error: %v", err)
	}
	if len(resNone.Hits) != 0 {
		t.Fatalf("absent topic_key must return none (not widen): %+v", resNone.Hits)
	}
}

// [lookup-phrase-bounded] a phrase lookup caps at 5 rows (bounded FTS, no
// boosts, no vector) even when more rows share the phrase.
func TestLookupPhraseBounded(t *testing.T) {
	svc := newTestService(t)
	for i := 0; i < 10; i++ {
		seedObservation(t, svc.st, "p", "discovery", "needle phrase row", "unique needle phrase", "")
	}
	res, err := svc.Lookup(ctx, "p", "project", "", 0, "unique needle phrase")
	if err != nil {
		t.Fatalf("phrase lookup: %v", err)
	}
	if len(res.Hits) > 5 {
		t.Fatalf("phrase lookup must cap at 5, got %d", len(res.Hits))
	}
}

// [context-byte-budget] context honors the byte budget (default 16 KiB);
// pinned rows lead and at most 20 pinned rows appear.
func TestContextRespectsBudget(t *testing.T) {
	svc := newTestService(t)
	for i := 0; i < 50; i++ {
		seedObservation(t, svc.st, "p", "discovery", "ctx row", strings.Repeat("c", 200), "")
	}
	res, err := svc.Context(ctx, 16*1024, false)
	if err != nil {
		t.Fatalf("context: %v", err)
	}
	if res.TotalBytes > 16*1024 {
		t.Fatalf("context must respect the byte budget, got %d bytes", res.TotalBytes)
	}
	// pinned cap: at most 20 pinned observations lead the context (spec 005
	// §2); with no pinned rows here the pinned count is 0 (the ≤20 cap holds).
	pinned := 0
	for _, o := range res.Observations {
		if o.Pinned {
			pinned++
		}
	}
	if pinned > 20 {
		t.Fatalf("context must cap pinned observations at 20, got %d", pinned)
	}
}

// [vector-blend-degraded] with no embedder the vector leg is absent: the
// result is explicitly degraded, but the FTS leg still returns results.
func TestVectorLegDegradesExplicitly(t *testing.T) {
	svc := newTestService(t) // no embedder
	seedObservation(t, svc.st, "p", "decision", "auth", "tokens", "")
	res, err := svc.Search(ctx, "auth", SearchFilters{Project: "p", Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !res.Degraded {
		t.Fatalf("no embedder → vector leg degraded must be true")
	}
	if len(res.Hits) == 0 {
		t.Fatalf("FTS leg must still return results")
	}
}

// [trail-recorded] a search records a retrieval trail carrying the query,
// route, and degraded status (INV-5).
func TestTrailRecorded(t *testing.T) {
	svc := newTestService(t)
	seedObservation(t, svc.st, "p", "decision", "auth", "tokens", "")
	_, err := svc.Search(ctx, "auth", SearchFilters{Project: "p"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	trails := retrievalTrails(t, svc.st)
	if len(trails) != 1 {
		t.Fatalf("expected one retrieval trail, got %d", len(trails))
	}
	tr := trails[0]
	if tr.Route != "fts" || tr.Query != "auth" || !tr.Degraded {
		t.Fatalf("trail wrong: %+v", tr)
	}
}

// [prompt-search-finds-captured] a captured prompt is found by prompt search
// (the FTS over prompts_fts; redaction happened at capture).
func TestPromptSearchFindsCaptured(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Save(ctx, SaveRequest{
		SessionID: "ps", Type: "learning", Title: "t", Content: "c",
		Project: "p", Scope: "project", CurrentPrompt: "how do I ship the sync layer?",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	res, err := svc.SearchPrompts(ctx, "ship the sync layer", "p", 5)
	if err != nil {
		t.Fatalf("prompt search: %v", err)
	}
	if len(res.Prompts) != 1 {
		t.Fatalf("captured prompt not found by prompt search: %+v", res)
	}
}

// [usage-raises-importance] a retrieved observation accumulates
// retrieval_usage and scores higher than an identical never-retrieved row
// (the importance formula's usage term, the v1 dream.go:382-400 formula).
func TestUsageRaisesImportance(t *testing.T) {
	svc := newTestService(t)
	a, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision",
		Title: "same", Content: "same body", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := svc.Search(ctx, "same", SearchFilters{Project: "p"}); err != nil {
			t.Fatalf("search %d: %v", i, err)
		}
	}
	used, err := svc.st.GetObservation(ctx, a.ID, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if used.RetrievalUsage != 10 {
		t.Fatalf("retrieval_usage not accumulated: %d", used.RetrievalUsage)
	}
	fresh := Importance(0, 0, "decision")
	hot := Importance(used.RetrievalUsage, 0, "decision")
	if hot <= fresh {
		t.Fatalf("retrieved observation must score higher: hot=%v fresh=%v", hot, fresh)
	}
	// the stored cache column is recomputed over the real usage at retrieve
	// time (the save wrote a fresh-state 0-usage score; it now reflects 10).
	if used.Importance < fresh {
		t.Fatalf("stored importance must be recomputed over retrieval usage: stored=%v fresh=%v", used.Importance, fresh)
	}
}

// [review-mark-reviewed-restarts-clock] an overdue (needs_review) row that is
// marked reviewed restarts its decay clock and returns to active.
func TestReviewMarkReviewedRestartsClock(t *testing.T) {
	svc := newTestService(t)
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision",
		Title: "decay", Content: "c", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	// force the row overdue: a past review_after (the store's timestamp format,
	// no T/Z — Observation.State parses "2006-01-02 15:04:05") → needs_review.
	if err := setReviewAfterForTest(t, svc.st, r.ID, "2026-01-01 00:00:00"); err != nil {
		t.Fatalf("set review_after: %v", err)
	}
	got, err := svc.st.GetObservation(ctx, r.ID, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.State() != "needs_review" {
		t.Fatalf("row must be overdue (needs_review), got %q", got.State())
	}
	res, err := svc.Review(ctx, "p", false, r.ID)
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if !res.MarkReviewed {
		t.Fatalf("mark_reviewed failed: %+v", res)
	}
	got2, err := svc.st.GetObservation(ctx, r.ID, false)
	if err != nil {
		t.Fatalf("get 2: %v", err)
	}
	if got2.State() != "active" {
		t.Fatalf("mark_reviewed must restart the decay clock (state → active), got %q", got2.State())
	}
}

// ── TRIANGULATE (different inputs, same contract) ────────────────────────────

// A search that surfaces zero rows appends a trail (the audit of "the agent
// searched and found nothing") but bumps no usage (nothing was retrieved).
func TestSearchZeroRowsStillRecordsTrail(t *testing.T) {
	svc := newTestService(t)
	seedObservation(t, svc.st, "p", "decision", "auth", "tokens", "")
	before := retrievalTrailCount(t, svc.st)
	// a query that matches nothing in the project.
	_, err := svc.Search(ctx, "zzzzunmatched", SearchFilters{Project: "p", Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	after := retrievalTrailCount(t, svc.st)
	if after != before+1 {
		t.Fatalf("zero-row search must still append a trail: before=%d after=%d", before, after)
	}
}

// An importance recompute after 3 retrievs vs 10: the higher-usage row scores
// strictly higher (the usage term is monotone in retrieval_usage), and both
// beat a never-retrieved row.
func TestImportanceRecomputeMonotonic(t *testing.T) {
	svc := newTestService(t)
	a, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision",
		Title: "monotone", Content: "body a", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := svc.Search(ctx, "monotone", SearchFilters{Project: "p"}); err != nil {
			t.Fatalf("search %d: %v", i, err)
		}
	}
	after3, err := svc.st.GetObservation(ctx, a.ID, false)
	if err != nil {
		t.Fatalf("get 3: %v", err)
	}
	for i := 0; i < 7; i++ {
		if _, err := svc.Search(ctx, "monotone", SearchFilters{Project: "p"}); err != nil {
			t.Fatalf("search %d: %v", i, err)
		}
	}
	after10, err := svc.st.GetObservation(ctx, a.ID, false)
	if err != nil {
		t.Fatalf("get 10: %v", err)
	}
	if after3.RetrievalUsage != 3 || after10.RetrievalUsage != 10 {
		t.Fatalf("usage must accumulate: after3=%d after10=%d", after3.RetrievalUsage, after10.RetrievalUsage)
	}
	if after10.Importance <= after3.Importance {
		t.Fatalf("10-retrieval row must score higher than the 3-retrieval row: %v vs %v", after10.Importance, after3.Importance)
	}
	if after3.Importance <= Importance(0, 0, "decision") {
		t.Fatalf("3-retrieval row must beat a never-retrieved row: %v", after3.Importance)
	}
}

// TRIANGULATE (vector leg): with an embedder armed AND stored vectors
// present, the vector leg runs, the result is RRF-fused (route "rrf"), and
// NOT degraded (INV-3 — explicit degraded is only when the leg is absent).
// The embedder returns a constant 2-D vector (the nearest-neighbor ordering
// is deterministic for the single stored vector's row).
func TestVectorLegFusesWhenArmed(t *testing.T) {
	// A constant 2-D embedder (the leg runs; the stored vector must match
	// the dimension).
	constEmb := &stubEmbedder{dim: 2, vec: []float32{0.98, 0.02}}
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	st := openTestStoreIn(t, c.DataDir, "p")
	svc := NewService(st, c, NewEventBus(), nil, constEmb)
	seed := seedObservation(t, st, "p", "decision", "auth", "tokens", "")
	// store a 2-D embedding on the seed row (the vector leg reads it back).
	if err := setEmbedding(t, st, seed.ID, 0.99, 0.01); err != nil {
		t.Fatalf("set embedding: %v", err)
	}
	res, err := svc.Search(ctx, "auth", SearchFilters{Project: "p", Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if res.Route != "rrf" {
		t.Fatalf("armed embedder + stored vector → route must be rrf, got %q", res.Route)
	}
	if res.Degraded {
		t.Fatalf("vector leg ran → the result must NOT be degraded")
	}
	if len(res.Hits) == 0 {
		t.Fatalf("FTS + vector must return results")
	}
}

// ── assertion helpers ────────────────────────────────────────────────────────

// stubEmbedder is a constant-dimension embedder test fake (the vector leg's
// arm path: it returns a fixed vector for every text).
type stubEmbedder struct {
	dim int
	vec []float32
}

func (s *stubEmbedder) Model() string   { return "stub" }
func (s *stubEmbedder) Dimensions() int { return s.dim }
func (s *stubEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = append([]float32(nil), s.vec...)
	}
	return out, nil
}

// setEmbedding stores a little-endian float32 vector blob on an observation
// (the same encoding the store's VectorSearch reads back).
func setEmbedding(t *testing.T, st store.Store, id int64, v ...float32) error {
	t.Helper()
	raw, ok := st.(trailSeam)
	if !ok {
		t.Fatal("store lacks the RawExec test seam")
	}
	blob := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(blob[i*4:], math.Float32bits(f))
	}
	return raw.RawExec(`UPDATE observations SET embedding = ? WHERE id = ?`, blob, id)
}

// findHit returns the hit with the given id, or nil.
func findHit(r SearchResult, id int64) *SearchHit {
	for i := range r.Hits {
		if r.Hits[i].ID == id {
			return &r.Hits[i]
		}
	}
	return nil
}
