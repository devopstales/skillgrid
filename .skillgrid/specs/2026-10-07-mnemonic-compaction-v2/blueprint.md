# Mnemonic Compaction v2 — Implementation Blueprint (steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Journey

**Goal:** Enhance mnemonic compaction with five additive capabilities — advisory timing gate, structured six-section prompt, CLM steering, proactive instant build, and a full context TUI with keyboard navigation — wired to the 5 new TS plugins (replace-hooks plan assumed to land first).

**Architecture:** This plan extends ADR-0027 (CLM) and ADR-0028 (`context_revisions`). The advisory gate is a new `internal/advice` package using the existing `internal/llm` client (one combined call). The structured prompt extends `CompactionContext`. Steering is a new `steering` column on the existing `context_revisions` table (migration 053). The proactive build reuses the async-tiering goroutine pattern. The context TUI is a new `skillgrid context` CLI subcommand (Bubbletea, already in `go.mod` as indirect) with three keyboard-navigable screens (Usage, Revisions, Injection) polling new `/context/*` HTTP routes. All plugin wiring targets `plugins/opencode/{skillgrid-compaction,skillgrid-events,mnemonic-memory}.ts`.

**Tech Stack:** Go 1.22+, SQLite, OpenAI-compatible LLM client (`internal/llm`), OpenCode v2 plugins (TS).

**Spec:** `.skillgrid/specs/2026-10-07-mnemonic-compaction-v2/briefing.md`

## Terms

- **Advisory Gate:** The compact-adviser logic — a 0–1 score deciding whether to compact now (floor 0.90 empty → 0.50 full).
- **Steering:** A natural-language instruction the model can evolve, re-injected into each compaction prompt.
- **Proactive Build:** A background pre-summarization so the real compaction event is instant.
- **CLM Mirror:** The ephemeral context file the model edits (ADR-0027); this plan's structured prompt feeds it.
- **Context TUI:** The `skillgrid context` Bubbletea subcommand — three keyboard-navigable screens (Usage, Revisions, Injection) visualizing what occupies the context window.
- **Frozen Injections:** The session-start context that never changes mid-session (system prompt, tool definitions, AGENTS.md, skill prompts, MCP instructions).

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- `GET /compaction/advice` returns `{score, hint, floor, reason}` from one combined LLM call; LLM-down returns `hint:false, reason:"llm unavailable"`.
- `CompactionContext` returns six structured sections; a user correction appears verbatim in `Errors & Corrections`.
- Two `hookCompact` runs produce `context_revisions` rev 1 and rev 2; the rev-2 prompt input contains rev-1's `steering`.
- `proactive:true, interval:15m, contextFraction 0.4` produces a pre-built revision within one interval without `session.idle`.
- `mem_compact_advice` MCP tool returns `{score, hint, floor, reason}`.
- `mnemonic.compaction.adviser_enabled: true` → `SetCompaction(AdviserEnabled:true)`; absent section → defaults.
- `GET /context/usage?session_id=...` returns six categories (system prompt, tool definitions, extension injections, messages, compaction summary, free space) summing to ≤ budget, with `fraction`, `floor`, `hint`, `score`.
- `GET /context/revisions?session_id=...` returns ordered `context_revisions` rows with `revision`, `created_at`, `size_estimate`, `steering`, `messages`, `trigger`.
- `skillgrid context <session_id>` launches a Bubbletea TUI: Tab cycles Usage → Revisions → Injection, `↑↓` navigates, `Enter` expands, `r` refreshes, `q` quits.

