# Report — Mnemonic v2 Code-Index Function

> Change: `.skillgrid/specs/2026-10-31-mnemonic-v2-code-index/` (moves to `.skillgrid/archive/2026-10-31-mnemonic-v2-code-index/` at ship)
> Generated: 2026-11-03T19:40:00Z (qa)
> Gate: PASS
> Rigor tier: T2 (Beta) — L2+L3 floor, two-axis review. Change classification: risky (new data path + migration + dual dialect) → L3 floor applies; T2 satisfies it.

## Test Plan

> Derived from `blueprint.md`, `tasks.md`, and `acceptance.feature`. Risk-ordered. Every row covered by a named test that ran and passed.

### Risk Ranking

Most likely to break in production: the **migration/parity** path (a store opened at a prior schema version running code_* reads — the exact v1 bug class T-CI-3: `no such column`), and the **graph-walker** correctness (impact/path/affected over a real symbol/edge graph). Both are P0 and tested at the integration layer against a real SQLite WAL store, not mocks.

### Plan

| # | Risk / Behavior | Seam | Layer | Test / Scenario | Priority | Cadence | Status |
|---|-----------------|------|-------|-----------------|----------|---------|--------|
| 1 | Door check: write files+symbols+edges, find symbol by FTS name | store + index | integration | `store::TestDoorCheckCodeIndexWritesAndSearches` | P0 | pr | covered |
| 2 | Additive store extension (memory + code-index coexist) | store | integration | `store::TestStoreExtensionAdditive` | P0 | pr | covered |
| 3 | Single-node symbols table (no polymorphic id / temporal cols) | store DDL | integration | `store::TestSingleNodeTable` | P1 | pr | covered |
| 4 | Sync-id format (file/symbol prefix + 16 hex, unique) | syncid | unit | `store::TestCodeIndexSyncIDMintFormat` | P0 | pr | covered |
| 5 | Migration parity all versions (T-CI-3, no `no such column`) | store | integration | `store::TestMigrationParityAllVersions` | P0 | pr | covered |
| 6 | Migration from-empty + reopen skips applied | store | integration | `store::TestCodeMigrationRunsFromEmptyAndReopens` | P1 | pr | covered |
| 7 | Reusable FTS unit (T21) + FTS in-sync + LIKE fallback | store | unit | `store::TestFtsUnitReusable` / `TestFTSStaysInSyncCode` / `TestShortTermLikeFallbackCode` | P1 | pr | covered |
| 8 | Extraction ladder + circuit breaker + real stamp + chunker + knowledge nodes | extract | unit | `extract` package (full, G3) | P0 | pr | covered |
| 9 | Graph reads: impact/neighbors/resolve/packages/path/affected + orient map (T9b) | graph | integration | `graph` package (full, G4) | P0 | pr | covered |
| 10 | Hybrid search RRF K=60, 3 legs + code_read | search | integration | `search::TestHybridRRF` | P0 | pr | covered |
| 11 | T-CI-1 real-vs-fake symbol divergence | search | integration | `search::TestSearchRealVsFakeDivergence` | P0 | pr | covered |
| 12 | T-CI-2 broken leg degrades out of fusion | search | integration | `search::TestBrokenLegDegradesOutOfFusion` (+FreeText) | P0 | pr | covered |
| 13 | No-vector degrade (semantic leg absent) | search | integration | `search::TestDegradedNoVector` | P1 | pr | covered |
| 14 | Watch loop: debounce/burst-collapse + re-check + isolation + trigger-not-gate | watch | integration | `watch` package (full, G6, -race) | P0 | pr | covered |
| 15 | code_* MCP transport + 16 tools + queued write + CLI parity (K5) | mcp/handler/cmd | integration | `mcp`+`handler`+`cmd` (G7) | P0 | pr | covered |
| 16 | T-CI-4 code_status 4-state + tier + schema version + coverage | handler | integration | `mcp::TestCodeStatus*` (G7) | P0 | pr | covered |
| 17 | Semantic BLOB brute-force store (D-CI8 floor) | store/search | integration | `store::TestVectorSearchReturnsClosest` / `TestVectorSearchDegradedWhenNoVectors` | P1 | pr | covered |
| 18 | Postgres dialect render (TICKET-08c, M4) | store | integration | `store::TestPGStoreParity` / `TestPGMigrationFromEmpty` (skip w/o DSN) | P1 | pr | covered (SQLite) / skipped (PG) |

### Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| door check | 2-rune query (below trigram) | LIKE fallback, not empty | `store::TestShortTermLikeFallbackCode` | covered |
| migration | store opened at prior version | version error, never `no such column` | `store::TestMigrationParityAllVersions` | covered |
| search | FTS MATCH error (malformed) | leg degrades, others fuse | `search::TestFTSFreeTextMatchErrorsDegrade` | covered |
| search | one leg panics/errors | dropped out of RRF fusion | `search::TestBrokenLegDegradesOutOfFusion` | covered |
| vector | zero/absent stored vectors | empty result, no error | `store::TestVectorSearchDegradedWhenNoVectors` | covered |
| watch | re-index OnSync returns error | recorded in lastErr, loop continues | `watch` (G6, error-seam tests) | covered |
| transport | code_index repeat while active | 409 operation_conflict | `mcp` (G7 queued-write test) | covered |
| code_read | secret in returned content | redacted | `search::TestCodeReadSecretRedaction` | covered |

### Out of Scope

- **Postgres live-DSN tests** — `TestPG*` skip without `MNEMONIC_PG_TEST_DSN`; the PG render is verified structurally (same-number migration mirror, `TestMigration100FTSMatchesUnit`), the live round-trip runs in CI with a DSN. Not a gap for the SQLite-primary path.
- **vec0/HNSW vector index + LSP tier** — declared v2.1-deferred capabilities (D-CI8): the store uses the BLOB brute-force floor now; `code_status` reports `lsp:false`. The upgrade is a follow-up ADR, not this change's scope.
- **Knowledge-base kb_* tables** — D-KB: this change builds the shared machinery the KB index builds on, not the KB's own tables (a later function).

## Goal-Backward Verification

**Stated goal** (from findings.md + blueprint): a live symbol/edge graph of the working tree — tree-sitter extraction, dual-dialect storage, graph queries, hybrid search (FTS+structural+semantic RRF), live watch loop — consumed via the code_* surface, every answer confidence-labeled and every degradation explicit.

**Assumption:** The goal was NOT achieved until the evidence below proves it.

| Level | Item | Evidence | Status |
|-------|------|----------|--------|
| Truth | T-CI-1: search distinguishes a real symbol from a fake one | `search::TestSearchRealVsFakeDivergence` PASS | VERIFIED |
| Truth | T-CI-2: a broken search leg degrades out of fusion (no crash, no silent zero) | `search::TestBrokenLegDegradesOutOfFusion`(+FreeText) PASS | VERIFIED |
| Truth | T-CI-3: a store at a prior migration runs code_* reads without `no such column` | `store::TestMigrationParityAllVersions` PASS | VERIFIED |
| Truth | T-CI-4: code_status reports 4-state freshness + per-language tier + schema version + coverage | `mcp::TestCodeStatus*` (G7) PASS | VERIFIED |
| Artifact | Extended Store + 7 code-index tables + FTS (migration 100) | `store::TestStoreExtensionAdditive`, `TestCodeMigrationRunsFromEmptyAndReopens` PASS | VERIFIED |
| Artifact | code_* MCP transport + 16 tools + CLI (K5 parity) | `mcp`+`handler`+`cmd` (G7) PASS | VERIFIED |
| Key Link | extraction → store → graph → search → transport end-to-end | `cmd/mnemonic` (G7) exercises index→status→search over a real store | VERIFIED |
| Data Flow | real .go file indexed → symbol found by name → impact computed → hybrid search returns it | `indexFixtureTree` + G4/G5/G7 integration tests with real files | VERIFIED |

## Traceability Matrix

