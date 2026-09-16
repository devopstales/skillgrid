package mcp

import (
	"context"
	"database/sql"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerRouteTools registers the framework route/navigation query tools
// (code_route / code_navigates). They are distinct code_* tools — none clashes
// with the 005 baseline tools — and are part of the narrow code_* menu
// (unlisted by default; re-enabled via CodeToolsEnvVar). They are query-only
// (code_affected is step 02, not this step).
func registerRouteTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{codeRouteTool(), handleCodeRoute},
		{codeNavigatesTool(), handleCodeNavigates},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func codeRouteTool() mcplib.Tool {
	return mcplib.NewTool("code_route",
		mcplib.WithDescription("Query framework routes: list route nodes (URL pattern + method + framework) and their references edges to handler symbols. A handler name returns the routes that serve it (URL patterns surfaced); a path pattern returns the matching route and its handler. Every references edge carries a confidence label (EXTRACTED for explicit syntax, INFERRED for convention-bound)."),
		mcplib.WithString("handler", mcplib.Description("Handler symbol name: return the routes that reference it (URL patterns)")),
		mcplib.WithString("path", mcplib.Description("URL path pattern: return the matching route node and its handler")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum routes to return (default 20)")),
	)
}

func codeNavigatesTool() mcplib.Tool {
	return mcplib.NewTool("code_navigates",
		mcplib.WithDescription("Query framework navigation (screen -> screen) edges. A from-screen returns the screens it navigates to; a to-screen returns the senders that navigate to it. Literal, programmatic destinations are EXTRACTED; markup-written links (<Link>, <Route>) are INFERRED. Computed/unserved destinations are not fabricated (they stay unresolved and are excluded from blast-radius math)."),
		mcplib.WithString("from", mcplib.Description("Source screen path (e.g. /home): return screens it navigates to")),
		mcplib.WithString("to", mcplib.Description("Destination screen path (e.g. /about): return senders that navigate to it")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum navigations to return (default 20)")),
	)
}

func handleCodeRoute(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	handler, _ := req.RequireString("handler")
	path, _ := req.RequireString("path")
	limit := int(req.GetFloat("limit", 20))

	if handler == "" && path == "" {
		return toolError(fmt.Errorf("code_route: provide 'handler' or 'path' (or both)"))
	}
	if limit <= 0 {
		limit = 20
	}

	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	out, err := mcpCodeRoute(h.Store().DB, handler, path, limit)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(out)
}

func handleCodeNavigates(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	from, _ := req.RequireString("from")
	to, _ := req.RequireString("to")
	limit := int(req.GetFloat("limit", 20))

	if from == "" && to == "" {
		return toolError(fmt.Errorf("code_navigates: provide 'from' or 'to' (or both)"))
	}
	if limit <= 0 {
		limit = 20
	}

	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	out, err := mcpCodeNavigates(h.Store().DB, from, to, limit)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(out)
}

// mcpCodeRoute is the single-open backing for code_route. It returns route
// nodes + their references edges, filtered by handler name and/or path
// pattern. Never empty, never fabricated.
func mcpCodeRoute(db *sql.DB, handler, path string, limit int) (map[string]any, error) {
	query := `
		SELECT r.name, r.kind, r.language, f.path AS file_path,
		       rm.method, rm.path_pattern, rm.framework,
		       h.name AS handler_name, e.confidence
		FROM symbols r
		JOIN files f ON f.id = r.file_id
		LEFT JOIN route_meta rm ON rm.symbol_id = r.id
		LEFT JOIN edges e ON e.kind = 'references' AND e.from_id = r.id
		LEFT JOIN symbols h ON h.id = e.to_id
		WHERE r.kind = 'route'`
	args := []any{}
	if handler != "" {
		query += ` AND h.name = ?`
		args = append(args, handler)
	}
	if path != "" {
		query += ` AND rm.path_pattern = ?`
		args = append(args, path)
	}
	query += ` ORDER BY r.id LIMIT ?`
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]any{"routes": []map[string]any{}}
	for rows.Next() {
		var (
			name, kind, lang, filePath string
			method, pattern, framework string
			handlerName                sql.NullString
			confidence                 sql.NullString
		)
		if err := rows.Scan(&name, &kind, &lang, &filePath, &method, &pattern, &framework, &handlerName, &confidence); err != nil {
			return nil, err
		}
		r := map[string]any{
			"name":       name,
			"framework":  framework,
			"method":     method,
			"path":       pattern,
			"file":       filePath,
		}
		if handlerName.Valid {
			r["handler"] = handlerName.String
		}
		if confidence.Valid {
			r["references_confidence"] = confidence.String
		}
		out["routes"] = append(out["routes"].([]map[string]any), r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// mcpCodeNavigates is the single-open backing for code_navigates. It returns
// navigates edges filtered by from-screen and/or to-screen. Computed/unserved
// destinations are not fabricated (a navigates edge is only present when the
// destination was a literal at extraction).
func mcpCodeNavigates(db *sql.DB, from, to string, limit int) (map[string]any, error) {
	query := `
		SELECT e.to_name AS screen, e.confidence, e.line,
		       f.path AS file_path, s.name AS from_symbol
		FROM edges e
		JOIN files f ON f.id = e.file_id
		JOIN symbols s ON s.id = e.from_id
		WHERE e.kind = 'navigates'`
	args := []any{}
	if from != "" {
		query += ` AND s.name = ?`
		args = append(args, from)
	}
	if to != "" {
		query += ` AND e.to_name = ?`
		args = append(args, to)
	}
	query += ` ORDER BY e.id LIMIT ?`
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]any{"navigations": []map[string]any{}}
	for rows.Next() {
		var screen, conf string
		var line int
		var filePath, fromSym string
		if err := rows.Scan(&screen, &conf, &line, &filePath, &fromSym); err != nil {
			return nil, err
		}
		out["navigations"] = append(out["navigations"].([]map[string]any), map[string]any{
			"screen":     screen,
			"confidence": conf,
			"line":       line,
			"file":       filePath,
			"from":       fromSym,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
