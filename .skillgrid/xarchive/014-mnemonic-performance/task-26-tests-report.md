# Step 26 — Tests: Unit + integration coverage for all steps (01-25)

## 26.1 — Core packages (store, memory, service, config, embedder, codeindex)

**Run:** `go test ./internal/mnemonic/store/ ./internal/mnemonic/memory/ ./internal/mnemonic/service/ ./internal/mnemonic/config/ ./internal/mnemonic/embedder/ ./internal/mnemonic/codeindex/ -count=1`

**Baseline:** 5/6 packages pass; `service` failed on the known pre-existing
`TestReindexStructuralIsEmbedderFree` (`sql: database is closed`).

**Gaps identified + tests added** (the brief listed ~26 areas; the vast majority
already had solid coverage from steps 01-25. The genuinely thin areas, now
closed):

| Gap area | Test added | File |
|----------|-----------|------|
| TTL boundary (exact-second `<=` retire vs `>` search-exclusion) | `TestTTLBoundaryExactSecond` | `memory/ttl_boundary_test.go` |
| FTS trigram empty-results fallback (empty → not error) | `TestFTSTrigramEmptyResultsFallback` | `memory/fts_trigram_empty_test.go` |
| Parallel search 50+ stores (existing test covered 10) | `TestSearchObservationsAllParallelManyStores` (60 stores) | `service/parallel_search_many_test.go` |
| Temporal edges boundary (active / expired / not-yet-active window) | `TestTemporalEdgesBoundary`, `TestTemporalEdgeExactActiveBoundary` | `memory/temporal_edges_boundary_test.go` |
| memfs URI resolution with 10k+ observations (scale + LIMIT cap) | `TestMemURIResolutionWith10kObservations` | `memfs/memfs_scale_test.go` |

Already covered (no new test needed — verified present): store pooling concurrent
opens (`TestStoreOpenReusesCachedHandle`), embedder config-driven selection
(`TestEmbedderConfigDrivenSelection`), triple-store join (`TestTripleStoreCrossLinkQuery`),
improve() no regression (`TestImproveDisabledNoRegression`), session promotion dedup
(`TestSessionEndIdempotentPromotion`), export roundtrip (`TestExportImportRoundtrip`),
dream lock+rollback (`TestDreamLockPreventsConcurrent` / `TestDreamRollbackOnFailure`),
importance skew (`TestImportanceScoreComputation`), relations confidence
(`TestRelationConfidenceFiltering`), provenance immutability
(`TestProvenanceImmutability`), federated dedup (`TestFederatedQueryDedup`),
distill lock timeout (`TestDistillLockTimeoutAutoRelease`), memory types async
(`TestAsyncTwoPhaseCommit`), directory retrieval depth limit
(`TestDirectoryRetrievalDepthLimit`), snapshots auto-prune (`TestSnapshotAutoPrune`),
handoff staleness (`TestHandoffDeltaDynamic`), context envelope size limit
(`TestContextEnvelopeSizeLimit`), hub accuracy (`TestHubFileIdentification`),
skills hook timeout (`TestHooksOptInWithTimeout`), LLM extraction error handling
(`TestCapturePassiveLLMFailureFallsBackToRegex`).

**Final result:** all 6 packages `ok`.

## 26.2 — Integration tests (MCP, HTTP, CLI)

**Run:** `go test ./internal/mnemonic/mcp/ ./internal/mnemonic/http/ ./cmd/skillgrid/ -count=1`

**Baseline:** all 3 packages pass; broad existing coverage. The CLI subcommands
(expire, export, relations, provenance, dream, snapshot, handoff, context,
graph --risk, skills, hook, fs) already had dedicated integration tests
(`TestMemExpireCLI`, `TestMemExportCLI`, `TestMemRelationsCLI`, `TestMemProvenanceCLI`,
`TestMemSnapshotCLI`, `TestMemHandoffCLI`, `TestMemContextCLI`, `TestMemGraphRiskCLI`,
`TestMemSkillsAndHookCLI`, `TestMemLSCoexistsWithMemList`). memfs has no HTTP
endpoint (CLI-only surface — `mem fs`), so no HTTP test applies to it.

