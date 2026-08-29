package mcp

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
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
		{memSessionStartTool(), handleMemSessionStart},
		{memSessionEndTool(), handleMemSessionEnd},
		{memSessionSummaryTool(), handleMemSessionSummary},
		{memCapturePassiveTool(), handleMemCapturePassive},
		{memSessionSetTitleTool(), handleMemSessionSetTitle},
		{memSuggestTopicKeyTool(), handleMemSuggestTopicKey},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func memSaveTool() mcplib.Tool {
	return mcplib.NewTool("mem_save",
		mcplib.WithDescription("Save a curated observation to persistent memory. Use structured content with **What**, **Why**, **Where**, and **Learned** sections. Reuse topic_key to upsert evolving topics."),
		mcplib.WithString("title", mcplib.Required(), mcplib.Description("Short searchable title (verb + what)")),
		mcplib.WithString("type", mcplib.Required(), mcplib.Description("Observation type: decision, architecture, bugfix, pattern, config, discovery, learning, preference, convention")),
		mcplib.WithString("content", mcplib.Required(), mcplib.Description("Structured body with What/Why/Where/Learned sections")),
		mcplib.WithString("session_id", mcplib.Required(), mcplib.Description("Active session ID from mem_session_start")),
		mcplib.WithString("scope", mcplib.Description("Visibility scope: project (default), user, or global")),
		mcplib.WithString("topic_key", mcplib.Description("Stable key for upserts, e.g. architecture/auth-model")),
	)
}

func memSearchTool() mcplib.Tool {
	return mcplib.NewTool("mem_search",
		mcplib.WithDescription("Full-text search over saved observations using FTS5."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search keywords")),
		mcplib.WithString("match_mode", mcplib.Description("Term matching: any (default) or all")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum results (default 20)")),
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
	svc, projectID, cleanup, err := openService()
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

	id, err := svc.SaveObservation(ctx, projectID, service.SaveObservationInput{
		Title:     title,
		Type:      typ,
		Content:   content,
		Scope:     req.GetString("scope", "project"),
		TopicKey:  req.GetString("topic_key", ""),
		SessionID: sessionID,
	})
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"id": id})
}

func handleMemSearch(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
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

	hits, err := svc.SearchObservations(ctx, projectID, query, matchMode, limit)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"observations": observationDTOs(hits)})
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
