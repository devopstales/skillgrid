package tracker

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ─────────────────────────── 2.1 detection ───────────────────────────

// 2.1 [RED] Threat: Taxonomy — provider detection never guesses.
func TestPhase2_Detection(t *testing.T) {
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
	} else if code := statusForHTTP(err); code != 501 {
		t.Errorf("unknown tracker must map to 501, got %d", code)
	}
	// Tracker doc wins over config.
	doc := "# Issue tracker: GitHub\n\nSome body."
	if got, _ := ResolveProvider("", doc, "backlogmd"); got != ProviderGitHub {
		t.Errorf("tracker doc: got %q; want github", got)
	}
	// Config value used when doc is absent.
	if got, _ := ResolveProvider("", "", "gitlab"); got != ProviderGitLab {
		t.Errorf("config value: got %q; want gitlab", got)
	}
	// Total silence → default backlogmd.
	if got, _ := ResolveProvider("", "", ""); got != ProviderBacklogMD {
		t.Errorf("default: got %q; want backlogmd", got)
	}
	// Jira doc resolves; the project key is enforced by the adapter (501 if missing).
	if got, _ := ResolveProvider("", "# Issue tracker: Jira\n", ""); got != ProviderJira {
		t.Errorf("jira doc: got %q; want jira", got)
	}
	// Jira without a project key → errUnresolvable (501), never a guess.
	t.Setenv("SKILLGRID_JIRA_PROJECT", "")
	t.Chdir(t.TempDir()) // no tracker doc
	ja := &jiraAdapter{}
	if _, err := ja.List(context.Background()); err == nil {
		t.Error("jira without project key must error")
	} else if code := statusForHTTP(err); code != 501 {
		t.Errorf("jira missing key must map to 501, got %d", code)
	}
}

// mkbin writes a fake CLI onto an isolated PATH and runs it once with the
// `__warm__` argument. macOS assesses a freshly written executable on its
// first launch (~0.5s idle, well past the 10s CLI timeout while the full
// suite builds dozens of new test binaries); the warm-up run takes that cost
// outside the timed call. Scripts exit 0 on `__warm__` before their body.
func mkbin(t *testing.T, name, script string) {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/" + name
	body := "#!/bin/sh\n[ \"$1\" = __warm__ ] && exit 0\n" + script
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
	_ = exec.Command(path, "__warm__").Run()
	t.Setenv("PATH", dir)
}

// ─────────────────────── 2.2/2.3 CLI error semantics ───────────────────────

// 2.2 [RED] Threat: Subprocess — CLI-backed provider missing → 503.
func TestPhase2_CLI_Missing(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty: no CLIs at all
	g := &githubAdapter{}
	if _, err := g.List(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	} else if _, ok := err.(*errCLIMissing); !ok {
		t.Fatalf("missing CLI must be errCLIMissing, got %T (%v)", err, err)
	}
	if code := statusForHTTP(&errCLIMissing{CLI: "gh"}); code != 503 {
		t.Errorf("missing CLI must map to 503, got %d", code)
	}
	// Backlog.md (file-based) is unaffected by a missing PATH.
	a := &backlogAdapter{tasksDir: t.TempDir()}
	if _, err := a.List(context.Background()); err != nil {
		t.Errorf("file-based backlog must work without CLI, got %v", err)
	}
}

