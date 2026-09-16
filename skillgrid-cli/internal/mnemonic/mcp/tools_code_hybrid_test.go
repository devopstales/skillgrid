package mcp

import (
	"context"
	"encoding/json"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// TestHybridToolsDistinct covers @step-04 (threat: Mnemonic tool surface):
// code_hybrid_search registers as a distinct code_* tool, never clashing with
// memory semantic_search; the code_search name + required query schema stays
// stable; and bad args (missing required query) on the new tools are rejected
// clearly.
func TestHybridToolsDistinct(t *testing.T) {
	s := NewServer()
	tools := s.ListTools()

	for _, name := range []string{
		"code_hybrid_search", "code_semantic_search", "code_embedding_status",
	} {
		if _, ok := tools[name]; !ok {
			t.Errorf("expected tool %q to be registered", name)
		}
	}
	// Distinct from memory semantic_search.
	if _, ok := tools["semantic_search"]; !ok {
		t.Error("memory semantic_search should still be registered")
	}
	if _, ok := tools["code_hybrid_search"]; !ok {
		t.Error("code_hybrid_search must be distinct from semantic_search")
	}

	// code_search name + required query schema still stable.
	assertCodeToolStable(t, codeSearchTool(), "code_search", []string{"query"})

	// Bad args: missing required query on code_hybrid_search → clear error.
	SetService(service.New(t.TempDir()))
	req := mcplib.CallToolRequest{}
	req.Params.Name = "code_hybrid_search"
	req.Params.Arguments = map[string]any{}
	res, err := handleCodeHybridSearch(context.Background(), req)
	if err != nil {
		t.Fatalf("handleCodeHybridSearch dispatch: %v", err)
	}
	if !res.IsError {
		t.Errorf("code_hybrid_search with missing 'query' should be an error, got: %v", res.Content)
	}

	// Bad args: missing required query on code_semantic_search → clear error.
	req2 := mcplib.CallToolRequest{}
	req2.Params.Name = "code_semantic_search"
	req2.Params.Arguments = map[string]any{}
	res2, err2 := handleCodeSemanticSearch(context.Background(), req2)
	if err2 != nil {
		t.Fatalf("handleCodeSemanticSearch dispatch: %v", err2)
	}
	if !res2.IsError {
		t.Errorf("code_semantic_search with missing 'query' should be an error, got: %v", res2.Content)
	}

	// code_embedding_status requires no args and returns JSON.
	SetService(service.New(t.TempDir()))
	req3 := mcplib.CallToolRequest{}
	req3.Params.Name = "code_embedding_status"
	req3.Params.Arguments = map[string]any{}
	res3, err3 := handleCodeEmbeddingStatus(context.Background(), req3)
	if err3 != nil {
		t.Fatalf("handleCodeEmbeddingStatus dispatch: %v", err3)
	}
	if res3.IsError {
		t.Errorf("code_embedding_status with no args should not error, got: %v", res3.Content)
	}
	text := jsonResultText(t, res3)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("embedding status output not JSON: %v (text: %s)", err, text)
	}
}
