package service

import (
	"context"
	"testing"
)

// TestCodeExplainCommunityMembersAndEntryPoints covers @step-01 (Scenario:
// code_explain_community explains a subsystem): a valid community id returns
// its members (symbol + path + line) and its key entry points (top god
// nodes); a non-existent id returns a not-found reason (no invented
// community).
func TestCodeExplainCommunityMembersAndEntryPoints(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	// Open a project so the store + migrations exist.
	h, cleanup, err := svc.Open("explain-community")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db := h.Store().DB

	seedFile := func(path string) int64 {
		res, err := db.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES (?, 1, 1, 'h', 'now')`, path)
		if err != nil {
			t.Fatalf("seed file: %v", err)
		}
		id, _ := res.LastInsertId()
		return id
	}
	seedSymbol := func(fileID int64, name string, line int) int64 {
		res, err := db.Exec(`INSERT INTO symbols (file_id, name, kind, start_line, end_line, content_hash, uid) VALUES (?, ?, 'function', ?, ?, 'h', ?)`,
			fileID, name, line, line, name+"-uid")
		if err != nil {
			t.Fatalf("seed symbol: %v", err)
		}
		id, _ := res.LastInsertId()
		return id
	}
	seedEdge := func(from, to int64, line int) {
		if _, err := db.Exec(`INSERT INTO edges (kind, from_id, to_id, confidence, line) VALUES ('calls', ?, ?, 'EXTRACTED', ?)`, from, to, line); err != nil {
			t.Fatalf("seed edge: %v", err)
		}
	}

	fa := seedFile("a/core.go")
	a1 := seedSymbol(fa, "aOne", 1)
	a2 := seedSymbol(fa, "aTwo", 10)
	a3 := seedSymbol(fa, "aThree", 20)
	seedEdge(a1, a2, 2)
	seedEdge(a2, a3, 11)
	seedEdge(a1, a3, 3)

	// Run the community pass to populate the communities table.
	if _, err := svc.CodeCommunities(context.Background(), "explain-community", CommunityOptions{}); err != nil {
		t.Fatalf("code communities: %v", err)
	}
	cleanup()

	// Find a community that contains a1.
	var commID int
	found := false
	_ = db.QueryRow(`SELECT id FROM communities WHERE symbol_id = ?`, a1).Scan(&commID)
	found = commID >= 0

	if !found {
		t.Fatalf("expected a community containing a1")
	}

	out, err := svc.CodeExplainCommunity(context.Background(), "explain-community", commID)
	if err != nil {
		t.Fatalf("CodeExplainCommunity: %v", err)
	}
	if !out.Found {
		t.Fatalf("expected found, got reason %q", out.Reason)
	}
	if len(out.Members) == 0 {
		t.Fatalf("expected members, got none")
	}
	// Members carry symbol + path + line.
	if m := out.Members[0]; m["path"] == nil || m["name"] == nil || m["start_line"] == nil {
		t.Errorf("members must carry name/path/start_line, got %v", m)
	}
	// Entry points are the top god nodes (at least the highest-degree member).
	if len(out.EntryPts) == 0 {
		t.Errorf("expected entry points, got none")
	}

	// A non-existent community id → not found (no invented community).
	bad, err := svc.CodeExplainCommunity(context.Background(), "explain-community", 9999)
	if err != nil {
		t.Fatalf("CodeExplainCommunity bad id: %v", err)
	}
	if bad.Found {
		t.Errorf("non-existent community 9999 should be not found")
	}
	if bad.Reason == "" {
		t.Errorf("not-found explanation should carry a reason")
	}
}
