package memory

import (
	"context"
	"strings"
	"testing"
)

// TestDreamSynthesize (014 step 12.2): L0/L1 session summaries are lifted into
// a single higher-tier (L2) summary that references the source sessions.
func TestDreamSynthesize(t *testing.T) {
	_, svc := newTestStore(t, "synthesize")
	sidA := newSession(t, svc)
	sidB := newSession(t, svc)
	ctx := context.Background()
	de := NewDreamExecutor(svc)
	de.SetLLM(&stubDreamLLM{synthesize: "L2 summary of the two sessions."})

	sum, err := de.synthesize(ctx, []Memory{
		{SessionID: sidA, Content: "Session A: built the store pooling layer."},
		{SessionID: sidB, Content: "Session B: added the FTS trigram tokenizer."},
	})
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if sum.Tier != DreamTierL2 {
		t.Fatalf("tier = %q, want %q", sum.Tier, DreamTierL2)
	}
	if sum.SourceCount != 2 {
		t.Fatalf("source count = %d, want 2", sum.SourceCount)
	}
	stored, err := svc.Get(ctx, sum.ID)
	if err != nil {
		t.Fatalf("get synthesis: %v", err)
	}
	for _, s := range []string{sidA, sidB} {
		if !strings.Contains(stored.Content, s) {
			t.Errorf("synthesis does not reference source session %q", s)
		}
	}
	if !strings.Contains(stored.Content, "L2 summary of the two sessions.") {
		t.Errorf("synthesis missing LLM summary: %q", stored.Content)
	}
	if len(sum.SessionIDs) != 2 {
		t.Fatalf("session ids = %d, want 2", len(sum.SessionIDs))
	}
}
