package mcp

import (
	"context"
	"database/sql"
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
		mcplib.WithString("owner", mcplib.Description("Optional creating user/agent identity. Blank falls back to the session_id, so the same session that saved can read it back and a different owner is gated by mem_search / mem_get_observation.")),
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
		mcplib.WithString("reader_owner", mcplib.Description("Optional reader identity for per-owner visibility enforcement. Blank means the current session's owner. A private observation is invisible to a different owner until shared via mem_share.")),
		mcplib.WithString("reader_agent", mcplib.Description("Optional reader agent id for restricted/agent ACL grants.")),
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
		mcplib.WithString("reader_owner", mcplib.Description("Optional reader identity for per-owner visibility enforcement. Blank means the current session's owner. A private observation the reader can't see returns not-found (no visibility error leak).")),
		mcplib.WithString("reader_agent", mcplib.Description("Optional reader agent id for restricted/agent ACL grants.")),
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
	_, h, cleanup, err := openService()
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

	res, err := h.Memory().CapturePassive(ctx, memory.PassiveInput{
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
	svc, h, cleanup, err := openService()
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
	projectID := h.ProjectID()
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

	// Reproduce service.SaveObservation's scope normalization ("" → project,
	// "personal" → user) through the handle — the second-open the rewire removes.
	scope := req.GetString("scope", "project")
	if scope == "" {
		scope = "project"
	}
	if scope == "personal" {
		scope = "user"
	}
	// Owner (change 013 step 01): the creating user/agent identity. The MCP
	// tool surfaces an optional `owner`; blank falls back to the session id
	// (the save path's own fallback), so the SAME session that saved can read
	// back and a DIFFERENT owner is gated. This is the read-side mirror that
	// makes per-owner enforcement meaningful end-to-end at the tool boundary.
	owner := strings.TrimSpace(req.GetString("owner", ""))
	if owner == "" {
		owner = sessionID
	}

	id, err := h.Memory().Save(ctx, memory.SaveInput{
		Title:         title,
		Type:          typ,
		Content:       content,
		Scope:         scope,
		TopicKey:      req.GetString("topic_key", ""),
		SessionID:     sessionID,
		CapturePrompt: capturePrompt,
		ProjectName:   projectName,
		ToolName:      req.GetString("tool_name", ""),
		Owner:         owner,
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
	svc, h, cleanup, err := openService()
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
	// Reader identity (change 013 step 01). The per-owner read-enforcement seam
	// (canRead/visibilityFilter) is wired here so a live reader is gated by
	// owner/visibility end-to-end, not just at the service layer. It mirrors
	// the owner the save path uses (session_id fallback): the SAME owner who
	// saved can read; a DIFFERENT owner is gated. Optional `reader_owner` /
	// `reader_agent` name the caller; blank means "this session" (the owner of
	// the session id, resolved from the row).
	readerOwner := strings.TrimSpace(req.GetString("reader_owner", ""))
	readerAgent := strings.TrimSpace(req.GetString("reader_agent", ""))
	if readerOwner == "" {
		readerOwner = req.GetString("session_id", "")
	}

	if allProjects {
		// Budget (013 step 03) applies to the cross-project read too (uniform
		// item cap + char budget + context timeout).
		res, err := applyBudget(h.Memory().Budget(), ctx, func(bctx context.Context) ([]memory.Observation, error) {
			return svc.SearchAllProjects(bctx, query, matchMode, scope, limit)
		})
		if err != nil {
			return toolError(err)
		}
		out := map[string]any{
			"project":      "all",
			"all_projects": true,
			"count":        len(res.Hits),
			"observations": budgetedObservationDTOs(h.Memory().Budget(), res.Hits),
		}
		if res.Truncated {
			out["truncated"] = true
			out["truncation_reason"] = res.Reason
		}
		return JSONResult(out)
	}

	explicitProject := strings.TrimSpace(req.GetString("project", ""))
	projectID := h.ProjectID()
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

	// Scoped + owner-enforced search through the handle (no second open).
	// SearchOwnerScoped applies the visibilityFilter for the current reader, so
	// a private observation is invisible to a different owner until shared.
	//
	// Budget (change 013, step 03): the in-list read is uniformly budgeted —
	// item cap + char budget (explicit "N chars omitted") + context timeout
	// (enforced by a deadline-bound read context, so a slow read is cut, never
	// hung). The full content is pulled on demand via mem_get_observation (the
	// only full-content path) using each hit's id.
	res, err := applyBudget(h.Memory().Budget(), ctx, func(bctx context.Context) ([]memory.Observation, error) {
		return h.Memory().SearchOwnerScoped(bctx, readerOwner, readerAgent, query, matchMode, scope, limit)
	})
	if err != nil {
		return toolError(err)
	}
	out := map[string]any{
		"project":      projectID,
		"observations": budgetedObservationDTOs(h.Memory().Budget(), res.Hits),
		"count":        len(res.Hits),
	}
	if res.Truncated {
		out["truncated"] = true
		out["truncation_reason"] = res.Reason
	}
	if drift != nil {
		out["project_drift"] = drift
	}
	return JSONResult(out)
}

func handleMemContext(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	limit := int(req.GetFloat("limit", 5))
	b := h.Memory().Budget()
	// Budget (013 step 03): the context read is uniformly budgeted — item cap
	// (how many sessions are in-list) + char budget (summaries truncated with an
	// explicit "N chars omitted") + context timeout (enforced by the
	// deadline-bound read context, so a slow read is cut, never hung). The item
	// cap + char budget are the same uniform Apply mem_search uses (finding
	// 03.4); the timeout is enforced on the query via Bound.
	bctx := enforceContextTimeout(b, ctx)
	sessions, err := h.Memory().RecentContext(bctx, limit)
	if err != nil && contextTimedOut(bctx) {
		// A deadline cut surfaces as a truncated partial, not a hard failure.
		sessions = sessions[:0]
	} else if err != nil {
		return toolError(err)
	}
	budgeted, truncated, reason := budgetedSessions(b, sessions)
	if contextTimedOut(bctx) {
		truncated = true
		reason = "timeout"
	}
	out := map[string]any{"sessions": sessionDTOs(budgeted)}
	if truncated {
		out["truncated"] = true
		out["truncation_reason"] = reason
	}
	return JSONResult(out)
}

func handleMemGetObservation(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	id, err := req.RequireFloat("id")
	if err != nil {
		return toolError(err)
	}
	// Reader identity (change 013 step 01). The per-owner read-enforcement seam
	// (canRead) is wired here so a live reader is gated by owner/visibility
	// end-to-end. It mirrors the owner the save path uses (session_id fallback):
	// the SAME owner who saved can read; a DIFFERENT owner is gated. Optional
	// `reader_owner` / `reader_agent` name the caller; blank means "this
	// session" (the owner of the session id, resolved from the row).
	readerOwner := strings.TrimSpace(req.GetString("reader_owner", ""))
	readerAgent := strings.TrimSpace(req.GetString("reader_agent", ""))
	if readerOwner == "" {
		readerOwner = req.GetString("session_id", "")
	}

	// ReadAs enforces canRead for the reader. A reader that can't see the
	// observation gets an "absent" result (not-found shape), not a visibility
	// error leak — consistent with ErrNotFoundForReader.
	obs, err := h.Memory().ReadAs(ctx, readerOwner, int64(id), readerAgent)
	if err != nil {
		if errors.Is(err, memory.ErrNotFoundForReader) {
			return JSONResult(map[string]any{
				"id":            int64(id),
				"not_found":     true,
				"not_found_for_reader": true,
			})
		}
		return toolError(err)
	}
	return JSONResult(observationDTO(obs))
}

func handleMemSessionSetTitle(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
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
	if err := h.Memory().SessionSetTitle(ctx, sessionID, title); err != nil {
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
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return toolError(err)
	}

	if err := h.Memory().SessionEnd(ctx, sessionID, req.GetString("summary", "")); err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"session_id": sessionID, "status": "ended"})
}

func handleMemSessionSummary(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
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

	if err := h.Memory().SessionSummary(ctx, sessionID, summary); err != nil {
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
	_, h, cleanup, err := openService()
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
	b := h.Memory().Budget()
	// Budget (013 step 03): the timeline read is uniformly budgeted — the item
	// cap is the per-direction `limit` (already applied by Timeline), the char
	// budget truncates each in-list snippet (explicit "N chars omitted"), and
	// the context timeout is enforced by the deadline-bound read context (a slow
	// read is cut, never hung). Same uniform Apply as mem_search/mem_context
	// (finding 03.4).
	bctx := enforceContextTimeout(b, ctx)
	tl, err := h.Memory().Timeline(bctx, int64(id), window, limit)
	if err != nil && contextTimedOut(bctx) {
		// A deadline cut surfaces as a truncated partial, not a hard failure.
		tl = memory.Timeline{}
	} else if err != nil {
		return toolError(err)
	}
	truncated := false
	reason := ""
	chars := b.Config().Chars
	truncTimeline := func(e []memory.TimelineEntry) {
		for i := range e {
			if len(e[i].Content) > chars {
				e[i].Content = memory.TruncateWithMarker(e[i].Content, chars)
				truncated = true
				if reason == "" {
					reason = "char-budget"
				}
			}
		}
	}
	truncTimeline(tl.Before)
	truncTimeline(tl.After)
	if contextTimedOut(bctx) {
		truncated = true
		reason = "timeout"
	}
	out := map[string]any{
		"anchor_id": int64(id),
		"before":    tl.Before,
		"after":     tl.After,
	}
	if truncated {
		out["truncated"] = true
		out["truncation_reason"] = reason
	}
	return JSONResult(out)
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
	_, h, cleanup, err := openService()
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
	if err := h.Memory().Update(ctx, int64(id), in); err != nil {
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
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	id, err := req.RequireFloat("id")
	if err != nil {
		return toolError(err)
	}
	hard := req.GetBool("hard", false)
	if err := h.Memory().Delete(ctx, int64(id), hard); err != nil {
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
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	st, err := h.Memory().Status(ctx)
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
	_, h, cleanup, err := openService()
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
	id, err := h.Memory().SavePrompt(ctx, memory.PromptInput{
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
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	out, err := mcpMemoryDoctor(ctx, h)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(out)
}

// mcpMemoryDoctor is the single-open backing for mem_doctor: read-only store
// diagnostics (schema version, WAL, FTS counts/drift, by-type, disk size) run
// directly on the already-open handle store instead of re-opening the project
// inside service.MemoryDoctor.
func mcpMemoryDoctor(ctx context.Context, h *service.ProjectHandle) (service.MemoryDoctor, error) {
	out := service.MemoryDoctor{}
	db := h.Store().DB
	if err := db.QueryRowContext(ctx, `SELECT schema_version FROM index_meta WHERE key='schema_version'`).Scan(&out.SchemaVersion); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return out, fmt.Errorf("schema_version: %w", err)
		}
	}
	if err := db.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&out.WALMode); err != nil {
		return out, fmt.Errorf("journal_mode: %w", err)
	}
	mcpRowCount(ctx, db, "observations", &out.Observations)
	mcpRowCount(ctx, db, "files", &out.Files)
	mcpRowCount(ctx, db, "chunks", &out.Chunks)
	mcpRowCount(ctx, db, "web_cache", &out.WebCache)
	mcpRowCount(ctx, db, "prompts", &out.Prompts)

	ftsObs, ftsChunks, ftsWeb := int64(0), int64(0), int64(0)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM observations_fts`).Scan(&ftsObs)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chunks_fts`).Scan(&ftsChunks)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM web_cache_fts`).Scan(&ftsWeb)
	out.ObservationsFTS = int(ftsObs)
	out.ChunksFTS = int(ftsChunks)
	out.WebCacheFTS = int(ftsWeb)

	rows, err := db.QueryContext(ctx, `
		SELECT type, COUNT(*) FROM observations
		WHERE project = ? AND deleted_at IS NULL GROUP BY type`, h.ProjectID())
	if err == nil {
		byType := map[string]int{}
		for rows.Next() {
			var t string
			var c int
			if rows.Scan(&t, &c) == nil {
				byType[t] = c
			}
		}
		rows.Close()
		out.ByType = byType
	}

	if info, err := os.Stat(h.Store().Path()); err == nil {
		out.DiskSizeBytes = info.Size()
	}
	out.FTSDrift = int(ftsObs) - out.Observations
	out.FTSIntegrityOK = out.FTSDrift >= 0
	return out, nil
}

func mcpRowCount(ctx context.Context, db *sql.DB, table string, out *int) {
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(out)
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
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	action := req.GetString("action", "list")
	switch action {
	case "list":
		limit := int(req.GetFloat("limit", 20))
		due, err := h.Memory().ListReviews(ctx, limit)
		if err != nil {
			return toolError(err)
		}
		return JSONResult(map[string]any{"due": due, "count": len(due)})
	case "mark_reviewed":
		id, err := req.RequireFloat("id")
		if err != nil {
			return toolError(err)
		}
		reviewAfter, err := h.Memory().MarkReviewed(ctx, int64(id))
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
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	return recordRelation(ctx, h, req, "verdict")
}

func handleMemCompare(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
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
		rels, err := h.Memory().RelationsBetween(ctx, int64(srcID), int64(dstID))
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
	return recordRelation(ctx, h, req, "relation")
}

// recordRelation is the shared implementation for mem_judge and mem_compare.
// The only difference is the name of the verdict parameter.
func recordRelation(ctx context.Context, h *service.ProjectHandle, req mcplib.CallToolRequest, verdictParam string) (*mcplib.CallToolResult, error) {
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
		removed, err := h.Memory().RemoveRelation(ctx, int64(srcID), int64(dstID), "conflicts_with")
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

	rel, err := h.Memory().RecordRelation(ctx, int64(srcID), int64(dstID), verdict, reason, confPtr)
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
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	id, err := req.RequireFloat("id")
	if err != nil {
		return toolError(err)
	}
	if err := h.Memory().Pin(ctx, int64(id)); err != nil {
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
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	id, err := req.RequireFloat("id")
	if err != nil {
		return toolError(err)
	}
	if err := h.Memory().Unpin(ctx, int64(id)); err != nil {
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

// openService resolves the CWD project and opens its store exactly once,
// returning the handle plus the root service. Handlers perform their domain
// ops through the handle (h.Memory()/h.Web()/h.Store()) so the project is not
// opened a second time inside a service facade method. svc is kept for
// cross-store ops (drift, all-projects search) and ops with no handle method.
func openService() (*service.Service, *service.ProjectHandle, func(), error) {
	svc, err := rootService()
	if err != nil {
		return nil, nil, nil, err
	}
	h, cleanup, err := svc.OpenForCWD()
	if err != nil {
		return nil, nil, nil, err
	}
	return svc, h, cleanup, nil
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

// inListContent projects an in-list observation's content for a budgeted read
// (mem_search / mem_context / mem_timeline). It applies the char budget with an
// explicit "N chars omitted" marker (never silent). mem_get_observation uses
// the raw observationDTO (no char budget) so it stays the ONLY full-content
// path.
func inListContent(b *memory.Budget, content string) string {
	return memory.TruncateWithMarker(content, b.Config().Chars)
}

// applyBudget runs a budgeted read uniformly (change 013, step 03): it derives
// a deadline-bound context from the budget (so the context timeout is ENFORCED
// on the query — a slow read is cut, never hung) and applies the full uniform
// budget (item cap + char budget + context timeout) to the result. If the read
// was cut by the deadline, the partial it returned is surfaced with
// truncated:true / reason "timeout". Delegates to memory.Budget.ApplyRead so
// the timeout enforcement is identical to the layered CLI path.
func applyBudget(b *memory.Budget, ctx context.Context, read func(context.Context) ([]memory.Observation, error)) (memory.BudgetResult, error) {
	return b.ApplyRead(ctx, read)
}

// budgetedSessions applies the uniform read budget to a session list: the item
// cap bounds how many sessions are in-list, the char budget truncates each
// summary (explicit "N chars omitted"), and the context timeout is enforced by
// the deadline-bound read context (see enforceContextTimeout). This is the
// uniform Apply for mem_context — the same three caps as mem_search (finding
// 03.4). It returns the budgeted session list plus the truncation flags.
func budgetedSessions(b *memory.Budget, sessions []memory.Session) ([]memory.Session, bool, string) {
	capped := sessions
	if len(capped) > b.Config().Items {
		capped = capped[:b.Config().Items]
	}
	truncated := len(sessions) > b.Config().Items
	reason := ""
	if truncated {
		reason = "item-cap"
	}
	out := make([]memory.Session, len(capped))
	copy(out, capped)
	for i := range out {
		if len(out[i].Summary) > b.Config().Chars {
			out[i].Summary = memory.TruncateWithMarker(out[i].Summary, b.Config().Chars)
			truncated = true
			if reason == "" {
				reason = "char-budget"
			}
		}
	}
	return out, truncated, reason
}

// enforceContextTimeout derives the budget's deadline-bound context for a
// non-observation read (mem_context / mem_timeline) so the context timeout is
// enforced on the query (a slow read is cut, never hung). The returned context
// is the same one the char/truncation logic below inspects for the timeout flag.
func enforceContextTimeout(b *memory.Budget, ctx context.Context) context.Context {
	return b.Bound(ctx)
}

// contextTimedOut reports whether a deadline-bound context has lapsed (the read
// ran up against the budget's context timeout and was cut).
func contextTimedOut(ctx context.Context) bool {
	dl, ok := ctx.Deadline()
	return ok && !time.Now().Before(dl)
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
		// Additive governance fields (013 step 01). Private-by-default: a new
		// observation carries visibility=private + its owner; status defaults
		// to active. Additive on existing mem_* responses (never required).
		"owner":           o.Owner,
		"visibility":      o.Visibility,
		"status":          o.Status,
		"retrieval_usage": o.RetrievalUsage,
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

// budgetedObservationDTOs shapes in-list results for a budgeted read: the char
// budget is applied to each snippet (explicit "N chars omitted"), and every
// result carries its full-content fetch id (mem_get_observation is the only
// full-content path).
func budgetedObservationDTOs(b *memory.Budget, obs []memory.Observation) []map[string]any {
	out := make([]map[string]any, len(obs))
	for i, o := range obs {
		o.Content = inListContent(b, o.Content)
		out[i] = observationDTO(o)
	}
	return out
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
