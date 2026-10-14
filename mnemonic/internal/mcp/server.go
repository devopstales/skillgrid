package mcp

import (
	"context"
	"fmt"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

const (
	serverName    = "skillgrid-mnemonic"
	serverVersion = "0.1.0"
)

// svc is the injected service used by all tool handlers. It defaults to the
// data directory from SKILLGRID_MNEMONIC_DATA_DIR / ~/.skillgrid/mnemonic.
var svc *service.Service

// pool is the lazy per-project store pool shared by the MCP server. It is
// initialized on the first query; tests may replace it.
var pool *StorePool

// serverOptions returns the MCP server options: the initialize-injected
// guidance (steer to the composite primary tool, don't re-grep), the
// tool filter that hides the narrow code_* menu tools by default, and
// the unknown-tool suggestion hook (did-you-mean for hallucinated names).
func serverOptions() []server.ServerOption {
	return []server.ServerOption{
		server.WithInstructions(exploreInitializeGuidance()),
		server.WithToolFilter(applyCodeToolFilter()),
	}
}

// applyUnknownToolSuggestion wraps a tool handler so that when the MCP
// library's handleToolCall returns a "tool not found" error for a name
// that is close to a registered tool, the error message is augmented
// with a "did you mean X?" suggestion. This prevents agents from
// hallucinating tool names (e.g. code_scan) and having no recovery
// hint.
func applyUnknownToolSuggestion(next server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		res, err := next(ctx, req)
		if err == nil {
			return res, nil
		}
		msg := err.Error()
		if !strings.Contains(msg, "not found") {
			return res, err
		}
		// Extract the requested tool name from the error message.
		name := extractToolName(msg)
		if name == "" {
			return res, err
		}
		s := server.ServerFromContext(ctx)
		if s == nil {
			return res, err
		}
		tools := s.ListTools()
		if tools == nil {
			return res, err
		}
		names := make([]string, 0, len(tools))
		for n := range tools {
			names = append(names, n)
		}
		if hint := toolSuggestion(name, names); hint != "" {
			return res, fmt.Errorf("%s%s", msg, hint)
		}
		return res, err
	}
}

// extractToolName pulls the tool name from a "tool 'X' not found" error message.
func extractToolName(msg string) string {
	// Format: "tool 'code_scan' not found: ..."
	start := strings.Index(msg, "'")
	if start < 0 {
		return ""
	}
	rest := msg[start+1:]
	end := strings.Index(rest, "'")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// ensurePool lazily creates the shared store pool on first use.
func ensurePool() *StorePool {
	if pool == nil {
		d, _ := service.DefaultDataDir()
		pool = NewStorePool(d, 0, 0)
	}
	return pool
}

// Start blocks on the stdio MCP loop until the client disconnects.
func Start() error {
	s := server.NewMCPServer(serverName, serverVersion, serverOptions()...)
	s.Use(applyUnknownToolSuggestion)
	registerMemoryTools(s)
	registerCodeTools(s)
	registerOrientTools(s)
	registerGrepTools(s)
	registerGraphTools(s)
	registerExploreTools(s)
	registerHybridTools(s)
	registerCommunityTools(s)
	registerRouteTools(s)
	registerUnresolvedTools(s)
	registerProcessTools(s)
	registerKnowledgeTools(s)
	registerWebTools(s)
	registerTeamsTools(s)
	registerRetrievalTools(s)
	registerCompactionTools(s)
	registerAffectedTools(s)
	registerPdgTools(s)
	registerTaintTools(s)
	registerMemoryGovernanceTools(s)
	registerMemoryLayerTools(s)
	registerSessionTools(s)
	registerSessionInjectTools(s)
	registerQueryEventTools(s)
	registerFactTools(s)
	registerSkillTools(s)
	registerMemoryHybridTools(s)
	registerSecondBrainTools(s)
	registerScanTools(s)
	registerDepTools(s)
	return server.ServeStdio(s)
}

// Server is an MCP server instance with all mem_*/code_*/web_*/team_* tools
// registered, ready to be served over stdio.
type Server = server.MCPServer

// NewServer returns an MCP server instance with all tools registered.
func NewServer() *Server {
	s := server.NewMCPServer(serverName, serverVersion, serverOptions()...)
	s.Use(applyUnknownToolSuggestion)
	registerMemoryTools(s)
	registerCodeTools(s)
	registerOrientTools(s)
	registerGrepTools(s)
	registerGraphTools(s)
	registerExploreTools(s)
	registerHybridTools(s)
	registerCommunityTools(s)
	registerRouteTools(s)
	registerUnresolvedTools(s)
	registerProcessTools(s)
	registerKnowledgeTools(s)
	registerWebTools(s)
	registerTeamsTools(s)
	registerRetrievalTools(s)
	registerCompactionTools(s)
	registerAffectedTools(s)
	registerPdgTools(s)
	registerTaintTools(s)
	registerMemoryGovernanceTools(s)
	registerMemoryLayerTools(s)
	registerSessionTools(s)
	registerSessionInjectTools(s)
	registerQueryEventTools(s)
	registerFactTools(s)
	registerSkillTools(s)
	registerMemoryHybridTools(s)
	registerSecondBrainTools(s)
	registerScanTools(s)
	registerDepTools(s)
	return s
}
