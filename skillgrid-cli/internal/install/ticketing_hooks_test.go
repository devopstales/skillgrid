package install

// End-to-end tests for the tracker-aware git hooks (hooks/ticketing*.js,
// wired through hooks/checkpoint-state.js). Like hooks_sh_test.go, these run
// the real node scripts against throwaway git repos. Skips cleanly when node
// or git is unavailable.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runNode runs a repo hook script with args, env extras, and optional stdin,
// returning (exitCode, stdout, stderr). The command runs with cwd=dir so the
// hooks resolve the throwaway repo (not the skillgrid checkout).
func runNode(t *testing.T, dir, script string, args []string, env []string, stdin string) (int, string, string) {
	t.Helper()
	cmdArgs := append([]string{script}, args...)
	cmd := exec.Command("node", cmdArgs...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("running %s: %v", script, err)
		}
	}
	return code, out.String(), errb.String()
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const ticketingTestConfig = `schema: skillgrid/v1
project: t
ticketing:
  enabled: true
  type: backlogmd
  backlogmd:
    dir: .backlog
`

const ticketingCompleteTask = `---
id: TASK-1
title: '[FEATURE] probe (hooks)'
status: needs-triage
type: feature
references:
  - hooks/ticketing.js
priority: medium
---

## Definition of Done

<!-- DOD:BEGIN -->
- [ ] Tests pass
<!-- DOD:END -->
`

const ticketingStubTask = `---
id: TASK-999
title: stub
status: needs-triage
---

## Description

text
`

// newTicketingRepo builds a temp git repo with a ticketing config and returns
// its root.
func newTicketingRepo(t *testing.T, config string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".skillgrid", "config.yaml"), config)
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@skillgrid.local"},
		{"config", "user.name", "Skillgrid Test"},
		{"add", "-A"},
		{"commit", "-q", "-m", "chore: init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	return root
}

func TestGuardMsgAcceptsValidTrackerFooters(t *testing.T) {
	requireTools(t, "node", "git")
	hooksDir := repoHooksDir(t)
	state := filepath.Join(hooksDir, "checkpoint-state.js")
	cases := map[string]string{
		"backlog refs":   "feat(hooks): probe\n\nBody.\n\nRefs: TASK-069\n",
		"backlog closes": "fix(hooks): probe\n\nCloses: TASK-019.03\n",
		"gh closes":      "fix(api): probe\n\nCloses #42\n",
		"jira refs":      "feat(api): probe\n\nRefs: PROJ-123\n",
		"no footer":      "feat(hooks): probe without a tracker link\n",
	}
	for name, msg := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			msgFile := filepath.Join(dir, "msg.txt")
			writeFile(t, msgFile, msg)
			code, _, stderr := runNode(t, dir, state, []string{"guard-msg", msgFile}, nil, "")
			if code != 0 {
				t.Fatalf("expected exit 0, got %d: %s", code, stderr)
			}
		})
	}
}

func TestGuardMsgRejectsMalformedTrackerFooter(t *testing.T) {
	requireTools(t, "node", "git")
	hooksDir := repoHooksDir(t)
	state := filepath.Join(hooksDir, "checkpoint-state.js")
	for _, msg := range []string{
		"feat(hooks): probe\n\nRefs: someday-maybe\n",
		"fix(hooks): probe\n\nCloses: TBD\n",
	} {
		dir := t.TempDir()
		msgFile := filepath.Join(dir, "msg.txt")
		writeFile(t, msgFile, msg)
		code, _, stderr := runNode(t, dir, state, []string{"guard-msg", msgFile}, nil, "")
		if code != 1 {
			t.Fatalf("expected exit 1, got %d: %s", code, stderr)
		}
		if !strings.Contains(stderr, "Tracker ID") {
			t.Errorf("expected 'Tracker ID' in stderr, got: %s", stderr)
		}
	}
}

func TestTicketingGuardBlocksIncompleteTask(t *testing.T) {
	requireTools(t, "node", "git")
	hooksDir := repoHooksDir(t)
	root := newTicketingRepo(t, ticketingTestConfig)
	writeFile(t, filepath.Join(root, ".backlog", "tasks", "TASK-999.md"), ticketingStubTask)
	if out, err := gitIn(root, "add", ".backlog/tasks/TASK-999.md").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}
	code, _, stderr := runNode(t, root, filepath.Join(hooksDir, "ticketing-guard.js"), nil, nil, "")
	if code != 1 {
		t.Fatalf("expected exit 1 for a stub task, got %d: %s", code, stderr)
	}
	if !strings.Contains(stderr, "completeness gate") {
		t.Errorf("expected 'completeness gate' in stderr, got: %s", stderr)
	}
}

