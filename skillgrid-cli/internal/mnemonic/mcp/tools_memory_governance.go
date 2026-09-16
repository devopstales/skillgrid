package mcp

import (
	"context"
	"fmt"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// registerMemoryGovernanceTools registers the additive mem_share +
// mem_governance tools (change 013, step 01). They are additive on the 005
// mem_* surface: no existing tool is renamed or re-parameterised.
func registerMemoryGovernanceTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{memShareTool(), handleMemShare},
		{memGovernanceTool(), handleMemGovernance},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

// memShareTool is the explicit visibility widen. `acl` is the comma-separated
// grantee list, meaningful for restricted (and agent) visibility; an empty
// restricted grant set is owner-only (surfaced, not an error).
func memShareTool() mcplib.Tool {
	return mcplib.NewTool("mem_share",
		mcplib.WithDescription("Widen an observation's visibility — the only explicit way it leaves `private`. target must be team|restricted|agent (bad targets are rejected, visibility unchanged). For restricted/agent, acl is a comma-separated grantee list (empty = owner-only)."),
		mcplib.WithNumber("id", mcplib.Required(), mcplib.Description("Observation ID")),
		mcplib.WithString("target", mcplib.Required(), mcplib.Description("Visibility to set: team | restricted | agent")),
		mcplib.WithString("acl", mcplib.Description("Comma-separated grantee list for restricted/agent (empty = owner-only)")),
	)
}

// handleMemShare implements mem_share. Bad args (unknown target, empty
// grantee) are rejected with a clear validation error; the observation's
// visibility is unchanged.
func handleMemShare(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	id, err := req.RequireFloat("id")
	if err != nil {
		return toolError(err)
	}
	target := strings.TrimSpace(req.GetString("target", ""))
	if !memory.IsValidVisibility(target) || target == memory.VisibilityPrivate {
		return toolError(fmt.Errorf("invalid target %q (valid: team|restricted|agent)", target))
	}
	var grants []string
	if acl := strings.TrimSpace(req.GetString("acl", "")); acl != "" {
		for _, g := range strings.Split(acl, ",") {
			if t := strings.TrimSpace(g); t != "" {
				grants = append(grants, t)
			}
		}
	}
	if err := h.Memory().Share(ctx, int64(id), memory.ShareInput{Visibility: target, Grants: grants}); err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"id":         int64(id),
		"visibility": target,
		"shared":     true,
	})
}

// memGovernanceTool is the asset-field query: owner, version history, status,
// retrieval usage, visibility (+ ACL grants).
func memGovernanceTool() mcplib.Tool {
	return mcplib.NewTool("mem_governance",
		mcplib.WithDescription("Query a memory asset's governance fields: owner, append-only version history (prior content recoverable), status, retrieval usage count, and visibility (with ACL grants). The latest version is the read path."),
		mcplib.WithNumber("id", mcplib.Required(), mcplib.Description("Observation ID")),
	)
}

func handleMemGovernance(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	id, err := req.RequireFloat("id")
	if err != nil {
		return toolError(err)
	}
	gov, err := h.Memory().Governance(ctx, int64(id))
	if err != nil {
		return toolError(err)
	}
	return JSONResult(gov)
}
