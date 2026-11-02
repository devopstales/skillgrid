package mcp

// contract_test.go — the D1-pinned `mem_*` tool contract (Task 13, spec
// 004-transport-mcp §3). Any drift in a tool name, a required parameter, or
// the profile membership is a contract break, not a style issue.

import (
	"slices"
	"testing"
)

// agentProfileTools are the spec table's | agent | rows, pinned literally
// (K5: 20). Order is irrelevant; membership + count are the contract.
var agentProfileTools = []string{
	"mem_save",
	"mem_search",
	"mem_get_observation",
	"mem_lookup",
	"mem_search_prompts",
	"mem_context",
	"mem_save_prompt",
	"mem_capture_passive",
	"mem_update",
	"mem_judge",
	"mem_judge_many",
	"mem_compare",
	"mem_consolidate",
	"mem_suggest_topic_key",
	"mem_session_start",
	"mem_session_end",
	"mem_session_summary",
	"mem_session_finalize",
	"mem_current_project",
	"mem_list_projects",
	"mem_pin",
	"mem_unpin",
}

// adminTools are the spec table's | admin | rows (K5: 6). They exist only in
// the All profile; the agent profile must not leak them.
var adminTools = []string{
	"mem_delete",
	"mem_stats",
	"mem_timeline",
	"mem_merge_projects",
	"mem_review",
	"mem_doctor",
}

func TestMemToolContract(t *testing.T) {
	// golden per tool: name, required params, return-shape keys. Any drift = fail.
	s := NewServer(newTestHandler(t), Agent)
	tools := s.ToolNames()
	for _, want := range agentProfileTools {
		if !slices.Contains(tools, want) {
			t.Errorf("missing agent tool %s", want)
		}
	}
	for _, admin := range adminTools {
		if slices.Contains(tools, admin) {
			t.Errorf("admin tool %s leaked into agent profile", admin)
		}
	}
	// The agent profile is exactly the agentProfileTools set (the spec
	// 004-transport-mcp §3 | agent | rows — 22 tools; the brief's "20"
	// counts mem_pin/mem_unpin as one line, but the spec table has them as
	// two rows, so the pin is the literal row count, 22).
	if len(tools) != len(agentProfileTools) {
		t.Fatalf("agent profile must expose exactly the %d agent tools, got %d", len(agentProfileTools), len(tools))
	}
	// save's required params: title, type, content, session_id (project/scope/topic_key optional)
	save := s.ToolSchema("mem_save")
	for _, req := range []string{"title", "type", "content", "session_id"} {
		if !save.Required[req] {
			t.Errorf("mem_save must require %s", req)
		}
	}
}

// TestAllProfileIsAgentPlusAdmin triangulates the other profile: the All
// profile is exactly agent ∪ admin (28 tools) — pinning the "all = agent +
// admin" composition so a future edit cannot add a tool to only one side.
func TestAllProfileIsAgentPlusAdmin(t *testing.T) {
	s := NewServer(newTestHandler(t), All)
	tools := s.ToolNames()
	for _, want := range append(append([]string{}, agentProfileTools...), adminTools...) {
		if !slices.Contains(tools, want) {
			t.Errorf("all profile missing tool %s", want)
		}
	}
	if len(tools) != len(agentProfileTools)+len(adminTools) {
		t.Fatalf("all profile must expose exactly %d tools, got %d", len(agentProfileTools)+len(adminTools), len(tools))
	}
}

// TestSuggestTopicKeyContract pins the D1-pinned under-specified tool (spec
// 004-transport-mcp §3 + 004-dedup §6): the family heuristic's arg surface.
func TestSuggestTopicKeyContract(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent)
	sch := s.ToolSchema("mem_suggest_topic_key")
	for _, req := range []string{"topic", "project"} {
		if !sch.Required[req] {
			t.Errorf("mem_suggest_topic_key must require %s", req)
		}
	}
}

// TestUpdateRequiresExpectedProject pins the ownership assertion (spec
// 004-transport-mcp §4: update/delete take a mandatory expected_project).
func TestUpdateRequiresExpectedProject(t *testing.T) {
	s := NewServer(newTestHandler(t), All) // mem_delete is admin-profile
	for _, tool := range []string{"mem_update", "mem_delete"} {
		sch := s.ToolSchema(tool)
		if !sch.Required["expected_project"] {
			t.Errorf("%s must require expected_project", tool)
		}
	}
}

// TestJudgeCarriesModelArg pins the agent's model-name arg (Task 10: the
// transport is where marked_by_model enters — the service hardcodes ""
// otherwise, and the envelope must not pretend provenance it cannot carry).
func TestJudgeCarriesModelArg(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent)
	for _, tool := range []string{"mem_judge", "mem_judge_many"} {
		if _, ok := s.ToolSchema(tool).Props["model"]; !ok {
			t.Errorf("%s must accept the agent's model name (marked_by_model)", tool)
		}
	}
}
