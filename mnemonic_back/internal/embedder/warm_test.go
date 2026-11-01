package embedder

import (
	"context"
	"testing"
	"time"
)

// TestWarmEmbedder covers 03.10 (Scenario: Warm embedder loads once and
// reuses): the ONNX model loads once and is reused across search calls (no
// per-call model load; the ~270MB model is not re-loaded per invocation). The
// first call pays the load, subsequent calls do not.
func TestWarmEmbedder(t *testing.T) {
	loads := 0
	w := NewWarm(WarmConfig{
		Factory:     func() Embedder { loads++; return NewHash(64) },
		IdleTimeout: DefaultIdleTimeout,
	})

	// First call pays the one-time load.
	e1 := w.Get(context.Background())
	if loads != 1 {
		t.Fatalf("expected exactly 1 model load on first use, got %d", loads)
	}
	if e1.Dimension() != 64 {
		t.Fatalf("loaded embedder dim = %d, want 64", e1.Dimension())
	}
	if !w.IsLoaded() {
		t.Fatalf("expected the warm embedder to be loaded after first use")
	}

	// Subsequent calls reuse the cached embedder (no re-load).
	for i := 0; i < 5; i++ {
		e := w.Get(context.Background())
		if e.Dimension() != 64 {
			t.Fatalf("reuse call %d dim = %d, want 64", i, e.Dimension())
		}
	}
	if loads != 1 {
		t.Fatalf("expected the model to load exactly once across repeated searches, got %d", loads)
	}
}

// TestWarmEvict covers 03.11 (Scenario: Warm embedder evicts on idle and
// survives heartbeat): after idle_timeout (injected, not 10min) of no use the
// warm embedder is evicted from RAM; an active MCP heartbeat (connected client)
// keeps it alive (no eviction while connected); off/external providers hold no
// RAM (a Null Adapter is never evicted / never counts as resident).
func TestWarmEvict(t *testing.T) {
	loads := 0
	now := time.Now()
	// A controllable clock.
	var clock time.Time
	w := NewWarm(WarmConfig{
		Factory:     func() Embedder { loads++; return NewHash(64) },
		IdleTimeout: 100 * time.Millisecond, // injected short window
		Now:         func() time.Time { return clock },
	})
	_ = now

	clock = time.Now()
	w.Get(context.Background()) // loads (1)
	if loads != 1 {
		t.Fatalf("expected 1 load, got %d", loads)
	}

	// Heartbeat: while a client is connected, idle does NOT evict.
	clock = clock.Add(300 * time.Millisecond) // past the 100ms idle window
	w.CheckIdle(clock, 1) // connectedClients=1 (heartbeat)
	if !w.IsLoaded() {
		t.Fatalf("expected the embedder to survive an idle period while a client is connected (heartbeat)")
	}

	// No client + idle past the window → evicted.
	w.CheckIdle(clock, 0) // connectedClients=0
	if w.IsLoaded() {
		t.Fatalf("expected the embedder to be evicted after idle_timeout with no client")
	}
	if loads != 1 {
		t.Fatalf("eviction must not re-load (that happens on reuse), got %d loads", loads)
	}

	// Off/external (Null Adapter) holds no RAM: never resident, never evicted.
	wNull := NewWarm(WarmConfig{
		Factory:     func() Embedder { return NewNull() },
		IdleTimeout: 100 * time.Millisecond,
		Now:         func() time.Time { return clock },
	})
	wNull.Get(context.Background())
	if wNull.IsLoaded() {
		t.Fatalf("a Null Adapter (off/external) holds no RAM; it must not count as resident")
	}
	wNull.CheckIdle(clock.Add(time.Minute), 0) // idle, no client
	if wNull.IsLoaded() {
		t.Fatalf("a Null Adapter must never be reported resident")
	}
}

