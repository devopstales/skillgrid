package mcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/project"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

func registerMemoryTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{memSaveTool(), handleMemSave},
		{memSearchTool(), handleMemSearch},
		{memContextTool(), handleMemContext},
		{memGetObservationTool(), handleMemGetObservation},
		{memTimelineTool(), handleMemTimeline},
		{memUpdateTool(), handleMemUpdate},
		{memDeleteTool(), handleMemDelete},
		{memStatsTool(), handleMemStats},
		{memSavePromptTool(), handleMemSavePrompt},
		{memCurrentProjectTool(), handleMemCurrentProject},
		{memDoctorTool(), handleMemDoctor},
		{memReviewTool(), handleMemReview},
		{memJudgeTool(), handleMemJudge},
		{memCompareTool(), handleMemCompare},
		{memMergeProjectsTool(), handleMemMergeProjects},
		{memSessionStartTool(), handleMemSessionStart},
		{memSessionEndTool(), handleMemSessionEnd},
		{memSessionSummaryTool(), handleMemSessionSummary},
		{memSessionSetTitleTool(), handleMemSessionSetTitle},
		{memCapturePassiveTool(), handleMemCapturePassive},
		{memSuggestTopicKeyTool(), handleMemSuggestTopicKey},
		{memPinTool(), handleMemPin},
		{memUnpinTool(), handleMemUnpin},
		{memUnifyTool(), handleMemUnify},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func memSaveTool() mcplib.Tool {
	return mcplib.NewTool("mem_save",
		mcplib.WithDescription("Save a curated observation to persistent memory. Use structured content with **What**, **Why**, **Where**, and **Learned** sections. Reuse topic_key to upsert evolving topics. Best-effort links the session's most recent user prompt when capture_prompt=true (default); pass capture_prompt=false for automated saves that should not carry prompt context."),
		mcplib.WithString("title", mcplib.Required(), mcplib.Description("Short searchable title (verb + what)")),
		mcplib.WithString("type", mcplib.Required(), mcplib.Description("Observation type: decision, architecture, bugfix, pattern, config, discovery, learning, preference, convention")),
		mcplib.WithString("content", mcplib.Required(), mcplib.Description("Structured body with What/Why/Where/Learned sections")),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Active session ID from mem_session_start")),
		mcplib.WithString("scope", mcplib.Description("Visibility scope: project (default), user, or global")),
		mcplib.WithString("topic_key", mcplib.Description("Stable key for upserts, e.g. architecture/auth-model")),
		mcplib.WithString("project", mcplib.Description("Optional explicit project name to record under (defaults to the CWD-resolved project). Surfaced as a drift warning if a prior mem_merge_projects retired it.")),
		mcplib.WithBoolean("capture_prompt", mcplib.Description("Link the session's latest user prompt to this observation (default true). Pass false for SDD artifacts / automated saves that should not carry prompt context.")),
		mcplib.WithString("tool_name", mcplib.Description("Optional provenance for which tool produced the save (e.g. mem_save).")),
	)
}

