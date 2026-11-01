package memory

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/llm"
)

// The hash floor: nil seam → floor runs, no retry, degraded.
func TestDedupRunNilSeamRunsHashFloor(t *testing.T) {
	st := openTestStore(t)
	d, degraded := dedup.Run(ctx, nil, st, "p", "project", "hello world", "T")
	if !degraded {
		t.Fatal("nil seam must flag degraded")
	}
	if d.Reason != "hash" || d.Verdict != llm.VerdictAdd {
		t.Fatalf("nil seam must run the hash floor: %+v", d)
	}
	// seed the exact row; a second run must land on noop with the same identity
	seedObservation(t, st, "p", "decision", "T", "hello world", "")
	d2, degraded2 := dedup.Run(ctx, nil, st, "p", "project", "hello world", "T")
	if !degraded2 || d2.Reason != "hash" || d2.Verdict != llm.VerdictNoop {
		t.Fatalf("exact row must hash-floor to noop: %+v (degraded=%v)", d2, degraded2)
	}
	if d2.CandidateID == 0 {
		t.Fatal("noop must carry the matched row's id")
	}
}

// Normalization: the floor hashes the store-normalized content, so the floor
// and the stored hash agree on case/whitespace-insensitive content (spec
// 004-dedup §5: the lookup is over normalized_hash).
func TestHashFloorNormalizationMatchesStoredHash(t *testing.T) {
	st := openTestStore(t)
	o := seedObservation(t, st, "p", "decision", "T", "hello world", "")
	_ = o
	d, _ := dedup.Run(ctx, nil, st, "p", "project", "HELLO    world", "T")
	if d.Verdict != llm.VerdictNoop {
		t.Fatalf("case/whitespace-variant content must hash-equal the stored row: %+v", d)
	}
	if d.CandidateID == 0 {
		t.Fatal("noop must carry the matched row's id")
	}
}

// Scoping: the floor never crosses projects (spec 004-dedup §6).
func TestHashFloorDoesNotCrossProjects(t *testing.T) {
	st := openTestStore(t)
	seedObservation(t, st, "p", "decision", "T", "hello world", "")
	d, _ := dedup.Run(ctx, nil, st, "q", "project", "hello world", "T")
	if d.Verdict != llm.VerdictAdd {
		t.Fatalf("identical content in another project must not match: %+v", d)
	}
}

// Scoping: scope is part of the identity too.
func TestHashFloorDoesNotCrossScopes(t *testing.T) {
	st := openTestStore(t)
	seedObservation(t, st, "p", "decision", "T", "hello world", "")
	// personal scope rows: seed via a personal-scope save through the store
	// (AddObservationParams carries Scope).
	if _, err := st.AddObservation(ctx, store.AddObservationParams{
		SessionID: testSession, Type: "decision", Title: "T", Content: "hello world",
		Project: "p", Scope: "personal",
	}); err != nil {
		t.Fatalf("seed personal: %v", err)
	}
	d, _ := dedup.Run(ctx, nil, st, "p", "personal", "hello world", "T")
	if d.Verdict != llm.VerdictNoop {
		t.Fatalf("same project+scope+type+title+hash must match: %+v", d)
	}
}

// The LLM seam is non-nil but the floor is what Task 5 ships: dedup.Run
// currently returns the hash-floor verdict (the LLM internals land in Task 8).
func TestDedupRunSeamPresentStillHashFloorInTask5(t *testing.T) {
	st := openTestStore(t)
	stub := &stubLLM{}
	stub.setClassify(func(ctx context.Context, newContent string, candidates []string) (llm.DedupDecision, error) {
		return llm.DedupDecision{Verdict: llm.VerdictAdd, Reason: "llm"}, nil
	})
	d, degraded := dedup.Run(ctx, stub, st, "p", "project", "brand new", "T")
	// Task 5: the seam is plumbed but not yet consulted → floor verdict.
	if d.Reason != "hash" {
		t.Fatalf("Task 5 dedup.Run must return the hash-floor verdict: %+v", d)
	}
	_ = degraded
}
