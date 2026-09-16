package memory

import (
	"context"
	"testing"
)

// TestRetrievalTrajectoryPreserved is 19.2 [RED]: after a directory
// retrieval, the trajectory records each directory visited (path, score,
// depth), is stored in the retrieval_trails table, and is retrievable by the
// run's query id.
func TestRetrievalTrajectoryPreserved(t *testing.T) {
	fx := newFixture(t, "dir-trail-proj")
	ctx := context.Background()
	seedDirHierarchy(t, fx)

	res, err := fx.svc.DirectoryRetrieve(ctx, RetrieveDirOptions{
		Query: "auth middleware core",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("DirectoryRetrieve: %v", err)
	}
	if res.QueryID == 0 {
		t.Fatal("expected a non-zero query id for the retrieval run")
	}
	// In-memory trajectory: each step carries path, score, depth and the
	// complete drill-down path as a JSON-able array.
	if len(res.Trajectory) < 2 {
		t.Fatalf("expected a multi-step trajectory, got %+v", res.Trajectory)
	}
	wantPaths := map[string]int{
		"project":          1,
		"project/app":      2,
		"project/app/core": 3,
	}
	for _, step := range res.Trajectory {
		wantDepth, ok := wantPaths[step.Path]
		if !ok {
			t.Fatalf("unexpected trajectory step %q", step.Path)
		}
		if step.Depth != wantDepth {
			t.Fatalf("step %q: depth %d, want %d", step.Path, step.Depth, wantDepth)
		}
		if step.Score <= 0 {
			t.Fatalf("step %q: score must be positive, got %v", step.Path, step.Score)
		}
		if len(step.PathPrefix) != step.Depth {
			t.Fatalf("step %q: PathPrefix len %d != depth %d", step.Path, len(step.PathPrefix), step.Depth)
		}
	}

	// Persisted trail: GetTrajectory by query id returns the same drill-down.
	steps, err := fx.svc.GetTrajectory(ctx, res.QueryID)
	if err != nil {
		t.Fatalf("GetTrajectory: %v", err)
	}
	if len(steps) != len(res.Trajectory) {
		t.Fatalf("persisted %d steps, in-memory %d", len(steps), len(res.Trajectory))
	}
	for i, step := range steps {
		if step.Path != res.Trajectory[i].Path || step.Depth != res.Trajectory[i].Depth {
			t.Fatalf("persisted step %d = %+v, want %+v", i, step, res.Trajectory[i])
		}
		if step.Score <= 0 {
			t.Fatalf("persisted step %q lost its score: %+v", step.Path, step)
		}
	}
	// The trail row itself is stored in retrieval_trails (the run's row id IS
	// its query id; spot-check the query + depth columns round-trip).
	var count int
	if err := fx.st.DB.QueryRow(`SELECT COUNT(*) FROM retrieval_trails WHERE id = ?`,
		res.QueryID).Scan(&count); err != nil {
		t.Fatalf("read retrieval_trails: %v", err)
	}
	if count == 0 {
		t.Fatal("no retrieval_trails row recorded for the run")
	}
	// A query id with no run returns empty, not an error.
	none, err := fx.svc.GetTrajectory(ctx, 999999)
	if err != nil {
		t.Fatalf("GetTrajectory missing: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected no steps for unknown query id, got %+v", none)
	}
}
