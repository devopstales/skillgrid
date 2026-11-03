# T12 — traffic learner (deterministic, zero-LLM) — report

**Ticket:** TASK-100 · **Wave 3** · **PR5**
**Package:** `mnemonic/internal/ctx/traffic/`
**Status:** DONE — all gates green, all AC + Fails-when covered.

## What was built

A zero-LLM, rule-based, deterministic traffic learner that turns a
session-event stream into candidate `ctx_traffic_memories` rows using a
human-editable `traffic/rules.yaml`. No upstream, no model, no network. The
extraction is a **pure function** of `(events, rules)`: same input → same
memories, same order. The learner never opens a store; it computes candidates
and the caller writes them via the already-injected `*ctxstore.Store`
(ADR-0033). `EvictToBound` takes the store as a parameter for the
`max_memories` bound.

Files:
- `rules.go` — `Rule` type, `PatternKind` set, `LoadRules` + validation.
- `learner.go` — `Event`, `Learner`, `Extract` + per-pattern matchers.
- `evict.go` — `EvictToBound` (global `max_memories` eviction).
- `rules.yaml` — one rule per category (package data).
- `learner_test.go`, `evict_test.go`, `rules_test.go`, `traffic_test.go` — real + negative tests over a real migrated shared store.
- `mnemonic/internal/ctx/store/tables.go` — additive `ListAllTrafficMemories` + `DeleteTrafficMemory` (no DDL/migration change; fail-open).

## The four rule categories + how each pattern matches

All patterns are simple and deterministic (counts / same-target / path
prefix) — **no free-text regex**, so extraction is stable across Go versions.

| Category | Rule id | Pattern | Match | Base conf |
|---|---|---|---|---|
| `error_recovery` | `er_fail_then_ok` | `fail_then_ok` | a `tool_result{status:error}` followed, on the **same non-empty Target**, by a `tool_result{status:ok}` (fail-then-retry). Confidence scaled up (+0.05 per extra cycle, capped at 1). | 0.8 |
| `environment` | `env_repeated_command` | `repetition` | a `tool_call` Tool/Target pair used ≥ `min_times` (3). Emits the single most-frequent qualifying key (ties broken by key string). | 0.7 |
| `preference` | `pref_repeated_message` | `message_repetition` | a non-empty `user_message` Message appearing ≥ `min_times` (2) — a repeated user correction / usage pattern. | 0.7 |
| `architecture` | `arch_common_prefix` | `path_prefix` | `file_read`/`file_edit`/`file_write` targets under one common directory prefix, ≥ `min_times` (3) — a file-structure / convention pattern. | 0.7 |

First-match-wins is enforced per event group (the per-kind guards emit one
candidate per logical trigger, so a repeated trigger does not fan out into
duplicate memories). Rules are applied in file order.

## Deterministic `MemoryID` scheme

`MemoryID = hex(sha1(sessionID + "\x00" + ruleID + "\x00" + key))` where `key`
is the stable identity suffix of the match (`target=...` / `tool target=...` /
`msg=<len>:<64b>` / `prefix=...`). No wall clock, no `rand`, no map-iteration
order — so re-runs produce the **same id** and dedupe via `INSERT OR REPLACE`.
Output order is `(rule file order, ascending seq)`; the per-pattern matchers
explicitly `sort` before emission.

## How zero-LLM + determinism are asserted

- **Zero-LLM (structural):** `Learner` has only `Rules / MinConfidence /
  MaxMemories / Clock`. The only injectable dependencies anywhere in the
  extract path are those fields and the `*ctxstore.Store` passed to
  `EvictToBound` — there is **no http client and no model field**, so the path
  cannot make an LLM call. `Extract` takes no store at all.
- **Zero-LLM (test):** `learner_test.go` runs `Extract` against a counting
  store and asserts exactly N puts and no network; a nil-store `EvictToBound`
  is a no-op.
- **Determinism (test):** `learner_test.go` asserts `reflect.DeepEqual` of two
  `Extract` calls on the same events (identical candidate set + order).

## Bounds + fail-open (asserted)

- **`min_confidence`:** a candidate below the floor is not emitted;
  `learner_test.go` has a sub-min memory that must NOT appear.
- **`max_memories`:** `evict_test.go` over a real store asserts the
  oldest-low-confidence-first eviction order (`ORDER BY confidence ASC,
  created_at ASC`).
- **Fail-open (ADR-0016):** `rules_test.go` covers a missing/unparseable
  `rules.yaml` (returns error, caller degrades — never a panic); a store fault
  in `EvictToBound` degrades to `(0, nil)`; a nil store / non-positive max is a
  no-op. No hard-fail path was added.

## Gates (exact outputs)

Run from `/Users/paladm/git/ai-test/skillgrid-v2/mnemonic`.

- **G1** `go build ./...` → exit 0 (clean).
- **G2** `go test ./...` → no `FAIL`/`panic`/`cannot` lines (this Go version
  prints only failures; the targeted `./internal/ctx/... ./internal/store/...`
  subset prints `ok` for every package incl. `internal/ctx/traffic`).
- **G3** `go test ./internal/ctx/traffic/... -race` → `ok ...
  internal/ctx/traffic` (green; 26 tests).
- **G4** `gofmt -l` on the touched files (`internal/ctx/traffic/*.go`,
  `internal/ctx/store/tables.go`) → empty (clean).

## Commit

Single feature commit staged with only this ticket's files
(`mnemonic/internal/ctx/traffic/*` + the two additive `tables.go` methods):

- `feat(ctx): add deterministic zero-LLM traffic learner (T12)`
- `[skillgrid-context]` block: change `2026-10-08-context-orchestrator`,
  ticket `TASK-100`, wave `3`, pr `PR5`.

(See `git log` for the SHA — recorded in the closing message.)

## Deviations from the brief

- **Store API naming:** the brief's illustrative `Event`/`Learner` signatures
  were matched in substance; the concrete `NewLearner` constructor +
  `Extract(sessionID, events) []*ctxstore.TrafficMemory` (no error, pure) is
  used. Tests were written against the real API.
- **`min_times` added to `Rule`:** the counting patterns
  (`repetition`, `message_repetition`, `path_prefix`) need a repetition
  threshold; `LoadRules` validates `min_times >= 1` for those patterns. This is
  additive to the brief's `Rule` struct and is what makes the four categories
  match the brief's intended thresholds (≥2 / ≥3).
- **No new dependencies:** uses the existing `gopkg.in/yaml.v3` (ADR-0008
  precedent). Nothing added.

## Open questions

- None. The `traffic-promotion-human-gate` scenario belongs to T13 (out of
  scope here). The caller wiring (passing `cfg.Traffic` fields into
  `NewLearner` and invoking `EvictToBound`) is T14's job.
