package memory

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// commitWithMessage stages name with body and commits using msg as the full
// commit message (multi-line context blocks need -F, not -m).
func commitWithMessage(t *testing.T, dir, name, body, msg string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	msgFile := filepath.Join(t.TempDir(), "msg.txt")
	if err := os.WriteFile(msgFile, []byte(msg), 0o644); err != nil {
		t.Fatalf("write msg file: %v", err)
	}
	runGit(t, dir, "add", name)
	runGit(t, dir, "-c", "user.email=tracer@test", "-c", "user.name=tracer",
		"commit", "--no-verify", "-F", msgFile)
}

// eventCount returns the session_events row count for sid.
func eventCount(t *testing.T, svc *Service, sid string) int {
	t.Helper()
	var n int
	if err := svc.store.DB.QueryRow(
		`SELECT COUNT(*) FROM session_events WHERE session_id = ?`, sid,
	).Scan(&n); err != nil {
		t.Fatalf("count events: %v", err)
	}
	return n
}

// TestCommitEventParsesContextBlock is the TICKET-03 tracer thread: a work-unit
// commit carrying a [skillgrid-context] block records a commit event with the
// sha + parsed payload; a block-less commit records with empty payload and no
// error; non-repo / unborn-HEAD paths error with no row. It also covers the
// empty-sessionID fallback (latest active session) and its not-found path.
func TestCommitEventParsesContextBlock(t *testing.T) {
	ctx := context.Background()

	const blockMsg = `feat: tracer work unit

[skillgrid-context]
Task: TASK-019.03
Decisions: use json payload on the commit event
Remaining: trigger wiring lands in a later wave
Tried: plain body first
[/skillgrid-context]
`

	t.Run("block commit parses", func(t *testing.T) {
		svc, _ := openTracerService(t)
		repoDir := initGitRepo(t)
		sid, err := svc.SessionStart(ctx, repoDir, "tracer")
		if err != nil {
			t.Fatalf("start: %v", err)
		}

		commitWithMessage(t, repoDir, "b.txt", "two\n", blockMsg)
		head := runGit(t, repoDir, "rev-parse", "HEAD")

		ev, err := svc.RecordCommitEvent(ctx, sid, repoDir)
		if err != nil {
			t.Fatalf("record: %v", err)
		}
		if ev.ActionType != "commit" {
			t.Errorf("action_type = %q, want commit", ev.ActionType)
		}
		if ev.Commit != head {
			t.Errorf("commit = %q, want HEAD %q", ev.Commit, head)
		}
		if ev.SessionID != sid {
			t.Errorf("session_id = %q, want %q", ev.SessionID, sid)
		}
		var got map[string]string
		if err := json.Unmarshal([]byte(ev.Payload), &got); err != nil {
			t.Fatalf("payload is not JSON %q: %v", ev.Payload, err)
		}
		for key, want := range map[string]string{
			"task":      "TASK-019.03",
			"decisions": "use json payload on the commit event",
			"remaining": "trigger wiring lands in a later wave",
			"tried":     "plain body first",
		} {
			if got[key] != want {
				t.Errorf("payload[%q] = %q, want %q", key, got[key], want)
			}
		}

		// Visible in sequence order via the resume read path.
		evts, _, _, err := svc.SessionChanges(ctx, sid)
		if err != nil {
			t.Fatalf("changes: %v", err)
		}
		if len(evts) != 2 || evts[1].ActionType != "commit" || evts[1].Commit != head {
			t.Errorf("changes events = %+v, want start + commit at HEAD", evts)
		}
		if evts[1].Payload == "" {
			t.Errorf("stored commit event has empty payload, want parsed block")
		}
	})

	t.Run("block-less commit records empty payload", func(t *testing.T) {
		svc, _ := openTracerService(t)
		repoDir := initGitRepo(t)
		sid, err := svc.SessionStart(ctx, repoDir, "tracer")
		if err != nil {
			t.Fatalf("start: %v", err)
		}

		commitFile(t, repoDir, "b.txt", "two\n") // plain "b.txt" subject, no block
		head := runGit(t, repoDir, "rev-parse", "HEAD")

		ev, err := svc.RecordCommitEvent(ctx, sid, repoDir)
		if err != nil {
			t.Fatalf("record block-less: %v", err)
		}
		if ev.Commit != head {
			t.Errorf("commit = %q, want HEAD %q", ev.Commit, head)
		}
		if ev.Payload != "" {
			t.Errorf("payload = %q, want empty for block-less commit", ev.Payload)
		}
	})

	t.Run("block without tried still parses", func(t *testing.T) {
		svc, _ := openTracerService(t)
		repoDir := initGitRepo(t)
		sid, err := svc.SessionStart(ctx, repoDir, "tracer")
		if err != nil {
			t.Fatalf("start: %v", err)
		}

		commitWithMessage(t, repoDir, "b.txt", "two\n", `fix: tracer unit

[skillgrid-context]
Task: TASK-019.03
Decisions: tried is optional
Remaining: done
[/skillgrid-context]
`)
		ev, err := svc.RecordCommitEvent(ctx, sid, repoDir)
		if err != nil {
			t.Fatalf("record: %v", err)
		}
		var got map[string]string
		if err := json.Unmarshal([]byte(ev.Payload), &got); err != nil {
			t.Fatalf("payload is not JSON %q: %v", ev.Payload, err)
		}
		if got["task"] != "TASK-019.03" || got["tried"] != "" {
			t.Errorf("payload = %v, want task set and tried empty", got)
		}
	})

	t.Run("empty session resolves latest active", func(t *testing.T) {
		svc, _ := openTracerService(t)
		repoDir := initGitRepo(t)
		first, err := svc.SessionStart(ctx, repoDir, "first")
		if err != nil {
			t.Fatalf("start first: %v", err)
		}
		second, err := svc.SessionStart(ctx, repoDir, "second")
		if err != nil {
			t.Fatalf("start second: %v", err)
		}

		commitFile(t, repoDir, "b.txt", "two\n")
		ev, err := svc.RecordCommitEvent(ctx, "", repoDir)
		if err != nil {
			t.Fatalf("record with fallback: %v", err)
		}
		if ev.SessionID != second {
			t.Errorf("resolved session = %q, want latest %q", ev.SessionID, second)
		}
		if n := eventCount(t, svc, first); n != 1 {
			t.Errorf("first session has %d events, want 1 (start only)", n)
		}
	})

	t.Run("empty session without active session is not-found", func(t *testing.T) {
		svc, _ := openTracerService(t)
		repoDir := initGitRepo(t) // no session started

		if _, err := svc.RecordCommitEvent(ctx, "", repoDir); err == nil ||
			!strings.Contains(err.Error(), "not found") {
			t.Errorf("fallback without session err = %v, want session-not-found", err)
		}
		var n int
		if err := svc.store.DB.QueryRow(`SELECT COUNT(*) FROM session_events`).Scan(&n); err != nil {
			t.Fatalf("count events: %v", err)
		}
		if n != 0 {
			t.Errorf("wrote %d events on not-found path, want 0", n)
		}
	})

	t.Run("unknown explicit session is not-found", func(t *testing.T) {
		svc, _ := openTracerService(t)
		repoDir := initGitRepo(t)

		if _, err := svc.RecordCommitEvent(ctx, "no-such-session", repoDir); err == nil ||
			!strings.Contains(err.Error(), "not found") {
			t.Errorf("unknown session err = %v, want session-not-found", err)
		}
	})

	t.Run("non-repo directory errors with no row", func(t *testing.T) {
		svc, _ := openTracerService(t)
		repoDir := initGitRepo(t)
		sid, err := svc.SessionStart(ctx, repoDir, "tracer")
		if err != nil {
			t.Fatalf("start: %v", err)
		}
		before := eventCount(t, svc, sid)

		plainDir := t.TempDir()
		if _, err := svc.RecordCommitEvent(ctx, sid, plainDir); err == nil ||
			!strings.Contains(err.Error(), "no commit to record") {
			t.Errorf("non-repo err = %v, want no-commit-to-record", err)
		}
		if n := eventCount(t, svc, sid); n != before {
			t.Errorf("events went %d -> %d on non-repo path, want no row", before, n)
		}
	})

	t.Run("unborn HEAD errors with no row", func(t *testing.T) {
		svc, _ := openTracerService(t)
		emptyDir := t.TempDir()
		runGit(t, emptyDir, "init") // no commits: HEAD is unborn

		if _, err := svc.RecordCommitEvent(ctx, "", emptyDir); err == nil ||
			!strings.Contains(err.Error(), "no commit to record") {
			t.Errorf("unborn HEAD err = %v, want no-commit-to-record", err)
		}
		var n int
		if err := svc.store.DB.QueryRow(`SELECT COUNT(*) FROM session_events`).Scan(&n); err != nil {
			t.Fatalf("count events: %v", err)
		}
		if n != 0 {
			t.Errorf("wrote %d events on unborn-HEAD path, want 0", n)
		}
	})
}
