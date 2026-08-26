package mcp

import (
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "skillgrid-mnemonic"
	serverVersion = "0.1.0"
)

// Start blocks on the stdio MCP loop until the client disconnects.
func Start() error {
	s := server.NewMCPServer(serverName, serverVersion)
	registerMemoryTools(s)
	registerCodeTools(s)
	return server.ServeStdio(s)
}