**Artifacts** (files that must exist with real implementation, not stubs):
- `mnemonic/internal/advice/advice.go` — `Adviser`, `Advice`, `Advise`, floor curve.
- `mnemonic/internal/config/load.go` — `Compaction` struct + `DefaultCompaction` + `compactionSection` + `mergeCompaction`.
- `mnemonic/internal/service/service.go` — `compactionCfg` + `SetCompaction` + LLM seam.
- `mnemonic/internal/service/compaction.go` — proactive build goroutine + revision write.
- `mnemonic/internal/memory/skills.go` — `hookCompact` → append `context_revisions` + steering.
- `mnemonic/internal/memory/service.go` — `CompactionContext.Sections` + `steering` read.
- `mnemonic/internal/checkpoint/prompt.go` — six-section structured prompt.
- `mnemonic/internal/store/migrations/053_context_revisions_steering.sql` — `steering` column.
- `mnemonic/internal/http/compaction.go` — `/compaction/advice` routes.
- `plugins/opencode/skillgrid-compaction.ts` — advisory hint + structured context POST + injection breakdown POST.
- `plugins/opencode/skillgrid-events.ts` — context-char count POST on `tool.execute.after`.
- `plugins/opencode/mnemonic-memory.ts` — `mem_compact_advice` + `mem_context_usage` + `mem_context_revisions` tools.
- `mnemonic/internal/http/context.go` — `/context/usage`, `/context/revisions`, `/context/injections` routes.
- `skillgrid-cli/internal/cmd/context.go` — `skillgrid context` cobra subcommand.
- `skillgrid-cli/internal/tui/context_model.go` — Bubbletea `Model` with three screens.
- `skillgrid-cli/internal/tui/context_usage.go` — Usage screen (block grid + category bars + advisory line).
- `skillgrid-cli/internal/tui/context_revisions.go` — Revisions screen (ordered list + expand).
- `skillgrid-cli/internal/tui/context_injections.go` — Injection screen (frozen breakdown + expand).
- `skillgrid-cli/internal/tui/context_test.go` — Bubbletea teatest cases.

**Key links** (critical connections between artifacts that must work together):
- `hookCompact` must append (not upsert) a `context_revisions` row and carry the prior `steering` forward.
- `CompactionContext.Sections` must be the body that `POST /compaction/advice` persists as the next revision.
- The adviser's `contextFraction` must come from the same `budget_chars` config the proactive build reads.
- `skillgrid-events.ts` `tool.execute.after` must POST the context-char count that `GET /compaction/advice` consumes.
- `skillgrid-compaction.ts` `experimental.session.compacting` must POST the injection breakdown to `POST /context/injections` so `GET /context/usage` can report frozen categories.
- The TUI must poll `GET /context/usage` and `GET /context/revisions` at 2s interval; a failed poll shows stale data with a warning, never crashes.

