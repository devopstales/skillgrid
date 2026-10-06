package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
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

// saveAdditive invokes mem_save and returns both the typed struct (for the
// legacy id/project/action assertions) and the raw key map (so the ADDITIVE
// contract can assert a field is ABSENT, not merely zero — a struct field
// without omitempty would pass a `!= 0` check whether the key is present or
// not).
func saveAdditive(t *testing.T, title, typ, content, sessionID string) (memSaveResult, map[string]any) {
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
	text := callResultText(t, res)
	var out memSaveResult
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal mem_save result: %v (text %s)", err, text)
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		t.Fatalf("unmarshal raw mem_save result: %v (text %s)", err, text)
	}
	return out, raw
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
		out, raw := saveAdditive(t, "add probe unique", "decision", "body for add probe", startOut.SessionID)
		if out.ID <= 0 {
			t.Errorf("expected id > 0, got %d", out.ID)
		}
		if out.Project == "" {
			t.Errorf("expected project to be present, got empty")
		}
		if out.Action != "add" {
			t.Errorf("expected action \"add\", got %q", out.Action)
		}
		// The additive contract is that superseded_id is ABSENT (not present
		// with 0) for a plain add — assert the key is actually missing.
		if _, present := raw["superseded_id"]; present {
			t.Errorf("expected superseded_id to be absent on add, got %v", raw["superseded_id"])
		}
	})

	t.Run("hash-dup → action noop", func(t *testing.T) {
		first, _ := saveAdditive(t, "dup probe stable", "decision", "body for dup probe", startOut.SessionID)
		if first.Action != "add" {
			t.Fatalf("seed expected action \"add\", got %q", first.Action)
		}
		dup, raw := saveAdditive(t, "dup probe stable", "decision", "body for dup probe", startOut.SessionID)
		if dup.Action != "noop" {
			t.Errorf("expected action \"noop\" on hash-dup, got %q", dup.Action)
		}
		if dup.ID != first.ID {
			t.Errorf("expected noop to return the existing row's id %d, got %d", first.ID, dup.ID)
		}
		if dup.Project == "" {
			t.Errorf("expected project to be present on noop, got empty")
		}
		// Noop does not supersede: superseded_id must be absent, not zero.
		if _, present := raw["superseded_id"]; present {
			t.Errorf("expected superseded_id to be absent on noop, got %v", raw["superseded_id"])
		}
	})
}
