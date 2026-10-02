package memory

import (
	"context"
	"testing"
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
