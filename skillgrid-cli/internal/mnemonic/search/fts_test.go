package search_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"skillgrid-cli/internal/mnemonic/codeindex"
	"skillgrid-cli/internal/mnemonic/search"
	"skillgrid-cli/internal/mnemonic/store"
)

func TestCodeSearchFindsIndexedString(t *testing.T) {
	root := t.TempDir()
	const marker = "UniqueSymbolXYZ123"
	filePath := filepath.Join(root, "sample.go")
	content := "package main\n\nfunc " + marker + "() {}\n"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	dataDir := t.TempDir()
	st, err := store.Open(dataDir, "test-project")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	idx := codeindex.New(st)
	if _, err := idx.Run(context.Background(), root, codeindex.Config{
		Include:      []string{"**/*.go"},
		Exclude:      []string{},
		ChunkLines:   80,
		ChunkOverlap: 10,
	}); err != nil {
		t.Fatal(err)
	}

	hits, err := search.CodeSearch(st.DB, marker, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected at least one hit")
	}

	hit := hits[0]
	if hit.Path != "sample.go" {
		t.Fatalf("path=%q, want sample.go", hit.Path)
	}
	if hit.StartLine < 1 || hit.EndLine < hit.StartLine {
		t.Fatalf("line range=%d-%d, want valid range", hit.StartLine, hit.EndLine)
	}
	if !strings.Contains(hit.Snippet, marker) {
		t.Fatalf("snippet=%q, want to contain %q", hit.Snippet, marker)
	}
	if hit.Score == 0 {
		t.Fatalf("expected non-zero score")
	}
}

func TestCodeSearchEmptyQuery(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, "test-project")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	hits, err := search.CodeSearch(st.DB, "   ", 10)
	if err != nil {
		t.Fatal(err)
	}
	if hits != nil {
		t.Fatalf("expected nil hits for empty query, got %d", len(hits))
	}
}
