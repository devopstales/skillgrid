package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// InvokeMemSave runs the mem_save tool handler (integration tests).
func InvokeMemSave(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	return handleMemSave(ctx, req)
}