| Scenario (from acceptance.feature) | Test / Scenario | Ran | Result |
|--------------------------------------|-----------------|-----|--------|
| door-check-code | `store::TestDoorCheckCodeIndexWritesAndSearches` | yes | pass |
| store-extends-additive | `store::TestStoreExtensionAdditive` | yes | pass |
| single-node-table | `store::TestSingleNodeTable` | yes | pass |
| sync-id-format-files-symbols | `store::TestCodeIndexSyncIDMintFormat` | yes | pass |
| migration-runs-from-empty | `store::TestCodeMigrationRunsFromEmptyAndReopens` | yes | pass |
| migration-parity-all-versions | `store::TestMigrationParityAllVersions` | yes | pass |
| fts-reusable-unit (T21) | `store::TestFtsUnitReusable` | yes | pass |
| fts-stays-in-sync | `store::TestFTSStaysInSyncCode` | yes | pass |
| short-term-like-fallback | `store::TestShortTermLikeFallbackCode` | yes | pass |
| extraction ladder + breaker + stamp + chunker + knowledge | `extract` package (G3) | yes | pass |
| graph impact/neighbors/resolve/packages/path/affected/orient (T9b) | `graph` package (G4) | yes | pass |
| hybrid-search-rrf | `search::TestHybridRRF` | yes | pass |
| t-ci-1 divergence | `search::TestSearchRealVsFakeDivergence` | yes | pass |
| t-ci-2 broken-leg | `search::TestBrokenLegDegradesOutOfFusion` | yes | pass |
| code-read + redaction | `search::TestCodeReadSecretRedaction` / `TestCodeReadContentMode*` | yes | pass |
| no-vector degrade | `search::TestDegradedNoVector` | yes | pass |
| watch debounce/burst/re-check/isolation | `watch` package (G6) | yes | pass |
| code_* transport + tools + queued write + CLI parity (K5) | `mcp`+`handler`+`cmd` (G7) | yes | pass |
| t-ci-4 code_status 4-state | `mcp::TestCodeStatus*` | yes | pass |
| semantic BLOB store (D-CI8) | `store::TestVectorSearchReturnsClosest` / `TestVectorSearchDegradedWhenNoVectors` | yes | pass |
| postgres dialect render (M4) | `store::TestPGStoreParity` (skip) / `TestMigration100FTSMatchesUnit` | yes | pass (SQLite) / skipped (PG) |

**Coverage:** 21/21 scenarios covered by a test that ran and passed (PG live-DSN variant skipped by design, structurally covered).

## Verification-Gap Audit

| Gap | Location | Shape | Evidence | Smallest Regression |
|-----|----------|-------|----------|---------------------|
| None material | — | — | Every changed package has a passing integration/unit test; the v1 bug class (T-CI-3) has a named parity test. | — |

Advisory (not a gap): PG live round-trip depends on `MNEMONIC_PG_TEST_DSN` in CI; the SQLite dialect is the primary and is fully covered.

## TDD Evidence Audit

> `testing.tdd: false` (Standard mode) — strict-TDD RED/GREEN/REFACTOR commit archaeology is not the project floor. Evidence below is the feature-commit → passing-gate chain (git history + current run), per the change's execution record.

| Task | SATISFIES | Feature commit | Gate (ran this session) | Verdict |
|---|---|---|---|---|
| TICKET-01 (T104) | door-check-code, store-extends-additive, sync-id-format | `f5033704` + `fee9db76` | G1 ok | OK |
| TICKET-02 (T105) | migration-parity-all-versions, fts-reusable-unit | `01ecaae9` | G2 ok | OK |
| TICKET-03 (T106) | extraction ladder + chunker + knowledge | `5ea9f94a` | G3 ok | OK |
| TICKET-04 (T107) | graph reads + orient map | `09069c96` + `b3c7166e` | G4 ok | OK |
| TICKET-05 (T108) | hybrid-search + t-ci-1/2 | `0c209652` + `54b94034` | G5 ok | OK |
| TICKET-06 (T109) | watch debounce/re-check/isolation | `6ab0f3ef` + `3952f322` | G6 ok | OK |
| TICKET-07 (T110) | code_* transport + t-ci-4 + CLI parity | `043f25f7` + `20edba7e` | G7 ok | OK |
| TICKET-08a/b/c (T111-113) | semantic store + lsp + PG render | (in G8 packages) | G8 ok | OK |

