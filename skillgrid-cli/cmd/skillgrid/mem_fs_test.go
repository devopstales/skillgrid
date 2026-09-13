package main

import (
	"context"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// openStoreForTest opens a store for CLI tests.
func openStoreForTest(t *testing.T, dataDir, project string) *store.Store {
	t.Helper()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return st
}

// TestMemLSCoexistsWithMemList is 25.5 [AFK] — `mem fs ls` coexists with
// `mem list` without conflict. Both query the same store but with different
// filters (scoped vs flat).
func TestMemLSCoexistsWithMemList(t *testing.T) {
	dataDir := t.TempDir()
	project := "memfs-coexist"

	// Seed observations with topic_keys in different scopes.
	st := openStoreForTest(t, dataDir, project)
	defer st.Close()
	mem := memory.New(st, project)
	ctx := context.Background()
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('sess-coexist', ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	for _, o := range []struct {
		title  string
		topic  string
		memory string
	}{
		{"pref-x", "project/A/preferences/x", "preferences"},
		{"pref-y", "project/A/preferences/y", "preferences"},
		{"ent-z", "project/A/entities/z", "entities"},
	} {
		if _, err := mem.Save(ctx, memory.SaveInput{
			SessionID:  "sess-coexist",
			Type:       "learning",
			Title:      o.title,
			Content:    o.title + " content",
			TopicKey:   o.topic,
			MemoryType: o.memory,
		}); err != nil {
			t.Fatalf("save %q: %v", o.title, err)
		}
	}

	// mem list (flat, all observations).
	out := runMemCLI(t, dataDir, "list", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "pref-x") || !strings.Contains(out, "ent-z") {
		t.Fatalf("mem list should show all observations (flat), got: %s", out)
	}

	// mem fs ls project/A/preferences → only preferences (scoped).
	out = runMemCLI(t, dataDir, "fs", "ls", "project/A/preferences", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "pref-x") || !strings.Contains(out, "pref-y") {
		t.Fatalf("mem fs ls project/A/preferences should show preferences, got: %s", out)
	}
	if strings.Contains(out, "ent-z") {
		t.Errorf("mem fs ls project/A/preferences should NOT show entities, got: %s", out)
	}

	// mem fs tree project/A/ → hierarchical view.
	out = runMemCLI(t, dataDir, "fs", "tree", "project/A/", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "preferences") || !strings.Contains(out, "entities") {
		t.Fatalf("mem fs tree project/A/ should show both dirs, got: %s", out)
	}

	// mem fs find "pref*" project/A/ → glob search.
	out = runMemCLI(t, dataDir, "fs", "find", "pref*", "project/A/", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "pref-x") {
		t.Fatalf("mem fs find pref* should match pref-x, got: %s", out)
	}

	// Verify mem list is unaffected (still flat).
	out = runMemCLI(t, dataDir, "list", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "ent-z") {
		t.Errorf("mem list should still show all observations after mem fs operations, got: %s", out)
	}
}
