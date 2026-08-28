package codeindex

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func openStoreFor(t *testing.T) (*store.Store, string, func()) {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, "code-test")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	clean := func() { st.Close() }
	return st, dataDir, clean
}

func newTestIndexer(t *testing.T) (*Indexer, func()) {
	st, _, clean := openStoreFor(t)
	return New(st), clean
}

func writeTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "main.go"), "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n")
	mustWrite(t, filepath.Join(root, "lib.ts"), "export function hello() { return 'world' }\n")
	mustWrite(t, filepath.Join(root, "node_modules", "dep.js"), "/* excluded */\n")
	return root
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

var testCfg = Config{
	Include:      []string{"**/*.go", "**/*.ts"},
	Exclude:      []string{"**/node_modules/**", "**/.git/**"},
	ChunkLines:   80,
	ChunkOverlap: 10,
}

// TestColdIndex indexes an empty store from scratch.
func TestColdIndex(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeTestRepo(t)
	stats, err := idx.Run(context.Background(), root, testCfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if stats.FilesIndexed != 2 {
		t.Errorf("expected 2 files indexed, got %d (node_modules should be excluded)", stats.FilesIndexed)
	}
	if stats.ChunksAdded < 2 {
		t.Errorf("expected at least 2 chunks added, got %d", stats.ChunksAdded)
	}
}

// TestWarmNoOp second run on an unchanged tree skips all files.
func TestWarmNoOp(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeTestRepo(t)
	if _, err := idx.Run(context.Background(), root, testCfg); err != nil {
		t.Fatalf("first run: %v", err)
	}
	// No changes between runs: mtime/size/hash are identical, so the warm
	// path skips every file.
	stats, err := idx.Run(context.Background(), root, testCfg)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if stats.FilesIndexed != 0 {
		t.Errorf("expected 0 files indexed (warm), got %d", stats.FilesIndexed)
	}
	if stats.FilesSkipped != 2 {
		t.Errorf("expected 2 files skipped, got %d", stats.FilesSkipped)
	}
}

// TestIncrementalUpdate only reindexes changed files.
func TestIncrementalUpdate(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeTestRepo(t)
	if _, err := idx.Run(context.Background(), root, testCfg); err != nil {
		t.Fatalf("first run: %v", err)
	}
	// Modify lib.ts with new content.
	if err := os.WriteFile(filepath.Join(root, "lib.ts"), []byte("export function hello2() { return 'world2' }\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	stats, err := idx.Run(context.Background(), root, testCfg)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if stats.FilesIndexed != 1 {
		t.Errorf("expected 1 file reindexed, got %d", stats.FilesIndexed)
	}
	if stats.FilesSkipped != 1 {
		t.Errorf("expected 1 file skipped, got %d", stats.FilesSkipped)
	}
}

// TestExcludeGlobs verifies that node_modules and .git are excluded.
func TestExcludeGlobs(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeTestRepo(t)
	mustWrite(t, filepath.Join(root, ".git", "config"), "[core]\n")
	stats, err := idx.Run(context.Background(), root, testCfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if stats.FilesIndexed != 2 {
		t.Errorf("expected 2 files indexed (node_modules/.git excluded), got %d", stats.FilesIndexed)
	}
}

// TestDeleteRemovedFiles verifies that a deleted file's chunks are removed.
func TestDeleteRemovedFiles(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeTestRepo(t)
	mustWrite(t, filepath.Join(root, "extra.go"), "package main\n\nvar extra = true\n")
	if _, err := idx.Run(context.Background(), root, testCfg); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if err := os.Remove(filepath.Join(root, "extra.go")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	stats, err := idx.Run(context.Background(), root, testCfg)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if stats.FilesDeleted != 1 {
		t.Errorf("expected 1 file deleted, got %d", stats.FilesDeleted)
	}
	var n int
	if err := idx.store.DB.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 files after delete, got %d", n)
	}
}

// TestStatus reports file/chunk counts and last-indexed timestamp.
func TestStatus(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writeTestRepo(t)
	if _, err := idx.Run(context.Background(), root, testCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	status, err := GetStatus(idx.store)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.FileCount != 2 {
		t.Errorf("expected FileCount 2, got %d", status.FileCount)
	}
	if status.LastIndexed == "" {
		t.Errorf("expected LastIndexed to be set")
	}
}

// TestChunkLines splits content into overlapping windows.
func TestChunkLines(t *testing.T) {
	lines := make([]string, 0, 200)
	for i := 0; i < 200; i++ {
		lines = append(lines, "line-"+itoa(i))
	}
	content := joinLines(lines)
	chunks := ChunkLines([]byte(content), 80, 10)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
	// First chunk covers line 1..80.
	if chunks[0].StartLine != 1 || chunks[0].EndLine != 80 {
		t.Errorf("first chunk unexpected bounds: %d..%d", chunks[0].StartLine, chunks[0].EndLine)
	}
	// Chunks should have content_hash and text.
	for i, c := range chunks {
		if c.ContentHash == "" {
			t.Errorf("chunk %d missing content_hash", i)
		}
		if c.Text == "" {
			t.Errorf("chunk %d empty text", i)
		}
	}
}

// TestScanRespectsMaxFileSize skips files > 512KB.
func TestScanRespectsMaxFileSize(t *testing.T) {
	root := t.TempDir()
	small := filepath.Join(root, "small.go")
	mustWrite(t, small, "package main\n")
	big := filepath.Join(root, "big.go")
	bigContent := make([]byte, maxFileSize+1)
	for i := range bigContent {
		bigContent[i] = 'x'
	}
	if err := os.WriteFile(big, bigContent, 0o644); err != nil {
		t.Fatalf("write big: %v", err)
	}
	files, err := Scan(root, []string{"**/*.go"}, nil)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file (big.go skipped), got %d", len(files))
	}
	if files[0].Path != "small.go" {
		t.Errorf("unexpected path: %s", files[0].Path)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

func joinLines(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}
