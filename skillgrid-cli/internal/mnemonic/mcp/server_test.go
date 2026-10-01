package mcp

import (
	"context"
	"encoding/json"
	"strings"
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
		"mem_pin", "mem_unpin", "mem_unify",
		"code_status", "code_index", "code_search", "code_read",
		"code_orient", "code_signature", "code_file_toc", "code_rationale",
		"code_grep",
		// Step 03 graph + composite tools.
		"code_explore", "code_impact", "code_path", "code_explain",
		"code_get_callers", "code_get_callees", "code_get_dependents",
		"code_get_implementors", "code_get_hierarchy", "code_get_tests_for",
		// Step 04 hybrid search tools.
		"code_hybrid_search", "code_semantic_search", "code_embedding_status",
		// Community detection tools.
		"code_communities", "code_god_nodes", "code_explain_community",
		// Framework route / navigation query tools.
		"code_route", "code_navigates",
		// Unresolved-ref drop log query tool (034).
		"code_unresolved_refs",
		// Precomputed process-flow tools.
		"code_processes", "code_process",
		// Knowledge-graph query tools.
		"code_docs", "code_configs", "code_sql_schema", "code_sql_access",
		"web_cache_lookup", "web_cache_save", "web_cache_search",
		"web_cache_get", "web_cache_status",
		"team_spawn_task", "agent_pull_next_task", "agent_read_task",
		"agent_submit_output", "agent_submit_review", "agent_mark_done",
		"semantic_search", "load_full_details",
		"mnemonic_commit",
		// PR-command tools (010 step 02).
		"code_affected", "code_rename",
		// Per-function CFG/PDG query tool (011 step 01).
		"code_pdg_query",
		// Intraprocedural source->sink taint tool (011 step 02).
		"code_taint",
		// Asset-governance tools (013 step 01).
		"mem_share", "mem_governance",
		// Layered-distill inspector (013 step 02).
		"mem_layers",
		// Session events read (session events layer, TICKET-04).
		"session_changes",
		// Cross-session event query + JSONL export (TICKET-03).
		"mem_query_events", "mem_export_events",
		// Fact Memory (2026-09-04-hermes-memory, TICKET-01 + TICKET-02).
		"fact_add", "fact_search", "fact_forget", "fact_decay", "fact_decay_all",
		// Agent Skill registry (2026-09-04-hermes-memory, TICKET-03).
		"write_skill", "list_skills", "search_skills",
		// Agent Skill sandbox executor (2026-09-04-hermes-memory, TICKET-04).
		"use_skill",
		// Hybrid BM25+vector fact/skill search (2026-09-04-hermes-memory, TICKET-05).
		"hybrid_search",
		// On-demand session context injection (2026-09-24-mnemonic-session-inject, TICKET-05).
		"mem_inject_session",
		// Second-brain ask + lifecycle (2026-09-30-mnemonic-second-brain, TICKET-01).
		"mem_ask", "mem_lifecycle",
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

// TestMemSave asserts mem_save still works on the session fixture: a mem_save
// round-trip (seed a real session, then save) returns a saved id.
func TestMemSave(t *testing.T) {
	sessionHandoffFixture(t)

	startRes, err := handleMemSessionStart(context.Background(), newCallTool("mem_session_start", nil))
	if err != nil {
		t.Fatalf("handleMemSessionStart dispatch: %v", err)
	}
	if startRes.IsError {
		t.Fatalf("mem_session_start errored: %s", callResultText(t, startRes))
	}
	var startOut struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, startRes)), &startOut); err != nil {
		t.Fatalf("unmarshal session start: %v (text %s)", err, callResultText(t, startRes))
	}

	res, err := handleMemSave(context.Background(), newCallTool("mem_save", map[string]any{
		"title":      "mem_save still works",
		"type":       "decision",
		"content":    "session tools removed, mem_save keeps working",
		"session_id": startOut.SessionID,
	}))
	if err != nil {
		t.Fatalf("dispatch mem_save: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected mem_save to succeed, got error: %s", callResultText(t, res))
	}
	if !strings.Contains(callResultText(t, res), `"id"`) {
		t.Errorf("mem_save should return an id, got: %s", callResultText(t, res))
	}
}

// callReq builds an MCP call request with the given name + arguments (nil args
// → empty).
func callReq(name string, args map[string]any) mcplib.CallToolRequest {
	req := mcplib.CallToolRequest{}
	req.Params.Name = name
	if args == nil {
		args = map[string]any{}
	}
	req.Params.Arguments = args
	return req
}

// resultText extracts the raw JSON text from a tool result's content.
func resultText(res *mcplib.CallToolResult) string {
	if res == nil {
		return ""
	}
	for _, c := range res.Content {
		if tc, ok := c.(mcplib.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}
