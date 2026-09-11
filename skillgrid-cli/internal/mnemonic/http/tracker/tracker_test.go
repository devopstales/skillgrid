package tracker

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// 02.1 [RED] provider detection never guesses.
func TestStep02_Detection(t *testing.T) {
	// Explicit overrides win.
	for env, want := range map[string]string{
		"backlogmd":  ProviderBacklogMD,
		"backlog":    ProviderBacklogMD,
		"Backlog.md": ProviderBacklogMD,
		"github":     ProviderGitHub,
		"gh":         ProviderGitHub,
		"gitlab":     ProviderGitLab,
		"glab":       ProviderGitLab,
		"jira":       ProviderJira,
	} {
		got, err := ResolveProvider(env, "", "")
		if err != nil || got != want {
			t.Errorf("env %q: got %q, %v; want %q", env, got, err, want)
		}
	}
	// Unknown override → 501-class error, never a guess.
	if _, err := ResolveProvider("not-a-tracker", "", ""); err == nil {
		t.Error("unknown SKILLGRID_TRACKER must error, got nil")
	} else if _, ok := err.(*errUnresolvable); !ok {
		t.Errorf("unknown tracker must be errUnresolvable, got %T", err)
	}
	// Tracker doc wins over config.
	doc := "# Issue tracker: GitHub\n\nSome body."
	got, err := ResolveProvider("", doc, "backlogmd")
	if err != nil || got != ProviderGitHub {
		t.Errorf("tracker doc: got %q, %v; want github", got, err)
	}
	// Config value used when doc is absent.
	got, err = ResolveProvider("", "", "gitlab")
	if err != nil || got != ProviderGitLab {
		t.Errorf("config value: got %q, %v; want gitlab", got, err)
	}
	// Total silence → default backlogmd.
	got, err = ResolveProvider("", "", "")
	if err != nil || got != ProviderBacklogMD {
		t.Errorf("default: got %q, %v; want backlogmd", got, err)
	}
	// Tracker doc naming Jira resolves (project key handled by the adapter).
	got, err = ResolveProvider("", "# Issue tracker: Jira\n", "")
	if err != nil || got != ProviderJira {
		t.Errorf("jira doc: got %q, %v; want jira", got, err)
	}
}

func mkbin(t *testing.T, name, script string) {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/" + name
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
	t.Setenv("PATH", dir)
}

func TestStep02_CLI_Missing(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty: no CLIs at all
	a := &backlogAdapter{}
	if _, err := a.List(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	} else if _, ok := err.(*errCLIMissing); !ok {
		t.Fatalf("missing CLI must be errCLIMissing, got %T (%v)", err, err)
	}
	if code := statusForHTTP(&errCLIMissing{CLI: "backlog"}); code != 503 {
		t.Errorf("missing CLI must map to 503, got %d", code)
	}
}

