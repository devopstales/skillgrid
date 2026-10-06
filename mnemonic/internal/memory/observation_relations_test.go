package memory

import (
	"context"
	"testing"
)

// seedObservation saves a distinct observation and returns its id. The title
// is unique per call (the dedup key is title+content+type within 24h), so the
// returned ids are always distinct.
func seedObservation(t *testing.T, svc *Service, title, content string) int64 {
	t.Helper()
	// The observations table has a FK on session_id; guarantee the session
	// exists (idempotent) before the first save in this store.
	if _, err := svc.store.DB.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status, title, summary)
		VALUES ('sess-rel', ?, '/tmp', '2026-01-01T00:00:00Z', 'ended', 'rel session', 'rel session summary')`,
		svc.projectID); err != nil {
		t.Fatalf("ensure session: %v", err)
	}
	id, err := svc.Save(context.Background(), SaveInput{
		SessionID: "sess-rel",
		Type:      "decision",
		Title:     title,
		Content:   content,
	})
	if err != nil {
		t.Fatalf("save %q: %v", title, err)
	}
	return id
}

// TestRelationEdgeCreation covers 14.1: typed relation edges between
// observations store source_id, target_id, relation_type, and confidence.
// AddRelation(A, B, "mentions", 0.9) round-trips through GetRelations with the
// correct fields, and every one of the 5 canonical relation types is accepted.
func TestRelationEdgeCreation(t *testing.T) {
	_, svc := newTestStore(t, "relproj")
	ctx := context.Background()

	a := seedObservation(t, svc, "rel note A", "content A")
	b := seedObservation(t, svc, "rel note B", "content B")

	if _, err := svc.AddRelation(ctx, a, b, "mentions", 0.9); err != nil {
		t.Fatalf("AddRelation: %v", err)
	}

	rels, err := svc.GetRelations(ctx, a, 0.0)
	if err != nil {
		t.Fatalf("GetRelations: %v", err)
	}
	if len(rels) != 1 {
		t.Fatalf("expected 1 relation for A, got %d: %+v", len(rels), rels)
	}
	got := rels[0]
	if got.SourceID != a {
		t.Fatalf("source_id: got %d, want %d", got.SourceID, a)
	}
	if got.TargetID != b {
		t.Fatalf("target_id: got %d, want %d", got.TargetID, b)
	}
	if got.RelationType != "mentions" {
		t.Fatalf("relation_type: got %q, want %q", got.RelationType, "mentions")
	}
	if got.Confidence != 0.9 {
		t.Fatalf("confidence: got %v, want 0.9", got.Confidence)
	}

	// All 5 canonical relation types can be created on a fresh pair.
	types := []string{"mentions", "depends_on", "contradicts", "supports", "references"}
	for i, rt := range types {
		src := seedObservation(t, svc, "rel type src "+string(rune('a'+i)), "src "+rt)
		dst := seedObservation(t, svc, "rel type dst "+string(rune('a'+i)), "dst "+rt)
		if _, err := svc.AddRelation(ctx, src, dst, rt, 0.5); err != nil {
			t.Fatalf("AddRelation type %q: %v", rt, err)
		}
		got, err := svc.GetRelations(ctx, src, 0.0)
		if err != nil {
			t.Fatalf("GetRelations after %q: %v", rt, err)
		}
		if len(got) != 1 || got[0].RelationType != rt {
			t.Fatalf("type %q: expected 1 %q relation, got %+v", rt, rt, got)
		}
	}
}

// TestRelationConfidenceFiltering covers 14.3: GetRelations honors a
// minConfidence floor (WHERE confidence >= minConfidence). With three
// relations at 0.3/0.7/0.95, a floor of 0.5 returns only the 0.7 and 0.95
// edges; a floor of 0.0 returns all three.
func TestRelationConfidenceFiltering(t *testing.T) {
	_, svc := newTestStore(t, "relconf")
	ctx := context.Background()

	src := seedObservation(t, svc, "rel conf src", "content src")
	dst1 := seedObservation(t, svc, "rel conf dst low", "content low")
	dst2 := seedObservation(t, svc, "rel conf dst mid", "content mid")
	dst3 := seedObservation(t, svc, "rel conf dst high", "content high")

	for _, c := range []struct {
		dst        int64
		rel        string
		confidence float64
	}{
		{dst1, "mentions", 0.3},
		{dst2, "supports", 0.7},
		{dst3, "references", 0.95},
	} {
		if _, err := svc.AddRelation(ctx, src, c.dst, c.rel, c.confidence); err != nil {
			t.Fatalf("AddRelation %v: %v", c.confidence, err)
		}
	}

	// minConfidence=0.5 → only the 0.7 and 0.95 relations survive.
	filt, err := svc.GetRelations(ctx, src, 0.5)
	if err != nil {
		t.Fatalf("GetRelations 0.5: %v", err)
	}
	if len(filt) != 2 {
		t.Fatalf("minConfidence=0.5: expected 2 relations, got %d: %+v", len(filt), filt)
	}
	seen := map[float64]bool{}
	for _, r := range filt {
		if r.Confidence < 0.5 {
			t.Fatalf("minConfidence=0.5 returned a below-floor relation: %+v", r)
		}
		seen[r.Confidence] = true
	}
	if !seen[0.7] || !seen[0.95] {
		t.Fatalf("minConfidence=0.5 must keep 0.7 and 0.95: %+v", filt)
	}

	// minConfidence=0.0 → all three relations.
	all, err := svc.GetRelations(ctx, src, 0.0)
	if err != nil {
		t.Fatalf("GetRelations 0.0: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("minConfidence=0.0: expected all 3 relations, got %d: %+v", len(all), all)
	}
}
