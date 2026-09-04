---
name: sdd-tasks
description: "Break an SDD change into a step tree (steps/<NN-name>/) and per-step execution punch-lists from the intent and plan. Use to launch task planning after sdd-design and before sdd-spec. Owns step allocation and NN numbering, carries the plan's applicable threat-matrix rows into per-step RED-test tasks ordered before their production code, and persists to both the filesystem and Mnemonic. Uses Mnemonic memory + code index; no external binaries."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  author: devopstales
  version: "2.0"
  family: sdd
  part-of: skillgrid
  phase-order: "init → explore → propose → design → tasks → spec → apply → verify → archive"
  prev: [sdd-propose, sdd-design]
  next: [sdd-spec]
  artifact: tasks (per step)
  delegate_only: true
---

# sdd-tasks

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-tasks` sub-agent **unless** you loaded this skill directly through the `skill()` tool.

- **Sub-agent (primary)**: you were delegated here by the SDD orchestrator. Continue with the phase work below. Do not re-delegate. Do not call the `skill()` tool again.
- **Orchestrator (skill() loaded this directly)**: STOP. Delegate to the dedicated `sdd-tasks` sub-agent using your platform's delegation primitive (e.g. `task(...)`) instead of doing the work inline.

## Purpose

You are the TASKS phase. You take the intent (WHY + Step Blueprint) and the plan (HOW + Impacted Files map + per-step WHAT) and produce the **step tree** — one folder per step under `docs/skillgrid/changes/<NNN-slug>/steps/` — each holding a `tasks.md` execution punch-list. This phase **owns the step tree and the NN numbering**: you create the folders `sdd-spec` will fill with `acceptance.feature` and `sdd-verify` will gate with `verification.md`, and you write every step's punch-list.

Phase order is `… → design → tasks → spec → …`. You run **before** `sdd-spec` because `acceptance.feature` lives inside `steps/<NN-name>/` — the tree must exist before acceptance is authored. Two consequences that drive this phase:

1. You are the **step allocator**. The intent's Step Blueprint names the steps; the plan's Impacted Files map assigns each file to a step. You reconcile the two and create `steps/<NN-name>/` for each, resolving ordering and dependencies.
2. Every **plan threat-matrix row marked `Applicable`** MUST become an explicit RED-test task in the owning step's `tasks.md`, ordered **before** the production task it guards.

## What You Receive

From the orchestrator:

- **Change id** — `<NNN-slug>` (e.g. `001-oauth-login`). The folder exists with `intent.md` and `plan.md`.
- **Delivery strategy** (`ask-on-risk` (default) | `auto-chain` | `single-pr` | `exception-ok`) — the four-value domain for the review-workload/PR-split guard; any other value is invalid (report it, do not guess). Applies at the change level; each step still names its own work unit.
- Optional: **ticket/issue id** (carry-through to `sdd-apply`'s commit close-token per `_shared/conventions/commits.md`; tasks itself does not use it)
- Optional: a `## Skills to load before work` block

**Artifact store mode is `hybrid` — the only mode for this phase.** Every run does BOTH: creates the `steps/<NN-name>/` folders and writes each `steps/<NN-name>/tasks.md`, **and** persists to Mnemonic under `sdd/<NNN-slug>/tasks` (a single concatenated observation). The filesystem write and the Mnemonic save are each their own obligations — the Mnemonic save does not stand in for the file. There is no filesystem-only or memory-only mode here; hybrid is the only mode. Do not branch your behavior on the mode.

## Vertical Slice Contract

Every step in the step tree MUST be a **tracer-bullet vertical slice**: a narrow but COMPLETE path through every relevant layer. A step is not vertical if it only touches one layer while deferring others to later steps.

