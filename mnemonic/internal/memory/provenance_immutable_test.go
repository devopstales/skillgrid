package memory

import (
	"context"
	"encoding/json"
	"testing"
)

// TestProvenanceImmutability covers 15.2: provenance is immutable once set
// during curation. (1) A Save() whose TopicKey upserts an existing
// observation that ALREADY carries provenance must preserve it — only the
// initial save may establish the chain; a save attempting a DIFFERENT
// provenance is a no-op on the column (a warning is logged). (2) The
// mem_update path (Update) never touches the provenance column: after any
// update the original provenance is byte-for-byte intact. (3) An
// observation that never had provenance keeps provenance NULL after an
// update — the update path cannot establish one either.
func TestProvenanceImmutability(t *testing.T) {
	_, svc := newTestStore(t, "provimm")
	ctx := context.Background()
	sid := newSession(t, svc)

	original := &Provenance{
		SessionID:     sid,
		CurateCommand: "mem_save --scope project",
		SourceFiles:   []string{"src/original.go"},
		LLMReasoning:  "original extraction reasoning",
	}

	// Initial save establishes the chain. (No TopicKey: a topic-key upsert is
	// an UPDATE of the same row — the 24h hash-dedup path INSERTs a NEW row,
	// and a fresh row legitimately starts its own chain. Immutability
	// applies per row, per curation, so the upsert path is the one that must
	// preserve the stored chain.)
	id, err := svc.Save(ctx, SaveInput{
		SessionID:  sid,
		Type:       "decision",
		Title:      "prov immutable note",
		Content:    "first content",
		Provenance: original,
	})
	if err != nil {
		t.Fatalf("initial save: %v", err)
	}
	orig, err := svc.Get(ctx, id)
	if err != nil || orig.Provenance == nil {
		t.Fatalf("initial provenance not set: obs=%+v err=%v", orig, err)
	}
	origJSON, err := json.Marshal(orig.Provenance)
	if err != nil {
		t.Fatalf("marshal original: %v", err)
	}

	// (1a) Topic-key upsert WITHOUT a provenance → preserved unchanged.
	if _, err := svc.Save(ctx, SaveInput{
		SessionID: sid,
		Type:      "decision",
		Title:     "prov immutable note",
		Content:   "revised content, no provenance",
		TopicKey:  "prov/immutable",
	}); err != nil {
		t.Fatalf("upsert without provenance: %v", err)
	}
	after, err := svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get after upsert: %v", err)
	}
	if after.Provenance == nil || !provenanceEqual(after.Provenance, original) {
		t.Fatalf("upsert without provenance must preserve it: got %+v", after.Provenance)
	}

	// (1b) Topic-key upsert WITH a DIFFERENT provenance → original preserved
	// (the attempt is a warning-level no-op, not an overwrite).
	different := &Provenance{
		SessionID:     "some-other-session",
		CurateCommand: "mem_save --other",
		SourceFiles:   []string{"src/other.go"},
		LLMReasoning:  "different reasoning entirely",
	}
	if _, err := svc.Save(ctx, SaveInput{
		SessionID:  sid,
		Type:       "decision",
		Title:      "prov immutable note",
		Content:    "revised content again",
		TopicKey:   "prov/immutable",
		Provenance: different,
	}); err != nil {
		t.Fatalf("upsert with different provenance: %v", err)
	}
	after, err = svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get after different-provenance upsert: %v", err)
	}
	if after.Provenance == nil || !provenanceEqual(after.Provenance, original) {
		t.Fatalf("different provenance must NOT overwrite the original: got %+v", after.Provenance)
	}
	gotJSON, err := json.Marshal(after.Provenance)
	if err != nil {
		t.Fatalf("marshal after: %v", err)
	}
	if string(gotJSON) != string(origJSON) {
		t.Fatalf("original provenance not intact:\n got %s\nwant %s", gotJSON, origJSON)
	}

	// (2) Update() never touches the provenance column.
	if err := svc.Update(ctx, id, UpdateInput{Content: "updated via mem_update"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	after, err = svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if after.Provenance == nil || !provenanceEqual(after.Provenance, original) {
		t.Fatalf("Update() must leave provenance intact: got %+v", after.Provenance)
	}

	// (3) No provenance initially → still NULL after an update (the update
	// path cannot establish a chain either).
	plainID, err := svc.Save(ctx, SaveInput{
		SessionID: sid,
		Type:      "decision",
		Title:     "prov none note",
		Content:   "no chain ever",
	})
	if err != nil {
		t.Fatalf("save plain: %v", err)
	}
	if err := svc.Update(ctx, plainID, UpdateInput{Content: "updated plain"}); err != nil {
		t.Fatalf("update plain: %v", err)
	}
	plain, err := svc.Get(ctx, plainID)
	if err != nil {
		t.Fatalf("get plain: %v", err)
	}
	if plain.Provenance != nil {
		t.Fatalf("update must not establish provenance on a chain-less obs: got %+v", plain.Provenance)
	}
}

func provenanceEqual(a, b *Provenance) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}
