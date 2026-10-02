package service

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// TestNewDedupLLMBackendSatisfiesDedupLLM is TICKET-05: newDedupLLMBackend
// returns a non-nil value that satisfies memory.DedupLLM (the 4-way Classify
// seam). This is the compile-time contract the openProject wiring relies on.
func TestNewDedupLLMBackendSatisfiesDedupLLM(t *testing.T) {
	var b memory.DedupLLM = newDedupLLMBackend()
	if b == nil {
		t.Fatal("newDedupLLMBackend must return non-nil")
	}
}

// TestDedupLLMBackendClassify is TICKET-05: the backend reduces the LLM's
// strict 4-way JSON verdict to a memory.DedupDecision, mapping the 0-based
// candidate index to a 1-based observation id for update/delete, and errors on
// an unknown verdict or a nil LLM function (the hash-floor fallback trigger).
func TestDedupLLMBackendClassify(t *testing.T) {
	ctx := context.Background()
	candidates := []string{"existing one", "existing two", "existing three"}
	cases := []struct {
		name    string
		resp    string
		want    memory.DedupVerdict
		wantID  int64
		wantErr bool
	}{
		{"add", `{"verdict":"add","candidate":0}`, memory.VerdictAdd, 0, false},
		{"noop", `{"verdict":"noop","candidate":0}`, memory.VerdictNoop, 0, false},
		{"update", `{"verdict":"update","candidate":1}`, memory.VerdictUpdate, 2, false},
		{"delete", `{"verdict":"delete","candidate":2}`, memory.VerdictDelete, 3, false},
		{"update-first-candidate", `{"verdict":"update","candidate":0}`, memory.VerdictUpdate, 1, false},
		{"update-no-candidate", `{"verdict":"update","candidate":null}`, memory.VerdictUpdate, 0, false},
		{"update-candidate-out-of-range", `{"verdict":"update","candidate":9}`, memory.VerdictUpdate, 0, false},
		{"unknown-verdict", `{"verdict":"maybe"}`, "", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() { SetDedupLLMFunc(nil) })
			SetDedupLLMFunc(func(context.Context, string, string) (string, error) { return tc.resp, nil })
			got, err := newDedupLLMBackend().Classify(ctx, "new content", candidates)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Verdict != tc.want {
				t.Fatalf("verdict = %q, want %q", got.Verdict, tc.want)
			}
			if got.CandidateID != tc.wantID {
				t.Fatalf("candidate id = %d, want %d", got.CandidateID, tc.wantID)
			}
		})
	}
}

// TestDedupLLMBackendNilFunc is TICKET-05: with no LLM function wired, Classify
// errors so the caller falls back to the deterministic hash dedup floor.
func TestDedupLLMBackendNilFunc(t *testing.T) {
	t.Cleanup(func() { SetDedupLLMFunc(nil) })
	SetDedupLLMFunc(nil)
	_, err := newDedupLLMBackend().Classify(context.Background(), "new", []string{"a"})
	if err == nil {
		t.Fatal("nil LLM function must error (hash-floor fallback trigger)")
	}
}

// TestDedupLLMBackendDedupDeprecated is TICKET-05: the deprecated binary Dedup
// path reduces the 4-way verdict (update/noop → duplicate, else not).
func TestDedupLLMBackendDedupDeprecated(t *testing.T) {
	ctx := context.Background()
	candidates := []string{"a", "b"}
	t.Cleanup(func() { SetDedupLLMFunc(nil) })

	SetDedupLLMFunc(func(context.Context, string, string) (string, error) {
		return `{"verdict":"update","candidate":1}`, nil
	})
	dup, id, err := newDedupLLMBackend().Dedup(ctx, "new", candidates)
	if err != nil || !dup || id != 2 {
		t.Fatalf("update → dup=%v id=%d err=%v, want true/2/nil", dup, id, err)
	}

	SetDedupLLMFunc(func(context.Context, string, string) (string, error) {
		return `{"verdict":"add","candidate":0}`, nil
	})
	dup, id, err = newDedupLLMBackend().Dedup(ctx, "new", candidates)
	if err != nil || dup || id != 0 {
		t.Fatalf("add → dup=%v id=%d err=%v, want false/0/nil", dup, id, err)
	}
}
