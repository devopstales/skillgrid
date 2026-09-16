package mcp

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/process"
)

// registerProcessTools registers the precomputed process-flow MCP tools
// (code_processes / code_process). They are distinct code_* tools — none
// clashes with the 005 baseline tools — and are part of the narrow code_*
// menu (unlisted by default; re-enabled via CodeToolsEnvVar). They are
// query-only over the precomputed processes/process_steps tables: no
// per-query traversal.
func registerProcessTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{codeProcessesTool(), handleCodeProcesses},
		{codeProcessTool(), handleCodeProcess},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func codeProcessesTool() mcplib.Tool {
	return mcplib.NewTool("code_processes",
		mcplib.WithDescription("Return the precomputed process flows (execution flows traced from entry points through call chains). One call returns every flow with its named steps, a cross-community flag, and its LLM label (or empty when the LLM is down — never fabricated). No per-query traversal: the flows are precomputed and cached by content-hash."),
		mcplib.WithNumber("limit", mcplib.Description("Maximum processes to return (default 50)")),
	)
}

func codeProcessTool() mcplib.Tool {
	return mcplib.NewTool("code_process",
		mcplib.WithDescription("Return the full step-by-step trace of one named process flow: each hop with its confidence label and the 'stops at <symbol> (<reason>)' note when the trace truncated at a dispatch boundary. A non-existent process name is rejected with a clear validation error (no invented process)."),
		mcplib.WithString("name", mcplib.Required(), mcplib.Description("Process name (from code_processes)")),
	)
}

func handleCodeProcesses(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	limit := int(req.GetFloat("limit", 50))
	if limit <= 0 {
		limit = 50
	}
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	out, err := mcpCodeProcesses(h.Store().DB, limit)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(out)
}

func handleCodeProcess(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return toolError(err)
	}
	if strings.TrimSpace(name) == "" {
		return toolError(fmt.Errorf("code_process: 'name' is required and must be non-empty"))
	}
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	p, err := process.Get(h.Store().DB, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return toolError(fmt.Errorf("code_process: no process named %q (see code_processes)", name))
		}
		return toolError(err)
	}
	return JSONResult(processDTO(p))
}

// mcpCodeProcesses is the single-open backing for code_processes: return the
// stored precomputed flows (precomputed — no per-query traversal), with their
// full step traces.
func mcpCodeProcesses(db *sql.DB, limit int) (map[string]any, error) {
	listed, err := process.List(db)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(listed) > limit {
		listed = listed[:limit]
	}
	dtos := make([]map[string]any, 0, len(listed))
	for _, p := range listed {
		dtos = append(dtos, processDTO(&p))
	}
	return map[string]any{"processes": dtos, "total": len(listed)}, nil
}

// processDTO renders one process with its step-by-step trace (each hop's
// confidence label + the stop note when present).
func processDTO(p *process.Process) map[string]any {
	steps := make([]map[string]any, 0, len(p.Steps))
	for i, s := range p.Steps {
		steps = append(steps, map[string]any{
			"step":       i + 1,
			"symbol":     s.Name,
			"symbol_id":  s.SymbolID,
			"kind":       s.Kind,
			"confidence": s.Confidence,
		})
	}
	out := map[string]any{
		"name":            p.Name,
		"entry_symbol_id": p.EntrySymbolID,
		"entry_kind":      p.EntryKind,
		"cross_community": p.CrossCommunity,
		"content_hash":    p.ContentHash,
		"label":           p.Label,
		"label_status":    p.LabelStatus,
		"steps":           steps,
	}
	if p.Stop != nil {
		out["stop"] = map[string]any{
			"symbol": p.Stop.Symbol,
			"reason": p.Stop.Reason,
			"line":   p.Stop.Line,
		}
	}
	return out
}
