package mcp

// tools_mem.go — the 26 mem_* tool definitions (D1-pinned, spec
// 004-transport-mcp §3) + their dispatch fns. Each tool is: parse args →
// Handler call → the dispatch fn returns the result (the server wraps it
// in the envelope). The profile (Agent/All) filters which tools are
// registered (K5: 20 agent / 6 admin, the spec table's literal rows).
//
// This file is the MCP adapter's tool surface — the CLI (Task 14) has its
// own adapter over the same Handler. The tool definitions here are the
// D1 contract: the plugins' only interface into the engine.

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// toolEntry is one registered tool: its mcp-go definition + its dispatch
// fn (the args → handler call → result mapping).
type toolEntry struct {
	name    string
	def     mcplib.Tool
	isWrite bool
}

// toolSet returns the profile-filtered tool set (K5: the spec table's
// literal rows — 20 agent + 6 admin).
func (s *Server) toolSet() []toolEntry {
	entries := []toolEntry{
		{name: "mem_save", def: memSaveTool(), isWrite: true},
		{name: "mem_search", def: memSearchTool(), isWrite: false},
		{name: "mem_get_observation", def: memGetObservationTool(), isWrite: false},
		{name: "mem_lookup", def: memLookupTool(), isWrite: false},
		{name: "mem_search_prompts", def: memSearchPromptsTool(), isWrite: false},
		{name: "mem_context", def: memContextTool(), isWrite: false},
		{name: "mem_save_prompt", def: memSavePromptTool(), isWrite: true},
		{name: "mem_capture_passive", def: memCapturePassiveTool(), isWrite: true},
		{name: "mem_update", def: memUpdateTool(), isWrite: true},
		{name: "mem_judge", def: memJudgeTool(), isWrite: true},
		{name: "mem_judge_many", def: memJudgeManyTool(), isWrite: true},
		{name: "mem_compare", def: memCompareTool(), isWrite: false},
		{name: "mem_consolidate", def: memConsolidateTool(), isWrite: true},
		{name: "mem_suggest_topic_key", def: memSuggestTopicKeyTool(), isWrite: false},
		{name: "mem_session_start", def: memSessionStartTool(), isWrite: false},
		{name: "mem_session_end", def: memSessionEndTool(), isWrite: false},
		{name: "mem_session_summary", def: memSessionSummaryTool(), isWrite: false},
		{name: "mem_session_finalize", def: memSessionFinalizeTool(), isWrite: false},
		{name: "mem_current_project", def: memCurrentProjectTool(), isWrite: false},
		{name: "mem_list_projects", def: memListProjectsTool(), isWrite: false},
		{name: "mem_pin", def: memPinTool(), isWrite: true},
		{name: "mem_unpin", def: memUnpinTool(), isWrite: true},
		// admin (All profile only)
		{name: "mem_delete", def: memDeleteTool(), isWrite: true},
		{name: "mem_stats", def: memStatsTool(), isWrite: false},
		{name: "mem_timeline", def: memTimelineTool(), isWrite: false},
		{name: "mem_merge_projects", def: memMergeProjectsTool(), isWrite: true},
		{name: "mem_review", def: memReviewTool(), isWrite: true},
		{name: "mem_doctor", def: memDoctorTool(), isWrite: false},
	}
	if s.profile == Agent {
		// Filter to agent tools (drop the 6 admin).
		admin := map[string]bool{
			"mem_delete": true, "mem_stats": true, "mem_timeline": true,
			"mem_merge_projects": true, "mem_review": true, "mem_doctor": true,
		}
		out := make([]toolEntry, 0, len(entries)-len(admin))
		for _, e := range entries {
			if !admin[e.name] {
				out = append(out, e)
			}
		}
		return out
	}
	return entries
}

