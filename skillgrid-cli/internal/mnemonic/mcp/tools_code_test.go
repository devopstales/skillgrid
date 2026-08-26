package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func TestCodeSearchIntegration(t *testing.T) {
	root := t.TempDir()
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)

	const marker = "UniqueMCPCodeMarker789"
	filePath := filepath.Join(root, "sample.go")
	content := "package main\n\nfunc " + marker + "() {}\n"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, root)

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	indexRes, err := handleCodeIndex(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Name: "code_index"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertRawJSONResult(t, indexRes)

	searchRes, err := handleCodeSearch(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{
			Name:      "code_search",
			Arguments: map[string]any{"query": marker},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := assertRawJSONResult(t, searchRes)

	var payload struct {
		Hits []struct {
			Path      string  `json:"path"`
			StartLine int     `json:"start_line"`
			EndLine   int     `json:"end_line"`
			Snippet   string  `json:"snippet"`
			Score     float64 `json:"score"`
		} `json:"hits"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Hits) == 0 {
		t.Fatal("expected at least one hit")
	}
	if payload.Hits[0].Path != "sample.go" {
		t.Fatalf("path=%q, want sample.go", payload.Hits[0].Path)
	}
	if !strings.Contains(payload.Hits[0].Snippet, marker) {
		t.Fatalf("snippet=%q, want to contain %q", payload.Hits[0].Snippet, marker)
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

func assertRawJSONResult(t *testing.T, res *mcplib.CallToolResult) string {
	t.Helper()
	if res == nil {
		t.Fatal("nil result")
	}
	if res.IsError {
		t.Fatalf("tool error: %+v", res.Content)
	}
	if len(res.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(res.Content))
	}
	tc, ok := res.Content[0].(mcplib.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	if !IsRawJSON(tc.Text) {
		t.Fatalf("expected raw JSON, got %q", tc.Text)
	}
	return tc.Text
}
