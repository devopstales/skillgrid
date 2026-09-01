---
name: sdd-tasks
description: "Break an SDD change into phased, dependency-ordered implementation tasks from the proposal, design, and spec. Use to launch task planning after sdd-spec and before sdd-apply. Inherits the design's applicable threat-matrix rows as RED-test tasks before their production code, forecasts review workload against the 400-line budget with chained-PR work units, and persists to both openspec and Mnemonic. Uses Mnemonic memory + code index; no external binaries."
metadata:
  family: sdd
  phase-order: "propose → design → spec → tasks"
  prev: [sdd-propose, sdd-design, sdd-spec]
  next: [sdd-apply]
  artifact: tasks
  delegate_only: true
---

# sdd-tasks

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-tasks` sub-agent **unless** you loaded this skill directly through the `skill()` tool.

- **Sub-agent (primary)**: you were delegated here by the SDD orchestrator. Continue with the phase work below. Do not re-delegate. Do not call the `skill()` tool again.
- **Orchestrator (skill() loaded this directly)**: STOP. Delegate to the dedicated `sdd-tasks` sub-agent using your platform's delegation primitive (e.g. `task(...)`) instead of doing the work inline.

## Purpose

You are the TASKS phase. You take the proposal, design, and spec and produce a concrete `tasks.md`: implementation work broken into **dependency-ordered phases**, each task **small and verifiable**, with the design's applicable threat-matrix rows carried forward as **RED-test tasks that land before their production code**, and a **review-workload forecast** that protects the reviewer's 400-line budget.

Phase order is `propose → design → spec → tasks`. You run last in the planning chain and hand a `sdd-apply`-ready task list to the implementation phase. Two consequences that drive this phase:

1. Every **design threat-matrix row marked `Applicable`** MUST become an explicit RED-test task, ordered before the production task it guards. This is the last planning checkpoint: a row that never becomes a task here is a handoff gap from design/spec, not a tasks defect.
2. The **spec is your test contract.** Testing tasks MUST reference the concrete scenarios in the delta spec (`openspec/changes/{change-name}/specs/`), not generic "add tests" wishes.

## What You Receive

From the orchestrator:

- **Change name** (kebab-case, e.g. `add-dark-mode`)
- **Delivery strategy** (`ask-on-risk` (default) | `auto-chain` | `single-pr` | `exception-ok`) — the four-value domain for the review-workload guard; any other value is invalid (report it, do not guess).
- Optional: **ticket/issue id** (carry-through to `sdd-apply`'s commit close-token per `_shared/conventions/commits.md`; tasks itself does not use it)
- Optional: a `## Skills to load before work` block

**Artifact store mode is `hybrid` — the only mode for this phase.** Every run does BOTH: writes `openspec/changes/{change-name}/tasks.md` **and** persists to Mnemonic under `sdd/{change-name}/tasks`. There is no filesystem-only or memory-only mode here; a mode token of `openspec` / `engram-compat` / `none` from the orchestrator is honored as `hybrid` for this phase. Do not branch your behavior on the mode.

## Execution + Persistence Conventions

Follow, on each save, rather than restating here:

- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — Mnemonic save shape (`title == topic_key`, `scope: "project"`, active `session_id`; **no** `project:` parameter, **no** `capture_prompt` field; `mem_search` returns previews — always `mem_get_observation(id)` for full content).
- [`../_shared/conventions/openspec.md`](../_shared/conventions/openspec.md) — change-folder layout, `rules.tasks` from `openspec/config.yaml`, and that `tasks.md` is later updated by `sdd-apply` (marks `[x]`).
- [`../_shared/conventions/mnemonic-code-indexing.md`](../_shared/conventions/mnemonic-code-indexing.md) — the `code_*` ladder, used when a task's concrete file path needs to be confirmed against real code (see Step 2).
- [`references/threat-matrix.md`](references/threat-matrix.md) — the boundary rows the design filled in; the **applicable** ones feed this phase's RED-test tasks (local copy of `sdd-design`'s matrix for a self-contained skill).

