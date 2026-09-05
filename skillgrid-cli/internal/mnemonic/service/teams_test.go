package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/files"
)

func newTeamsTestService(t *testing.T) (*Service, string) {
	t.Helper()
	dataDir := t.TempDir()
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, ".skillgrid"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return New(dataDir), workspace
}

func TestSpawnTaskReturnsPendingID(t *testing.T) {
	svc, ws := newTeamsTestService(t)
	id, err := svc.SpawnTask(context.Background(), SpawnTaskParams{
		Directory: ws,
		Title:     "implement feature",
		Brief:     "# Do the thing\n",
		Priority:  10,
		CreatedBy: "lead",
	})
	if err != nil {
		t.Fatalf("SpawnTask: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty task id")
	}

	view, brief, err := svc.ReadTask(context.Background(), ws, id)
	if err != nil {
		t.Fatalf("ReadTask: %v", err)
	}
	if view.Status != taskStatusPending {
		t.Errorf("status = %q, want pending", view.Status)
	}
	if brief != "# Do the thing\n" {
		t.Errorf("brief = %q", brief)
	}
}

func TestPullClaimsTopPriorityWithBrief(t *testing.T) {
	svc, ws := newTeamsTestService(t)
	low, err := svc.SpawnTask(context.Background(), SpawnTaskParams{
		Directory: ws, Title: "low", Brief: "low brief", Priority: 1,
	})
	if err != nil {
		t.Fatalf("spawn low: %v", err)
	}
	high, err := svc.SpawnTask(context.Background(), SpawnTaskParams{
		Directory: ws, Title: "high", Brief: "high brief", Priority: 50,
	})
	if err != nil {
		t.Fatalf("spawn high: %v", err)
	}
	_ = low

	claimed, err := svc.PullNextTask(context.Background(), ws, "agent-1")
	if err != nil {
		t.Fatalf("PullNextTask: %v", err)
	}
	if claimed.ID != high {
		t.Errorf("claimed %q, want high-priority %q", claimed.ID, high)
	}
	if claimed.Status != taskStatusInProgress {
		t.Errorf("status = %q", claimed.Status)
	}
	_, brief, err := svc.ReadTask(context.Background(), ws, claimed.ID)
	if err != nil {
		t.Fatalf("ReadTask: %v", err)
	}
	if brief != "high brief" {
		t.Errorf("brief = %q", brief)
	}
}

func TestSubmitOutputReviewMarkDoneAdvanceStatus(t *testing.T) {
	svc, ws := newTeamsTestService(t)
	id, err := svc.SpawnTask(context.Background(), SpawnTaskParams{
		Directory: ws, Title: "lifecycle", Brief: "brief", Priority: 1,
	})
	if err != nil {
		t.Fatalf("SpawnTask: %v", err)
	}
	if _, err := svc.PullNextTask(context.Background(), ws, "dev-1"); err != nil {
		t.Fatalf("PullNextTask: %v", err)
	}
	if err := svc.SubmitOutput(context.Background(), ws, id, "done summary", "# output\n"); err != nil {
		t.Fatalf("SubmitOutput: %v", err)
	}
	view, _, err := svc.ReadTask(context.Background(), ws, id)
	if err != nil {
		t.Fatalf("ReadTask: %v", err)
	}
	if view.Status != taskStatusReviewSpec {
		t.Errorf("after output status = %q, want review_spec", view.Status)
	}
	if view.OutputPath == "" {
		t.Error("expected output_path set")
	}

	if err := svc.SubmitReview(context.Background(), SubmitReviewParams{
		Directory: ws, TaskID: id, ReviewerID: "rev-1",
		ReviewType: "spec_compliance", Passed: true, Comments: "lgtm",
	}); err != nil {
		t.Fatalf("SubmitReview: %v", err)
	}
	if err := svc.MarkDone(context.Background(), ws, id); err != nil {
		t.Fatalf("MarkDone: %v", err)
	}
	view, _, err = svc.ReadTask(context.Background(), ws, id)
	if err != nil {
		t.Fatalf("ReadTask after done: %v", err)
	}
	if view.Status != taskStatusDone {
		t.Errorf("status = %q, want done", view.Status)
	}

	// task_results row exists
	h, cleanup, err := svc.openTeamsHandle(ws)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	var n int
	if err := h.store.DB.QueryRow(`SELECT COUNT(*) FROM task_results WHERE task_id = ?`, id).Scan(&n); err != nil {
		t.Fatalf("count results: %v", err)
	}
	if n != 1 {
		t.Errorf("task_results count = %d", n)
	}
	var reviews int
	if err := h.store.DB.QueryRow(`SELECT COUNT(*) FROM reviews WHERE task_id = ? AND passed = 1`, id).Scan(&reviews); err != nil {
		t.Fatalf("count reviews: %v", err)
	}
	if reviews != 1 {
		t.Errorf("passed reviews = %d", reviews)
	}
}

