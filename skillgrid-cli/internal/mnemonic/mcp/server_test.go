package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/relay"
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
		// Session relay handoff/resume (006 step 02).
		"session_handoff", "session_resume",
		// Session status + thin knowledge compact (006 step 03).
		"session_status", "knowledge_compact",
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

// TestSessionStatus asserts (006 step 03) the session_status tool is registered
// and, on a fresh store with no handoffs yet, reports zero counts (warn +
// continue — not a crash). It runs under the step-02 fixture so the pinned
// project maps to a fresh store and the cleave dir lives under the temp root.
func TestSessionStatus(t *testing.T) {
	sessionHandoffFixture(t)

	res, err := handleSessionStatus(context.Background(), newCallTool("session_status", nil))
	if err != nil {
		t.Fatalf("dispatch session_status: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", callResultText(t, res))
	}
	if !strings.Contains(callResultText(t, res), `"handoff_count":0`) {
		t.Errorf("expected handoff_count 0 on an empty store, got: %s", callResultText(t, res))
	}
}

// TestKnowledgeCompact asserts (006 step 03) the knowledge_compact tool is
// registered and a THIN compact succeeds with NO Fact Memory: a session that
// has handoff inputs (the .cleave/ bundle) but no Fact Memory still gets a
// KNOWLEDGE.md written, with no error. The handoff bundle is seeded via the
// session_handoff MCP call (which writes the cleave files + a handoff row) and
// no Fact Memory observation is ever created — only the relay bundle is input.
func TestKnowledgeCompact(t *testing.T) {
	dir, _ := sessionHandoffFixture(t)

	// Seed handoff inputs on disk (the .cleave/ bundle) via the MCP tool, with
	// NO Fact Memory written. The relay handoff writes the bundle + a
	// session_handoffs row, which is the "handoff input" the thin compact reads.
	hoRes, err := handleSessionHandoff(context.Background(), newCallTool("session_handoff", map[string]any{
		"progress":        "did the thing",
		"knowledge":       "fail closed first",
		"next_prompt":     "resume later",
		"handoff_id":      "ho-compact",
		"context_summary": "compact me",
	}))
	if err != nil {
		t.Fatalf("seed session_handoff: %v", err)
	}
	if hoRes.IsError {
		t.Fatalf("seed handoff errored: %s", callResultText(t, hoRes))
	}

	res, err := handleKnowledgeCompact(context.Background(), newCallTool("knowledge_compact", nil))
	if err != nil {
		t.Fatalf("dispatch knowledge_compact: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected compact to succeed (no Fact Memory), got error: %s", callResultText(t, res))
	}
	if !strings.Contains(callResultText(t, res), `"knowledge_path"`) {
		t.Errorf("expected a knowledge_path in the result, got: %s", callResultText(t, res))
	}
	// The thin refresh wrote a KNOWLEDGE.md (no error, even with no Fact Memory).
	cleaveRoot, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("evalsymlinks dir: %v", err)
	}
	kp := filepath.Join(relay.BundleDir(cleaveRoot), relay.FileKnowledge)
	if _, err := os.Stat(kp); err != nil {
		t.Errorf("expected a refreshed KNOWLEDGE.md at %s: %v", kp, err)
	}
}

// TestMemSave asserts (006 step 03) the new session_status / knowledge_compact
// tools are additive: mem_save still works after they are registered. A mem_save
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
		"title":      "Step 03 mem_save still works",
		"type":       "decision",
		"content":    "session_status and knowledge_compact are additive",
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
