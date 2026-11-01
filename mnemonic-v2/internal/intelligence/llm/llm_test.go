package llm_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/degrade"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/llm"
)

// [llm-seam-4-way-classify] — the OpenAI-compatible provider must parse the
// strict 4-way JSON verdict into the typed DedupDecision, mapping the 0-based
// candidate index to a 1-based id (v1 service/dedup_llm.go:98-100).
func TestClassifyReturnsFourWayVerdict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/chat/completions") {
			fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"verdict\":\"update\",\"candidate\":1}"}}]}`)
			return
		}
		t.Fatalf("unexpected path %s", r.URL.Path)
	}))
	defer srv.Close()
	c, err := llm.NewClient(srv.URL, "test-model", "k", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	d, err := c.Classify(context.Background(), "new text", []string{"old one", "old two"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Verdict != llm.VerdictUpdate || d.CandidateID != 2 { // 0-based index 1 → 1-based id 2
		t.Fatalf("bad decision %+v", d)
	}
	if d.Reason != "llm" {
		t.Fatalf("reason must be llm, got %q", d.Reason)
	}
}

// [degraded-explicit] — a nil (absent) LLM seam runs the deterministic floor
// and the result is flagged degraded: never a silent floor (INV-3).
func TestDegradeNilSeamRunsFloorExplicitly(t *testing.T) {
	var l llm.LLM // nil
	var ranFloor bool
	res, degraded := degrade.Run(l, func() any {
		ranFloor = true
		return llm.DedupDecision{Verdict: llm.VerdictAdd, Reason: "hash"}
	}, "classify")
	d, ok := res.(llm.DedupDecision)
	if !ok {
		t.Fatalf("floor result type = %T, want llm.DedupDecision", res)
	}
	if !ranFloor || !degraded || d.Verdict != llm.VerdictAdd {
		t.Fatalf("floor must run, flagged degraded: ranFloor=%v degraded=%v d=%+v", ranFloor, degraded, d)
	}
}

// [degraded-bounded-retry] — a present seam that fails on every call gets
// exactly one bounded retry, then the floor runs and the result is degraded.
func TestDegradeErroredSeamRetriesOnceThenFloor(t *testing.T) {
	calls := 0
	// ponytail: the brief's snippet passes a struct value to a generic
	// parameter; interface-typed seams (a nil llm.LLM) can only be passed via
	// an interface value, so degrade.Run takes the call as an interface.
	seam := failingSeam{fn: func() (llm.DedupDecision, error) {
		calls++
		return llm.DedupDecision{}, errors.New("boom")
	}}
	res, degraded := degrade.Run(seam, func() any {
		return llm.DedupDecision{Verdict: llm.VerdictAdd, Reason: "hash"}
	}, "classify")
	if _, ok := res.(llm.DedupDecision); !ok {
		t.Fatalf("floor result type = %T, want llm.DedupDecision", res)
	}
	if calls != 2 {
		t.Fatalf("expected exactly one retry (2 calls), got %d", calls)
	}
	if !degraded {
		t.Fatalf("must be degraded")
	}
}

// failingSeam is a seam that errors on every call. It models the *failing*
// state (present but erroring) as opposed to the *absent* state (nil), which
// is a different state per INV-3 (no retry for absent).
type failingSeam struct {
	fn func() (llm.DedupDecision, error)
}

func (s failingSeam) Call(ctx context.Context, a, b any) (any, error) { return s.fn() }
