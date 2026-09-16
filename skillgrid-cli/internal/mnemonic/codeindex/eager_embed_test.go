package codeindex

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
)

// resetFileFirstSymbol clears the package-level cache. The cache is a
// process-global keyed by file id (not by store), so a prior test in the same
// process can leave stale entries that collide with this test's ids.
func resetFileFirstSymbol() {
	for k := range fileFirstSymbol {
		delete(fileFirstSymbol, k)
	}
}

func newTestIndexerWithEmbedder(t *testing.T) (*Indexer, func()) {
	t.Helper()
	st, _, clean := openStoreFor(t)
	resetFileFirstSymbol()
	return New(st).WithEmbedder(embedder.NewHash(64)), clean
}

func writeEmbedFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t,
		root+"/main.go",
		"package main\n\nimport \"fmt\"\n\nfunc greet(name string) string {\n\treturn \"hello, \" + name\n}\n\nfunc main() {\n\tfmt.Println(greet(\"world\"))\n}\n",
	)
	return root
}

func countEmbeddings(t *testing.T, idx *Indexer) int {
	t.Helper()
	var n int
	if err := idx.store.DB.QueryRow(`SELECT COUNT(*) FROM embeddings`).Scan(&n); err != nil {
		t.Fatalf("count embeddings: %v", err)
	}
	return n
}

func embeddedSymbolNames(t *testing.T, idx *Indexer) map[string]bool {
	t.Helper()
	rows, err := idx.store.DB.Query(`
		SELECT s.name FROM embeddings v
		JOIN symbols s ON s.id = v.symbol_id`)
	if err != nil {
		t.Fatalf("query embedded symbols: %v", err)
	}
	defer rows.Close()
	names := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan name: %v", err)
		}
		names[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return names
}

func readEmbedModel(t *testing.T, idx *Indexer) string {
	t.Helper()
	var model string
	err := idx.store.DB.QueryRow(`SELECT value FROM embed_meta WHERE key = 'embedding_model'`).Scan(&model)
	if err != nil {
		t.Fatalf("read embedding_model: %v", err)
	}
	return model
}

func setEmbedModel(t *testing.T, idx *Indexer, model string) {
	t.Helper()
	if _, err := idx.store.DB.Exec(`
		INSERT OR REPLACE INTO embed_meta (key, value) VALUES ('embedding_model', ?)`, model); err != nil {
		t.Fatalf("set embedding_model: %v", err)
	}
}

// TestEagerEmbedSymbolLevel covers (04.4): the eager dual-granularity pass
// embeds symbol-level vectors (name + signature) into the embeddings table,
// records the model in embed_meta, and stays idempotent on a re-run.
func TestEagerEmbedSymbolLevel(t *testing.T) {
	idx, clean := newTestIndexerWithEmbedder(t)
	defer clean()
	root := writeEmbedFixture(t)

	if _, err := idx.Run(context.Background(), root, fixtureCfg); err != nil {
		t.Fatalf("first run: %v", err)
	}

	// The embeddings table must hold rows for the indexed symbols.
	n := countEmbeddings(t, idx)
	if n == 0 {
		t.Fatalf("expected symbol-level embeddings, got 0 rows in embeddings")
	}

	// Both functions must be embedded.
	names := embeddedSymbolNames(t, idx)
	for _, want := range []string{"main", "greet"} {
		if !names[want] {
			t.Errorf("expected symbol %q to be embedded; got %v", want, names)
		}
	}

	// embed_meta must record the embedder model.
	if got := readEmbedModel(t, idx); got != "hash-embedder-v1" {
		t.Errorf("expected embedding_model = hash-embedder-v1, got %q", got)
	}

	// Re-run the same files: the file is unchanged (content-hash + mtime
	// guard), so no new embeddings must be added.
	if _, err := idx.Run(context.Background(), root, fixtureCfg); err != nil {
		t.Fatalf("second run: %v", err)
	}
	n2 := countEmbeddings(t, idx)
	if n2 != n {
		t.Errorf("re-run must be idempotent: first run had %d embeddings, second run has %d", n, n2)
	}
}

