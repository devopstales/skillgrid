package memfs

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// newCodeFSTestFixture opens a store and returns a MemFS bound to it.
func newCodeFSTestFixture(t *testing.T, project string) (*MemFS, *store.Store) {
	t.Helper()
	st, err := store.Open(t.TempDir(), project)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return New(st, project), st
}

// TestListDirectoryAndFile re-points 25.1 to the code index: `ls` on a
// directory lists subdirs + files; `ls` on a file lists its symbols.
func TestListDirectoryAndFile(t *testing.T) {
	fs, st := newCodeFSTestFixture(t, "code-ls-dirfile")
	seedCodeIndex(t, st)
	ctx := context.Background()

	// Root → the top-level "src" directory.
	res, err := fs.List(ctx, "")
	if err != nil {
		t.Fatalf("ls root: %v", err)
	}
	var foundSrc bool
	for _, e := range res {
		if e.Name == "src" && e.Kind == "dir" {
			foundSrc = true
		}
	}
	if !foundSrc {
		t.Errorf("ls root: expected dir 'src', got %+v", res)
	}

	// src/ → files auth (dir, since login.go is nested) — actually src/ holds
	// "auth" dir (login.go under it) and "util.go" file.
	res, err = fs.List(ctx, "src/")
	if err != nil {
		t.Fatalf("ls src/: %v", err)
	}
	names := map[string]string{}
	for _, e := range res {
		names[e.Name] = e.Kind
	}
	if names["auth"] != "dir" || names["util.go"] != "file" {
		t.Errorf("ls src/: expected auth dir + util.go file, got %+v", names)
	}

	// src/auth/login.go → its symbols.
	res, err = fs.List(ctx, "src/auth/login.go")
	if err != nil {
		t.Fatalf("ls file: %v", err)
	}
	if len(res) != 1 || res[0].Kind != "symbol" || res[0].Name != "Handler" {
		t.Errorf("ls file: expected symbol Handler, got %+v", res)
	}
}

// TestFindGlob re-points 25.3 to the code index: glob over file paths + symbol names.
func TestFindGlob(t *testing.T) {
	fs, st := newCodeFSTestFixture(t, "code-find-glob")
	seedCodeIndex(t, st)
	ctx := context.Background()

	// *.go → both files.
	res, err := fs.Find(ctx, "*.go", "")
	if err != nil {
		t.Fatalf("find *.go: %v", err)
	}
	var fileCount, symCount int
	for _, e := range res {
		switch e.Kind {
		case "file":
			fileCount++
		case "symbol":
			symCount++
		}
	}
	if fileCount != 2 {
		t.Errorf("find *.go: expected 2 files, got %d (%+v)", fileCount, res)
	}
	if symCount != 0 {
		t.Errorf("find *.go: expected 0 symbols, got %d (%+v)", symCount, res)
	}

	// Handler → the symbol.
	res, err = fs.Find(ctx, "Handler", "")
	if err != nil {
		t.Fatalf("find Handler: %v", err)
	}
	if len(res) != 1 || res[0].Kind != "symbol" || res[0].Name != "Handler" {
		t.Errorf("find Handler: expected 1 symbol Handler, got %+v", res)
	}

	// Scoped find narrows to a directory.
	res, err = fs.Find(ctx, "*.go", "src/auth/")
	if err != nil {
		t.Fatalf("find scoped: %v", err)
	}
	if len(res) != 1 || res[0].Name != "src/auth/login.go" {
		t.Errorf("find scoped: expected only login.go, got %+v", res)
	}
}
