package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// showRepoGit runs git in dir, failing the test on error.
func showRepoGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// showRepoInit creates a temp git repo with one commit and returns its dir
// plus HEAD. Commits pass identity via -c flags with --no-verify so the test
// never touches global git config or the template's commit-msg hook.
func showRepoInit(t *testing.T) (dir, head string) {
	t.Helper()
	dir = t.TempDir()
	showRepoGit(t, dir, "init")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	showRepoGit(t, dir, "add", ".")
	showRepoGit(t, dir, "-c", "user.email=show@test", "-c", "user.name=show", "commit", "--no-verify", "-m", "first")
	return dir, showRepoGit(t, dir, "rev-parse", "HEAD")
}

// showRepoCommit appends one commit to the repo at dir and returns the new HEAD.
func showRepoCommit(t *testing.T, dir, name, body string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	showRepoGit(t, dir, "add", ".")
	showRepoGit(t, dir, "-c", "user.email=show@test", "-c", "user.name=show", "commit", "--no-verify", "-m", name)
	return showRepoGit(t, dir, "rev-parse", "HEAD")
}

// TestSessionShow is G1 (resume-from-events happy path, CLI): for a session
// with a start entry and an end entry, `skillgrid session <id>` prints every
// entry in position order with the net start-to-end change summary;
// --show-diff adds git diff --stat from..to; --json emits the same shape as
// JSON. The session is seeded straight into the store file the CLI opens via
// --project/--dir (SessionChanges reads by id, no project filter).
func TestSessionShow(t *testing.T) {
	dataDir, proj, st := sessionCLIFixture(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	repoDir, head1 := showRepoInit(t)
	svc := memory.New(st, proj)
	sid, err := svc.SessionStart(context.Background(), repoDir, "show-probe")
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	head2 := showRepoCommit(t, repoDir, "b.txt", "two\n")
	if err := svc.SessionEnd(context.Background(), sid, "done"); err != nil {
		t.Fatalf("session end: %v", err)
	}

	t.Run("table", func(t *testing.T) {
		out, code := runSessionCLI(t, dataDir, proj, cwd,
			sid, "--project", proj, "--dir", dataDir)
		if code != 0 {
			t.Fatalf("session show exited non-zero: %d\n%s", code, out)
		}
		for _, want := range []string{sid, head1, head2, "session_start", "session_end"} {
			if !strings.Contains(out, want) {
				t.Errorf("show output should contain %q, got:\n%s", want, out)
			}
		}
		// Sequence order: start before end.
		if strings.Index(out, "session_start") > strings.Index(out, "session_end") {
			t.Errorf("events out of order in:\n%s", out)
		}
	})

	t.Run("diff", func(t *testing.T) {
		out, code := runSessionCLI(t, dataDir, proj, cwd,
			sid, "--show-diff", "--project", proj, "--dir", dataDir)
		if code != 0 {
			t.Fatalf("session show --show-diff exited non-zero: %d\n%s", code, out)
		}
		if !strings.Contains(out, "b.txt") {
			t.Errorf("--show-diff should include the diff --stat (b.txt), got:\n%s", out)
		}
	})

	t.Run("json", func(t *testing.T) {
		out, code := runSessionCLI(t, dataDir, proj, cwd,
			sid, "--json", "--show-diff", "--project", proj, "--dir", dataDir)
		if code != 0 {
			t.Fatalf("session show --json exited non-zero: %d\n%s", code, out)
		}
		var parsed struct {
			SessionID  string `json:"session_id"`
			FromCommit string `json:"from_commit"`
			ToCommit   string `json:"to_commit"`
			Events     []struct {
				Sequence   int    `json:"sequence"`
				ActionType string `json:"action_type"`
			} `json:"events"`
			DiffStat string `json:"diff_stat"`
		}
		if err := json.Unmarshal([]byte(out), &parsed); err != nil {
			t.Fatalf("unmarshal show --json: %v (out %s)", err, out)
		}
		if parsed.SessionID != sid || parsed.FromCommit != head1 || parsed.ToCommit != head2 {
			t.Errorf("json header = (%q, %q, %q), want (%q, %q, %q)",
				parsed.SessionID, parsed.FromCommit, parsed.ToCommit, sid, head1, head2)
		}
		if len(parsed.Events) != 2 ||
			parsed.Events[0].ActionType != "session_start" || parsed.Events[0].Sequence != 0 ||
			parsed.Events[1].ActionType != "session_end" || parsed.Events[1].Sequence != 1 {
			t.Errorf("json events out of order: %+v", parsed.Events)
		}
		if !strings.Contains(parsed.DiffStat, "b.txt") {
			t.Errorf("json diff_stat should mention b.txt, got %q", parsed.DiffStat)
		}
	})

	t.Run("quiet", func(t *testing.T) {
		// Resume-from-events-quiet-session: start + end with no tool entries
		// and an empty range (outside a repo) — --show-diff prints a note
		// and never errors.
		plainDir := t.TempDir()
		quietID, err := svc.SessionStart(context.Background(), plainDir, "quiet-probe")
		if err != nil {
			t.Fatalf("quiet start: %v", err)
		}
		if err := svc.SessionEnd(context.Background(), quietID, "done"); err != nil {
			t.Fatalf("quiet end: %v", err)
		}
		out, code := runSessionCLI(t, dataDir, proj, cwd,
			quietID, "--show-diff", "--project", proj, "--dir", dataDir)
		if code != 0 {
			t.Fatalf("quiet session show exited non-zero: %d\n%s", code, out)
		}
		if !strings.Contains(out, "session_start") || !strings.Contains(out, "session_end") {
			t.Errorf("quiet show should list start + end, got:\n%s", out)
		}
		if !strings.Contains(out, "no commit range") {
			t.Errorf("quiet --show-diff should note the empty range, got:\n%s", out)
		}
	})
}

// TestSessionShowUnknownID is G3 (resume-from-events-unknown-session): an
// unknown id errors cleanly (non-zero exit + session-not-found on stderr).
func TestSessionShowUnknownID(t *testing.T) {
	dataDir, proj, _ := sessionCLIFixture(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	out, code := runSessionCLI(t, dataDir, proj, cwd,
		"no-such-session", "--project", proj, "--dir", dataDir)
	if code == 0 {
		t.Fatalf("show with an unknown id should exit non-zero, got output: %s", out)
	}
	if !strings.Contains(out, "not found") {
		t.Fatalf("show unknown id should carry a session-not-found message, got: %s", out)
	}
}