func TestEmptyPullFailsClearly(t *testing.T) {
	svc, ws := newTeamsTestService(t)
	_, err := svc.PullNextTask(context.Background(), ws, "agent-1")
	if !errors.Is(err, ErrNoPendingTasks) {
		t.Fatalf("want ErrNoPendingTasks, got %v", err)
	}
}

func TestPullRowsAffectedRejectsLostClaim(t *testing.T) {
	svc, ws := newTeamsTestService(t)
	id, err := svc.SpawnTask(context.Background(), SpawnTaskParams{
		Directory: ws, Title: "one", Brief: "b", Priority: 1,
	})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	first, err := svc.PullNextTask(context.Background(), ws, "agent-a")
	if err != nil {
		t.Fatalf("first pull: %v", err)
	}
	if first.ID != id {
		t.Fatalf("claimed %q want %q", first.ID, id)
	}
	// Simulate lost race: another claimer already owns the only pending task.
	_, err = svc.PullNextTask(context.Background(), ws, "agent-b")
	if !errors.Is(err, ErrNoPendingTasks) {
		t.Fatalf("second pull want ErrNoPendingTasks, got %v", err)
	}
}

func TestSubmitReviewUniquePathPerReview(t *testing.T) {
	svc, ws := newTeamsTestService(t)
	id, err := svc.SpawnTask(context.Background(), SpawnTaskParams{
		Directory: ws, Title: "rev", Brief: "b", Priority: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SubmitReview(context.Background(), SubmitReviewParams{
		Directory: ws, TaskID: id, ReviewerID: "r1",
		ReviewType: "spec_compliance", Passed: false, Comments: "first",
	}); err != nil {
		t.Fatalf("review1: %v", err)
	}
	if err := svc.SubmitReview(context.Background(), SubmitReviewParams{
		Directory: ws, TaskID: id, ReviewerID: "r2",
		ReviewType: "spec_compliance", Passed: true, Comments: "second",
	}); err != nil {
		t.Fatalf("review2: %v", err)
	}
	h, cleanup, err := svc.openTeamsHandle(ws)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	rows, err := h.store.DB.Query(`SELECT comments_path FROM reviews WHERE task_id = ? ORDER BY created_at`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	if len(paths) != 2 {
		t.Fatalf("want 2 review paths, got %v", paths)
	}
	if paths[0] == paths[1] {
		t.Fatalf("review paths must be unique, both %q", paths[0])
	}
	for i, p := range paths {
		body, err := h.content.Read(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		want := []string{"first", "second"}[i]
		if string(body) != want {
			t.Errorf("path %s content %q want %q", p, body, want)
		}
	}
}

// TestTeamsAtomicity ensures FS-first write + SQL failure leaves no orphan (via ContentPlane).
func TestTeamsAtomicity(t *testing.T) {
	svc, ws := newTeamsTestService(t)
	h, cleanup, err := svc.openTeamsHandle(ws)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	if err := h.ensureTeam(defaultTeamID, defaultTeamName); err != nil {
		t.Fatal(err)
	}
	_, err = h.content.Write(files.KindTasks, "atom-1", "brief.md", []byte("x"), func(relPath string) error {
		return errors.New("forced sql failure")
	})
	if err == nil {
		t.Fatal("expected commit failure")
	}
	abs := filepath.Join(ws, ".skillgrid", "files", "tasks", "atom-1", "brief.md")
	if _, statErr := os.Stat(abs); !os.IsNotExist(statErr) {
		t.Fatalf("orphan file remained: %v", statErr)
	}
}

func TestReadUnknownTaskFailsClearly(t *testing.T) {
	svc, ws := newTeamsTestService(t)
	_, _, err := svc.ReadTask(context.Background(), ws, "missing-id")
	if !errors.Is(err, ErrUnknownTask) {
		t.Fatalf("want ErrUnknownTask, got %v", err)
	}
}
