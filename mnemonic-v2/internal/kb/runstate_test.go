package kb

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

const runFP = "sha256:test-fingerprint"

func TestBeginFreshAndResume(t *testing.T) {
	ctx := context.Background()
	st := openKbStore(t)
	root := t.TempDir()

	rs, err := Begin(ctx, st, root, "proj-a", "init", runFP)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if rs.Phase != PhasePlanning || rs.RunID == "" || rs.Mode != "init" {
		t.Fatalf("fresh run: %+v", rs)
	}
	if _, err := os.Stat(filepath.Join(root, "index", "last-update.json")); err != nil {
		t.Fatalf("projection missing: %v", err)
	}

	// Same fingerprint → resume the same run.
	rs2, err := Begin(ctx, st, root, "proj-a", "init", runFP)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if rs2.RunID != rs.RunID || rs2.Phase != PhasePlanning {
		t.Fatalf("resume drifted: %+v vs %+v", rs2, rs)
	}
}

func TestBeginDriftClearsPlanKeepsProcessed(t *testing.T) {
	ctx := context.Background()
	st := openKbStore(t)
	root := t.TempDir()

	if _, err := Begin(ctx, st, root, "proj-a", "init", runFP); err != nil {
		t.Fatalf("begin: %v", err)
	}
	jobs := []PageJob{{RelPath: "decisions/a.md", Title: "A", Purpose: "p"}}
	if err := SetPlan(ctx, st, root, "proj-a", jobs); err != nil {
		t.Fatalf("plan: %v", err)
	}
	rs, err := LoadRun(ctx, st, "proj-a")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	rs.Processed["obs-1"] = "h1"
	if err := saveRun(ctx, st, rs); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Drift → replan: plan deleted, processed survives.
	rs2, err := Begin(ctx, st, root, "proj-a", "update", "sha256:changed")
	if err != nil {
		t.Fatalf("begin drift: %v", err)
	}
	if rs2.Phase != PhasePlanning || rs2.Plan != nil {
		t.Fatalf("drift must reset to planning with a nil plan: %+v", rs2)
	}
	if rs2.Processed["obs-1"] != "h1" {
		t.Fatalf("processed must survive drift: %+v", rs2.Processed)
	}
}

func TestResumePromotesCompletedPendingJob(t *testing.T) {
	ctx := context.Background()
	st := openKbStore(t)
	root := t.TempDir()

	if _, err := Begin(ctx, st, root, "proj-a", "init", runFP); err != nil {
		t.Fatalf("begin: %v", err)
	}
	jobs := []PageJob{{RelPath: "decisions/a.md", Title: "A", Purpose: "p"}}
	if err := SetPlan(ctx, st, root, "proj-a", jobs); err != nil {
		t.Fatalf("plan: %v", err)
	}
	// The job's page bytes + baseline land (the crash window: page written,
	// MarkJob never ran).
	page := "---\ntitle: A\ntype: Decision\n---\n\nbody"
	writeVaultPage(t, root, "decisions/a.md", page)
	if err := RecordBaseline(ctx, st, root, "proj-a", "decisions/a.md", runFP, hashBody(page), "run-x", "agent"); err != nil {
		t.Fatalf("baseline: %v", err)
	}
	// Manifest projection must have been written.
	if _, err := os.Stat(filepath.Join(root, "index", ".page-manifest.json")); err != nil {
		t.Fatalf("manifest projection missing: %v", err)
	}

	// Resume with the same fingerprint → the pending job is promoted.
	rs, err := Begin(ctx, st, root, "proj-a", "init", runFP)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if len(rs.Plan) != 1 || rs.Plan[0].Status != JobComplete || rs.Plan[0].CompletedBy != "agent" {
		t.Fatalf("promotion failed: %+v", rs.Plan)
	}
}

func TestSetPlanValidatesPaths(t *testing.T) {
	ctx := context.Background()
	st := openKbStore(t)
	root := t.TempDir()
	if _, err := Begin(ctx, st, root, "proj-a", "init", runFP); err != nil {
		t.Fatalf("begin: %v", err)
	}
	for _, bad := range []string{
		"decisions/a/b.md",  // wrong depth
		"notes/a.md",        // unknown type folder
		"decisions/a.txt",   // not .md
		"../decisions/a.md", // traversal
	} {
		if err := SetPlan(ctx, st, root, "proj-a", []PageJob{{RelPath: bad}}); err == nil {
			t.Fatalf("path %q should be rejected", bad)
		}
	}
	// Duplicate detection (two jobs, same path).
	err := SetPlan(ctx, st, root, "proj-a",
		[]PageJob{{RelPath: "decisions/a.md"}, {RelPath: "decisions/a.md"}})
	if err == nil {
		t.Fatal("duplicate planned page should be rejected")
	}
}

func TestFinishCleanAndFailure(t *testing.T) {
	ctx := context.Background()
	st := openKbStore(t)
	root := t.TempDir()
	if _, err := Begin(ctx, st, root, "proj-a", "full", runFP); err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := SetPlan(ctx, st, root, "proj-a", []PageJob{{RelPath: "decisions/a.md"}}); err != nil {
		t.Fatalf("plan: %v", err)
	}

	// Failure → the checkpoint survives intact (phase generating, plan kept).
	if err := Finish(ctx, st, root, "proj-a", false, false); err != nil {
		t.Fatalf("finish fail: %v", err)
	}
	rs, err := LoadRun(ctx, st, "proj-a")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if rs.Phase != PhaseGenerating || len(rs.Plan) != 1 {
		t.Fatalf("failure must keep the resumable checkpoint: %+v", rs)
	}

	// Clean → idle, plan cleared, last_full_run_at set.
	if err := Finish(ctx, st, root, "proj-a", true, true); err != nil {
		t.Fatalf("finish clean: %v", err)
	}
	rs, err = LoadRun(ctx, st, "proj-a")
	if err != nil {
		t.Fatalf("load2: %v", err)
	}
	if rs.Phase != PhaseIdle || rs.Plan != nil || rs.LastFullRunAt == "" {
		t.Fatalf("clean finish: %+v", rs)
	}
}