Rules:
- A completed step must be **demoable or verifiable on its own** — no "layer 1 done, layers 2-3 coming later"
- Each step is sized to fit in a **single fresh context window** — if a step crosses >2 distinct files or exceeds ~30 task lines, flag it in `risks` rather than silently splitting (re-splitting changes the NN tree and invalidates `sdd-propose`'s reservation)
- Work the **frontier first**: steps with no blockers come before steps that depend on them
- Every step declares its blocking edges explicitly as `Depends on:` — a step with no dependency starts immediately
- Prefactoring that makes the slice easier belongs **inside** the slice, not as a separate horizontal step

**Wide refactors are the exception.** A wide refactor is one mechanical change (rename a column, retype a shared symbol) whose blast radius fans across the whole codebase, so no single vertical slice can land green. Sequence it as **expand-contract**:
1. **Expand**: add the new form beside the old so nothing breaks
2. **Migrate**: move call sites over in batches sized by blast radius (per package/directory), each batch its own step blocked by the expand
3. **Contract**: delete the old form once no caller remains, in a step blocked by every migrate batch

Even the migrate batches should be vertical where possible — each batch must leave the system in a working state.

## Execution + Persistence Conventions

Follow, on each save, rather than restating here:

- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — Mnemonic save shape (`title == topic_key`, `scope: "project"`, active `session_id`; **no** `project:` parameter, **no** `capture_prompt` field; `mem_search` returns previews — always `mem_get_observation(id)` for full content).
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — change-folder layout, NN numbering, `rules.tasks` from `docs/skillgrid/config.yaml`, and that each step's `tasks.md` is later marked `[x]` by `sdd-apply`.
- [`../_shared/conventions/mnemonic-code-indexing.md`](../_shared/conventions/mnemonic-code-indexing.md) — the `code_*` ladder, used when a task's concrete file path needs to be confirmed against real code (see Step 3).
- [`references/threat-matrix.md`](references/threat-matrix.md) — the boundary rows the plan filled in; the **applicable** ones feed this phase's RED-test tasks (local copy of `sdd-design`'s matrix for a self-contained skill).

## Skill Loading

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first.
2. Otherwise recover the required inputs from Mnemonic (previews are not enough — always fetch full content):
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/intent")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; the **Step Blueprint** is your step contract.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/plan")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; its Impacted Files map, per-step WHAT blocks, and threat-matrix applicable rows are your primary inputs.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/research")` → `skillgrid-mnemonic_mem_get_observation(id)` — optional context.
   - `skillgrid-mnemonic_mem_search(query: "sdd-init/{project}")` → `skillgrid-mnemonic_mem_get_observation(id)` — detected project facts (stack, testing, tracker).
3. Read from the change folder (the filesystem is primary in hybrid mode): `docs/skillgrid/changes/<NNN-slug>/intent.md` and `docs/skillgrid/changes/<NNN-slug>/plan.md`.
4. Read `docs/skillgrid/config.yaml` if present — `context:` and `rules.tasks` bind this phase.

## What to Do

### Step 1: Allocate the Step Tree

Reconcile two inputs into one canonical step list:

- **Intent's Step Blueprint** — the steps that were named up-front (each `01-<slug>`, `02-<slug>`, …).
- **Plan's Impacted Files map** — each file's owning step (the `Step` column).
- **Plan's per-step WHAT blocks** — what each step must deliver.

Rules:

- The step **set** comes from the intent. You do not invent steps the intent did not name. If the plan's Impacted Files map references a step that is not in the intent, flag it in the envelope `risks` — that is a plan defect (a file with no home), not a tasks decision.
- **NN numbering** is 2-digit zero-padded, sequential, and strictly follows the intent's Step Blueprint order. `01-…`, `02-…`, `03-…`. The NN value is fixed once you create the folder — `sdd-apply` and `sdd-verify` do not renumber.
- A step's **goal** is taken from the intent's Step Blueprint entry and restated in the step's `tasks.md` header.
- **Frontier-first ordering**: steps with no blockers come before steps that depend on them. Dependency order is part of the step tree contract — `sdd-apply` works the frontier.
- **Vertical slice check**: every step must cut a complete path through the layers it touches. A step that only modifies one layer while another needed layer is deferred to a later step is horizontal, not vertical — flag it in `risks`.
- **Wide refactor exception**: if the change is a mechanical blast-radius change (rename, retype, global migration), do not force it into a single vertical slice. Encode it as expand-contract: an expand step first, then migrate batches (each blocked by expand, each its own step), then a contract step blocked by all migrate batches.
- Create every step folder: `mkdir -p docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/`. Do not create folders for `acceptance.feature` or `verification.md` — those files are authored by `sdd-spec` and `sdd-verify`, but the folders exist because **you** created them.

### Step 2: Inherit the Plan's Applicable Threat Rows (per step)

