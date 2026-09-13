package service

import (
	"context"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// TestSearchObservationsAllParallelManyStores is 26.1 [RED] — the cross-store
// parallel search scales to 50+ stores: the existing parallel test (step 03)
// covers 10 stores; this one pins the contract at 60. With the step-01 store
// pool, all 60 handles stay open (refcounted close), so the run measures the
// semaphore-wave scheduling + per-store query, not cold opens. The assertions
// mirror the 10-store test: parallel is faster than sequential, stays within
// its wave ceiling, and the merged + deduped result content equals the
// sequential path exactly (one unique hit per store).
func TestSearchObservationsAllParallelManyStores(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	const stores = 60
	const query = "many-store latency marker"
	for i := 0; i < stores; i++ {
		pid := "many-proj-" + strconv.Itoa(i)
		seedProjectObs(t, svc, pid, query+" "+pid, "**What** x **Why** x **Where** x **Learned** — "+pid)
	}

	lat := 15 * time.Millisecond
	searchStoreLatency.Store(int64(lat))
	t.Cleanup(func() { searchStoreLatency.Store(0) })
	waves := (stores + runtime.NumCPU() - 1) / runtime.NumCPU()
	// 60 stores at 15ms: sequential ~= 900ms; parallel ~= waves*lat. The
	// ceiling absorbs -race scheduling inflation + slow CI runners.
	parallelCeil := time.Duration(3*waves)*lat + 600 * time.Millisecond

	ctx := context.Background()

	// Warm-up: prime every pooled store handle once so the timed runs measure
	// scheduling, not the one-time modernc connection/migration cost.
	setParallelSearchForTest(t, true)
	if _, err := svc.SearchObservationsAll(ctx, query, "any", "", stores+10); err != nil {
		t.Fatalf("warm-up: %v", err)
	}

	setParallelSearchForTest(t, false)
	start := time.Now()
	seqHits, seqErr := svc.SearchObservationsAll(ctx, query, "any", "", stores+10)
	seqElapsed := time.Since(start)
	if seqErr != nil {
		t.Fatalf("sequential search: %v", seqErr)
	}

	setParallelSearchForTest(t, true)
	start = time.Now()
	parHits, parErr := svc.SearchObservationsAll(ctx, query, "any", "", stores+10)
	parElapsed := time.Since(start)
	if parErr != nil {
		t.Fatalf("parallel search: %v", parErr)
	}

	if len(seqHits) == 0 {
		t.Fatalf("no hits seeded, expected %d stores to match", stores)
	}
	if parElapsed >= seqElapsed/2 {
		t.Fatalf("parallel search (60 stores) not faster than sequential: parallel=%v sequential=%v", parElapsed, seqElapsed)
	}
	if parElapsed > parallelCeil {
		t.Fatalf("parallel search (60 stores) too slow: %v (ceiling %v, waves=%d NumCPU=%d latency=%v)", parElapsed, parallelCeil, waves, runtime.NumCPU(), lat)
	}

	if len(parHits) != len(seqHits) {
		t.Fatalf("parallel hit count=%d, sequential=%d", len(parHits), len(seqHits))
	}
	seen := map[string]bool{}
	for _, h := range parHits {
		key := strconv.FormatInt(h.ID, 10) + "/" + h.Project + "/" + h.Title
		if seen[key] {
			t.Fatalf("duplicate observation in parallel results: %s", key)
		}
		seen[key] = true
	}
	if len(seen) != stores {
		t.Fatalf("expected %d unique merged hits across 60 stores, got %d", stores, len(seen))
	}
}