func TestStep02_CLI_Exit1(t *testing.T) {
	mkbin(t, "backlog", `echo "boom: something broke" >&2; exit 1`)
	a := &backlogAdapter{}
	_, err := a.List(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ferr, ok := err.(*errCLIFailed)
	if !ok {
		t.Fatalf("exit-1 must be errCLIFailed, got %T (%v)", err, err)
	}
	if !strings.Contains(ferr.Error(), "boom: something broke") {
		t.Errorf("stderr excerpt must surface, got %q", ferr.Error())
	}
	if code := statusForHTTP(err); code != 502 {
		t.Errorf("CLI failure must map to 502, got %d", code)
	}
}

func TestStep02_CLI_Timeout(t *testing.T) {
	mkbin(t, "backlog", `/bin/sleep 11; echo '{}'`)
	t.Setenv("SKILLGRID_TRACKER_TIMEOUT_SECS", "")
	a := &backlogAdapter{}
	start := time.Now()
	_, err := a.List(context.Background())
	el := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	ferr, ok := err.(*errCLIFailed)
	if !ok || !ferr.Timeout {
		t.Fatalf("timeout must be errCLIFailed{Timeout}, got %T (%v)", err, err)
	}
	if el > 11*time.Second {
		t.Errorf("must time out before the 11s fixture finishes (took %v)", el)
	}
}

func TestStep02_BadOutput(t *testing.T) {
	mkbin(t, "backlog", `echo 'not json'`)
	a := &backlogAdapter{}
	_, err := a.List(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := err.(*errBadOutput); !ok {
		t.Fatalf("garbage stdout must be errBadOutput, got %T (%v)", err, err)
	}
	if code := statusForHTTP(err); code != 502 {
		t.Errorf("bad output must map to 502, got %d", code)
	}
}

const fixtureConfigText = `Configuration:
  projectName: skillgrid
  statuses: [needs-triage, ready-for-agent, done]
  labels: []
  priorities: [high, medium, low]
  types: [feature, bug, chore]
`

const fixtureTaskList = `{"schemaVersion": 1, "kind": "task-list", "tasks": [
  {"id": "TASK-001", "title": "First", "description": "Do the thing", "status": "needs-triage", "type": "feature", "priority": "high",
   "assignees": ["@ana"], "labels": ["ui"], "acceptanceCriteriaCompleted": 1, "acceptanceCriteriaCount": 4,
   "createdAt": "2026-09-03", "updatedAt": "2026-09-04", "isReady": true,
   "references": ["docs/a.md"], "documentation": ["docs/b.md"]},
  {"id": "TASK-002", "title": "Second", "status": "done", "type": "bug", "priority": "low",
   "assignees": [], "labels": [], "acceptanceCriteriaCompleted": 2, "acceptanceCriteriaCount": 2,
   "updatedAt": "2026-09-05", "isReady": true}
]}`

const fixtureTaskView = `{"schemaVersion": 1, "kind": "task-view", "task":
  {"id": "TASK-001", "title": "First", "status": "ready-for-agent", "type": "feature", "priority": "high",
   "assignees": [], "labels": [], "acceptanceCriteriaCompleted": 0, "acceptanceCriteriaCount": 1,
   "updatedAt": "2026-09-05", "isReady": false, "references": ["docs/skillgrid/changes/009-web-admin-dashboard/change.md"]}}`

// 02.6 [RED] backlog adapter: text config + list/view/status via real commands.
func TestStep02_Backlog(t *testing.T) {
	mkbin(t, "backlog", `case "$1 $2" in
  "config list") printf '%s' "$FIX_CONFIG" ;;
  "task list") printf '%s' "$FIX_LIST" ;;
  "task view") if [ "$3" = "TASK-001" ]; then printf '%s' "$FIX_VIEW"; else echo "Task $3 not found."; fi ;;
  "task edit") exit 0 ;;
  *) echo "unexpected: $*" >&2; exit 1 ;;
esac`)
	t.Setenv("FIX_CONFIG", fixtureConfigText)
	t.Setenv("FIX_LIST", fixtureTaskList)
	t.Setenv("FIX_VIEW", fixtureTaskView)
	a := &backlogAdapter{}

	cfg, err := a.Config(context.Background())
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if len(cfg.Statuses) != 3 || cfg.Statuses[0] != "needs-triage" {
		t.Errorf("statuses parsed wrong: %v", cfg.Statuses)
	}
	if len(cfg.Types) != 3 || len(cfg.Priorities) != 3 {
		t.Errorf("types/priorities parsed wrong: %v / %v", cfg.Types, cfg.Priorities)
	}

	items, err := a.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 2 || items[0].ID != "TASK-001" || items[0].ACCompleted != 1 || items[0].ACTotal != 4 {
		t.Errorf("list DTO wrong: %+v", items)
	}
	if items[0].Board != "todo" || items[0].Description != "Do the thing" || items[0].CreatedAt != "2026-09-03" {
		t.Errorf("list demo fields wrong: %+v", items[0])
	}
	if len(items[0].DocRefs) != 2 || items[0].DocRefs[0] != "docs/a.md" {
		t.Errorf("list doc_refs wrong: %+v", items[0].DocRefs)
	}
	if items[0].Provider != ProviderBacklogMD || items[0].IsReady == nil || !*items[0].IsReady {
		t.Errorf("list provider/isReady wrong: %+v", items[0])
	}

	got, err := a.Get(context.Background(), "TASK-001")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != "ready-for-agent" || !strings.Contains(got.StatusDetail, "009-web-admin-dashboard") {
		t.Errorf("get DTO wrong: %+v", got)
	}
	if got.Board != "in_progress" {
		t.Errorf("get board wrong: %+v", got)
	}
	if _, err := a.Get(context.Background(), "TASK-999"); err == nil {
		t.Fatal("unknown id must error")
	} else if _, ok := err.(*errNotFound); !ok {
		t.Fatalf("unknown id must be errNotFound, got %T (%v)", err, err)
	}

	moved, err := a.SetStatus(context.Background(), "TASK-001", "ready-for-agent")
	if err != nil {
		t.Fatalf("setstatus: %v", err)
	}
	if moved.Status != "ready-for-agent" {
		t.Errorf("status change not reflected: %+v", moved)
	}
	if _, err := a.SetStatus(context.Background(), "TASK-001", "not-a-real-status"); err == nil {
		t.Fatal("invalid status must error")
	} else if _, ok := err.(*errBadStatus); !ok {
		t.Fatalf("invalid status must be errBadStatus, got %T (%v)", err, err)
	}
}

const fixtureGHList = `[{"number": 12, "title": "Fix login", "state": "OPEN",
  "labels": [{"name": "bug"}], "assignees": [{"login": "ana"}], "updatedAt": "2026-09-01T00:00:00Z"},
  {"number": 13, "title": "Docs", "state": "CLOSED",
   "labels": [], "assignees": [], "updatedAt": "2026-09-02T00:00:00Z"}]`

const fixtureGHView = `{"number": 12, "title": "Fix login", "body": "details here", "state": "OPEN",
  "labels": [{"name": "bug"}], "assignees": [{"login": "ana"}], "comments": [],
  "updatedAt": "2026-09-01T00:00:00Z"}`

// 02.7 [RED] GitHub + GitLab adapters: JSON list/view + close/reopen.
func TestStep02_GitHubGitLab(t *testing.T) {
	mkbin(t, "gh", `case "$1 $2" in
  "issue list") printf '%s' "$FIX_GH_LIST" ;;
  "issue view") if [ "$3" = "12" ]; then printf '%s' "$FIX_GH_VIEW"; else echo "Could not resolve to an Issue" >&2; exit 1; fi ;;
  "issue close") exit 0 ;;
  "issue reopen") exit 0 ;;
  *) echo "unexpected: $*" >&2; exit 1 ;;
esac`)
	t.Setenv("FIX_GH_LIST", fixtureGHList)
	t.Setenv("FIX_GH_VIEW", fixtureGHView)
	g := &githubAdapter{}

	cfg, err := g.Config(context.Background())
	if err != nil {
		t.Fatalf("gh config: %v", err)
	}
	if len(cfg.Statuses) != 2 || cfg.Statuses[0] != "open" {
		t.Errorf("gh statuses wrong: %v", cfg.Statuses)
	}
	items, err := g.List(context.Background())
	if err != nil {
		t.Fatalf("gh list: %v", err)
	}
	if len(items) != 2 || items[0].ID != "12" || items[0].Status != "open" {
		t.Errorf("gh list DTO wrong: %+v", items)
	}
	if len(items[0].Assignees) != 1 || items[0].Assignees[0] != "ana" || len(items[0].Labels) != 1 {
		t.Errorf("gh labels/assignees wrong: %+v", items[0])
	}
	got, err := g.Get(context.Background(), "12")
	if err != nil {
		t.Fatalf("gh get: %v", err)
	}
	if got.Title != "Fix login" || got.Provider != ProviderGitHub {
		t.Errorf("gh get DTO wrong: %+v", got)
	}
	if _, err := g.Get(context.Background(), "999"); err == nil {
		t.Fatal("gh unknown id must error")
	} else if _, ok := err.(*errNotFound); !ok {
		t.Fatalf("gh unknown id must be errNotFound, got %T (%v)", err, err)
	}
	closed, err := g.SetStatus(context.Background(), "12", "closed")
	if err != nil {
		t.Fatalf("gh close: %v", err)
	}
	_ = closed
	if _, err := g.SetStatus(context.Background(), "12", "needs-triage"); err == nil {
		t.Fatal("gh custom status must error")
	} else if _, ok := err.(*errUnsupported); !ok {
		t.Fatalf("gh custom status must be errUnsupported, got %T (%v)", err, err)
	}

	mkbin(t, "glab", `case "$1 $2" in
  "issue list") printf '%s' "$FIX_GLAB_LIST" ;;
  "issue view") if [ "$3" = "5" ]; then printf '%s' "$FIX_GLAB_VIEW"; else echo "404 Not Found" >&2; exit 1; fi ;;
  "issue close") exit 0 ;;
  "issue reopen") exit 0 ;;
  *) echo "unexpected: $*" >&2; exit 1 ;;
esac`)
	t.Setenv("FIX_GLAB_LIST", `[{"iid": 5, "title": "Pipeline fix", "state": "opened",
  "labels": ["ci"], "updated_at": "2026-09-03T00:00:00Z"}]`)
	t.Setenv("FIX_GLAB_VIEW", `{"iid": 5, "title": "Pipeline fix", "description": "d",
  "state": "opened", "labels": ["ci"], "updated_at": "2026-09-03T00:00:00Z"}`)
	gl := &gitlabAdapter{}
	litems, err := gl.List(context.Background())
	if err != nil {
		t.Fatalf("glab list: %v", err)
	}
	if len(litems) != 1 || litems[0].ID != "5" || litems[0].Status != "opened" {
		t.Errorf("glab list DTO wrong: %+v", litems)
	}
	if _, err := gl.SetStatus(context.Background(), "5", "closed"); err != nil {
		t.Fatalf("glab close: %v", err)
	}
	if _, err := gl.SetStatus(context.Background(), "5", "needs-triage"); err == nil {
		t.Fatal("glab custom status must error")
	} else if _, ok := err.(*errUnsupported); !ok {
		t.Fatalf("glab custom status must be errUnsupported, got %T (%v)", err, err)
	}
}

const fixtureJiraList = `KEY        TYPE  SUMMARY      STATUS       PRIORITY  UPDATED
PROJ-1     Task  Fix login    In Progress  High      2026-09-01
PROJ-2     Bug   Crash on x   Done         Medium    2026-09-02
`

const fixtureJiraView = `Summary: Fix login
Status: In Progress
Assignee: ana
Priority: High
Updated: 2026-09-01
`

// 02.8 [RED] Jira adapter: JQL list/view/move with project key from tracker doc.
func TestStep02_Jira(t *testing.T) {
	mkbin(t, "jira", `case "$1 $2" in
  "issue list") printf '%s' "$FIX_JIRA_LIST" ;;
  "issue view") if [ "$3" = "PROJ-1" ]; then printf '%s' "$FIX_JIRA_VIEW"; else echo "does not exist" >&2; exit 1; fi ;;
  "issue move") if [ "$4" = "Done" ]; then exit 0; else echo "invalid transition" >&2; exit 1; fi ;;
  *) echo "unexpected: $*" >&2; exit 1 ;;
esac`)
	t.Setenv("FIX_JIRA_LIST", fixtureJiraList)
	t.Setenv("FIX_JIRA_VIEW", fixtureJiraView)
	j := &jiraAdapter{projectKey: func() (string, error) { return "PROJ", nil }}

	cfg, err := j.Config(context.Background())
	if err != nil {
		t.Fatalf("jira config: %v", err)
	}
	if cfg.ProjectKey != "PROJ" {
		t.Errorf("jira project key wrong: %+v", cfg)
	}
	items, err := j.List(context.Background())
	if err != nil {
		t.Fatalf("jira list: %v", err)
	}
	if len(items) != 2 || items[0].ID != "PROJ-1" || items[0].Status != "In Progress" {
		t.Errorf("jira list DTO wrong: %+v", items)
	}
	got, err := j.Get(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("jira get: %v", err)
	}
	if got.Title != "Fix login" || len(got.Assignees) != 1 {
		t.Errorf("jira get DTO wrong: %+v", got)
	}
	if _, err := j.Get(context.Background(), "PROJ-9"); err == nil {
		t.Fatal("jira unknown key must error")
	} else if _, ok := err.(*errNotFound); !ok {
		t.Fatalf("jira unknown key must be errNotFound, got %T (%v)", err, err)
	}
	if _, err := j.SetStatus(context.Background(), "PROJ-1", "Done"); err != nil {
		t.Fatalf("jira move: %v", err)
	}
	// Missing project key is never guessed.
	j2 := &jiraAdapter{projectKey: func() (string, error) {
		return "", &errUnresolvable{Reason: "no project key"}
	}}
	if _, err := j2.List(context.Background()); err == nil {
		t.Fatal("missing project key must error")
	} else if _, ok := err.(*errUnresolvable); !ok {
		t.Fatalf("missing key must be errUnresolvable, got %T (%v)", err, err)
	}
}

// Non-zero exit carrying "not found" text must map to 404-class (live bug 2026-09-11).
func TestStep02_NotFoundNonZeroExit(t *testing.T) {
	mkbin(t, "backlog", `echo "Task TASK-999 not found." >&2; exit 1`)
	a := &backlogAdapter{}
	if _, err := a.Get(context.Background(), "TASK-999"); err == nil {
		t.Fatal("expected error, got nil")
	} else if _, ok := err.(*errNotFound); !ok {
		t.Fatalf("must be errNotFound, got %T (%v)", err, err)
	}
	if code := statusForHTTP(&errNotFound{ID: "x"}); code != 404 {
		t.Errorf("not-found must map to 404, got %d", code)
	}
}

// Canonical board columns (demo STATUS_COLUMNS): every provider maps to one.
func TestStep02_BoardMapping(t *testing.T) {
	cases := []struct{ provider, status, want string }{
		{ProviderBacklogMD, "needs-triage", "todo"},
		{ProviderBacklogMD, "needs-info", "todo"},
		{ProviderBacklogMD, "ready-for-agent", "in_progress"},
		{ProviderBacklogMD, "ready-for-human", "in_progress"},
		{ProviderBacklogMD, "in-progress", "in_progress"},
		{ProviderBacklogMD, "blocked", "blocked"},
		{ProviderBacklogMD, "done", "done"},
		{ProviderBacklogMD, "wontfix", "done"},
		{ProviderBacklogMD, "something-new", "todo"},
		{ProviderGitHub, "OPEN", "todo"},
		{ProviderGitHub, "CLOSED", "done"},
		{ProviderGitLab, "opened", "todo"},
		{ProviderGitLab, "closed", "done"},
		{ProviderJira, "In Progress", "in_progress"},
		{ProviderJira, "In Review", "in_progress"},
		{ProviderJira, "Done", "done"},
		{ProviderJira, "Closed", "done"},
		{ProviderJira, "Blocked", "blocked"},
		{ProviderJira, "To Do", "todo"},
		{ProviderJira, "Backlog", "todo"},
	}
	for _, c := range cases {
		if got := boardFor(c.provider, c.status); got != c.want {
			t.Errorf("boardFor(%s, %q) = %q, want %q", c.provider, c.status, got, c.want)
		}
	}
}
