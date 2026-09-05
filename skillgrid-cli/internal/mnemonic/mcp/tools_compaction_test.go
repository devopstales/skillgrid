package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func TestMemSaveRemainsRegisteredAdditive(t *testing.T) {
	s := NewServer()
	tools := s.ListTools()
	if _, ok := tools["mem_save"]; !ok {
		t.Fatal("mem_save missing after compaction tools registered")
	}
	if _, ok := tools["mnemonic_commit"]; !ok {
		t.Fatal("mnemonic_commit not registered")
	}

	dataDir := t.TempDir()
	project := "reg"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()
	SetService(service.New(dataDir))

	// Start a session so mem_save can write.
	start := mcplib.CallToolRequest{}
	start.Params.Name = "mem_session_start"
	start.Params.Arguments = map[string]any{"project": project, "directory": dataDir}
	// mem_session_start may resolve via directory — use SaveObservation via handler path:
	req := mcplib.CallToolRequest{}
	req.Params.Name = "mem_save"
	req.Params.Arguments = map[string]any{
		"type":       "discovery",
		"title":      "still works",
		"content":    "body",
		"session_id": "sess-additive",
		"project":    project,
	}
	// Ensure session exists
	st2, _ := store.Open(dataDir, project)
	_, _ = st2.DB.Exec(`INSERT INTO sessions (id, project, directory, started_at, status) VALUES ('sess-additive', ?, ?, datetime('now'), 'active')`, project, dataDir)
	st2.Close()

	res, err := handleMemSave(context.Background(), req)
	if err != nil {
		t.Fatalf("mem_save dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_save error: %v", res.Content)
	}
}

func TestMnemonicCommitMCP(t *testing.T) {
	dataDir := t.TempDir()
	project := "mcpc"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()
	SetService(service.New(dataDir))

	req := mcplib.CallToolRequest{}
	req.Params.Name = "mnemonic_commit"
	req.Params.Arguments = map[string]any{
		"project":          project,
		"title":            "From MCP",
		"lessons_learned":  "Persist explicitly.",
	}
	res, err := handleMnemonicCommit(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("%v", res.Content)
	}
	text := retrievalToolText(t, res)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatal(err)
	}
	if out["memory_id"] == nil {
		t.Fatalf("no memory_id: %s", text)
	}
	// Allow async tier goroutine to finish before TempDir cleanup.
	time.Sleep(300 * time.Millisecond)
}
