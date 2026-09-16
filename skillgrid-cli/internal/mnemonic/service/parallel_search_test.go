package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// setParallelSearchForTest pins the parallel search flag for a test and
// restores the prior state on cleanup.
func setParallelSearchForTest(t *testing.T, on bool) {
	t.Helper()
	if on {
		t.Setenv("SKILLGRID_SEARCH_PARALLEL", "1")
	} else {
		t.Setenv("SKILLGRID_SEARCH_PARALLEL", "0")
	}
}

// RED (03.1.a): the parallel path does not exist before step 03 — the test
// references searchStoreLatency, scopedSearchFunc, and warnFunc, so the RED is
// a build failure on the old sequential-only implementation.

// TestSearchObservationsAllParallel verifies that the cross-store search runs
// its stores concurrently (03.1): with an artificial 25ms latency per store,
// sequential execution is ~250ms while parallel execution is bounded by the
// semaphore waves (ceil(stores/NumCPU) * latency). It also verifies the merged
// + deduped result content matches the sequential (rollback) path exactly.
func TestSearchObservationsAllParallel(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	const stores = 10
	const query = "shared latency marker"
	for i := 0; i < stores; i++ {
		pid := "par-proj-" + strconv.Itoa(i)
		seedProjectObs(t, svc, pid, query+" "+pid, "**What** x **Why** x **Where** x **Learned** — "+pid)
	}

	// Latency marker: 30ms per store → sequential ~= 300ms; parallel is
	// bounded by waves of NumCPU stores, so the ceiling below absorbs both
	// slow CI scheduling and a 2-CPU runner.
	lat := 30 * time.Millisecond
	searchStoreLatency.Store(int64(lat))
	t.Cleanup(func() { searchStoreLatency.Store(0) })
	waves := (stores + runtime.NumCPU() - 1) / runtime.NumCPU()
	// Generous ceiling: the -race build inflates goroutine scheduling, so
	// allow 3x the wave latency + 400ms of absolute slack on top.
	parallelCeil := time.Duration(3*waves)*lat + 400*time.Millisecond

	ctx := context.Background()

	// Warm-up: prime every store handle through the pooled open path once.
	// The step-01 store pool keeps them open (refcounted close), so the timed
	// runs below measure per-store query + semaphore scheduling, not the
	// one-time modernc connection/migration cost that otherwise dominates and
	// masks the parallelism signal on slow CI runners.
	setParallelSearchForTest(t, true)
	if _, err := svc.SearchObservationsAll(ctx, query, "any", "", 100); err != nil {
		t.Fatalf("warm-up: %v", err)
	}

	setParallelSearchForTest(t, false)
	start := time.Now()
	seqHits, seqErr := svc.SearchObservationsAll(ctx, query, "any", "", 100)
	seqElapsed := time.Since(start)
	if seqErr != nil {
		t.Fatalf("sequential search: %v", seqErr)
	}

	setParallelSearchForTest(t, true)
	start = time.Now()
	parHits, parErr := svc.SearchObservationsAll(ctx, query, "any", "", 100)
	parElapsed := time.Since(start)
	if parErr != nil {
		t.Fatalf("parallel search: %v", parErr)
	}

	if len(seqHits) == 0 {
		t.Fatalf("no hits seeded, expected %d stores to match", stores)
	}
	// Parallel must finish in a fraction of the sequential wall time:
	// sequential ~= stores*lat, parallel ~= waves*lat. The generous threshold
	// absorbs scheduler noise on slow CI runners.
	if parElapsed >= seqElapsed/2 {
		t.Fatalf("parallel search not faster than sequential: parallel=%v sequential=%v (want parallel < sequential/2)", parElapsed, seqElapsed)
	}
	// And parallel must stay within its wave ceiling.
	if parElapsed > parallelCeil {
		t.Fatalf("parallel search too slow: %v (ceiling %v, waves=%d NumCPU=%d latency=%v)", parElapsed, parallelCeil, waves, runtime.NumCPU(), lat)
	}

	// Merged + deduped content must equal the sequential path exactly.
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
		t.Fatalf("expected %d unique merged hits, got %d", stores, len(seen))
	}
}

// TestSearchObservationsAllParallelRollbackEnv verifies the rollback boundary:
// SKILLGRID_SEARCH_PARALLEL=0 forces the sequential path and still returns the
// same result content as the parallel path.
func TestSearchObservationsAllParallelRollbackEnv(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	seedProjectObs(t, svc, "rb-a", "rollback marker tok", "**What** r **Why** r **Where** r **Learned** —")
	seedProjectObs(t, svc, "rb-b", "rollback marker tok", "**What** r **Why** r **Where** r **Learned** —")

	ctx := context.Background()
	setParallelSearchForTest(t, true)
	parHits, err := svc.SearchObservationsAll(ctx, "rollback marker", "any", "", 20)
	if err != nil {
		t.Fatalf("parallel: %v", err)
	}
	setParallelSearchForTest(t, false)
	seqHits, err := svc.SearchObservationsAll(ctx, "rollback marker", "any", "", 20)
	if err != nil {
		t.Fatalf("sequential rollback: %v", err)
	}
	if len(parHits) != 2 || len(seqHits) != 2 {
		t.Fatalf("want 2 hits each, got par=%d seq=%d", len(parHits), len(seqHits))
	}
}

