package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
)

func enabledHooksConfig() memory.HooksConfig {
	return memory.HooksConfig{Enabled: true}
}

// hookPayload builds a post_tool_use payload. When action is non-empty it is
// the explicit action_type to force (only honored by RunHook when tool is
// empty); otherwise the action is derived from the tool name.
func hookPayload(sid, tool, file, cmd, status, action string) memory.HookPayload {
	return memory.HookPayload{
		SessionID:    sid,
		ToolName:     tool,
		File:         file,
		Command:      cmd,
		ResultStatus: status,
		ActionType:   action,
	}
}

type queryEventsOut struct {
	Events            []struct {
		ToolName   string `json:"tool_name"`
		ActionType string `json:"action_type"`
		Path       string `json:"path"`
	} `json:"events"`
	Count             int  `json:"count"`
	SensitiveIncluded bool `json:"sensitive_included"`
}

type exportEventsOut struct {
	JSONL string `json:"jsonl"`
}

func callQueryEvents(t *testing.T, args map[string]any) queryEventsOut {
	t.Helper()
	res, err := handleQueryEvents(context.Background(), newCallTool("mem_query_events", args))
	if err != nil {
		t.Fatalf("handleQueryEvents dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_query_events errored: %s", callResultText(t, res))
	}
	var out queryEventsOut
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("unmarshal mem_query_events: %v (text %s)", err, callResultText(t, res))
	}
	return out
}

func callExportEvents(t *testing.T, args map[string]any) exportEventsOut {
	t.Helper()
	res, err := handleExportEvents(context.Background(), newCallTool("mem_export_events", args))
	if err != nil {
		t.Fatalf("handleExportEvents dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_export_events errored: %s", callResultText(t, res))
	}
	var out exportEventsOut
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("unmarshal mem_export_events: %v (text %s)", err, callResultText(t, res))
	}
	return out
}

// TestQueryEventsTool covers the MCP mem_query_events tool: seeded events are
// returned with correct count, sensitive exclusion by default, and the
// sensitive_included flag. SATISFIES: query-action-filter,
// query-sensitive-default-excluded, query-sensitive-included, query-count-only.
func TestQueryEventsTool(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "queryevents-probe")

	// Seed a session with tool-call events.
	ctx := context.Background()
	_, h, cleanup, err := openService()
	if err != nil {
		t.Fatalf("openService: %v", err)
	}
	h.Memory().SetHooks(enabledHooksConfig())
	sid, err := h.Memory().SessionStart(ctx, t.TempDir(), "qep")
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	cleanup()

	// Re-open to seed events (hooks must be enabled on the same service).
	_, h2, cleanup2, err := openService()
	if err != nil {
		t.Fatalf("openService2: %v", err)
	}
	defer cleanup2()
	h2.Memory().SetHooks(enabledHooksConfig())
	if _, err := h2.Memory().RunHook(ctx, "post_tool_use", hookPayload(sid, "write", "src/main.go", "", "success", "")); err != nil {
		t.Fatalf("seed write: %v", err)
	}
	if _, err := h2.Memory().RunHook(ctx, "post_tool_use", hookPayload(sid, "", "", "go test", "success", "command_exec")); err != nil {
		t.Fatalf("seed bash: %v", err)
	}

	// Default: sensitive excluded, count reflects non-sensitive events.
	// session_start (1) + 2 tool calls = 3 non-sensitive.
	out := callQueryEvents(t, map[string]any{})
	if out.Count != 3 {
		t.Errorf("count = %d, want 3 (session_start + 2 tool calls)", out.Count)
	}
	if out.SensitiveIncluded {
		t.Errorf("sensitive_included = true, want false (default)")
	}
	if len(out.Events) != 3 {
		t.Errorf("events = %d, want 3", len(out.Events))
	}

	// Action filter: only command_exec.
	out = callQueryEvents(t, map[string]any{"action": "command_exec"})
	if out.Count != 1 {
		t.Errorf("action=command_exec count = %d, want 1", out.Count)
	}
	if len(out.Events) != 1 || out.Events[0].ActionType != "command_exec" {
		t.Errorf("action=command_exec events = %+v, want 1 command_exec", out.Events)
	}

	// Count only: no events returned, count still correct.
	out = callQueryEvents(t, map[string]any{"count": true})
	if out.Count != 3 {
		t.Errorf("count-only count = %d, want 3", out.Count)
	}
	if len(out.Events) != 0 {
		t.Errorf("count-only events = %d, want 0", len(out.Events))
	}

	// Session filter.
	out = callQueryEvents(t, map[string]any{"session": sid})
	if out.Count != 3 {
		t.Errorf("session filter count = %d, want 3", out.Count)
	}
}

