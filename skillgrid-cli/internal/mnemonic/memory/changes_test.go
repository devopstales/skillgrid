package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// runGit runs git in dir, failing the test on error. Commits pass identity via
// -c flags so the test never touches global git config.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// TestSessionChangesIgnoresStaleCheckpoint: a checkpoint.json left on disk from
// before the events-layer consolidation is ignored — a session's position comes
// from its event stream, not the stale file. A positive control proves the read
// path returns the real commit range, and an unknown session still errors with
// the file present. SATISFIES old-surfaces-removed-stale-file.
//
// Runs in its own process: openTracerService pins MNEMONIC_PROJECT via
// t.Setenv, which is not goroutine-safe under the parallel test scheduler — a
// leaked override from a sibling test would flip project.Resolve and shift
// from_commit. A dedicated process keeps the env pin intact for this test.
func TestSessionChangesIgnoresStaleCheckpoint(t *testing.T) {
	t.Setenv("GO_TEST_PROCESS", "1")
	svc, _ := openTracerService(t)
	ctx := context.Background()

	repoDir := initGitRepo(t)
	head1 := runGit(t, repoDir, "rev-parse", "HEAD")
	commitFile(t, repoDir, "b.txt", "two\n")
	head2 := runGit(t, repoDir, "rev-parse", "HEAD")

	sid, err := svc.SessionStart(ctx, repoDir, "stale-checkpoint")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := svc.SessionEnd(ctx, sid, ""); err != nil {
		t.Fatalf("end: %v", err)
	}

	// Stale checkpoint from before the consolidation. from_commit carries the
	// PRE-session commit (head1) — a reader of the file would resume at head1
	// instead of the session's real event-derived from_commit, so the
	// distinction is non-tautological (head1 != head2).
	stale := map[string]any{
		"version":     1,
		"from_commit": head1,
		"to_commit":   head2,
	}
	staleJSON, err := json.Marshal(stale)
	if err != nil {
		t.Fatalf("marshal stale checkpoint: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "checkpoint.json"), staleJSON, 0o644); err != nil {
		t.Fatalf("write stale checkpoint.json: %v", err)
	}

	// Position comes from the event stream, not the stale file.
	evts, from, to, err := svc.SessionChanges(ctx, sid)
	if err != nil {
		t.Fatalf("changes: %v", err)
	}
	if from != head2 || to != head2 {
		t.Errorf("change range = (%q, %q), want (%q, %q) from events — stale checkpoint.json (from=%s) was not ignored",
			from, to, head2, head2, head1)
	}
	if len(evts) != 2 {
		t.Errorf("events = %d, want 2 (start + end) from the event stream", len(evts))
	}
	if evts[0].ActionType != "session_start" || evts[1].ActionType != "session_end" {
		t.Errorf("event stream = (%q, %q), want (session_start, session_end)",
			evts[0].ActionType, evts[1].ActionType)
	}

	// Unknown session still errors with the stale file present (the file is not
	// a fallback source for session position).
	if _, _, _, err := svc.SessionChanges(ctx, "no-such-session"); err == nil ||
		!strings.Contains(err.Error(), "not found") {
		t.Errorf("unknown session err = %v, want session-not-found (stale file present)", err)
	}
}

// initGitRepo creates a temp git repo with one commit and returns its dir.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, dir, "add", ".")
	// --no-verify: the test repo inherits the global init template's
	// commit-msg hook (conventional-commits), which the fixture subjects
	// intentionally do not follow.
	runGit(t, dir, "-c", "user.email=tracer@test", "-c", "user.name=tracer", "commit", "--no-verify", "-m", "first")
	return dir
}

// commitFile appends a commit to the repo at dir.
func commitFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.email=tracer@test", "-c", "user.name=tracer", "commit", "--no-verify", "-m", name)
}

// openTracerService opens a store pinned to project "tracer" with session
// project resolution pinned there too (MNEMONIC_PROJECT override), so
// SessionStart in any directory lands in the same bucket as SessionEnd.
func openTracerService(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	t.Setenv("MNEMONIC_PROJECT", "tracer")
	st, err := store.Open(t.TempDir(), "tracer")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return New(st, "tracer"), st
}

