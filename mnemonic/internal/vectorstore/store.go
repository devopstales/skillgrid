// Package vectorstore is the durable in-SQL vector path (ADR-0009, option G).
// It wraps the vec0 virtual tables (vec_symbols, vec_chunks) created by
// migration 042, providing top-K cosine search plus the upsert/delete that
// the indexer's dual-write uses to keep the vec tables in sync with the BLOB
// embedding tables (which remain the source of truth).
//
// The blank import of modernc.org/sqlite/vec registers the vec0 vtab +
// vec_distance_cosine + vec_f32 functions with the modernc driver. Any caller
// that opens a store and calls this package gets the vtab for free.
package vectorstore

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"sync"

	_ "modernc.org/sqlite/vec"
)

// Dim is the pinned embedding dimension (DefaultOnnxDim). The vec0 tables are
// created with float[768]; a vector of a different dimension is rejected by
// the vec_f32 constructor (dimension-strict). The indexer's dual-write mirrors
// a vector into the vec tables only when the embedder's dimension matches Dim.
const Dim = 768

// TableExists reports whether the named vec0 table is present in the store.
func TableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).
		Scan(&n)
	if err != nil {
		return false, fmt.Errorf("table exists %s: %w", table, err)
	}
	return n > 0, nil
}

// Count returns the number of rows in the named vec0 table.
func Count(ctx context.Context, db *sql.DB, table string) (int, error) {
	var n int
	err := db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %s`, table)).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count %s: %w", table, err)
	}
	return n, nil
}

// CountSource returns the number of HNSW graph nodes in one source partition
// (migration 058: the hnsw_nodes pair is keyed (source, node_id)). symbol_id
// and chunk_id are two independent autoincrement spaces (both start at 1), so
// each leg must count only its own partition — a combined count cannot
// distinguish "symbol graph empty" from "chunk graph empty" (L-1, TICKET-06).
func CountSource(ctx context.Context, db *sql.DB, source string) (int, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM hnsw_nodes WHERE source = ?`, source).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count hnsw source %s: %w", source, err)
	}
	return n, nil
}

// UpsertSymbol writes one symbol vector into vec_symbols (rowid = symbolID).
// vecBytes is the little-endian float32 blob (memory.EncodeVector layout).
func UpsertSymbol(ctx context.Context, tx *sql.Tx, symbolID int64, model string, _ int, vecBytes []byte) error {
	return upsert(ctx, tx, "vec_symbols", symbolID, vecBytes)
}

// UpsertChunk writes one chunk vector into vec_chunks (rowid = chunkID).
func UpsertChunk(ctx context.Context, tx *sql.Tx, chunkID int64, model string, _ int, vecBytes []byte) error {
	return upsert(ctx, tx, "vec_chunks", chunkID, vecBytes)
}

func upsert(ctx context.Context, tx *sql.Tx, table string, id int64, vecBytes []byte) error {
	// vec0 virtual tables do not implement UPSERT (ON CONFLICT ... DO UPDATE
	// is a no-op / error on vtabs). A plain INSERT is enough: the indexer's
	// embedPass clears the vec tables on model swap (DeleteSymbols/Chunks)
	// before re-embedding, so a rowid collision only happens on a fresh
	// index or an in-place re-embed of an already-cleared table. For the
	// incremental path (a single symbol re-embedded under the same model) the
	// BLOB table's ON CONFLICT(symbol_id) DO UPDATE handles the overwrite;
	// the vec table mirrors via a DELETE + INSERT in the same transaction.
	if _, err := tx.ExecContext(ctx,
		fmt.Sprintf(`DELETE FROM %s WHERE rowid=?`, table), id); err != nil {
		return fmt.Errorf("delete %s %d: %w", table, id, err)
	}
	if _, err := tx.ExecContext(ctx,
		fmt.Sprintf(`INSERT INTO %s(rowid, embedding) VALUES (?, vec_f32(?))`, table),
		id, vecBytes); err != nil {
		return fmt.Errorf("insert %s %d: %w", table, id, err)
	}
	return nil
}

// DeleteSymbols clears vec_symbols (model-swap path).
func DeleteSymbols(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM vec_symbols`); err != nil {
		return fmt.Errorf("delete vec_symbols: %w", err)
	}
	return nil
}

// DeleteChunks clears vec_chunks (model-swap path).
func DeleteChunks(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM vec_chunks`); err != nil {
		return fmt.Errorf("delete vec_chunks: %w", err)
	}
	return nil
}

// hnswCache keeps one warmed HnswIndex per (db, source) partition so a
// durable hybrid query does not bulk-reload the whole graph from SQLite on
// every call (TICKET-06, A-4). The key embeds the *sql.DB pointer: every
// store opens its own DB, and tests that seed raw SQL rows directly (bypassing
// the cache) reset it before re-querying.
var hnswCache = struct {
	sync.Mutex
	cur  *sql.DB
	syms *HnswIndex
	chns *HnswIndex
}{}

// GetHnswSymbolIndex returns the cached symbol HNSW index (partition "symbol"
// over the embeddings BLOB table), building it on first use.
func GetHnswSymbolIndex(db *sql.DB) *HnswIndex {
	hnswCache.Lock()
	defer hnswCache.Unlock()
	if hnswCache.cur != db {
		hnswCache.cur = db
		hnswCache.syms = NewHnswSource(db, "embeddings", "symbol_id", "symbol")
		hnswCache.chns = NewHnswSource(db, "chunk_embeddings", "chunk_id", "chunk")
	}
	return hnswCache.syms
}

