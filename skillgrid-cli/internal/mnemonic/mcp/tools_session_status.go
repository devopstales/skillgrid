package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/relay"
)

// sessionStatusTool is the session_status MCP tool (change 006, step 03). It
// reports the handoff count for the current project plus the last known
// cost/context ONLY when the caller supplies them (optional
// context_usage_percent? / cost_usd?). It never invents cost or context it does
// not have. A project with no handoffs yet reports zero counts (warn +
// continue, not a crash).
func sessionStatusTool() mcplib.Tool {
	return mcplib.NewTool("session_status",
		mcplib.WithDescription("Report the session relay status for the current project: the handoff count from session_handoffs plus the last known cost/context when supplied. context_usage_percent and cost_usd are OPTIONAL caller-supplied stats — when omitted the relay does not invent them (they are absent from the result). With no handoffs yet the count is 0 (warn + continue, not an error)."),
		mcplib.WithNumber("context_usage_percent", mcplib.Description("Optional: the caller-supplied context usage percent (0-100). Omit to leave it out of the result.")),
		mcplib.WithNumber("cost_usd", mcplib.Description("Optional: the caller-supplied last known cost in USD. Omit to leave it out of the result.")),
	)
}

func handleSessionStatus(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	var stats *relay.Stats
	ctxPct := req.GetFloat("context_usage_percent", -1)
	costUSD := req.GetFloat("cost_usd", -1)
	if ctxPct >= 0 || costUSD >= 0 {
		stats = &relay.Stats{}
		if ctxPct >= 0 {
			v := ctxPct
			stats.ContextUsagePercent = &v
		}
		if costUSD >= 0 {
			v := costUSD
			stats.CostUSD = &v
		}
	}

	st, err := relay.Status(ctx, h.Store().DB, h.ProjectID(), stats)
	if err != nil {
		return toolError(err)
	}

	out := map[string]any{
		"handoff_count": st.HandoffCount,
	}
	if st.ContextUsagePercent != nil {
		out["context_usage_percent"] = *st.ContextUsagePercent
	}
	if st.CostUSD != nil {
		out["cost_usd"] = *st.CostUSD
	}
	return JSONResult(out)
}

// knowledgeCompactTool is the thin knowledge_compact MCP tool (change 006,
// step 03). It refreshes .cleave/KNOWLEDGE.md from the handoff inputs / session
// notes ONLY — it has no Fact Memory dependency, so it succeeds on a session
// with handoff inputs but no Fact Memory. Empty/missing knowledge inputs
// yield a minimal KNOWLEDGE.md (warn + continue, not an error).
func knowledgeCompactTool() mcplib.Tool {
	return mcplib.NewTool("knowledge_compact",
		mcplib.WithDescription("Thin-refresh .cleave/KNOWLEDGE.md from the handoff inputs / session notes ONLY (no Fact Memory dependency). Succeeds with no Fact Memory. With empty/missing knowledge inputs it writes an empty or minimal KNOWLEDGE.md (warn + continue, not an error). Returns the knowledge_path of the refreshed file."),
	)
}

func handleKnowledgeCompact(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	res, err := relay.CompactKnowledge(ctx, h.Store().DB, h.ProjectID(), h.Root())
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"knowledge_path": res.KnowledgePath,
	})
}
