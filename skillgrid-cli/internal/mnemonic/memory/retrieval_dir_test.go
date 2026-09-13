package memory

import (
	"context"
	"testing"
)

// seedDirHierarchy seeds observations under path-like topic_keys so the
// directory hierarchy is project/app/core, project/app/api, project/docs.
// The core observations carry the query words; the others do not.
func seedDirHierarchy(t *testing.T, fx *fixture) {
	t.Helper()
	ctx := context.Background()
	saves := []SaveInput{
		{SessionID: session1, Type: "architecture", TopicKey: "project/app/core",
			Title:   "Core auth middleware design",
			Content: "auth middleware core pipeline for login"},
		{SessionID: session1, Type: "architecture", TopicKey: "project/app/core",
			Title:   "Core token refresh loop",
			Content: "token refresh core loop auth"},
		{SessionID: session1, Type: "decision", TopicKey: "project/app/api",
			Title:   "REST api endpoint layout",
			Content: "rest api endpoints layout"},
		{SessionID: session1, Type: "decision", TopicKey: "project/docs",
			Title:   "Docs site structure",
			Content: "docs site structure notes"},
	}
	for _, in := range saves {
		if _, err := fx.svc.Save(ctx, in); err != nil {
			t.Fatalf("save %s: %v", in.TopicKey, err)
		}
	}
}

// TestDirectoryRetrievalDrillDown is 19.1 [RED]: the directory retrieval first
// identifies project/app as the highest-scoring top-level directory, drills
// down into it, then identifies core as the highest-scoring child, and the
// final results are scoped to the drilled-down leaf directory.
func TestDirectoryRetrievalDrillDown(t *testing.T) {
	fx := newFixture(t, "dir-retrieve-proj")
	ctx := context.Background()
	seedDirHierarchy(t, fx)

	res, err := fx.svc.DirectoryRetrieve(ctx, RetrieveDirOptions{
		Query: "auth middleware core",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("DirectoryRetrieve: %v", err)
	}
	if len(res.Results) == 0 {
		t.Fatal("expected results scoped to the drilled-down directory")
	}
	// The trajectory must show the full drill-down: the top-level directory
	// (project), then project/app, then the leaf project/app/core.
	wantSteps := []struct {
		path  string
		depth int
	}{
		{"project", 1},
		{"project/app", 2},
		{"project/app/core", 3},
	}
	if len(res.Trajectory) != len(wantSteps) {
		t.Fatalf("expected %d-step drill-down trajectory, got %d steps: %+v", len(wantSteps), len(res.Trajectory), res.Trajectory)
	}
	for i, want := range wantSteps {
		if res.Trajectory[i].Path != want.path {
			t.Fatalf("step %d: path %q, want %q", i, res.Trajectory[i].Path, want.path)
		}
		if res.Trajectory[i].Depth != want.depth {
			t.Fatalf("step %d: depth %d, want %d", i, res.Trajectory[i].Depth, want.depth)
		}
		if res.Trajectory[i].Score <= 0 {
			t.Fatalf("step %d: score must be positive, got %v", i, res.Trajectory[i].Score)
		}
	}
	if res.Leaf != "project/app/core" {
		t.Fatalf("expected leaf project/app/core, got %q", res.Leaf)
	}
	// All returned results must live under the leaf directory.
	for _, o := range res.Results {
		if !topicKeyUnder(o.TopicKey, "project/app/core") {
			t.Fatalf("result %q not under drilled-down leaf %q", o.TopicKey, res.Leaf)
		}
	}
}

// TestDirectoryRetrievalDepthLimit is 19.4 [AFK]: a hierarchy 10 levels deep
// stops at depth 5 (the configured limit) and returns the best results found
// at that depth — no stack overflow, no infinite loop.
func TestDirectoryRetrievalDepthLimit(t *testing.T) {
	fx := newFixture(t, "dir-depth-proj")
	ctx := context.Background()

	// A 10-level-deep chain, each level carrying the query words so every
	// level scores and the drill-down wants to keep going past the limit.
	parts := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	for i := 0; i < len(parts); i++ {
		key := parts[0]
		for _, p := range parts[1 : i+1] {
			key += "/" + p
		}
		if _, err := fx.svc.Save(ctx, SaveInput{
			SessionID: session1,
			Type:      "decision",
			TopicKey:  key,
			Title:     "deep node " + parts[i],
			Content:   "deepchain marker words " + parts[i],
		}); err != nil {
			t.Fatalf("save %s: %v", key, err)
		}
	}

	res, err := fx.svc.DirectoryRetrieve(ctx, RetrieveDirOptions{
		Query: "deepchain marker words",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("DirectoryRetrieve deep: %v", err)
	}
	// The trajectory must stop at depth 5: exactly 5 visits, last at depth 5.
	if len(res.Trajectory) != retrievalMaxDepth {
		t.Fatalf("expected %d trajectory steps at the depth limit, got %d: %+v",
			retrievalMaxDepth, len(res.Trajectory), res.Trajectory)
	}
	last := res.Trajectory[len(res.Trajectory)-1]
	if last.Depth != retrievalMaxDepth {
		t.Fatalf("expected last step at depth %d, got %d", retrievalMaxDepth, last.Depth)
	}
	// The results come from the directory at the limit (a/b/c/d/e), and the
	// leaf is that path — the drill-down did not run past the limit.
	wantLeaf := "a/b/c/d/e"
	if res.Leaf != wantLeaf {
		t.Fatalf("expected leaf at depth limit %q, got %q", wantLeaf, res.Leaf)
	}
	for _, o := range res.Results {
		if !topicKeyUnder(o.TopicKey, wantLeaf) {
			t.Fatalf("result %q not under depth-limit leaf %q", o.TopicKey, wantLeaf)
		}
	}
}

// TestIntentAnalysisClassification is 19.3 [RED]: ClassifyIntent maps a query
// to one of exploration/debugging/review/refactor, and the intent adjusts the
// retrieval strategy (debugging → wider scope, exploration → directory-first).
func TestIntentAnalysisClassification(t *testing.T) {
	cases := []struct {
		query string
		want  Intent
	}{
		{"why does the build fail", IntentDebugging},
		{"what files are in the auth module", IntentExploration},
		{"review the changes to payment.go", IntentReview},
		{"how should I split this function", IntentRefactor},
	}
	for _, c := range cases {
		got := ClassifyIntent(c.query)
		if got != c.want {
			t.Fatalf("ClassifyIntent(%q) = %v, want %v", c.query, got, c.want)
		}
	}
}
