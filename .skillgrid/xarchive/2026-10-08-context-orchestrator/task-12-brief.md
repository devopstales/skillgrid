# T12 — traffic learner: zero-LLM, rule-based, deterministic candidate rules

**Ticket:** TASK-100 · **Wave 3** · **PR5** (traffic + judgment + CLI) · **Size M (~850 lines)**
**Scope:** `mnemonic/internal/ctx/traffic/` — a zero-LLM, rule-based, deterministic traffic
learner that emits candidate `ctx_traffic_memories` rows from a session-event stream. No
upstream, no LLM, no other service. Same input → same candidate set (deterministic).

## Goal

Implement the real-time traffic learner described in
`.skillgrid/artifacts/context-orchestrator-plan.md` §18 (Phase 18 — Real-Time Traffic Learning).
It is a **pure function** over the session-event stream plus a human-editable rules file. It
makes **zero** LLM/upstream calls. It stores nothing directly except via the existing
`ctxstore` `PutTrafficMemory` — the learner itself computes candidates and applies the
`min_confidence` and `max_memories` bounds.

## Non-negotiable constraints

- **ADR-0033:** `mnemonic ctx` is a subcommand of the existing binary; no second DB handle.
  The learner writes through the **already-injected** `*ctxstore.Store` (which wraps the shared
  migrated `*store.Store`). Never open a store itself.
- **ADR-0016 (fail-open):** a rules-file parse error or a store error must NOT panic or fail the
  process; the learner degrades (returns the candidates it could compute, or an empty set) and
  logs at most one line. The store methods are already fail-open; the learner must not add a
  hard-fail path.
- **No LLM call.** Zero network, zero model. The test asserts zero invocations (see below).
- **No new dependencies.** `gopkg.in/yaml.v3` is already in `go.mod` (ADR-0008 precedent). Do not
  add anything else. If you think you need one, stop and report.
- **Determinism (§18.6):** extraction is a pure function of (events, rules). Same input → same
  memories, same order. No `map` iteration order leaking into output order; no wall-clock or
  `rand` in candidate identity/ordering (created_at may use an injected clock for testability).

## What to build

New package `mnemonic/internal/ctx/traffic/`:

1. **`rules.go`** — load + validate `traffic/rules.yaml`. Each rule declares:
   `id` (stable, unique), `category` (one of the four), `pattern` (a matcher descriptor — see
   below), `confidence` (weight 0..1, the base confidence for a match), and `description`.
   Rules are applied **in file order; the first matching rule wins** for a given event
   (deterministic). Provide a small typed representation, e.g.
   `type Rule struct { ID, Category, Pattern, Description string; Confidence float64 }` and
   `LoadRules(path string) ([]Rule, error)`. A missing/unreadable file is a fail-open concern:
   `LoadRules` returns an error and the caller (Learner) degrades gracefully.

2. **`learner.go`** — the extractor. Define the event input type:
   ```go
   type Event struct {
       Kind     string // e.g. "tool_call", "tool_result", "file_read", "user_message", ...
       Tool     string // tool name when Kind=="tool_call"
       Target   string // path/command/arg the tool acted on
       Status   string // "ok" | "error" | "retry" | "" (tool_result)
       Message  string // free text (user_message / tool_result message)
       Seq      int    // monotonic event order within the session
   }
   ```
   `type Learner struct { Rules []Rule; MinConfidence float64; MaxMemories int; Clock func() int64 }`.
   `func (l *Learner) Extract(sessionID string, events []Event) []*ctxstore.TrafficMemory`.
   - For each rule, scan events; when a rule's pattern matches, emit a `TrafficMemory` with
     `Category`, `MemoryText` (a short human-auditable sentence), `EvidenceJSON` (a JSON array of
     the `{seq,kind,tool,target,status}` events that produced it), `Confidence` (rule weight,
     optionally scaled by repetition count), and `CreatedAt` from `l.Clock()`.
   - `MemoryID` must be a **stable, deterministic** string (e.g. sha1 of sessionID+ruleID+
     matched-target, or a slug) so re-runs dedupe via `INSERT OR REPLACE`.
   - **`min_confidence` bound:** a candidate whose confidence < `MinConfidence` is NOT emitted.
   - **First-match-wins:** once an event is consumed by an earlier rule, later rules do not
     re-emit for that event (prevents duplicate/contradictory candidates).
   - No LLM/upstream call anywhere in this path.

3. **`evict.go`** — the `max_memories` bound with **oldest-low-confidence-first** eviction.
   ```go
   // EvictToBound loads the current store (all sessions) and, if the total exceeds max,
   // deletes the oldest low-confidence rows first until count <= max.
   func EvictToBound(cs *ctxstore.Store, max int) (int, error)
   ```
   Eviction order: `ORDER BY confidence ASC, created_at ASC` — lowest confidence and oldest
   first. Return the number evicted. Fail-open on store errors.

4. **`rules.yaml`** (package data) — a real rules file with at least one rule per category:
   - `error_recovery`: a `tool_result{status:error}` followed (same `Target`) by a
     `tool_result{status:ok}` — i.e. fail-then-retry on the same target. Confidence ~0.8.
   - `environment`: a stable fact — e.g. the same command/tool appearing repeatedly (repetition
     threshold ≥2). Confidence ~0.6.
   - `preference`: a repeated user correction/usage pattern (same `user_message` fragment ≥2).
     Confidence ~0.6.
   - `architecture`: a file-structure/convention pattern — repeated reads/edits of files under a
     common path prefix (≥3). Confidence ~0.6.
   Keep patterns **simple and deterministic** (repetition counts, same-target, prefix). No regex
   over free text that could be non-deterministic across Go versions unless you pin it.

