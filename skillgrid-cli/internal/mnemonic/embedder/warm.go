package embedder

import (
	"context"
	"sync"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// DefaultIdleTimeout is how long an unused warm embedder stays resident before
// it is evicted from RAM (10min; tunable). While an MCP client connection is
// active, a heartbeat keeps it alive (no eviction while connected).
const DefaultIdleTimeout = 10 * time.Minute

// WarmConfig configures a warm embedder.
type WarmConfig struct {
	// Factory builds the underlying embedder when it needs (re)loading.
	Factory func() Embedder
	// IdleTimeout is the idle eviction window (<=0 uses DefaultIdleTimeout).
	IdleTimeout time.Duration
	// Now is the clock (injectable for tests).
	Now func() time.Time
	// EvictNotify is called when the embedder is evicted (test/observability).
	EvictNotify func()
}

// WarmEmbedder is a load-once, idle-evictable, heartbeat-kept warm cache around
// an Embedder. The expensive model load happens once; subsequent search calls
// reuse the cached embedder (no per-call model load). After IdleTimeout of no
// use the embedder is evicted from RAM; an active MCP heartbeat suppresses
// eviction while a client is connected. An evicted-then-reused embedder reloads
// transparently on the next use.
//
// "off"/external providers hold no RAM: when the underlying embedder reports
// Dimension()==0 (the Null Adapter) the warm cache does no caching and no
// eviction, and never loads a model.
type WarmEmbedder struct {
	cfg  WarmConfig
	mu   sync.Mutex
	emb  Embedder
	last time.Time
}

// NewWarm returns a warm embedder. A nil cfg.Factory means "absent" (the warm
// cache degrades to a Null Embedder → the FTS + signals floor).
func NewWarm(cfg WarmConfig) *WarmEmbedder {
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = DefaultIdleTimeout
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &WarmEmbedder{cfg: cfg, last: cfg.Now()}
}

// Get returns the live embedder, loading it on first use (and after an idle
// eviction). It records the access time (which resets the idle-evict window)
// and is heartbeat-safe: a connected MCP client calls this per tool call, so
// the embedder is never evicted while the session is active.
//
// A nil/absent factory (or a factory error) degrades to the Null Adapter so the
// pipeline never hard-fails: the caller falls back to the FTS + signals floor
// and no model-load error is surfaced.
func (w *WarmEmbedder) Get(ctx context.Context) Embedder {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.last = w.cfg.Now()
	if w.emb != nil {
		return w.emb
	}
	if w.cfg.Factory == nil {
		// Absent model: degrade to FTS (Null Adapter), no load, no error.
		w.emb = NewNull()
		return w.emb
	}
	e := w.cfg.Factory()
	if e == nil {
		w.emb = NewNull()
		return w.emb
	}
	w.emb = e
	return w.emb
}

// Evict evicts the embedder from RAM (freeing the model). A nil factory then
// means the next Get reloads transparently. Evicting a Null Adapter (no RAM)
// is a no-op.
func (w *WarmEmbedder) Evict() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.emb != nil && w.emb.Dimension() != 0 {
		w.emb = nil
	}
	if w.cfg.EvictNotify != nil {
		w.cfg.EvictNotify()
	}
}

// CheckIdle evicts the embedder if it has been idle for longer than
// IdleTimeout and there is no active heartbeat (connectedClients == 0). This
// is the periodic eviction sweep (run by the idle-evict timer). An evicted
// embedder reloads transparently on the next Get.
func (w *WarmEmbedder) CheckIdle(now time.Time, connectedClients int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if connectedClients > 0 {
		// Heartbeat: a connected MCP client keeps the embedder alive.
		return
	}
	if w.emb == nil || w.emb.Dimension() == 0 {
		// Nothing resident (or a Null Adapter holding no RAM): nothing to evict.
		return
	}
	if now.Sub(w.last) >= w.cfg.IdleTimeout {
		w.emb = nil
		if w.cfg.EvictNotify != nil {
			w.cfg.EvictNotify()
		}
	}
}

// IsLoaded reports whether a (non-Null) embedder is currently resident.
func (w *WarmEmbedder) IsLoaded() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.emb != nil && w.emb.Dimension() != 0
}

// Model returns the current embedder's model name ("" when not loaded).
func (w *WarmEmbedder) Model() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.emb == nil {
		return ""
	}
	return w.emb.Model()
}

// lastAccess returns the last access time (test hook).
func (w *WarmEmbedder) lastAccess() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.last
}

// Embed satisfies the Embedder interface by delegating to Get(ctx).Embed —
// it exists so a *WarmEmbedder can be passed where an Embedder is expected
// (the search path). It records the access (resets the idle window) and degrades
// to the FTS floor when the model is absent.
func (w *WarmEmbedder) Embed(ctx context.Context, text string) (memory.Vector, error) {
	return w.Get(ctx).Embed(ctx, text)
}

// EmbedQuery satisfies the Embedder interface by delegating to Get(ctx).
func (w *WarmEmbedder) EmbedQuery(ctx context.Context, text string) (memory.Vector, error) {
	return w.Get(ctx).EmbedQuery(ctx, text)
}

// Dimension satisfies the Embedder interface (the first-loaded dimension; 0
// until the first Get).
func (w *WarmEmbedder) Dimension() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.emb == nil {
		return 0
	}
	return w.emb.Dimension()
}