## Skill Loading

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first.
2. Otherwise recover the required inputs from Mnemonic (previews are not enough — always fetch full content):
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/proposal")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**.
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/design")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; its threat-matrix applicable rows are your primary RED-test input.
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/spec")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; your testing tasks reference its scenarios.
   - `skillgrid-mnemonic_mem_search(query: "sdd-init/{project}")` → `skillgrid-mnemonic_mem_get_observation(id)` — detected project facts (stack, testing, tracker).
3. Read from the change folder (the filesystem is primary in hybrid mode): `openspec/changes/{change-name}/design.md` and every `openspec/changes/{change-name}/specs/{domain}/spec.md`.
4. Read `openspec/config.yaml` if present — `context:` and `rules.tasks` bind this phase.

## What to Do

### Step 1: Inherit the Design's Applicable Threat Rows

Read the design's `## Threat Matrix`. For **each row marked `Applicable`**, you MUST emit an explicit RED-test task that carries the row's concrete adversarial case and its expected safe/failure behavior, ordered **before** the production task that could regress it.

- Preserve the concrete case and behavior from design (and the matching spec scenario) — do not re-derive it into a vaguer "test the edge case".
- `N/A` rows need no task.
- If an applicable row has **no covering spec scenario**, flag it in the envelope `risks` — the spec is the enforcement boundary upstream; do not silently invent the test here, name the gap.
- Do not mark a row `N/A` on a guess here — you are inheriting the design's call, not making it.

### Step 2: Analyze the Design and Confirm Real Files

From the design's `## File Changes` and the spec, identify:

- Every file to be created / modified / deleted — with a **concrete path** (not an abstraction).
- The **dependency order** — what must exist before something that references it.
- Testing requirements per component, mapped to the spec scenarios.
- Integration/wiring seams (routes, config, UI) that only work once earlier phases are done.

Confirm any uncertain path against the code index — a task that cites a file that does not exist is a task with a hole:

```
skillgrid-mnemonic_code_status              # stale? file_count 0?
  → if stale, skillgrid-mnemonic_code_index
skillgrid-mnemonic_code_search(query: "<file-or-symbol>", limit: 20)
  → skillgrid-mnemonic_code_read(path: <hit.path>, start_line: <hit.start_line>, end_line: <hit.end_line>)
```

Search first, then read the slice — never read a whole file speculatively. If the index is empty and `code_index` cannot run, confirm the paths with the filesystem read tool and note it in `risks`.

### Step 3: Write tasks.md

Create / update the file in the change folder (hybrid mode always writes it):

```
openspec/changes/{change-name}/
├── proposal.md
├── design.md
├── specs/
└── tasks.md               ← you create (or update) this
```

If a `tasks.md` already exists, **READ it first and UPDATE it** — `sdd-apply` marks tasks `[x]` in place, so a re-run must preserve completed state and only adjust the pending set it affects.

#### Task file format

```markdown
# Tasks: {Change Title}

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | <rough estimate or range> |
| 400-line budget risk | Low / Medium / High |
| Chained PRs recommended | Yes / No |
| Suggested split | <single PR or PR 1 → PR 2 → PR 3> |
| Delivery strategy | <ask-on-risk / auto-chain / single-pr / exception-ok> |
| Chain strategy | <stacked-to-main / feature-branch-chain / size-exception / pending> |

Decision needed before apply: {Yes | No}
Chained PRs recommended: {Yes | No}
Chain strategy: {stacked-to-main | feature-branch-chain | size-exception | pending}
400-line budget risk: {Low | Medium | High}

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | <standalone deliverable> | PR 1 | <smallest proving command> | <real scenario/command or N/A + reason> | <files/behavior removable without unrelated rollback> |
| 2 | <standalone deliverable> | PR 2 | <smallest proving command> | <real scenario/command or N/A + reason> | <independent revert boundary> |

## Phase 1: {Phase Name} (e.g., Infrastructure / Foundation)

- [ ] 1.1 {Concrete action — which file, what change}
- [ ] 1.2 {Concrete action}

## Phase 2: {Phase Name} (e.g., Core Implementation)

- [ ] 2.1 {Concrete action}
- [ ] 2.2 {Write the RED test for {design threat row}: {concrete case + expected behavior}}
- [ ] 2.3 {Make 2.2 pass — {production code path}}

## Phase 3: {Phase Name} (e.g., Testing / Verification)

- [ ] 3.1 {Cover spec scenario `{domain}/…`: {behavior}}
- [ ] 3.2 {Cover spec scenario `{domain}/…`: {edge case}}
- [ ] 3.3 {Verify integration between {A} and {B}}

## Phase 4: {Phase Name} (e.g., Cleanup / Documentation)

- [ ] 4.1 {Update docs/comments for {path}}
- [ ] 4.2 {Remove temporary code {path}}
```

