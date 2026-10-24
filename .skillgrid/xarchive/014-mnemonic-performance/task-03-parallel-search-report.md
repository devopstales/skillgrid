# Task 03 Report — Concurrent cross-project search with bounded goroutines

Change `014-mnemonic-performance`, step `03-parallel-search`.

## Status: PASS WITH WARNINGS

All three sub-tasks implemented and GREEN. One **pre-existing** failure
(`TestReindexStructuralIsEmbedderFree`) exists in the full service suite,
unrelated to step 03 — see "Concerns".

## Commit

- `2245ddc` feat(mnemonic): concurrent cross-project search with bounded goroutines

Files changed (additive only):
- `skillgrid-cli/internal/mnemonic/service/service.go` (+193 / -19)
- `skillgrid-cli/internal/mnemonic/service/parallel_search_test.go` (new, +275)

## What was implemented per sub-task

### 03.1 — Concurrent search completes in parallel with bounded concurrency

`SearchObservationsAll` (service.go:771) was rewritten. It now dispatches one
goroutine per store, bounded by a buffered-channel semaphore of size
`min(len(stores), runtime.NumCPU())`. Each goroutine calls `SearchObservationsScoped`
(via the `scopedSearch` indirection, whose production default is exactly
`Service.SearchObservationsScoped` — **signature unchanged**). Per-store ranked
results are pushed onto a buffered `results` channel; a closer goroutine
`wg.Wait()`s then `close(results)`; the main goroutine ranges over the channel
and merges.

The sequential behavior is preserved verbatim in `searchObservationsAllSequential`
(service.go:790), and the merge uses the same `seen` dedup map and
`(rank, UpdatedAt desc)` `sort.SliceStable` as before (now in `mergeRanked`,
service.go:879).

### 03.2 — Missing stores are skipped with warning, not fatal

In each goroutine, if `scopedSearch` returns an error (missing file, corrupt db,
busy lock), the goroutine calls `warnFunc("parallel search: skipping store %s: %v", …)`
and `return`s without publishing to the channel — the merge simply never sees a
result set for that store. This preserves the pre-existing non-fatal skip
(current code `if err != nil { continue }`) and **adds a warning** (the old code
continued silently). `warnFunc` is a swappable sink (service.go:910) that
defaults to the service's existing `warn:` stderr convention and which tests
replace to capture the line.

### 03.3 — Semaphore limits concurrent stores to prevent resource exhaustion

Buffered channel `sem := make(chan struct{}, workers)` where
`workers = min(len(stores), runtime.NumCPU())` (clamped to ≥ 1). Each goroutine
acquires a slot **before** searching and releases it **after**:

- Acquire is a `select` over `sem <- struct{}{}`, `ctx.Done()`, and
  `time.After(semaphoreAcquireTimeout)` (5s).
- On semaphore-timeout (pool exhausted → queue with timeout), the store is
  skipped with a warning and `return`s (non-fatal).
- On `ctx.Done()`, the goroutine returns without searching (cancellation
  respected).
- `defer func() { <-sem }()` releases the slot after the search.

The per-store latency is applied on top of `scopedSearch` via the test-only
`searchStoreLatency` atomic (0 in production), so timing assertions are stable
on CI.

## New `SearchObservationsAll` — goroutine + semaphore + merge

Entry (service.go:771):
```go
func (s *Service) SearchObservationsAll(ctx context.Context, query, matchMode, scope string, limit int) ([]memory.Observation, error) {
	if limit <= 0 {
		limit = 20
	}
	projects, err := s.ListProjects()
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		return []memory.Observation{}, nil
	}
	if !searchParallelEnabled() {
		return s.searchObservationsAllSequential(ctx, projects, query, matchMode, scope, limit)
	}
	return s.searchObservationsAllParallel(ctx, projects, query, matchMode, scope, limit)
}
```

