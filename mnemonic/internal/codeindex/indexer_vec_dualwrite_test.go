package codeindex

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/embedder"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/vectorstore"
)

// TestEmbedPassDualWritesVecTables pins that after an embed pass with a
// 768-dim embedder (the production onnx default, matching the vec tables'
// pinned float[768]), the vec tables mirror the BLOB tables: same row count.
// The dual-write is dimension-gated: a non-768 embedder leaves the vec tables
// empty (the BLOB tables remain the source of truth for any dimension).
func TestEmbedPassDualWritesVecTables(t *testing.T) {
	st, _, clean := openStoreFor(t)
	resetFileFirstSymbol()
	// 768-dim hash embedder matches the vec tables' pinned dimension.
	idx := New(st).WithEmbedder(embedder.NewHash(768))
	defer clean()
	root := writeEmbedFixture(t)

	if _, err := idx.Run(context.Background(), root, fixtureCfg); err != nil {
		t.Fatalf("run: %v", err)
	}

	ctx := context.Background()
	blobSyms, err := countRow(t, st, "embeddings")
	if err != nil {
		t.Fatalf("count embeddings: %v", err)
	}
	vecSyms, err := vectorstore.Count(ctx, st.DB, "vec_symbols")
	if err != nil {
		t.Fatalf("count vec_symbols: %v", err)
	}
	if blobSyms == 0 {
		t.Fatalf("expected symbol-level embeddings, got 0 rows in embeddings")
	}
	if blobSyms != vecSyms {
		t.Fatalf("vec_symbols count=%d, embeddings count=%d (dual-write mismatch)", vecSyms, blobSyms)
	}

	blobChunks, err := countRow(t, st, "chunk_embeddings")
	if err != nil {
		t.Fatalf("count chunk_embeddings: %v", err)
	}
	vecChunks, err := vectorstore.Count(ctx, st.DB, "vec_chunks")
	if err != nil {
		t.Fatalf("count vec_chunks: %v", err)
	}
	if blobChunks != vecChunks {
		t.Fatalf("vec_chunks count=%d, chunk_embeddings count=%d (dual-write mismatch)", vecChunks, blobChunks)
	}
}

// TestEmbedPassSkipsVecTablesOnDimMismatch pins the dimension guard: a 64-dim
// embedder writes to the BLOB tables (source of truth, dimension-agnostic) but
// leaves the vec tables empty (they are pinned to float[768]).
func TestEmbedPassSkipsVecTablesOnDimMismatch(t *testing.T) {
	st, _, clean := openStoreFor(t)
	resetFileFirstSymbol()
	idx := New(st).WithEmbedder(embedder.NewHash(64)) // 64-dim != vec dim 768
	defer clean()
	root := writeEmbedFixture(t)

	if _, err := idx.Run(context.Background(), root, fixtureCfg); err != nil {
		t.Fatalf("run: %v", err)
	}

	ctx := context.Background()
	blobSyms, err := countRow(t, st, "embeddings")
	if err != nil {
		t.Fatalf("count embeddings: %v", err)
	}
	if blobSyms == 0 {
		t.Fatalf("expected symbol-level embeddings, got 0 rows in embeddings")
	}
	vecSyms, err := vectorstore.Count(ctx, st.DB, "vec_symbols")
	if err != nil {
		t.Fatalf("count vec_symbols: %v", err)
	}
	if vecSyms != 0 {
		t.Fatalf("vec_symbols count=%d with a 64-dim embedder, want 0 (dimension guard)", vecSyms)
	}
}

func countRow(t *testing.T, st *store.Store, table string) (int, error) {
	t.Helper()
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
