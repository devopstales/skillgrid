package context_harness

import (
	"strings"
	"testing"
)

func TestRenderRouting(t *testing.T) {
	got := RenderRouting()
	if !strings.HasPrefix(got, "## Context Routing") {
		t.Fatalf("expected header ## Context Routing, got:\n%s", got)
	}
	if len(got) > 600 {
		t.Fatalf("routing block must be <= 600 chars, got %d", len(got))
	}
	for _, want := range []string{"ctx_query", "ctx_search", "ctx index", "mem_*"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
}