Read the plan's `## Threat Matrix`. For **each row marked `Applicable`**, the plan named an owning step. In that step's `tasks.md`, emit an explicit RED-test task that carries the row's concrete adversarial case and its expected safe/failure behavior, ordered **before** the production task that could regress it.

- Preserve the concrete case and behavior from the plan — do not re-derive it into a vaguer "test the edge case".
- `N/A` rows need no task.
- If an applicable row's owning step is missing from the intent's Step Blueprint, flag it in the envelope `risks` — the plan and intent disagree; do not silently reassign.
- Do not mark a row `N/A` on a guess here — you are inheriting the plan's call, not making it.

### Step 3: Analyze the Plan and Confirm Real Files (per step)

For each step, from the plan's `## Impacted Files Map` (rows assigned to that step) and the per-step WHAT block:

- Every file in the step — with a **concrete path** (not an abstraction).
- The **dependency order** within the step — what must exist before something else in the same step.
- **Inter-step dependencies** — which step must be done before another (e.g. `01-db-migration` before `02-api-route`). Encode these as a top-of-file note in the later step's `tasks.md`.
- Testing requirements per step, mapped to the per-step WHAT (these will become the spec's scenarios after `sdd-spec` runs — reference the WHAT bullets, not the scenario names, since they do not exist yet).
- Integration/wiring seams that only work once earlier steps are done.

Confirm any uncertain path against the code index — a task that cites a file that does not exist is a task with a hole:

```
skillgrid-mnemonic_code_status              # stale? file_count 0?
  → if stale, skillgrid-mnemonic_code_index
skillgrid-mnemonic_code_search(query: "<file-or-symbol>", limit: 20)
  → skillgrid-mnemonic_code_read(path: <hit.path>, start_line: <hit.start_line>, end_line: <hit.end_line>)
```

Search first, then read the slice — never read a whole file speculatively. If the index is empty and `code_index` cannot run, confirm the paths with the filesystem read tool and note it in `risks`.

### Step 4: Write each step's tasks.md

Create / update each file in its step folder (hybrid mode always writes it):

```
docs/skillgrid/changes/<NNN-slug>/
├── intent.md
├── plan.md
└── steps/
    ├── 01-<name>/
    │   └── tasks.md      ← you create (or update) this
    ├── 02-<name>/
    │   └── tasks.md
    └── 03-<name>/
        └── tasks.md
```

If a step's `tasks.md` already exists (re-run), **READ it first and UPDATE it** — `sdd-apply` marks tasks `[x]` in place, so a re-run must preserve completed state and only adjust the pending set it affects.

#### tasks.md format (per step)

```markdown
# Tasks: {NNN-slug} — Step {NN}-{name}

> Goal: {one-line goal from intent Step Blueprint}
> Depends on: {none | 01-…, 02-…}

## Review Workload Forecast (change-level, present in step 01's tasks.md; carried by reference in others)
| Field | Value |
|---|---|
| Estimated changed lines (this step) | <estimate> |
| Estimated changed lines (change) | <estimate> |
| 400-line budget risk (change) | Low / Medium / High |
| Chained PRs recommended (change) | Yes / No |
| Delivery strategy | <ask-on-risk / auto-chain / single-pr / exception-ok> |
| Chain strategy | <stacked-to-main / feature-branch-chain / size-exception / pending> |

Decision needed before apply: {Yes | No}
Chained PRs recommended: {Yes | No}
Chain strategy: {stacked-to-main | feature-branch-chain | size-exception | pending}
400-line budget risk: {Low | Medium | High}

## Execution

- [ ] {NN}.1 {Concrete action — which file, what change}
- [ ] {NN}.2 {Concrete action}
- [ ] {NN}.3 {Write the RED test for {plan threat row}: {concrete case + expected behavior}}   ← RED before GREEN
- [ ] {NN}.4 {Make {NN}.3 pass — {production code path}}
- [ ] {NN}.5 {Cover WHAT: {bullet from plan's Step WHAT block for this step}}
```

The four plain-text `Decision needed / Chained PRs recommended / Chain strategy / 400-line budget risk` lines are the **guard contract** — downstream phases and any reviewer match them literally. They live in **step 01's `tasks.md`** as the change-level forecast; every other step's `tasks.md` carries the same four lines by reference (or a one-line pointer to `01-…/tasks.md`) so a reviewer reading any one step sees the change-level guard.

#### Task writing rules

Each task MUST be all four of:

| Criteria | Example ✅ | Anti-example ❌ |
|---|---|---|
| **Specific** | "Create `db/migrations/001_oauth_token.go`" | "Add auth" |
| **Actionable** | "Add `ValidateToken()` method to `auth.Service`" | "Handle tokens" |
| **Verifiable** | "Test: `POST /login` returns 401 without token" | "Make sure it works" |
| **Small** | One file or one logical unit of work | "Implement the feature" |

Additional rules:

- **RED before GREEN for threat rows.** For every applicable plan threat-row in this step, the RED-test task precedes the production task that guards it (Step 2).
- **If the project uses TDD** (`rules.apply.tdd: true` in `docs/skillgrid/config.yaml`), order each unit RED (failing test) → GREEN (make it pass) → REFACTOR (clean up).
- **Testing tasks name a WHAT bullet** from the plan's per-step WHAT block — not "add tests". (`sdd-spec` will re-attach the scenario name after `acceptance.feature` exists; do not reference scenario names you have not seen.)
- Use hierarchical numbering `{NN}.{i}` (`01.1`, `01.2`, `02.1`, …) where `NN` is the step and `i` the in-step index.
- NEVER vague tasks: "implement feature", "add tests", "wire it up".
- **One step, one deliverable.** If a step's punch-list exceeds ~30 lines or crosses >2 distinct files, re-decompose it — but re-decomposition changes the step tree (new NN), which means going back to `sdd-propose` and re-reserving. Flag in `risks` rather than silently splitting.
- **Vertical slice**: a step must deliver a complete, verifiable path through the layers it touches. Do not split a single behavior across multiple steps as "schema here, API there, UI later" — that is horizontal slicing. The step's WHAT block is the behavior contract; tasks must implement enough of that behavior to verify it end-to-end within the step.
- **Wide refactor exception**: for mechanical blast-radius changes, use expand-contract instead of a single vertical slice. The expand step adds the new form beside the old; migrate batches move call sites over (each batch its own step, blocked by expand, sized by blast radius); the contract step deletes the old form (blocked by all migrate batches). Each batch and each phase must leave the system in a working state.

### Step 5: Self-Check (no external validator binary)

In place of an admission validator, before you persist confirm each — fix any failure before returning `success`, else return `partial` with the failed item in `risks`:

1. Every step in the intent's Step Blueprint has a `steps/<NN-name>/tasks.md` file.
2. Every task in every step cites a **concrete** file path or is an explicit test/verification action (no "implement feature").
3. Every applicable plan threat-row has a **RED-test task ordered before** the production task it guards, in the owning step (Step 2).
4. Every file in the plan's Impacted Files map is assigned to exactly one step (`Step` column) and appears in that step's tasks.
5. Inter-step dependencies are encoded as `Depends on:` notes in the later step.
6. **Vertical slice**: every step's tasks collectively deliver the step's WHAT block as a complete, verifiable behavior — no step is a horizontal single-layer pass that defers the rest to later steps. If the change is a wide refactor, the expand-contract sequence is present and each phase leaves the system working.
7. **Frontier order**: steps with no `Depends on:` appear before steps that list them as dependencies; no step depends on a step that comes after it in NN order.
8. **Four plain-text guard lines present** in at least step 01's `tasks.md` exactly as specified; other steps carry them by reference.
9. If change-level risk is High / above 400 lines, `Chained PRs recommended: Yes` AND the delivery strategy is honored.
10. Word count per step's `tasks.md` within the **Size Budget** (Rules).
11. Filesystem `steps/*` tree and the Mnemonic content are **consistent** (the Mnemonic observation enumerates every step folder and its task count).

### Step 6: Persist Artifact (hybrid — MANDATORY, do not skip)

Follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md). Hybrid = BOTH writes:

