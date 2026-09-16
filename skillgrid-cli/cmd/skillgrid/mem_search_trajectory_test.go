package main

import (
	"context"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedDirTrajectoryProject plants hierarchical observations (path-like
// topic_keys) so `mem search --trajectory` has a directory to drill down.
func seedDirTrajectoryProject(t *testing.T, dataDir, project string) {
	t.Helper()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	mem := memory.New(st, project)
	ctx := context.Background()
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('sess-dirtraj', ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	saves := []memory.SaveInput{
		{SessionID: "sess-dirtraj", Type: "architecture", TopicKey: "project/app/core",
			Title: "Core auth middleware design", Content: "auth middleware core pipeline"},
		{SessionID: "sess-dirtraj", Type: "decision", TopicKey: "project/app/api",
			Title: "REST api endpoint layout", Content: "rest api endpoints layout"},
		{SessionID: "sess-dirtraj", Type: "decision", TopicKey: "project/docs",
			Title: "Docs site structure", Content: "docs site structure notes"},
	}
	for _, in := range saves {
		if _, err := mem.Save(ctx, in); err != nil {
			t.Fatalf("save %s: %v", in.TopicKey, err)
		}
	}
}

// TestSearchTrajectoryCLI is the `mem search --trajectory` [RED] test: the
// flag runs the directory retrieval and prints the drill-down trajectory
// (query_id, path per step, scores, depth) alongside the search results.
func TestSearchTrajectoryCLI(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-dirtraj"
	seedDirTrajectoryProject(t, dataDir, project)

	out := runMemCLI(t, dataDir, "search", "auth", "middleware",
		"--trajectory", "--project", project, "--dir", dataDir)

	if !strings.Contains(out, `"trajectory":`) {
		t.Fatalf("mem search --trajectory must include the trajectory: %s", out)
	}
	if !strings.Contains(out, `"query_id"`) {
		t.Fatalf("mem search --trajectory must include the query id: %s", out)
	}
	// The drill-down path into the highest-scoring directory is shown.
	if !strings.Contains(out, "project/app") || !strings.Contains(out, "project/app/core") {
		t.Fatalf("mem search --trajectory must show the drill-down path: %s", out)
	}
	if !strings.Contains(out, `"depth": 1`) {
		t.Fatalf("mem search --trajectory must show step depths: %s", out)
	}
}