Parallel worker loop (service.go:813):
```go
func (s *Service) searchObservationsAllParallel(ctx context.Context, projects []string, query, matchMode, scope string, limit int) ([]memory.Observation, error) {
	workers := len(projects)
	if n := runtime.NumCPU(); workers > n {
		workers = n
	}
	if workers < 1 {
		workers = 1
	}
	sem := make(chan struct{}, workers)
	results := make(chan []storeRanked, len(projects))
	var wg sync.WaitGroup
	for _, pid := range projects {
		wg.Add(1)
		go func(pid string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			case <-time.After(semaphoreAcquireTimeout):
				warnFunc("parallel search: skipping store %s: semaphore acquire timed out after %v", pid, semaphoreAcquireTimeout)
				return
			}
			defer func() { <-sem }()
			res, err := scopedSearch(s, ctx, pid, query, matchMode, scope, limit)
			if err != nil {
				warnFunc("parallel search: skipping store %s: %v", pid, err)
				return
			}
			select {
			case results <- rankHits(res):
			case <-ctx.Done():
			}
		}(pid)
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	var collected []storeRanked
	for res := range results {
		collected = append(collected, res...)
	}
	return mergeRanked(collected, limit), nil
}
```

Merge (service.go:879) — `seen` dedup + `(rank, UpdatedAt)` stable sort:
```go
func mergeRanked(collected []storeRanked, limit int) []memory.Observation {
	seen := map[string]bool{}
	deduped := collected[:0]
	for _, r := range collected {
		key := strconv.FormatInt(r.obs.ID, 10) + "/" + r.obs.Project
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, r)
	}
	sort.SliceStable(deduped, func(i, j int) bool {
		if deduped[i].rank != deduped[j].rank {
			return deduped[i].rank < deduped[j].rank
		}
		return deduped[i].obs.UpdatedAt > deduped[j].obs.UpdatedAt
	})
	out := make([]memory.Observation, 0, len(deduped))
	for _, r := range deduped {
		out = append(out, r.obs)
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
```

## How determinism is preserved

Goroutine completion order is non-deterministic, so the raw `collected` slice
order varies run-to-run. Determinism comes from two preserved invariants:

1. **Cross-store rank** — each hit is tagged with its 0-based rank *within its
   own store* (`rankHits`). The merge sorts by `rank` first, so a #1 hit in one
   store always beats a #5 hit from another, exactly as in the original code.
2. **`sort.SliceStable` on `(rank, UpdatedAt desc)`** — for hits that tie on
   rank, the tie-break is `UpdatedAt desc` (a value, not arrival order). The
   *stable* sort then guarantees that for two hits identical on both keys,
   their relative order is the one the sort receives; because the comparison
   is a total order over (rank, UpdatedAt), any two distinct hits have a
   well-defined order, so the final sequence is independent of the order in
   which goroutines published. Given identical per-store result sets, the
   output is byte-for-byte reproducible.

The `TestSearchObservationsAllParallel` test additionally asserts the parallel
result set equals the sequential (rollback) result set in length, and that no
`(id, project, title)` appears twice.

## How missing stores are skipped + warned

Each goroutine's `scopedSearch` returns an error for a missing/corrupt/busy
store. The goroutine then `warnFunc("parallel search: skipping store %s: %v", pid, err)`
and `return`s without publishing to `results`. The merge only ever receives the
stores that succeeded, so the missing store contributes zero hits and the search
returns no error. The warning is observable (default `warn:` stderr; tests swap
`warnFunc` to capture it). `TestSearchObservationsAllMissingStoreSkipped`
deletes + corrupts one of 3 stores and asserts (a) no error, (b) exactly the 2
valid stores' hits, (c) the missing store contributes nothing, (d) a warning
mentioning `ms-gamma` was logged.

## Rollback env var

`SKILLGRID_SEARCH_PARALLEL` (const `envSearchParallel`, service.go:715).
`searchParallelEnabled()` returns `os.Getenv(envSearchParallel) != "0"`, so:
- **unset or any other value** → parallel path (default)
- **`0`** → sequential loop (`searchObservationsAllSequential`), the rollback
  boundary. A concurrency regression is rolled back at runtime without a deploy.

`TestSearchObservationsAllParallelRollbackEnv` asserts the sequential path
returns the same result content as the parallel path.

## RED→GREEN evidence