func memSearchTool() mcplib.Tool {
	return mcplib.NewTool("mem_search",
		mcplib.WithDescription("Full-text search over saved observations using FTS5. Pass `project` to scope or override the CWD-resolved project, and `scope` to restrict the visibility scope (project/user/global). Set `all_projects=true` to span every store — ranks are merged across projects so a parent directory can find memories saved under a child project."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search keywords")),
		mcplib.WithString("match_mode", mcplib.Description("Term matching: any (default) or all")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum results (default 20)")),
		mcplib.WithString("project", mcplib.Description("Optional project name to search under (defaults to the CWD-resolved project). If a prior mem_merge_projects retired it, a drift warning is returned alongside the hits.")),
		mcplib.WithString("scope", mcplib.Description("Optional visibility scope filter (project|user|global).")),
		mcplib.WithBoolean("all_projects", mcplib.Description("Span every project store and merge results by cross-project rank (default false). Useful when the CWD is a parent of several repositories or when you don't know which bucket the memory is in.")),
	)
}

func memContextTool() mcplib.Tool {
	return mcplib.NewTool("mem_context",
		mcplib.WithDescription("Recent session summaries for fast recall before a full search."),
		mcplib.WithNumber("limit", mcplib.Description("Maximum sessions (default 5)")),
	)
}

func memGetObservationTool() mcplib.Tool {
	return mcplib.NewTool("mem_get_observation",
		mcplib.WithDescription("Fetch full untruncated observation content by ID."),
		mcplib.WithNumber("id", mcplib.Required(), mcplib.Description("Observation ID from mem_search")),
	)
}

func memSessionStartTool() mcplib.Tool {
	return mcplib.NewTool("mem_session_start",
		mcplib.WithDescription("Create a new workspace session. Required before mem_save in OpenCode plugin flows. Optional `title` names the session — shown in the web dashboard session list."),
		mcplib.WithString("directory", mcplib.Description("Workspace directory (defaults to cwd)")),
		mcplib.WithString("title", mcplib.Description("Optional human-readable session name (e.g. 'Skillgrid CLI dashboard status card updates').")),
	)
}

func memSessionEndTool() mcplib.Tool {
	return mcplib.NewTool("mem_session_end",
		mcplib.WithDescription("End a session with optional summary."),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session ID to end")),
		mcplib.WithString("summary", mcplib.Description("Optional end-of-session summary")),
	)
}

func memSessionSummaryTool() mcplib.Tool {
	return mcplib.NewTool("mem_session_summary",
		mcplib.WithDescription("Persist structured end-of-session summary before closing."),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session ID")),
		mcplib.WithString("summary", mcplib.Required(), mcplib.Description("Structured session summary (Goal, Discoveries, Accomplished, Next Steps, Relevant Files)")),
	)
}

func memSessionSetTitleTool() mcplib.Tool {
	return mcplib.NewTool("mem_session_set_title",
		mcplib.WithDescription("Rename a session. The title is shown in the web dashboard session list (mem-sessions)."),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("ID of the session to rename")),
		mcplib.WithString("title", mcplib.Required(), mcplib.Description("New human-readable title, e.g. 'Skillgrid CLI dashboard status card updates'")),
	)
}

func memCapturePassiveTool() mcplib.Tool {
	return mcplib.NewTool("mem_capture_passive",
		mcplib.WithDescription("Extract and save structured learnings from pasted text (e.g. a finished task transcript). The server recognises 'Key Learnings:' sections and labelled Lesson/Discovery lines and stores each as a passive observation. Idempotent — re-capturing the same text does not duplicate rows."),
		mcplib.WithString("content", mcplib.Required(), mcplib.Description("Text to scan for extractable learnings (a '## Key Learnings:' section or numbered/bulleted items)")),
		mcplib.WithString("session_id", mcplib.Description("Session to attribute the capture to (defaults to the current one)")),
		mcplib.WithString("source", mcplib.Description("Provenance label, e.g. 'task-complete' (default 'passive')")),
	)
}

func handleMemCapturePassive(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	content, err := req.RequireString("content")
	if err != nil {
		return toolError(err)
	}
	sessionID := req.GetString("session_id", "")
	if sessionID == "" {
		return toolError(errors.New("session_id is required"))
	}
	source := req.GetString("source", "passive")

	res, err := svc.CapturePassive(ctx, projectID, service.PassiveInput{
		Content:   content,
		SessionID: sessionID,
		Source:    source,
	})
	if err != nil {
		return toolError(err)
	}
	return JSONResult(res)
}

func memSuggestTopicKeyTool() mcplib.Tool {
	return mcplib.NewTool("mem_suggest_topic_key",
		mcplib.WithDescription("Suggest a stable topic_key from type and title for mem_save upserts."),
		mcplib.WithString("type", mcplib.Required(), mcplib.Description("Observation type")),
		mcplib.WithString("title", mcplib.Description("Observation title (preferred for key segment)")),
		mcplib.WithString("content", mcplib.Description("Fallback text when title is empty")),
	)
}

func handleMemSave(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, defaultProjectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	title, err := req.RequireString("title")
	if err != nil {
		return toolError(err)
	}
	typ, err := req.RequireString("type")
	if err != nil {
		return toolError(err)
	}
	content, err := req.RequireString("content")
	if err != nil {
		return toolError(err)
	}
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return toolError(err)
	}

	explicitProject := strings.TrimSpace(req.GetString("project", ""))
	projectName := explicitProject
	projectID := defaultProjectID
	var drift *service.ProjectDrift
	if explicitProject != "" {
		projectID = project.NormalizeID(explicitProject)
		// If the explicit name is a retired alias, route the write to the
		// canonical store (where the session/FK lives) and surface the drift
		// warning so the caller learns to name the canonical project directly.
		if d, dErr := svc.CheckProjectDrift(ctx, explicitProject); dErr == nil && d != nil {
			drift = d
			projectID = project.NormalizeID(d.CanonicalName)
		}
	}

	// capture_prompt defaults to true (Engram's documented default is false, but
	// Mnemonic's pipeline feeds prompts via mem_save_prompt, so defaulting to
	// link-on-save is the safer fidelity choice; automated saves opt out).
	capturePrompt := req.GetBool("capture_prompt", true)

	id, err := svc.SaveObservation(ctx, projectID, service.SaveObservationInput{
		Title:         title,
		Type:          typ,
		Content:       content,
		Scope:         req.GetString("scope", "project"),
		TopicKey:      req.GetString("topic_key", ""),
		SessionID:     sessionID,
		CapturePrompt: capturePrompt,
		ProjectName:   projectName,
		ToolName:      req.GetString("tool_name", ""),
	})
	if err != nil {
		return toolError(err)
	}

	out := map[string]any{"id": id, "project": projectID}
	if drift != nil {
		out["project_drift"] = drift
	}
	return JSONResult(out)
}

