package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// currentSessionFixture pins the project to a stable bucket so handlers that
// open the CWD project resolve to one store.
func currentSessionFixture(t *testing.T) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("MNEMONIC_PROJECT", "currentmcp-probe")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
}

// TestMemSaveSessionIDCurrentResolvesInTargetStore is the FK fix scenario:
// session_id "current" must NOT hit the observations.session_id FK, whether or
// not a session row exists yet in the target store. The alias resolves to a
// row registered in the store the save lands in.
func TestMemSaveSessionIDCurrentResolvesInTargetStore(t *testing.T) {
	currentSessionFixture(t)

	// Fresh store: no "current" row yet. The save must register it and land.
	saveRes, err := handleMemSave(context.Background(), newCallTool("mem_save", map[string]any{
		"title":      "current alias save",
		"type":       "discovery",
		"content":    "saved via session_id=current",
		"session_id": "current",
	}))
	if err != nil {
		t.Fatalf("handleMemSave dispatch: %v", err)
	}
	if saveRes.IsError {
		t.Fatalf("mem_save with session_id=current errored: %s", callResultText(t, saveRes))
	}
	var saveOut struct {
		ID        int64  `json:"id"`
		ProjectID string `json:"project"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, saveRes)), &saveOut); err != nil {
		t.Fatalf("unmarshal save: %v (text %s)", err, callResultText(t, saveRes))
	}
	if saveOut.ID == 0 {
		t.Errorf("expected a non-zero observation id, got %s", callResultText(t, saveRes))
	}

	// A second save with the same alias is an upsert of the same session row —
	// still no FK error, no duplicate session.
	saveRes, err = handleMemSave(context.Background(), newCallTool("mem_save", map[string]any{
		"title":      "current alias save again",
		"type":       "discovery",
		"content":    "second save via session_id=current",
		"session_id": "current",
	}))
	if err != nil {
		t.Fatalf("handleMemSave (2nd) dispatch: %v", err)
	}
	if saveRes.IsError {
		t.Fatalf("second mem_save with session_id=current errored: %s", callResultText(t, saveRes))
	}
}

// TestMemSaveSessionIDCurrentReanchorsAcrossStores: the "current" row was
// registered in store A (an earlier save with project A). A save that targets
// store B must re-anchor the row into B so the FK holds, instead of failing.
func TestMemSaveSessionIDCurrentReanchorsAcrossStores(t *testing.T) {
	currentSessionFixture(t)

	// Register "current" under the CWD-resolved store first.
	saveRes, err := handleMemSave(context.Background(), newCallTool("mem_save", map[string]any{
		"title":      "anchor save",
		"type":       "discovery",
		"content":    "anchors the current session in the default store",
		"session_id": "current",
	}))
	if err != nil || saveRes.IsError {
		t.Fatalf("anchor mem_save: %v / %s", err, callResultText(t, saveRes))
	}

	// Now target a DIFFERENT store with an explicit project arg. The FK is
	// per-store, so the alias must re-anchor the row into the target store.
	otherRes, err := handleMemSave(context.Background(), newCallTool("mem_save", map[string]any{
		"title":      "cross-store save",
		"type":       "discovery",
		"content":    "saved into an explicit other project",
		"session_id": "current",
		"project":    "currentmcp-other",
	}))
	if err != nil {
		t.Fatalf("cross-store handleMemSave dispatch: %v", err)
	}
	if otherRes.IsError {
		t.Fatalf("cross-store mem_save with session_id=current errored: %s", callResultText(t, otherRes))
	}
	var otherOut struct {
		ID        int64  `json:"id"`
		ProjectID string `json:"project"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, otherRes)), &otherOut); err != nil {
		t.Fatalf("unmarshal cross-store save: %v (text %s)", err, callResultText(t, otherRes))
	}
	if otherOut.ProjectID != "currentmcp-other" {
		t.Errorf("cross-store save landed in project %q, want currentmcp-other", otherOut.ProjectID)
	}
}
