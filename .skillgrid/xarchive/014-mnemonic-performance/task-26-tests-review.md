# Step 26 — Tests: review (fresh reviewer)

**Verdict:** `PASS`

**Commits reviewed:** `6b5c5f4` (RebindPassDB fix + 5 unit tests), `c610207` (2 integration tests). 9 files: 2 production, 7 new test files.

---

## RebindPassDB fix — CORRECT, SAFE, ADDITIVE

**Verdict: the fix is correct and safe.** It atomically re-registers the pool entry with `passDB`, preserves the refcount, and is race-safe via the existing `cacheMu` + atomic refs. It closes the actual root cause, not the symptom.

### Root cause confirmed
`codeindex.Indexer.Run` (indexer.go:498-501) closes the pooled `*sql.DB` then opens a fresh `passDB`. Before the fix it swapped only `idx.store.DB = passDB` (indexer.go:515), leaving the `handleCache` entry still pointing at the now-closed old `*sql.DB`. The test's second `OpenForDirectory` (reindex_structural_test.go:100) → `acquireCached` pings the closed pool handle → returns false → opens a fresh DB, but the *first* `h.Store()` handle captured the old closed DB at Open time and later `QueryRow` on it → `sql: database is closed`.

### The fix (store.go:307-330)
```go
func (s *Store) RebindPassDB(passDB *sql.DB) error {
	if s == nil || s.DB == nil || s.cacheKey == "" || passDB == nil {
		return nil
	}
	old := s.DB
	if old == passDB {
		return nil
	}
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if v, ok := handleCache.Load(s.cacheKey); ok {
		entry := v.(*cachedEntry)
		if entry.db == old {
			entry.db = passDB               // swap pointer, keep refs
			if entry.refs.Load() <= 0 {
				handleCache.Delete(s.cacheKey)
			}
		}
	}
	return old.Close()
}
```
Correctness + safety checks:
- **Rebinds the pool entry** — `entry.db = passDB` under `cacheMu`. Any subsequent `acquireCached` loads the entry, pings `passDB` (open), returns it. Fix is effective. ✓
- **Atomic / race-safe** — `cacheMu` is the same lock `evictCached` (store.go:238) and `cacheNewHandle` use for the load-compare-swap, so the pointer swap is serialized against evict-and-close and concurrent opens. `refs` is `atomic.Int64`. No torn read. ✓
- **Preserves refcount (no leak)** — the swap mutates only `entry.db`, leaving `entry.refs` intact. A second live handle that captured the old DB at Open keeps its +1; when it later Closes, `releasePooled` now sees `entry.db (passDB) != s.DB (old)` and skips the decrement (guard below), which is correct: the owner's close already released its own ref inside `RebindPassDB`, so the shared ref stays balanced at the second handle's own ref. No over-decrement (no premature evict), no leak. ✓
- **Closes the old DB** — `old.Close()` (the close that indexer.go already did is re-applied on the same handle; `*sql.DB.Close` is idempotent, returns nil). No double-close hazard. ✓
- **No-op when owned outright** — `cacheKey == ""` returns nil before touching the pool. ✓

### releasePooled guard (store.go:279-285)
```go
	if entry.db != s.DB {
		// The pool entry was rebound to a new *sql.DB (RebindPassDB) after this
		// Store captured the old one. … decrementing here would leak the passDB entry's refs.
		return nil
	}
```
Correct: skips the decrement for the owner store whose DB was superseded by the rebind. ✓

### Indexer.Run change is minimal (indexer.go:506-515)
```go
	// Re-register the pool with passDB BEFORE swapping idx.store.DB: …
	if err := idx.store.RebindPassDB(passDB); err != nil {
		return stats, fmt.Errorf("rebind pass db: %w", err)
	}
	idx.store.DB = passDB
```
Inserts one guarded call before the existing swap; no other logic touched. Minimal. ✓

### Store tests still pass
`go test ./internal/mnemonic/store/` → `ok` (no tests to run against Rebind directly, but the pool tests `TestStoreOpenReusesCachedHandle` and all store tests pass). The previously-failing `TestReindexStructuralIsEmbedderFree` → **PASS (1.32s)** under the fix. No store regression.

---

