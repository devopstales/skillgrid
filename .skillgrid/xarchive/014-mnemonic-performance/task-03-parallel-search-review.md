# Task 03 Review — Concurrent cross-project search with bounded goroutines

Change `014-mnemonic-performance`, step `03-parallel-search`.
Reviewer: fresh, no prior context. Judged artifact: `2245ddc` (1 commit, 2 files).

## Verdict: PASS WITH WARNINGS

**Approved to merge: yes.**

The concurrency is structurally race-free and deterministic given the pre-existing
tie-break caveat. All four brief sub-tasks (03.1–03.3 + rollback) are delivered.
Findings are minor/nit — none block merge.

---

## Findings

### Minor

**M1 — `warnMu` is declared but never used (dead symbol).** `service.go:909`
```go
var warnMu sync.Mutex
var warnFunc = func(format string, args ...any) {
    fmt.Fprintf(os.Stderr, "warn: "+format+"\n", args...)
}
```
The comment says "Guarded by warnMu" but no code path takes `warnMu`. The default
`fmt.Fprintf` is safe (a single Glibc write is a single syscall, and concurrent
`Fprintf` to one `*os.File` is data-race-free), and the test replacement guards its
own slice with a local `mu` (`parallel_search_test.go:216-223`). So there is no
actual race — but the mutex is misleading dead weight. Fix: delete `var warnMu`
and the "Guarded by warnMu" clause, or actually take it inside the default func
(`warnMu.Lock(); defer warnMu.Unlock()`). Not a correctness issue.

**M2 — Missing-store test does not actually test "missing" (only "corrupt").**
`parallel_search_test.go:174-180` removes `ms-gamma.sqlite` then writes a corrupt
file back. `store.Open` runs `os.MkdirAll` + `sql.Open` before any read, and
modernc `sql.Open` does **not** fail on an absent file (lazy connect). So a truly
missing store would be *created fresh and return empty* (no warning, no skip); only
the corrupt-file path returns an error and exercises the 03.2 warning branch. The
test is still a valid test of the warning-on-error branch (it passes: I re-ran
`TestSearchObservationsAllMissingStoreSkipped` — PASS), and 03.2 only requires
"skip + warn on error," but the brief's literal scenario ("delete the file for one")
is not the scenario actually verified. Fix: either drop the corrupt-file rewrite and
rely on the modernc-open-successful-empty-store behavior (then assert no warning), or
keep corrupt but rename the comment to "corrupt store." Minor — branch coverage is
correct; the label is slightly off.

### Nit

**N1 — Determinism tie-break for identical `(rank, UpdatedAt)` is arrival-order, not
fully stable.** `mergeRanked` (`service.go:879-904`) sorts with
```go
sort.SliceStable(deduped, func(i, j int) bool {
    if deduped[i].rank != deduped[j].rank { return deduped[i].rank < deduped[j].rank }
    return deduped[i].obs.UpdatedAt > deduped[j].obs.UpdatedAt
})
```
`sort.SliceStable` preserves input order *only for elements the less-fn calls equal*.
Two distinct observations (different `Project`, e.g. the same content saved in two
child-project stores — the exact `all_projects` use case) with equal rank and equal
`UpdatedAt` are "equal" per less-fn, so their order follows `collected` order — which
is **non-deterministic** (channel publish order = goroutine completion order). The
report's claim "byte-for-byte reproducible" is correct only when
`(rank, UpdatedAt)` uniquely identifies each hit; it does not when two stores carry
the same-`UpdatedAt` row. `UpdatedAt` is RFC3339 second-precision (memory writes use
`time.Now().UTC().Format(time.RFC3339)`), so same-second multi-store saves are a
plausible tie. **This behavior is inherited verbatim from the original sequential
`SearchObservationsAll`** (identical `sort.SliceStable` key), so it is not a
regression — but the parallel path widens the blast radius because a cross-store tie
now collides where the sequential path ordered by fixed `projects` order. Fix (cheap,
backward-compatible): add a deterministic final key,
`return r.obs.Project + "/" + strconv.FormatInt(r.obs.ID,10) < ...`.

**N2 — Pre-existing full-suite failure is not step 03.** `TestReindexStructuralIsEmbedderFree`
fails (`count gamma: sql: database is closed`, reindex_structural_test.go:107) — the
step-01 codeindex store-pool close race, correctly attributed in the report. No
data-race report in the output; it is a functional failure in a different subsystem.
Tracked by its owning step.

---

## Evidence checked (all 8)

1. **Data races — VERIFIED clean by inspection.** The only shared mutable state is
   the `collected` slice, `seen` map, and `deduped`, all touched exclusively by the
   **main/merge goroutine**:
   - Workers (`service.go:826-846`) never touch `collected`/`seen`. Each worker only
     (a) acquires the `sem` channel, (b) calls `scopedSearch`, (c) sends its
     `rankHits(res)` slice to `results`. Quote:
     ```go
     go func(pid string) {
         defer wg.Done()
         select { case sem <- struct{}{}:
                  case <-ctx.Done(): return
                  case <-time.After(semaphoreAcquireTimeout): warnFunc(...); return }
         defer func() { <-sem }()
         res, err := scopedSearch(s, ctx, pid, query, matchMode, scope, limit)
         if err != nil { warnFunc("... skipping store %s: %v", pid, err); return }
         select { case results <- rankHits(res): case <-ctx.Done(): }
     }(pid)
     ```
   - The collector is single-goroutine:
     ```go
     go func() { wg.Wait(); close(results) }()   // closer
     var collected []storeRanked
     for res := range results { collected = append(collected, res...) }  // merge
     return mergeRanked(collected, limit), nil
     ```
     `range` over a channel delivers each value exactly once; `close` happens after
     `wg.Wait()`, so the loop terminates. `rankHits` builds a fresh slice per store —
     no cross-goroutine aliasing. `scopedSearch` reads the `scopedSearchFunc` pointer
     under `scopedSearchMu` (read once, local var), and the test replaces it
     *before* the search (happens-before via the goroutine spawn), so no torn read.
   - No shared slice/map is appended from workers. **Race-free structure confirmed**,
     not just trusted from the `-race` run.

