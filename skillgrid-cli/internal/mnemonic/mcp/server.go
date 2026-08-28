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

// Start blocks on the stdio MCP loop until the client disconnects.
func Start() error {
	s := server.NewMCPServer(serverName, serverVersion)
	registerMemoryTools(s)
	registerCodeTools(s)
	registerWebTools(s)
	return server.ServeStdio(s)
}

// Server is an MCP server instance with all mem_*/code_*/web_* tools
// registered, ready to be served over stdio.
type Server = server.MCPServer

// NewServer returns an MCP server instance with all tools registered.
func NewServer() *Server {
	s := server.NewMCPServer(serverName, serverVersion)
	registerMemoryTools(s)
	registerCodeTools(s)
	registerWebTools(s)
	return s
}
