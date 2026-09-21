package memory

// TICKET-02 tracer thread: post_tool_use hook branch → ordered event rows,
// counter bumps in-transaction, sensitive redaction with hash/preview-only
// storage. SATISFIES tool-call-stream, sensitive-redaction.

import (
	"context"
	"strings"
	"testing"
)

// enableToolHooks opens a tracer service with lifecycle hooks enabled (RunHook
// rejects everything while the opt-in switch is off) and starts one session.
func enableToolHooks(t *testing.T) (*Service, string) {
	t.Helper()
	svc, _ := openTracerService(t)
	svc.SetHooks(HooksConfig{Enabled: true})
	sid, err := svc.SessionStart(context.Background(), t.TempDir(), "tools")
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	return svc, sid
}

// sessionCounters reads the per-session activity counters for assertions.
func sessionCounters(t *testing.T, svc *Service, sid string) (filesRead, filesWritten, commandsExec, errors, sensitive int) {
	t.Helper()
	if err := svc.store.DB.QueryRow(
		`SELECT files_read, files_written, commands_exec, errors, sensitive_actions
		 FROM sessions WHERE id = ?`, sid,
	).Scan(&filesRead, &filesWritten, &commandsExec, &errors, &sensitive); err != nil {
		t.Fatalf("read counters: %v", err)
	}
	return filesRead, filesWritten, commandsExec, errors, sensitive
}

// TestPostToolUseAppendsOrderedEvents: two hook calls (file write, then shell)
// yield sequences 1 and 2 with matching counters bumped; an unknown hook type
// is rejected with no entry appended; an unknown session id is session-not-found.
func TestPostToolUseAppendsOrderedEvents(t *testing.T) {
	svc, sid := enableToolHooks(t)
	ctx := context.Background()

	if _, err := svc.RunHook(ctx, HookPostToolUse, HookPayload{
		SessionID: sid, ToolName: "Write", File: "src/main.go",
		ContentPreview: "package main\n",
	}); err != nil {
		t.Fatalf("write hook: %v", err)
	}
	if _, err := svc.RunHook(ctx, HookPostToolUse, HookPayload{
		SessionID: sid, ToolName: "Shell", Command: "go test ./...",
	}); err != nil {
		t.Fatalf("shell hook: %v", err)
	}

	evts, _, _, err := svc.SessionChanges(ctx, sid)
	if err != nil {
		t.Fatalf("changes: %v", err)
	}
	if len(evts) != 3 {
		t.Fatalf("expected 3 events (start + 2 tool calls), got %d: %+v", len(evts), evts)
	}
	if evts[1].ActionType != "file_write" || evts[1].Sequence != 1 {
		t.Errorf("event[1] = (%q, seq %d), want (file_write, 1)",
			evts[1].ActionType, evts[1].Sequence)
	}
	if evts[1].ToolName != "Write" || evts[1].Path != "src/main.go" {
		t.Errorf("event[1] detail = tool %q path %q, want (Write, src/main.go)",
			evts[1].ToolName, evts[1].Path)
	}
	if evts[2].ActionType != "command_exec" || evts[2].Sequence != 2 {
		t.Errorf("event[2] = (%q, seq %d), want (command_exec, 2)",
			evts[2].ActionType, evts[2].Sequence)
	}
	if evts[2].Command != "go test ./..." {
		t.Errorf("event[2] command = %q, want go test ./...", evts[2].Command)
	}

	_, filesWritten, commandsExec, _, _ := sessionCounters(t, svc, sid)
	if filesWritten != 1 {
		t.Errorf("files_written = %d, want 1", filesWritten)
	}
	if commandsExec != 1 {
		t.Errorf("commands_exec = %d, want 1", commandsExec)
	}

	// Rejected: unknown hook type appends nothing.
	if _, err := svc.RunHook(ctx, "no-such-hook", HookPayload{SessionID: sid}); err == nil {
		t.Errorf("unknown hook type: expected error, got nil")
	}
	evts, _, _, err = svc.SessionChanges(ctx, sid)
	if err != nil {
		t.Fatalf("changes after rejection: %v", err)
	}
	if len(evts) != 3 {
		t.Errorf("rejected hook appended an entry: %d events, want 3", len(evts))
	}

	// Unknown session: session-not-found, no insert possible.
	if _, err := svc.RunHook(ctx, HookPostToolUse, HookPayload{
		SessionID: "no-such-session", ToolName: "Read", File: "x.go",
	}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("unknown session err = %v, want session-not-found", err)
	}
}

