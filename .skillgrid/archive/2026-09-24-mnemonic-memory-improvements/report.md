# Report — Mnemonic memory improvements

> Change: `.skillgrid/specs/2026-09-24-mnemonic-memory-improvements/`
> Generated: 2026-10-02T11:40:00+02:00 (qa)
> Gate: PASS
> Tier: T2 (blueprint). Floor: L3 (classification risky: migration + MCP contract). `testing.tdd: false`.

## Test Plan

> Derived from `blueprint.md`, `tasks.md`, and `acceptance.feature`.

### Risk Ranking

The load-bearing risks are a private row leaking through the new blend, keyword order moving when no embedder is active, and a migration that never creates `entity_aliases` on a store that already recorded `045`. Those are what the plan tests first.

### Plan

| # | Risk / Behavior | Seam | Layer | Test / Scenario | Priority | Cadence | Status |
|---|-----------------|------|-------|-----------------|----------|---------|--------|
| 1 | Private row hidden from another reader | SearchOwnerScopedBlend | integration | memory/search_blend_test.go::TestOwnerScopedBlendHidesPrivate | P0 | pr | covered |
| 2 | No embedder keeps BM25 and matched_via=keyword | mem_search + blend | integration | TestOwnerScopedBlendKeywordFloor, mcp/tools_memory_signals_test.go::TestMemSearchSignalsKeyword | P0 | pr | covered |
| 3 | Empty query is a tool error | handleMemSearch | integration | mcp/tools_memory_retrieval_test.go::TestBudgetedRetrievalMCP | P0 | pr | covered |
| 4 | Embedder on fuses both legs and signals stay in [0,1] | SearchOwnerScopedBlend | integration | TestOwnerScopedBlendHybrid | P0 | pr | covered |
| 5 | Embedder error stays on the keyword floor | mem_search | integration | TestOwnerScopedBlendEmbedderError, mcp/tools_memory_signals_test.go::TestMemSearchEmbedderError | P0 | pr | covered |
| 6 | Hot retrieval_usage outranks a cold twin | Reinforcement | unit | memory/decay_test.go::TestReinforcementDecayRanksHotRow | P0 | pr | covered |
| 7 | Decay disabled keeps BM25 | SearchOwnerScopedBlend | integration | TestDecayDisabledKeepsBM25 | P0 | pr | covered |
| 8 | Identical query within 7 days skips the embedder | CachedEmbedQuery | integration | memory/query_cache_test.go::TestQueryCacheHit | P0 | pr | covered |
| 9 | Stale or other-model cache misses | CachedEmbedQuery | integration | TestQueryCacheMiss | P0 | pr | covered |
| 10 | Embedder error does not write query_cache | CachedEmbedQuery | integration | TestQueryCacheEmbedderError | P1 | pr | covered |
| 11 | Case-insensitive alias resolves; unknown alias is empty | LookupAliases | integration | codeindex/aliases_test.go::TestEntityAliasLookup | P0 | pr | covered |
| 12 | Alias is written at index time and prepended | SymbolFTS | integration | TestEntityAliasIndexedOnWrite, TestEntityAliasPrepended | P1 | pr | covered |
| 13 | 045 and 046 apply once | store.Open | integration | store/migrations_045_test.go::TestMigration045 | P0 | pr | covered |
| 14 | Compact hook writes one continuity observation | RunHook compact | integration | memory/compact_hook_test.go::TestCompactHookSaves | P0 | pr | covered |
| 15 | Compact past 3s returns a result | RunHook compact | integration | TestCompactHookFailOpen | P0 | pr | covered |
| 16 | Absent decay YAML stays enabled; explicit false stays off | config.Load | unit | config/load_test.go::TestLoadDecayConfig | P1 | pr | covered |
| 17 | Immunity freezes the half-life term | Reinforcement | unit | TestDecayImmunityFreezesHalfLife | P1 | pr | covered |
| 18 | Equal decay factors keep order | rankByDecay | unit | TestEqualDecayKeepsOrder | P1 | pr | covered |

### Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| keyword-floor-search | MNEMONIC_EMBED unset | matched_via keyword, signals.vector 0 | TestMemSearchSignalsKeyword | covered |
| keyword-floor-search | other owner's private row | absent | TestOwnerScopedBlendHidesPrivate | covered |
| keyword-floor-search | no query | tool error | TestBudgetedRetrievalMCP | covered |
| hybrid-rrf-signals | both legs and vector-only | hybrid and vector | TestOwnerScopedBlendHybrid | covered |
| hybrid-rrf-signals | EmbedQuery error | keyword floor, no cache row | TestOwnerScopedBlendEmbedderError, TestQueryCacheEmbedderError | covered |
| reinforcement-decay | usage 20 vs 0, 40 days | hot row first | TestReinforcementDecayRanksHotRow | covered |
| reinforcement-decay | enabled false | BM25 order | TestDecayDisabledKeepsBM25 | covered |
| query-embedding-cache | same model within 7 days | inner called once | TestQueryCacheHit | covered |
| query-embedding-cache | 8 days or other model | inner called again | TestQueryCacheMiss | covered |
| entity-aliases | "The Renderer" / "missing" | qualified name / no rows | TestEntityAliasLookup | covered |
| compact-hook | session present | topic_key compaction/<session> | TestCompactHookSaves | covered |
| compact-hook | work exceeds 3s | result returned | TestCompactHookFailOpen | covered |

### Out of Scope

- C6 auto-learning observer. No task.
- `hybrid_search` still calls `EmbedQuery` directly and does not use `CachedEmbedQuery`.
- All-projects `mem_search` is keyword-stamped only. It is not the hybrid path.
- Trivy `fail_on` is empty, so vulnerability findings are advisory and were not a blocking scan this run.

## Goal-Backward Verification

**Stated goal** (from briefing.md): `mem_search` explains and re-ranks hits, query embeddings are cached, code-index aliases resolve, and a continuity note is saved before compaction.

**Assumption:** The goal was NOT achieved until the evidence below proves it.

| Level | Item | Evidence | Status |
|-------|------|----------|--------|
| Truth | No embedder: BM25, matched_via=keyword, signals in [0,1] | TestOwnerScopedBlendKeywordFloor, TestMemSearchSignalsKeyword | VERIFIED |
| Truth | Embedder on: hybrid or vector, signals in [0,1] | TestOwnerScopedBlendHybrid | VERIFIED |
| Truth | Reader B does not see reader A's private row | TestOwnerScopedBlendHidesPrivate | VERIFIED |
| Truth | Embedder error returns the FTS leg as keyword | TestOwnerScopedBlendEmbedderError, TestMemSearchEmbedderError | VERIFIED |
| Truth | Hot usage outranks a cold twin; disabled decay keeps BM25 | TestReinforcementDecayRanksHotRow, TestDecayDisabledKeepsBM25 | VERIFIED |
| Truth | Immunity freezes half-life at importance ≥ 4 or usage ≥ 3 | TestDecayImmunityFreezesHalfLife | VERIFIED |
| Truth | Cache hit within 7 days; miss when stale or other model | TestQueryCacheHit, TestQueryCacheMiss | VERIFIED |
| Truth | Alias hit and miss | TestEntityAliasLookup, TestEntityAliasPrepended | VERIFIED |
| Truth | Compact saves one note and fails open | TestCompactHookSaves, TestCompactHookFailOpen | VERIFIED |
| Truth | C6 has no task | briefing out-of-scope; no code added | VERIFIED |
| Artifact | decay.go, search_blend.go, query_cache.go, aliases.go, 045 + 046 | files present; tests open the store and call the functions | VERIFIED |
| Key Link | handleMemSearch uses SearchOwnerScopedBlend for one project | TestMemSearchSignalsKeyword returns signals | VERIFIED |
| Key Link | MCP drops EmbedQuery error before the blend | TestMemSearchEmbedderError: external embedder with no base URL, matched_via=keyword, query_cache rows = 0 | VERIFIED |
| Key Link | Decay on only when config says so | TestLoadDecayConfig, TestDecayDisabledKeepsBM25; service open calls SetDecay | VERIFIED |
| Key Link | SymbolFTS prepends alias hits | TestEntityAliasPrepended | VERIFIED |
| Data Flow | query text is not stored in query_cache | TestQueryCacheHit assertQueryTextNotStored | VERIFIED |

## Traceability Matrix

| Scenario (from acceptance.feature) | Test / Scenario (from test plan) | Ran | Result |
|--------------------------------------|----------------------------------|-----|--------|
| mem_search without an embedder is keyword only | TestOwnerScopedBlendKeywordFloor, TestMemSearchSignalsKeyword | yes | pass |
| owner blend hides another owner's private row | TestOwnerScopedBlendHidesPrivate | yes | pass |
| mem_search without a query is rejected | TestBudgetedRetrievalMCP | yes | pass |
| mem_search with an embedder returns hybrid signals | TestOwnerScopedBlendHybrid | yes | pass |
| embedder error degrades to keyword | TestOwnerScopedBlendEmbedderError, TestMemSearchEmbedderError | yes | pass |
| high retrieval_usage outranks a cold twin | TestReinforcementDecayRanksHotRow | yes | pass |
| decay disabled keeps BM25 order | TestDecayDisabledKeepsBM25 | yes | pass |
| second identical query is a cache hit | TestQueryCacheHit | yes | pass |
| stale or other-model cache misses | TestQueryCacheMiss | yes | pass |
| alias lookup finds the symbol | TestEntityAliasLookup | yes | pass |
| unknown alias falls through | TestEntityAliasLookup, TestEntityAliasPrepended | yes | pass |
| compact hook saves continuity | TestCompactHookSaves | yes | pass |
| compact hook swallows a slow save | TestCompactHookFailOpen | yes | pass |

