package codeindex_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"skillgrid-cli/internal/mnemonic/codeindex"
	"skillgrid-cli/internal/mnemonic/store"
)

func TestIndexerHashSkip(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "hello.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dataDir := t.TempDir()
	st, err := store.Open(dataDir, "test-project")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	idx := codeindex.New(st)
	cfg := codeindex.Config{
		Include:      []string{"**/*.go"},
		Exclude:      []string{},
		ChunkLines:   80,
		ChunkOverlap: 10,
	}

	ctx := context.Background()

	stats1, err := idx.Run(ctx, root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if stats1.FilesIndexed != 1 {
		t.Fatalf("first run: FilesIndexed=%d, want 1", stats1.FilesIndexed)
	}

	stats2, err := idx.Run(ctx, root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if stats2.FilesSkipped != 1 {
		t.Fatalf("second run: FilesSkipped=%d, want 1", stats2.FilesSkipped)
	}
	if stats2.FilesIndexed != 0 {
		t.Fatalf("second run: FilesIndexed=%d, want 0", stats2.FilesIndexed)
	}
}
