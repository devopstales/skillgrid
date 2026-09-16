package mcp

import (
	"context"
	"errors"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer"
)

// registerMemoryLayerTools registers the additive mem_layers tool (change 013,
// step 02). It is additive on the 005 + step-01 mem_* surface: no existing tool
// is renamed or re-parameterised; the surface grows 77→78.
func registerMemoryLayerTools(s *server.MCPServer) {
	s.AddTool(memLayersTool(), handleMemLayers)
}

// memLayersTool is the layer inspector: it returns the L0→L1→L2→L3 chain for a
// session or topic, with each layer's provenance link to its L0 source. The
// target is a session_id (a session UUID) or a topic_key; a missing/invalid
// argument is rejected clearly.
func memLayersTool() mcplib.Tool {
	return mcplib.NewTool("mem_layers",
		mcplib.WithDescription("Inspect the layered memory for a session or topic: returns the L0→L1→L2→L3 chain (atoms, scenario, persona delta) with each layer's provenance link to its resolvable L0 source. Pass either session_id (a session UUID) or topic_key. A missing or invalid argument is rejected with a clear error."),
		mcplib.WithString("session_id", mcplib.Description("Session UUID to inspect (preferred)")),
		mcplib.WithString("topic_key", mcplib.Description("Topic to inspect (alternative to session_id)")),
	)
}

// handleMemLayers implements mem_layers. It requires exactly one of session_id
// / topic_key (bad or missing args are rejected clearly); it then runs the
// layer read path and returns the chain. A target that resolves to nothing
// returns an empty chain (the L0 is surfaced, no layers), not a fabricated one.
func handleMemLayers(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	sessionID := strings.TrimSpace(req.GetString("session_id", ""))
	topicKey := strings.TrimSpace(req.GetString("topic_key", ""))
	if sessionID == "" && topicKey == "" {
		return toolError(errors.New("mem_layers requires session_id or topic_key"))
	}

	// session_id takes precedence when present; otherwise inspect by topic.
	var chain layer.Chain
	var lerr error
	if sessionID != "" {
		chain, lerr = layer.Inspect(ctx, h.Memory(), sessionID)
	} else {
		chain, lerr = layer.InspectByTopic(ctx, h.Memory(), topicKey)
	}
	if lerr != nil {
		return toolError(lerr)
	}
	return JSONResult(chain)
}