// 2.3 [RED] Threat: Subprocess — CLI non-zero exit / timeout / bad output → 502.
func TestPhase2_CLI_Failure(t *testing.T) {
	// exit-1 with stderr.
	mkbin(t, "gh", `echo "boom: something broke" >&2; exit 1`)
	g := &githubAdapter{}
	_, err := g.List(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if ferr, ok := err.(*errCLIFailed); !ok || !strings.Contains(ferr.Error(), "boom: something broke") {
		t.Fatalf("exit-1 must be errCLIFailed with stderr, got %T (%v)", err, err)
	}
	if code := statusForHTTP(err); code != 502 {
		t.Errorf("CLI failure must map to 502, got %d", code)
	}
}

func TestPhase2_CLI_Timeout(t *testing.T) {
	mkbin(t, "gh", `/bin/sleep 11; echo '{}'`)
	g := &githubAdapter{}
	start := time.Now()
	_, err := g.List(context.Background())
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	ferr, ok := err.(*errCLIFailed)
	if !ok || !ferr.Timeout {
		t.Fatalf("timeout must be errCLIFailed{Timeout}, got %T (%v)", err, err)
	}
	if time.Since(start) > 11*time.Second {
		t.Errorf("must time out before the 11s fixture finishes")
	}
	if code := statusForHTTP(err); code != 502 {
		t.Errorf("timeout must map to 502, got %d", code)
	}
}

func TestPhase2_BadOutput(t *testing.T) {
	mkbin(t, "gh", `echo 'not json'`)
	g := &githubAdapter{}
	_, err := g.List(context.Background())
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

// ─────────────────────── 2.4 Backlog.md file adapter ───────────────────────

const taskA = `---
id: TASK-001
title: 'First task'
status: needs-triage
priority: high
type: feature
assignee:
  - "@ana"
labels:
  - ui
  - p1
dependencies: []
milestone: m1
---
## Description

Do the thing.
`

const taskB = `---
id: TASK-002
title: Second task
status: done
priority: low
type: bug
assignee: []
labels: []
dependencies:
  - TASK-001
---
Second body.
`

const taskC = `---
id: TASK-003
title: Third task
status: in-progress
priority: medium
type: chore
assignee: []
labels: [docs]
dependencies: []
parent: TASK-001
---
Third body.
`

func writeTasks(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

// 2.4 [RED] Backlog.md adapter: parse frontmatter → UnifiedTask (no CLI).
func TestPhase2_Backlog(t *testing.T) {
	dir := t.TempDir()
	writeTasks(t, dir, map[string]string{
		"task-001.md": taskA,
		"task-002.md": taskB,
		"task-003.md": taskC,
	})
	a := &backlogAdapter{tasksDir: dir}

	items, err := a.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(items))
	}
	byID := map[string]UnifiedTask{}
	for _, it := range items {
		byID[it.ID] = it
	}
	a1 := byID["TASK-001"]
	if a1.Title != "First task" || a1.Status != "needs-triage" || a1.Board != "todo" {
		t.Errorf("TASK-001 wrong: %+v", a1)
	}
	if a1.Priority != "high" || a1.Type != "feature" || a1.Milestone != "m1" {
		t.Errorf("TASK-001 meta wrong: %+v", a1)
	}
	if len(a1.Assignees) != 1 || a1.Assignees[0] != "@ana" {
		t.Errorf("TASK-001 assignee wrong: %+v", a1)
	}
	if len(a1.Labels) != 2 || a1.Labels[0] != "ui" {
		t.Errorf("TASK-001 labels wrong: %+v", a1)
	}
	b1 := byID["TASK-002"]
	if len(b1.Dependencies) != 1 || b1.Dependencies[0] != "TASK-001" {
		t.Errorf("TASK-002 deps wrong: %+v", b1)
	}
	if b1.Board != "done" {
		t.Errorf("TASK-002 board wrong: %+v", b1)
	}
	c1 := byID["TASK-003"]
	if c1.Parent != "TASK-001" || c1.Board != "in_progress" {
		t.Errorf("TASK-003 parent/board wrong: %+v", c1)
	}
	// provider always "backlogmd"
	if a1.Provider != ProviderBacklogMD {
		t.Errorf("provider wrong: %q", a1.Provider)
	}
	// No CLI was required: this ran with the real PATH but a temp tasksDir.
}

// 2.4 Get: one item; unknown id → 404-class.
func TestPhase2_Backlog_Get(t *testing.T) {
	dir := t.TempDir()
	writeTasks(t, dir, map[string]string{"task-001.md": taskA})
	a := &backlogAdapter{tasksDir: dir}
	got, err := a.Get(context.Background(), "TASK-001")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != "TASK-001" || got.Status != "needs-triage" {
		t.Errorf("get DTO wrong: %+v", got)
	}
	if _, err := a.Get(context.Background(), "TASK-999"); err == nil {
		t.Fatal("unknown id must error")
	} else if code := statusForHTTP(err); code != 404 {
		t.Errorf("unknown id must map to 404, got %d", code)
	}
}

// 2.4 SetStatus: invalid status → 400-class; valid status shells out to task edit.
func TestPhase2_Backlog_SetStatus(t *testing.T) {
	dir := t.TempDir()
	writeTasks(t, dir, map[string]string{"task-001.md": taskA})
	mkbin(t, "backlog", `case "$1 $2" in
  "task edit") exit 0 ;;
  *) echo "unexpected: $*" >&2; exit 1 ;;
esac`)
	a := &backlogAdapter{tasksDir: dir}
	if _, err := a.SetStatus(context.Background(), "TASK-001", "ready-for-agent"); err != nil {
		t.Fatalf("valid status: %v", err)
	}
	if _, err := a.SetStatus(context.Background(), "TASK-001", "not-a-real-status"); err == nil {
		t.Fatal("invalid status must error")
	} else if code := statusForHTTP(err); code != 400 {
		t.Errorf("invalid status must map to 400, got %d", code)
	}
}

// 2.4 Dependencies: TASK-002 depends on TASK-001, so TASK-001 has DepsIn=[TASK-002].
func TestPhase2_Backlog_Deps(t *testing.T) {
	dir := t.TempDir()
	writeTasks(t, dir, map[string]string{"task-001.md": taskA, "task-002.md": taskB})
	a := &backlogAdapter{tasksDir: dir}
	d, err := a.Dependencies(context.Background(), "TASK-001")
	if err != nil {
		t.Fatalf("deps: %v", err)
	}
	if len(d.DepsIn) != 1 || d.DepsIn[0] != "TASK-002" {
		t.Errorf("TASK-001 DepsIn wrong: %+v", d)
	}
	d2, err := a.Dependencies(context.Background(), "TASK-002")
	if err != nil {
		t.Fatalf("deps2: %v", err)
	}
	if len(d2.DepsOut) != 1 || d2.DepsOut[0] != "TASK-001" {
		t.Errorf("TASK-002 DepsOut wrong: %+v", d2)
	}
}

// ─────────────────────── 2.5 remote adapters ───────────────────────

const fixtureGHList = `[{"number": 12, "title": "Fix login", "state": "OPEN",
  "labels": [{"name": "bug"}], "assignees": [{"login": "ana"}], "updatedAt": "2026-09-01T00:00:00Z"},
  {"number": 13, "title": "Docs", "state": "CLOSED",
   "labels": [], "assignees": [], "updatedAt": "2026-09-02T00:00:00Z"}]`

const fixtureGHView = `{"number": 12, "title": "Fix login", "body": "details here", "state": "OPEN",
  "labels": [{"name": "bug"}], "assignees": [{"login": "ana"}], "comments": [],
  "updatedAt": "2026-09-01T00:00:00Z"}`

// 2.5 [RED] GitHub + GitLab adapters: JSON list/get + close/reopen.
func TestPhase2_RemoteAdapters(t *testing.T) {
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

	items, err := g.List(context.Background())
	if err != nil {
		t.Fatalf("gh list: %v", err)
	}
	if len(items) != 2 || items[0].ID != "12" || items[0].Status != "open" {
		t.Errorf("gh list DTO wrong: %+v", items)
	}
	if len(items[0].Assignees) != 1 || items[0].Assignees[0] != "ana" {
		t.Errorf("gh assignees wrong: %+v", items[0])
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
	} else if code := statusForHTTP(err); code != 404 {
		t.Errorf("gh unknown id must map to 404, got %d", code)
	}
	// done → close
	if _, err := g.SetStatus(context.Background(), "12", "done"); err != nil {
		t.Fatalf("gh close: %v", err)
	}
	// custom status → 501
	if _, err := g.SetStatus(context.Background(), "12", "needs-triage"); err == nil {
		t.Fatal("gh custom status must error")
	} else if code := statusForHTTP(err); code != 501 {
		t.Errorf("gh custom status must map to 501, got %d", code)
	}

	// GitLab.
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
	if _, err := gl.SetStatus(context.Background(), "5", "done"); err != nil {
		t.Fatalf("glab close: %v", err)
	}
	if _, err := gl.SetStatus(context.Background(), "5", "needs-triage"); err == nil {
		t.Fatal("glab custom status must error")
	} else if code := statusForHTTP(err); code != 501 {
		t.Errorf("glab custom status must map to 501, got %d", code)
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

// 2.5 [RED] Jira adapter: JQL list/get/move; project key never guessed.
func TestPhase2_Jira(t *testing.T) {
	mkbin(t, "jira", `case "$1 $2" in
  "issue list") printf '%s' "$FIX_JIRA_LIST" ;;
  "issue view") if [ "$3" = "PROJ-1" ]; then printf '%s' "$FIX_JIRA_VIEW"; else echo "does not exist" >&2; exit 1; fi ;;
  "issue move") if [ "$4" = "Done" ]; then exit 0; else echo "invalid transition" >&2; exit 1; fi ;;
  *) echo "unexpected: $*" >&2; exit 1 ;;
esac`)
	t.Setenv("FIX_JIRA_LIST", fixtureJiraList)
	t.Setenv("FIX_JIRA_VIEW", fixtureJiraView)
	j := &jiraAdapter{projectKey: func() (string, error) { return "PROJ", nil }}

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
	} else if code := statusForHTTP(err); code != 404 {
		t.Errorf("jira unknown key must map to 404, got %d", code)
	}
	if _, err := j.SetStatus(context.Background(), "PROJ-1", "Done"); err != nil {
		t.Fatalf("jira move: %v", err)
	}
	// Missing project key is never guessed → 501.
	j2 := &jiraAdapter{projectKey: func() (string, error) {
		return "", &errUnresolvable{Reason: "no project key"}
	}}
	if _, err := j2.List(context.Background()); err == nil {
		t.Fatal("missing project key must error")
	} else if code := statusForHTTP(err); code != 501 {
		t.Errorf("missing key must map to 501, got %d", code)
	}
}

// Non-zero exit carrying "not found" text maps to 404-class.
func TestPhase2_NotFoundNonZeroExit(t *testing.T) {
	mkbin(t, "gh", `echo "Could not resolve to an Issue" >&2; exit 1`)
	g := &githubAdapter{}
	if _, err := g.Get(context.Background(), "999"); err == nil {
		t.Fatal("expected error, got nil")
	} else if code := statusForHTTP(err); code != 404 {
		t.Errorf("not-found must map to 404, got %d", code)
	}
}

// Canonical board columns: every provider maps to one.
func TestPhase2_BoardMapping(t *testing.T) {
	cases := []struct{ provider, status, want string }{
		{ProviderBacklogMD, "needs-triage", "todo"},
		{ProviderBacklogMD, "ready-for-agent", "ready"},
		{ProviderBacklogMD, "ready-for-human", "ready"},
		{ProviderBacklogMD, "in-progress", "in_progress"},
		{ProviderBacklogMD, "blocked", "blocked"},
		{ProviderBacklogMD, "done", "done"},
		{ProviderBacklogMD, "wontfix", "done"},
		{ProviderBacklogMD, "ready", "todo"}, // literal "ready" is not a status → default
		{ProviderBacklogMD, "needs-info", "todo"},
		{ProviderGitHub, "OPEN", "todo"},
		{ProviderGitHub, "CLOSED", "done"},
		{ProviderGitLab, "opened", "todo"},
		{ProviderGitLab, "closed", "done"},
		{ProviderJira, "In Progress", "in_progress"},
		{ProviderJira, "Done", "done"},
		{ProviderJira, "Blocked", "blocked"},
		{ProviderJira, "To Do", "todo"},
	}
	for _, c := range cases {
		if got := boardFor(c.provider, c.status); got != c.want {
			t.Errorf("boardFor(%s, %q) = %q, want %q", c.provider, c.status, got, c.want)
		}
	}
}

// 2.6 [RED] Milestones: providers without milestone support return an empty
// (non-nil) slice and no error. A regression to nil or an error would surface
// here.
func TestPhase2_MilestonesEmptyForRemoteProviders(t *testing.T) {
	ctx := context.Background()
	for name, p := range map[string]TicketProvider{
		"github": &githubAdapter{},
		"gitlab": &gitlabAdapter{},
		"jira":   &jiraAdapter{},
	} {
		ms, err := p.Milestones(ctx)
		if err != nil {
			t.Errorf("%s Milestones: unexpected error %v", name, err)
		}
		if ms == nil {
			t.Errorf("%s Milestones: must return non-nil empty slice, got nil", name)
		}
		if len(ms) != 0 {
			t.Errorf("%s Milestones: want 0 milestones, got %d", name, len(ms))
		}
	}
}
