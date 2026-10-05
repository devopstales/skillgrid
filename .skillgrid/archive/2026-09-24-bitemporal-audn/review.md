# Integration Review — 2026-09-24-bitemporal-audn

> Change: `.skillgrid/specs/2026-09-24-bitemporal-audn/` (blueprint 044_bitemporal, ADR-0011)
> Reviewed range: `dbd8111d..HEAD` on `release/2`, scoped to `skillgrid-cli/internal/mnemonic/**`
> Scope: 12 wave commits (TICKET-01..06 impl + fixes + L3 floor); `063f7b91`/`2ca9009d` (hooks/sdd tooling) and `9b9dcad5` (concurrent session-prime change) are NOT part of this wave.
> Method: read-only whole-branch integration review (per-ticket two-axis reviews already passed in-flight).

## Verdict

**APPROVED-WITH-ITEMS** — integrates cleanly; all 7 blueprint read sites filtered, ADR-0011 invariants met, no production caller broken. Two deferred areas: 5 LOW/INFO read-path hardening items + the already-recorded task-029 (no production LLM client). No blockers.

## Integration findings

| # | Sev | Seam | Evidence | Status |
|---|-----|------|----------|--------|
| 1 | LOW | Bi-temporal filter completeness | `crosslink.go:33-37` — `CrossLinkQuery` selects `o2.invalid_at` but WHERE has no filter; superseded rows can appear as cross-links. | Deferred — opt-in method, not one of the 7 named read sites; not in default search path. |
| 2 | LOW | Bi-temporal filter completeness | `service.go:1057-1060` — `Get` point-lookup has no `invalid_at` filter; a superseded row is returned by explicit ID. | Deferred — intentional: Get is a by-ID inspection path. |
| 3 | LOW | Bi-temporal filter completeness | `skills.go:125-131` — skill-match query has no `invalid_at` filter. | Deferred — `memory_type='skill'` rows unlikely to be superseded via AUDN. |
| 4 | LOW | Bi-temporal filter completeness | `export.go:97-98` — export query has no `invalid_at` filter. | Deferred — export is archival; including superseded rows is arguably correct for a complete dump. |
| 5 | INFO | Bi-temporal filter completeness | `retrieval.go:269,378,460` — directory-walk retrieval (scoreDirectories / directoryLeafResults / hasDirectoryChildren) has no `invalid_at` filter. | Deferred — directory walk is a structural index (topic_key tree), not content search; superseded rows don't rank in final FTS+vector results. |
| 6 | INFO | AUDN routing | `service.go:556-569` — hash-dedup lookup uses `deleted_at IS NULL` only (no `invalid_at`). | Correct by design — exact-hash dup is identical content; row exists whether superseded or not. |
| 7 | INFO | Config symmetry | `config/load.go` — `Dedup`/`dedupSection` mirrors `Extraction`/`extractionSection` identically (struct, yaml tag, merge, zero-value default false). No `validate` fn exists in the config package. | No finding — symmetric. |
| 8 | INFO | MCP additive contract | `tools_memory.go:284-293` — `action` gated on `!= ""`, `superseded_id` gated on `> 0`. Legacy `id`+`project` always present. | Correct — purely additive. |

### The 7 named read sites — all filtered

| Site | File:Line | Filter |
|------|-----------|--------|
| `SearchWithScope` | `service.go:880` | ✓ |
| `SearchOwnerScoped` | `service.go:940` | ✓ |
| List (recent) | `service.go:978` | ✓ |
| List (by status) | `service.go:1004` | ✓ |
| List (by type) | `service.go:1040` | ✓ |
| `SearchOwner` | `governance.go:456` | ✓ |
| `AdminCrossOwnerList` | `governance.go:503` | ✓ |

All 7 use the canonical idiom: `AND (o.invalid_at IS NULL OR o.invalid_at = '' OR strftime('%s', o.invalid_at) > strftime('%s', 'now'))`.

## Known open item — task-029 (production LLM client)

Confirmed structurally reachable + functionally inert:
- `SetDedupLLMFunc` is called only from `service/dedup_llm_test.go` (5 test sites); no non-test production code calls it.
- `service.go:420-422` — when `cfg.Dedup.LLM` is true, `SetDedupLLM(newDedupLLMBackend())` + `EnableDedupLLM(true)` attach the seam. The backend's `Classify` (`dedup_llm.go:69-71`) checks `dedupLLMFuncFn == nil` and returns `"no dedup LLM configured"` → deterministic hash floor.
- **The wave is functionally complete without task-029.** The LLM pass is OPT-IN (default `false`); the deterministic hash floor is the correct live path in all current configurations. Task-029 makes the 4-way classifier *active* (an optimization), not required.

## ADR-0011 fidelity

| Invariant | Status | Evidence |
|-----------|--------|----------|
| C1: `valid_at` stamped on creation | ✓ | `insertObservation` (`service.go:756-760`): `valid_at` = `now`. `TestSaveStampsBiTemporalColumns` passes. |
| C1.5: `invalid_at` set on supersede | ✓ | `MarkSuperseded` (`lifecycle.go:144-148`): `SET invalid_at = ?` in a single tx. |
| C2: `superseded_by` back-pointer | ✓ | `MarkSuperseded` (`lifecycle.go:146-148`): `SET superseded_by = ?` + `supersedes` edge in same tx (`lifecycle.go:156-164`). |
| Time-travel queryable | ✓ | `ValidAtTime` (`service.go:2394-2424`): `valid_at <= since AND (invalid_at IS NULL OR invalid_at > since)`. `TestValidAtTimeWindow` passes. |
| `created_at` immutable | ✓ | `MarkSuperseded` does not touch `created_at` (commit `d0f8073e`); re-supersede leaves existing edge untouched. |
| `Save` contract preserved | ✓ | `Save` (`service.go:452-458`) delegates to `SaveWithAction`, returns `(int64, error)`; 7 production callers unchanged. |
| `DedupLLM` backward compat | ✓ | `Dedup` retained as deprecated wrapper (`types.go`); `Classify` added. |

## Followups

- **task-029** (already recorded, needs-triage): wire a production LLM client to `SetDedupLLMFunc` so the 4-way classifier is active when `mnemonic.dedup.llm: true`. No action required for this wave.
- Findings #1-5 (deferred LOW/INFO read-path hardening) are candidates for a future "bi-temporal filter hardening" ticket if exhaustive coverage is wanted; none affect the wave's acceptance criteria.

## L3 regression floor (TICKET-07 / task-028)

`go build` clean · `go test ./...` all ok (http/tracker flake passed in isolation) · `go vet` only the known pre-existing `budget.go:114` · `memory` coverage 70.0% (= pre-wave baseline) · gofmt 0 unformatted wave files. No regressions, no commit.
