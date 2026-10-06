package mcp

import (
	"context"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// TestMemLifecycleDispatch exercises the TICKET-05 action dispatch: every
// action routes to its handler and returns a JSON result (or a value error
// for missing required params). No real store mutations are asserted — the
// goal is that the dispatch never throws and never claims "not implemented".
func TestMemLifecycleDispatch(t *testing.T) {
	SetService(service.New(t.TempDir()))

	// health: no required params beyond action
	res, err := handleMemLifecycle(context.Background(), newCallTool("mem_lifecycle", map[string]any{
		"action":  "health",
		"project": "test-project",
	}))
	if err != nil {
		t.Fatalf("health dispatch: %v", err)
	}
	if res.IsError {
		t.Errorf("health returned error: %s", callResultText(t, res))
	}

	// dedup_scan: read-only, no mutations
	res, err = handleMemLifecycle(context.Background(), newCallTool("mem_lifecycle", map[string]any{
		"action":    "dedup_scan",
		"project":   "test-project",
		"dry_run":   true,
	}))
	if err != nil {
		t.Fatalf("dedup_scan dispatch: %v", err)
	}
	if res.IsError {
		t.Errorf("dedup_scan returned error: %s", callResultText(t, res))
	}

	// dedup_merge: missing obs_ids → value error
	res, err = handleMemLifecycle(context.Background(), newCallTool("mem_lifecycle", map[string]any{
		"action":  "dedup_merge",
		"project": "test-project",
	}))
	if err != nil {
		t.Fatalf("dedup_merge dispatch: %v", err)
	}
	if !res.IsError {
		t.Error("dedup_merge without obs_ids should be a value error")
	}

	// consolidate: missing obs_ids → value error
	res, err = handleMemLifecycle(context.Background(), newCallTool("mem_lifecycle", map[string]any{
		"action":  "consolidate",
		"project": "test-project",
	}))
	if err != nil {
		t.Fatalf("consolidate dispatch: %v", err)
	}
	if !res.IsError {
		t.Error("consolidate without obs_ids should be a value error")
	}

	// archive: missing subaction → value error
	res, err = handleMemLifecycle(context.Background(), newCallTool("mem_lifecycle", map[string]any{
		"action":  "archive",
		"project": "test-project",
	}))
	if err != nil {
		t.Fatalf("archive dispatch: %v", err)
	}
	if !res.IsError {
		t.Error("archive without subaction should be a value error")
	}

	// unknown action → value error
	res, err = handleMemLifecycle(context.Background(), newCallTool("mem_lifecycle", map[string]any{
		"action":  "bogus",
		"project": "test-project",
	}))
	if err != nil {
		t.Fatalf("unknown action dispatch: %v", err)
	}
	if !res.IsError {
		t.Error("unknown action should be a value error")
	}
}

// TestMemAskModeDispatch exercises the TICKET-02 routing: mode is validated
// (invalid → value error), and cited vs llm both dispatch to the handler
// without throwing. The llm call fails open to the cited floor (no LLM
// attached in the test process), so it must return success, not error.
func TestMemAskModeDispatch(t *testing.T) {
	SetService(service.New(t.TempDir()))

	mk := func(mode string) mcplib.CallToolRequest {
		req := mcplib.CallToolRequest{}
		req.Params.Name = "mem_ask"
		req.Params.Arguments = map[string]any{
			"query":     "auth",
			"mode":      mode,
			"project":   "test-project",
			"all_projects": false,
		}
		return req
	}

	res, err := handleMemAsk(context.Background(), mk("invalid"))
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if !res.IsError {
		t.Error("invalid mode should be a value error")
	}

	for _, mode := range []string{"cited", "llm"} {
		res, err := handleMemAsk(context.Background(), mk(mode))
		if err != nil {
			t.Fatalf("dispatch %s: %v", mode, err)
		}
		if res.IsError {
			t.Errorf("mode %s returned an error: %v", mode, res.Content)
		}
	}
}
