package pdg

import (
	"context"
	"sync"
	"testing"
	"time"
)

// hangingReader blocks on Read until released (models a hung server stdout).
type hangingReader struct{ release chan struct{} }

func (h *hangingReader) Read(p []byte) (int, error) {
	<-h.release
	return 0, nil
}

// TestReadAllCtxTimesOut covers fix #1 (the LSP 30s timeout is a real
// wall-clock bound, not a no-op): readAllCtx on a reader that blocks returns
// an error as soon as the context is cancelled (the timeout), instead of
// blocking indefinitely. The timeout is the injected ctx deadline — no 30s wait.
func TestReadAllCtxTimesOut(t *testing.T) {
	h := &hangingReader{release: make(chan struct{})}
	defer close(h.release)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := readAllCtx(ctx, h)
	if err == nil {
		t.Fatal("expected readAllCtx to return an error when ctx times out, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
	if time.Since(start) < 20*time.Millisecond {
		t.Errorf("readAllCtx returned too fast (%v) — ctx was not honored", time.Since(start))
	}
}

// TestCtxWithTimeoutHonored asserts the injectable ctxWithTimeout default
// actually derives a bounded context (fix #1: it used to return ctx unchanged).
func TestCtxWithTimeoutHonored(t *testing.T) {
	ctx, cancel := ctxWithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	select {
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			t.Fatalf("expected DeadlineExceeded, got %v", ctx.Err())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ctxWithTimeout did not enforce the timeout (no deadline)")
	}
}

// TestResolveMemberCallsHonorsBoundedCtx covers fix #1 at the seam level: with
// a very short injected round-trip timeout, a resolver seam that observes the
// bounded ctx (returns when it fires) lets ResolveMemberCalls complete fast
// (best-effort no-op) instead of running to 30s. A seam that observes ctx is
// the hermetic analog of a server read that unblocks on the deadline.
func TestResolveMemberCallsHonorsBoundedCtx(t *testing.T) {
	// A resolver that blocks until the bounded ctx fires, then returns.
	watcher := func(ctx context.Context, filePath string, line int, receiver, callee string) (string, bool) {
		<-ctx.Done() // block until the round-trip deadline fires
		return "", false
	}
	file := LSPFile{Path: "a.go", Contents: []byte("package p\n\nfunc f() {\n\ts.Do(1)\n}\n\ntype S struct{}\n")}
	client := NewLSPClientForTest(LSPClientOptions{
		Root:    "",
		Files:   func() []LSPFile { return []LSPFile{file} },
		Timeout: 40 * time.Millisecond, // << 30s: the injected short timeout
	}, watcher)
	// Bound the round-trip exactly as the pass does (ctxWithTimeout, fix #1).
	bctx, cancel := ctxWithTimeout(context.Background(), client.opts.Timeout)
	defer cancel()
	start := time.Now()
	edges, err := client.ResolveMemberCalls(bctx)
	elapsed := time.Since(start)
	// Ceiling is -race-tolerant: it must stay far under the 30s default the
	// test guards against (proof the injected bound is honored), but wide
	// enough that a 40ms deadline under -race scheduler latency cannot flake.
	// Observed -race latency for the 40ms deadline to fire is ~2s; 5s is 6x
	// under the 30s default yet far above that latency.
	if elapsed >= 5*time.Second {
		t.Errorf("resolver under a 40ms timeout took %v — the injected timeout was not honored", elapsed)
	}
	if err != nil {
		// The seam observed ctx.Done and returned (no error); resolveWith3 then
		// surfaces ctx.Err(). Accept either a clean no-op or a timeout error —
		// the contract is: fast, no partial set.
		t.Logf("note: seam surfaced ctx error: %v", err)
	}
	if len(edges) != 0 {
		t.Errorf("a timed-out resolver must not return a partial edge set, got %d", len(edges))
	}
}

// TestResolveMemberCallsNoTimeoutIsUnbounded verifies that with timeout=0 (the
// default hermetic path, no real server) the seam is not artificially bounded:
// a fast resolver returns its edges and no error.
func TestResolveMemberCallsNoTimeoutIsUnbounded(t *testing.T) {
	file := LSPFile{Path: "a.go", Contents: []byte("package p\n\nfunc f() {\n\ts.Do(1)\n}\n\ntype S struct{}\n")}
	var mu sync.Mutex
	calls := 0
	resolver := func(ctx context.Context, filePath string, line int, receiver, callee string) (string, bool) {
		mu.Lock()
		calls++
		mu.Unlock()
		if receiver == "s" && callee == "Do" {
			return "Do", true
		}
		return "", false
	}
	client := NewLSPClientForTest(LSPClientOptions{
		Root:  "",
		Files: func() []LSPFile { return []LSPFile{file} },
		// Timeout left 0 -> unbounded hermetic path (no artificial deadline).
	}, resolver)
	edges, err := client.ResolveMemberCalls(context.Background())
	if err != nil {
		t.Fatalf("unbounded hermetic resolver errored: %v", err)
	}
	if len(edges) == 0 {
		t.Fatal("expected at least one resolved edge, got 0")
	}
	if calls == 0 {
		t.Fatal("resolver seam was never invoked")
	}
}
