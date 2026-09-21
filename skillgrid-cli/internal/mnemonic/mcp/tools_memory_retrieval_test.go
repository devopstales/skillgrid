package mcp

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// memRetrievalFixture pins the project to a stable bucket so the read handlers
// resolve to one store.
func memRetrievalFixture(t *testing.T) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("MNEMONIC_PROJECT", "retrievmcp-probe")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
}

// startMCPSession drives mem_session_start and returns the session id.
func startMCPSession(t *testing.T) string {
	t.Helper()
	startRes, err := handleMemSessionStart(context.Background(), newCallTool("mem_session_start", map[string]any{}))
	if err != nil {
		t.Fatalf("handleMemSessionStart dispatch: %v", err)
	}
	if startRes.IsError {
		t.Fatalf("mem_session_start errored: %s", callResultText(t, startRes))
	}
	var out struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, startRes)), &out); err != nil {
		t.Fatalf("unmarshal session start: %v", err)
	}
	return out.SessionID
}

// TestBudgetedRetrievalMCP is the 03.2 surface proof at the tool boundary:
// mem_search returns budgeted (char-truncated, id-carrying) in-list results,
// mem_get_observation is the ONLY full-content path, and the 005 mem_* tools
// keep their names + required params while bad retrieval args are rejected.
//
// Scenarios: layered-retrieval-l2-l3-first-with-l1-l0-rrf-fallback,
// mem-get-observation-is-only-full-content-path, budgeted-reads-005-stable,
// bad-retrieval-args-rejected.
func TestBudgetedRetrievalMCP(t *testing.T) {
	memRetrievalFixture(t)
	ctx := context.Background()
	sid := startMCPSession(t)

	// Save 12 observations with long bodies (well over the default 1200-char
	// budget) so the in-list read must truncate.
	body := "Budget retrieval zebraquilt body " + strings.Repeat("b", 1500)
	for i := 0; i < 12; i++ {
		saveRes, err := handleMemSave(ctx, newCallTool("mem_save", map[string]any{
			"title":      "Retrieval zebraquilt probe " + itoaMCP(i),
			"type":       "learning",
			"content":    body,
			"session_id": sid,
		}))
		if err != nil {
			t.Fatalf("mem_save %d dispatch: %v", i, err)
		}
		if saveRes.IsError {
			t.Fatalf("mem_save %d errored: %s", i, callResultText(t, saveRes))
		}
	}

	// 005 mem_* tools keep their names + required params unchanged (budgeted
	// behavior is additive).
	tools := NewServer().ListTools()
	if len(tools) != 80 {
		t.Fatalf("expected 80 tools, got %d", len(tools))
	}
	for name, wantRequired := range expectedMemToolSurface {
		st, ok := tools[name]
		if !ok {
			t.Errorf("005 mem tool %q is no longer registered", name)
			continue
		}
		got := append([]string(nil), st.Tool.InputSchema.Required...)
		want := append([]string(nil), wantRequired...)
		sort.Strings(got)
		sort.Strings(want)
		if len(got) != len(want) || !equalStrings(got, want) {
			t.Errorf("mem tool %q required params changed: got %v want %v", name, got, want)
		}
	}

	// mem_search: in-list results are char-truncated (explicit "N chars
	// omitted") and every one carries its full-content fetch id.
	// Read as the session owner so the per-owner visibility filter (013 step 01)
	// returns the private observations.
	searchRes, err := handleMemSearch(ctx, newCallTool("mem_search", map[string]any{
		"query": "zebraquilt", "reader_owner": sid,
	}))
	if err != nil {
		t.Fatalf("handleMemSearch dispatch: %v", err)
	}
	if searchRes.IsError {
		t.Fatalf("mem_search errored: %s", callResultText(t, searchRes))
	}
	var searchOut struct {
		Observations []struct {
			ID      int64  `json:"id"`
			Content string `json:"content"`
		} `json:"observations"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, searchRes)), &searchOut); err != nil {
		t.Fatalf("unmarshal mem_search: %v", err)
	}
	if len(searchOut.Observations) == 0 {
		t.Fatal("mem_search returned no observations")
	}
	var firstID int64
	for _, o := range searchOut.Observations {
		if o.ID <= 0 {
			t.Fatalf("in-list result missing full-content fetch id: %+v", o)
		}
		// The char budget truncates the in-list snippet explicitly.
		if !strings.Contains(o.Content, "chars omitted") {
			t.Fatalf("mem_search in-list content not char-budgeted: %q", o.Content)
		}
		firstID = o.ID
	}

	// mem_get_observation (the ONLY full-content path) returns the full,
	// untruncated content for the fetched id.
	getRes, err := handleMemGetObservation(ctx, newCallTool("mem_get_observation", map[string]any{
		"id": float64(firstID), "reader_owner": sid,
	}))
	if err != nil {
		t.Fatalf("handleMemGetObservation dispatch: %v", err)
	}
	if getRes.IsError {
		t.Fatalf("mem_get_observation errored: %s", callResultText(t, getRes))
	}
	var getOut struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, getRes)), &getOut); err != nil {
		t.Fatalf("unmarshal mem_get_observation: %v", err)
	}
	if strings.Contains(getOut.Content, "chars omitted") {
		t.Fatalf("mem_get_observation must be the only FULL-content path (untruncated), got: %q", getOut.Content)
	}
	if len(getOut.Content) <= len(searchOut.Observations[0].Content) {
		t.Fatalf("full content (%d) must be longer than the budgeted in-list snippet (%d)", len(getOut.Content), len(searchOut.Observations[0].Content))
	}

	// Bad retrieval args are rejected clearly: mem_search with no query is a
	// validation error, not a hang or a fabricated result.
	badRes, _ := handleMemSearch(ctx, newCallTool("mem_search", map[string]any{}))
	if !badRes.IsError {
		t.Fatalf("mem_search without a query must be a validation error, got: %s", callResultText(t, badRes))
	}
}

func itoaMCP(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