The four plain-text `Decision needed / Chained PRs recommended / Chain strategy / 400-line budget risk` lines are the **guard contract** — downstream phases and any reviewer match them literally. You may keep the table for readability, but the plain-text lines are load-bearing.

#### Task writing rules

Each task MUST be all four of:

| Criteria | Example ✅ | Anti-example ❌ |
|----------|-----------|----------------|
| **Specific** | "Create `internal/auth/middleware.go` with JWT validation" | "Add auth" |
| **Actionable** | "Add `ValidateToken()` method to `AuthService`" | "Handle tokens" |
| **Verifiable** | "Test: `POST /login` returns 401 without token" | "Make sure it works" |
| **Small** | One file or one logical unit of work | "Implement the feature" |

Additional rules:

- **RED before GREEN for threat rows.** For every applicable design threat-row, the RED-test task precedes the production task that guards it (Step 1).
- **If the project uses TDD** (`rules.apply.tdd: true` in `openspec/config.yaml`), order each unit RED (failing test) → GREEN (make it pass) → REFACTOR (clean up).
- **Testing tasks name a spec scenario** — not "add tests".
- Use hierarchical numbering (`1.1`, `1.2`, `2.1`, …).
- NEVER vague tasks: "implement feature", "add tests", "wire it up".

#### Phase organization

```
Phase 1: Foundation / Infrastructure
  └─ new types, interfaces, DB changes, config — things other tasks depend on

Phase 2: Core Implementation
  └─ main logic, business rules — the meat of the change + RED tests for threat rows

Phase 3: Integration / Wiring
  └─ connect components, routes, UI wiring — make it work together

Phase 4: Testing / Verification
  └─ unit / integration / e2e against spec scenarios

Phase 5: Cleanup (if needed)
  └─ docs, remove dead code, polish
```

Order MUST respect dependency: Phase 1 tasks should not depend on Phase 2. (Skip a phase that has no work rather than forcing all five.)

#### Review Workload Forecast rules

Before finalizing, estimate whether the implementation is likely to exceed the **400 changed-line review budget** (`additions + deletions`). This is a planning guard, not an exact diff count. Use available signals: number of files, phases, integration points, tests, migrations, and how many concerns the change crosses.

If the estimate is **High** or likely above 400 lines:

1. Mark `Chained PRs recommended` as `Yes`.
2. Split into **work units** that can become chained or stacked PRs — each with a clear start, clear finish, autonomous scope, verification, and a rollback boundary.
3. **Ask the user which chain strategy to use** (a team decision):
   - **Stacked PRs to main** — each PR merges to main in order. Fast, fix on the go. Best for independent slices.
   - **Feature Branch Chain** — PR #1 targets the feature/tracker branch; later PRs target the previous PR branch so each child diff stays focused; only the tracker merges to main. Best for rollback control.
   - **`size:exception`** — keep as one PR with maintainer approval. Best for generated code, migrations, vendor diffs.
4. Set `Decision needed before apply` from the **delivery strategy**:
   - `ask-on-risk` → `Yes` (orchestrator asks before apply)
   - `auto-chain` → `No` (orchestrator proceeds with slice 1 using the chosen strategy)
   - `single-pr` → `Yes` (orchestrator requires `size:exception` before apply)
   - `exception-ok` → `No` (maintainer accepted `size:exception`)

For `feature-branch-chain`, name the intended base boundary per work unit: PR #1 base = tracker; PR #2 base = PR #1 branch; PR #3 base = PR #2 branch.

Put the forecast near the top of the artifact so the user sees it before implementation starts. Do not bury it in prose.

### Step 4: Self-Check (no external validator binary)