// TestSensitiveWriteRedacted: a file write to config/.env flags is_sensitive=1
// and stores only hash + masked preview (never the full secret); the
// sensitive-actions counter is 1. A clean-path write is unflagged with the
// counter at 0, and the matcher covers the gryph set.
func TestSensitiveWriteRedacted(t *testing.T) {
	svc, sid := enableToolHooks(t)
	ctx := context.Background()
	secret := "SECRET=abc123"

	if _, err := svc.RunHook(ctx, HookPostToolUse, HookPayload{
		SessionID: sid, ToolName: "Write", File: "config/.env",
		ContentPreview: secret + "\n",
	}); err != nil {
		t.Fatalf("env write hook: %v", err)
	}
	evts, _, _, err := svc.SessionChanges(ctx, sid)
	if err != nil {
		t.Fatalf("changes: %v", err)
	}
	if len(evts) != 2 {
		t.Fatalf("expected 2 events, got %d", len(evts))
	}
	env := evts[1]
	if !env.IsSensitive {
		t.Errorf(".env write: is_sensitive = false, want true")
	}
	if !strings.Contains(env.Payload, "content_hash") {
		t.Errorf(".env payload missing content_hash (positive control): %q", env.Payload)
	}
	if !strings.Contains(env.Payload, "SECRET=***") {
		t.Errorf(".env payload missing masked preview (positive control): %q", env.Payload)
	}
	// Absence oracle: the full secret value appears nowhere stored.
	if strings.Contains(env.Payload, secret) || strings.Contains(env.Payload, "abc123") {
		t.Errorf(".env payload leaks the secret: %q", env.Payload)
	}

	_, filesWritten, _, _, sensitive := sessionCounters(t, svc, sid)
	if filesWritten != 1 {
		t.Errorf("files_written = %d, want 1", filesWritten)
	}
	if sensitive != 1 {
		t.Errorf("sensitive_actions = %d, want 1", sensitive)
	}

	// Clean path: unflagged, counter for a fresh session stays 0.
	svc2, sid2 := enableToolHooks(t)
	if _, err := svc2.RunHook(ctx, HookPostToolUse, HookPayload{
		SessionID: sid2, ToolName: "Write", File: "src/main.go",
		ContentPreview: "package main\n",
	}); err != nil {
		t.Fatalf("clean write hook: %v", err)
	}
	evts2, _, _, err := svc2.SessionChanges(ctx, sid2)
	if err != nil {
		t.Fatalf("changes: %v", err)
	}
	if evts2[1].IsSensitive {
		t.Errorf("src/main.go write: is_sensitive = true, want false")
	}
	if _, _, _, _, sensitive := sessionCounters(t, svc2, sid2); sensitive != 0 {
		t.Errorf("clean session sensitive_actions = %d, want 0", sensitive)
	}

	// Matcher gryph set (unit level).
	sensitivePaths := []string{
		".env", "config/.env", ".env.local",
		"tls/server.pem", "id_rsa.key", "certs/key.PEM",
		"config/my-secret.yaml", "secrets/token.txt",
		"/home/u/.ssh/config", "~/.aws/credentials",
	}
	for _, p := range sensitivePaths {
		if !isSensitivePath(p) {
			t.Errorf("isSensitivePath(%q) = false, want true", p)
		}
	}
	cleanPaths := []string{"src/main.go", "README.md", "my.aws.backup", "src/keyboard.go"}
	for _, p := range cleanPaths {
		if isSensitivePath(p) {
			t.Errorf("isSensitivePath(%q) = true, want false", p)
		}
	}
}
