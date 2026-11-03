package memory

import (
	"context"
	"testing"
)

// TestQueryEvents seeds a store with known events via the real RunHook writer,
// then asserts each filter returns the expected subset.
// SATISFIES: query-action-filter, query-sensitive-default-excluded,
// query-sensitive-included, query-count-only, query-file-glob, query-time-window.
func TestQueryEvents(t *testing.T) {
	svc, _ := openTracerService(t)
	svc.SetHooks(HooksConfig{Enabled: true})
	ctx := context.Background()

	// Start two sessions so RunHook can find them.
	sid1, err := svc.SessionStart(ctx, t.TempDir(), "q1")
	if err != nil {
		t.Fatalf("start s1: %v", err)
	}
	sid2, err := svc.SessionStart(ctx, t.TempDir(), "q2")
	if err != nil {
		t.Fatalf("start s2: %v", err)
	}

	// Seed events via RunHook (the real writer).
	events := []HookPayload{
		{SessionID: sid1, ToolName: "bash", ActionType: "command_exec", Command: "go test", ResultStatus: "success"},
		{SessionID: sid1, ToolName: "read", ActionType: "file_read", File: "/home/u/.env", ResultStatus: "success"},
		{SessionID: sid1, ToolName: "write", ActionType: "file_write", File: "src/auth.go", ResultStatus: "success"},
		{SessionID: sid2, ToolName: "bash", ActionType: "command_exec", Command: "ls", ResultStatus: "error"},
	}
	for _, p := range events {
		if _, err := svc.RunHook(ctx, HookPostToolUse, p); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	t.Run("action filter", func(t *testing.T) {
		// The only file_read is the .env read (sensitive), so include sensitive.
		got, count, err := svc.QueryEvents(ctx, QueryEventOpts{Action: "file_read", IncludeSensitive: true})
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 || len(got) != 1 {
			t.Fatalf("count=%d len=%d, want 1", count, len(got))
		}
		if got[0].ToolName != "read" {
			t.Errorf("got %q, want read", got[0].ToolName)
		}
	})

	t.Run("sensitive default excluded", func(t *testing.T) {
		_, count, err := svc.QueryEvents(ctx, QueryEventOpts{})
		if err != nil {
			t.Fatal(err)
		}
		// sid1: session_start + bash + write = 3 non-sensitive (read/.env is sensitive)
		// sid2: session_start + bash = 2 non-sensitive
		// Total: 5 non-sensitive
		if count != 5 {
			t.Errorf("count=%d, want 5 (sensitive excluded)", count)
		}
	})

	t.Run("sensitive included", func(t *testing.T) {
		_, count, err := svc.QueryEvents(ctx, QueryEventOpts{IncludeSensitive: true})
		if err != nil {
			t.Fatal(err)
		}
		// sid1: 4 events, sid2: 2 events = 6 total
		if count != 6 {
			t.Errorf("count=%d, want 6", count)
		}
	})

	t.Run("count only", func(t *testing.T) {
		got, count, err := svc.QueryEvents(ctx, QueryEventOpts{CountOnly: true})
		if err != nil {
			t.Fatal(err)
		}
		if got != nil {
			t.Errorf("got non-nil events in CountOnly mode")
		}
		if count != 5 {
			t.Errorf("count=%d, want 5", count)
		}
	})

	t.Run("file glob", func(t *testing.T) {
		_, count, err := svc.QueryEvents(ctx, QueryEventOpts{File: "src/%", IncludeSensitive: true})
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("count=%d, want 1 (src/auth.go)", count)
		}
	})

	t.Run("session filter", func(t *testing.T) {
		_, count, err := svc.QueryEvents(ctx, QueryEventOpts{Session: sid1})
		if err != nil {
			t.Fatal(err)
		}
		// s1 has 4 events (session_start + 3 tool calls), one sensitive (.env).
		if count != 3 {
			t.Errorf("count=%d, want 3", count)
		}
	})

	t.Run("agent tool command filters", func(t *testing.T) {
		if _, err := svc.store.DB.ExecContext(ctx, `UPDATE sessions SET agent = 'opencode' WHERE id = ?`, sid2); err != nil {
			t.Fatal(err)
		}
		_, count, err := svc.QueryEvents(ctx, QueryEventOpts{Agent: "opencode"})
		if err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Errorf("agent count=%d, want 2 (sid2 start + ls)", count)
		}
		_, count, err = svc.QueryEvents(ctx, QueryEventOpts{Tool: "BASH"})
		if err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Errorf("tool count=%d, want 2", count)
		}
		_, count, err = svc.QueryEvents(ctx, QueryEventOpts{Command: "go %"})
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("command count=%d, want 1 (go test)", count)
		}
	})

	t.Run("time window", func(t *testing.T) {
		// Baseline: all non-sensitive events.
		_, count, err := svc.QueryEvents(ctx, QueryEventOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if count != 5 {
			t.Fatalf("baseline count=%d, want 5", count)
		}
		// A window in the far future should exclude everything.
		_, futureCount, err := svc.QueryEvents(ctx, QueryEventOpts{Since: "2099-01-01T00:00:00Z"})
		if err != nil {
			t.Fatal(err)
		}
		if futureCount != 0 {
			t.Errorf("future window count=%d, want 0", futureCount)
		}
	})
}

// TestExportEvents verifies JSONL export of matching events.
// SATISFIES: export-jsonl.
func TestExportEvents(t *testing.T) {
	svc, _ := openTracerService(t)
	svc.SetHooks(HooksConfig{Enabled: true})
	ctx := context.Background()

	sid, err := svc.SessionStart(ctx, t.TempDir(), "export")
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if _, err := svc.RunHook(ctx, HookPostToolUse, HookPayload{
		SessionID: sid, ToolName: "write", File: "src/main.go", ResultStatus: "success",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := svc.RunHook(ctx, HookPostToolUse, HookPayload{
		SessionID: sid, ToolName: "bash", Command: "ls -la", ResultStatus: "success",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	jsonl, err := svc.ExportEvents(ctx, QueryEventOpts{Session: sid})
	if err != nil {
		t.Fatal(err)
	}
	if jsonl == "" {
		t.Fatal("empty export")
	}
	lines := 0
	for i := 0; i < len(jsonl); i++ {
		if jsonl[i] == '\n' {
			lines++
		}
	}
	// 2 tool-call events (session_start is also included but has no tool_name
	// so it's still an event row). 3 events total, 2 newline separators.
	if lines != 2 {
		t.Errorf("expected 3 lines (2 newlines), got %d newlines", lines)
	}
}
