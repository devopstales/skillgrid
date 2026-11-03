package memory

import (
	"context"
	"testing"
	"time"
)

// TestTTLBoundaryExactSecond is 26.1 [RED] — the TTL boundary semantics at the
// exact second where expires_at == now. The retire predicate is
// `strftime('%s', expires_at) <= strftime('%s', 'now')` (a row expiring on the
// current second IS retired — inclusive boundary), while the search exclusion
// is `strftime('%s', expires_at) > strftime('%s', 'now')` (the same row is
// hidden — a row on the boundary second is NOT search-visible). The two must
// agree: the exact-boundary row is retired AND hidden from search, while a row
// one second into the future is neither. This pins the `<=` vs `>` pairing so
// a future edit that loosens one operator without the other fails loudly.
func TestTTLBoundaryExactSecond(t *testing.T) {
	_, svc := newTestStore(t, "ttlboundary")
	sid := newSession(t, svc)
	ctx := context.Background()

	// Exact-boundary row: expires_at == now (truncated to the second, matching
	// the strftime('%s') granularity both predicates compare at).
	boundary := time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
	// One second into the future: still live at the second-granularity compare.
	future := time.Now().UTC().Add(time.Second).Truncate(time.Second).Format(time.RFC3339)

	boundaryID, err := svc.Save(ctx, SaveInput{
		Title: "ttl boundary", Type: "decision", Content: "expires on the current second",
		SessionID: sid, Scope: "project",
	})
	if err != nil {
		t.Fatalf("save boundary: %v", err)
	}
	if err := svc.SetExpiresAt(ctx, boundaryID, boundary); err != nil {
		t.Fatalf("set boundary expiry: %v", err)
	}
	futureID, err := svc.Save(ctx, SaveInput{
		Title: "ttl future second", Type: "decision", Content: "expires one second out",
		SessionID: sid, Scope: "project",
	})
	if err != nil {
		t.Fatalf("save future: %v", err)
	}
	if err := svc.SetExpiresAt(ctx, futureID, future); err != nil {
		t.Fatalf("set future expiry: %v", err)
	}

	// Search must hide the exact-boundary row (predicate `> now` excludes it)
	// but return the one-second-future row.
	hits, err := svc.Search(ctx, "ttl", "any", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	seen := map[int64]bool{}
	for _, h := range hits {
		seen[h.ID] = true
	}
	if seen[boundaryID] {
		t.Fatalf("search returned the exact-boundary row (expires_at == now); predicate `> now` must hide it")
	}
	if !seen[futureID] {
		t.Fatalf("search did not return the one-second-future row; predicate `> now` must keep it")
	}

	// TTLRetire (predicate `<= now`) must retire the exact-boundary row and
	// leave the future row live.
	retired, err := svc.TTLRetire(ctx)
	if err != nil {
		t.Fatalf("ttl retire: %v", err)
	}
	if retired != 1 {
		t.Fatalf("TTLRetire retired %d, want exactly 1 (only the boundary row)", retired)
	}

	// The boundary row is now soft-deleted (unreadable); the future row is not.
	if _, err := svc.Get(ctx, boundaryID); err == nil {
		t.Fatal("exact-boundary row still readable after TTLRetire; `<= now` must have retired it")
	}
	if _, err := svc.Get(ctx, futureID); err != nil {
		t.Fatalf("one-second-future row unreadable after TTLRetire: %v", err)
	}
}
