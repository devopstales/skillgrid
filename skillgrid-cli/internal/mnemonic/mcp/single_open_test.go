package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedObservation opens the project handle once and saves an observation,
// mirroring the scope normalization the old SaveObservation facade applied
// (blank scope -> project, personal -> user). Test-only seed helper.
func seedObservation(t *testing.T, dataDir, projectID string, in memory.SaveInput) {
	t.Helper()
	h, cleanup, err := service.New(dataDir).Open(projectID)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	if in.Scope == "" {
		in.Scope = "project"
	}
	if in.Scope == "personal" {
		in.Scope = "user"
	}
	if _, err := h.Memory().Save(context.Background(), in); err != nil {
		t.Fatalf("save observation: %v", err)
	}
}

// pinProjectCwd pins the project via MNEMONIC_PROJECT, chdirs into dir, and
// injects a service rooted at dataDir. Returns the pinned project id. Every
// test that drives a project-scoped MCP handler through openService must pin
// the project this way: in a temp dir (no git/config) the fallback is a
// directory hash that changes per TempDir, so a stable override is required
// for the store the counter measures to be the one the handler opens.
func pinProjectCwd(t *testing.T, dataDir, dir, projID string) string {
	t.Helper()
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	t.Setenv("MNEMONIC_PROJECT", projID)
	SetService(service.New(dataDir))
	t.Cleanup(func() { SetService(nil) })
	oldDir, _ := os.Getwd()
	if err := os.Chdir(abs); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	return projID
}

// TestMemSaveSingleOpen proves the mem_save happy path opens its project
// store exactly once (Scenario: "mem_save opens store once"). Before the
// single-open rewire it fails (count 2: openService OPEN #1 + the
// SaveObservation facade OPEN #2); after the rewire it passes (count 1).
func TestMemSaveSingleOpen(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	proj := pinProjectCwd(t, dataDir, dir, "singleopen-probe")

	// Seed an active session for the pinned project so mem_save has a valid
	// session_id to link against.
	sid, _, err := (service.New(dataDir)).SessionStart(context.Background(), dir, "singleopen")
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}

	store.ResetOpenCount()
	req := newCallTool("mem_save", map[string]any{
		"title":      "single open probe",
		"type":       "decision",
		"content":    "**What** open once **Why** single-open **Where** mcp **Learned** —",
		"session_id": sid,
	})
	res, err := handleMemSave(context.Background(), req)
	if err != nil {
		t.Fatalf("handleMemSave: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_save returned error: %s", callResultText(t, res))
	}
	text := callResultText(t, res)
	if !strings.Contains(text, `"project":"singleopen-probe"`) {
		t.Fatalf("expected save under project %q, got %s", proj, text)
	}

	count := store.OpenCount()
	if count != 1 {
		t.Errorf("mem_save opened the store %d times, want exactly 1 (double-open)", count)
	}
}

// TestMemSavePromptSingleOpen is a second single-open probe through a different
// (write) handler, guarding that the rewire is not limited to mem_save.
func TestMemSavePromptSingleOpen(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "singleopen-prompt")

	sid, _, err := (service.New(dataDir)).SessionStart(context.Background(), dir, "singleopen")
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}

	store.ResetOpenCount()
	req := newCallTool("mem_save_prompt", map[string]any{
		"content":    "why did we rewire the mcp handlers to a single project handle",
		"session_id": sid,
	})
	res, err := handleMemSavePrompt(context.Background(), req)
	if err != nil {
		t.Fatalf("handleMemSavePrompt: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_save_prompt returned error: %s", callResultText(t, res))
	}
	if got := store.OpenCount(); got != 1 {
		t.Errorf("mem_save_prompt opened the store %d times, want exactly 1", got)
	}
}

