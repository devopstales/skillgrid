package codeindex

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ErrSymbolEmbeddingMissing is returned by GetSymbolEmbedding when the symbol
// has no embedding row in the bridge table.
var ErrSymbolEmbeddingMissing = errors.New("no embedding for symbol")

// The symbol_embeddings bridge links codeindex symbols to embedding vectors.
// The `embeddings` table (migration 011) already provides exactly this
// linkage and lives in the SAME per-project SQLite store as observations, so
// it is reused as the bridge rather than creating a duplicate table:
//
//	symbol_id INTEGER PRIMARY KEY REFERENCES symbols(id) ON DELETE CASCADE
//	model     TEXT NOT NULL
//	dim       INTEGER NOT NULL
//	vector    BLOB NOT NULL   (little-endian float32)
//	updated_at TEXT NOT NULL
//
// These are the CRUD accessors the memory layer (change 014 step 07) uses to
// read symbol embeddings across the triple-store cross-link. They are
// store-agnostic (*sql.DB) so the memory package can call them without
// importing codeindex (which imports memory — import cycle otherwise).

// UpsertSymbolEmbedding writes or replaces the embedding row for symbolID.
// vector is the raw little-endian float32 payload; dim is len(vector)/4.
func UpsertSymbolEmbedding(ctx context.Context, db *sql.DB, symbolID int64, vector []byte, model string) error {
	if len(vector) == 0 {
		return errors.New("vector is empty")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.ExecContext(ctx, `
		INSERT INTO embeddings (symbol_id, model, dim, vector, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(symbol_id) DO UPDATE SET
		  model = excluded.model,
		  dim = excluded.dim,
		  vector = excluded.vector,
		  updated_at = excluded.updated_at`,
		symbolID, model, len(vector)/4, vector, now,
	)
	if err != nil {
		return err
	}
	return nil
}

// GetSymbolEmbedding returns the embedding vector and model for symbolID, or
// ErrSymbolEmbeddingMissing when the symbol has no embedding row.
func GetSymbolEmbedding(ctx context.Context, db *sql.DB, symbolID int64) ([]byte, string, error) {
	var blob []byte
	var model string
	err := db.QueryRowContext(ctx, `
		SELECT vector, model FROM embeddings WHERE symbol_id = ?`, symbolID,
	).Scan(&blob, &model)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrSymbolEmbeddingMissing
	}
	if err != nil {
		return nil, "", err
	}
	return blob, model, nil
}

// DeleteSymbolEmbedding removes the embedding row for symbolID (a no-op when
// absent). Used when a symbol is re-extracted without an embedder.
func DeleteSymbolEmbedding(ctx context.Context, db *sql.DB, symbolID int64) error {
	_, err := db.ExecContext(ctx, `DELETE FROM embeddings WHERE symbol_id = ?`, symbolID)
	return err
}