func handleMemSearch(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, defaultProjectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	query, err := req.RequireString("query")
	if err != nil {
		return toolError(err)
	}

	matchMode := req.GetString("match_mode", "any")
	limit := int(req.GetFloat("limit", 20))
	scope := strings.TrimSpace(req.GetString("scope", ""))
	allProjects := req.GetBool("all_projects", false)

	if allProjects {
		hits, err := svc.SearchAllProjects(ctx, query, matchMode, scope, limit)
		if err != nil {
			return toolError(err)
		}
		return JSONResult(map[string]any{
			"project":      "all",
			"all_projects": true,
			"count":        len(hits),
			"observations": observationDTOs(hits),
		})
	}

	explicitProject := strings.TrimSpace(req.GetString("project", ""))
	projectID := defaultProjectID
	var drift *service.ProjectDrift
	if explicitProject != "" {
		projectID = project.NormalizeID(explicitProject)
		// A retired alias: route the read to the canonical store so the caller
		// still sees the consolidated memories, and warn.
		if d, dErr := svc.CheckProjectDrift(ctx, explicitProject); dErr == nil && d != nil {
			drift = d
			projectID = project.NormalizeID(d.CanonicalName)
		}
	}

	hits, err := svc.SearchObservationsScoped(ctx, projectID, query, matchMode, scope, limit)
	if err != nil {
		return toolError(err)
	}

	out := map[string]any{"project": projectID, "observations": observationDTOs(hits)}
	if drift != nil {
		out["project_drift"] = drift
	}
	return JSONResult(out)
}

func handleMemContext(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	limit := int(req.GetFloat("limit", 5))
	sessions, err := svc.RecentContext(ctx, projectID, limit)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"sessions": sessionDTOs(sessions)})
}

func handleMemGetObservation(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	id, err := req.RequireFloat("id")
	if err != nil {
		return toolError(err)
	}

	obs, err := svc.GetObservation(ctx, projectID, int64(id))
	if err != nil {
		return toolError(err)
	}
	return JSONResult(observationDTO(obs))
}

func handleMemSessionSetTitle(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return toolError(err)
	}
	title, err := req.RequireString("title")
	if err != nil {
		return toolError(err)
	}
	if err := svc.SessionSetTitle(ctx, projectID, sessionID, title); err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"session_id": sessionID, "title": title, "title_set": true})
}

