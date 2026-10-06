package setup

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestCanonicalMCPConfigIncludesBacklog(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// mnemonic/internal/setup → repo root
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	entries, err := LoadMCPConfig(repoRoot)
	if err != nil {
		t.Fatalf("LoadMCPConfig(%s): %v", repoRoot, err)
	}

	var backlog *MCPServerConfig
	for i := range entries {
		if entries[i].Name == "backlog" {
			backlog = &entries[i]
			break
		}
	}
	if backlog == nil {
		t.Fatal("config.d/mcp.yaml must include backlog (installed for cursor, opencode, kilo)")
	}
	if backlog.Type != "local" {
		t.Fatalf("backlog type = %q, want local", backlog.Type)
	}
	want := []string{"backlog", "mcp", "start"}
	if len(backlog.Command) != len(want) {
		t.Fatalf("backlog command = %v, want %v", backlog.Command, want)
	}
	for i, c := range want {
		if backlog.Command[i] != c {
			t.Fatalf("backlog command = %v, want %v", backlog.Command, want)
		}
	}
}