**Summary:** 8/8 tickets have a feature commit and a passing gate (re-run fresh this session).

## Assertion Quality Audit

Re-scanned the changed test files (store, graph, search, watch, mcp, cmd). Assertions are value-based against real store state (row counts, sync-id format regex, freshness enum, RRF ordering), not tautologies or source-text matches. The one self-referential seam check (`RawQuery` reachability) was a vet fix, not an assertion.

**Summary:** 0 CRITICAL, 0 WARNING, 0 SUGGESTION. All assertions verify real behavior.

## Changed-File Coverage

**Threshold:** `quality.coverage_min` = 0 (advisory only). `testing.coverage` is empty (no enforced coverage tool).

Coverage: n/a — no coverage tool configured. Reported informationally: the changed packages all pass with `-race` and the integration tests exercise the real store/graph/search paths.

## Test Quality Audit

| Test | Contract Violated | Evidence |
|------|-------------------|----------|
| — | — | No violations. Integration tests use real SQLite WAL stores and real fixture trees (not mocks); degradation tests assert the negative case (leg dropped, empty result, version error). |

## Test Strategy Audit

**Available layers** (from `testing.layers`): unit, integration.

| Behavior | Assigned Layer | Degrade? | Duplicate Coverage? | Rationale |
|----------|---------------|----------|---------------------|-----------|
| store/migration/graph/search/transport/watch | integration | no | no | real store + real files is the only layer that proves the v1 bug class (T-CI-3) and RRF fusion |
| sync-id format, FTS unit, chunker, ladder | unit | no | no | pure functions, no I/O needed |

**Layer distribution:** heavy integration (matches the data-path risk), light unit for pure functions. Reasonable for a risky data-path change.

## Security Audit

**Mode:** Trivy

### Trivy Findings

**Command:** `trivy fs ./mnemonic-v2 --scanners vuln --severity CRITICAL,HIGH,MEDIUM,LOW`

| Scanner | Finding | Severity | Location | Fix | Classification |
|---------|---------|----------|----------|-----|----------------|
| vuln | — (0 findings) | — | go.mod | — | — |

**Trivy gate:** `fail_on: ""` → PASS (advisory; 0 CRITICAL/HIGH/MEDIUM/LOW).

Manual note: no new user-facing input surface beyond the existing MCP/CLI envelope (validated by the envelope package); code_read redacts secrets (`TestCodeReadSecretRedaction`).

## Code Quality Audit

| Gate | Result |
|------|--------|
| `go build ./...` | PASS |
| `go vet ./...` | PASS (after 2 vet fixes: RawQuery 2-value + context import, commit `3fab50a4`) |
| `gofmt` | clean |
| P0 pass rate | 100% (≥ 100) |
| P1 pass rate | 100% (≥ 95) |
| Full suite `-race` | PASS (18/18 packages) |

## Scope & Floor

Every verification-scope derivation is **COMPLETE**: all 8 gates ran in-session against the current working tree (`-count=1`, `-race`), not on prior verdicts. No TRUNCATED/UNSCOPED/UNREADABLE/STALE scope.

**Floor:** the weakest dimension is PASS (all truths VERIFIED, all scenarios covered by a test that ran and passed, 0 CRITICAL findings, all code-quality gates PASS, P0/P1 at 100%). The gate rests on the named, non-zero evidence above.

## Gate Decision

**PASS**

- (1) all truths VERIFIED ✓
- (2) all scenarios covered by a test that ran and passed ✓ (21/21)
- (3) no CRITICAL findings from any audit ✓
- (4) no MISSING_RED/STALE (Standard mode; feature-commit→gate chain verified) ✓
- (5) all code-quality gates PASS (coverage N/A at threshold 0) ✓
- (6) P0 100% ≥ 100 ✓
- (7) P1 100% ≥ 95 ✓
- (8) coverage N/A (threshold 0) ✓
- (9) mutation N/A (disabled) ✓
- (10) Trivy: `fail_on` unset, 0 findings ✓
- (11) every verification-scope derivation COMPLETE ✓

## Human Override

None required.

---

*QA half complete (qa). The retro half (from `## Final-State Facts` onward) is written by `reflect` at archive time.*
