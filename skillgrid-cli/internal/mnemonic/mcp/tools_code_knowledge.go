package mcp

import (
	"context"
	"database/sql"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/knowledge"
)

// registerKnowledgeTools registers the knowledge-graph query MCP tools
// (code_docs / code_configs / code_sql_schema / code_sql_access). They are
// distinct code_* tools — none clashes with the 005/008/010 baseline tools —
// and are part of the narrow code_* menu (unlisted by default; re-enabled via
// CodeToolsEnvVar). They are query-only over the 015 knowledge tables: no
// per-query traversal.
func registerKnowledgeTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{codeDocsTool(), handleCodeDocs},
		{codeConfigsTool(), handleCodeConfigs},
		{codeSQLSchemaTool(), handleCodeSQLSchema},
		{codeSQLAccessTool(), handleCodeSQLAccess},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func codeDocsTool() mcplib.Tool {
	return mcplib.NewTool("code_docs",
		mcplib.WithDescription("Return indexed markdown docs (doc nodes) and their references edges between docs. Each reference carries a confidence label (EXTRACTED for a markdown link, INFERRED for a wikilink). Query by a doc path or list all."),
		mcplib.WithString("path", mcplib.Description("A doc path to narrow to (omit to list all docs)")),
	)
}

func codeConfigsTool() mcplib.Tool {
	return mcplib.NewTool("code_configs",
		mcplib.WithDescription("Return indexed config files (config nodes) and their configures edges to the code they configure. Each configures edge carries a confidence label (EXTRACTED for an explicit symbol reference, INFERRED for a conventional one, AMBIGUOUS for an unresolvable reference that is kept, not dropped). Query by a config path or list all."),
		mcplib.WithString("path", mcplib.Description("A config path to narrow to (omit to list all configs)")),
	)
}

func codeSQLSchemaTool() mcplib.Tool {
	return mcplib.NewTool("code_sql_schema",
		mcplib.WithDescription("Return indexed SQL schema nodes (tables + columns parsed from DDL). Query by a table name or list all."),
		mcplib.WithString("table", mcplib.Description("A table name to narrow to (omit to list all)")),
	)
}

func codeSQLAccessTool() mcplib.Tool {
	return mcplib.NewTool("code_sql_access",
		mcplib.WithDescription("Return the reads/writes edges between code symbols and SQL tables (parsed from DML). Each edge carries a confidence label (EXTRACTED — the SQL statement names the table). Query by a table name or list all."),
		mcplib.WithString("table", mcplib.Description("A table name to narrow to (omit to list all)")),
	)
}

func handleCodeDocs(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	path, _ := req.RequireString("path")
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	out, err := mcpKnowledgeDocs(h.Store().DB, path)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(out)
}

func handleCodeConfigs(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	path, _ := req.RequireString("path")
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	out, err := mcpKnowledgeConfigs(h.Store().DB, path)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(out)
}

func handleCodeSQLSchema(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	table, _ := req.RequireString("table")
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	out, err := mcpKnowledgeSQLSchema(h.Store().DB, table)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(out)
}

func handleCodeSQLAccess(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	table, _ := req.RequireString("table")
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	out, err := mcpKnowledgeSQLAccess(h.Store().DB, table)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(out)
}

