package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeMCPPreservesComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kilo.jsonc")
	original := `{
  // existing comment
  "theme": "dark",
  "mcp": {
    "old-tool": {
      "type": "remote",
      "url": "https://old.example.com"
    }
  }
}`
	os.WriteFile(path, []byte(original), 0644)

	servers := map[string]*McpServer{
		"new-tool": {Type: "remote", URL: "https://new.example.com", Enabled: true},
	}

	plan, err := MergeMCP(path, servers, true)
	if err != nil {
		t.Fatalf("MergeMCP failed: %v", err)
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(plan.Changes))
	}
	if plan.Changes[0].Action != ActionAdd {
		t.Fatalf("expected add action, got %s", plan.Changes[0].Action)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "// existing comment") {
		t.Fatalf("comment was stripped:\n%s", string(data))
	}
	if !strings.Contains(string(data), `"old-tool"`) {
		t.Fatalf("existing key was removed:\n%s", string(data))
	}
	if strings.Contains(string(data), `"new-tool"`) {
		t.Fatalf("dry-run modified file on disk:\n%s", string(data))
	}
}

func TestMergeMCPOverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kilo.jsonc")
	original := `{
  "mcp": {
    "context7-http": {
      "type": "remote",
      "url": "https://old.example.com"
    }
  }
}`
	os.WriteFile(path, []byte(original), 0644)

	servers := map[string]*McpServer{
		"context7-http": {Type: "remote", URL: "https://new.example.com", Enabled: true},
	}

	plan, err := MergeMCP(path, servers, true)
	if err != nil {
		t.Fatalf("MergeMCP failed: %v", err)
	}
	if plan.Changes[0].Action != ActionUpdate {
		t.Fatalf("expected update action, got %s", plan.Changes[0].Action)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "https://old.example.com") {
		t.Fatalf("old url was removed in dry-run:\n%s", string(data))
	}
}
