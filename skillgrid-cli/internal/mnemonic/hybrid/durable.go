package hybrid

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/vectorstore"
)

// DurableEnabled reports whether the durable in-SQL vector path (ADR-0009,
// option G) is active for this process. It is opt-in (MNEMONIC_VECTOR_DB=1 /
// true / yes / on) so the default path stays the in-memory cosine cache (the
// hot path, ~40ms at 100K) and the durable path (~1.4s at 20K, ~6.9s at
// 100K) is only used when the caller explicitly wants the in-SQL vec tables.
// The parsing mirrors memory.EmbeddingEnabled.
func DurableEnabled() bool {
	switch os.Getenv("MNEMONIC_VECTOR_DB") {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// durableSymbolHits returns the top-K symbol ids via the in-SQL vec table
// (vectorstore.SearchSymbols) when the durable path is active AND the vec
// table is non-empty. It returns ok=true with the hits to use, or ok=false
// (the caller falls back to the in-memory path) when the flag is off, the vec
// table is absent, or the vec table is empty. The returned vectorHits carry
// the sym:<id> ID shape the in-memory leg produces so the caller's metadata
// join (rank.go) is shared between both paths.
func durableSymbolHits(ctx context.Context, db *sql.DB, qVec memory.Vector, limit int) ([]vectorHit, bool, error) {
	if !DurableEnabled() {
		return nil, false, nil
	}
	n, err := vectorstore.Count(ctx, db, "vec_symbols")
	if err != nil {
		return nil, false, fmt.Errorf("durable count vec_symbols: %w", err)
	}
	if n == 0 {
		// Empty-table degrade: no durable vectors yet, fall back to the
		// in-memory path (which may also be empty - same behavior as today).
		return nil, false, nil
	}
	ids, err := vectorstore.SearchSymbols(ctx, db, qVec.Data, limit)
	if err != nil {
		return nil, false, fmt.Errorf("durable search vec_symbols: %w", err)
	}
	hits := make([]vectorHit, 0, len(ids))
	for _, id := range ids {
		hits = append(hits, vectorHit{ID: fmt.Sprintf("sym:%d", id)})
	}
	return hits, true, nil
}

// durableChunkHits is the chunk analogue of durableSymbolHits (vec_chunks /
// SearchChunks / chunk:<id> IDs).
func durableChunkHits(ctx context.Context, db *sql.DB, qVec memory.Vector, limit int) ([]vectorHit, bool, error) {
	if !DurableEnabled() {
		return nil, false, nil
	}
	n, err := vectorstore.Count(ctx, db, "vec_chunks")
	if err != nil {
		return nil, false, fmt.Errorf("durable count vec_chunks: %w", err)
	}
	if n == 0 {
		return nil, false, nil
	}
	ids, err := vectorstore.SearchChunks(ctx, db, qVec.Data, limit)
	if err != nil {
		return nil, false, fmt.Errorf("durable search vec_chunks: %w", err)
	}
	hits := make([]vectorHit, 0, len(ids))
	for _, id := range ids {
		hits = append(hits, vectorHit{ID: fmt.Sprintf("chunk:%d", id)})
	}
	return hits, true, nil
}
