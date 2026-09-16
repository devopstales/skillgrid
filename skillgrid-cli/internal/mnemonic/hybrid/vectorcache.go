package hybrid

import (
	"sync"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
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
