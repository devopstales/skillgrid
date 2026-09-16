package mcp

import (
	"context"
	"encoding/json"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// listToolNames drives the real MCP tools/list handler (applying tool filters)
// and returns the listed tool names.
func listToolNames(t *testing.T, s *mcpserver.MCPServer) map[string]bool {
	t.Helper()
	msg, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]any{},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp := s.HandleMessage(context.Background(), msg)
	if resp == nil {
		t.Fatal("nil response to tools/list")
	}
	var out struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("parse tools/list result: %v (raw %s)", err, string(b))
	}
	names := map[string]bool{}
	for _, tool := range out.Result.Tools {
		names[tool.Name] = true
	}
	return names
}

// callResultText extracts the text content of a CallToolResult for assertions.
func callResultText(t *testing.T, res *mcplib.CallToolResult) string {
	t.Helper()
	if res == nil {
		t.Fatal("nil CallToolResult")
	}
	var out string
	for _, c := range res.Content {
		if tc, ok := c.(mcplib.TextContent); ok {
			out += tc.Text
		}
	}
	return out
}