// buildToolDispatch maps each tool name → its dispatch fn (the args →
// handler call → result mapping). The dispatch fns are the MCP adapter's
// half of the K6 seam: the CLI adapter (Task 14) has its own, over the
// same Handler.
func (s *Server) buildToolDispatch() map[string]func(ctx context.Context, args map[string]any) (any, error) {
	h := s.h
	d := map[string]func(ctx context.Context, args map[string]any) (any, error){
		"mem_save":              dispatchSave(h),
		"mem_search":            dispatchSearch(h),
		"mem_get_observation":   dispatchGet(h),
		"mem_lookup":            dispatchLookup(h),
		"mem_search_prompts":    dispatchSearchPrompts(h),
		"mem_context":           dispatchContext(h),
		"mem_save_prompt":       dispatchSavePrompt(h),
		"mem_capture_passive":   dispatchCapturePassive(h),
		"mem_update":            dispatchUpdate(h),
		"mem_judge":             dispatchJudge(h),
		"mem_judge_many":        dispatchJudgeMany(h),
		"mem_compare":           dispatchCompare(h),
		"mem_consolidate":       dispatchConsolidate(h),
		"mem_suggest_topic_key": dispatchSuggestTopicKey(h),
		"mem_session_start":     dispatchNoop(),
		"mem_session_end":       dispatchNoop(),
		"mem_session_summary":   dispatchNoop(),
		"mem_session_finalize":  dispatchNoop(),
		"mem_current_project":   dispatchCurrentProject(h),
		"mem_list_projects":     dispatchNoop(),
		"mem_pin":               dispatchPin(h),
		"mem_unpin":             dispatchUnpin(h),
		"mem_delete":            dispatchDelete(h),
		"mem_stats":             dispatchNoop(),
		"mem_timeline":          dispatchNoop(),
		"mem_merge_projects":    dispatchNoop(),
		"mem_review":            dispatchReview(h),
		"mem_doctor":            dispatchNoop(),
	}
	return d
}

// ── tool definitions (the D1 contract — the plugins' only interface) ──────

func memSaveTool() mcplib.Tool {
	return mcplib.NewTool("mem_save",
		mcplib.WithDescription("Save a curated observation to persistent memory. Reuse topic_key to upsert evolving topics. capture_prompt is tri-state: absent → true (default), explicit false → off. Returns candidates[] + judgment_required on conflict detection."),
		mcplib.WithString("title", mcplib.Required(), mcplib.Description("Short searchable title (verb + what)")),
		mcplib.WithString("type", mcplib.Required(), mcplib.Description("Observation type: decision, architecture, bugfix, pattern, config, discovery, learning, preference, convention")),
		mcplib.WithString("content", mcplib.Required(), mcplib.Description("Structured body with What/Why/Where/Learned sections")),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Active session ID")),
		mcplib.WithString("scope", mcplib.Description("Visibility scope: project (default), user, or global")),
		mcplib.WithString("topic_key", mcplib.Description("Stable key for upserts, e.g. architecture/auth-model")),
		mcplib.WithString("project", mcplib.Description("Optional explicit project name (defaults to the CWD-resolved project)")),
		mcplib.WithBoolean("capture_prompt", mcplib.Description("Link the session's latest user prompt (tri-state: absent → true, explicit false → off)")),
		mcplib.WithString("tool_name", mcplib.Description("Optional provenance for which tool produced the save")),
		mcplib.WithString("operation_id", mcplib.Description("Optional idempotency key (retry-safe saves)")),
		mcplib.WithString("current_prompt", mcplib.Description("The current turn's prompt, when capture is on")),
	)
}

