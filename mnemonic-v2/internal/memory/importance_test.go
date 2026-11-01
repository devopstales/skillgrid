package memory

import "testing"

// [usage-raises-importance] the usage term is the named source: more
// retrieval usage scores higher, everything else equal.
func TestImportanceUsageRaisesScore(t *testing.T) {
	low := Importance(10, 0, "decision")
	high := Importance(50, 0, "decision")
	if high <= low {
		t.Fatalf("50 usage must outscore 10 usage: %f vs %f", high, low)
	}
}

// The formula's pinned values (spec 002-schema §2.2.3, v1 dream.go:382-400):
// clamp(0.6*min(usage,50)/50 + 0.37*recency + 0.2*typeDurability, 0, 1).
func TestImportanceFormulaPinnedValues(t *testing.T) {
	cases := []struct {
		name             string
		usage, ageDays   int
		typ              string
		want             float64
	}{
		{"fresh zero-usage transient", 0, 0, "learning", 0.37 + 0.08},
		{"fresh zero-usage durable", 0, 0, "decision", 0.37 + 0.18},
		{"fresh zero-usage config", 0, 0, "config", 0.37 + 0.18},
		{"fresh zero-usage convention", 0, 0, "convention", 0.37 + 0.18},
		{"usage capped at 50", 500, 0, "learning", 0.6 + 0.37 + 0.08},
		{"age at grace window keeps full recency", 0, 30, "learning", 0.37 + 0.08},
		{"age 45 decays recency to 0.5", 0, 45, "learning", 0.37 * 0.5 + 0.08},
		{"age 60 recency zero", 0, 60, "learning", 0.08},
		{"age past 60 clamps at zero", 0, 90, "learning", 0.08},
		{"all terms max clamp to 1", 50, 0, "decision", 0.6 + 0.37 + 0.18},
	}
	for _, c := range cases {
		got := Importance(c.usage, c.ageDays, c.typ)
		if diff := got - c.want; diff < -1e-9 || diff > 1e-9 {
			t.Errorf("%s: Importance(%d, %d, %q) = %f, want %f", c.name, c.usage, c.ageDays, c.typ, got, c.want)
		}
	}
}

// Durable vocabulary: exactly {decision, architecture, convention, config}
// carry the 0.9 weight; the rest 0.4.
func TestImportanceTypeDurabilityVocabulary(t *testing.T) {
	for _, typ := range []string{"decision", "architecture", "convention", "config"} {
		if got := Importance(0, 0, typ); got != 0.37+0.18 {
			t.Errorf("durable type %q: got %f, want %f", typ, got, 0.37+0.18)
		}
	}
	for _, typ := range []string{"bugfix", "pattern", "discovery", "preference", "learning", "session_summary"} {
		if got := Importance(0, 0, typ); got != 0.37+0.08 {
			t.Errorf("transient type %q: got %f, want %f", typ, got, 0.37+0.08)
		}
	}
}

// [usage-raises-importance] triangulation — the pipeline's stored importance
// column is the formula over the row's retrieval_usage (0 on a fresh insert;
// Task 11's BumpRetrievalUsage feeds later reads).
func TestSaveStoresImportanceFromRowState(t *testing.T) {
	svc := newTestService(t)
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "T", Content: "C", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := svc.st.GetObservation(ctx, r.ID, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	// fresh row: usage 0, age 0 → 0.37 + 0.2*0.9
	if want := 0.37 + 0.18; got.Importance < want-1e-9 || got.Importance > want+1e-9 {
		t.Fatalf("stored importance %f, want %f (usage=0, fresh)", got.Importance, want)
	}
}
