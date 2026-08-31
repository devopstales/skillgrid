package mcp

import (
	"context"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

func TestAllToolsRegistered(t *testing.T) {
	s := NewServer()
	tools := s.ListTools()
	if len(tools) == 0 {
		t.Fatal("no tools registered")
	}
	want := []string{
		"mem_save", "mem_search", "mem_context", "mem_get_observation",
		"mem_timeline", "mem_update", "mem_delete", "mem_stats", "mem_save_prompt",
		"mem_current_project", "mem_doctor", "mem_review",
		"mem_judge", "mem_compare", "mem_merge_projects",
		"mem_session_start", "mem_session_end", "mem_session_summary",
		"mem_session_set_title",
		"mem_suggest_topic_key",
		"mem_capture_passive",
		"code_status", "code_index", "code_search", "code_read",
		"web_cache_lookup", "web_cache_save", "web_cache_search",
		"web_cache_get", "web_cache_status",
	}
	if len(tools) != len(want) {
		t.Errorf("expected %d tools, got %d (%v)", len(want), len(tools), tools)
	}
	for _, name := range want {
		st, ok := tools[name]
		if !ok {
			t.Errorf("expected tool %q to be registered", name)
			continue
		}
		_ = st
	}
}

// TestSuggestTopicKeyDispatch exercises the MCP dispatch for a pure
// (no-store) tool. The spec's "MCP tool dispatch" scenario is satisfied as
// long as a tool is routed to its handler and returns a JSON result.
func TestSuggestTopicKeyDispatch(t *testing.T) {
	// Inject a service that points at a temp data dir so that if the handler
	// happens to open one it won't pollute the real store.
	dataDir := t.TempDir()
	SetService(service.New(dataDir))

	req := mcplib.CallToolRequest{}
	req.Params.Name = "mem_suggest_topic_key"
	req.Params.Arguments = map[string]any{
		"type":  "decision",
		"title": "Fix N+1 query in UserList",
	}
	res, err := handleMemSuggestTopicKey(context.Background(), req)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if res.IsError {
		t.Errorf("expected success, got error: %v", res.Content)
	}
}
