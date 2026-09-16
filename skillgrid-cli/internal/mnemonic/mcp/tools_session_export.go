package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// HandleSessionResumeForTest exposes the session_resume handler to the CLI
// package's session tests (change 006, step 04) so the "CLI mirrors MCP on the
// same store" proof can drive the MCP path in-process and compare its outcome
// against the real CLI. It is a pure alias of the registered handler — no
// behavior, no new tool surface.
func HandleSessionResumeForTest(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	return handleSessionResume(ctx, req)
}

// HandleSessionStatusForTest exposes the session_status handler to the CLI
// package's session tests (change 006, step 04). It is a pure alias of the
// registered handler — no behavior, no new tool surface.
func HandleSessionStatusForTest(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	return handleSessionStatus(ctx, req)
}
