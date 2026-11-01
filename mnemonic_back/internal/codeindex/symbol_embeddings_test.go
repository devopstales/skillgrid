package codeindex

import (
	"context"
	"database/sql"
	"testing"
)

// seedBridgeSymbol inserts a minimal file+symbol row and returns the symbol id,
// so the bridge tests do not depend on running the full indexer.
func seedBridgeSymbol(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var fileID int64
	if err := db.QueryRow(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('/tmp/bridge.go', 1, 10, 'h1', '2026-01-01T00:00:00Z')
		RETURNING id`).Scan(&fileID); err != nil {
		t.Fatalf("insert file: %v", err)
	}
	var symID int64
	if err := db.QueryRow(`
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		VALUES (?, 'bridgeFunc', 'bridgeFunc', 'function', 'go', 'func bridgeFunc()', 1, 2, 'ch1', 'uid-bridge-1')
		RETURNING id`, fileID).Scan(&symID); err != nil {
		t.Fatalf("insert symbol: %v", err)
	}
	return symID
}

func embeddingColumns(t *testing.T, db *sql.DB) map[string]bool {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM pragma_table_info('embeddings')`)
	if err != nil {
		t.Fatalf("table_info: %v", err)
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table_info: %v", err)
		}
		cols[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return cols
}

// TestSymbolEmbeddingsBridge covers @step-07: the bridge table links codeindex
// symbols to embedding vectors. The `embeddings` table (migration 011) lives
// in the same per-project store as observations, so it is reused as the
// bridge; these accessors are its CRUD surface for the memory layer.
func TestSymbolEmbeddingsBridge(t *testing.T) {
	st, _, clean := openStoreFor(t)
	defer clean()
	db := st.DB
	ctx := context.Background()

	// The bridge table must exist with the expected columns.
	if !tableExists(t, db, "embeddings") {
		t.Fatalf("expected table embeddings (reused as the symbol_embeddings bridge)")
	}
	if cols := embeddingColumns(t, db); !cols["symbol_id"] || !cols["model"] || !cols["vector"] {
		t.Fatalf("embeddings columns missing: symbol_id/model/vector (got %v)", cols)
	}

	symID := seedBridgeSymbol(t, db)

	// Insert a bridge row linking the symbol to an embedding vector.
	if err := UpsertSymbolEmbedding(ctx, db, symID, []byte{0x01, 0x02, 0x03, 0x04}, "test-model"); err != nil {
		t.Fatalf("upsert bridge: %v", err)
	}

	// Query the bridge: the symbol id must map to the correct embedding.
	blob, model, err := GetSymbolEmbedding(ctx, db, symID)
	if err != nil {
		t.Fatalf("get bridge: %v", err)
	}
	if len(blob) != 4 || blob[0] != 1 || blob[3] != 4 {
		t.Fatalf("bridge blob=%v want [1 2 3 4]", blob)
	}
	if model != "test-model" {
		t.Fatalf("bridge model=%q want test-model", model)
	}

	// Re-embedding replaces the row (primary key symbol_id).
	if err := UpsertSymbolEmbedding(ctx, db, symID, []byte{0x0a, 0x0b}, "v2-model"); err != nil {
		t.Fatalf("re-upsert bridge: %v", err)
	}
	blob, model, err = GetSymbolEmbedding(ctx, db, symID)
	if err != nil {
		t.Fatalf("re-get bridge: %v", err)
	}
	if len(blob) != 2 || model != "v2-model" {
		t.Fatalf("after re-embed: blob=%v model=%q want [10 11] v2-model", blob, model)
	}

	// A symbol without an embedding yields ErrSymbolEmbeddingMissing.
	if _, _, err := GetSymbolEmbedding(ctx, db, 999999); err == nil || err != ErrSymbolEmbeddingMissing {
		t.Fatalf("missing embedding err=%v want ErrSymbolEmbeddingMissing", err)
	}
}
