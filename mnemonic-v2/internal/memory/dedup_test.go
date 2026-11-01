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
	d, degraded := runDedup(ctx, nil, st, "p", "project", "hello world", "decision", "T")
	if !degraded {
		t.Fatal("nil seam must flag degraded")
	}
	if d.Reason != "hash" || d.Verdict != llm.VerdictAdd {
		t.Fatalf("nil seam must run the hash floor: %+v", d)
	}
	// seed the exact row; a second run must land on noop with the same identity
	seedObservation(t, st, "p", "decision", "T", "hello world", "")
	d2, degraded2 := runDedup(ctx, nil, st, "p", "project", "hello world", "decision", "T")
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
	d, _ := runDedup(ctx, nil, st, "p", "project", "HELLO    world", "decision", "T")
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
	d, _ := runDedup(ctx, nil, st, "q", "project", "hello world", "decision", "T")
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
	d, _ := runDedup(ctx, nil, st, "p", "personal", "hello world", "decision", "T")
	if d.Verdict != llm.VerdictNoop {
		t.Fatalf("same project+scope+type+title+hash must match: %+v", d)
	}
}

// The LLM seam is non-nil: dedup.Run consults the seam (Task 8), not the
// floor — the LLM-first inversion (spec 004-dedup §1) over v1's hash default.
func TestDedupRunSeamPresentConsultsLLM(t *testing.T) {
	st := openTestStore(t)
	stub := &stubLLM{}
	stub.setClassify(func(ctx context.Context, newContent string, candidates []string) (llm.DedupDecision, error) {
		return llm.DedupDecision{Verdict: llm.VerdictAdd, Reason: "llm"}, nil
	})
	d, degraded := runDedup(ctx, stub, st, "p", "project", "brand new", "decision", "T")
	if d.Reason != "llm" || d.Verdict != llm.VerdictAdd {
		t.Fatalf("the seam must be consulted when present (LLM-first): %+v", d)
	}
	if degraded {
		t.Fatalf("a successful seam is not degraded")
	}
}
