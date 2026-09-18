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

// TestMemFSCoexistsWithMemory is 25.5 [AFK] — `mem fs` now browses the CODE
// INDEX, not memory observations. It verifies (a) ls/tree/find/cat over the
// code index, and (b) that the memory store is untouched: `mem list` still
// returns observations seeded alongside the code index.
func TestMemFSCoexistsWithMemory(t *testing.T) {
	dataDir := t.TempDir()
	project := "memfs-code-coexist"

	st := openStoreForTest(t, dataDir, project)
	defer st.Close()
	db := st.DB
	ctx := context.Background()

	// Seed the code index: one file with one symbol + one chunk.
	if _, err := db.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES (?, 1, 100, 'h1', '2026-01-01T00:00:00Z')`, "src/auth/login.go"); err != nil {
		t.Fatalf("insert file: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES (?, 1, 50, 'h2', '2026-01-01T00:00:00Z')`, "src/util.go"); err != nil {
		t.Fatalf("insert file util: %v", err)
	}
	var loginID int64
	if err := db.QueryRow(`SELECT id FROM files WHERE path = 'src/auth/login.go'`).Scan(&loginID); err != nil {
		t.Fatalf("login id: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO symbols (file_id, name, kind, language, signature, start_line, end_line, content_hash, uid) VALUES (?, 'Handler', 'function', 'go', 'func Handler() error', 1, 10, 'c1', 'uid-login-handler')`, loginID); err != nil {
		t.Fatalf("insert symbol: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO chunks (file_id, start_line, end_line, text, content_hash) VALUES (?, 1, 10, 'func Handler() error { return nil }', 'ck1')`, loginID); err != nil {
		t.Fatalf("insert chunk: %v", err)
	}

	// Seed a memory observation (proves the memory store coexists + is untouched).
	mem := memory.New(st, project)
	if _, err := db.Exec(`INSERT INTO sessions (id, project, directory, started_at, status) VALUES ('sess-code-coexist', ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := mem.Save(ctx, memory.SaveInput{
		SessionID: "sess-code-coexist", Type: "learning", Title: "mem-note",
		Content: "a memory note", TopicKey: "project/A/preferences/x", MemoryType: "preferences",
	}); err != nil {
		t.Fatalf("save obs: %v", err)
	}

	// mem fs ls (repo root) → the "src" dir, NOT observations.
	out := runMemCLI(t, dataDir, "fs", "ls", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "src") {
		t.Fatalf("mem fs ls should list the code index (src dir), got: %s", out)
	}
	if strings.Contains(out, "mem-note") {
		t.Errorf("mem fs ls should NOT list memory observations, got: %s", out)
	}

	// mem fs ls <file> → its symbols.
	out = runMemCLI(t, dataDir, "fs", "ls", "src/auth/login.go", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "Handler") {
		t.Fatalf("mem fs ls <file> should list symbols, got: %s", out)
	}

	// mem fs tree → repo tree with symbol counts.
	out = runMemCLI(t, dataDir, "fs", "tree", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "login.go") || !strings.Contains(out, "1 symbols") {
		t.Fatalf("mem fs tree should show the code tree + symbol count, got: %s", out)
	}

	// mem fs find *.go → the two files.
	out = runMemCLI(t, dataDir, "fs", "find", "*.go", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "login.go") || !strings.Contains(out, "util.go") {
		t.Fatalf("mem fs find *.go should match both files, got: %s", out)
	}

	// mem fs cat <file>::<symbol> → source.
	out = runMemCLI(t, dataDir, "fs", "cat", "src/auth/login.go::Handler", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "func Handler() error") {
		t.Fatalf("mem fs cat should return symbol source, got: %s", out)
	}

	// Memory is untouched: mem list still shows the observation (flat).
	out = runMemCLI(t, dataDir, "list", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "mem-note") {
		t.Errorf("mem list should still show observations after mem fs (code) ops, got: %s", out)
	}
}

// TestMemFSCatUnknownErrors is 25.5b — `mem fs cat` on an unknown symbol
// exits non-zero.
func TestMemFSCatUnknownErrors(t *testing.T) {
	dataDir := t.TempDir()
	project := "memfs-cat-err"
	st := openStoreForTest(t, dataDir, project)
	defer st.Close()
	db := st.DB
	if _, err := db.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES (?, 1, 100, 'h1', '2026-01-01T00:00:00Z')`, "src/a.go"); err != nil {
		t.Fatalf("insert file: %v", err)
	}
	_, err := runMemCLIExpectError(t, dataDir, "fs", "cat", "src/a.go::Nope", "--project", project, "--dir", dataDir)
	if err == nil {
		t.Fatalf("mem fs cat <unknown symbol> should error")
	}
}