// mcpKnowledgeDocs is the single-open backing for code_docs: return doc nodes
// (narrowed by path when given) and their references edges.
func mcpKnowledgeDocs(db *sql.DB, path string) (map[string]any, error) {
	where := ""
	var args []any
	if path != "" {
		where = " WHERE dn.path = ?"
		args = append(args, path)
	}
	rows, err := db.Query(`
		SELECT dn.path, dn.title, COALESCE(e.to_name,''), COALESCE(e.target_path,''), COALESCE(e.confidence,''), COALESCE(e.line,0)
		FROM doc_nodes dn
		LEFT JOIN edges e ON e.from_id = dn.id AND e.kind = 'references'`+where, args...)
	if err != nil {
		if isNoTable(err) {
			return map[string]any{"docs": []map[string]any{}}, nil
		}
		return nil, err
	}
	defer rows.Close()
	docs := map[string]map[string]any{}
	var order []string
	for rows.Next() {
		var p, title, toName, target, conf string
		var line int
		if err := rows.Scan(&p, &title, &toName, &target, &conf, &line); err != nil {
			return nil, err
		}
		d, ok := docs[p]
		if !ok {
			d = map[string]any{"path": p, "title": title, "references": []map[string]any{}}
			docs[p] = d
			order = append(order, p)
		}
		if toName != "" {
			refs := d["references"].([]map[string]any)
			d["references"] = append(refs, map[string]any{
				"target":     target,
				"confidence": conf,
				"line":       line,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(order))
	for _, p := range order {
		out = append(out, docs[p])
	}
	return map[string]any{"docs": out, "total": len(out)}, nil
}

// mcpKnowledgeConfigs is the single-open backing for code_configs: return
// config nodes (narrowed by path when given) and their configures edges.
func mcpKnowledgeConfigs(db *sql.DB, path string) (map[string]any, error) {
	where := ""
	var args []any
	if path != "" {
		where = " WHERE cn.path = ?"
		args = append(args, path)
	}
	rows, err := db.Query(`
		SELECT cn.path, cn.title, COALESCE(e.to_name,''), COALESCE(s.name,''), COALESCE(e.confidence,''), COALESCE(e.line,0)
		FROM config_nodes cn
		LEFT JOIN edges e ON e.from_id = cn.id AND e.kind = 'configures'
		LEFT JOIN symbols s ON s.id = e.to_id`+where, args...)
	if err != nil {
		if isNoTable(err) {
			return map[string]any{"configs": []map[string]any{}}, nil
		}
		return nil, err
	}
	defer rows.Close()
	configs := map[string]map[string]any{}
	var order []string
	for rows.Next() {
		var p, title, toName, symName, conf string
		var line int
		if err := rows.Scan(&p, &title, &toName, &symName, &conf, &line); err != nil {
			return nil, err
		}
		c, ok := configs[p]
		if !ok {
			c = map[string]any{"path": p, "title": title, "configures": []map[string]any{}}
			configs[p] = c
			order = append(order, p)
		}
		if toName != "" {
			refs := c["configures"].([]map[string]any)
			c["configures"] = append(refs, map[string]any{
				"target":     toName,
				"symbol":     symName,
				"confidence": conf,
				"line":       line,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(order))
	for _, p := range order {
		out = append(out, configs[p])
	}
	return map[string]any{"configs": out, "total": len(out)}, nil
}

// mcpKnowledgeSQLSchema is the single-open backing for code_sql_schema:
// return sql_schema_nodes (narrowed by table when given).
func mcpKnowledgeSQLSchema(db *sql.DB, table string) (map[string]any, error) {
	where := ""
	var args []any
	if table != "" {
		where = " WHERE ss.table_name = ?"
		args = append(args, table)
	}
	rows, err := db.Query(`
		SELECT ss.table_name, ss.column_name, ss.kind, ss.path
		FROM sql_schema_nodes ss`+where+` ORDER BY ss.table_name, ss.kind, ss.column_name`, args...)
	if err != nil {
		if isNoTable(err) {
			return map[string]any{"tables": []map[string]any{}}, nil
		}
		return nil, err
	}
	defer rows.Close()
	tables := map[string]map[string]any{}
	var order []string
	for rows.Next() {
		var tn, col, kind, p string
		if err := rows.Scan(&tn, &col, &kind, &p); err != nil {
			return nil, err
		}
		tbl, ok := tables[tn]
		if !ok {
			tbl = map[string]any{"table": tn, "path": p, "columns": []string{}}
			tables[tn] = tbl
			order = append(order, tn)
		}
		if kind == knowledge.KindColumn && col != "" {
			cols := tbl["columns"].([]string)
			tbl["columns"] = append(cols, col)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(order))
	for _, tn := range order {
		out = append(out, tables[tn])
	}
	return map[string]any{"tables": out, "total": len(out)}, nil
}

// mcpKnowledgeSQLAccess is the single-open backing for code_sql_access:
// return the reads/writes edges (narrowed by table when given).
func mcpKnowledgeSQLAccess(db *sql.DB, table string) (map[string]any, error) {
	where := ""
	var args []any
	if table != "" {
		where = " WHERE e.to_name = ?"
		args = append(args, table)
	}
	rows, err := db.Query(`
		SELECT e.kind, s.name, f.path, e.to_name, e.confidence, e.line
		FROM edges e
		JOIN symbols s ON s.id = e.from_id
		JOIN files f ON f.id = s.file_id
		WHERE e.kind IN ('reads','writes')`+where+` ORDER BY e.to_name, e.kind, s.id`, args...)
	if err != nil {
		if isNoTable(err) {
			return map[string]any{"access": []map[string]any{}}, nil
		}
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var kind, sym, p, tn, conf string
		var line int
		if err := rows.Scan(&kind, &sym, &p, &tn, &conf, &line); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"op":         kind,
			"symbol":     sym,
			"path":       p,
			"table":      tn,
			"confidence": conf,
			"line":       line,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []map[string]any{}
	}
	return map[string]any{"access": out, "total": len(out)}, nil
}

// isNoTable reports whether a query error is "no such table" (the knowledge
// tables are absent when the store predates the 015 migration) — the tools
// return an empty result rather than erroring.
func isNoTable(err error) bool {
	if err == nil {
		return false
	}
	return sql.ErrNoRows != err && containsNoTable(err.Error())
}

func containsNoTable(s string) bool {
	return strings.Contains(s, "no such table") || strings.Contains(s, "NO SUCH TABLE")
}
