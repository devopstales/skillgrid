package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/relay"
)

// sessionHandoffFixture pins the project to a stable bucket, points the CWD at
// a temp dir, and injects a service rooted at a temp data dir. It chdirs into
// dir so handlers that open the CWD project resolve to that store, and so the
// cleave bundle lands under dir/.skillgrid/.cleave/. Returns the CWD dir
// first (the cleave root's parent).
func sessionHandoffFixture(t *testing.T) (dir, dataDir string) {
	t.Helper()
	dataDir = t.TempDir()
	dir = t.TempDir()
	pinProjectCwd(t, dataDir, dir, "handoffmcp-probe")
	return dir, dataDir
}

// TestSessionHandoffTools covers @step-02 RED "Mnemonic tool surface"
// (Scenario: Fail closed and mem tools remain): session_handoff and
// session_resume are registered with distinct session_* names, the tool
// surface grows additively 78 -> 80, and every 005 mem_* tool keeps its name
// + required params unchanged.
func TestSessionHandoffTools(t *testing.T) {
	sessionHandoffFixture(t)

	tools := NewServer().ListTools()
	for _, name := range []string{"session_handoff", "session_resume"} {
		if _, ok := tools[name]; !ok {
			t.Errorf("expected tool %q to be registered", name)
		}
	}

	// Tool surface grows additively: 78 baseline + 2 session + 2 status/compact = 82.
	if len(tools) != 83 {
		t.Errorf("expected 83 tools (78 baseline + 2 session + 2 status/compact), got %d", len(tools))
	}

	// Existing 005 mem_* tools keep their names + required params unchanged.
	for name, wantRequired := range expectedMemToolSurface {
		st, ok := tools[name]
		if !ok {
			t.Errorf("005 mem tool %q is no longer registered", name)
			continue
		}
		got := append([]string(nil), st.Tool.InputSchema.Required...)
		want := append([]string(nil), wantRequired...)
		sort.Strings(got)
		sort.Strings(want)
		if len(got) != len(want) {
			t.Errorf("%q: required params changed: got %v, want %v", name, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%q: required param %d changed: got %q, want %q", name, i, got[i], want[i])
			}
		}
	}

	// Bad session args rejected clearly:
	//  - session_handoff with no progress -> validation error.
	res, err := handleSessionHandoff(context.Background(), newCallTool("session_handoff", map[string]any{}))
	if err != nil {
		t.Fatalf("handleSessionHandoff dispatch: %v", err)
	}
	if !res.IsError {
		t.Errorf("session_handoff without progress should be a validation error, got: %s", callResultText(t, res))
	}
	//  - session_resume with no handoff_id -> validation error.
	res, err = handleSessionResume(context.Background(), newCallTool("session_resume", map[string]any{}))
	if err != nil {
		t.Fatalf("handleSessionResume dispatch: %v", err)
	}
	if !res.IsError {
		t.Errorf("session_resume without handoff_id should be a validation error, got: %s", callResultText(t, res))
	}
}

