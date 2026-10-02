package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// memSaveAdditiveFixture pins the project to a stable bucket so handlers that
// open the CWD project resolve to one store.
func memSaveAdditiveFixture(t *testing.T) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("MNEMONIC_PROJECT", "savenoaddmcp-probe")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
}

// memSaveResult is the additive mem_save MCP response shape (TICKET-06): the
// legacy id + project fields stay intact, and action / superseded_id are added
// only when present.
type memSaveResult struct {
	ID           int64  `json:"id"`
	Project      string `json:"project"`
	Action       string `json:"action,omitempty"`
	SupersededID int64  `json:"superseded_id,omitempty"`
}

func saveAdditive(t *testing.T, title, typ, content, sessionID string) memSaveResult {
	t.Helper()
	res, err := handleMemSave(context.Background(), newCallTool("mem_save", map[string]any{
		"title":      title,
		"type":       typ,
		"content":    content,
		"session_id": sessionID,
	}))
	if err != nil {
		t.Fatalf("handleMemSave dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_save errored: %s", callResultText(t, res))
	}
	var out memSaveResult
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("unmarshal mem_save result: %v (text %s)", err, callResultText(t, res))
	}
	return out
}

// TestMemSaveMCPAdditive covers TICKET-06: mem_save now routes through
// SaveWithAction and surfaces the AUDN verdict as ADDITIVE response fields.
// The legacy id + project fields must stay intact for old consumers.
func TestMemSaveMCPAdditive(t *testing.T) {
	memSaveAdditiveFixture(t)

	startRes, err := handleMemSessionStart(context.Background(), newCallTool("mem_session_start", map[string]any{}))
	if err != nil {
		t.Fatalf("handleMemSessionStart dispatch: %v", err)
	}
	if startRes.IsError {
		t.Fatalf("mem_session_start errored: %s", callResultText(t, startRes))
	}
	var startOut struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, startRes)), &startOut); err != nil {
		t.Fatalf("unmarshal session start: %v (text %s)", err, callResultText(t, startRes))
	}

	t.Run("new observation → action add", func(t *testing.T) {
		out := saveAdditive(t, "add probe unique", "decision", "body for add probe", startOut.SessionID)
		if out.ID <= 0 {
			t.Errorf("expected id > 0, got %d", out.ID)
		}
		if out.Project == "" {
			t.Errorf("expected project to be present, got empty")
		}
		if out.Action != "add" {
			t.Errorf("expected action \"add\", got %q", out.Action)
		}
		if out.SupersededID != 0 {
			t.Errorf("expected no superseded_id, got %d", out.SupersededID)
		}
	})

	t.Run("hash-dup → action noop", func(t *testing.T) {
		first := saveAdditive(t, "dup probe stable", "decision", "body for dup probe", startOut.SessionID)
		if first.Action != "add" {
			t.Fatalf("seed expected action \"add\", got %q", first.Action)
		}
		dup := saveAdditive(t, "dup probe stable", "decision", "body for dup probe", startOut.SessionID)
		if dup.Action != "noop" {
			t.Errorf("expected action \"noop\" on hash-dup, got %q", dup.Action)
		}
		if dup.ID != first.ID {
			t.Errorf("expected noop to return the existing row's id %d, got %d", first.ID, dup.ID)
		}
		if dup.Project == "" {
			t.Errorf("expected project to be present on noop, got empty")
		}
		if dup.SupersededID != 0 {
			t.Errorf("expected no superseded_id on noop, got %d", dup.SupersededID)
		}
	})
}
