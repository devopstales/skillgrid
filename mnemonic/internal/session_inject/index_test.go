package session_inject

import (
	"fmt"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
)

func TestRenderIndex_empty(t *testing.T) {
	if got := RenderIndex(nil, nil, IndexConfig{MaxTokens: 800}); got != "" {
		t.Fatalf("empty inputs: got %q", got)
	}
}

func TestRenderIndex_pinnedFirstAndDates(t *testing.T) {
	summaries := []memory.Session{{
		ID:        "sess-abcdef12",
		StartedAt: "2026-10-01T12:00:00Z",
		Summary:   "Shipped the index. More work later.",
	}}
	obs := []memory.Observation{
		{ID: 1, Type: "discovery", Title: "older unpinned", CreatedAt: "2026-09-28T10:00:00Z", Pinned: false},
		{ID: 2, Type: "decision", Title: "pinned item", CreatedAt: "2026-09-27T10:00:00Z", Pinned: true},
		{ID: 3, Type: "pattern", Title: "newest unpinned", CreatedAt: "2026-10-02T10:00:00Z", Pinned: false},
	}
	got := RenderIndex(summaries, obs, IndexConfig{Summaries: 5, Observations: 20, MaxTokens: 800})

	if !strings.Contains(got, "sess-abc 2026-10-01: Shipped the index.") {
		t.Errorf("summary line missing or wrong:\n%s", got)
	}
	idx2 := strings.Index(got, "#2 decision pinned item")
	idx3 := strings.Index(got, "#3 pattern newest unpinned")
	idx1 := strings.Index(got, "#1 discovery older unpinned")
	if idx2 < 0 || idx3 < 0 || idx1 < 0 {
		t.Fatalf("missing observation lines:\n%s", got)
	}
	if idx2 > idx3 || idx3 > idx1 {
		t.Errorf("want pinned first then newest; order in output:\n%s", got)
	}
	if !strings.Contains(got, "2026-09-27") || !strings.Contains(got, "2026-10-02") {
		t.Errorf("dates missing:\n%s", got)
	}
	for _, tok := range []string{"mem_get_observation", "mem_timeline", "mem_search"} {
		if !strings.Contains(got, tok) {
			t.Errorf("footer missing %q:\n%s", tok, got)
		}
	}
}

func TestRenderIndex_tokenCapKeepsNewest(t *testing.T) {
	var obs []memory.Observation
	for i := 1; i <= 60; i++ {
		obs = append(obs, memory.Observation{
			ID:        int64(i),
			Type:      "learning",
			Title:     strings.Repeat("x", 24) + " " + strings.Repeat("y", 24),
			CreatedAt: fmt.Sprintf("2026-%02d-%02dT12:00:00Z", 9+(i/28), (i%28)+1),
		})
	}
	got := RenderIndex(nil, obs, IndexConfig{Observations: 60, MaxTokens: 200})
	if got == "" {
		t.Fatal("expected non-empty index")
	}
	if EstimateTokens(got) > 200 {
		t.Fatalf("over token cap (%d):\n%s", EstimateTokens(got), got)
	}
	if !strings.Contains(got, "#60 learning") {
		t.Errorf("newest observation #60 should be retained:\n%s", got)
	}
	if !strings.Contains(got, "omitted") {
		t.Errorf("expected omitted count:\n%s", got)
	}
}
