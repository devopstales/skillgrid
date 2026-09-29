package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// TestFactAddRegistered covers @step-02: fact_add is listed alongside the
// unchanged mem_* tools.
func TestFactAddRegistered(t *testing.T) {
	s := NewServer()
	tools := s.ListTools()
	if _, ok := tools["fact_add"]; !ok {
		t.Fatal("fact_add not registered")
	}
	if _, ok := tools["mem_save"]; !ok {
		t.Fatal("mem_save missing after fact tools registered")
	}
}

// TestFactAddMCP covers @step-02: the tool inserts a fact, returns its id, and
// records a session_events row with action_type=fact_add and the fact id.
func TestFactAddMCP(t *testing.T) {
	dataDir := t.TempDir()
	project := "factmcp"
	t.Setenv("MNEMONIC_PROJECT", project)
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	SetService(service.New(dataDir))
	t.Cleanup(func() { SetService(nil) })

	sid := "sess-fact-mcp"
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
		VALUES (?, ?, ?, ?, 'active')`, sid, project, dataDir, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	req := mcplib.CallToolRequest{}
	req.Params.Name = "fact_add"
	req.Params.Arguments = map[string]any{
		"project":    project,
		"session_id": sid,
		"content":    "MCP fact: session events trail must carry the fact id",
	}
	res, err := handleFactAdd(context.Background(), req)
	if err != nil {
		t.Fatalf("handleFactAdd: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_add error: %s", callResultText(t, res))
	}
	text := callResultText(t, res)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("parse result %q: %v", text, err)
	}
	factID, ok := out["fact_id"].(float64)
	if !ok || factID <= 0 {
		t.Fatalf("no positive fact_id in %s", text)
	}

	// Session trail: action_type=fact_add with the fact id in the payload.
	var actionType, payload string
	if err := st.DB.QueryRow(`
		SELECT action_type, payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_add'`, sid).
		Scan(&actionType, &payload); err != nil {
		t.Fatalf("fact_add session event missing: %v", err)
	}
	if !strings.Contains(payload, `"fact_id":`) {
		t.Errorf("payload %q does not carry the fact id", payload)
	}
	st.Close()
}

// TestFactAddRequiresContent: a missing/blank content is a clean tool error.
func TestFactAddRequiresContent(t *testing.T) {
	dataDir := t.TempDir()
	project := "factmcp2"
	t.Setenv("MNEMONIC_PROJECT", project)
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	SetService(service.New(dataDir))
	t.Cleanup(func() { SetService(nil) })
	defer st.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
		VALUES ('sess-nocontent', ?, ?, ?, 'active')`, project, dataDir, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	req := mcplib.CallToolRequest{}
	req.Params.Name = "fact_add"
	req.Params.Arguments = map[string]any{
		"project":    project,
		"session_id": "sess-nocontent",
		"content":    "",
	}
	res, err := handleFactAdd(context.Background(), req)
	if err != nil {
		t.Fatalf("handleFactAdd: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected tool error for blank content")
	}
}
