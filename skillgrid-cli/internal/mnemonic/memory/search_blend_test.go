package memory

import (
	"context"
	"testing"
	"time"
)

func TestOwnerScopedBlendKeywordFloor(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	fx := newOwnerFixture(t, "blend-kw")
	ctx := context.Background()
	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "banana stand", Content: "banana stand notes", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	hits, err := fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerA, "", "banana", "any", "", 10, Vector{})
	if err != nil {
		t.Fatalf("blend: %v", err)
	}
	if len(hits) != 1 || hits[0].Observation.ID != id {
		t.Fatalf("hits = %+v", hits)
	}
	if hits[0].MatchedVia != "keyword" {
		t.Fatalf("matched_via = %q", hits[0].MatchedVia)
	}
	if hits[0].Signals.Vector != 0 || hits[0].Signals.Entity != 0 {
		t.Fatalf("signals = %+v", hits[0].Signals)
	}
	if hits[0].Signals.Keyword <= 0 || hits[0].Signals.Keyword > 1 {
		t.Fatalf("keyword signal out of range: %v", hits[0].Signals.Keyword)
	}
}

func TestOwnerScopedBlendHidesPrivate(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	fx := newOwnerFixture(t, "blend-priv")
	ctx := context.Background()
	if _, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "secret banana", Content: "secret banana", Owner: fx.ownerA,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	hits, err := fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerB, "", "banana", "any", "", 10, Vector{})
	if err != nil {
		t.Fatalf("blend: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("private row leaked: %+v", hits)
	}
}

func TestOwnerScopedBlendHybrid(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "1")
	fx := newOwnerFixture(t, "blend-hyb")
	ctx := context.Background()
	lexID, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "lexical banana", Content: "lexical banana", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save lex: %v", err)
	}
	semID, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "unrelated topic", Content: "unrelated topic", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save sem: %v", err)
	}
	if err := fx.svc.SetEmbedding(ctx, lexID, EncodeVector(Vector{Data: []float32{0, 1}}), "test"); err != nil {
		t.Fatalf("embed lex: %v", err)
	}
	if err := fx.svc.SetEmbedding(ctx, semID, EncodeVector(Vector{Data: []float32{1, 0}}), "test"); err != nil {
		t.Fatalf("embed sem: %v", err)
	}
	hits, err := fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerA, "", "banana", "any", "", 10, Vector{Data: []float32{1, 0}})
	if err != nil {
		t.Fatalf("blend: %v", err)
	}
	via := map[int64]string{}
	for _, h := range hits {
		via[h.Observation.ID] = h.MatchedVia
		for _, v := range []float64{h.Signals.Keyword, h.Signals.Vector, h.Signals.Recency, h.Signals.Entity, h.Signals.Decay, h.Signals.Importance} {
			if v < 0 || v > 1 {
				t.Fatalf("signal out of range on %d: %+v", h.Observation.ID, h.Signals)
			}
		}
	}
	if via[lexID] != "hybrid" {
		t.Fatalf("lexical hit matched_via = %q, want hybrid", via[lexID])
	}
	if via[semID] != "vector" {
		t.Fatalf("semantic hit matched_via = %q, want vector", via[semID])
	}
}

// TestDecayDisabledKeepsBM25 is TICKET-03: with decay explicitly off, blend
// order matches SearchOwnerScoped (BM25) on an equivalent pair. The hot row
// would outrank the cold row if decay ran; it must not.
func TestDecayDisabledKeepsBM25(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	ctx := context.Background()
	seen := time.Now().UTC().Add(-40 * 24 * time.Hour).Format(time.RFC3339)

	seed := func(t *testing.T, project string) (fx *ownerFixture, coldID, hotID int64) {
		t.Helper()
		fx = newOwnerFixture(t, project)
		var err error
		coldID, err = fx.svc.Save(ctx, SaveInput{
			SessionID: fx.sessionID, Type: "decision",
			Title: "banana banana banana", Content: "banana banana banana", Owner: fx.ownerA,
		})
		if err != nil {
			t.Fatalf("save cold: %v", err)
		}
		hotID, err = fx.svc.Save(ctx, SaveInput{
			SessionID: fx.sessionID, Type: "decision",
			Title: "banana", Content: "banana", Owner: fx.ownerA,
		})
		if err != nil {
			t.Fatalf("save hot: %v", err)
		}
		if _, err := fx.svc.DB().Exec(`UPDATE observations SET retrieval_usage = 0, last_seen_at = ? WHERE id = ?`, seen, coldID); err != nil {
			t.Fatalf("stamp cold: %v", err)
		}
		if _, err := fx.svc.DB().Exec(`UPDATE observations SET retrieval_usage = 20, last_seen_at = ? WHERE id = ?`, seen, hotID); err != nil {
			t.Fatalf("stamp hot: %v", err)
		}
		return fx, coldID, hotID
	}

	plain, _, _ := seed(t, "decay-off-bm25")
	raw, err := plain.svc.SearchOwnerScoped(ctx, plain.ownerA, "", "banana", "any", "", 10)
	if err != nil {
		t.Fatalf("bm25: %v", err)
	}
	if len(raw) != 2 {
		t.Fatalf("bm25 hits = %d, want 2", len(raw))
	}

	fx, coldID, hotID := seed(t, "decay-off-blend")
	fx.svc.SetDecay(DecayConfig{Enabled: false})
	hits, err := fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerA, "", "banana", "any", "", 10, Vector{})
	if err != nil {
		t.Fatalf("blend: %v", err)
	}
	if len(hits) != len(raw) {
		t.Fatalf("blend hits = %d, bm25 hits = %d", len(hits), len(raw))
	}
	for i := range raw {
		if hits[i].Observation.Title != raw[i].Title {
			t.Fatalf("decay reordered hit %d: blend %q, bm25 %q", i, hits[i].Observation.Title, raw[i].Title)
		}
	}
	if hits[0].Observation.ID != coldID {
		t.Fatalf("first hit = %d, want cold %d (hot %d must stay second)", hits[0].Observation.ID, coldID, hotID)
	}
}

// TestOwnerScopedBlendEmbedderError is the caller contract for an embedder
// failure: mem_search drops the error and passes an empty query vector.
// Embedding stays enabled so a non-empty vector would have taken the hybrid
// leg; the empty vector must still return hits as keyword.
func TestOwnerScopedBlendEmbedderError(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "1")
	fx := newOwnerFixture(t, "blend-emb-err")
	ctx := context.Background()
	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID, Type: "decision",
		Title: "banana stand", Content: "banana stand notes", Owner: fx.ownerA,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	hits, err := fx.svc.SearchOwnerScopedBlend(ctx, fx.ownerA, "", "banana", "any", "", 10, Vector{})
	if err != nil {
		t.Fatalf("blend: %v", err)
	}
	if len(hits) != 1 || hits[0].Observation.ID != id {
		t.Fatalf("hits = %+v", hits)
	}
	if hits[0].MatchedVia != "keyword" {
		t.Fatalf("matched_via = %q", hits[0].MatchedVia)
	}
	if hits[0].Signals.Vector != 0 {
		t.Fatalf("signals.vector = %v", hits[0].Signals.Vector)
	}
}
