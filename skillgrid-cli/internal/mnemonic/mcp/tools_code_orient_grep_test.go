package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// jsonResultText extracts the first text content from a tool result.
func jsonResultText(t *testing.T, res *mcplib.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatalf("tool result has no content")
	}
	if tc, ok := res.Content[0].(mcplib.TextContent); ok {
		return tc.Text
	}
	t.Fatalf("tool result content[0] is %T, not TextContent", res.Content[0])
	return ""
}

func writeFileUnder(root, name, content string) error {
	p := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(content), 0o644)
}

// TestOrientGrepToolsRegistered covers @step-02 failure (threat: Mnemonic tool
// surface): the Tier-1 orientation tools and structural code_grep register
// alongside the existing four code_* tools, which stay registered.
func TestOrientGrepToolsRegistered(t *testing.T) {
	s := NewServer()
	tools := s.ListTools()
	for _, name := range []string{
		// existing four code_* stay registered
		"code_status", "code_index", "code_search", "code_read",
		// new Tier-1 orientation tools
		"code_orient", "code_signature", "code_file_toc", "code_rationale",
		// new structural grep
		"code_grep",
	} {
		if _, ok := tools[name]; !ok {
			t.Errorf("expected tool %q to be registered", name)
		}
	}
	// All new code_* tools are distinct from memory semantic_search.
	if _, ok := tools["semantic_search"]; !ok {
		t.Errorf("memory semantic_search should remain registered")
	}
	for _, name := range []string{"code_orient", "code_grep"} {
		if name == "semantic_search" {
			t.Errorf("new tool %q clashes with memory semantic_search", name)
		}
	}
}

// TestCodeGrepIndexFree covers @step-02 happy: code_grep runs index-free (no
// store required). A grep over a temp dir of source returns matches without an
// indexed store.
func TestCodeGrepIndexFree(t *testing.T) {
	dataDir := t.TempDir()
	SetService(service.New(dataDir)) // fresh, empty data dir — no index built

	root := t.TempDir()
	writeFixture := func(name, content string) {
		if err := writeFileUnder(root, name, content); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	writeFixture("b.py", "def foo(a, b):\n    return a+b\n\ndef bar():\n    return foo(1, 2)\n")
	writeFixture("c.txt", "def notpython()\n")

	req := mcplib.CallToolRequest{}
	req.Params.Name = "code_grep"
	req.Params.Arguments = map[string]any{
		"pattern": `(function_definition) \fn`,
		"path":    root,
	}
	res, err := handleCodeGrep(context.Background(), req)
	if err != nil {
		t.Fatalf("code_grep: %v", err)
	}
	if res.IsError {
		t.Fatalf("code_grep returned an error: %v", res.Content)
	}
	text := jsonResultText(t, res)
	// It matched the python file, not the unknown c.txt.
	if !strings.Contains(text, "b.py") {
		t.Errorf("expected a b.py match in the index-free grep result: %s", text)
	}
	if strings.Contains(text, "c.txt") {
		t.Errorf("unknown file c.txt should be skipped: %s", text)
	}
}

// TestOrientGrepBadArgsRejected covers @step-02 failure: bad/missing args on
// the new orient + grep tools are rejected clearly (no invented defaults that
// invent hits).
func TestOrientGrepBadArgsRejected(t *testing.T) {
	dataDir := t.TempDir()
	SetService(service.New(dataDir))

	// code_orient with a missing `symbol` arg.
	req := mcplib.CallToolRequest{}
	req.Params.Name = "code_orient"
	req.Params.Arguments = map[string]any{}
	res, err := handleCodeOrient(context.Background(), req)
	if err != nil {
		t.Fatalf("code_orient dispatch: %v", err)
	}
	if !res.IsError {
		t.Errorf("code_orient with missing 'symbol' should be an error, got: %v", res.Content)
	}

	// code_grep with a missing `pattern` arg.
	req2 := mcplib.CallToolRequest{}
	req2.Params.Name = "code_grep"
	req2.Params.Arguments = map[string]any{}
	res2, err := handleCodeGrep(context.Background(), req2)
	if err != nil {
		t.Fatalf("code_grep dispatch: %v", err)
	}
	if !res2.IsError {
		t.Errorf("code_grep with missing 'pattern' should be an error, got: %v", res2.Content)
	}

	// code_grep with a structurally invalid pattern aborts clearly.
	req3 := mcplib.CallToolRequest{}
	req3.Params.Name = "code_grep"
	req3.Params.Arguments = map[string]any{"pattern": `\\`, "path": t.TempDir()}
	res3, err := handleCodeGrep(context.Background(), req3)
	if err != nil {
		t.Fatalf("code_grep dispatch: %v", err)
	}
	if !res3.IsError {
		t.Errorf("code_grep with a dangling-backslash pattern should be an error, got: %v", res3.Content)
	}
}