**Gaps identified + tests added** (the new 014 features at the tool boundary):

| Layer | Gap area | Test added | File |
|-------|----------|-----------|------|
| MCP | `mem_search` federated query (`all_projects=true`) | `TestMemSearchAllProjectsFederated` | `mcp/tools_memory_federated_test.go` |
| HTTP | `GET /search` new `match_mode` param (trigram) | `TestSearchMatchModeTrigram` | `http/search_match_mode_test.go` |

**Final result:** all 3 packages `ok`.

## 26.3 — Full suite `go test ./...` (module root `skillgrid-cli/`)

**Run:** `go test ./... -count=1`

**Result:** every package `ok` **except one pre-existing flaky failure**:
`TestStep02_CLI_Exit1` in `internal/mnemonic/http/tracker`.

- **Root cause:** the test's `backlog` CLI stub (`echo "boom" >&2; exit 1`) hits
  the tracker's 10s CLI timeout when the machine is saturated by the parallel
  full-suite run, so `List` returns a timeout error (`"backlog CLI timed out"`)
  instead of the exit-1 stderr surface the assertion expects.
- **Pre-existing + flaky, not a regression:** `tracker_test.go` was last modified
  in `e0321de4` (the dashboard change, before 014) and is unmodified by step 26
  (verified via `git diff`). It passes reliably in isolation
  (`go test ./internal/mnemonic/http/tracker/ -run TestStep02_CLI_Exit1` → PASS
  in ~0.76s) and the whole tracker package passes when not under full-suite load.
- **Disposition:** documented as a known pre-existing flaky failure. It is outside
  the 014 core packages and unrelated to the mnemonic performance change; fixing
  it (e.g. lengthening the tracker CLI timeout for the test) would touch the
  tracker package's production config, which is out of scope for 014.

## Pre-existing `TestReindexStructuralIsEmbedderFree` — FIXED

**Decision:** fixed (root cause was clear and the fix was small).

**Root cause:** `codeindex.Indexer.Run` closes the store's pooled `*sql.DB` after
the 005 commit and swaps in a fresh single-connection `passDB`
(`indexer.go:499-506`) because the store's `MaxOpenConns=1` pool deadlocks a
second connection once the committed tx holds the write lock. But the swap
replaced `idx.store.DB` only — the **pool entry** still pointed at the now-closed
old `*sql.DB`. Any other live handle for the same store (which cached the old
DB at `Open` time) kept using the closed pool handle → `sql: database is closed`
on the test's second `OpenForDirectory` + `QueryRow`.

**Fix (production, minimal, additive):**
1. `store.Store.RebindPassDB(passDB)` — atomically swaps the pool entry's DB
   pointer to `passDB` (inheriting the old entry's live reference count, so no
   refs are lost/leaked) and closes the old DB. No-op when the store owns its DB
   outright (cache disabled).
2. `codeindex.Indexer.Run` calls `RebindPassDB(passDB)` before the
   `idx.store.DB = passDB` swap.
3. `store.releasePooled` skips the ref decrement when the pool entry was rebound
   to a new DB after this store captured the old one (the owner's close already
   closed the old DB inside `RebindPassDB`).

`TestReindexStructuralIsEmbedderFree` now passes; no regression to the store
pooling tests, service tests, or codeindex tests (all re-run green).

## Commits

| Short SHA | Subject |
|-----------|---------|
| `6b5c5f4` | feat(mnemonic): close RED coverage gaps + fix pool rebind in Indexer.Run |
| `c610207` | feat(mnemonic): integration tests for MCP federated search + HTTP match_mode |

## Concerns

- **Flaky `TestStep02_CLI_Exit1`** (`http/tracker`): pre-existing, load-induced
  (10s CLI timeout under full-suite saturation). Passes in isolation. Out of
  scope for 014 — documented, not fixed. A future change to the tracker CLI
  timeout default (or a test-side timeout bump) would make it deterministic.
- The 26.1 brief's list of ~26 gap areas is mostly already-covered; only 5 areas
  were genuinely missing unit coverage and those are now closed. The remaining
  areas map to existing named tests (listed in 26.1 above).
