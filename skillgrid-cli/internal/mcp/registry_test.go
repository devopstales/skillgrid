package mcp

import (
	"skillgrid-cli/internal/config"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRegistry(t *testing.T) {
	dir := filepath.Join("testdata")
	servers, err := LoadRegistry(dir)
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}
	if _, ok := servers["context7-http"]; !ok {
		t.Fatalf("missing context7-http")
	}
	if _, ok := servers["engram"]; !ok {
		t.Fatalf("missing engram")
	}
	if servers["context7-http"].Type != "remote" {
		t.Fatalf("unexpected type: %s", servers["context7-http"].Type)
	}
	if servers["engram"].Command[0] != "engam" {
		t.Fatalf("unexpected command: %v", servers["engram"].Command)
	}
}

func TestPrecheckDependenciesWarnsMissing(t *testing.T) {
	servers := map[string]*config.McpServer{
		"missing-tool": {Type: "local", Command: []string{"nonexistent-binary", "mcp"}},
	}
	missing := PrecheckDependencies(servers)
	if len(missing) != 1 {
		t.Fatalf("expected 1 missing tool, got %d", len(missing))
	}
	if !strings.Contains(missing[0], "nonexistent-binary") {
		t.Fatalf("unexpected missing message: %s", missing[0])
	}
}
