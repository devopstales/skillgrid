package handoff

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newRepo creates a temp git repo with the given commits (oldest first).
// Each entry is "subject" or "subject|Task: X|Decisions: Y".
func newRepo(t *testing.T, commits []string) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"HOME="+dir,
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("git", "init", "-q", "-b", "main")
	run("git", "config", "user.name", "t")
	run("git", "config", "user.email", "t@t")
	for i, c := range commits {
		fields := strings.Split(c, "|")
		body := fields[0]
		if len(fields) > 1 {
			body += "\n\n[skillgrid-context]\n"
			for _, f := range fields[1:] {
				body += f + "\n"
			}
			body += "[/skillgrid-context]\n"
		}
		p := filepath.Join(dir, "f"+string(rune('a'+i))+".txt")
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		run("git", "add", ".")
		cmd := exec.Command("git", "commit", "-q", "-m", body)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"HOME="+dir,
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("commit %d: %v\n%s", i, err, out)
		}
	}
	return dir
}

// TestParseContext verifies the [skillgrid-context] parser against the
// checkpoint-state.sh format (all fields, missing fields, absent block).
func TestParseContext(t *testing.T) {
	c := parseContext("feat: x\n\n[skillgrid-context]\nTask: do it\nDecisions: a,b\nRemaining: rest\nTried: nothing\n[/skillgrid-context]\n")
	if c.Task != "do it" || c.Decisions != "a,b" || c.Remaining != "rest" || c.Tried != "nothing" {
		t.Fatalf("full parse = %+v", c)
	}
	// Absent block.
	if got := parseContext("feat: x\n\nno block here"); got != (Context{}) {
		t.Fatalf("absent block = %+v", got)
	}
	// Partial block (no closing marker still parses).
	c2 := parseContext("[skillgrid-context]\nTask: only\n")
	if c2.Task != "only" || c2.Decisions != "" {
		t.Fatalf("partial parse = %+v", c2)
	}
	// Case-insensitive markers.
	c3 := parseContext("[SkillGrid-Context]\ntask: lower\n[/skillgrid-context]")
	if c3.Task != "lower" {
		t.Fatalf("case parse = %+v", c3)
	}
}

