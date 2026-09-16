package layer

import (
	"context"
	"fmt"
	"testing"
)

// countingLLM is a stub LLM that counts its Summarise calls and returns a
// marker string so a cache hit (no second call) is observable.
type countingLLM struct {
	calls int
	scen  string
	delt  string
}

func (c *countingLLM) Summarise(ctx context.Context, summary string) (string, string, error) {
	c.calls++
	return c.scen, c.delt, nil
}

// TestDistillCacheHash is 02.5 (Scenario: distill-llm-cached-by-content-hash).
// Re-running the pass on an unchanged L0 source is a cache hit: the LLM is
// called ONCE, and the cached L2/L3 are reused. A changed L0 source misses and
// re-distills (a second LLM call).
func TestDistillCacheHash(t *testing.T) {
	fx := newDistillFixture(t, "cache", "sess-cache", distillableSummary)
	ctx := context.Background()

	llm := &countingLLM{scen: "scenario-A", delt: "delta-A"}
	res1, err := Distill(ctx, fx.svc, fx.sessionID, DistillOptions{LLM: llm})
	if err != nil {
		t.Fatalf("first distill: %v", err)
	}
	if llm.calls != 1 {
		t.Fatalf("first run must call the LLM once, got %d", llm.calls)
	}
	if !res1.LLMUsed {
		t.Errorf("first run must use the LLM")
	}

	// Re-run on the UNCHANGED source: a cache hit, no second LLM call.
	res2, err := Distill(ctx, fx.svc, fx.sessionID, DistillOptions{LLM: llm})
	if err != nil {
		t.Fatalf("second distill: %v", err)
	}
	if llm.calls != 1 {
		t.Fatalf("unchanged source must be a cache hit (LLM called %d times, want 1)", llm.calls)
	}
	if res2.LLMUsed {
		t.Errorf("unchanged re-run must reuse the cache, not re-distill (llmUsed=%v)", res2.LLMUsed)
	}
	// The cached scenario is surfaced via Inspect.
	chain, err := Inspect(ctx, fx.svc, fx.sessionID)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if len(chain.Scenarios) == 0 {
		t.Fatal("no scenario in chain")
	}
	if chain.Scenarios[0].Content != "scenario-A" {
		t.Errorf("cached scenario not reused: got %q", chain.Scenarios[0].Content)
	}

	// Change the L0 source: a cache miss, a second LLM call.
	changed := "## Key Learnings:\n\n1. a totally different learning about caching\n2. another distinct learning here\n"
	if _, err := fx.st.DB.Exec(`
		UPDATE sessions SET summary = ? WHERE id = ? AND project = ?`,
		changed, fx.sessionID, fx.projectID); err != nil {
		t.Fatalf("change summary: %v", err)
	}
	res3, err := Distill(ctx, fx.svc, fx.sessionID, DistillOptions{LLM: llm})
	if err != nil {
		t.Fatalf("third distill: %v", err)
	}
	if llm.calls != 2 {
		t.Fatalf("changed source must re-distill (LLM called %d times, want 2)", llm.calls)
	}
	if !res3.LLMUsed {
		t.Errorf("changed source must re-distill (llmUsed=false)")
	}
}

// TestDistillNoOp is 02.7 (Scenario: session-with-no-l1-content-is-noop). A
// session with no distillable content is a no-op: no empty atoms, scenarios, or
// persona delta are fabricated.
func TestDistillNoOp(t *testing.T) {
	fx := newDistillFixture(t, "noop", "sess-noop", "A short session with nothing extractable here.")
	ctx := context.Background()

	beforeL1 := countRows(t, fx.st, fx.projectID, "observation_layers", `layer = 'L1'`)
	beforeP := countRows(t, fx.st, fx.projectID, "personas", "")

	res, err := Distill(ctx, fx.svc, fx.sessionID, DistillOptions{})
	if err != nil {
		t.Fatalf("distill: %v", err)
	}
	if res.Created {
		t.Fatalf("no L1-able content must be a no-op, got created=%v", res.Created)
	}
	if res.L1Count != 0 || res.ScenarioCount != 0 || res.PersonaDeltaCount != 0 {
		t.Errorf("no-op fabricated layers: L1=%d L2=%d L3=%d",
			res.L1Count, res.ScenarioCount, res.PersonaDeltaCount)
	}
	if afterL1 := countRows(t, fx.st, fx.projectID, "observation_layers", `layer = 'L1'`); afterL1 != beforeL1 {
		t.Fatalf("L1 rows grew: before=%d after=%d", beforeL1, afterL1)
	}
	if afterP := countRows(t, fx.st, fx.projectID, "personas", ""); afterP != beforeP {
		t.Fatalf("persona rows grew: before=%d after=%d", beforeP, afterP)
	}
}

