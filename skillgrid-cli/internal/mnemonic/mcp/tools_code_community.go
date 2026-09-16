package mcp

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// registerCommunityTools registers the community-detection MCP tools
// (code_communities / code_god_nodes / code_explain_community). They are part
// of the narrow code_* menu (unlisted by default; re-enabled via
// CodeToolsEnvVar) and are all exposed by the CLI.
func registerCommunityTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{codeCommunitiesTool(), handleCodeCommunities},
		{codeGodNodesTool(), handleCodeGodNodes},
		{codeExplainCommunityTool(), handleCodeExplainCommunity},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func codeCommunitiesTool() mcplib.Tool {
	return mcplib.NewTool("code_communities",
		mcplib.WithDescription("Detect subsystems in the indexed code graph: Leiden-clusters the symbols/edges into communities, each labeled LLM-free (derived from the community's top god-node names + file paths, never an API call). Seeded and reproducible (content-hash cache key); advisory, not load-bearing. Returns labeled subsystems with their members."),
	)
}

func codeGodNodesTool() mcplib.Tool {
	return mcplib.NewTool("code_god_nodes",
		mcplib.WithDescription("Return the most-connected symbols (god nodes) ranked by degree. exclude_hubs suppresses utility super-hubs (symbols referenced across many files) so the ranking surfaces subsystem concepts rather than glue."),
		mcplib.WithBoolean("exclude_hubs", mcplib.Description("Suppress utility super-hubs from the ranking (default false)")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum god nodes to return (default 20)")),
	)
}

func codeExplainCommunityTool() mcplib.Tool {
	return mcplib.NewTool("code_explain_community",
		mcplib.WithDescription("Explain a detected community: its members (symbol, kind, path, line) and its key entry points (top god nodes). A non-existent community id is rejected with a clear validation error (no invented community)."),
		mcplib.WithNumber("id", mcplib.Required(), mcplib.Description("Community id (from code_communities)")),
	)
}

func handleCodeCommunities(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	out, err := community.Detect(ctx, h.Store().DB, community.Options{})
	if err != nil {
		return toolError(err)
	}
	return JSONResult(out)
}

func handleCodeGodNodes(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	excludeHubs := req.GetBool("exclude_hubs", false)
	limit, err := numericArg(req, "limit", 20)
	if err != nil {
		return toolError(err)
	}
	limitInt := int(limit)
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	gods, err := community.RankGodNodes(h.Store().DB, allSymbolIDs(h.Store()), excludeHubs)
	if err != nil {
		return toolError(err)
	}
	if limitInt > 0 && len(gods) > limitInt {
		gods = gods[:limitInt]
	}
	return JSONResult(map[string]any{
		"god_nodes":      gods,
		"exclude_hubs":   excludeHubs,
		"total_returned": len(gods),
	})
}

func handleCodeExplainCommunity(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	id, err := numericArg(req, "id", -1)
	if err != nil {
		return toolError(err)
	}
	if id < 0 {
		return toolError(fmt.Errorf("code_explain_community: 'id' is required and must be a non-negative community id"))
	}
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	out, err := mcpCodeExplainCommunity(ctx, h, int(id))
	if err != nil {
		return toolError(err)
	}
	if !out.Found {
		return toolError(fmt.Errorf("code_explain_community: %s", out.Reason))
	}
	return JSONResult(out)
}

// communityCohesion reads the stored cohesion for a community (nil when NULL
// — a community built before the 038 pass, or a store predating the column).
func communityCohesion(db *sql.DB, id int) *float64 {
	var coh sql.NullFloat64
	if err := db.QueryRow(`SELECT cohesion FROM community_meta WHERE id = ?`, id).Scan(&coh); err != nil || !coh.Valid {
		return nil
	}
	return &coh.Float64
}

// numericArg returns a numeric argument or a clear validation error when the
// argument is present but not a number (no silent default-inventing).
func numericArg(req mcplib.CallToolRequest, key string, def float64) (float64, error) {
	args := req.GetArguments()
	raw, ok := args[key]
	if !ok {
		return def, nil
	}
	switch v := raw.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, nil
		}
	case bool, nil:
		// present but not numeric (bool / null) → validation error.
	}
	return def, fmt.Errorf("%q must be a number (got %v)", key, raw)
}

// allSymbolIDs returns every symbol id in the store (for god-node ranking).
func allSymbolIDs(st *store.Store) []int64 {
	var ids []int64
	rows, err := st.DB.Query(`SELECT id FROM symbols`)
	if err != nil {
		return ids
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// mcpCodeExplainCommunity is the single-open backing for code_explain_community.
func mcpCodeExplainCommunity(ctx context.Context, h *service.ProjectHandle, id int) (*service.CommunityExplanation, error) {
	db := h.Store().DB
	var maxID int
	if err := db.QueryRow(`SELECT COALESCE(MAX(id), -1) FROM communities`).Scan(&maxID); err != nil {
		return nil, err
	}
	if id < 0 || id > maxID {
		return &service.CommunityExplanation{Found: false, Reason: fmt.Sprintf("community %d not found (0-%d exist)", id, maxID)}, nil
	}
	rows, err := db.Query(`SELECT symbol_id FROM communities WHERE id = ? ORDER BY symbol_id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var memberIDs []int64
	for rows.Next() {
		var mid int64
		if err := rows.Scan(&mid); err != nil {
			return nil, err
		}
		memberIDs = append(memberIDs, mid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(memberIDs) == 0 {
		return &service.CommunityExplanation{Found: false, Reason: fmt.Sprintf("community %d not found", id)}, nil
	}
	label := "community-" + fmt.Sprintf("%d", id)
	_ = db.QueryRow(`SELECT label FROM community_meta WHERE id = ?`, id).Scan(&label)
	out := &service.CommunityExplanation{Found: true, ID: id, Label: label, EntryPts: []community.GodNode{}}
	// Cohesion (038): the internal-edge density graphify surfaces per
	// community. NULL (pre-038 store) leaves the field nil (absent in JSON).
	if c := communityCohesion(db, id); c != nil {
		out.Cohesion = c
	}
	for _, mid := range memberIDs {
		var name, kind, lang, path string
		var startLine, endLine int
		if err := db.QueryRow(`
			SELECT s.name, s.kind, COALESCE(s.language,''), f.path, s.start_line, s.end_line
			FROM symbols s INNER JOIN files f ON f.id = s.file_id WHERE s.id = ?`, mid).
			Scan(&name, &kind, &lang, &path, &startLine, &endLine); err == nil {
			out.Members = append(out.Members, map[string]any{
				"id": mid, "name": name, "kind": kind, "language": lang,
				"path": path, "start_line": startLine, "end_line": endLine,
			})
		}
	}
	gods, err := community.RankGodNodes(db, memberIDs, false)
	if err == nil {
		out.EntryPts = gods
	}
	return out, nil
}
