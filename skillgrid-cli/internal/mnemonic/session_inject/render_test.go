package session_inject

import (
	"strings"
	"testing"
)

func containsAll(s string, subs ...string) []string {
	var missing []string
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			missing = append(missing, sub)
		}
	}
	return missing
}

func TestRenderContextBlock_Format(t *testing.T) {
	res := &RetrieveResult{
		Items: []InjectItem{
			{ID: 1, Source: "observation", Project: "p", Title: "Fixed login race", Snippet: "snippet one", TokenCost: 50},
			{ID: 2, Source: "observation", Project: "p", Title: "Auth model decision", Snippet: "snippet two", TokenCost: 80},
		},
		Degraded:    false,
		TotalTokens: 130,
	}
	out := RenderContextBlock(res, "p")
	if missing := containsAll(out, "50 tokens", "80 tokens", "130 tokens total", "Fixed login race", "Auth model decision"); len(missing) > 0 {
		t.Fatalf("missing substrings: %v\nblock:\n%s", missing, out)
	}
}

func TestRenderContextBlock_Degraded(t *testing.T) {
	res := &RetrieveResult{
		Items:       []InjectItem{{ID: 1, Source: "observation", Project: "p", Title: "T", Snippet: "s", TokenCost: 10}},
		Degraded:    true,
		TotalTokens: 10,
	}
	out := RenderContextBlock(res, "p")
	if !strings.Contains(out, "BM25-only") {
		t.Fatalf("missing BM25-only marker\nblock:\n%s", out)
	}
}

func TestRenderContextBlock_Empty(t *testing.T) {
	if got := RenderContextBlock(nil, "p"); got != "" {
		t.Fatalf("nil result: expected empty string, got %q", got)
	}
	if got := RenderContextBlock(&RetrieveResult{}, "p"); got != "" {
		t.Fatalf("empty result: expected empty string, got %q", got)
	}
}