**Coverage:** 13/13 scenarios covered by a test that ran and passed.

G2's named test `TestMemSearchRequiresQuery` does not exist. The same assertion lives in `TestBudgetedRetrievalMCP`. G3's `TestMemSearchSignalsHybrid` does not exist. Hybrid fusion is `TestOwnerScopedBlendHybrid`; the MCP JSON stamp is the same path `TestMemSearchSignalsKeyword` already exercises.

## Verification-Gap Audit

Changed behavior enumerated from the commits `b9ab00de`..`30fac16b` on this change (decay, blend, config, query cache, compact, aliases, embedder-error tests).

- MCP embed-error drop is covered by `TestMemSearchEmbedderError` (re-verification). A failed `EmbedQuery` leaves `query_cache` empty and the hit on the keyword floor.
- `entity_aliases` is migration `046`, not a second statement inside the already-committed `045`. Fresh stores and stores that already recorded the query-cache-only `045` both get the table. `TestMigration045` asserts both filenames once.

## TDD Evidence Audit

| Task | SATISFIES Scenario | RED Evidence | GREEN Evidence | Verdict |
|------|--------------------|--------------|----------------|---------|
| TICKET-01 decay | high retrieval_usage outranks a cold twin | `b9ab00de` lands decay.go and decay_test.go together | TestReinforcementDecayRanksHotRow PASS | MISSING_RED |
| TICKET-02 blend | keyword floor, hybrid, private hide | `aeefb5e7` lands search_blend.go and tests together | named blend tests PASS | MISSING_RED |
| TICKET-03 config | decay disabled keeps BM25 | `596958c9` lands config and TestDecayDisabledKeepsBM25 together | PASS | MISSING_RED |
| TICKET-04 cache | cache hit and miss | `f5b25b22` lands query_cache.go and tests together | TestQueryCacheHit, TestQueryCacheMiss PASS | MISSING_RED |
| TICKET-05 aliases | alias hit and miss | `a0d05555` lands aliases.go and tests together | TestEntityAliasLookup PASS | MISSING_RED |
| TICKET-06 compact | compact save and fail-open | `5da7fb13` lands hook and tests together | TestCompactHookSaves, TestCompactHookFailOpen PASS | MISSING_RED |

`testing.tdd: false`. Combined commits are the project baseline. These are WARNING (provenance not separable), not CRITICAL. GREEN evidence is this run. `30fac16b` added the embedder-error tests after the behavior existed; that is the same combined-commit pattern, recorded here rather than as a new CRITICAL.

## Test Quality Audit

| Test | Contract Violated | Evidence |
|------|-------------------|----------|
| TestOwnerScopedBlendHidesPrivate | none | asserts zero hits for the other owner |
| TestOwnerScopedBlendEmbedderError | none on the blend contract | would fail if an empty vector took the hybrid leg while MNEMONIC_EMBED=1 |
| TestQueryCacheEmbedderError | none | asserts the inner error and zero cache rows |
| TestEntityAliasLookup | none | hit and miss in one test |
| TestCompactHookFailOpen | none | asserts a result rather than a timeout error |

P0 triangulation: keyword floor has a service test and an MCP test. Private-hide has a single test. That single-test P0 is a SUGGESTION, not a gap that leaves the scenario uncovered.

## Security Audit

No new trust boundary. `query_cache` stores a hash and a vector, not the query text (`TestQueryCacheHit`). Owner visibility is the existing filter (`TestOwnerScopedBlendHidesPrivate`).

Trivy `fail_on` is empty (advisory). A full `trivy fs` scan was not run this gate. Verdict for the blocking security gate: N/A.

`go vet` on `./internal/mnemonic/memory/` reports a pre-existing context leak at `budget.go:114`. That file is not part of this change. Recorded as SUGGESTION.

## Code Quality Gate

| Gate | Command | Threshold | Result |
|------|---------|-----------|--------|
| Coverage | unset | 0 | N/A |
| Mutation | unset | 0 | N/A |
| Lint | unset | — | N/A |
| P0 | named tests above | 100 | PASS (100) |
| P1 | named tests above | 95 | PASS (100) |
| Trivy | fail_on empty | advisory | N/A |
| Format | gofmt -w on the new tests | clean | PASS |