// TestSessionStartEndEvents is the TICKET-01 tracer thread: start writes the
// sessions row + session_start at sequence 0 with the git HEAD, end writes
// session_end + to_commit + ended_at, and SessionChanges returns start/end in
// order with the from/to commits. It also covers the outside-repo,
// unknown-session, and ByClientID paths.
func TestSessionStartEndEvents(t *testing.T) {
	svc, st := openTracerService(t)
	ctx := context.Background()

	repoDir := initGitRepo(t)
	head1 := runGit(t, repoDir, "rev-parse", "HEAD")
	if head1 == "" {
		t.Fatalf("expected non-empty HEAD after first commit")
	}

	sid, err := svc.SessionStart(ctx, repoDir, "tracer")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if sid == "" {
		t.Fatalf("expected non-empty session id")
	}

	// sessions row carries the start commit, active status, agent identity.
	var fromCommit, status, agentID string
	var toRaw, endedRaw sql.NullString
	if err := st.DB.QueryRow(
		`SELECT from_commit, status, agent_session_id, to_commit, ended_at FROM sessions WHERE id = ?`,
		sid).Scan(&fromCommit, &status, &agentID, &toRaw, &endedRaw); err != nil {
		t.Fatalf("read session row: %v", err)
	}
	if fromCommit != head1 {
		t.Errorf("from_commit = %q, want HEAD %q", fromCommit, head1)
	}
	if status != "active" {
		t.Errorf("status = %q, want active", status)
	}
	if agentID != sid {
		t.Errorf("agent_session_id = %q, want session id %q", agentID, sid)
	}

	// After start: one event (session_start, sequence 0), range open.
	evts, from, to, err := svc.SessionChanges(ctx, sid)
	if err != nil {
		t.Fatalf("changes after start: %v", err)
	}
	if from != head1 {
		t.Errorf("changes from = %q, want %q", from, head1)
	}
	if to != "" {
		t.Errorf("changes to = %q, want empty before end", to)
	}
	if len(evts) != 1 {
		t.Fatalf("expected 1 event after start, got %d", len(evts))
	}
	if evts[0].ActionType != "session_start" || evts[0].Sequence != 0 {
		t.Errorf("first event = (%q, seq %d), want (session_start, seq 0)",
			evts[0].ActionType, evts[0].Sequence)
	}
	if evts[0].Commit != head1 {
		t.Errorf("start event commit = %q, want %q", evts[0].Commit, head1)
	}

	// Advance the repo, then end the session.
	commitFile(t, repoDir, "b.txt", "two\n")
	head2 := runGit(t, repoDir, "rev-parse", "HEAD")
	if head2 == head1 {
		t.Fatalf("expected HEAD to advance after second commit")
	}
	if err := svc.SessionEnd(ctx, sid, "done"); err != nil {
		t.Fatalf("end: %v", err)
	}

	var toCommit, endedAt, endStatus string
	if err := st.DB.QueryRow(
		`SELECT COALESCE(to_commit,''), COALESCE(ended_at,''), status FROM sessions WHERE id = ?`,
		sid).Scan(&toCommit, &endedAt, &endStatus); err != nil {
		t.Fatalf("read ended session: %v", err)
	}
	if toCommit != head2 {
		t.Errorf("to_commit = %q, want HEAD %q", toCommit, head2)
	}
	if endedAt == "" {
		t.Errorf("expected ended_at to be set")
	}
	if endStatus != "ended" {
		t.Errorf("status = %q, want ended", endStatus)
	}

	// After end: start + end in order, net range head1..head2.
	evts, from, to, err = svc.SessionChanges(ctx, sid)
	if err != nil {
		t.Fatalf("changes after end: %v", err)
	}
	if from != head1 || to != head2 {
		t.Errorf("range = %q..%q, want %q..%q", from, to, head1, head2)
	}
	if len(evts) != 2 {
		t.Fatalf("expected 2 events after end, got %d", len(evts))
	}
	if evts[0].ActionType != "session_start" || evts[0].Sequence != 0 {
		t.Errorf("event[0] = (%q, seq %d), want (session_start, 0)",
			evts[0].ActionType, evts[0].Sequence)
	}
	if evts[1].ActionType != "session_end" || evts[1].Sequence != 1 {
		t.Errorf("event[1] = (%q, seq %d), want (session_end, 1)",
			evts[1].ActionType, evts[1].Sequence)
	}
	if evts[1].Commit != head2 {
		t.Errorf("end event commit = %q, want %q", evts[1].Commit, head2)
	}

	// Unknown session: end and changes both fail with session-not-found.
	if err := svc.SessionEnd(ctx, "no-such-session", ""); err == nil ||
		!strings.Contains(err.Error(), "not found") {
		t.Errorf("end unknown session err = %v, want session-not-found", err)
	}
	if _, _, _, err := svc.SessionChanges(ctx, "no-such-session"); err == nil ||
		!strings.Contains(err.Error(), "not found") {
		t.Errorf("changes unknown session err = %v, want session-not-found", err)
	}

	// Outside a repo: start succeeds with an empty change range, no error.
	plainDir := t.TempDir()
	outID, err := svc.SessionStart(ctx, plainDir, "outside")
	if err != nil {
		t.Fatalf("start outside repo: %v", err)
	}
	evts, from, to, err = svc.SessionChanges(ctx, outID)
	if err != nil {
		t.Fatalf("changes outside repo: %v", err)
	}
	if from != "" || to != "" {
		t.Errorf("outside-repo range = %q..%q, want empty", from, to)
	}
	if len(evts) != 1 || evts[0].ActionType != "session_start" {
		t.Errorf("outside-repo events = %+v, want single session_start", evts)
	}

	// ByClientID path: same start-event + range contract under the caller id.
	clientID := "client-uuid-1"
	gotID, _, existed, err := svc.SessionStartByClientID(ctx, clientID, repoDir, "client")
	if err != nil {
		t.Fatalf("start by client id: %v", err)
	}
	if gotID != clientID || existed {
		t.Errorf("by-client-id = (%q, existed=%v), want (%q, false)", gotID, existed, clientID)
	}
	evts, from, _, err = svc.SessionChanges(ctx, clientID)
	if err != nil {
		t.Fatalf("changes by client id: %v", err)
	}
	if from == "" {
		t.Errorf("by-client-id from commit empty, want repo HEAD")
	}
	if len(evts) != 1 || evts[0].Sequence != 0 || evts[0].ActionType != "session_start" {
		t.Errorf("by-client-id events = %+v, want single session_start seq 0", evts)
	}
	// Idempotent re-entry: no duplicate start event.
	if _, _, existed, err := svc.SessionStartByClientID(ctx, clientID, repoDir, "client"); err != nil || !existed {
		t.Errorf("re-entry = (existed=%v, err=%v), want (true, nil)", existed, err)
	}
	evts, _, _, err = svc.SessionChanges(ctx, clientID)
	if err != nil {
		t.Fatalf("changes after re-entry: %v", err)
	}
	if len(evts) != 1 {
		t.Errorf("re-entry appended a duplicate start event: %+v", evts)
	}
}
