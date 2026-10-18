package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// memEnsureSessionFixture pins the project to a stable bucket so handlers that
// open the CWD project resolve to one store (same shape as memSaveAdditiveFixture).
func memEnsureSessionFixture(t *testing.T) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("MNEMONIC_PROJECT", "mcpensuresession-probe")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
}

// sessionsRowCount returns the number of sessions rows for sessionID in the
// CWD-resolved store. Returns -1 when the store cannot be opened (should not
// happen for a fixture that has already written through the same store).
func sessionsRowCount(t *testing.T, sessionID string) int {
	t.Helper()
	svc, h, cleanup, err := openNamed("")
	if err != nil {
		t.Fatalf("open store for sessions count: %v", err)
	}
	defer cleanup()
	_ = svc
	var n int
	if err := h.Store().DB.QueryRow(`SELECT count(*) FROM sessions WHERE id = ?`, sessionID).Scan(&n); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	return n
}

// TestMemSaveEnsuresUnregisteredSession covers TASK-080: an MCP mem_save with a
// named session_id that has no sessions row must succeed (not fail the FK on
// observations_core.session_id) and the session row must self-heal. Before the
// fix the core INSERT failed with a FOREIGN KEY constraint error and the whole
// save rolled back.
func TestMemSaveEnsuresUnregisteredSession(t *testing.T) {
	memEnsureSessionFixture(t)

	const named = "task-080-named-session"

	// Precondition: the named session is NOT registered.
	if n := sessionsRowCount(t, named); n != 0 {
		t.Fatalf("precondition: expected no sessions row for %q, found %d", named, n)
	}

	// mem_save with the unregistered named session_id. Before the fix this
	// returns an IsError result carrying the FK constraint failure.
	res, err := handleMemSave(context.Background(), newCallTool("mem_save", map[string]any{
		"title":      "task-080 named session save",
		"type":       "decision",
		"content":    "body proving a named session self-heals",
		"session_id": named,
	}))
	if err != nil {
		t.Fatalf("handleMemSave dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_save with unregistered named session errored (expected self-heal): %s", callResultText(t, res))
	}

	// The save landed: a positive observation id is returned.
	var out memSaveResult
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("unmarshal mem_save result: %v (text %s)", err, callResultText(t, res))
	}
	if out.ID <= 0 {
		t.Errorf("expected observation id > 0, got %d", out.ID)
	}

	// The session row now exists (self-healed).
	if n := sessionsRowCount(t, named); n != 1 {
		t.Errorf("expected exactly 1 sessions row for %q after save, found %d", named, n)
	}
}

// TestMemSaveRegisteredSessionIdempotent is the triangulating second case: a
// session_id that already has a row must save AND leave exactly one sessions
// row (EnsureSession is idempotent — no duplicate). This guards against a fix
// that inserts a fresh session row on every save.
func TestMemSaveRegisteredSessionIdempotent(t *testing.T) {
	memEnsureSessionFixture(t)

	// Register a session the way mem_session_start does.
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
	if startOut.SessionID == "" {
		t.Fatalf("expected a non-empty session id from mem_session_start")
	}

	// Save against the already-registered session.
	res, err := handleMemSave(context.Background(), newCallTool("mem_save", map[string]any{
		"title":      "task-080 registered session save",
		"type":       "decision",
		"content":    "body proving a registered session stays single-row",
		"session_id": startOut.SessionID,
	}))
	if err != nil {
		t.Fatalf("handleMemSave dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_save with registered session errored: %s", callResultText(t, res))
	}
	var out memSaveResult
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("unmarshal mem_save result: %v (text %s)", err, callResultText(t, res))
	}
	if out.ID <= 0 {
		t.Errorf("expected observation id > 0, got %d", out.ID)
	}

	// Exactly one sessions row still exists (no duplicate from EnsureSession).
	if n := sessionsRowCount(t, startOut.SessionID); n != 1 {
		t.Errorf("expected exactly 1 sessions row for %q after save, found %d", startOut.SessionID, n)
	}
}
