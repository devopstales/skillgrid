# Review — Mnemonic memory improvements

> Change: `.skillgrid/specs/2026-09-24-mnemonic-memory-improvements/`
> Generated: 2026-10-02T12:20:00+02:00 (parallel-code-review)
> Diff range: `aa0aa791be652c7f3ca14fbc5207ffbce644e8c5..4bb1eeb3f986f2890c3568f40ba7723c402d1197`
> Rigor tier: T2 change, escalated review (migration + MCP contract, ~3100 lines)
> Independence: Grade B per axis (fresh subagents, different model families from the implementer, same Cursor toolchain)

Specialists: Standards, Spec, Edge cases, Verification gaps, Security, Performance, then Red team. Accessibility skipped (no UI). `personas_test.go` and `setup/protocol_test.go` are outside this change.

## What Important means

- **Critical** — must fix before merge. A retracted or out-of-scope observation returned by `mem_search`, or a continuity note that copies another owner's private titles.
- **Important** — should fix before merge. A requirement the acceptance scenarios claim that the test does not actually exercise, or config layering that drops an explicit opt-out.
- **Minor** — smells, duplication, and performance redesigns. Capped below.

## Passes

### Standards

- **Worst issue:** production decay sort is `sortHitsByDecay`; the tests that claim to rank the hot row call the unused `rankByDecay`.
- **Findings:** 0 Critical / 2 Important / 5 Minor listed (6 further Minor collapsed)
- **Verdict:** met-with-fixes

### Spec

- **Worst issue:** acceptance gate G2 names `TestMemSearchRequiresQuery`, which does not exist, so that check exits 0 without running a test.
- **Findings:** 0 Critical / 2 Important / 1 Minor
- **Verdict:** met-with-fixes

13/13 acceptance scenarios have a behavior test. Briefing items the blueprint deferred (comment mining, `mem alias add`, `hybrid_search` cache, importance-filtered compact) are not defects against the blueprint.

### Security

- **Worst issue:** the vector leg returns superseded and out-of-scope observations.
- **Findings:** 0 Critical / 1 High / 0 Medium / 0 Low
- **Verdict:** secure-with-fixes

## Findings

### Critical

- [Security + Edge] `memory/search_blend.go:63-92` — `SearchByVector` and `canRead` do not apply the FTS leg's `invalid_at`, `expires_at`, or `scope` predicates, and `Get` returns superseded rows. A retracted observation with an embedding comes back as `matched_via=vector`. Drop those hits before fusion, and test a superseded row.

### Important

- [Red team] `memory/service.go:2037` via `skills.go:572` — `CompactionContext` loads `Recent`, which is project-wide, then `hookCompact` saves those titles. Another session's private titles can land in this session's continuity note. Limit the saved lines to that session.
- [Red team] `config/load.go:639` — `mergeDecay` always starts from `DefaultDecay()` and replaces the previous layer. A home file with `enabled: false` is undone by any repo `indexing.yaml` that omits `mnemonic.decay`. Overlay onto the already-merged value.
- [Verification + Standards] `memory/decay_test.go:15` — `TestReinforcementDecayRanksHotRow` calls `rankByDecay`, which production search does not. `SearchOwnerScopedBlend` uses `sortHitsByDecay` (`search_blend.go:159`). Add a blend-level test with decay enabled.
- [Verification] `acceptance.feature` gate G2 — the check runs `TestMemSearchRequiresQuery`, which is not a test. Point it at `TestBudgetedRetrievalMCP`, which already rejects an empty query.
- [Standards] `mcp/tools_memory.go:1463` — `stampKeywordSignals` sets `recency` and `decay` to 1 and `importance` to 0 for every all-projects hit. Unmeasured signals should be 0, matching the blueprint's keyword stamp.

### Minor

- [Standards] `decay.go:146` — `rankByDecay` duplicates `sortHitsByDecay` and has no production caller.
- [Standards] `skills.go` `init` mutates `validTypes`; `codeindex` `init` registers `aliasLookup` to avoid an import cycle the blueprint accepted.
- [Standards] `symbol_fts.go:158` — alias lookup uses `context.Background()` and swallows lookup errors so FTS still runs.
- [Performance] `search_blend.go:63` — each embedded search decodes every observation vector; visibility is one query per candidate.
- [Performance] `query_cache.go:47` — expired rows are never deleted.
- (6 further Minor items: RRF score recomputed beside `ReciprocalRankFusion`, project id parsed from the sqlite filename, alias inserts not pruned on rename, vector window truncated before ACL so private hits can starve visible ones, decay-off blend does not call `rankByUse`, `mem hook list` test omits `compact`.)

### Rejected

- Comment mining, `mem alias add`, and wrapping `hybrid_search` with the query cache. The blueprint defers the first two and limits the cache task to the `mem_search` call site.
- Compact importance filtering. The blueprint task says `CompactionContext` plus one `session_summary` upsert. The session-scope leak above is separate.
- Migration 046. `045` was already recorded as query-cache only; a second file is what makes `entity_aliases` apply.
- Missing separate RED commits. `testing.tdd` is false.

## Independence

| Axis | Grade | Rationale |
|------|-------|-----------|
| Standards | B | Fresh subagent, Claude, no session history. Same Cursor toolchain as the implementer. |
| Spec | B | Fresh subagent, GPT, no session history. Same toolchain. |
| Security | B | Fresh subagent, Claude, no session history. Same toolchain. |
| Edge, verification, performance, red team | B | Fresh subagents (Claude, Gemini, GPT). Same toolchain. |

## Verdict

- **Standards:** met-with-fixes
- **Spec:** met-with-fixes
- **Security:** secure-with-fixes
- **Floor (decides):** not met until the vector-leg filter and the continuity-note session limit land
- **Worst issue:** superseded observations resurface on the vector leg of `mem_search`
- **Unfixed Important count:** 6
