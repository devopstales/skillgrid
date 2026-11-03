package memory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// queryCacheTTL is how long a cached query embedding stays valid (7 days).
const queryCacheTTL = 168 * time.Hour

// CachedEmbedQuery returns a cached query embedding when one exists and its
// created_at is within 168 hours. The key is the hex SHA-256 of model, a
// newline, and the query. The query text is not written to the database.
// A miss, a stale row, or an unreadable blob calls inner, then stores the
// encoded vector with INSERT OR REPLACE.
func CachedEmbedQuery(ctx context.Context, db *sql.DB, model string, inner func(context.Context, string) (Vector, error), query string) (Vector, error) {
	if db == nil {
		return Vector{}, fmt.Errorf("query cache: nil db")
	}
	if inner == nil {
		return Vector{}, fmt.Errorf("query cache: nil embedder")
	}
	hash := queryCacheHash(model, query)
	var blob []byte
	var createdAt string
	err := db.QueryRowContext(ctx,
		`SELECT embedding, created_at FROM query_cache WHERE query_hash = ?`, hash,
	).Scan(&blob, &createdAt)
	if err == nil {
		if created, parseErr := time.Parse(time.RFC3339, createdAt); parseErr == nil && queryCacheFresh(created, time.Now().UTC()) {
			if v, decErr := DecodeVector(blob); decErr == nil {
				return v, nil
			}
		}
	} else if err != sql.ErrNoRows {
		return Vector{}, fmt.Errorf("query cache lookup: %w", err)
	}

	v, err := inner(ctx, query)
	if err != nil {
		return Vector{}, err
	}
	if _, err := db.ExecContext(ctx,
		`INSERT OR REPLACE INTO query_cache (query_hash, embedding, model, created_at) VALUES (?, ?, ?, ?)`,
		hash, EncodeVector(v), model, time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return Vector{}, fmt.Errorf("query cache store: %w", err)
	}
	return v, nil
}

func queryCacheHash(model, query string) string {
	sum := sha256.Sum256([]byte(model + "\n" + query))
	return hex.EncodeToString(sum[:])
}

func queryCacheFresh(created, now time.Time) bool {
	return !created.Before(now.Add(-queryCacheTTL))
}
