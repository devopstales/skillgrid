package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

type memInjectOut struct {
	Block       string `json:"block"`
	TotalTokens int    `json:"total_tokens"`
	Degraded    bool   `json:"degraded"`
	Items       []struct {
		Project   string `json:"project"`
		TokenCost int    `json:"token_cost"`
	} `json:"items"`
}

// seedInjectObs inserts one FTS-indexable observation into the project bucket
// of dataDir via the real save path; the FTS auto-sync trigger indexes it.
// Mirrors the seeding pattern in session_inject/retrieve_test.go (observations
// row + FTS trigger) through the service facade.
func seedInjectObs(t *testing.T, dataDir, project, title, content string) {
	t.Helper()
	s := service.New(dataDir)
	h, cleanup, err := s.Open(project)
	if err != nil {
		t.Fatalf("open project: %v", err)
	}
	defer cleanup()
	mem := h.Memory()
	sid, err := mem.SessionStart(context.Background(), t.TempDir(), "inject")
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if _, err := mem.Save(context.Background(), memory.SaveInput{
		Title:     title,
		Type:      "learning",
		Content:   content,
		Scope:     "project",
		SessionID: sid,
	}); err != nil {
		t.Fatalf("seed observation: %v", err)
	}
}

func callMemInject(t *testing.T, args map[string]any) memInjectOut {
	t.Helper()
	res, err := HandleMemInjectSessionForTest(context.Background(), newCallTool("mem_inject_session", args))
	if err != nil {
		t.Fatalf("handleMemInjectSession dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_inject_session errored: %s", callResultText(t, res))
	}
	var out memInjectOut
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("unmarshal mem_inject_session: %v (text %s)", err, callResultText(t, res))
	}
	return out
}

// TestMemInjectSession_ReturnsRankedItems is the happy path (T5 scenario
// "mem_inject_session returns ranked items"): seeded "auth" observations come
// back as a non-empty block, an items array, every item token_cost > 0, and
// total_tokens equal to the sum of item costs.
func TestMemInjectSession_ReturnsRankedItems(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "inject-ranked")
	t.Setenv("MNEMONIC_EMBED", "")

	seedInjectObs(t, dataDir, "inject-ranked", "auth note 1",
		"authentication token rotation detail one")
	seedInjectObs(t, dataDir, "inject-ranked", "auth note 2",
		"authentication token rotation detail two")

	out := callMemInject(t, map[string]any{"query": "auth"})
	if out.Block == "" {
		t.Fatalf("expected non-empty block, got empty")
	}
	if len(out.Items) == 0 {
		t.Fatalf("expected items, got none")
	}
	sum := 0
	for i, it := range out.Items {
		if it.TokenCost <= 0 {
			t.Errorf("item[%d] token_cost = %d, want > 0", i, it.TokenCost)
		}
		sum += it.TokenCost
	}
	if out.TotalTokens != sum {
		t.Errorf("total_tokens = %d, want %d (sum of item costs)", out.TotalTokens, sum)
	}
}

// TestMemInjectSession_AllProjects is the permissive all-projects scenario:
// observations seeded in two project buckets; all_projects=true may return
// items spanning both.
func TestMemInjectSession_AllProjects(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "inject-all-a")
	t.Setenv("MNEMONIC_EMBED", "")

	seedInjectObs(t, dataDir, "inject-all-a", "auth note a",
		"authentication token rotation detail a")
	seedInjectObs(t, dataDir, "inject-all-b", "auth note b",
		"authentication token rotation detail b")

	out := callMemInject(t, map[string]any{"query": "auth", "all_projects": true})
	if len(out.Items) == 0 {
		t.Fatalf("expected items, got none")
	}
	seen := map[string]bool{}
	for _, it := range out.Items {
		if it.Project != "inject-all-a" && it.Project != "inject-all-b" {
			t.Errorf("item from unexpected project %q", it.Project)
		}
		seen[it.Project] = true
	}
	for _, p := range []string{"inject-all-a", "inject-all-b"} {
		if !seen[p] {
			t.Errorf("expected items from both projects, saw %v", seen)
		}
	}
}

// TestMemInjectSession_Degraded is the no-embedder scenario: with
// MNEMONIC_EMBED unset and no embedder attached, retrieval degrades to
// BM25-only (degraded=true) and still returns items.
func TestMemInjectSession_Degraded(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "inject-degraded")
	t.Setenv("MNEMONIC_EMBED", "")

	seedInjectObs(t, dataDir, "inject-degraded", "auth note d",
		"authentication token rotation detail d")

	out := callMemInject(t, map[string]any{"query": "auth"})
	if !out.Degraded {
		t.Errorf("expected degraded=true with no embedder, got false")
	}
	if len(out.Items) == 0 {
		t.Fatalf("expected items from the FTS floor, got none")
	}
}

// TestMemInjectSession_Registered asserts the tool is registered and
// invocable through the registered-handler path: the same handler instance
// the MCP server dispatches (HandleMemInjectSessionForTest, the alias of
// handleMemInjectSession registered via s.AddTool) returns a result with a
// block field for a seeded query, and a missing query is a validation error
// — a name-only GetTool would not prove either.
func TestMemInjectSession_Registered(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "inject-registered")
	t.Setenv("MNEMONIC_EMBED", "")

	seedInjectObs(t, dataDir, "inject-registered", "auth note r",
		"authentication token rotation detail r")

	out := callMemInject(t, map[string]any{"query": "auth"})
	if out.Block == "" {
		t.Fatalf("registered handler returned no block: %s", fmt.Sprint(out))
	}

	res, err := HandleMemInjectSessionForTest(context.Background(),
		newCallTool("mem_inject_session", map[string]any{}))
	if err != nil {
		t.Fatalf("dispatch without query: %v", err)
	}
	if !res.IsError {
		t.Errorf("missing query should be a validation error, got: %s", callResultText(t, res))
	}
}
