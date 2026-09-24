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
// the sym:<id> ID shape and the in-SQL Provenance.Sim (1 - cosine distance)
// so the durable leg is behaviorally identical to the in-memory leg; the
// caller's metadata join (rank.go) is shared between both paths. warn is the
// degenerate-embedding count (matching the in-memory leg's warning) computed
// from the BLOB source of truth (embeddings).
func durableSymbolHits(ctx context.Context, db *sql.DB, qVec memory.Vector, limit int) ([]vectorHit, string, bool, error) {
	if !DurableEnabled() {
		return nil, "", false, nil
	}
	n, err := vectorstore.Count(ctx, db, "vec_symbols")
	if err != nil {
		return nil, "", false, fmt.Errorf("durable count vec_symbols: %w", err)
	}
	if n == 0 {
		// Empty-table degrade: no durable vectors yet, fall back to the
		// in-memory path (which may also be empty - same behavior as today).
		return nil, "", false, nil
	}
	hits, err := vectorstore.SearchSymbols(ctx, db, qVec.Data, limit)
	if err != nil {
		return nil, "", false, fmt.Errorf("durable search vec_symbols: %w", err)
	}
	vh := make([]vectorHit, 0, len(hits))
	for _, h := range hits {
		vh = append(vh, vectorHit{ID: fmt.Sprintf("sym:%d", h.ID), Sim: h.Sim})
	}
	warn, err := degenerateCount(ctx, db, "SELECT symbol_id, vector FROM embeddings")
	if err != nil {
		return nil, "", false, err
	}
	return vh, warn, true, nil
}

// durableChunkHits is the chunk analogue of durableSymbolHits (vec_chunks /
// SearchChunks / chunk:<id> IDs, BLOB source chunk_embeddings).
func durableChunkHits(ctx context.Context, db *sql.DB, qVec memory.Vector, limit int) ([]vectorHit, string, bool, error) {
	if !DurableEnabled() {
		return nil, "", false, nil
	}
	n, err := vectorstore.Count(ctx, db, "vec_chunks")
	if err != nil {
		return nil, "", false, fmt.Errorf("durable count vec_chunks: %w", err)
	}
	if n == 0 {
		return nil, "", false, nil
	}
	hits, err := vectorstore.SearchChunks(ctx, db, qVec.Data, limit)
	if err != nil {
		return nil, "", false, fmt.Errorf("durable search vec_chunks: %w", err)
	}
	vh := make([]vectorHit, 0, len(hits))
	for _, h := range hits {
		vh = append(vh, vectorHit{ID: fmt.Sprintf("chunk:%d", h.ID), Sim: h.Sim})
	}
	warn, err := degenerateCount(ctx, db, "SELECT chunk_id, vector FROM chunk_embeddings")
	if err != nil {
		return nil, "", false, err
	}
	return vh, warn, true, nil
}

// degenerateCount scans the BLOB embedding table (the source of truth) and
// returns the "%d degenerate embeddings skipped" warning when any row is
// degenerate (fails to decode, is empty, or is a zero vector — the same
// definition loadVectorCache uses for the in-memory leg's degen flag). The
// durable leg must surface the identical warning so the two legs are
// behaviorally identical for the same store.
func degenerateCount(ctx context.Context, db *sql.DB, query string) (string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("durable degenerate scan: %w", err)
	}
	defer rows.Close()
	degenerate := 0
	for rows.Next() {
		var id int64
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			continue
		}
		vec, derr := memory.DecodeVector(blob)
		if derr != nil || len(vec.Data) == 0 || memory.CosineSimilarity(vec, vec) == 0 {
			degenerate++
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("durable degenerate rows: %w", err)
	}
	if degenerate == 0 {
		return "", nil
	}
	return fmt.Sprintf("%d degenerate embeddings skipped", degenerate), nil
}
