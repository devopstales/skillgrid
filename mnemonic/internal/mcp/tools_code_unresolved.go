package mcp

import (
	"context"
	"database/sql"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerUnresolvedTools registers the unresolved-ref query tool
// (code_unresolved_refs). It is a distinct code_* tool — none clashes with the
// 005/013 baseline — and is part of the narrow code_* menu (unlisted by
// default; re-enabled via CodeToolsEnvVar). Query-only: it reads the 034
// unresolved_refs table and never fabricates a row the index did not record.
func registerUnresolvedTools(s *server.MCPServer) {
	s.AddTool(codeUnresolvedRefsTool(), handleCodeUnresolvedRefs)
}

func codeUnresolvedRefsTool() mcplib.Tool {
	return mcplib.NewTool("code_unresolved_refs",
		mcplib.WithDescription("Query unresolved reference drops: route-handler references that failed to resolve at extraction (drop-not-guess). Each row carries the reference name, kind, line, the number of global symbols matched by name, its status, and the file path. Filter by name substring and/or status. Zero rows means no unresolved refs."),
		mcplib.WithString("name", mcplib.Description("Substring match on the reference name")),
		mcplib.WithString("status", mcplib.Description("Status filter (default 'failed')")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum refs to return (default 50, max 500)")),
	)
}

func handleCodeUnresolvedRefs(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_ = ctx
	name, _ := req.RequireString("name")
	status, _ := req.RequireString("status")
	if status == "" {
		status = "failed"
	}
	limit := int(req.GetFloat("limit", 50))
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	out, err := mcpCodeUnresolvedRefs(h.Store().DB, name, status, limit)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(out)
}

// mcpCodeUnresolvedRefs is the single-open backing for code_unresolved_refs:
// the dropped references matching the name/status filters, one row per ref,
// plus the matching total count.
func mcpCodeUnresolvedRefs(db *sql.DB, name, status string, limit int) (map[string]any, error) {
	var total int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM unresolved_refs ur
		WHERE (? IS NULL OR ur.reference_name LIKE '%' || ? || '%')
		  AND ur.status = ?`,
		nameOrNil(name), name, status,
	).Scan(&total); err != nil {
		return nil, err
	}
	rows, err := db.Query(`
		SELECT ur.reference_name, ur.reference_kind, ur.line, ur.candidates,
		       ur.status, ur.last_seen_at, f.path
		FROM unresolved_refs ur
		JOIN files f ON f.id = ur.file_id
		WHERE (? IS NULL OR ur.reference_name LIKE '%' || ? || '%')
		  AND ur.status = ?
		ORDER BY ur.reference_name, ur.line
		LIMIT ?`,
		nameOrNil(name), name, status, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]any{"total": total, "refs": []map[string]any{}}
	if total == 0 {
		out["message"] = "no unresolved refs"
		return out, nil
	}
	for rows.Next() {
		var (
			refName, kind, statusVal, lastSeen, path string
			line, candidates                         int
		)
		if err := rows.Scan(&refName, &kind, &line, &candidates, &statusVal, &lastSeen, &path); err != nil {
			return nil, err
		}
		out["refs"] = append(out["refs"].([]map[string]any), map[string]any{
			"reference_name": refName,
			"reference_kind": kind,
			"line":           line,
			"candidates":     candidates,
			"status":         statusVal,
			"last_seen_at":   lastSeen,
			"path":           path,
		})
	}
	return out, rows.Err()
}

// nameOrNil returns nil for the empty filter (so `? IS NULL` matches all) and
// the value otherwise (LIKE gets the substring).
func nameOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}