func TestTicketingGuardPassesCompleteTask(t *testing.T) {
	requireTools(t, "node", "git")
	hooksDir := repoHooksDir(t)
	root := newTicketingRepo(t, ticketingTestConfig)
	writeFile(t, filepath.Join(root, ".backlog", "tasks", "TASK-1.md"), ticketingCompleteTask)
	if out, err := gitIn(root, "add", ".backlog/tasks/TASK-1.md").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}
	// No Implementation Plan section: WARNING on stderr, but exit 0 (the CLI
	// itself creates plan-less triage tasks, so the plan never blocks).
	code, _, stderr := runNode(t, root, filepath.Join(hooksDir, "ticketing-guard.js"), nil, nil, "")
	if code != 0 {
		t.Fatalf("expected exit 0 for a complete task, got %d: %s", code, stderr)
	}
	if !strings.Contains(stderr, "Implementation Plan") {
		t.Errorf("expected plan WARNING in stderr, got: %s", stderr)
	}
}

func TestTicketingGuardFailOpenWhenDisabled(t *testing.T) {
	requireTools(t, "node", "git")
	hooksDir := repoHooksDir(t)
	root := newTicketingRepo(t, "schema: skillgrid/v1\nproject: t\nticketing:\n  enabled: false\n  type: backlogmd\n")
	writeFile(t, filepath.Join(root, ".backlog", "tasks", "TASK-999.md"), ticketingStubTask)
	if out, err := gitIn(root, "add", "-A").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}
	code, _, stderr := runNode(t, root, filepath.Join(hooksDir, "ticketing-guard.js"), nil, nil, "")
	if code != 0 {
		t.Fatalf("expected fail-open exit 0 when disabled, got %d: %s", code, stderr)
	}
}

func TestPrepareMsgInjectsRef(t *testing.T) {
	requireTools(t, "node", "git")
	hooksDir := repoHooksDir(t)
	state := filepath.Join(hooksDir, "checkpoint-state.js")

	dir := t.TempDir()
	msgFile := filepath.Join(dir, "msg.txt")
	writeFile(t, msgFile, "feat(hooks): probe\n")
	code, _, stderr := runNode(t, dir, state, []string{"prepare-msg", msgFile}, []string{"SKILLGRID_TRACKER_REF=TASK-069"}, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr)
	}
	body, err := os.ReadFile(msgFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Refs: TASK-069") {
		t.Errorf("expected injected Refs footer, got: %q", body)
	}

	// No env: no-op, message untouched.
	writeFile(t, msgFile, "feat(hooks): probe\n")
	code, _, _ = runNode(t, dir, state, []string{"prepare-msg", msgFile}, nil, "")
	if code != 0 {
		t.Fatalf("expected exit 0 without env, got %d", code)
	}
	body, _ = os.ReadFile(msgFile)
	if strings.Contains(string(body), "Refs:") {
		t.Errorf("expected message untouched without env, got: %q", body)
	}
}

func TestPrePushBlocksMalformedCloses(t *testing.T) {
	requireTools(t, "node", "git")
	hooksDir := repoHooksDir(t)
	root := newTicketingRepo(t, ticketingTestConfig)
	if out, err := gitIn(root, "commit", "-q", "--allow-empty", "-m", "fix(t): bad close\n\nCloses: someday-maybe").CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, out)
	}
	shaOut, err := gitIn(root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	sha := strings.TrimSpace(string(shaOut))
	refLine := "refs/heads/main " + sha + " refs/heads/main 0000000000000000000000000000000000000000\n"
	script := filepath.Join(hooksDir, "ticketing-pre-push.js")
	code, _, stderr := runNode(t, root, script, []string{"origin", "url"}, nil, refLine)
	if code != 1 {
		t.Fatalf("expected exit 1 for malformed Closes, got %d: %s", code, stderr)
	}

	// Advisory Refs with no ID: warning only, exit 0 (amend so the range
	// holds a single footer-bearing commit).
	if out, err := gitIn(root, "commit", "-q", "--amend", "--allow-empty", "-m", "feat(t): advisory\n\nRefs: someday-maybe").CombinedOutput(); err != nil {
		t.Fatalf("git commit --amend: %v: %s", err, out)
	}
	shaOut, err = gitIn(root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	sha = strings.TrimSpace(string(shaOut))
	refLine = "refs/heads/main " + sha + " refs/heads/main 0000000000000000000000000000000000000000\n"
	code, _, stderr = runNode(t, root, script, []string{"origin", "url"}, nil, refLine)
	if code != 0 {
		t.Fatalf("expected exit 0 for advisory Refs, got %d: %s", code, stderr)
	}
}