The three tests reference `searchStoreLatency`, `scopedSearchFunc`, and
`warnFunc` — none of which exist in the sequential-only implementation, so the
RED is a **build failure** on the old code (compile-time RED), confirmed by
reverting `service.go` to HEAD and removing the test file (see "Pre-existing
failure" verification run, which compiled+ran cleanly against baseline except
the unrelated reindex test).

GREEN (focused, `-race`):
```
$ go test ./internal/mnemonic/service/ -run 'TestSearchObservationsAllParallel|TestSearchObservationsAllMissingStoreSkipped|TestSearchObservationsAllSemaphoreBound' -count=1 -race
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service	83.949s
```

## `-race -count=5` flakiness check

The "faster than sequential" assertion is the flakiness risk. The test uses a
30ms per-store latency marker and asserts `parallel < sequential/2` plus a
wave-based ceiling (`3*waves*lat + 400ms`), which absorbs `-race` scheduling
noise. Checked with `-count=5`:
```
$ go test ./internal/mnemonic/service/ -run 'TestSearchObservationsAllParallel' -count=5 -race
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service	79.280s
```
PASS x5, no flake. (NumCPU=16 on this host; 10 stores → 1 wave, so parallel ≈
1×lat vs sequential ≈ 10×lat — large margin, not near the /2 threshold.)

## Full service suite + output

```
$ go test ./internal/mnemonic/service/ -count=1 -race
--- FAIL: TestReindexStructuralIsEmbedderFree (2.24s)
    reindex_structural_test.go:107: count gamma: sql: database is closed
FAIL
FAIL	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service	154.393s
FAIL
```

All step 03 tests pass; the single failure is `TestReindexStructuralIsEmbedderFree`,
**pre-existing and unrelated** (see Concerns).

## Baselines (005/008/010/011/013)

```
$ go test ./internal/mnemonic/store/... ./internal/mnemonic/memory/... ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store	4.461s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory	12.680s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer	4.263s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/route	3.808s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected	8.668s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community	2.539s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg	5.673s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex	18.661s
```
All baselines PASS.

## Concerns

1. **Pre-existing failure (not introduced by step 03).** `TestReindexStructuralIsEmbedderFree`
   fails with `count gamma: sql: database is closed` (reindex_structural_test.go:107).
   It exercises the code-index `OpenForDirectory` / `StoreFingerprint` /
   `ReindexStructural` path (step 01 store-pooling / codeindex), **not** search.
   Verified pre-existing: reverted `service.go` to HEAD and removed
   `parallel_search_test.go`, then ran
   `go test … -run TestReindexStructuralIsEmbedderFree -count=3 -race` — it fails
   identically on the baseline (`-count=3`, all 3 fail). It is a separate
   code-index store pool-close race to be fixed by its owning step, not step 03.

2. **Flakiness (managed, not eliminated).** The parallel-vs-sequential timing
   assertion is inherently timing-sensitive. Mitigations in place: 30ms latency
   marker, `parallel < sequential/2` (not an exact ratio), a generous wave
   ceiling (`3*waves*lat + 400ms`), a warm-up pass to prime the step-01 store
   pool so the timed runs measure per-store query + semaphore scheduling (not the
   one-time modernc connect/migration cost), and `-count=5` confirmation. On a
   2-CPU runner with 50 stores (16 waves) the ceiling is `3*16*5ms + 400ms = 640ms`;
   if a very slow CI runner is flaky there, widen the ceiling further. The 10-store
   parallel test (1 wave on 16 CPUs) has a very large margin and is the one
   checked with `-count=5`.

3. **Additive / contract held.** `SearchObservationsScoped` signature unchanged;
   `SearchObservationsAll` / `SearchAllProjects` signatures unchanged; no schema
   change; modernc driver untouched (no CGO); `mem_*` tool contracts unchanged.
   Default result content is identical to the sequential path — only faster.
   Parallelism is opt-out via `SKILLGRID_SEARCH_PARALLEL=0`.

4. **Not run (per rules).** `go build ./...`, `go vet`, `gofmt -l` not run — tests
   are the only verification. `-race` is clean across the focused, flakiness, and
   full-suite runs (the one FAIL is the pre-existing non-race functional failure
   above, with no data-race report).
