package mcp

import (
	"context"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

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
