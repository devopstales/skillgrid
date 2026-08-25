package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDryRunDoesNotWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonc")
	os.WriteFile(path, []byte(`{"mcp":{}}`), 0644)

	servers := map[string]*McpServer{
		"new-tool": {Type: "remote", URL: "https://example.com", Enabled: true},
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
	if strings.Contains(string(data), "new-tool") {
		t.Fatalf("dry-run modified file on disk")
	}
}

func TestDryRunReportsUpdateForExistingKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonc")
	os.WriteFile(path, []byte(`{"mcp":{"context7-http":{"type":"remote","url":"https://old"}}}`), 0644)

	servers := map[string]*McpServer{
		"context7-http": {Type: "remote", URL: "https://new", Enabled: true},
	}

	plan, err := MergeMCP(path, servers, true)
	if err != nil {
		t.Fatalf("MergeMCP failed: %v", err)
	}
	if plan.Changes[0].Action != ActionUpdate {
		t.Fatalf("expected update action, got %s", plan.Changes[0].Action)
	}
}
