package service

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

func seedProjectObs(t *testing.T, svc *Service, projectID, title, content string) {
	t.Helper()
	h, cleanup, err := svc.openProject(projectID, ".")
	if err != nil {
		t.Fatalf("open %s: %v", projectID, err)
	}
	defer cleanup()
	sid, err := h.memory.SessionStart(context.Background(), t.TempDir(), "seed")
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	if _, err := h.memory.Save(context.Background(), memory.SaveInput{
		Title: title, Type: "decision", Content: content, SessionID: sid, Scope: "project",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
}

func TestSearchAllProjectsMergesTwoStores(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	seedProjectObs(t, svc, "store-alpha", "alpha unique token xyz", "**What** a **Why** t **Where** t **Learned** —")
	seedProjectObs(t, svc, "store-beta", "beta unique token xyz", "**What** b **Why** t **Where** t **Learned** —")

	hits, err := svc.SearchAllProjects(context.Background(), "unique token xyz", "any", "", 20)
	if err != nil {
		t.Fatalf("SearchAllProjects: %v", err)
	}
	projects := map[string]bool{}
	for _, h := range hits {
		projects[h.Project] = true
	}
	if !projects["store-alpha"] || !projects["store-beta"] {
		t.Fatalf("expected both stores in merge, got projects=%v hits=%d", projects, len(hits))
	}
}

func TestSearchAllProjectsEmptyDir(t *testing.T) {
	svc := New(t.TempDir())
	hits, err := svc.SearchAllProjects(context.Background(), "nothing-here", "any", "", 10)
	if err != nil {
		t.Fatalf("empty search must not error: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("want 0 hits, got %d", len(hits))
	}
}

func TestSearchAllProjectsSpansEveryStore(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	for _, pid := range []string{"p1", "p2", "p3"} {
		seedProjectObs(t, svc, pid, "shared recall phrase", "**What** x **Why** x **Where** x **Learned** — "+pid)
	}
	hits, err := svc.SearchAllProjects(context.Background(), "shared recall phrase", "all", "", 50)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, h := range hits {
		seen[h.Project] = true
	}
	if len(seen) != 3 {
		t.Fatalf("want 3 projects, got %v", seen)
	}
}

func TestUnifyIdempotent(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	seedProjectObs(t, svc, "legacy-hash", "unify payload", "**What** u **Why** u **Where** u **Learned** —")

	ctx := context.Background()
	moved1, err := svc.Unify(ctx, "canonical-proj", "legacy-hash")
	if err != nil {
		t.Fatalf("first unify: %v", err)
	}
	moved2, err := svc.Unify(ctx, "canonical-proj", "legacy-hash")
	if err != nil {
		t.Fatalf("second unify must be idempotent: %v", err)
	}
	if moved1 < 0 || moved2 < 0 {
		t.Fatalf("moved counts negative: %d %d", moved1, moved2)
	}
}

func TestUnifyFragmentedStoresOneLogicalIndex(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	seedProjectObs(t, svc, "frag-a", "fragmented secret token", "**What** f **Why** f **Where** f **Learned** —")

	ctx := context.Background()
	if _, err := svc.Unify(ctx, "frag-canonical", "frag-a"); err != nil {
		t.Fatalf("unify: %v", err)
	}
	hits, err := svc.SearchObservationsScoped(ctx, "frag-canonical", "fragmented secret token", "any", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatalf("canonical store should find unified content")
	}
}

func TestOpenForDirectoryIdempotentAcrossWorktrees(t *testing.T) {
	main := t.TempDir()
	initGitRepo(t, main)
	runGit(t, main, "remote", "add", "origin", "git@github.com:acme/remap-wt.git")

	dataDir := t.TempDir()
	svc := New(dataDir)
	h1, cleanup1, err := svc.OpenForDirectory(main)
	if err != nil {
		t.Fatalf("main open: %v", err)
	}
	id1 := h1.ProjectID()
	cleanup1()

	wt := filepath.Join(filepath.Dir(main), "remap-wt-linked")
	runGit(t, main, "worktree", "add", "--detach", wt)
	t.Cleanup(func() {
		_ = exec.Command("git", "-C", main, "worktree", "remove", "--force", wt).Run()
	})

	h2, cleanup2, err := svc.OpenForDirectory(wt)
	if err != nil {
		t.Fatalf("worktree open: %v", err)
	}
	defer cleanup2()
	if h2.ProjectID() != id1 {
		t.Fatalf("remapped open: wt=%q main=%q", h2.ProjectID(), id1)
	}
}