1. **Filesystem** — every `steps/<NN-name>/tasks.md` (already written in Step 4).
2. **Mnemonic** — start one session, then save a single concatenated observation per change:

```
sid = skillgrid-mnemonic_mem_session_start(title: "sdd/<NNN-slug>/tasks")

skillgrid-mnemonic_mem_save(
  title:      "sdd/<NNN-slug>/tasks",
  topic_key:  "sdd/<NNN-slug>/tasks",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "## Change-level forecast\n{four guard lines}\n\n## Step 01-{name}\n{full 01 task list}\n\n## Step 02-{name}\n{full 02 task list}\n..."
)
```

`topic_key` upserts — re-running the phase replaces the observation in place, not duplicates. Mnemonic save notes: `title == topic_key` exactly; `scope: "project"`; pass the active `session_id`; there is **no** `project:` parameter and **no** `capture_prompt` field in the Mnemonic schema — omit both.

The file must actually exist on disk at `docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/tasks.md` — a Mnemonic save without the file is incomplete.

### Step 7: Return Envelope

**Your FINAL output MUST be text — not a tool call.** Do the `mem_save` (Step 6) *before* this text. A trailing tool call buries the analysis in the tool result; returning text is what the orchestrator reads back.

```markdown
## Step Tree Created
**Change**: {NNN-slug}
**Location**: `docs/skillgrid/changes/<NNN-slug>/steps/` · Mnemonic `sdd/<NNN-slug>/tasks` (hybrid)

**Status**: success | partial | blocked
**Executive summary**: 1–3 sentences.

### Step Breakdown
| Step | Folder | Tasks | Focus |
|---|---|---|---|
| 01 | 01-{name} | {N} | {goal} |
| 02 | 02-{name} | {N} | {goal} |
| Total | — | {N} | |

**Depends-on graph**: {01 → none · 02 → 01 · …}
**Plan threat-matrix handoff**: {K applicable rows → each in its owning step as a RED task} | {not applicable} | {gaps flagged}
**TDD applied**: {yes — RED/GREEN/REFACTOR} | {no} | {n/a}

### Change-level Review Workload Forecast
- Estimated changed lines: {estimate or range}
- 400-line budget risk: {Low | Medium | High}
- Chained PRs recommended: {Yes | No}
- Delivery strategy: {ask-on-risk | auto-chain | single-pr | exception-ok}
- Decision needed before apply: {Yes | No}
- Chain strategy: {…}

**Mnemonic**: observation `{id or 'none'}` · session `{sid}`
**Open questions**: {list, or "None"}
**Skill resolution**: paths-injected | fallback-registry | none
**Risks**: {list, or "None"}
**Next**: sdd-spec
```

