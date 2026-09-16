package memory

import (
	"context"
	"testing"
)

func TestBlendedSearchFusesWhenEmbedOn(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "1")
	_, svc := newTestStore(t, "fuseproj")
	sid := newSession(t, svc)
	ctx := context.Background()

	lexID, err := svc.Save(ctx, SaveInput{
		Title: "authentication login password", Type: "decision",
		Content: "lexical heavy match", SessionID: sid, Scope: "project",
	})
	if err != nil {
		t.Fatal(err)
	}
	semID, err := svc.Save(ctx, SaveInput{
		Title: "session design notes", Type: "decision",
		Content: "semantic neighbour", SessionID: sid, Scope: "project",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Strong vector on the semantic row; weak/absent on lexical.
	if err := svc.SetEmbedding(ctx, semID, EncodeVector(Vector{Data: []float32{1, 0}}), "test"); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetEmbedding(ctx, lexID, EncodeVector(Vector{Data: []float32{0.1, 0.9}}), "test"); err != nil {
		t.Fatal(err)
	}

	hits, err := svc.BlendedSearch(ctx, "authentication login password", "any", "", Vector{Data: []float32{1, 0}}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) < 2 {
		t.Fatalf("want fused hits, got %d", len(hits))
	}
	// With embed on + query aligned to semID, RRF should keep both present.
	ids := map[int64]bool{}
	for _, h := range hits {
		ids[h.ID] = true
	}
	if !ids[lexID] || !ids[semID] {
		t.Fatalf("fusion missing rows: %+v", ids)
	}
}

func TestMissingEmbedderDegradesToKeywordOnly(t *testing.T) {
	// Flag on but no stored embeddings and a query vector present → FTS floor.
	t.Setenv("MNEMONIC_EMBED", "1")
	_, svc := newTestStore(t, "degradproj")
	sid := newSession(t, svc)
	ctx := context.Background()
	if _, err := svc.Save(ctx, SaveInput{
		Title: "degrade keyword hit", Type: "decision", Content: "only fts",
		SessionID: sid, Scope: "project",
	}); err != nil {
		t.Fatal(err)
	}
	hits, err := svc.BlendedSearch(ctx, "degrade keyword hit", "any", "", Vector{Data: []float32{1, 0, 0}}, 10)
	if err != nil {
		t.Fatalf("must not hard-fail when vectors absent: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected FTS-only results")
	}
}

func TestDisabledFlagYieldsNoVectorRecall(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	_, svc := newTestStore(t, "embedoff")
	sid := newSession(t, svc)
	ctx := context.Background()
	a, err := svc.Save(ctx, SaveInput{
		Title: "embed off alpha", Type: "decision", Content: "a", SessionID: sid, Scope: "project",
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Save(ctx, SaveInput{
		Title: "embed off beta", Type: "decision", Content: "b", SessionID: sid, Scope: "project",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SetEmbedding(ctx, b, EncodeVector(Vector{Data: []float32{1, 0}}), "test"); err != nil {
		t.Fatal(err)
	}
	ftsOnly, err := svc.Search(ctx, "embed off", "any", 10)
	if err != nil {
		t.Fatal(err)
	}
	blended, err := svc.BlendedSearch(ctx, "embed off", "any", "", Vector{Data: []float32{1, 0}}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(ftsOnly) != len(blended) {
		t.Fatalf("flag off must ignore vector leg: fts=%d blended=%d", len(ftsOnly), len(blended))
	}
	for i := range ftsOnly {
		if ftsOnly[i].ID != blended[i].ID {
			t.Fatalf("order diverged under EmbedOff at %d: fts=%d blended=%d (a=%d b=%d)", i, ftsOnly[i].ID, blended[i].ID, a, b)
		}
	}
}