func handleMemSessionStart(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	dir := req.GetString("directory", "")
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return toolError(err)
		}
	}
	title := req.GetString("title", "")

	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}

	sessionID, _, err := svc.SessionStart(ctx, dir, title)
	if err != nil {
		return toolError(err)
	}
	out := map[string]string{"session_id": sessionID}
	if title != "" {
		out["title"] = title
	}
	return JSONResult(out)
}

func handleMemSessionEnd(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return toolError(err)
	}

	if err := svc.SessionEnd(ctx, projectID, sessionID, req.GetString("summary", "")); err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"session_id": sessionID, "status": "ended"})
}

func handleMemSessionSummary(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return toolError(err)
	}
	summary, err := req.RequireString("summary")
	if err != nil {
		return toolError(err)
	}

	if err := svc.SessionSummary(ctx, projectID, sessionID, summary); err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"session_id": sessionID, "saved": true})
}

func handleMemSuggestTopicKey(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	typ, err := req.RequireString("type")
	if err != nil {
		return toolError(err)
	}
	key := suggestTopicKey(typ, req.GetString("title", ""), req.GetString("content", ""))
	return JSONResult(map[string]string{"topic_key": key})
}

// ────────────────────────────── Mem: timeline ─────────────────────────────

func memTimelineTool() mcplib.Tool {
	return mcplib.NewTool("mem_timeline",
		mcplib.WithDescription("Chronological context around a specific observation — 'what happened before and after' — the progressive-disclosure middle layer between mem_search (compact) and mem_get_observation (full content)."),
		mcplib.WithNumber("id", mcplib.Required(), mcplib.Description("Anchor observation ID from mem_search or mem_get_observation")),
		mcplib.WithString("window", mcplib.Description("Time window on each side (e.g. '30m', '2h'); default 1h")),
		mcplib.WithNumber("limit", mcplib.Description("Max entries per direction; default 5")),
	)
}

var durationRe = regexp.MustCompile(`^(\d+)\s*(s|m|h|d)$`)

func parseTimelineWindow(s string) time.Duration {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return 1 * time.Hour
	}
	if m := durationRe.FindStringSubmatch(trimmed); m != nil {
		n, _ := strconv.Atoi(m[1])
		switch m[2] {
		case "s":
			return time.Duration(n) * time.Second
		case "m":
			return time.Duration(n) * time.Minute
		case "h":
			return time.Duration(n) * time.Hour
		case "d":
			return time.Duration(n) * 24 * time.Hour
		}
	}
	return 1 * time.Hour
}

func handleMemTimeline(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	id, err := req.RequireFloat("id")
	if err != nil {
		return toolError(err)
	}
	window := parseTimelineWindow(req.GetString("window", ""))
	limit := int(req.GetFloat("limit", 5))

	tl, err := svc.ObservationTimeline(ctx, projectID, int64(id), window, limit)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"anchor_id": int64(id),
		"before":    tl.Before,
		"after":     tl.After,
	})
}

// ────────────────────────────── Mem: update / delete ───────────────────────

func memUpdateTool() mcplib.Tool {
	return mcplib.NewTool("mem_update",
		mcplib.WithDescription("Update an existing observation in place. Only non-empty fields are applied; omitted fields are unchanged. Bumps updated_at; FTS stays in sync."),
		mcplib.WithNumber("id", mcplib.Required(), mcplib.Description("Observation ID")),
		mcplib.WithString("title", mcplib.Description("New title (leave blank to keep)")),
		mcplib.WithString("content", mcplib.Description("New content (leave blank to keep)")),
		mcplib.WithString("type", mcplib.Description("New type (leave blank to keep)")),
		mcplib.WithString("scope", mcplib.Description("New scope (leave blank to keep)")),
		mcplib.WithString("topic_key", mcplib.Description("New topic_key (leave blank to keep)")),
	)
}

func handleMemUpdate(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	id, err := req.RequireFloat("id")
	if err != nil {
		return toolError(err)
	}
	in := memory.UpdateInput{
		Title:    req.GetString("title", ""),
		Content:  req.GetString("content", ""),
		Type:     req.GetString("type", ""),
		Scope:    req.GetString("scope", ""),
		TopicKey: req.GetString("topic_key", ""),
	}
	if err := svc.UpdateObservation(ctx, projectID, int64(id), in); err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"id": int64(id), "updated": true})
}