Close the final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up. Do not call `mem_session_summary` in a sub-agent context — the orchestrator owns session close.

## Rules

- **You own the step tree and NN numbering.** Every step in the intent's Step Blueprint has a `steps/<NN-name>/` folder; no step exists that the intent did not name; NN is 2-digit zero-padded and sequential in intent order.
- **Always reference concrete file paths** in tasks — never "a new middleware" / "the handler".
- Tasks MUST be **ordered by dependency within a step**, and inter-step dependencies MUST be declared as `Depends on:` notes.
- **TDD threat rows**: every applicable plan threat-row becomes a RED-test task before its production task in the owning step (Step 2); if TDD is on, the unit is RED → GREEN → REFACTOR.
- Testing tasks MUST reference a WHAT bullet from the plan's per-step WHAT block — not "add tests" and not a scenario name (scenarios do not exist until `sdd-spec` runs).
- Each task completable in **one session**; if a step's punch-list is too big, flag in `risks` (re-decomposition is a propose-level decision).
- Use hierarchical numbering (`{NN}.{i}`).
- Apply any `rules.tasks` from `docs/skillgrid/config.yaml`.
- **Size budget**: each step's `tasks.md` MUST be **under 400 words**. Task lines: 1–2 lines max. Checklist format, not paragraphs.
- **Review workload guard**: ALWAYS include the four plain-text guard lines in step 01's `tasks.md` (change-level), carried by reference in other steps. If likely above 400 changed lines, recommend chained PRs and honor the received delivery strategy for whether a decision/exception is needed before apply.
- **Hybrid is the only mode** — always write every step's `tasks.md` AND save the concatenated Mnemonic observation.
- No external binaries. Mnemonic (`mem_*`) and the code index (`code_*`) are the only knowledge sources; no `gentle-ai`, no `gentleman-ai`, no CLI validator.
- Return envelope per Step 7 — final action is text, not a tool call.

## Gotchas