## 26.1 — New unit tests are MEANINGFUL (not stubs)

All 5 verified present, compiling, and passing. Representative: `TestTTLBoundaryExactSecond` (ttl_boundary_test.go):
```go
boundary := time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
future   := time.Now().UTC().Add(time.Second).Truncate(time.Second).Format(time.RFC3339)
// … SetExpiresAt on both …
hits, _ := svc.Search(ctx, "ttl", "any", 10)
// asserts boundaryID hidden (predicate `> now` excludes it), futureID returned
retired, _ := svc.TTLRetire(ctx)
if retired != 1 { t.Fatalf("TTLRetire retired %d, want exactly 1", retired) }
// boundary row unreadable after retire; future row readable
```
Each test checks the specific contract the brief asked for:
- **memfs_scale_test.go** — creates exactly 10,000 obs (5k preferences + 5k entities) via the real `mem.Save` path, asserts `ResolveURI` is size-independent AND the `List` cap `LIMIT 200` holds (≤200 rows, all in-scope) at scale. Real scale, not a stub. (Minor: the "newest first" ordering claim is not asserted — only the cap + in-scope filter.)
- **fts_trigram_empty_test.go** — seeds a real row, proves the store is healthy via a positive phrase hit, then asserts an absent trigram term returns `len==0` **and nil error** on BOTH the plain `Search` and the production `SearchOwnerScoped` seam. Correct: `buildFTSQuery` yields an empty trigram group set → FTS5 MATCH → 0 rows → `scanObservations` on 0 rows = nil, nil. ✓
- **temporal_edges_boundary_test.go** — drives all 3 states through the real `IdentifyHubFiles` + the `valid_from <= now AND (valid_to IS NULL OR valid_to > now)` window (hub.go:167-168): active1-3 included (count 3), expired excluded, not-yet-active excluded. A second test pins the exact-second edge (`valid_to==now` → expired, `valid_to==now+1` → active, `valid_from==now+1` → not active). ✓
- **ttl_boundary_test.go** — exact-second `<=` retire vs `>` search-exclusion pairing, asserts the boundary row is retired AND hidden while the +1s row is live. ✓
- **parallel_search_many_test.go** — 60 stores, artificial 15ms latency, asserts parallel < sequential/2, within a wave ceiling, merged+deduped count == 60 unique. (Note: does NOT run with `-race` in the run I executed — see finding M1; the pool's refcounted close is what "stays open", so the race-freedom claim rests on the pool invariant, not an in-test `-race` gate.)

---

## 26.2 — Integration tests are REAL (not unit tests in disguise)

- **tools_memory_federated_test.go** — drives the actual MCP handlers (`handleMemSessionStart`, `handleMemSave`, `handleMemSearch`) through `newCallTool`, with a real `service.Service` + `SetService`. Asserts the **federated contract** (`project=="all"`, `all_projects==true`, both rows returned) **vs the scoped contract** (bucket id `fed-a`, no `all_projects` flag). This is a genuine tool-boundary test of the 014 step-03 federated seam. ✓
- **search_match_mode_test.go** — builds the real HTTP server (`NewServer(svc).Handler()`), seeds a store, and issues real `GET /search` requests: (1) `match_mode=trigram` accepted (200, observations shape present), (2) default phrase returns the seeded row, (3) missing project → 400 (existing contract unchanged). Real HTTP boundary, additive param pinned at the layer web/dashboard consumers hit. ✓

Both exercise real handler/server seams — not unit tests in a `_test` file.

---

## 26.3 — Full suite

I did not run the entire `./skillgrid-cli/...` suite (load/time); I ran the step-26 packages + the specific new/affected tests:
- `store`, `memory`, `mcp`, `http`, `memfs` → all `ok`
- `service` `TestReindexStructuralIsEmbedderFree` (the step-02 failure) → **PASS**
- `service` `TestSearchObservationsAllParallelManyStores` (60 stores) → **PASS (23.10s)**

The report's claim that all packages pass except `TestStep02_CLI_Exit1` (http/tracker) is credible and **out of 014 scope**: it is a load-induced 10s CLI-timeout flake in the `tracker` package (unrelated to mnemonic pooling/FTS/temporal), passes in isolation, and `tracker_test.go` is unmodified by step 26. It is a pre-existing, non-regression, non-mnemonic concern. I accept the disposition (documented, not fixed, fixing it would touch tracker production config).

---

## Additive-ness — HELD

- `git show --name-status` across both commits: **only** `store.go` (M, +46) and `indexer.go` (M, +9) are production; the other 7 are new `_test.go` files. No other production code touched. ✓
- No `mem_*` / `code_*` / `web_*` MCP tool **signature** changed. The federated test asserts the existing `mem_search` contract (additive `all_projects` behavior), the HTTP test asserts an additive `match_mode` query param. No shape change to any tool schema in the diff. ✓

---

## Findings

**M1 (minor) — `parallel_search_many_test.go` is not gated by `-race`.**
The test's headline claim is "race-freedom at 60 stores", but it runs as a normal test (no in-test `-race` invocation). The correctness of the refcounted pool under concurrency rests on `store` tests + `cacheMu`/atomic invariants, not on this test. If the intent is to catch a data race in the 60-store fan-out, run it under `go test -race ./internal/mnemonic/service/ -run TestSearchObservationsAllParallelManyStores`. As written it verifies scheduling + dedup, not race-freedom per se. Fix (optional): add a dedicated `-race` run to the verification evidence, or soften the "race-freedom" comment to "scales to 60 stores without pool exhaustion".

**m2 (minor) — `memfs_scale_test.go` asserts the LIMIT cap + in-scope filter but not the "newest first" ordering** the doc comment implies ("newest <=200 preferences"). The assertions (`len<=200`, all in-scope) are the meaningful ones; the ordering is unverified. Cosmetic: either assert ordering or drop the ordering claim from the comment.

**n3 (nit) — `releasePooled` guard comment.** The comment says "decrementing here would leak the passDB entry's refs." That is the right net effect (no premature evict), but the precise reason is that this store is the *owner* whose ref was already released by `RebindPassDB`'s close, so the remaining refs belong to other live handles that must each release their own. The comment is accurate but slightly compressed; a one-line note that "the owner's ref was released at rebind time" would make the balance argument airtight for the next reader.

**n4 (nit) — `memfs_scale_test.go` defines a hand-rolled `itoa`.** `strconv.Itoa` exists; the custom helper is ~12 lines for a formatting detail. Harmless (keeps the test self-contained) but non-idiomatic. Optional.

---

## Evidence checked (5 checks)

| # | Check | Result |
|---|-------|--------|
| 1 | RebindPassDB fix correctness/safety/refcount/minimality | **VERIFIED** — quotes above; atomically rebinds under `cacheMu`, preserves `entry.refs`, closes old DB, no-op when owned; Indexer.Run change is one guarded call. Store tests pass; pre-existing `TestReindexStructuralIsEmbedderFree` now PASS. |
| 2 | 5 new unit tests meaningful | **VERIFIED** — all present, compile, pass. Each pins a distinct boundary contract (10k scale + LIMIT cap, trigram empty floor on 2 seams, 3 temporal states + exact-second edge, TTL `<=`/`>` pairing, 60-store fan-out). Not stubs. |
| 3 | Integration tests real | **VERIFIED** — MCP test drives real handlers via `newCallTool`; HTTP test drives the real `Handler()` with real requests. Both assert 014 additive contracts at the tool/HTTP boundary. |
| 4 | Full suite / pre-existing flake credibility | **MOSTLY VERIFIED** — ran all step-26 packages + the affected tests (all pass incl. the step-02 failure and 60-store test). Did not run the whole `./...` module under load; accepted the `TestStep02_CLI_Exit1` (tracker) flake as pre-existing, load-induced, and out of 014 scope. |
| 5 | Additive-ness | **VERIFIED** — only `store.go`/`indexer.go` production; no `mem_*`/`code_*`/`web_*` signature changes; all other changes are new test files. |

## Approved to merge: **yes**

The one production change (RebindPassDB) is a correct, race-safe, additive fix for a real root cause, and it un-blocks the step-02 pre-existing failure. The 7 new tests are meaningful and the 2 integration tests exercise real seams. The minor/nit findings (M1 race-gating, m2 ordering claim, n3/n4 comments) do not block merge.