// TestInspectChain is 02.1 (Scenario: mem-layers-inspects-l0-to-l3-chain).
// Inspect returns the L0→L1→L2→L3 chain with each layer's provenance link to
// its L0 source.
func TestInspectChain(t *testing.T) {
	fx := newDistillFixture(t, "chain", "sess-chain", distillableSummary)
	ctx := context.Background()

	if _, err := Distill(ctx, fx.svc, fx.sessionID, DistillOptions{}); err != nil {
		t.Fatalf("distill: %v", err)
	}

	chain, err := Inspect(ctx, fx.svc, fx.sessionID)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if chain.L0Session != fx.sessionID {
		t.Fatalf("L0 session = %q, want %q", chain.L0Session, fx.sessionID)
	}
	if len(chain.Atoms) == 0 {
		t.Fatal("chain has no L1 atoms")
	}
	if len(chain.Scenarios) == 0 {
		t.Fatal("chain has no L2 scenario")
	}
	if len(chain.Persona) == 0 {
		t.Fatal("chain has no L3 persona delta")
	}
	// Every layer carries a resolvable provenance link (source_session matches
	// the L0 and a non-empty source topic).
	for _, l := range append(append(chain.Atoms, chain.Scenarios...), chain.Persona...) {
		if l.SourceSession != fx.sessionID {
			t.Errorf("%s layer source_session = %q, want %q", l.Layer, l.SourceSession, fx.sessionID)
		}
		if l.SourceTopic == "" {
			t.Errorf("%s layer has an empty source topic (orphan)", l.Layer)
		}
	}
}

// TestInspectByTopic is 02.9 (mem_layers <topic_key>): the chain can also be
// reached by a topic that appears in the distilled L0 source.
func TestInspectByTopic(t *testing.T) {
	fx := newDistillFixture(t, "btopic", "sess-btopic", distillableSummary)
	ctx := context.Background()
	if _, err := Distill(ctx, fx.svc, fx.sessionID, DistillOptions{}); err != nil {
		t.Fatalf("distill: %v", err)
	}
	// "rot" appears in the distilled source topic ("atomic rotation to avoid races").
	chain, err := InspectByTopic(ctx, fx.svc, "rot")
	if err != nil {
		t.Fatalf("inspect by topic: %v", err)
	}
	if chain.L0Session != fx.sessionID {
		t.Fatalf("by-topic L0 session = %q, want %q", chain.L0Session, fx.sessionID)
	}
	if len(chain.Atoms) == 0 {
		t.Fatal("by-topic chain has no L1 atoms")
	}
}

// TestInspectNoLayers returns a chain with only the L0 for a session that has
// never been distilled — no layers are invented.
func TestInspectNoLayers(t *testing.T) {
	fx := newDistillFixture(t, "nolayers", "sess-nolayers", "a summary that was never distilled")
	ctx := context.Background()
	chain, err := Inspect(ctx, fx.svc, fx.sessionID)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if chain.L0Session != fx.sessionID {
		t.Fatalf("L0 session = %q, want %q", chain.L0Session, fx.sessionID)
	}
	if len(chain.Atoms)+len(chain.Scenarios)+len(chain.Persona) != 0 {
		t.Errorf("never-distilled session must have no layers, got atoms=%d scen=%d persona=%d",
			len(chain.Atoms), len(chain.Scenarios), len(chain.Persona))
	}
}

var _ = fmt.Sprintf