func memSearchTool() mcplib.Tool {
	return mcplib.NewTool("mem_search",
		mcplib.WithDescription("FTS (+ vector when armed) search over saved observations. Compact rows with relation annotations."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search keywords")),
		mcplib.WithString("type", mcplib.Description("Filter by observation type")),
		mcplib.WithString("project", mcplib.Description("Filter by project")),
		mcplib.WithString("scope", mcplib.Description("Filter by visibility scope")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum results (default 10)")),
		mcplib.WithString("match_mode", mcplib.Description("Term matching: any (default) or all")),
		mcplib.WithBoolean("all_projects", mcplib.Description("Span every project (default false)")),
	)
}

func memGetObservationTool() mcplib.Tool {
	return mcplib.NewTool("mem_get_observation",
		mcplib.WithDescription("Full untruncated content by id. Accepts both sync_id (INV-6, the agent-visible id) and the local numeric id. include_history/history_cursor for version paging."),
		mcplib.WithNumber("id", mcplib.Description("Local numeric observation id (fallback)")),
		mcplib.WithString("sync_id", mcplib.Description("The sync_id (INV-6, the agent-visible id)")),
		mcplib.WithBoolean("include_history", mcplib.Description("Include version history (default false)")),
		mcplib.WithNumber("history_cursor", mcplib.Description("Version page cursor (default 0)")),
	)
}

func memLookupTool() mcplib.Tool {
	return mcplib.NewTool("mem_lookup",
		mcplib.WithDescription("The wiki-first fallback (narrow raw-fact lookup by id or exact topic_key). Never widens: a bounded phrase hit set (≤5) at most."),
		mcplib.WithNumber("id", mcplib.Description("Local numeric observation id")),
		mcplib.WithString("sync_id", mcplib.Description("The sync_id (INV-6)")),
		mcplib.WithString("topic_key", mcplib.Description("Exact topic_key (the highest-precision path)")),
		mcplib.WithString("phrase", mcplib.Description("Distinctive phrase (bounded FTS, ≤5 hits)")),
		mcplib.WithString("project", mcplib.Description("Project filter")),
		mcplib.WithString("scope", mcplib.Description("Scope filter")),
	)
}

func memSearchPromptsTool() mcplib.Tool {
	return mcplib.NewTool("mem_search_prompts",
		mcplib.WithDescription("Search stored user prompts."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search keywords")),
		mcplib.WithString("project", mcplib.Description("Project filter")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum results (default 10)")),
	)
}

func memContextTool() mcplib.Tool {
	return mcplib.NewTool("mem_context",
		mcplib.WithDescription("Recent context (sessions/prompts/observations) with a byte budget (16–64 KiB), pinned ≤20, older_history."),
		mcplib.WithNumber("budget", mcplib.Description("Byte budget (default 16384, max 65536)")),
		mcplib.WithBoolean("older_history", mcplib.Description("Page older history (default false)")),
	)
}

func memSavePromptTool() mcplib.Tool {
	return mcplib.NewTool("mem_save_prompt",
		mcplib.WithDescription("Store a user prompt; feeds process-local context for capture_prompt."),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session ID")),
		mcplib.WithString("content", mcplib.Required(), mcplib.Description("The prompt text")),
		mcplib.WithString("project", mcplib.Description("Project (defaults to CWD-resolved)")),
		mcplib.WithString("source_inbox_id", mcplib.Description("Optional source inbox id")),
	)
}

func memCapturePassiveTool() mcplib.Tool {
	return mcplib.NewTool("mem_capture_passive",
		mcplib.WithDescription("Extract ## Key Learnings: items from text into observations."),
		mcplib.WithString("text", mcplib.Required(), mcplib.Description("The text to extract from")),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session ID")),
		mcplib.WithString("project", mcplib.Description("Project (defaults to CWD-resolved)")),
	)
}

func memUpdateTool() mcplib.Tool {
	return mcplib.NewTool("mem_update",
		mcplib.WithDescription("Partial update by id; mandatory expected_project ownership assertion."),
		mcplib.WithNumber("id", mcplib.Required(), mcplib.Description("Observation id")),
		mcplib.WithString("expected_project", mcplib.Required(), mcplib.Description("The project the observation belongs to (ownership assertion)")),
		mcplib.WithString("type", mcplib.Description("New type")),
		mcplib.WithString("title", mcplib.Description("New title")),
		mcplib.WithString("content", mcplib.Description("New content")),
		mcplib.WithString("topic_key", mcplib.Description("New topic_key")),
	)
}

func memJudgeTool() mcplib.Tool {
	return mcplib.NewTool("mem_judge",
		mcplib.WithDescription("Record a verdict on a pending judgment_id (relation/reason/evidence/confidence). The agent's model name is stamped as marked_by_model."),
		mcplib.WithNumber("judgment_id", mcplib.Required(), mcplib.Description("The pending relation id")),
		mcplib.WithString("relation", mcplib.Required(), mcplib.Description("The verdict: conflicts_with, supersedes, refines, compatible, or custom")),
		mcplib.WithString("reason", mcplib.Description("Why this verdict")),
		mcplib.WithString("evidence", mcplib.Description("Supporting evidence")),
		mcplib.WithNumber("confidence", mcplib.Description("Confidence 0–1 (optional)")),
		mcplib.WithString("model", mcplib.Description("The agent's model name (marked_by_model provenance)")),
	)
}

func memJudgeManyTool() mcplib.Tool {
	return mcplib.NewTool("mem_judge_many",
		mcplib.WithDescription("Batch 1–32 judgments in one queued write (one transaction)."),
		mcplib.WithString("judgments", mcplib.Required(), mcplib.Description("JSON array of {judgment_id, relation, reason, evidence, confidence}")),
		mcplib.WithString("model", mcplib.Description("The agent's model name (marked_by_model provenance)")),
	)
}

func memCompareTool() mcplib.Tool {
	return mcplib.NewTool("mem_compare",
		mcplib.WithDescription("Proactive semantic comparison → creates a judged relation directly (idempotent per pair)."),
		mcplib.WithNumber("a_id", mcplib.Required(), mcplib.Description("First observation id")),
		mcplib.WithNumber("b_id", mcplib.Required(), mcplib.Description("Second observation id")),
	)
}

func memConsolidateTool() mcplib.Tool {
	return mcplib.NewTool("mem_consolidate",
		mcplib.WithDescription("Preview/confirm merge of ≥2 observations (fingerprint-gated, destructive)."),
		mcplib.WithString("ids_to_delete", mcplib.Required(), mcplib.Description("JSON array of observation ids to merge")),
		mcplib.WithString("new_title", mcplib.Required(), mcplib.Description("The merged observation's title")),
		mcplib.WithString("new_content", mcplib.Required(), mcplib.Description("The merged observation's content")),
		mcplib.WithString("expected_project", mcplib.Required(), mcplib.Description("The project (ownership assertion)")),
		mcplib.WithString("type", mcplib.Description("Override the survivor's type")),
		mcplib.WithString("topic_key", mcplib.Description("Override the survivor's topic_key")),
		mcplib.WithString("fingerprint", mcplib.Description("The preview's fingerprint (required for confirm)")),
		mcplib.WithBoolean("confirm", mcplib.Description("Confirm the merge (default false = preview)")),
	)
}

func memSuggestTopicKeyTool() mcplib.Tool {
	return mcplib.NewTool("mem_suggest_topic_key",
		mcplib.WithDescription("Family-heuristic topic key suggestion. A suggested topic_key is a hint, not proof of identity (spec 004-dedup §6): the agent picks one or writes a new key."),
		mcplib.WithString("topic", mcplib.Required(), mcplib.Description("The draft title or a topic fragment")),
		mcplib.WithString("project", mcplib.Required(), mcplib.Description("The project to suggest for")),
	)
}

func memSessionStartTool() mcplib.Tool {
	return mcplib.NewTool("mem_session_start",
		mcplib.WithDescription("Register a session."),
		mcplib.WithString("id", mcplib.Required(), mcplib.Description("Session ID")),
		mcplib.WithString("project", mcplib.Description("The project (defaults to CWD-resolved)")),
		mcplib.WithString("directory", mcplib.Description("Workspace directory")),
		mcplib.WithString("ownership_mode", mcplib.Description("shared (default) or project_owned")),
		mcplib.WithBoolean("resume", mcplib.Description("Resume an existing session (default false)")),
		mcplib.WithBoolean("isolated", mcplib.Description("Isolated session (default false)")),
	)
}

func memSessionEndTool() mcplib.Tool {
	return mcplib.NewTool("mem_session_end",
		mcplib.WithDescription("End a session (optional free-text summary)."),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session ID to end")),
		mcplib.WithString("summary", mcplib.Description("Optional end-of-session summary")),
	)
}

func memSessionSummaryTool() mcplib.Tool {
	return mcplib.NewTool("mem_session_summary",
		mcplib.WithDescription("Persist the six-section structured summary (Goal/Instructions/Discoveries/Accomplished/Next Steps/Relevant Files)."),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session ID")),
		mcplib.WithString("goal", mcplib.Description("Goal section")),
		mcplib.WithString("instructions", mcplib.Description("Instructions section")),
		mcplib.WithString("discoveries", mcplib.Description("Discoveries section")),
		mcplib.WithString("accomplished", mcplib.Description("Accomplished section")),
		mcplib.WithString("next_steps", mcplib.Description("Next Steps section")),
		mcplib.WithString("relevant_files", mcplib.Description("Relevant Files section")),
	)
}

func memSessionFinalizeTool() mcplib.Tool {
	return mcplib.NewTool("mem_session_finalize",
		mcplib.WithDescription("Atomic summary + close + idempotent retry journal."),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session ID")),
		mcplib.WithString("summary", mcplib.Description("The summary (optional)")),
	)
}

func memCurrentProjectTool() mcplib.Tool {
	return mcplib.NewTool("mem_current_project",
		mcplib.WithDescription("Project discovery (the resolved current project)."),
		mcplib.WithString("project", mcplib.Description("Optional explicit project (overrides detection)")),
	)
}

func memListProjectsTool() mcplib.Tool {
	return mcplib.NewTool("mem_list_projects",
		mcplib.WithDescription("List known projects."),
	)
}

func memPinTool() mcplib.Tool {
	return mcplib.NewTool("mem_pin",
		mcplib.WithDescription("Local-only pin (search boost + context ordering)."),
		mcplib.WithNumber("id", mcplib.Required(), mcplib.Description("Observation id")),
		mcplib.WithString("project", mcplib.Description("Project (defaults to CWD-resolved)")),
	)
}

func memUnpinTool() mcplib.Tool {
	return mcplib.NewTool("mem_unpin",
		mcplib.WithDescription("Clear the pinned flag."),
		mcplib.WithNumber("id", mcplib.Required(), mcplib.Description("Observation id")),
		mcplib.WithString("project", mcplib.Description("Project (defaults to CWD-resolved)")),
	)
}

func memDeleteTool() mcplib.Tool {
	return mcplib.NewTool("mem_delete",
		mcplib.WithDescription("Soft/hard delete; mandatory expected_project ownership assertion."),
		mcplib.WithNumber("id", mcplib.Required(), mcplib.Description("Observation id")),
		mcplib.WithString("expected_project", mcplib.Required(), mcplib.Description("The project (ownership assertion)")),
		mcplib.WithBoolean("hard", mcplib.Description("Hard delete (default false = soft)")),
	)
}

func memStatsTool() mcplib.Tool {
	return mcplib.NewTool("mem_stats",
		mcplib.WithDescription("Per-type/project statistics."),
		mcplib.WithString("project", mcplib.Description("Project filter")),
	)
}

func memTimelineTool() mcplib.Tool {
	return mcplib.NewTool("mem_timeline",
		mcplib.WithDescription("Before/after in a session."),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session ID")),
	)
}

func memMergeProjectsTool() mcplib.Tool {
	return mcplib.NewTool("mem_merge_projects",
		mcplib.WithDescription("Merge project spelling variants."),
		mcplib.WithString("source", mcplib.Required(), mcplib.Description("The variant to merge from")),
		mcplib.WithString("target", mcplib.Required(), mcplib.Description("The canonical project")),
	)
}

func memReviewTool() mcplib.Tool {
	return mcplib.NewTool("mem_review",
		mcplib.WithDescription("list overdue / mark_reviewed (restart decay clock)."),
		mcplib.WithString("project", mcplib.Description("Project filter")),
		mcplib.WithBoolean("list", mcplib.Description("List overdue (default true)")),
		mcplib.WithNumber("mark_reviewed_id", mcplib.Description("Mark this id reviewed (restarts the decay clock)")),
	)
}

func memDoctorTool() mcplib.Tool {
	return mcplib.NewTool("mem_doctor",
		mcplib.WithDescription("Diagnostics (embed round-trip, capabilities)."),
	)
}
