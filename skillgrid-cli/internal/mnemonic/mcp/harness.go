package mcp

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/server"
)

// harnessFromClientName maps the MCP initialize clientInfo.name to the harness
// label stored on sessions.agent (cursor, opencode, kilo, …).
func harnessFromClientName(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	switch {
	case n == "":
		return ""
	case strings.HasPrefix(n, "cursor"):
		return "cursor"
	case strings.HasPrefix(n, "opencode"):
		return "opencode"
	case strings.HasPrefix(n, "kilo"):
		return "kilo"
	}
	return n
}

// harnessFromContext reads the connected client's name from the MCP session.
func harnessFromContext(ctx context.Context) string {
	sess := server.ClientSessionFromContext(ctx)
	withInfo, ok := sess.(server.SessionWithClientInfo)
	if !ok || withInfo == nil {
		return ""
	}
	return harnessFromClientName(withInfo.GetClientInfo().Name)
}
