package mcp

import (
	"context"
	"database/sql"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/mnemonic/internal/skills"
)

func registerSkillTools(s *server.MCPServer) {
	s.AddTool(writeSkillTool(), handleWriteSkill)
	s.AddTool(listSkillsTool(), handleListSkills)
	s.AddTool(searchSkillsTool(), handleSearchSkills)
	s.AddTool(useSkillTool(), handleUseSkill)
}

func writeSkillTool() mcplib.Tool {
	return mcplib.NewTool("write_skill",
		mcplib.WithDescription("Create (or with overwrite, replace) an Agent Skill in the registry: the code is written to .skillgrid/files/skills/{name}.{ext}, SQL metadata is stored in the skills table, and the FTS index mirrors name/description. overwrite=false (default) rejects a name collision with a clear error; overwrite=true replaces the live skill in place. Soft-deleted names are reused (resurrected)."),
		mcplib.WithString("name", mcplib.Required(), mcplib.Description("Skill name (required): a safe file basename, no path separators")),
		mcplib.WithString("language", mcplib.Required(), mcplib.Description("Language tag, e.g. sh, python, go, typescript — determines the file extension; unknown languages are rejected")),
		mcplib.WithString("description", mcplib.Description("What the skill does (indexed for search)")),
		mcplib.WithString("code", mcplib.Required(), mcplib.Description("Skill code written verbatim to the FS file")),
		mcplib.WithBoolean("overwrite", mcplib.Description("Replace the skill if the name is already taken (default false)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
	)
}

func handleWriteSkill(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	projectID, err := projectIDFor(svc, req.GetString("project", ""))
	if err != nil {
		return toolError(err)
	}
	name, err := req.RequireString("name")
	if err != nil {
		return toolError(err)
	}
	language, err := req.RequireString("language")
	if err != nil {
		return toolError(err)
	}
	code, err := req.RequireString("code")
	if err != nil {
		return toolError(err)
	}
	overwrite := req.GetBool("overwrite", false)

	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	id, err := skills.New(h.Store().DB, h.Root(), projectID).Write(ctx, name, language, req.GetString("description", ""), code, overwrite)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"skill_id": id,
		"name":     name,
		"project":  projectID,
		"event":    "skill_write",
	})
}

func listSkillsTool() mcplib.Tool {
	return mcplib.NewTool("list_skills",
		mcplib.WithDescription("List the Agent Skills registry: every live (non-soft-deleted) skill with its name, language, description, and code path, newest first."),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
	)
}

func handleListSkills(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	projectID, err := projectIDFor(svc, req.GetString("project", ""))
	if err != nil {
		return toolError(err)
	}

	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	out, err := skills.New(h.Store().DB, h.Root(), projectID).List(ctx)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"skills": out,
		"count":  len(out),
	})
}

func searchSkillsTool() mcplib.Tool {
	return mcplib.NewTool("search_skills",
		mcplib.WithDescription("Lexical FTS search over the Agent Skills registry (skills_fts: name + description), bm25-ranked. Soft-deleted skills are excluded by default; include_deleted lifts the filter (audit read)."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search terms (any-term recall)")),
		mcplib.WithNumber("limit", mcplib.Description("Max results (default 20)")),
		mcplib.WithBoolean("include_deleted", mcplib.Description("Include soft-deleted skills (default false)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
	)
}

func handleSearchSkills(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	projectID, err := projectIDFor(svc, req.GetString("project", ""))
	if err != nil {
		return toolError(err)
	}
	query, err := req.RequireString("query")
	if err != nil {
		return toolError(err)
	}
	limit := int(req.GetFloat("limit", 0))
	includeDeleted := req.GetBool("include_deleted", false)

	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	out, err := skills.New(h.Store().DB, h.Root(), projectID).SearchWith(ctx, query, limit, includeDeleted)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"skills": out,
		"count":  len(out),
	})
}

func useSkillTool() mcplib.Tool {
	return mcplib.NewTool("use_skill",
		mcplib.WithDescription("Execute a registered Agent Skill in the sandbox: the skill code runs under its language runner (bash/python/go) with a 10s deadline and 1MB output cap. Returns captured stdout/stderr and exit code. A skill_usage row is logged on success and a session_events trail records the call. Unknown languages, path-escaped code, and soft-deleted skills are rejected before any subprocess is spawned."),
		mcplib.WithString("name", mcplib.Required(), mcplib.Description("Skill name (required)")),
		mcplib.WithString("input", mcplib.Description("Optional input passed to the skill as a trailing argument")),
		mcplib.WithString("session_id", mcplib.Description("Session id for the trail event (defaults to the active session)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
	)
}

func handleUseSkill(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	projectID, err := projectIDFor(svc, req.GetString("project", ""))
	if err != nil {
		return toolError(err)
	}
	name, err := req.RequireString("name")
	if err != nil {
		return toolError(err)
	}
	input := req.GetString("input", "")
	sessionID := req.GetString("session_id", "")

	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	// The store owns the skill_use trail now (ExecuteWith writes it when
	// sessionID is non-empty), mirroring the facts trail convention — the MCP
	// tool no longer logs the event itself.
	res, err := skills.New(h.Store().DB, h.Root(), projectID).ExecuteWith(ctx, name, input, sessionID)
	if err != nil {
		return toolError(err)
	}

	return JSONResult(map[string]any{
		"stdout":    res.Stdout,
		"stderr":    res.Stderr,
		"exit_code": res.ExitCode,
		"timed_out": res.TimedOut,
		"usage_id":  res.UsageID,
		"status":    statusFor(res),
	})
}

func statusFor(res skills.SkillResult) string {
	if res.ExitCode != 0 || res.TimedOut {
		return "failure"
	}
	return "success"
}

// rootDB returns the active test/production store DB handle for direct
// assertions in tests. It delegates to the service's current project store.
func rootDB() *sql.DB {
	svc, err := rootService()
	if err != nil {
		return nil
	}
	h, cleanup, err := svc.OpenForCWD()
	if err != nil {
		cleanup()
		return nil
	}
	defer cleanup()
	return h.Store().DB
}