// GetHnswChunkIndex returns the cached chunk HNSW index (partition "chunk"
// over the chunk_embeddings BLOB table), building it on first use.
func GetHnswChunkIndex(db *sql.DB) *HnswIndex {
	hnswCache.Lock()
	defer hnswCache.Unlock()
	if hnswCache.cur != db {
		hnswCache.cur = db
		hnswCache.syms = NewHnswSource(db, "embeddings", "symbol_id", "symbol")
		hnswCache.chns = NewHnswSource(db, "chunk_embeddings", "chunk_id", "chunk")
	}
	return hnswCache.chns
}

// InvalidateHnswCache drops the cached HNSW indexes when the store handle
// changes, forcing a fresh warm-load from SQLite.
func InvalidateHnswCache() {
	hnswCache.Lock()
	defer hnswCache.Unlock()
	hnswCache.cur = nil
	hnswCache.syms = nil
	hnswCache.chns = nil
}

// ResetHnswCacheForTest clears the cache entirely. Test-only: tests that seed
// graph rows via raw SQL between Search calls must call this so the next
// query sees the fresh rows (mirrors hybrid.ResetVectorCacheForTest).
func ResetHnswCacheForTest() { InvalidateHnswCache() }

// SearchHit is one top-K result: the rowid and its cosine distance from the
// query vector (vec_distance_cosine, in [0, 2]; similarity = 1 - distance).
type SearchHit struct {
	ID   int64
	Dist float64
	Sim  float64
}

// SearchSymbols returns the top-K symbol_id by ascending vec_distance_cosine
// (most-similar first) for the query vector, with the in-SQL cosine distance
// (and derived similarity) so the durable leg can report Provenance.Sim.
func SearchSymbols(ctx context.Context, db *sql.DB, queryVec []float32, limit int) ([]SearchHit, error) {
	return search(ctx, db, "vec_symbols", queryVec, limit)
}

// SearchChunks returns the top-K chunk_id by ascending vec_distance_cosine,
// with the in-SQL cosine distance and derived similarity.
func SearchChunks(ctx context.Context, db *sql.DB, queryVec []float32, limit int) ([]SearchHit, error) {
	return search(ctx, db, "vec_chunks", queryVec, limit)
}

func search(ctx context.Context, db *sql.DB, table string, queryVec []float32, limit int) ([]SearchHit, error) {
	if limit <= 0 {
		limit = 10
	}
	qb := encodeF32(queryVec)
	rows, err := db.QueryContext(ctx,
		fmt.Sprintf(`SELECT rowid, vec_distance_cosine(embedding, vec_f32(?)) FROM %s
		 ORDER BY vec_distance_cosine(embedding, vec_f32(?)) LIMIT ?`, table),
		qb, qb, limit)
	if err != nil {
		return nil, fmt.Errorf("search %s: %w", table, err)
	}
	defer rows.Close()
	hits := make([]SearchHit, 0, limit)
	for rows.Next() {
		var id int64
		var dist float64
		if err := rows.Scan(&id, &dist); err != nil {
			return nil, fmt.Errorf("scan %s row: %w", table, err)
		}
		hits = append(hits, SearchHit{ID: id, Dist: dist, Sim: 1 - dist})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows %s: %w", table, err)
	}
	return hits, nil
}

// HnswSymbols returns the top-K symbol_id by cosine similarity served by the
// embedded HNSW index (ADR-0037). It reports ok=false (the caller falls
// through to the legacy path) when the store predates migration 058 or the
// symbol graph is empty; any other HNSW error degrades to the exact in-SQL
// vec0 scan (brute force), never a hard fail.
func HnswSymbols(ctx context.Context, db *sql.DB, queryVec []float32, limit int) ([]SearchHit, bool, error) {
	n, err := CountSource(ctx, db, "symbol")
	if err != nil {
		return nil, false, nil // table absent (pre-058 store) — fall through
	}
	if n == 0 {
		return nil, false, nil // empty symbol graph — fall through
	}
	idx := GetHnswSymbolIndex(db)
	hits, err := idx.Search(ctx, queryVec, limit)
	if err != nil {
		fb, ferr := SearchSymbols(ctx, db, queryVec, limit) // corrupt graph — exact brute force
		if ferr != nil {
			return nil, false, ferr
		}
		return fb, true, nil
	}
	return hits, true, nil
}

// HnswChunks returns the top-K chunk_id by cosine similarity served by the
// embedded HNSW index (ADR-0037). Same contract as HnswSymbols for the chunk
// path.
func HnswChunks(ctx context.Context, db *sql.DB, queryVec []float32, limit int) ([]SearchHit, bool, error) {
	n, err := CountSource(ctx, db, "chunk")
	if err != nil {
		return nil, false, nil
	}
	if n == 0 {
		return nil, false, nil
	}
	idx := GetHnswChunkIndex(db)
	hits, err := idx.Search(ctx, queryVec, limit)
	if err != nil {
		fb, ferr := SearchChunks(ctx, db, queryVec, limit)
		if ferr != nil {
			return nil, false, ferr
		}
		return fb, true, nil
	}
	return hits, true, nil
}

// encodeF32 is the package-local little-endian float32 encoder (same layout as
// memory.EncodeVector). Kept here so vectorstore has no dependency on the
// memory package; the production caller passes memory.EncodeVector output.
func encodeF32(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}