// openHub opens a store-backed hub for the given repo dir.
func openHub(t *testing.T, repoDir string) *Hub {
	t.Helper()
	st, err := storeOpen(t)
	if err != nil {
		t.Fatalf("store open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return &Hub{DB: st.DB, Project: "hubtest", RepoDir: repoDir}
}

// TestRecordUpsert proves Record is idempotent: the same HEAD produces one
// row, and the [skillgrid-context] block + changed files are captured.
func TestRecordUpsert(t *testing.T) {
	repo := newRepo(t, []string{
		"feat: first",
		"feat: second|Task: implement|Decisions: chose A|Remaining: tests|Tried: none",
	})
	h := openHub(t, repo)
	ctx := context.Background()

	s, err := h.Record(ctx)
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if s.Subject != "feat: second" {
		t.Fatalf("subject = %q", s.Subject)
	}
	var c Context
	if err := json.Unmarshal([]byte(s.ContextJSON), &c); err != nil || c.Task != "implement" {
		t.Fatalf("context json = %q (%v)", s.ContextJSON, err)
	}
	var files []ChangedFile
	if err := json.Unmarshal([]byte(s.ChangedFiles), &files); err != nil || len(files) != 1 || files[0].Path != "fb.txt" {
		t.Fatalf("changed files = %q (%v)", s.ChangedFiles, err)
	}

	// Second record of the same HEAD: still one row.
	if _, err := h.Record(ctx); err != nil {
		t.Fatalf("re-record: %v", err)
	}
	n, err := h.CountSnapshots(ctx)
	if err != nil || n != 1 {
		t.Fatalf("count after re-record = %d (%v)", n, err)
	}
}

// TestBackfillSeedsAllCommits proves backfill walks git log (newest first) and
// is idempotent (re-running does not duplicate).
func TestBackfillSeedsAllCommits(t *testing.T) {
	repo := newRepo(t, []string{
		"feat: one",
		"feat: two|Task: t2",
		"chore: three",
	})
	h := openHub(t, repo)
	ctx := context.Background()

	n, err := h.Backfill(ctx, 0)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if n != 3 {
		t.Fatalf("backfilled %d commits, want 3", n)
	}
	snapshots, err := h.ListSnapshots(ctx, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(snapshots) != 3 {
		t.Fatalf("listed %d snapshots, want 3", len(snapshots))
	}
	// Newest first (created_at ordinal makes this deterministic).
	if snapshots[0].Subject != "chore: three" {
		t.Fatalf("newest = %q", snapshots[0].Subject)
	}
	// The context commit's block was parsed during backfill.
	var mid *Snapshot
	for i := range snapshots {
		if snapshots[i].Subject == "feat: two" {
			mid = &snapshots[i]
		}
	}
	if mid == nil || mid.ContextJSON == "" {
		t.Fatalf("context commit missing/empty context: %+v", mid)
	}
	var c Context
	if err := json.Unmarshal([]byte(mid.ContextJSON), &c); err != nil || c.Task != "t2" {
		t.Fatalf("backfilled context = %q (%v)", mid.ContextJSON, err)
	}

	// Idempotent re-run.
	if _, err := h.Backfill(ctx, 0); err != nil {
		t.Fatalf("re-backfill: %v", err)
	}
	if n, _ := h.CountSnapshots(ctx); n != 3 {
		t.Fatalf("count after re-backfill = %d, want 3", n)
	}
}

// TestBackfillHonorsLimit proves the limit clamps the walk.
func TestBackfillHonorsLimit(t *testing.T) {
	repo := newRepo(t, []string{"a", "b", "c", "d"})
	h := openHub(t, repo)
	if n, err := h.Backfill(context.Background(), 2); err != nil || n != 2 {
		t.Fatalf("backfill limit=2 -> n=%d err=%v", n, err)
	}
	if n, _ := h.CountSnapshots(context.Background()); n != 2 {
		t.Fatalf("stored %d, want 2", n)
	}
}

// TestRecordCheckpointAndVerify proves the marker captures state, verify with
// no drift recommends continue (status verified), and a new commit flips it
// to refresh (status stale).
func TestRecordCheckpointAndVerify(t *testing.T) {
	repo := newRepo(t, []string{"feat: base"})
	h := openHub(t, repo)
	ctx := context.Background()

	cp, err := h.RecordCheckpoint(ctx, CheckpointInput{Name: "before-apply-test", Evidence: "lint ok"})
	if err != nil {
		t.Fatalf("record checkpoint: %v", err)
	}
	if cp.Status != "open" || cp.Branch != "main" || cp.Commit == "" {
		t.Fatalf("checkpoint = %+v", cp)
	}

	rep, err := h.VerifyCheckpoint(ctx, "before-apply-test")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if rep.Recommendation != "continue" || !rep.CommitMatch || !rep.BranchMatch {
		t.Fatalf("clean verify = %+v", rep)
	}
	cp, _ = h.getCheckpoint(ctx, "before-apply-test")
	if cp.Status != "verified" {
		t.Fatalf("status after clean verify = %q", cp.Status)
	}

	// New commit -> drift.
	runGit(t, repo, "commit", "--allow-empty", "-q", "-m", "feat: after")
	rep, err = h.VerifyCheckpoint(ctx, "before-apply-test")
	if err != nil {
		t.Fatalf("verify after drift: %v", err)
	}
	if rep.Recommendation != "refresh" || rep.CommitMatch {
		t.Fatalf("drift verify = %+v", rep)
	}
	cp, _ = h.getCheckpoint(ctx, "before-apply-test")
	if cp.Status != "stale" {
		t.Fatalf("status after drift = %q", cp.Status)
	}
}

// TestRecordCheckpointDirtyState proves a dirty working tree is recorded.
func TestRecordCheckpointDirtyState(t *testing.T) {
	repo := newRepo(t, []string{"feat: base"})
	if err := os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := openHub(t, repo)
	cp, err := h.RecordCheckpoint(context.Background(), CheckpointInput{Name: "dirty-check"})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if !cp.Dirty {
		t.Fatalf("expected dirty=true, got %+v", cp)
	}
	// Verify with the same commit but dirty now -> inspect-drift.
	rep, err := h.VerifyCheckpoint(context.Background(), "dirty-check")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if rep.Recommendation != "inspect-drift" || !rep.DirtyNow {
		t.Fatalf("dirty verify = %+v", rep)
	}
}

// TestRecordHandoffRefProves upsert semantics + rollback delete.
func TestRecordHandoffRef(t *testing.T) {
	h := openHub(t, newRepo(t, []string{"feat: base"}))
	ctx := context.Background()

	in := HandoffRefInput{HandoffID: "h1", HandoffType: "session", FromCommit: "a", ToCommit: "b", SpecDir: ".skillgrid/sdd/x"}
	if err := h.RecordHandoffRef(ctx, in); err != nil {
		t.Fatalf("record ref: %v", err)
	}
	// Idempotent upsert (second write allowed, no duplicate).
	if err := h.RecordHandoffRef(ctx, in); err != nil {
		t.Fatalf("re-record ref: %v", err)
	}
	refs, err := h.ListHandoffRefs(ctx, 0)
	if err != nil || len(refs) != 1 {
		t.Fatalf("refs = %+v (%v)", refs, err)
	}
	if refs[0].FromCommit != "a" || refs[0].ToCommit != "b" || refs[0].SpecDir != ".skillgrid/sdd/x" {
		t.Fatalf("ref fields = %+v", refs[0])
	}
	r, ok, err := h.GetHandoffRef(ctx, "h1", "session")
	if err != nil || !ok || r.HandoffID != "h1" {
		t.Fatalf("get ref: %v ok=%v err=%v", r, ok, err)
	}
	// Rollback path.
	if err := h.DeleteHandoffRef(ctx, "h1", "session"); err != nil {
		t.Fatalf("delete ref: %v", err)
	}
	if _, ok, _ := h.GetHandoffRef(ctx, "h1", "session"); ok {
		t.Fatalf("ref still present after delete")
	}
}

// TestRecordFailsCleanWithoutGit proves Record errors (not panics) when HEAD
// cannot be resolved (empty dir, not a repo).
func TestRecordFailsCleanWithoutGit(t *testing.T) {
	h := openHub(t, t.TempDir())
	if _, err := h.Record(context.Background()); err == nil {
		t.Fatalf("expected error without a repo")
	}
}
