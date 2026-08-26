package mcp_test

import (
	"strings"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	mcpmem "skillgrid-cli/internal/mnemonic/mcp"
)

func TestValidateRawJSON(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		wantOK bool
	}{
		{"object", `{"id":1}`, true},
		{"array", `[{"id":1}]`, true},
		{"whitespace object", "  \n{\"ok\":true}", true},
		{"prose prefix", "Found 3 results:\n{\"count\":3}", false},
		{"index status preamble", "Index status: {\"files\":0}", false},
		{"plain text", "hello", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOK := mcpmem.IsRawJSON(tt.text)
			if gotOK != tt.wantOK {
				t.Fatalf("IsRawJSON(%q) = %v, want %v", tt.text, gotOK, tt.wantOK)
			}
		})
	}
}

func TestJSONResultNoLeadingProse(t *testing.T) {
	res, err := mcpmem.JSONResult(map[string]any{
		"observations": []map[string]any{{"id": 1, "title": "Test"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(res.Content))
	}

	tc, ok := res.Content[0].(mcplib.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	if !mcpmem.IsRawJSON(tc.Text) {
		t.Fatalf("handler output is not raw JSON: %q", tc.Text)
	}
	if strings.HasPrefix(strings.TrimSpace(tc.Text), "Found") {
		t.Fatalf("unexpected prose in output: %q", tc.Text)
	}
}
