package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected"
)

// registerAffectedTools registers the PR-command tools (code_affected /
// code_rename). They are distinct code_* tools — none clashes with the 005/
// 008/010 baseline — and are part of the narrow code_* menu (unlisted by
// default; re-enabled via CodeToolsEnvVar). code_affected is query-only;
// code_rename is a write (dry_run default true).
func registerAffectedTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{codeAffectedTool(), handleCodeAffected},
		{codeRenameTool(), handleCodeRename},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func codeAffectedTool() mcplib.Tool {
	return mcplib.NewTool("code_affected",
		mcplib.WithDescription("PR command: which tests do I run after this change? Traverses the indexed graph from the changed files out to the affected test files via import + tests_for (and resolved route) edges, depth-capped (default 5). Query-only: it adds no nodes/edges and traverses ONLY resolved edges (step-01 drop policy), so a dropped/ambiguous reference can never inflate the radius. Pass changed as a file list, or stdin (a git diff --name-only list), or base (a git ref: the changed set is derived from git merge-base <base> HEAD + diff, grouped into affected areas with git-history owners)."),
		mcplib.WithArray("changed", mcplib.Description("Changed file paths (repo-relative). Mutually exclusive with stdin/base.")),
		mcplib.WithString("stdin", mcplib.Description("A git diff --name-only-style file list (one path per line). Mutually exclusive with changed/base.")),
		mcplib.WithString("base", mcplib.Description("A git ref: derive the changed set from git merge-base <ref> HEAD + diff, and group the result into affected areas + git-history owners. Mutually exclusive with changed/stdin.")),
		mcplib.WithNumber("depth", mcplib.Description("Bound the traversal depth (default 5)")),
		mcplib.WithString("filter", mcplib.Description("Restrict reported test files to a substring/segment match")),
		mcplib.WithBoolean("json", mcplib.Description("Emit machine-readable JSON")),
		mcplib.WithBoolean("quiet", mcplib.Description("Print only affected test file paths")),
	)
}

func codeRenameTool() mcplib.Tool {
	return mcplib.NewTool("code_rename",
		mcplib.WithDescription("PR command: what does renaming this symbol touch? Resolves the target via 005 disambiguation (ambiguous -> ranked candidate list, no silent pick) and splits the plan into graph edits (high-confidence: definition + typed references from the edges table) and text-search edits (lower-confidence: string matches, flagged review carefully). Every edit carries a confidence label. dry_run defaults true (returns the plan without writing); apply=true edits ONLY the files in the plan (no commit/push)."),
		mcplib.WithString("old", mcplib.Required(), mcplib.Description("Current symbol name")),
		mcplib.WithString("new", mcplib.Required(), mcplib.Description("New symbol name")),
		mcplib.WithString("file", mcplib.Description("Narrow disambiguation to a file path")),
		mcplib.WithString("uid", mcplib.Description("Narrow disambiguation to a symbol UID")),
		mcplib.WithString("kind", mcplib.Description("Narrow disambiguation to a symbol kind")),
		mcplib.WithBoolean("dry_run", mcplib.Description("Return the plan without writing (default true)")),
		mcplib.WithBoolean("apply", mcplib.Description("Write the planned edits (only listed files; no commit/push)")),
	)
}

// stringArg returns a string argument or "" when absent (never errors: the
// tool validates presence itself).
func stringArg(req mcplib.CallToolRequest, key string) string {
	args := req.GetArguments()
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

func handleCodeAffected(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	changedRaw := req.GetArguments()["changed"]
	changed := []string{}
	if arr, ok := changedRaw.([]any); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				changed = append(changed, strings.TrimSpace(s))
			}
		}
	}
	stdin := stringArg(req, "stdin")
	baseRef := stringArg(req, "base")
	if stdin != "" && baseRef != "" {
		return toolError(fmt.Errorf("code_affected: stdin and base are mutually exclusive"))
	}
	if len(changed) > 0 && (stdin != "" || baseRef != "") {
		return toolError(fmt.Errorf("code_affected: changed is mutually exclusive with stdin/base"))
	}
	if len(changed) == 0 && stdin == "" && baseRef == "" {
		return toolError(fmt.Errorf("code_affected: provide changed, stdin, or base"))
	}
	depth := int(req.GetFloat("depth", 0))
	filter := stringArg(req, "filter")

	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	var out any
	if baseRef != "" {
		cwd, err := os.Getwd()
		if err != nil {
			return toolError(err)
		}
		b, err := affected.AffectedBase(ctx, h.Store().DB, cwd, baseRef, affected.Options{Depth: depth, Filter: filter})
		if err != nil {
			return toolError(err)
		}
		out = b
	} else {
		var list []string
		if stdin != "" {
			list, err = affected.ParseFileList(stdin)
			if err != nil {
				return toolError(err)
			}
		} else {
			list = changed
		}
		res, err := affected.Affected(ctx, h.Store().DB, affected.Options{Changed: list, Depth: depth, Filter: filter})
		if err != nil {
			return toolError(err)
		}
		out = res
	}
	return JSONResult(out)
}

func handleCodeRename(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	old, err := req.RequireString("old")
	if err != nil {
		return toolError(fmt.Errorf("code_rename: %v", err))
	}
	newName, err := req.RequireString("new")
	if err != nil {
		return toolError(fmt.Errorf("code_rename: %v", err))
	}
	apply := req.GetBool("apply", false)
	dryRun := !apply
	if req.GetArguments()["dry_run"] != nil {
		if b, ok := req.GetArguments()["dry_run"].(bool); ok {
			dryRun = b
		}
	}
	opts := affected.RenameOptions{
		Old: old, New: newName,
		File: stringArg(req, "file"), UID: stringArg(req, "uid"), Kind: stringArg(req, "kind"),
		DryRun: dryRun, Apply: apply,
	}

	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	plan, err := affected.Rename(ctx, h.Store().DB, opts)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(plan)
}
