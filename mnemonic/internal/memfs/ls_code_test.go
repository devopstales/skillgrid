package memfs

import (
	"context"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// seedCodeIndex inserts two indexed files (src/auth/login.go with one symbol
// + one chunk, src/util.go with none) into a fresh store, for code-index tests.
func seedCodeIndex(t *testing.T, st *store.Store) {
	t.Helper()
	db := st.DB
	if _, err := db.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES (?, 1, 100, 'h1', '2026-01-01T00:00:00Z')`, "src/auth/login.go"); err != nil {
		t.Fatalf("insert file login: %v", err)
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
}

func TestListRepoRootListsDirsAndFiles(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-ls")
	t.Cleanup(func() { st.Close() })
	seedCodeIndex(t, st)
	fs := New(st, "code-ls")

	entries, err := fs.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List(root): %v", err)
	}
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name] = true
	}
	if !names["src"] {
		t.Errorf("root listing missing dir 'src': %+v", entries)
	}
}

func TestListFileListsSymbols(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-ls2")
	t.Cleanup(func() { st.Close() })
	seedCodeIndex(t, st)
	fs := New(st, "code-ls2")

	entries, err := fs.List(context.Background(), "src/auth/login.go")
	if err != nil {
		t.Fatalf("List(file): %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("List(file) = %d entries, want 1 symbol: %+v", len(entries), entries)
	}
	e := entries[0]
	if e.Kind != "symbol" || e.Name != "Handler" || e.Signature != "func Handler() error" || e.StartLine != 1 || e.EndLine != 10 {
		t.Errorf("symbol entry = %+v", e)
	}
}

func TestListEmptyStoreReturnsEmpty(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-ls-empty")
	t.Cleanup(func() { st.Close() })
	fs := New(st, "code-ls-empty")
	entries, err := fs.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List(empty) should not error, got %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("List(empty) = %+v, want 0", entries)
	}
}

func TestTreeEmptyStoreNotesNoIndex(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-tree-empty")
	t.Cleanup(func() { st.Close() })
	fs := New(st, "code-tree-empty")
	tree, err := fs.Tree(context.Background(), "")
	if err != nil {
		t.Fatalf("Tree(empty) should not error, got %v", err)
	}
	if !strings.Contains(tree, "no code index") {
		t.Errorf("Tree(empty) = %q, want 'no code index' note", tree)
	}
}