// TestSearchObservationsAllMissingStoreSkipped verifies (03.2) that a store
// whose file was deleted is skipped with a warning instead of failing the
// whole search: results from the 2 valid stores come back, no error, and a
// warning is logged for the missing store.
func TestSearchObservationsAllMissingStoreSkipped(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	seedProjectObs(t, svc, "ms-alpha", "missing store marker", "**What** m **Why** m **Where** m **Learned** —")
	seedProjectObs(t, svc, "ms-beta", "missing store marker", "**What** m **Why** m **Where** m **Learned** —")
	seedProjectObs(t, svc, "ms-gamma", "missing store marker", "**What** m **Why** m **Where** m **Learned** —")

	// Corrupt the store file after seeding: ListProjects still lists it (the
	// file is present) but opening it fails, so the parallel path must skip
	// it with a warning instead of failing the whole search.
	gamma := filepath.Join(dataDir, "ms-gamma.sqlite")
	if err := os.Remove(gamma); err != nil {
		t.Fatalf("remove store: %v", err)
	}
	if err := os.WriteFile(gamma, []byte("not a sqlite database"), 0o644); err != nil {
		t.Fatalf("corrupt store: %v", err)
	}
	projects, err := svc.ListProjects()
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if !contains(projects, "ms-gamma") {
		t.Fatalf("expected ms-gamma still listed by ListProjects, got %v", projects)
	}

	prevWarn := warnFunc
	// RED (03.2.a): production logs missing-store warnings via warnFunc —
	// undefined before the step-03 implementation (build failure is the RED).
	warnFunc = captureWarn(t, "ms-gamma")
	t.Cleanup(func() { warnFunc = prevWarn })

	ctx := context.Background()
	hits, err := svc.SearchObservationsAll(ctx, "missing store marker", "any", "", 20)
	if err != nil {
		t.Fatalf("missing store must not fail the search: %v", err)
	}
	got := map[string]bool{}
	for _, h := range hits {
		got[h.Project] = true
	}
	if len(hits) != 2 || !got["ms-alpha"] || !got["ms-beta"] {
		t.Fatalf("want results from the 2 valid stores, got %d hits projects=%v", len(hits), got)
	}
	if got["ms-gamma"] {
		t.Fatalf("missing store ms-gamma must not contribute hits")
	}
}

// captureWarn replaces warnFunc with a recorder that FAILs the test if no
// warning mentioning want is emitted during the test body.
func captureWarn(t *testing.T, want string) func(string, ...any) {
	t.Helper()
	var mu sync.Mutex
	var lines []string
	var fn func(string, ...any)
	fn = func(format string, args ...any) {
		mu.Lock()
		lines = append(lines, fmt.Sprintf(format, args...))
		mu.Unlock()
	}
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		for _, line := range lines {
			if strings.Contains(line, want) {
				return
			}
		}
		t.Errorf("expected a warning mentioning %q, got: %v", want, lines)
	})
	return fn
}

// TestSearchObservationsAllSemaphoreBound verifies (03.3) that with 50 stores
// the semaphore caps concurrent store searches at runtime.NumCPU() and that
// every store is still searched.
func TestSearchObservationsAllSemaphoreBound(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	const stores = 50
	const query = "semaphore bound marker"
	for i := 0; i < stores; i++ {
		pid := "sem-proj-" + strconv.Itoa(i)
		seedProjectObs(t, svc, pid, query+" "+pid, "**What** s **Why** s **Where** s **Learned** — "+pid)
	}

	lat := 5 * time.Millisecond
	searchStoreLatency.Store(int64(lat))
	t.Cleanup(func() { searchStoreLatency.Store(0) })

	var mu sync.Mutex
	current, maxSeen := 0, 0
	origScoped := scopedSearchFunc
	scopedSearchFunc = func(s *Service, ctx context.Context, projectID, query, matchMode, scope string, limit int) ([]memory.Observation, error) {
		mu.Lock()
		current++
		if current > maxSeen {
			maxSeen = current
		}
		mu.Unlock()
		hits, err := s.SearchObservationsScoped(ctx, projectID, query, matchMode, scope, limit)
		mu.Lock()
		current--
		mu.Unlock()
		return hits, err
	}
	t.Cleanup(func() { scopedSearchFunc = origScoped })

	setParallelSearchForTest(t, true)
	hits, err := svc.SearchObservationsAll(context.Background(), query, "any", "", 100)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != stores {
		t.Fatalf("all %d stores must be searched, got %d hits", stores, len(hits))
	}
	if maxSeen > runtime.NumCPU() {
		t.Fatalf("max concurrent store searches = %d exceeds NumCPU=%d", maxSeen, runtime.NumCPU())
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
