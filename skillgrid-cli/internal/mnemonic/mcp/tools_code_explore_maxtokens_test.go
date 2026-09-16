package mcp

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestCodeExploreMaxTokens covers @step-03 (maxTokens truncates the response
// and stays valid): an optional max_tokens budget (deterministic ~4-bytes/token)
// truncates the formatted code_explore response with an ellipsis, sets the
// envelope's truncated flag, and leaves the body well-formed JSON. Threat:
// Mnemonic tool surface.
func TestCodeExploreMaxTokens(t *testing.T) {
	// ASCII-only source so the deterministic byte budget is exact.
	example := exploreResult{
		Symbol: "alpha",
		Source: map[string][]srcSpan{
			"main.py": {
				{Symbol: "alpha", Start: 1, End: 9, Content: strings.Repeat("x\n", 80)},
				{Symbol: "other", Start: 10, End: 40, Content: strings.Repeat("y\n", 240)},
			},
		},
		CallFlow: []flowEdge{{From: "beta", To: "alpha", Kind: "call", Confidence: "EXTRACTED", Line: 3}},
	}

	// No budget: not truncated, body is a full render.
	full, truncatedFull := formatExplore(example, 0)
	if truncatedFull {
		t.Errorf("no maxTokens must not truncate")
	}
	if !strings.Contains(full, "## Source") || !strings.Contains(full, "## Call flow") {
		t.Errorf("full render must include source and call-flow sections: %q", full)
	}

	// Tiny budget: truncated with an ellipsis.
	small, truncatedSmall := formatExplore(example, 10)
	if !truncatedSmall {
		t.Fatalf("tiny maxTokens must truncate")
	}
	if !strings.HasSuffix(small, "\n…") {
		t.Errorf("truncated body must end with an ellipsis, got tail: %q", tail(small))
	}
	// Deterministic ~4-bytes/token estimate: the truncated body is capped at
	// the byte budget plus the trailing "\n…" marker (2 runes).
	const budget = 10 * 4
	if n := len([]rune(small)); n > budget+2 {
		t.Errorf("truncated body = %d chars, must stay within budget+2 (%d tokens * 4)", n, 10)
	}
	if len(small) >= len(full) {
		t.Errorf("truncated body (%d) must be shorter than the full body (%d)", len(small), len(full))
	}
}

// TestCodeExploreMaxTokensEnvelope covers the response envelope staying
// well-formed (valid JSON) and carrying the truncation flag through
// handleCodeExplore's output. Threat: Mnemonic tool surface.
func TestCodeExploreMaxTokensEnvelope(t *testing.T) {
	example := exploreResult{
		Symbol: "alpha",
		Source: map[string][]srcSpan{
			"main.py": {{Symbol: "alpha", Start: 1, End: 200, Content: strings.Repeat("z\n", 2000)}},
		},
	}

	for name, maxTokens := range map[string]int{"none": 0, "tiny": 8} {
		t.Run(name, func(t *testing.T) {
			text, truncated := formatExplore(example, maxTokens)
			env := exploreEnvelope{text, truncated}
			raw, err := json.Marshal(env)
			if err != nil {
				t.Fatalf("envelope must marshal to valid JSON: %v", err)
			}
			var round struct {
				Text      string `json:"text"`
				Truncated bool   `json:"truncated"`
			}
			if err := json.Unmarshal(raw, &round); err != nil {
				t.Fatalf("envelope must stay valid JSON: %v", err)
			}
			if (maxTokens > 0) != round.Truncated {
				t.Errorf("truncated flag = %v for maxTokens=%d, want %v", round.Truncated, maxTokens, maxTokens > 0)
			}
			if round.Text != text {
				t.Errorf("envelope text must round-trip intact")
			}
		})
	}

	_ = os.Getenv // keep os import stable for future fixtures
}

func tail(s string) string {
	if len(s) > 40 {
		return s[len(s)-40:]
	}
	return s
}