In place of an admission validator, before you persist confirm each — fix any failure before returning `success`, else return `partial` with the failed item in `risks`:

1. Every task cites a **concrete** file path or is an explicit test/verification action (no "implement feature").
2. Every applicable design threat-row has a **RED-test task ordered before** the production task it guards (Step 1). Every applicable threat-row named in the spec is covered.
3. Every testing task references a **specific spec scenario** (or an explicit threat-row case) — none read "add tests".
4. Phases respect **dependency order** (no Phase N task depends on a later phase).
5. **Four plain-text guard lines present** in the forecast exactly as specified.
6. If risk is High / above 400 lines, `Chained PRs recommended: Yes` AND every suggested work unit names a focused test command, a runtime harness (or `N/A + reason`), and a rollback boundary.
7. Word count within the **Size Budget** (Rules).
8. Filesystem `tasks.md` and the Mnemonic content are the **same content**.

### Step 5: Persist Artifact (hybrid — MANDATORY, do not skip)

Follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md). Hybrid = BOTH writes:

1. **Filesystem** — `openspec/changes/{change-name}/tasks.md` (already written in Step 3; ensure the change folder and prior artifacts exist per `openspec.md`).
2. **Mnemonic** — start one session, then save the same content:

```
sid = skillgrid-mnemonic_mem_session_start(title: "sdd/{change-name}/tasks")

skillgrid-mnemonic_mem_save(
  title:      "sdd/{change-name}/tasks",
  topic_key:  "sdd/{change-name}/tasks",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "{full tasks.md markdown}"
)
```

`topic_key` upserts — re-running the phase replaces the observation in place, not duplicates. Mnemonic save notes: `title == topic_key` exactly; `scope: "project"`; pass the active `session_id`; there is **no** `project:` parameter and **no** `capture_prompt` field in the Mnemonic schema — omit both.

### Step 6: Return Envelope

**Your FINAL output MUST be text — not a tool call.** Do the `mem_save` (Step 5) *before* this text. A trailing tool call buries the analysis in the tool result; returning text is what the orchestrator reads back.

```markdown
## Tasks Created
**Change**: {change-name}
**Location**: `openspec/changes/{change-name}/tasks.md` · Mnemonic `sdd/{change-name}/tasks` (hybrid)

**Status**: success | partial | blocked
**Executive summary**: 1–3 sentences.

### Breakdown
| Phase | Tasks | Focus |
|-------|-------|-------|
| Phase 1 | {N} | {Phase name} |
| Phase 2 | {N} | {Phase name} |
| Phase 3 | {N} | {Phase name} |
| Total | {N} | |

**Threat-matrix handoff**: {K applicable rows → each a RED-test task} | {not applicable} | {gaps flagged}
**TDD applied**: {yes — RED/GREEN/REFACTOR} | {no} | {n/a}

### Review Workload Forecast
- Estimated changed lines: {estimate or range}
- 400-line budget risk: {Low | Medium | High}
- Chained PRs recommended: {Yes | No}
- Delivery strategy: {ask-on-risk | auto-chain | single-pr | exception-ok}
- Decision needed before apply: {Yes | No}
- Suggested work-unit PR split: {brief list or "Not needed"}

**Mnemonic**: observation `{id or 'none'}` · session `{sid}`
**Open questions**: {list, or "None"}
**Skill resolution**: paths-injected | fallback-registry | none
**Risks**: {list, or "None"}
**Next**: sdd-apply
```

Close the final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up. Do not call `mem_session_summary` in a sub-agent context — the orchestrator owns session close.

## Rules