func memDeleteTool() mcplib.Tool {
	return mcplib.NewTool("mem_delete",
		mcplib.WithDescription("Delete an observation. Soft-delete by default (deleted_at set — excluded from search/context/timeline, still recoverable); pass hard=true to remove the row permanently."),
		mcplib.WithNumber("id", mcplib.Required(), mcplib.Description("Observation ID")),
		mcplib.WithBoolean("hard", mcplib.Description("Hard-delete (default false)")),
	)
}

func handleMemDelete(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	id, err := req.RequireFloat("id")
	if err != nil {
		return toolError(err)
	}
	hard := req.GetBool("hard", false)
	if err := svc.DeleteObservation(ctx, projectID, int64(id), hard); err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"id": int64(id), "deleted": true, "hard": hard})
}

// ────────────────────────────── Mem: stats / doctor / project ──────────────

func memStatsTool() mcplib.Tool {
	return mcplib.NewTool("mem_stats",
		mcplib.WithDescription("Memory system statistics for the current project: observation count by type, active/total sessions, created-range."),
	)
}

func handleMemStats(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	st, err := svc.MemoryStatus(ctx, projectID)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(st)
}

func memSavePromptTool() mcplib.Tool {
	return mcplib.NewTool("mem_save_prompt",
		mcplib.WithDescription("Record a user prompt so future sessions can recall what was asked. Prompts are trimmed (min 11 chars) and bounded (max 2000 chars), and deduplicated within the same session."),
		mcplib.WithString("content", mcplib.Required(), mcplib.Description("The user prompt to store")),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Session ID to attribute the prompt to")),
	)
}

func handleMemSavePrompt(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	content, err := req.RequireString("content")
	if err != nil {
		return toolError(err)
	}
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return toolError(err)
	}
	id, err := svc.SavePrompt(ctx, projectID, service.PromptInput{
		SessionID: sessionID,
		Content:   content,
	})
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"id": id, "saved": true})
}

func memCurrentProjectTool() mcplib.Tool {
	return mcplib.NewTool("mem_current_project",
		mcplib.WithDescription("Detect project from cwd — never errors, recommended first call. Returns the resolved project ID, the resolution source (config/git/fallback), and the list of all available projects."),
	)
}

func handleMemCurrentProject(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return toolError(err)
	}
	info, err := svc.CurrentProject(cwd)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(info)
}

func memDoctorTool() mcplib.Tool {
	return mcplib.NewTool("mem_doctor",
		mcplib.WithDescription("Run read-only operational diagnostics: schema version, WAL state, FTS row counts and drift against the base tables, by-type counts, and disk size."),
	)
}

func handleMemDoctor(ctx context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	out, err := svc.MemoryDoctor(ctx, projectID)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(out)
}

// ────────────────────────────── Mem: review cycle ──────────────────────────

func memReviewTool() mcplib.Tool {
	return mcplib.NewTool("mem_review",
		mcplib.WithDescription("List observations due for local review (review_after <= now), or mark an observation reviewed to reset its review cycle. `action=list` (default) returns due entries; `action=mark_reviewed` with id advances the cycle."),
		mcplib.WithString("action", mcplib.Description("list (default) or mark_reviewed")),
		mcplib.WithNumber("id", mcplib.Description("Observation ID (required for mark_reviewed)")),
		mcplib.WithNumber("limit", mcplib.Description("Max due entries to return (default 20)")),
	)
}

func handleMemReview(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	action := req.GetString("action", "list")
	switch action {
	case "list":
		limit := int(req.GetFloat("limit", 20))
		due, err := svc.ListReviews(ctx, projectID, limit)
		if err != nil {
			return toolError(err)
		}
		return JSONResult(map[string]any{"due": due, "count": len(due)})
	case "mark_reviewed":
		id, err := req.RequireFloat("id")
		if err != nil {
			return toolError(err)
		}
		reviewAfter, err := svc.MarkReviewReviewed(ctx, projectID, int64(id))
		if err != nil {
			return toolError(err)
		}
		return JSONResult(map[string]any{"id": int64(id), "marked_reviewed": true, "review_after": reviewAfter})
	default:
		return toolError(fmt.Errorf("unknown action %q (valid: list, mark_reviewed)", action))
	}
}