**One-way-door decisions** (hard to reverse — flag for explicit user approval before implementing):
- **Task 3.1:** Adding the `steering` column to `context_revisions` (migration 053, additive to ADR-0028's table).

## Global Constraints

- **Go 1.22+:** Minimum Go version to build (per `05-locked-constraints.md`).
- **No new dependencies:** No new libraries without an ADR (reuses `internal/llm`).
- **Serial development:** One change at a time (per `05-locked-constraints.md`).
- **Conventional commits:** No AI-attribution trailers.
- **Spec-zone commits before code-zone commits** (pre-commit zone guard).
- **Replace-first:** All plugin wiring targets the 5 new TS plugins; the replace-hooks plan lands first.

---

## Task 1: Config `Compaction` Section

**Files:**
- Modify: `mnemonic/internal/config/load.go`
- Test: `mnemonic/internal/config/load_test.go`

**Interfaces:**
- Produces: `type Compaction struct { AdviserEnabled bool; BudgetChars int; Proactive bool; Interval time.Duration; MaxRevisions int }`
- Produces: `func DefaultCompaction() Compaction`
- Produces: `type compactionSection struct { AdviserEnabled, Proactive *bool; BudgetChars *int; Interval *string; MaxRevisions *int }`
- Produces: `func mergeCompaction(base Compaction, section compactionSection) Compaction`

**SATISFIES:** `compaction-config` scenario in `acceptance.feature`

- [ ] **Step 1: Write the failing test**

```go
func TestMergeCompaction_Defaults(t *testing.T) {
    got := mergeCompaction(DefaultCompaction(), compactionSection{})
    if !got.AdviserEnabled || !got.Proactive { t.Fatal("defaults must be false") }
    if got.BudgetChars != 60000 || got.Interval != 15*time.Minute || got.MaxRevisions != 10 {
        t.Fatalf("unexpected defaults: %+v", got)
    }
}
func TestMergeCompaction_Override(t *testing.T) {
    on := true
    got := mergeCompaction(DefaultCompaction(), compactionSection{AdviserEnabled: &on, BudgetChars: intp(12000)})
    if !got.AdviserEnabled || got.BudgetChars != 12000 { t.Fatalf("override failed: %+v", got) }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/internal/config/... -run TestMergeCompaction -v`
Expected: FAIL with "undefined: Compaction / mergeCompaction"

- [ ] **Step 3: Write minimal implementation**

Add `Compaction` struct + `DefaultCompaction` + `compactionSection` + `mergeCompaction` (mirror `mergeCheckpoint`). Add `Compaction Compaction` to `Indexing`; `compactionSection compactionSection` to `mnemonicSection`. Wire `out.Compaction = mergeCompaction(DefaultCompaction(), section.Compaction)` in `Load` (after `mergeCheckpoint` at ~line 719).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mnemonic/internal/config/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/config/
git commit -m "feat(mnemonic): add mnemonic.compaction config section"
```

---

## Task 2: Structured Six-Section Prompt

**Files:**
- Modify: `mnemonic/internal/memory/service.go` (extend `CompactionContext`)
- Modify: `mnemonic/internal/checkpoint/prompt.go`
- Test: `mnemonic/internal/memory/service_compaction_test.go`, `mnemonic/internal/checkpoint/prompt_test.go`

**Interfaces:**
- Produces: `CompactionContext.Sections map[string]string` (keys: `User Intent`, `Actions Succeeded`, `Errors & Corrections`, `Active Work`, `Pending Tasks`, `Critical Details`)
- Produces: `CompactionContext.Steering string`
- Consumes: existing `recentForCompaction` + session events

**SATISFIES:** `structured-sections` scenario in `acceptance.feature`

- [ ] **Step 1: Write the failing test**

```go
func TestCompactionContext_StructuredSections(t *testing.T) {
    // Seed a session with: a user correction ("use bcrypt not md5"), an error, an in-progress task.
    // Call CompactionContext.
    // Assert Sections["Errors & Corrections"] contains the correction verbatim.
    // Assert Sections["Active Work"] is non-empty.
}
func TestPrompt_RendersSixSections(t *testing.T) {
    got := RenderPrompt(PromptInput{ ... })
    for _, s := range []string{"## User Intent", "## Actions Succeeded", "## Errors & Corrections", "## Active Work", "## Pending Tasks", "## Critical Details"} {
        if !strings.Contains(got, s) { t.Errorf("missing section %q", s) }
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/internal/memory/... ./mnemonic/internal/checkpoint/... -run 'StructuredSections|RendersSixSections' -v`
Expected: FAIL (no `Sections` field / sections missing)

- [ ] **Step 3: Write minimal implementation**

Extend `CompactionContext` with `Sections map[string]string` + `Steering string`. Build sections recent-weighted from session events + observations; corrections captured verbatim. `checkpoint/prompt.go`: render the six-section compression instruction with the preserve-order rule (`corrections > errors > active work > completed work`).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mnemonic/internal/memory/... ./mnemonic/internal/checkpoint/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/memory/ mnemonic/internal/checkpoint/
git commit -m "feat(mnemonic): structured six-section compaction prompt"
```

---

## Task 3: CLM Steering Column

> ⚠ **one-way:** Adding `steering` column to `context_revisions` (migration 053, additive to ADR-0028). STOP for user approval.

**Files:**
- Create: `mnemonic/internal/store/migrations/053_context_revisions_steering.sql`
- Modify: `mnemonic/internal/memory/skills.go` (`hookCompact` → append revision + steering)
- Modify: `mnemonic/internal/memory/service.go` (`CompactionContext` reads latest `steering`)
- Test: `mnemonic/internal/memory/compact_hook_test.go` (extend)

**Interfaces:**
- Produces: `steering TEXT` column on `context_revisions` (migration 053).
- Consumes: existing `context_revisions` table (ADR-0028, migration 052).

**SATISFIES:** `steering-reinjection` scenario in `acceptance.feature`

- [ ] **Step 1: Write the failing test**

```go
func TestHookCompact_AppendsRevisionWithSteering(t *testing.T) {
    // Run hookCompact twice for one session.
    // Assert context_revisions has rev 1 and rev 2 (not a single upserted row).
    // Set rev-1 steering to "keep all migration IDs"; run hookCompact again.
    // Assert the rev-3 prompt input contains "keep all migration IDs".
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/internal/memory/... -run TestHookCompact_AppendsRevisionWithSteering -v`
Expected: FAIL (no `steering` column / blind upsert)

- [ ] **Step 3: Write minimal implementation**

Create migration `053_context_revisions_steering.sql` (`ALTER TABLE context_revisions ADD COLUMN steering TEXT`). Rewrite `hookCompact` to append a `context_revisions` row (rev = max(revision)+1) carrying the structured sections as `messages` and the prior `steering` into the prompt input. `CompactionContext` reads the latest revision's `steering` and returns it in `Steering`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mnemonic/internal/memory/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/store/migrations/ mnemonic/internal/memory/
git commit -m "feat(mnemonic): CLM steering column + revision append in hookCompact"
```

---

## Task 4: Advisory Timing Gate

**Files:**
- Create: `mnemonic/internal/advice/advice.go`
- Create: `mnemonic/internal/advice/advice_test.go`
- Test: (same file)

**Interfaces:**
- Produces: `type AdviceInput struct { SessionID, RecentDigest string; ContextChars, BudgetChars int }`
- Produces: `type Advice struct { Score, Floor float64; Hint bool; Reason string }`
- Produces: `type Adviser struct { ... }` with `func New(c llm.Completer, budgetChars int) *Adviser`
- Produces: `func (a *Adviser) Advise(ctx context.Context, in AdviceInput) (Advice, error)`

**SATISFIES:** `advisory-hint-curve` scenario in `acceptance.feature`

- [ ] **Step 1: Write the failing test**

```go
func TestAdviser_FloorCurve(t *testing.T) {
    // Empty context (fraction 0) → floor 0.90; full (fraction 1) → floor 0.50.
    a := New(fakeCompleter{finished:true, handsOn:false}, 1000)
    a.Advise(ctx, AdviceInput{ContextChars:0, BudgetChars:1000})  // score 1.0 >= 0.90 → hint true
    a.Advise(ctx, AdviceInput{ContextChars:1000, BudgetChars:1000}) // score 1.0 >= 0.50 → hint true
    // hands_on=true → score 0.5; at fraction 0, 0.5 < 0.90 → hint false.
}
func TestAdviser_FailOpen(t *testing.T) {
    a := New(errCompleter{}, 1000)
    got, _ := a.Advise(ctx, AdviceInput{...})
    if got.Hint || got.Reason != "llm unavailable" { t.Fatalf("fail-open broken: %+v", got) }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/internal/advice/... -v`
Expected: FAIL with "undefined: Adviser"

- [ ] **Step 3: Write minimal implementation**

Implement `Adviser` wrapping `llm.Completer`. One combined call: system prompt asks the model to return JSON `{finished:bool, hands_on:bool}`; parse once. `contextFraction = min(1, contextChars/budgetChars)`; `score = 0.5*finished + 0.5*(1-handsOn)`; `floor = 0.90 - 0.40*contextFraction`; `hint = score >= floor`. On LLM error → `Advice{Hint:false, Reason:"llm unavailable"}` (fail-open).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mnemonic/internal/advice/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/advice/
git commit -m "feat(mnemonic): advisory timing gate (internal/advice)"
```

---

## Task 5: HTTP Routes `/compaction/advice`

**Files:**
- Create: `mnemonic/internal/http/compaction.go`
- Create: `mnemonic/internal/http/compaction_test.go`
- Modify: `mnemonic/internal/http/server.go` (wire `registerCompactionRoutes()`)

**Interfaces:**
- Produces: `GET /compaction/advice?session_id=...` → `{score, hint, floor, reason}`
- Produces: `POST /compaction/advice` (body: structured sections + steering) → `{revision}`

**SATISFIES:** `compaction-advice-routes` scenario in `acceptance.feature`

- [ ] **Step 1: Write the failing test**

```go
func TestCompactionAdvice_GET(t *testing.T) {
    // Seed a session; GET /compaction/advice?session_id=...
    // Assert 200 + JSON with score/hint/floor/reason.
}
func TestCompactionAdvice_POST(t *testing.T) {
    // POST a structured body; assert a context_revisions row is appended and the response carries the new revision.
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/internal/http/... -run CompactionAdvice -v`
Expected: FAIL (route not registered)

- [ ] **Step 3: Write minimal implementation**

`registerCompactionRoutes()`: `GET /compaction/advice` runs the adviser (resolving `contextChars` from the session + `budget_chars` from config); `POST /compaction/advice` persists the structured body as a `context_revisions` row (rev+1) and returns the new rev. Wire into `registerRoutes()` (mirror `teams.go`).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mnemonic/internal/http/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/http/
git commit -m "feat(mnemonic): /compaction/advice HTTP routes"
```

---

## Task 6: Proactive Instant Compaction

**Files:**
- Modify: `mnemonic/internal/service/compaction.go`
- Modify: `mnemonic/internal/service/service.go` (start the build goroutine)
- Test: `mnemonic/internal/service/compaction_proactive_test.go`

**Interfaces:**
- Consumes: `config.Compaction.{Proactive, Interval, BudgetChars}`
- Produces: a background goroutine that pre-builds the structured revision on `Interval`, skipped when `contextFraction < 0.3`.

**SATISFIES:** `proactive-prebuild` scenario in `acceptance.feature`

- [ ] **Step 1: Write the failing test**

```go
func TestProactiveBuild_Interval(t *testing.T) {
    // Set compactionCfg{Proactive:true, Interval: 50ms, BudgetChars:1000}; session at contextFraction 0.4.
    // Wait ~150ms; assert a context_revisions row exists WITHOUT a session.idle event.
}
func TestProactiveBuild_SkipBelowThreshold(t *testing.T) {
    // contextFraction 0.2 → no revision within one interval.
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/internal/service/... -run ProactiveBuild -v`
Expected: FAIL (no goroutine)

- [ ] **Step 3: Write minimal implementation**

Add the proactive build goroutine (mirror the async-tiering pattern in `MnemonicCommit`): on `Interval`, build the structured revision and append to `context_revisions`; skip if `contextFraction < 0.3`. Warn on stderr on error (never fail the request).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mnemonic/internal/service/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/service/
git commit -m "feat(mnemonic): proactive instant compaction background build"
```

---

## Task 7: Service `SetCompaction` + LLM Seam

**Files:**
- Modify: `mnemonic/internal/service/service.go`
- Test: `mnemonic/internal/service/service_test.go` (extend)

**Interfaces:**
- Produces: `func (s *Service) SetCompaction(cfg config.Compaction)`
- Consumes: `config.Load` at the retrieval-budget site (`service.go:396`, `retrieval.go:253`).

**SATISFIES:** `compaction-config` scenario in `acceptance.feature`

- [ ] **Step 1: Write the failing test**

```go
func TestSetCompaction_Wired(t *testing.T) {
    // Load config with mnemonic.compaction.adviser_enabled: true; open service.
    // Assert service.compactionCfg.AdviserEnabled == true.
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/internal/service/... -run TestSetCompaction_Wired -v`
Expected: FAIL (no `SetCompaction`)

- [ ] **Step 3: Write minimal implementation**

Add `compactionCfg config.Compaction` field + `SetCompaction` (mirror `SetBudget`). Wire `cfg.Compaction` → `SetCompaction` at the config-load site (next to `rb := cfg.RetrievalBudget`). Pass the `llm.Completer` to the adviser via the existing LLM seam.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mnemonic/internal/service/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/service/
git commit -m "feat(mnemonic): wire mnemonic.compaction config into service"
```

---

## Task 8: Plugin Wiring (5 TS plugins, replace-first)

**Files:**
- Modify: `plugins/opencode/skillgrid-compaction.ts`
- Modify: `plugins/opencode/skillgrid-events.ts`
- Modify: `plugins/opencode/mnemonic-memory.ts`
- Test: (typecheck + `pnpm test`)

**Interfaces:**
- `skillgrid-compaction.ts`: on `experimental.session.compacting` + `session.compacted` → `GET /compaction/advice` (hint line) + `POST /compaction/advice` (structured `CompactionContext.Sections` + steering).
- `skillgrid-events.ts`: on `tool.execute.after` → POST context-char count (feeds the adviser's `contextFraction`).
- `mnemonic-memory.ts`: new `mem_compact_advice` custom tool → `GET /compaction/advice` → `{score, hint, floor, reason}`.

**SATISFIES:** `compaction-advice-routes` + `advisory-hint-curve` scenarios in `acceptance.feature`

- [ ] **Step 1: Add the `mem_compact_advice` tool**

In `mnemonic-memory.ts`, register `mem_compact_advice` (POST/GET to `${BASE}/compaction/advice`). Convention: `// @ts-nocheck`, inlined types, no `@opencode-ai/plugin` import.

- [ ] **Step 2: Wire the compaction plugin**

In `skillgrid-compaction.ts`, on `experimental.session.compacting`/`session.compacted`: GET the advisory hint → inject a one-line hint; POST the structured `CompactionContext.Sections` + steering to `/compaction/advice`.

- [ ] **Step 3: Wire the events plugin**

In `skillgrid-events.ts`, on `tool.execute.after`: POST the context-char count to the adviser input (so `contextFraction` reflects real usage).

- [ ] **Step 4: Run typecheck + tests**

Run: `pnpm typecheck && pnpm lint && pnpm test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add plugins/opencode/
git commit -m "feat(mnemonic): wire compaction advice + proactive build into 5 TS plugins"
```

---

## Task 9: HTTP Routes `/context/usage` + `/context/revisions` + `/context/injections`

**Files:**
- Create: `mnemonic/internal/http/context.go`
- Create: `mnemonic/internal/http/context_test.go`
- Modify: `mnemonic/internal/http/server.go` (wire `registerContextRoutes()`)

**Interfaces:**
- Produces: `GET /context/usage?session_id=...` → `{categories: [{name, chars, tokens}], total_tokens, budget_tokens, fraction, floor, hint, score}`
- Produces: `GET /context/revisions?session_id=...` → `[{revision, created_at, size_estimate, steering, messages, trigger}]`
- Produces: `POST /context/injections` (body: `{session_id, categories: [{name, chars}]}`) → `{stored: true}`
- Consumes: `config.Compaction.BudgetChars`, `context_revisions` table, `internal/advice` floor curve

**SATISFIES:** `happy path context usage returns category breakdown` + `happy path context revisions returns ordered list` scenarios in `acceptance.feature`

- [ ] **Step 1: Write the failing test**

```go
func TestContextUsage_GET(t *testing.T) {
    // Seed session "s1" with context_revisions (2 rows), known event chars.
    // POST /context/injections with frozen breakdown (system prompt 12400, tools 8200, etc.).
    // GET /context/usage?session_id=s1
    // Assert 200 + six categories, sum <= budget, fraction correct.
}
func TestContextRevisions_GET(t *testing.T) {
    // Seed 3 context_revisions rows for session "s1" (rev 1, 2, 3).
    // GET /context/revisions?session_id=s1
    // Assert 3 items, ordered by revision descending, each has steering + messages.
}
func TestContextInjections_POST(t *testing.T) {
    // POST /context/injections with a frozen breakdown.
    // Assert 200 + {stored: true}.
    // GET /context/usage → frozen categories reflect the POSTed values.
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/internal/http/... -run 'Context(Usage|Revisions|Injections)' -v`
Expected: FAIL (route not registered)

- [ ] **Step 3: Write minimal implementation**

`registerContextRoutes()`:
- `GET /context/usage`: resolve `session_id`, read `context_revisions` (sum `size_estimate` for compaction summary), read stored injections (frozen categories), compute message chars from session events, compute free space = budget - total. Build `categories` array with `name`, `chars`, `tokens = ceil(chars/4)`. Compute `fraction = total/budget`, `floor = 0.90 - 0.40*fraction`, run adviser for `score` + `hint`.
- `GET /context/revisions`: read `context_revisions` for the session, ordered by `revision DESC`, map to response.
- `POST /context/injections`: store the frozen breakdown keyed by `session_id` (in-memory map or `context_revisions`-adjacent table; prefer in-memory map with session-end cleanup, mirroring ADR-0028 purge).

Wire into `registerRoutes()` (mirror `registerCompactionRoutes`).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mnemonic/internal/http/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/http/
git commit -m "feat(mnemonic): /context/usage + /context/revisions + /context/injections HTTP routes"
```

---

## Task 10: `skillgrid context` TUI (Bubbletea)

**Files:**
- Create: `skillgrid-cli/internal/cmd/context.go`
- Create: `skillgrid-cli/internal/tui/context_model.go`
- Create: `skillgrid-cli/internal/tui/context_usage.go`
- Create: `skillgrid-cli/internal/tui/context_revisions.go`
- Create: `skillgrid-cli/internal/tui/context_injections.go`
- Create: `skillgrid-cli/internal/tui/context_test.go`
- Modify: `skillgrid-cli/internal/cmd/root.go` (register `context` subcommand)
- Modify: `skillgrid-cli/go.mod` (promote `bubbletea`, `lipgloss`, `bubbles` from indirect to direct)

**Interfaces:**
- Produces: `skillgrid context <session_id>` cobra command
- Produces: `tui.ContextModel` — Bubbletea `Model` with `screen int` (0=Usage, 1=Revisions, 2=Injection), `cursor int`, `expanded bool`, `data *UsageData`, `revisions []Revision`, `injections []Injection`, `stale bool`
- Consumes: `GET /context/usage` + `GET /context/revisions` over HTTP (2s poll)

**SATISFIES:** `happy path TUI renders usage grid` + `happy path TUI keyboard navigation` scenarios in `acceptance.feature`

- [ ] **Step 1: Write the failing test**

```go
func TestContextModel_InitialRender(t *testing.T) {
    // Seed a mock HTTP server returning usage at 72% + 3 revisions.
    // Launch the model; wait for first poll.
    // Assert View() contains "72%" and 36 filled blocks in the grid.
}
func TestContextModel_TabCyclesScreens(t *testing.T) {
    // Launch; assert screen 0 (Usage).
    // Send Tab → assert screen 1 (Revisions), View() shows "Rev 3".
    // Send Tab → assert screen 2 (Injection), View() shows "System prompt".
    // Send Tab → assert screen 0 (Usage) again.
}
func TestContextModel_ExpandRevision(t *testing.T) {
    // Navigate to Revisions screen, select rev 2, send Enter.
    // Assert View() shows the six-section messages content.
    // Send Enter again → collapses.
}
func TestContextModel_Quit(t *testing.T) {
    // Send "q" → assert tea.Quit in the resulting msg.
}
func TestContextModel_Refresh(t *testing.T) {
    // Seed mock server with 50% usage; launch; wait for poll.
    // Change mock to 80%; send "r"; wait for re-poll.
    // Assert View() now shows "80%".
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/tui/... -v`
Expected: FAIL with "undefined: ContextModel"

- [ ] **Step 3: Write minimal implementation**

`context_model.go`:
- `type ContextModel struct { screen, cursor int; expanded bool; data *UsageData; revisions []Revision; injections []Injection; stale bool; width, height int; helpVisible bool }`
- `Init()` → send first poll tick.
- `Update()`: handle `tea.KeyMsg` (`↑↓` move cursor, `Enter` toggle expand, `Tab`/`Shift+Tab` cycle screen, `1`/`2`/`3` jump, `r` force re-poll, `q`/`Esc` quit, `?` toggle help), `tea.WindowSizeMsg`, `pollTick` (re-fetch from HTTP).
- `View()` → dispatch to `viewUsage`/`viewRevisions`/`viewInjections` based on `screen`.

`context_usage.go`:
- 10×5 block grid: `filled = int(50 * fraction)`; colored by dominant category.
- Per-category horizontal bar: `barLen = int(30 * chars/budget)`; lipgloss color per category (blue=system, green=tools, yellow=injections, red=messages, purple=compaction, grey=free).
- Advisory line: `floor 0.61 · hint WOULD FIRE (score 0.75)` or `hint not firing (score 0.45)`.

`context_revisions.go`:
- Table: `Rev  N  HH:MM  NNNN tok  trigger`
- `steering` text on the line below each row (truncated to width).
- `Enter` → expand: show `messages` (six-section content) in a scrollable sub-panel.

`context_injections.go`:
- Table: `Name ............ NNNN tok  [view]`
- `Enter` → expand raw text.
- Footer: `Total frozen: NNNN tok (NN% of budget)`.

`context.go` (cmd):
- `cobra.Command{Use: "context <session_id>", Args: cobra.ExactArgs(1), Run: runContext}`
- `runContext`: open Bubbletea program with `tea.WithAltScreen()`, model polls `BASE_URL` (same env as plugins: `SKILLGRID_MNEMONIC_HTTP_URL` or `127.0.0.1:7438`).

Promote `bubbletea`/`lipgloss`/`bubbles` in `go.mod` (remove `// indirect`).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/tui/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/tui/ skillgrid-cli/internal/cmd/ skillgrid-cli/go.mod skillgrid-cli/go.sum
git commit -m "feat(skillgrid): skillgrid context TUI (Bubbletea, three screens, keyboard nav)"
```

---

## Task 11: Plugin Wiring for TUI Data

**Files:**
- Modify: `plugins/opencode/skillgrid-compaction.ts`
- Modify: `plugins/opencode/mnemonic-memory.ts`
- Test: (typecheck + `pnpm test`)

**Interfaces:**
- `skillgrid-compaction.ts`: on `experimental.session.compacting` → POST injection breakdown `{session_id, categories: [{name, chars}]}` to `/context/injections`.
- `mnemonic-memory.ts`: new `mem_context_usage` tool → `GET /context/usage?session_id=...` → `{categories, total_tokens, budget_tokens, fraction, floor, hint, score}`.
- `mnemonic-memory.ts`: new `mem_context_revisions` tool → `GET /context/revisions?session_id=...` → `[{revision, created_at, size_estimate, steering, messages, trigger}]`.

**SATISFIES:** `happy path context usage returns category breakdown` + `happy path TUI renders usage grid` scenarios in `acceptance.feature`

- [ ] **Step 1: Wire the injection POST**

In `skillgrid-compaction.ts`, in the `experimental.session.compacting` handler: extract the chars per section from the compacting payload (system prompt, tool definitions, AGENTS.md, skill prompts, MCP instructions) and POST `{session_id, categories: [...]}` to `/context/injections`.

- [ ] **Step 2: Add `mem_context_usage` tool**

In `mnemonic-memory.ts`, register `mem_context_usage` → `GET ${BASE}/context/usage?session_id=...` → return the JSON body.

- [ ] **Step 3: Add `mem_context_revisions` tool**

In `mnemonic-memory.ts`, register `mem_context_revisions` → `GET ${BASE}/context/revisions?session_id=...` → return the JSON array.

- [ ] **Step 4: Run typecheck + tests**

Run: `pnpm typecheck && pnpm lint && pnpm test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add plugins/opencode/
git commit -m "feat(mnemonic): wire injection POST + mem_context tools into TS plugins"
```

---

## Self-Review Checklist

- [ ] **Spec coverage:** All 7 requirements from the briefing have corresponding tasks (1→4+5, 2→2, 3→3, 4→6, 5→1+7, 6→5+8, 7→9+10+11).
- [ ] **Must-haves coverage:** Every truth is mapped to a task's acceptance.
- [ ] **One-way-door:** The `steering` column migration (Task 3) is tagged.
- [ ] **Placeholder scan:** No "TBD" or "implement later" found.
- [ ] **Type consistency:** `CompactionContext.Sections`, `Advice`, `SetCompaction`, `ContextModel`, `UsageData` signatures match across tasks.
- [ ] **Replace-first:** All plugin wiring in Tasks 8+11 targets the 5 new TS plugins.
- [ ] **No new dependencies:** Bubbletea/Lipgloss/Bubbles already in `go.mod` (indirect → direct promotion only).

## Execution Handoff

**"Blueprint written and committed. Two execution options:**

**1. Subagent-Driven (recommended)** — Dispatch a fresh subagent per task.

**2. Inline Execution** — Execute tasks in this session using skillgrid:simple-execution.

**Which approach?"**