2. **Determinism — VERIFIED with one caveat (N1).** Sort key `(rank asc,
   UpdatedAt desc)` is preserved from the original. `sort.SliceStable` makes output
   independent of goroutine completion order **except** for hits equal on both keys,
   where order is arrival-order (non-deterministic, pre-existing, widened in
   parallel). See N1.

3. **Semaphore correctness — VERIFIED.** `workers = min(len(projects), NumCPU)`
   clamped `≥1` (`service.go:814-820`); `sem := make(chan struct{}, workers)`.
   Acquire is a `select` over `sem <- struct{}{}`, `ctx.Done()`, and
   `time.After(5s)` — i.e. acquired **before** search, released **after** via
   `defer func() { <-sem }()` (service.go:836). `len(stores)==0` returns early at
   `service.go:779` (`return []memory.Observation{}, nil`) — no channel, no
   deadlock. 5s acquire timeout implemented; on timeout the store is skipped with a
   warning and `return`s (non-fatal). Timeout is effectively unreachable in normal
   operation (each worker releases immediately after its search, so a waiting worker
   always finds a free slot in < search-time), so no 5s stall in practice.

4. **Missing-store handling — VERIFIED.** Each worker, on `scopedSearch` error, calls
   `warnFunc("parallel search: skipping store %s: %v", pid, err)` and returns without
   publishing (service.go:838-841); the merge only ever receives successful stores, so
   nil/absent result sets are naturally skipped. **Not silent** — it warns.
   Empirically confirmed: I re-ran `TestSearchObservationsAllMissingStoreSkipped`
   (`-race`) → PASS; it asserts no error, exactly the 2 valid stores' hits, the
   missing store contributes nothing, and a warning mentioning `ms-gamma` was logged
   (via `captureWarn` cleanup that fails the test if absent).

5. **Rollback boundary — VERIFIED.** `searchParallelEnabled()` returns
   `os.Getenv("SKILLGRID_SEARCH_PARALLEL") != "0"` (service.go:719-721): `0` →
   sequential, unset/other → parallel. The sequential path
   (`searchObservationsAllSequential`, service.go:790-804) reuses the **same**
   `scopedSearch` + `rankHits` + `mergeRanked` (identical dedup + `(rank, UpdatedAt)`
   sort + limit) as the pre-change loop, plus a non-regressive `ctx.Err()` early
   break and the new (improved) warning. Re-ran
   `TestSearchObservationsAllParallelRollbackEnv` (`-race`) → PASS (parallel and
   sequential return the same 2 hits).

6. **Tool contract — VERIFIED unchanged.** `git diff --stat 5207977..2245ddc` touches
   exactly 2 files (`service.go` +193/-19, new test file). No `mem_*`/`code_*`/`web_*`
   registration touched. `SearchObservationsAll` / `SearchAllProjects` /
   `SearchObservationsScoped` signatures are all unchanged. `SearchAllProjects`
   (service.go:929-930, the `mem_search all_projects=true` backing at
   tools_memory.go:300) still just delegates to `SearchObservationsAll`. modernc
   driver untouched.

7. **Test quality — VERIFIED, with the M2/N1 caveats.**
   - `TestSearchObservationsAllParallel` (03.1) proves parallelism via a 30ms
     per-store latency marker: asserts `parallel < sequential/2` AND a wave ceiling
     (`3*waves*lat + 400ms`). On a 16-CPU host with 10 stores that is 1 wave, so the
     margin is large (parallel ≈ 1×lat vs sequential ≈ 10×lat) — not near the /2
     threshold; the `-count=5 -race` clean run in the report is believable. It also
     asserts merged content == sequential count and no `(id, project, title)` dup.
   - `TestSearchObservationsAllSemaphoreBound` (03.3) genuinely tracks concurrency:
     swaps `scopedSearchFunc` with a wrapper that increments/decrements a
     mutex-guarded `current` counter around the real `SearchObservationsScoped`
     call, records `maxSeen`, and asserts `maxSeen <= runtime.NumCPU()` and all 50
     stores searched. This is a real concurrency-count assertion, not a guess.
   - `TestSearchObservationsAllMissingStoreSkipped` (03.2) verifies the warning via
     `captureWarn` (fails if no warning mentions `ms-gamma`) — not just "results came
     back." Correct per 03.2; see M2 for the missing-vs-corrupt nuance.
   - TDD: the RED is a compile failure (tests reference `searchStoreLatency`,
     `scopedSearchFunc`, `warnFunc`, which don't exist pre-change) — a legitimate
     compile-time RED.

8. **Goroutine leaks — VERIFIED none.** Every worker ends in `defer wg.Done()` and
   reaches a terminal path on all branches (acquire-ok → search → send-or-ctx,
   acquire-timeout → return, acquire-ctx → return, search-err → return). The closer
   `wg.Wait()` then `close(results)`; the main `range` over `results` terminates on
   close. A cancelled `ctx` is observed at the acquire `select` and the send
   `select`, so in-flight/cancelled workers exit without leaking. The only unbounded
   construct is `time.After(5s)` per waiting worker, which is GC-eligible and bounded
   by the 5s constant — not a leak. No goroutine outlives the call.