// ────────────────────────────── Mem: relations (judge/compare) ────────────

const relationsHelp = "related | compatible | scoped | conflicts_with | supersedes | not_conflict"

// memJudgeTool records a verdict for a pending memory conflict (Engram:
// `mem_judge` — "Record a verdict for a pending memory conflict surfaced by
// mem_save"). In Mnemonic the "conflict" is any pair of observations; the
// verdict is stored as a typed semantic link.
func memJudgeTool() mcplib.Tool {
	return mcplib.NewTool("mem_judge",
		mcplib.WithDescription("Record a verdict for a memory conflict between two observations. Verdicts: "+relationsHelp+". 'not_conflict' removes a previously-recorded conflicts_with link instead of adding one."),
		mcplib.WithNumber("src_id", mcplib.Required(), mcplib.Description("Source observation ID")),
		mcplib.WithNumber("dst_id", mcplib.Required(), mcplib.Description("Destination observation ID")),
		mcplib.WithString("verdict", mcplib.Required(), mcplib.Description("Verdict: "+relationsHelp)),
		mcplib.WithNumber("confidence", mcplib.Description("Optional 0.0-1.0 confidence in the verdict")),
		mcplib.WithString("reason", mcplib.Description("Short justification (stored as reason on the link)")),
	)
}

// memCompareTool records, clears, or inspects semantic relations between
// observations (Engram: `mem_compare` — "Persist a semantic relation verdict
// between two existing observations").
//
//   - src_id + dst_id + relation: record or clear a link.
//   - src_id + dst_id, no relation: list the live links between the two.
//   - src_id + dst_id + not_conflict: clear a conflicts_with link.
func memCompareTool() mcplib.Tool {
	return mcplib.NewTool("mem_compare",
		mcplib.WithDescription("Record, clear, or inspect semantic relations between two existing observations. Relations: related | compatible | scoped | conflicts_with | supersedes. 'not_conflict' clears a conflicts_with link. Omit the relation to list current links between the pair."),
		mcplib.WithNumber("src_id", mcplib.Required(), mcplib.Description("Source observation ID")),
		mcplib.WithNumber("dst_id", mcplib.Required(), mcplib.Description("Destination observation ID")),
		mcplib.WithString("relation", mcplib.Description("Relation to record or clear (omit to list)")),
		mcplib.WithNumber("confidence", mcplib.Description("Optional 0.0-1.0 confidence")),
		mcplib.WithString("reason", mcplib.Description("Optional justification")),
	)
}

func handleMemJudge(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	return recordRelation(ctx, svc, projectID, req, "verdict")
}

func handleMemCompare(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	relation := strings.TrimSpace(req.GetString("relation", ""))
	if relation == "" {
		// List mode: return the live links between the two observations.
		srcID, err := req.RequireFloat("src_id")
		if err != nil {
			return toolError(err)
		}
		dstID, err := req.RequireFloat("dst_id")
		if err != nil {
			return toolError(err)
		}
		rels, err := svc.RelationsBetween(ctx, projectID, int64(srcID), int64(dstID))
		if err != nil {
			return toolError(err)
		}
		return JSONResult(map[string]any{
			"src_id":    int64(srcID),
			"dst_id":    int64(dstID),
			"relations": rels,
			"count":     len(rels),
		})
	}
	return recordRelation(ctx, svc, projectID, req, "relation")
}

