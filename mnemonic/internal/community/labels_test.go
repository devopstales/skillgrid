package community

import (
	"context"
	"fmt"
	"testing"
)

// TestLabelFallsBackWhenNoGodNode covers @step-01 (Scenario: Community label
// falls back when no god node exists): a community whose members have no
// edges (all degree-0, so no god node) is labeled "community-N" — never
// fabricated. A community with a god node is labeled from that god node's
// name (LLM-free).
func TestLabelFallsBackWhenNoGodNode(t *testing.T) {
	db := openStore(t)
	// Two isolated symbols (no edges) → a single community with no god node.
	f := seedFile(t, db, "isolated.go")
	seedSymbol(t, db, f, "lonelyA", "function", 1)
	seedSymbol(t, db, f, "lonelyB", "function", 2)

	res, err := Detect(context.Background(), db, Options{})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(res.Communities) == 0 {
		t.Fatal("expected at least one community")
	}
	// All members are degree-0 (no edges) → every community label must be the
	// "community-N" fallback, never an invented name.
	for _, c := range res.Communities {
		if c.Label == "" {
			t.Errorf("community %d has an empty label", c.ID)
		}
		want := "community-" + fmt.Sprintf("%d", c.ID)
		if c.Label != want {
			t.Errorf("no-edge community %d should be labeled %q, got %q", c.ID, want, c.Label)
		}
	}
}

// TestLabelDerivedFromGodNode covers the LLM-free label: a community with a
// god node is labeled from that god node's name (plus its path dir), never an
// API call.
func TestLabelDerivedFromGodNode(t *testing.T) {
	db := communityFixture(t)
	res, err := Detect(context.Background(), db, Options{})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	// The fixture has edges, so at least one community must carry a derived
	// (non-fallback) label that mentions one of the god-node names.
	derived := 0
	for _, c := range res.Communities {
		for _, g := range c.GodNodes {
			if contains(c.Label, g) {
				derived++
			}
		}
	}
	if derived == 0 {
		t.Errorf("expected at least one community label derived from a god-node name, got: %v", labels(res))
	}
}

func labels(r *Result) []string {
	var out []string
	for _, c := range r.Communities {
		out = append(out, c.Label)
	}
	return out
}

func contains(haystack, needle string) bool {
	if needle == "" {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