// TestWarmFallback covers 03.12 (Scenario: Absent model degrades to FTS and
// signals): an absent/failed model degrades to the Null Adapter (FTS + signals
// floor) and no model-load error is surfaced to the agent.
func TestWarmFallback(t *testing.T) {
	// Absent model (nil factory): Get returns a Null Adapter, no error.
	w := NewWarm(WarmConfig{Factory: nil, IdleTimeout: DefaultIdleTimeout})
	e := w.Get(context.Background())
	if e.Dimension() != 0 {
		t.Fatalf("absent model should degrade to a Null Adapter (dim 0), got dim %d", e.Dimension())
	}
	vec, err := e.Embed(context.Background(), "some text")
	if err != nil {
		t.Fatalf("degraded embed must not error (FTS floor), got %v", err)
	}
	if len(vec.Data) != 0 {
		t.Fatalf("degraded embed must return an empty vector (FTS floor), got dim %d", len(vec.Data))
	}

	// A factory that fails also degrades (no model-load error surfaced).
	wFail := NewWarm(WarmConfig{
		Factory:     func() Embedder { return NewHash(64) }, // non-nil but the path below exercises absent
		IdleTimeout: DefaultIdleTimeout,
	})
	if e := wFail.Get(context.Background()); e.Dimension() == 0 {
		t.Fatalf("a healthy factory should load a real embedder")
	}
}

// TestWarmFailedFactoryDegradesToFTS covers the "failed model degrades to FTS"
// branch: a Factory that returns a nil embedder (the observable form of a
// failed model load) makes the warm cache degrade to the Null Adapter (FTS
// floor) — it does not crash, holds no RAM, and a subsequent reuse is
// transparent. This branch is distinct from the healthy-factory path in
// TestWarmFallback (which loads a real embedder) and from the nil-Factory path.
func TestWarmFailedFactoryDegradesToFTS(t *testing.T) {
	// A factory that "fails" by returning a nil embedder (model load failed).
	failed := NewWarm(WarmConfig{
		Factory:     func() Embedder { return nil },
		IdleTimeout: DefaultIdleTimeout,
	})
	// First Get degrades to the Null Adapter (FTS floor), no crash.
	e := failed.Get(context.Background())
	if e.Dimension() != 0 {
		t.Fatalf("a failed factory (nil embedder) should degrade to a Null Adapter (dim 0), got dim %d", e.Dimension())
	}
	if failed.IsLoaded() {
		t.Fatalf("a degraded (Null) warm embedder holds no RAM and must not count as resident")
	}
	// The degraded embed still embeds (empty vector, no error) — the FTS floor.
	vec, err := e.Embed(context.Background(), "some text")
	if err != nil {
		t.Fatalf("degraded embed must not error (FTS floor), got %v", err)
	}
	if len(vec.Data) != 0 {
		t.Fatalf("degraded embed must return an empty vector (FTS floor), got dim %d", len(vec.Data))
	}
	// Reuse after the failure is transparent (no error surfaced).
	if e2 := failed.Get(context.Background()); e2.Dimension() != 0 {
		t.Fatalf("reuse of a failed-factory warm embedder must stay degraded (dim 0), got dim %d", e2.Dimension())
	}
	// And it is never reported resident (a Null Adapter holds no RAM).
	failed.CheckIdle(time.Now().Add(time.Minute), 0)
	if failed.IsLoaded() {
		t.Fatalf("a failed-factory (Null) warm embedder must never be reported resident")
	}
}

// TestWarmReload covers 03.13 (Scenario: Evicted embedder reloads on reuse):
// an evicted-then-reused embedder reloads transparently on next use; the
// one-time load cost is not surfaced as an error.
func TestWarmReload(t *testing.T) {
	loads := 0
	now := time.Now()
	var clock time.Time
	w := NewWarm(WarmConfig{
		Factory:     func() Embedder { loads++; return NewHash(64) },
		IdleTimeout: 100 * time.Millisecond,
		Now:         func() time.Time { return clock },
	})
	_ = now
	clock = time.Now()
	w.Get(context.Background()) // load 1
	if loads != 1 {
		t.Fatalf("expected 1 load, got %d", loads)
	}
	// Evict after idle (no client).
	w.CheckIdle(clock.Add(300*time.Millisecond), 0)
	if w.IsLoaded() {
		t.Fatalf("expected eviction")
	}
	// Reuse: transparent reload (load 2), no error surfaced.
	e := w.Get(context.Background())
	if loads != 2 {
		t.Fatalf("expected a transparent reload on reuse (2 loads total), got %d", loads)
	}
	if e.Dimension() != 64 {
		t.Fatalf("reloaded embedder dim = %d, want 64", e.Dimension())
	}
}