// TestSessionToolsDispatch is the end-to-end happy path at the MCP boundary
// (Scenario: Handoff writes cleave bundle and row): a real session_handoff MCP
// call writes the three .cleave/ files and a session_handoffs row, and a
// follow-up session_resume MCP call returns the stored NEXT_PROMPT. It also
// drives mem_session_start + mem_save through the handlers so the RED test
// proves the mem tools still dispatch alongside the new session tools.
func TestSessionToolsDispatch(t *testing.T) {
	dir, dataDir := sessionHandoffFixture(t)

	// Seed a real session (mem_session_start dispatch) so the handoff has a
	// valid source_session to link against.
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

	// The mem tool still works (Scenario: Fail closed and mem tools remain).
	saveRes, err := handleMemSave(context.Background(), newCallTool("mem_save", map[string]any{
		"title":      "handoff probe",
		"type":       "learning",
		"content":    "mem_save still dispatches next to the session tools",
		"session_id": startOut.SessionID,
	}))
	if err != nil {
		t.Fatalf("handleMemSave: %v", err)
	}
	if saveRes.IsError {
		t.Fatalf("mem_save errored: %s", callResultText(t, saveRes))
	}

	// session_handoff writes the cleave bundle + row.
	relHandoff := relay.Bundle{
		Progress:   "implemented the relay module",
		Knowledge:  "fail closed: write files before the row",
		NextPrompt: "Resume: run the verify phase for change 006",
	}
	hoRes, err := handleSessionHandoff(context.Background(), newCallTool("session_handoff", map[string]any{
		"progress":    relHandoff.Progress,
		"knowledge":   relHandoff.Knowledge,
		"next_prompt": relHandoff.NextPrompt,
		"session_id":  startOut.SessionID,
	}))
	if err != nil {
		t.Fatalf("handleSessionHandoff: %v", err)
	}
	if hoRes.IsError {
		t.Fatalf("session_handoff errored: %s", callResultText(t, hoRes))
	}
	var hoOut struct {
		HandoffID string   `json:"handoff_id"`
		Paths     []string `json:"paths"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, hoRes)), &hoOut); err != nil {
		t.Fatalf("unmarshal handoff: %v (text %s)", err, callResultText(t, hoRes))
	}
	if hoOut.HandoffID == "" {
		t.Errorf("session_handoff must return a handoff_id, got %s", callResultText(t, hoRes))
	}
	if len(hoOut.Paths) != 3 {
		t.Errorf("session_handoff must return 3 cleave paths, got %v", hoOut.Paths)
	}

	// The three .cleave/ files exist on disk under dir/.skillgrid/.cleave/.
	// dir is t.TempDir() (a non-resolved /var path on macOS) while the handle
	// root is the resolved /private/var path, so resolve symlinks before stat.
	cleaveRoot, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("evalsymlinks dir: %v", err)
	}
	cleaveDir := filepath.Join(cleaveRoot, ".skillgrid", ".cleave")
	for _, name := range []string{"PROGRESS.md", "KNOWLEDGE.md", "NEXT_PROMPT.md"} {
		if _, err := os.Stat(filepath.Join(cleaveDir, name)); err != nil {
			t.Errorf("expected cleave file %s on disk: %v", name, err)
		}
	}

	// session_resume returns the stored NEXT_PROMPT.
	resRes, err := handleSessionResume(context.Background(), newCallTool("session_resume", map[string]any{
		"handoff_id": hoOut.HandoffID,
	}))
	if err != nil {
		t.Fatalf("handleSessionResume: %v", err)
	}
	if resRes.IsError {
		t.Fatalf("session_resume errored: %s", callResultText(t, resRes))
	}
	var resOut struct {
		Prompt    string `json:"prompt"`
		HandoffID string `json:"handoff_id"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, resRes)), &resOut); err != nil {
		t.Fatalf("unmarshal resume: %v (text %s)", err, callResultText(t, resRes))
	}
	if resOut.Prompt != relHandoff.NextPrompt {
		t.Errorf("session_resume prompt = %q, want the stored NEXT_PROMPT %q", resOut.Prompt, relHandoff.NextPrompt)
	}
	if resOut.HandoffID != hoOut.HandoffID {
		t.Errorf("session_resume handoff_id = %q, want %q", resOut.HandoffID, hoOut.HandoffID)
	}
	_ = dataDir
}

// TestMemSaveDispatch covers the @step-02 RED threat row directly: with the
// session tools registered, mem_save still dispatches to its handler and
// returns a saved id (the mem_* surface is additive, never dropped).
func TestMemSaveDispatch(t *testing.T) {
	_, dir := sessionHandoffFixture(t)

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

	res, err := handleMemSave(context.Background(), newCallTool("mem_save", map[string]any{
		"title":      "mem save dispatch probe",
		"type":       "decision",
		"content":    "mem_save must keep working once session tools land",
		"session_id": startOut.SessionID,
	}))
	if err != nil {
		t.Fatalf("handleMemSave dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_save errored: %s", callResultText(t, res))
	}
	if !strings.Contains(callResultText(t, res), `"id"`) {
		t.Errorf("mem_save should return an id, got: %s", callResultText(t, res))
	}
	_ = dir
}