// TestQueryEventsToolSensitiveInclusion verifies that sensitive: true includes
// sensitive events. SATISFIES: query-sensitive-included.
func TestQueryEventsToolSensitiveInclusion(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "queryevents-sens")

	ctx := context.Background()
	_, h, cleanup, err := openService()
	if err != nil {
		t.Fatalf("openService: %v", err)
	}
	h.Memory().SetHooks(enabledHooksConfig())
	sid, err := h.Memory().SessionStart(ctx, t.TempDir(), "qes")
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	cleanup()

	_, h2, cleanup2, err := openService()
	if err != nil {
		t.Fatalf("openService2: %v", err)
	}
	defer cleanup2()
	h2.Memory().SetHooks(enabledHooksConfig())
	// .env is sensitive.
	if _, err := h2.Memory().RunHook(ctx, "post_tool_use", hookPayload(sid, "read", "/home/u/.env", "", "success", "")); err != nil {
		t.Fatalf("seed .env: %v", err)
	}

	// Default: .env excluded → only session_start = 1.
	out := callQueryEvents(t, map[string]any{})
	if out.Count != 1 {
		t.Errorf("default count = %d, want 1 (.env excluded)", out.Count)
	}

	// sensitive: true → session_start + .env = 2.
	out = callQueryEvents(t, map[string]any{"sensitive": true})
	if out.Count != 2 {
		t.Errorf("sensitive=true count = %d, want 2", out.Count)
	}
	if !out.SensitiveIncluded {
		t.Errorf("sensitive_included = false, want true")
	}
}

// TestQueryEventsToolFileGlob verifies the file LIKE filter.
// SATISFIES: query-file-glob.
func TestQueryEventsToolFileGlob(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "queryevents-glob")

	ctx := context.Background()
	_, h, cleanup, err := openService()
	if err != nil {
		t.Fatalf("openService: %v", err)
	}
	h.Memory().SetHooks(enabledHooksConfig())
	sid, err := h.Memory().SessionStart(ctx, t.TempDir(), "qeg")
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	cleanup()

	_, h2, cleanup2, err := openService()
	if err != nil {
		t.Fatalf("openService2: %v", err)
	}
	defer cleanup2()
	h2.Memory().SetHooks(enabledHooksConfig())
	if _, err := h2.Memory().RunHook(ctx, "post_tool_use", hookPayload(sid, "write", "src/auth.go", "", "success", "")); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := h2.Memory().RunHook(ctx, "post_tool_use", hookPayload(sid, "write", "lib/util.go", "", "success", "")); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// file: src/% matches only src/auth.go.
	out := callQueryEvents(t, map[string]any{"file": "src/%"})
	if out.Count != 1 {
		t.Errorf("file=src/%% count = %d, want 1", out.Count)
	}
}

// TestQueryEventsToolTimeWindow verifies since/until RFC3339 filters.
// SATISFIES: query-time-window.
func TestQueryEventsToolTimeWindow(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "queryevents-time")

	ctx := context.Background()
	_, h, cleanup, err := openService()
	if err != nil {
		t.Fatalf("openService: %v", err)
	}
	h.Memory().SetHooks(enabledHooksConfig())
	sid, err := h.Memory().SessionStart(ctx, t.TempDir(), "qet")
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	cleanup()

	_, h2, cleanup2, err := openService()
	if err != nil {
		t.Fatalf("openService2: %v", err)
	}
	defer cleanup2()
	h2.Memory().SetHooks(enabledHooksConfig())
	if _, err := h2.Memory().RunHook(ctx, "post_tool_use", hookPayload(sid, "", "", "ls", "success", "command_exec")); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Far-future since: no events.
	out := callQueryEvents(t, map[string]any{"since": "2099-01-01T00:00:00Z"})
	if out.Count != 0 {
		t.Errorf("future since count = %d, want 0", out.Count)
	}

	// Far-past since: all events.
	out = callQueryEvents(t, map[string]any{"since": "2000-01-01T00:00:00Z"})
	if out.Count != 2 {
		t.Errorf("past since count = %d, want 2", out.Count)
	}
}

// TestExportEventsTool verifies JSONL export. SATISFIES: export-jsonl.
func TestExportEventsTool(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "exportevents-probe")

	ctx := context.Background()
	_, h, cleanup, err := openService()
	if err != nil {
		t.Fatalf("openService: %v", err)
	}
	h.Memory().SetHooks(enabledHooksConfig())
	sid, err := h.Memory().SessionStart(ctx, t.TempDir(), "eep")
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	cleanup()

	_, h2, cleanup2, err := openService()
	if err != nil {
		t.Fatalf("openService2: %v", err)
	}
	defer cleanup2()
	h2.Memory().SetHooks(enabledHooksConfig())
	if _, err := h2.Memory().RunHook(ctx, "post_tool_use", hookPayload(sid, "write", "src/main.go", "", "success", "")); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := h2.Memory().RunHook(ctx, "post_tool_use", hookPayload(sid, "", "", "ls -la", "success", "command_exec")); err != nil {
		t.Fatalf("seed: %v", err)
	}

	out := callExportEvents(t, map[string]any{"session": sid})
	if out.JSONL == "" {
		t.Fatal("empty JSONL export")
	}
	lines := strings.Count(out.JSONL, "\n") + 1
	// 3 events: session_start + 2 tool calls.
	if lines != 3 {
		t.Errorf("JSONL lines = %d, want 3", lines)
	}
}

// TestQueryEventsToolsRegistered verifies both tools appear in the MCP server.
func TestQueryEventsToolsRegistered(t *testing.T) {
	tools := NewServer().ListTools()
	if _, ok := tools["mem_query_events"]; !ok {
		t.Errorf("mem_query_events not registered")
	}
	if _, ok := tools["mem_export_events"]; !ok {
		t.Errorf("mem_export_events not registered")
	}
}
