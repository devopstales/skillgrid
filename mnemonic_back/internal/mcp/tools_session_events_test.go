package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// eventsRepoInit creates a temp git repo with one commit and returns its dir
// plus HEAD. Commits pass identity via -c flags and --no-verify so the test
// never touches global git config or the repo template's commit-msg hook.
func eventsRepoInit(t *testing.T) (dir, head string) {
	t.Helper()
	dir = t.TempDir()
	run := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	run("add", ".")
	run("-c", "user.email=events@test", "-c", "user.name=events", "commit", "--no-verify", "-m", "first")
	return dir, run("rev-parse", "HEAD")
}

// eventsRepoCommit appends one commit to the repo at dir and returns the new HEAD.
func eventsRepoCommit(t *testing.T, dir, name, body string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	run := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("add", ".")
	run("-c", "user.email=events@test", "-c", "user.name=events", "commit", "--no-verify", "-m", name)
	return run("rev-parse", "HEAD")
}

// seedEventsSession starts a session in repoDir, advances the repo, and ends
// the session, so the stream holds start + end with a non-empty commit range.
func seedEventsSession(t *testing.T, repoDir string) (sessionID, head1, head2 string) {
	t.Helper()
	ctx := context.Background()
	_, h, cleanup, err := openService()
	if err != nil {
		t.Fatalf("openService: %v", err)
	}
	defer cleanup()
	head1Cmd := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD")
	head1Out, err := head1Cmd.Output()
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	head1 = strings.TrimSpace(string(head1Out))
	sid, err := h.Memory().SessionStart(ctx, repoDir, "changes-probe")
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	head2 = eventsRepoCommit(t, repoDir, "b.txt", "two\n")
	if err := h.Memory().SessionEnd(ctx, sid, "done"); err != nil {
		t.Fatalf("session end: %v", err)
	}
	return sid, head1, head2
}

type sessionChangesOut struct {
	SessionID  string `json:"session_id"`
	FromCommit string `json:"from_commit"`
	ToCommit   string `json:"to_commit"`
	Events     []struct {
		Sequence   int    `json:"sequence"`
		ActionType string `json:"action_type"`
		Commit     string `json:"commit"`
	} `json:"events"`
}

func callSessionChanges(t *testing.T, args map[string]any) sessionChangesOut {
	t.Helper()
	res, err := handleSessionChanges(context.Background(), newCallTool("session_changes", args))
	if err != nil {
		t.Fatalf("handleSessionChanges dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("session_changes errored: %s", callResultText(t, res))
	}
	var out sessionChangesOut
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("unmarshal session_changes: %v (text %s)", err, callResultText(t, res))
	}
	return out
}

// TestSessionChangesTool is G2 (resume-from-events happy path, MCP tool): for
// a seeded session with start + end entries, session_changes returns every
// entry in position order with the net start-to-end change summary.
func TestSessionChangesTool(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "sesschanges-probe")

	repoDir, _ := eventsRepoInit(t)
	sid, head1, head2 := seedEventsSession(t, repoDir)

	out := callSessionChanges(t, map[string]any{"session_id": sid})
	if out.SessionID != sid {
		t.Errorf("session_id = %q, want %q", out.SessionID, sid)
	}
	if out.FromCommit != head1 || out.ToCommit != head2 {
		t.Errorf("range = %q..%q, want %q..%q", out.FromCommit, out.ToCommit, head1, head2)
	}
	if len(out.Events) != 2 {
		t.Fatalf("expected 2 events (start + end), got %d: %+v", len(out.Events), out.Events)
	}
	if out.Events[0].ActionType != "session_start" || out.Events[0].Sequence != 0 {
		t.Errorf("event[0] = (%q, seq %d), want (session_start, 0)",
			out.Events[0].ActionType, out.Events[0].Sequence)
	}
	if out.Events[1].ActionType != "session_end" || out.Events[1].Sequence != 1 {
		t.Errorf("event[1] = (%q, seq %d), want (session_end, 1)",
			out.Events[1].ActionType, out.Events[1].Sequence)
	}
	if out.Events[0].Commit != head1 || out.Events[1].Commit != head2 {
		t.Errorf("event commits = (%q, %q), want (%q, %q)",
			out.Events[0].Commit, out.Events[1].Commit, head1, head2)
	}
}

// TestSessionChangesToolQuietSession covers resume-from-events-quiet-session:
// a session with start + end but no tool entries returns only those two with
// an empty change summary and no error.
func TestSessionChangesToolQuietSession(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "sesschanges-quiet")

	// Outside a repo: start + end with an empty range.
	plainDir := t.TempDir()
	ctx := context.Background()
	_, h, cleanup, err := openService()
	if err != nil {
		t.Fatalf("openService: %v", err)
	}
	sid, err := h.Memory().SessionStart(ctx, plainDir, "quiet-probe")
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	if err := h.Memory().SessionEnd(ctx, sid, "done"); err != nil {
		t.Fatalf("session end: %v", err)
	}
	cleanup()

	out := callSessionChanges(t, map[string]any{"session_id": sid})
	if len(out.Events) != 2 {
		t.Fatalf("expected only start + end, got %d: %+v", len(out.Events), out.Events)
	}
	if out.FromCommit != "" || out.ToCommit != "" {
		t.Errorf("quiet range = %q..%q, want empty", out.FromCommit, out.ToCommit)
	}
}

// TestSessionChangesToolUnknown is G3-adjacent (resume-from-events-unknown-
// session, MCP side): an unknown id fails with a session-not-found error.
func TestSessionChangesToolUnknown(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "sesschanges-unknown")

	res, err := handleSessionChanges(context.Background(), newCallTool("session_changes", map[string]any{
		"session_id": "no-such-session",
	}))
	if err != nil {
		t.Fatalf("handleSessionChanges dispatch: %v", err)
	}
	if !res.IsError {
		t.Fatalf("unknown session id should be an error, got: %s", callResultText(t, res))
	}
	if !strings.Contains(callResultText(t, res), "not found") {
		t.Fatalf("unknown session should carry a session-not-found error, got: %s", callResultText(t, res))
	}
}

// TestSessionChangesToolValidation rejects a missing session_id clearly, and
// the tool is registered under its session_* name.
func TestSessionChangesToolValidation(t *testing.T) {
	dataDir := t.TempDir()
	dir := t.TempDir()
	pinProjectCwd(t, dataDir, dir, "sesschanges-validation")

	if _, ok := NewServer().ListTools()["session_changes"]; !ok {
		t.Errorf("expected tool %q to be registered", "session_changes")
	}

	res, err := handleSessionChanges(context.Background(), newCallTool("session_changes", map[string]any{}))
	if err != nil {
		t.Fatalf("handleSessionChanges dispatch: %v", err)
	}
	if !res.IsError {
		t.Errorf("session_changes without session_id should be a validation error, got: %s", callResultText(t, res))
	}
}