// recordRelation is the shared implementation for mem_judge and mem_compare.
// The only difference is the name of the verdict parameter.
func recordRelation(ctx context.Context, svc *service.Service, projectID string, req mcplib.CallToolRequest, verdictParam string) (*mcplib.CallToolResult, error) {
	srcID, err := req.RequireFloat("src_id")
	if err != nil {
		return toolError(err)
	}
	dstID, err := req.RequireFloat("dst_id")
	if err != nil {
		return toolError(err)
	}
	verdict := strings.TrimSpace(req.GetString(verdictParam, ""))
	if verdict == "" {
		return toolError(fmt.Errorf("%s is required (%s)", verdictParam, relationsHelp))
	}
	reason := req.GetString("reason", "")
	confidence := req.GetFloat("confidence", 0)
	var confPtr *float64
	if confidence > 0 {
		v := float64(confidence)
		confPtr = &v
	}

	if strings.EqualFold(verdict, "not_conflict") {
		removed, err := svc.RemoveRelation(ctx, projectID, int64(srcID), int64(dstID), "conflicts_with")
		if err != nil {
			return toolError(err)
		}
		return JSONResult(map[string]any{
			"verdict": "not_conflict",
			"removed": removed,
			"src_id":  int64(srcID),
			"dst_id":  int64(dstID),
		})
	}

	rel, err := svc.RecordRelation(ctx, projectID, int64(srcID), int64(dstID), verdict, reason, confPtr)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(rel)
}
func memMergeProjectsTool() mcplib.Tool {
	return mcplib.NewTool("mem_merge_projects",
		mcplib.WithDescription("Merge a source project name into the canonical name: copies rows tagged source into the canonical store (idempotent), re-tags them, and records the alias so future writes to the source name land in the canonical store. Admin tool — call only when the user has confirmed the merge."),
		mcplib.WithString("source", mcplib.Required(), mcplib.Description("Project name to merge (the variant being retired)")),
		mcplib.WithString("canonical", mcplib.Required(), mcplib.Description("Project name to merge into (the canonical name)")),
	)
}

func handleMemMergeProjects(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	source := strings.TrimSpace(req.GetString("source", ""))
	canonical := strings.TrimSpace(req.GetString("canonical", ""))
	if source == "" || canonical == "" {
		return toolError(errors.New("source and canonical project names are both required"))
	}
	if source == canonical {
		return JSONResult(map[string]any{"merged": false, "reason": "source and canonical are identical", "rows_moved": 0, "canonical": canonical})
	}
	moved, canonicalResolved, err := svc.MergeProjects(ctx, source, canonical)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"merged":         true,
		"source":         source,
		"canonical":      canonicalResolved,
		"rows_moved":     moved,
		"alias_recorded": true,
	})
}

// ────────────────────────────── Mem: pin / unpin / unify ──────────────────

func memPinTool() mcplib.Tool {
	return mcplib.NewTool("mem_pin",
		mcplib.WithDescription("Pin an observation so it sorts ahead of everything else in mem_context and boosts mem_search ordering. Pinning is local to this device store (not synced); use it for 'sticky' memories you want to surface every session."),
		mcplib.WithNumber("id", mcplib.Required(), mcplib.Description("Observation ID")),
	)
}

func handleMemPin(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	id, err := req.RequireFloat("id")
	if err != nil {
		return toolError(err)
	}
	if err := svc.PinObservation(ctx, projectID, int64(id)); err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"id": int64(id), "pinned": true})
}

func memUnpinTool() mcplib.Tool {
	return mcplib.NewTool("mem_unpin",
		mcplib.WithDescription("Unpin an observation, returning it to normal recency ordering in mem_context and mem_search."),
		mcplib.WithNumber("id", mcplib.Required(), mcplib.Description("Observation ID")),
	)
}

func handleMemUnpin(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	id, err := req.RequireFloat("id")
	if err != nil {
		return toolError(err)
	}
	if err := svc.UnpinObservation(ctx, projectID, int64(id)); err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"id": int64(id), "unpinned": true})
}

func memUnifyTool() mcplib.Tool {
	return mcplib.NewTool("mem_unify",
		mcplib.WithDescription("Consolidate one or more source project stores into a single canonical project: records each source as an alias and copies + re-tags its rows (idempotent). Use it to fold several directory-hash variants of the same repo into one bucket. Admin tool — call only when the user has confirmed the consolidation."),
		mcplib.WithString("canonical", mcplib.Required(), mcplib.Description("Project name to consolidate into (the canonical name)")),
		mcplib.WithString("sources", mcplib.Required(), mcplib.Description("Comma-separated project names to consolidate (the variants being retired)")),
	)
}