Run evidence (2026-10-02, `skillgrid-cli`, exit 0):

- `go test` memory, mcp, config, codeindex, store with the acceptance name filter.
- `go test` memory+mcp including `TestBudgetedRetrievalMCP`, `TestOwnerScopedBlendEmbedderError`, `TestQueryCacheEmbedderError`.

## Verification Scope

- Traceability matrix: SCOPE: COMPLETE (every scenario in acceptance.feature was read).
- Verification-gap audit: SCOPE: COMPLETE (commits for this change, not a truncated since-window).
- State drift: `state-drift-check.mjs` run directly — DRIFT: none, SCOPE: COMPLETE, exit 0.
- Skill size budget: `skill-size-budget.mjs check` — all 69 skills within ceiling, exit 0.
- Structure drift: skipped. `qa-gate.mjs` looks for scripts under `.agents/skills/qa/scripts`; they live under `.agents/skills/verification/qa/scripts`. The checks were run at the real path.
- Composite: SCOPE: COMPLETE.
- STALE: none for the prior scenarios. This re-verification adds `TestMemSearchEmbedderError` and rests on this run.

## Floor

| Dimension | Verdict |
|---|---|
| Goal-backward verification (weakest truth) | VERIFIED |
| Traceability (weakest scenario) | COMPLIANT |
| Verification-gap audit | none |
| TDD evidence (weakest ticket) | MISSING_RED (reclassified WARNING; strict-TDD off) |
| Test quality audit | SUGGESTION |
| Code-quality gates (weakest gate) | PASS |
| Security audit | N/A |
| Verification scope (composite) | COMPLETE |

**FLOOR:** PASS-eligible. Every truth is VERIFIED, every scenario ran and passed, and the deterministic checks that exist are clean. `MISSING_RED` stays a provenance note because `testing.tdd` is false; it does not cap the gate.

## Findings

### CRITICAL (must fix before merge)

- None.

### WARNING (should fix before archive)

- Six tickets have no separable RED commit. Strict-TDD is off (`testing.tdd: false`), so this is provenance, not a missing behavior test. Same classification as the session-events PASS gate. It does not cap this verdict.

### SUGGESTION (nice to have)

- `memory.Service` comment says a zero `decayCfg` is treated as the default-on config. `SearchOwnerScopedBlend` reranks only when `Enabled` is true. Production open calls `SetDecay` from config, whose absent section is enabled.
- `go vet` context leak in `budget.go:114` is pre-existing and outside this change.
- `qa-gate.mjs` looks for its helpers in `.agents/skills/qa/scripts`. The files are in `.agents/skills/verification/qa/scripts`. Invoked there, state drift and the size budget both exit 0.
- Private-row hiding has one test, not a second input shape.

## Gate Decision

**Verdict: PASS**

Re-verification (2026-10-02): `TestMemSearchEmbedderError` enters `handleMemSearch` with `MNEMONIC_EMBED=1` and an external embedder that has no base URL. The tool returns one hit, `matched_via=keyword`, `signals.vector=0`, and `query_cache` stays empty. Acceptance packages re-ran exit 0 (memory 5.3s, mcp 4.1s, config 0.7s, codeindex 5.9s, store 3.8s). State drift: none. Skill size budget: 69 skills within ceiling.

| # | Criterion | Result |
|---|-----------|--------|
| 1 | all truths VERIFIED | 10/10 VERIFIED, including the embedder-error key link |
| 2 | all scenarios covered by a test that ran and passed | 13/13 |
| 3 | no CRITICAL findings | yes |
| 4 | no MISSING_RED (strict-TDD) | reclassified; `testing.tdd: false` |
| 5 | code-quality gates PASS or N/A | yes |
| 6 | P0 pass rate ≥ 100 | 100 |
| 7 | P1 pass rate ≥ 95 | 100 |
| 8 | coverage ≥ min | N/A (threshold 0) |
| 9 | mutation ≥ min | N/A |
| 10 | Trivy ≥ fail_on | N/A (fail_on empty) |
| 11 | verification scope COMPLETE | yes |

## Human Override

None.

## Rulings

- `entity_aliases` is `046_entity_aliases.sql`. `045` was already committed as `query_cache` only, and the runner applies each filename once. Cost if wrong: an extra migration file versus a store that already applied `045` never gaining the alias table.
- MISSING_RED on combined commits is WARNING because `testing.tdd` is false. Cost if wrong: a later audit treats these tests as if they had been proven red.
