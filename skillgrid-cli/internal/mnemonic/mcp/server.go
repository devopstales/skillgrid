package mcp

import (
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
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
// guidance (steer to the composite primary tool, don't re-grep) and the
// tool filter that hides the narrow code_* menu tools by default.
func serverOptions() []server.ServerOption {
	return []server.ServerOption{
		server.WithInstructions(exploreInitializeGuidance()),
		server.WithToolFilter(applyCodeToolFilter()),
	}
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
	return server.ServeStdio(s)
}

// Server is an MCP server instance with all mem_*/code_*/web_*/team_* tools
// registered, ready to be served over stdio.
type Server = server.MCPServer

// NewServer returns an MCP server instance with all tools registered.
func NewServer() *Server {
	s := server.NewMCPServer(serverName, serverVersion, serverOptions()...)
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
	return s
}