func handleMemUnify(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	canonical := strings.TrimSpace(req.GetString("canonical", ""))
	sourcesRaw := strings.TrimSpace(req.GetString("sources", ""))
	if canonical == "" || sourcesRaw == "" {
		return toolError(errors.New("canonical and sources are both required"))
	}
	var sources []string
	for _, s := range strings.Split(sourcesRaw, ",") {
		if t := strings.TrimSpace(s); t != "" {
			sources = append(sources, t)
		}
	}
	moved, err := svc.Unify(ctx, canonical, sources...)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"unified":        true,
		"canonical":      canonical,
		"sources":        sources,
		"rows_moved":     moved,
		"alias_recorded": true,
	})
}

// projectIDFor resolves the store ID for an explicit project name (normalized
// so it matches store-file naming) or, when empty, the CWD-resolved project.
func projectIDFor(svc *service.Service, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name != "" {
		return project.NormalizeID(name), nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return svc.ResolveProject(cwd)
}

func openService() (*service.Service, string, func(), error) {
	svc, err := rootService()
	if err != nil {
		return nil, "", nil, err
	}
	h, cleanup, err := svc.OpenForCWD()
	if err != nil {
		return nil, "", nil, err
	}
	return svc, h.ProjectID(), cleanup, nil
}

func rootService() (*service.Service, error) {
	if svc != nil {
		return svc, nil
	}
	dataDir, err := service.DefaultDataDir()
	if err != nil {
		return nil, err
	}
	svc = service.New(dataDir)
	return svc, nil
}

// SetService overrides the service used by all tool handlers. Test hook and
// for embedding the MCP server with a caller-provided store (e.g. a shared
// SQLite directory).
func SetService(s *service.Service) {
	svc = s
}

func toolError(err error) (*mcplib.CallToolResult, error) {
	return mcplib.NewToolResultError(err.Error()), nil
}

func observationDTOs(obs []memory.Observation) []map[string]any {
	out := make([]map[string]any, len(obs))
	for i, o := range obs {
		out[i] = observationDTO(o)
	}
	return out
}

func observationDTO(o memory.Observation) map[string]any {
	m := map[string]any{
		"id":              o.ID,
		"session_id":      o.SessionID,
		"type":            o.Type,
		"title":           o.Title,
		"content":         o.Content,
		"project":         o.Project,
		"scope":           o.Scope,
		"normalized_hash": o.NormalizedHash,
		"revision_count":  o.RevisionCount,
		"created_at":      o.CreatedAt,
		"updated_at":      o.UpdatedAt,
	}
	if o.TopicKey != "" {
		m["topic_key"] = o.TopicKey
	}
	if o.Source != "" {
		m["source"] = o.Source
	}
	if o.PromptID != nil {
		m["prompt_id"] = *o.PromptID
	}
	return m
}

func sessionDTOs(sessions []memory.Session) []map[string]any {
	out := make([]map[string]any, len(sessions))
	for i, s := range sessions {
		out[i] = map[string]any{
			"id":         s.ID,
			"project":    s.Project,
			"directory":  s.Directory,
			"started_at": s.StartedAt,
			"ended_at":   s.EndedAt,
			"summary":    s.Summary,
			"status":     s.Status,
		}
	}
	return out
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func suggestTopicKey(typ, title, content string) string {
	family := topicFamily(typ)
	segment := slugSegment(title)
	if segment == "" {
		segment = slugSegment(content)
	}
	if segment == "" {
		segment = "untitled"
	}
	return family + "/" + segment
}

func topicFamily(typ string) string {
	normalized := strings.ToLower(strings.TrimSpace(typ))
	switch normalized {
	case "architecture", "decision", "bugfix", "bug", "pattern", "config", "discovery", "learning", "lesson", "preference", "convention":
		if normalized == "lesson" {
			return "learning"
		}
		return normalized
	default:
		return "topic"
	}
}

func slugSegment(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if idx := strings.Index(s, "["); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