// TestMemSearchContractShape pins the mem_search result shape (project +
// observations[]) after the handle rewire — the MCP contract must stay
// byte-compatible (Scenario: "mem_search result shape unchanged").
func TestMemSearchContractShape(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "shape-probe")

	sid, _, err := (service.New(dataDir)).SessionStart(context.Background(), dir, "shape")
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	// Seed an observation under the pinned project so search has a hit.
	seedObservation(t, dataDir, "shape-probe", memory.SaveInput{
		Title:     "shape probe note",
		Type:      "learning",
		Content:   "contract shape check for mem_search after single-open rewire",
		Scope:     "project",
		SessionID: sid,
	})

	// The seeded observation is private and owned by its session id (the
	// mem_save / save-path fallback, 013 step 01). Read as that owner so the
	// per-owner filter returns it — a different owner would be gated out.
	req := newCallTool("mem_search", map[string]any{"query": "contract shape check", "reader_owner": sid})
	res, err := handleMemSearch(context.Background(), req)
	if err != nil {
		t.Fatalf("handleMemSearch: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_search returned error: %s", callResultText(t, res))
	}
	var out struct {
		Project      string `json:"project"`
		Observations []struct {
			ID       int64  `json:"id"`
			Type     string `json:"type"`
			Title    string `json:"title"`
			Content  string `json:"content"`
			Project  string `json:"project"`
			Scope    string `json:"scope"`
			TopicKey string `json:"topic_key,omitempty"`
			Source   string `json:"source,omitempty"`
		} `json:"observations"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("unmarshal mem_search result: %v", err)
	}
	if out.Project != "shape-probe" {
		t.Errorf("expected project %q in result, got %q", "shape-probe", out.Project)
	}
	if len(out.Observations) != 1 {
		t.Fatalf("expected 1 observation, got %d: %s", len(out.Observations), callResultText(t, res))
	}
}

// TestMemSearchSingleOpen proves a scoped (non-all-projects) mem_search opens
// the target store exactly once — the second single-open probe on the read
// path (Scenario: "mem_search result shape unchanged" + single-open).
func TestMemSearchSingleOpen(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "shape-search")

	sid, _, err := (service.New(dataDir)).SessionStart(context.Background(), dir, "shape")
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	seedObservation(t, dataDir, "shape-search", memory.SaveInput{
		Title:     "search open once",
		Type:      "learning",
		Content:   "scoped search must open the store exactly once",
		Scope:     "project",
		SessionID: sid,
	})

	store.ResetOpenCount()
	// Same as TestMemSearchContractShape: the seeded row is private and owned
	// by its session id (save-path fallback, 013 step 01). Read as that owner.
	req := newCallTool("mem_search", map[string]any{"query": "scoped search once", "reader_owner": sid})
	res, err := handleMemSearch(context.Background(), req)
	if err != nil {
		t.Fatalf("handleMemSearch: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_search returned error: %s", callResultText(t, res))
	}
	if got := store.OpenCount(); got != 1 {
		t.Errorf("scoped mem_search opened the store %d times, want exactly 1", got)
	}
}

// TestCodeSearchSingleOpen and TestWebCacheLookupSingleOpen are single-open
// probes on the code_* and web_cache_* handlers (02.3 [AFK]).
func TestCodeSearchSingleOpen(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "codesearch-probe")

	store.ResetOpenCount()
	res, err := handleCodeSearch(context.Background(), newCallTool("code_search", map[string]any{"query": "anything"}))
	if err != nil {
		t.Fatalf("handleCodeSearch: %v", err)
	}
	if res.IsError {
		t.Fatalf("code_search returned error: %s", callResultText(t, res))
	}
	if got := store.OpenCount(); got != 1 {
		t.Errorf("code_search opened the store %d times, want exactly 1", got)
	}
}

func TestWebCacheLookupSingleOpen(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "weblookup-probe")

	store.ResetOpenCount()
	res, err := handleWebCacheLookup(context.Background(), newCallTool("web_cache_lookup", map[string]any{"source": "fetch", "url": "https://example.com/x"}))
	if err != nil {
		t.Fatalf("handleWebCacheLookup: %v", err)
	}
	if res.IsError {
		t.Fatalf("web_cache_lookup returned error: %s", callResultText(t, res))
	}
	if got := store.OpenCount(); got != 1 {
		t.Errorf("web_cache_lookup opened the store %d times, want exactly 1", got)
	}
}
