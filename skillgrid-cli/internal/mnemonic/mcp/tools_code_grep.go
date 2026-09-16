package mcp

import (
	"context"
	"os"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerGrepTools registers the structural code_grep MCP tool. It is
// index-free: no store or embeddings are required; it walks files under a
// directory and matches a by-example structural pattern against the syntax
// tree, per language, skipping unknown files.
func registerGrepTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{codeGrepTool(), handleCodeGrep},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func codeGrepTool() mcplib.Tool {
	return mcplib.NewTool("code_grep",
		mcplib.WithDescription("Structural search by example: match a by-example pattern with metavariables (\\name = named capture, \\(ARGS*) = argument run, \\* / \\_ = anonymous node) against the gotreesitter syntax tree of files under a directory. Index-free (no store/embeddings required); unknown files are skipped; a pattern invalid for a language skips that language with a note and matches the rest. Use to find the shape of code (every function def, every foo(...) call) without building an index."),
		mcplib.WithString("pattern", mcplib.Required(), mcplib.Description("By-example structural pattern (e.g. (function_declaration) \\fn or (call_expression) \\c \\(ARGS*))")),
		mcplib.WithString("path", mcplib.Description("Directory to search (defaults to the current working directory)")),
	)
}

func handleCodeGrep(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	pattern, err := req.RequireString("pattern")
	if err != nil {
		return toolError(err)
	}
	path, _ := req.RequireString("path")
	if path == "" {
		path, err = os.Getwd()
		if err != nil {
			return toolError(err)
		}
	}
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	res, err := svc.CodeGrep(ctx, path, pattern)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(res)
}
