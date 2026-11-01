package memory

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestBudget is 03.1 [RED] — the "Context-window flood" threat. A 20-hit read
// must be budgeted: (a) an item cap truncates the list (never more than the cap
// are returned), (b) the char budget truncates an in-list snippet with an
// explicit "N chars omitted" marker (never silent), and (c) the context timeout
// returns a truncated:true partial with a reason and never hangs.
//
// Scenarios: twenty-hit-search-is-budgeted,
// char-budget-truncates-with-explicit-omitted-count,
// context-timeout-returns-truncated-partial.
func TestBudget(t *testing.T) {
	fx := newFixture(t, "budget-proj")
	ctx := context.Background()
	const total = 20
	for i := 0; i < total; i++ {
		if _, err := fx.svc.Save(ctx, SaveInput{
			SessionID: session1,
			Type:      "learning",
			Title:     "Widget note " + itoaStr(i),
			Content:   "Body of widget note " + itoaStr(i) + " with enough text to exceed the char budget when repeated " + strings.Repeat("x", 2000),
		}); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}

	t.Run("item-cap-truncates-20-hits", func(t *testing.T) {
		hits, err := fx.svc.Search(ctx, "widget", "any", 20)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(hits) != total {
			t.Fatalf("expected %d raw hits, got %d", total, len(hits))
		}
		out := fx.svc.Budget().Apply(ctx, hits)
		// The item cap (default 10) must truncate the 20-hit list.
		if len(out.Hits) != defaultBudgetItems {
			t.Fatalf("expected %d budgeted hits (item cap), got %d", defaultBudgetItems, len(out.Hits))
		}
		// Every in-list result carries its full-content fetch id.
		for _, o := range out.Hits {
			if o.ID <= 0 {
				t.Fatalf("in-list result missing full-content id: %+v", o)
			}
		}
	})

	t.Run("char-budget-truncates-with-explicit-omitted", func(t *testing.T) {
		hits, err := fx.svc.Search(ctx, "widget", "any", 1)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(hits) != 1 {
			t.Fatalf("expected 1 hit, got %d", len(hits))
		}
		full := hits[0].Content
		if len(full) <= defaultBudgetChars {
			t.Fatalf("fixture content too short to exercise the char budget: %d", len(full))
		}
		out := fx.svc.Budget().Apply(ctx, hits)
		got := out.Hits[0].Content
		if len(got) >= len(full) {
			t.Fatalf("char budget did not truncate (got %d >= full %d)", len(got), len(full))
		}
		// The truncation must be explicit: an "N chars omitted" marker.
		if !strings.Contains(got, "chars omitted") {
			t.Fatalf("truncation not explicit (no 'chars omitted' marker): %q", got)
		}
	})

	t.Run("context-timeout-returns-truncated-partial-never-hangs", func(t *testing.T) {
		hits, err := fx.svc.Search(ctx, "widget", "any", 20)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		// Injectable timeout: a real short deadline on the read context (not a
		// real 3s wait). A lapsed deadline at Apply time means the read consumed
		// the whole context budget → truncated:true partial, never a hang.
		dctx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Millisecond))
		defer cancel()
		time.Sleep(2 * time.Millisecond) // let the (tiny) deadline lapse
		out := fx.svc.Budget().Apply(dctx, hits)
		if !out.Truncated {
			t.Fatalf("expected truncated=true on timeout, got %+v", out)
		}
		if out.Reason != "timeout" {
			t.Fatalf("expected reason 'timeout', got %q", out.Reason)
		}
		// A partial (not empty) is returned: the item cap still yields hits.
		if len(out.Hits) != defaultBudgetItems {
			t.Fatalf("expected a %d-hit partial, got %d", defaultBudgetItems, len(out.Hits))
		}
	})
}

// TestBudgetTimeoutEnforced is the finding-03.1 proof: the budget's context
// timeout is ENFORCED on the read (not just reported post-hoc). A genuinely
// slow read (its QueryContext returns context.DeadlineExceeded) is cut at the
// budget deadline and surfaced as a truncated:true partial with reason "timeout"
// — it does NOT run the slow read to completion. The slow read is injectable (a
// stub query that respects the context deadline), so the test is fast (no real
// wall-clock wait beyond a tiny injected delay).
func TestBudgetTimeoutEnforced(t *testing.T) {
	fx := newFixture(t, "budget-timeout-proj")
	ctx := context.Background()

	// An injectable slow read: it honors the context deadline (as a real
	// QueryContext would) and returns a partial on cancellation, taking long
	// enough that a short budget timeout always cuts it first.
	slowRead := func(bctx context.Context) ([]Observation, error) {
		select {
		case <-bctx.Done():
			// Cut by the deadline: return a partial (not the full result).
			return []Observation{
				{ID: 1, Title: "partial", Content: "partial result " + strings.Repeat("p", 200)},
			}, context.DeadlineExceeded
		case <-time.After(20 * time.Millisecond):
			// The full (slow) read would complete: this must NOT happen with a
			// short budget timeout.
			return make([]Observation, 20), nil
		}
	}

	// A short budget timeout (1ms) is well under the 20ms slow read.
	fx.svc.SetBudget(BudgetConfig{Items: 10, Chars: 1200, TimeoutNs: int64(time.Millisecond)})
	b := fx.svc.Budget()

	res, err := b.ApplyRead(ctx, slowRead)
	if err != nil {
		t.Fatalf("slow read under budget must be cut as a partial, not an error: %v", err)
	}
	// The slow read was CUT, not run to completion: a partial (1 hit) is
	// returned, not the full 20-hit result.
	if len(res.Hits) != 1 {
		t.Fatalf("expected a truncated partial (1 hit), got %d (the full slow read ran to completion)", len(res.Hits))
	}
	// It is marked truncated with reason "timeout" (the budget deadline lapsed
	// before the slow read could finish).
	if !res.Truncated {
		t.Fatalf("expected truncated=true on a deadline-cut slow read, got %+v", res)
	}
	if res.Reason != "timeout" {
		t.Fatalf("expected reason 'timeout', got %q", res.Reason)
	}

	// Control: with no timeout (a huge budget timeout), the slow read runs to
	// completion and is NOT marked truncated-by-timeout.
	fx.svc.SetBudget(BudgetConfig{Items: 10, Chars: 1200, TimeoutNs: int64(2 * time.Second)})
	res2, err := fx.svc.Budget().ApplyRead(ctx, slowRead)
	if err != nil {
		t.Fatalf("slow read with a generous timeout: %v", err)
	}
	// The full slow read completed (20 gathered), item-capped to 10 — distinct
	// from the 1-hit partial of the deadline cut.
	if len(res2.Hits) != defaultBudgetItems {
		t.Fatalf("a generous timeout must let the full slow read complete (capped to %d), got %d", defaultBudgetItems, len(res2.Hits))
	}
	if res2.Reason == "timeout" {
		t.Fatalf("a generous timeout must not mark reason 'timeout', got %+v", res2)
	}
}

func itoaStr(n int) string { return strconv.Itoa(n) }