- **Always reference concrete file paths** in tasks — never "a new middleware" / "the handler".
- Tasks MUST be **ordered by dependency**: Phase 1 does not depend on Phase 2.
- **TDD threat rows**: every applicable design threat-row becomes a RED-test task before its production task (Step 1); if TDD is on, the unit is RED → GREEN → REFACTOR.
- Testing tasks MUST reference **specific spec scenarios**.
- Each task completable in **one session**; if it feels too big, split it.
- Use hierarchical numbering (`1.1`, `2.1`, …).
- Apply any `rules.tasks` from `openspec/config.yaml`.
- **Size budget**: the tasks artifact MUST be **under 530 words**. Each task: 1–2 lines max. Checklist format, not paragraphs.
- **Review workload guard**: ALWAYS include the forecast with the four plain-text guard lines. If likely above 400 changed lines, recommend chained PRs and honor the received delivery strategy for whether a decision/exception is needed before apply.
- **Work-unit evidence**: every suggested work unit names a Focused test command, a Runtime harness (or explicit `N/A` + reason), and a Rollback boundary.
- **Hybrid is the only mode** — always write the filesystem file AND save to Mnemonic; never branch on `openspec` / `none` for this phase.
- No external binaries. Mnemonic (`mem_*`) and the code index (`code_*`) are the only knowledge sources; no `gentle-ai`, no `gentleman-ai`, no `sdd-phase-common.md`, no CLI validator.
- Return envelope per Step 6 — final action is text, not a tool call.

## Gotchas

- You are **inheriting** threat-row tests from design (via spec), not re-deriving them. If a `sdd-tasks` run invents a RED test from prose instead of carrying the spec's scenario / the design's case, the upstream design or spec was carrying it, not the spec — flag the gap rather than papering over it.
- `mem_search` returns **300-char previews**. Never use a preview as source material — always `mem_get_observation(id)` for full content of the proposal, design, or spec. A 300-char preview of a 2000-char design loses most of it.
- **`rules.tasks` + TDD interplay.** If `rules.apply.tdd: true`, do not emit a bare "implement X" task — emit RED (failing test) → GREEN (make it pass) as separate lines. A single line that does both is not TDD.
- **The four plain-text guard lines are the contract.** A reviewer or `sdd-apply` matches them literally — `Decision needed before apply:`, `Chained PRs recommended:`, `Chain strategy:`, `400-line budget risk:`. If you drop one or reword it, the guard silently no-ops.
- **Chain strategy is a human/team decision, not yours.** Forecast it, recommend it, and if `Decision needed before apply: Yes` STOP and let the orchestrator ask. Do not pick a chain strategy and proceed on `ask-on-risk` / `single-pr`.
- **Updating an existing `tasks.md`.** `sdd-apply` marks lines `[x]`. On re-run, READ first and preserve completed state — overwriting resets progress and re-opens done work.
- **Mnemonic ≠ Engram.** No `project:` parameter, no `capture_prompt`. `title == topic_key`, `scope: "project"`, active `session_id`. (See `conventions/mnemonic-memory.md` § Mnemonic Tool Mapping.)
- **Hybrid writes both, or it is half a save.** A `tasks.md` on disk with no Mnemonic save (or vice versa) breaks recovery — the filesystem survives branch switches, Mnemonic survives `/clear`; you need both.
- If the code index is `stale: true`, run `code_index` before `code_search`. A task citing a file you confirmed on a stale index is a task with a hole.
- Do not commit from this phase. Tasks is the plan; `sdd-apply` commits the DO. Commit conventions live in `_shared/conventions/commits.md` and the close-token footer applies to the apply commit, not this phase.

## References

- [`../sdd-design/SKILL.md`](../sdd-design/SKILL.md) — upstream; its `## Threat Matrix` applicable rows feed Step 1.
- [`../sdd-spec/SKILL.md`](../sdd-spec/SKILL.md) — upstream; its scenarios are the testing-task contract.
- [`../sdd-propose/SKILL.md`](../sdd-propose/SKILL.md) — upstream; its scope/approach bounds what these tasks should cover.
- [`references/threat-matrix.md`](references/threat-matrix.md) — the boundary rows the design may have marked applicable (local copy of `sdd-design`'s matrix).
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape, session protocol, recovery ladder.
- [`../_shared/conventions/openspec.md`](../_shared/conventions/openspec.md) — change-folder layout, `rules.tasks`, and that `sdd-apply` updates this `tasks.md`.
- [`../_shared/conventions/mnemonic-code-indexing.md`](../_shared/conventions/mnemonic-code-indexing.md) — the `code_status → code_index → code_search → code_read` ladder for confirming real file paths.
- [`../_shared/conventions/commits.md`](../_shared/conventions/commits.md) — commit contract (relevant to the downstream `sdd-apply` commit, not this phase).
