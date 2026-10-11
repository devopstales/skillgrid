package hybrid

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/vectorstore"
)

// vecEntry is one decoded vector ready for cosine scoring. A zero vector
// (cosine 0 with everything, no direction) is kept with degen=true so the
// degenerate-count warning still matches the pre-cache behavior.
type vecEntry struct {
	vec   memory.Vector
	degen bool
}

// vecCache is a per-store, model-scoped in-memory index of decoded vectors.
// The 55MB-of-BLOBs-per-query scan that dominated the semantic leg (measured:
// ~190ms of a ~220ms query) is paid once per (store, embedding-model) instead.
// The model in the key is the invalidation signal: a model swap re-embeds the
// table (see codeindex.embedPass) and therefore gets a fresh cache.
type vecCache struct {
	mu       sync.RWMutex
	cacheKey string
	symbols  map[int64]vecEntry
	chunks   map[int64]vecEntry
}

var vecCacheInstance = &vecCache{}

// InvalidateVectorCache clears the cache for a store+model pair. Exported so
// the indexer can drop the in-memory vectors after an embed pass, forcing the
// next semantic query to rebuild from the fresh table.
func InvalidateVectorCache(dbPath, model string) {
	vecCacheInstance.Invalidate(dbPath, model)
}

// ResetVectorCacheForTest clears the entire cache. Test-only: tests that seed
// embeddings via raw SQL between Search calls must call this so the next
// query sees the fresh rows.
func ResetVectorCacheForTest() {
	vecCacheInstance.Reset()
}

// buildKey namespaces the cache by store path and embedding model so two
// projects, or two models on the same store, never share entries.
func buildKey(dbPath, model string) string {
	return dbPath + "\x00" + model
}

func (c *vecCache) get(dbPath, model string) (map[int64]vecEntry, map[int64]vecEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.cacheKey != buildKey(dbPath, model) {
		return nil, nil, false
	}
	return c.symbols, c.chunks, true
}

func (c *vecCache) set(dbPath, model string, symbols, chunks map[int64]vecEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cacheKey = buildKey(dbPath, model)
	c.symbols = symbols
	c.chunks = chunks
}

// Reset drops the cache (called by the indexer after a re-embed so the next
// query rebuilds from the fresh vectors).
func (c *vecCache) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cacheKey = ""
	c.symbols = nil
	c.chunks = nil
}

// Invalidate clears the cache for the given store+model. Called by the
// indexer after an embed pass so the next semantic query sees fresh vectors
// without waiting for a model swap.
func (c *vecCache) Invalidate(dbPath, model string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cacheKey == buildKey(dbPath, model) {
		c.cacheKey = ""
		c.symbols = nil
		c.chunks = nil
	}
}

// HNSW symbol top-K via the embedded HNSW index (ADR-0037). Returns ok=true
// with the top-K hits when the HNSW graph is non-empty, or ok=false (the
// caller falls back to the vec0 path or in-memory cache). Any HNSW error
// degrades to ok=false — never a hard fail.
func hnswSymbolHits(ctx context.Context, db *sql.DB, qVec []float32, limit int) ([]vectorHit, bool, error) {
	hits, ok, err := vectorstore.HnswSymbols(ctx, db, qVec, limit)
	if err != nil || !ok {
		return nil, false, nil // table absent / empty graph — fall through
	}
	vh := make([]vectorHit, 0, len(hits))
	for _, h := range hits {
		vh = append(vh, vectorHit{ID: fmt.Sprintf("sym:%d", h.ID), Sim: h.Sim})
	}
	return vh, true, nil
}

// HNSW chunk top-K via the embedded HNSW index (ADR-0037). Same contract as
// hnswSymbolHits for the chunk path.
func hnswChunkHits(ctx context.Context, db *sql.DB, qVec []float32, limit int) ([]vectorHit, bool, error) {
	hits, ok, err := vectorstore.HnswChunks(ctx, db, qVec, limit)
	if err != nil || !ok {
		return nil, false, nil
	}
	vh := make([]vectorHit, 0, len(hits))
	for _, h := range hits {
		vh = append(vh, vectorHit{ID: fmt.Sprintf("chunk:%d", h.ID), Sim: h.Sim})
	}
	return vh, true, nil
}