// TestEagerEmbedModelSwap covers (04.4): when the indexed model in embed_meta
// differs from the active embedder's model, the pass clears the stale
// embeddings and re-embeds them. HashEmbedder.Model() is constant, so the
// swap is simulated by rewriting embed_meta to a different model name before
// the second run.
func TestEagerEmbedModelSwap(t *testing.T) {
	idx, clean := newTestIndexerWithEmbedder(t)
	defer clean()
	root := writeEmbedFixture(t)

	// First run with HashEmbedder(64): embeddings exist under hash-embedder-v1.
	if _, err := idx.Run(context.Background(), root, fixtureCfg); err != nil {
		t.Fatalf("first run: %v", err)
	}
	before := countEmbeddings(t, idx)
	if before == 0 {
		t.Fatalf("expected embeddings after first run, got 0")
	}
	if got := readEmbedModel(t, idx); got != "hash-embedder-v1" {
		t.Fatalf("expected model hash-embedder-v1, got %q", got)
	}

	// Simulate a model swap: the indexed model now claims a different name.
	setEmbedModel(t, idx, "old-model")
	if got := readEmbedModel(t, idx); got != "old-model" {
		t.Fatalf("expected embed_meta model old-model, got %q", got)
	}

	// Second run with the same HashEmbedder: indexedModel ("old-model") !=
	// model ("hash-embedder-v1"), so the pass must clear and re-embed.
	if _, err := idx.Run(context.Background(), root, fixtureCfg); err != nil {
		t.Fatalf("second run: %v", err)
	}

	// The model is restored to the active embedder's name.
	if got := readEmbedModel(t, idx); got != "hash-embedder-v1" {
		t.Errorf("expected model hash-embedder-v1 after re-embed, got %q", got)
	}

	// No stale rows may survive: every embedding row must carry the current
	// model, and the set must be fully re-populated.
	var stale int
	if err := idx.store.DB.QueryRow(`SELECT COUNT(*) FROM embeddings WHERE model != 'hash-embedder-v1'`).Scan(&stale); err != nil {
		t.Fatalf("count stale: %v", err)
	}
	if stale != 0 {
		t.Errorf("expected 0 stale embeddings after model swap, got %d", stale)
	}
	after := countEmbeddings(t, idx)
	if after != before {
		t.Errorf("expected %d embeddings after re-embed, got %d", before, after)
	}
	names := embeddedSymbolNames(t, idx)
	for _, want := range []string{"main", "greet"} {
		if !names[want] {
			t.Errorf("expected symbol %q re-embedded after swap; got %v", want, names)
		}
	}
}

// TestEagerEmbedChunkOverlap covers (04.4): ChunkLines produces overlapping
// windows — each subsequent chunk starts exactly chunkOverlap lines before
// the previous chunk ends.
func TestEagerEmbedChunkOverlap(t *testing.T) {
	var sb strings.Builder
	for i := 1; i <= 200; i++ {
		fmt.Fprintf(&sb, "line%03d = %d\n", i, i)
	}
	content := []byte(sb.String())

	chunks := ChunkLines(content, 80, 10)
	if len(chunks) == 0 {
		t.Fatalf("expected chunks for 200 lines, got 0")
	}

	// Consecutive chunks must overlap by exactly ChunkOverlap lines of
	// content: the window advances by (chunkLines - overlap) = 70, so the next
	// chunk starts 70 lines after the previous chunk starts — i.e. 10 lines
	// before the previous chunk's EndLine (the overlap).
	for i := 1; i < len(chunks); i++ {
		prev := chunks[i-1]
		cur := chunks[i]
		wantStart := prev.StartLine + 70
		if cur.StartLine != wantStart {
			t.Errorf("chunk %d: StartLine = %d, want %d (previous StartLine %d plus step 70)",
				i, cur.StartLine, wantStart, prev.StartLine)
		}
		// The overlap means the next chunk begins 10 lines before the previous
		// chunk's EndLine (EndLine is exclusive of the trailing empty entry,
		// so content overlap is prev.EndLine - cur.StartLine + 1 = 11 lines
		// including both boundaries; the window step is 70 = 80 - 10).
		if cur.StartLine >= prev.EndLine {
			t.Errorf("chunk %d does not overlap previous: StartLine %d >= prev EndLine %d",
				i, cur.StartLine, prev.EndLine)
		}
	}

	// The window advances by (chunkLines - overlap) = 70 lines per step, so
	// 200 lines produce ceil((200-80)/70)+1 = 3 chunks.
	if len(chunks) != 3 {
		t.Errorf("expected 3 chunks for 200 lines with step 70, got %d", len(chunks))
	}

	// The last chunk must reach the end of the file (201 entries after split:
	// 200 content lines + 1 trailing empty entry from the final newline).
	if last := chunks[len(chunks)-1]; last.EndLine != 201 {
		t.Errorf("expected last chunk to end at entry 201, got %d", last.EndLine)
	}

	// The first chunk starts at line 1 and spans 80 lines.
	if chunks[0].StartLine != 1 || chunks[0].EndLine != 80 {
		t.Errorf("expected first chunk to span 1..80, got %d..%d", chunks[0].StartLine, chunks[0].EndLine)
	}
}