5. **Tests** (`learner_test.go`, `evict_test.go`, `rules_test.go`) using `ctxstore` over a real
   migrated shared store (`store.Open(t.TempDir(), "ctx-traffic")` → `ctxstore.New(shared)`),
   mirroring the T06/T08 harness. Cover the acceptance criteria (below) **and** the
   `Fails-when` list.

## Acceptance (must all pass)

- **AC#1 / Fails-when:** `go test ./internal/ctx/traffic/...` green. A deterministic corpus
  produces the **same** candidate set on repeated runs (assert `reflect.DeepEqual` of two
  `Extract` calls on the same events). A fail-then-retry event stores an `error_recovery` memory
  with `evidence_json` and makes **zero** LLM calls — assert this by structuring the learner so
  no I/O dependency is injectable except the store+clock (no http client, no model field), and
  by a test that runs `Extract` with a store whose put is counted: exactly N puts, no network.
- **AC#2:** all four categories extract from fixture sessions; each `TrafficMemory.Category` is
  one of the four.
- **AC#3:** a candidate is created **only** when its trigger rule matches; a corpus with no
  matching events yields zero candidates (no false positives).
- **Fails-when:** `go test ./internal/ctx/traffic/...` exit != 0 OR an LLM call is observed OR a
  sub-`min_confidence` memory is stored.
- **Bounds:** a below-`min_confidence` memory is not stored; when the store exceeds
  `max_memories`, the oldest low-confidence memory is evicted first.

**SATISFIES** (acceptance.feature): `traffic-learner-extracts` (lines 169-174),
`traffic-learner-four-categories` (lines 177-183). (The `traffic-promotion-human-gate` scenario
belongs to T13, not this ticket.)

## Existing store API (reuse, do not modify the schema)

From `mnemonic/internal/ctx/store/tables.go` (already implemented in T02):
```go
type TrafficMemory struct {
    MemoryID, SessionID, Category, MemoryText, EvidenceJSON string
    Confidence float64
    CreatedAt int64
}
func (s *Store) PutTrafficMemory(m *TrafficMemory) error          // INSERT OR REPLACE, fail-open
func (s *Store) GetTrafficMemory(memoryID string) (*TrafficMemory, error)
func (s *Store) ListTrafficMemories(sessionID string) ([]*TrafficMemory, error)
```
Note: `ListTrafficMemories` filters by `session_id`. `EvictToBound` needs the **global** count and
a global eviction order — if a global `ListAllTrafficMemories` / `DeleteTrafficMemory(memoryID)`
/ count helper does not already exist, you MAY add the minimum additive store methods you need
(same fail-open style, same package `internal/ctx/store`). Adding additive store methods is
allowed; do NOT change the `ctx_traffic_memories` DDL or migration files. If you add store
methods, add them in `tables.go` next to the existing `ctx_traffic_memories` block and give them
their own fail-open treatment (mirror `queryFailOpen` / `execFailOpen`).

Config (already in T03, `mnemonic/internal/ctx/config/config.go`): `Traffic{Enabled, Realtime,
RulesPath, Categories, MinConfidence, MaxMemories, EmitEvents}` with defaults
`MinConfidence: 0.6, MaxMemories: 500, RulesPath: "traffic/rules.yaml"`. The learner takes these
as plain params (don't import the config package into `traffic` — keep `traffic` dependency-light;
the caller in T14 will pass `cfg.Traffic` fields in).

## Working rules

- **TDD is OFF** for this project — write the implementation and the tests together; do not do a
  separate red-green cycle. But the tests MUST be real and must fail if the code is wrong
  (do not write tautological tests).
- **Stage only your files** — never `git add -A`. The repo is shared. Do NOT stage the
  `mnemonic/mnemonic` ~62MB build artifact or any unrelated file.
- **Conventional commit** only, no AI-attribution trailer. Body block, exactly:
  ```
  [skillgrid-context]
  change: 2026-10-08-context-orchestrator
  ticket: TASK-100
  wave: 3
  pr: PR5
  ```

## Gates (run all, all must be green before you finish)

- **G1:** `go build ./...` (workdir `/Users/paladm/git/ai-test/skillgrid-v2/mnemonic`)
- **G2:** `go test ./...` (workdir `mnemonic/`)
- **G3:** `go test ./internal/ctx/traffic/... -race`
- **G4:** `gofmt -l ./...` from `mnemonic/` returns empty (no unformatted files). Run
  `gofmt -w` on any file you touch if needed.

## Definition of Done

- [ ] `mnemonic/internal/ctx/traffic/{learner,rules,evict}.go` + `rules.yaml` created
- [ ] `go test ./internal/ctx/traffic/... -race` green; all AC + Fails-when covered
- [ ] Zero LLM/upstream call in the extract path (structurally + asserted in test)
- [ ] Deterministic: repeated `Extract` on the same events → identical candidate set (asserted)
- [ ] `min_confidence` and `max_memories` bounds enforced (sub-min not stored; oldest-low-conf evicted first)
- [ ] Fail-open: rules-file error and store error do not panic
- [ ] G1–G4 green; only your files staged; conventional commit with the `[skillgrid-context]` block
- [ ] SATISFIES `traffic-learner-extracts`, `traffic-learner-four-categories`

## Report

Write your full report to
`.skillgrid/sdd/2026-10-08-context-orchestrator/task-12-report.md`:
what you built, the four rule categories + how each pattern matches, the deterministic MemoryID
scheme, how you assert zero-LLM + determinism, the exact gate outputs (G1–G4), the commit SHA(s),
any deviations from this brief, and any open questions. Your reply to me should be a short status
(<15 lines): status, commit SHA(s), gate results, any blockers.