- You are **inheriting** threat-row tests from the plan (which will then be restated in scenarios by `sdd-spec`), not re-deriving them. If a run invents a RED test from prose instead of carrying the plan's case, the upstream plan was carrying it, not the spec — flag the gap rather than papering over it.
- `mem_search` returns **300-char previews**. Never use a preview as source material — always `mem_get_observation(id)` for full content of the intent, plan, or prior tasks. A 300-char preview of a 2000-char plan loses most of it.
- **The change-level guard lines live in step 01's `tasks.md`.** A reviewer reading only `02-…/tasks.md` must still see the change-level forecast (by pointer or repeated line). If you bury the guard in `05-…/tasks.md`, `sdd-apply`'s Step 2 check will `blocked` because it cannot find the forecast in the step it was pointed at.
- **One step, one deliverable.** If a step crosses >2 distinct files or >30 task lines, you have not decomposed — flag in `risks` rather than silently splitting (splitting changes the NN tree and invalidates `sdd-propose`'s reservation).
- **Horizontal slicing is a planning smell, not a task.** If the plan assigns "schema" to step 01, "API" to step 02, and "UI" to step 03, that is horizontal slicing. Each step must deliver the behavior its WHAT block promises end-to-end, not just one layer of it. Flag horizontal plans in `risks` and propose a vertical re-decomposition.
- **Wide refactors need expand-contract, not a single slice.** A rename or global migration cannot land green in one tracer bullet. If the plan does not already sequence it as expand then migrate batches then contract, flag it in `risks` — the plan missed the exception.
- **Frontier order matters.** `sdd-apply` works the frontier: any step whose blockers are all done. If step 02 depends on step 01 but step 02's folder was created first and listed first in the tree, that is still correct as long as the `Depends on:` note is present. But if step 03 depends on step 02 which depends on step 01, and step 03's `Depends on:` does not transitively include step 01, that is a gap — `sdd-apply` may start step 03 before step 02 is done.
- **Updating an existing `tasks.md`.** `sdd-apply` marks lines `[x]`. On re-run, READ first and preserve completed state — overwriting resets progress and re-opens done work.
- **Mnemonic save rules**: `title == topic_key`, `scope: "project"`, active `session_id`. No `project:` parameter, no `capture_prompt` field. (See `conventions/mnemonic-memory.md` § Mnemonic Tool Mapping.)
- **Hybrid writes both, or it is half a save.** A `tasks.md` on disk with no Mnemonic save (or vice versa) breaks recovery — the filesystem survives branch switches, Mnemonic survives `/clear`; you need both.
- If the code index is `stale: true`, run `code_index` before `code_search`. A task citing a file you confirmed on a stale index is a task with a hole.
- Do not commit from this phase. Tasks is the plan; `sdd-apply` commits the DO. Commit conventions live in `_shared/conventions/commits.md` and the close-token footer applies to the apply commit, not this phase.

## References

- [`../sdd-design/SKILL.md`](../sdd-design/SKILL.md) — upstream; its Impacted Files map, per-step WHAT blocks, and `## Threat Matrix` applicable rows feed Steps 1–3.
- [`../sdd-propose/SKILL.md`](../sdd-propose/SKILL.md) — upstream; its intent Step Blueprint is your step contract.
- [`../sdd-spec/SKILL.md`](../sdd-spec/SKILL.md) — downstream; it fills each `steps/<NN-name>/` you create with `acceptance.feature`.
- [`references/threat-matrix.md`](references/threat-matrix.md) — the boundary rows the plan may have marked applicable (local copy of `sdd-design`'s matrix).
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape, session protocol, recovery ladder.
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — change-folder layout, NN numbering, `rules.tasks`, and that `sdd-apply` updates this `tasks.md`.
- [`../_shared/conventions/mnemonic-code-indexing.md`](../_shared/conventions/mnemonic-code-indexing.md) — the `code_status → code_index → code_search → code_read` ladder for confirming real file paths.
- [`../_shared/conventions/commits.md`](../_shared/conventions/commits.md) — commit contract (relevant to the downstream `sdd-apply` commit, not this phase).
- Vertical slice and tracer-bullet rules are adapted from the `to-tickets` skill (`engineering/to-tickets`), with the wide-refactor expand-contract exception.
