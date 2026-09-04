---
name: sdd-apply
description: "Implement SDD work from change-level change.md + tasks.md + acceptance.feature by writing real code, marking tasks [x] in docs/skillgrid/changes/<NNN-slug>/tasks.md (## NN-<name> sections), bumping ## State, and persisting apply-progress. Use after sdd-spec and before sdd-verify — enforcing RED/GREEN/REFACTOR (Strict TDD) when required, Run:/Expected: lines, BDD against @step-NN scenarios, the change-level review-workload/PR-split decision, and per-step work-unit evidence. Marks tasks [x] in change-level tasks.md AND persists to Mnemonic. Uses Mnemonic memory + code index; no external binaries."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  author: devopstales
  version: "3.0"
  family: sdd
  part-of: skillgrid
  phase-order: "init → explore → propose → spec → apply → verify → archive"
  prev: [sdd-spec]
  next: [sdd-verify]
  artifact: apply-progress
  delegate_only: true
---

# sdd-apply

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-apply` sub-agent **unless** you loaded this skill directly through the `skill()` tool.

- **Sub-agent (primary)**: you were delegated here by the SDD orchestrator. Continue with the phase work below. Do not re-delegate. Do not call the `skill()` tool again.
- **Orchestrator (skill() loaded this directly)**: STOP. Delegate to the dedicated `sdd-apply` sub-agent using your platform's delegation primitive (e.g. `task(...)`) instead of doing the work inline.

> **For agentic workers:** `sdd-apply` is a **dispatcher**. It does not write code itself. At Step 5 it MUST route the assigned step to exactly one of two execution skills and follow that skill's loop instead of implementing inline:
>
> - **`subagent-execution`** (see [`../subagent-execution/SKILL.md`](../subagent-execution/SKILL.md)) — REQUIRED SUB-SKILL when the workload decision is `auto-chain` / `chained-PR` / `stacked-PR`, or the step has ≥ 4 independent tasks (recommended for these). Fresh implementer subagent per task, task-scoped review after each, bounded fix loop.
> - **`simple-execution`** (see [`../simple-execution/SKILL.md`](../simple-execution/SKILL.md)) — REQUIRED SUB-SKILL when the step is small, tightly coupled, or the delivery shape is `single-pr` under budget. Inline loop, STRICT TDD if resolved active.
>
> Steps live as `## NN-<name>` sections inside change-level `docs/skillgrid/changes/<NNN-slug>/tasks.md` (checkbox `- [ ]` syntax). Whichever route is taken flips them `- [ ]` → `- [x]` **as they complete**, bumps `## State`, and records the Step Evidence rows this phase then persists (Step 7). The dispatch decision itself is recorded in the return envelope's `Workload / PR Boundary` block.

## Purpose

You are the APPLY phase — the only phase that writes production code. You take the assigned step(s) from change-level **`change.md`** (WHY + HOW + per-step WHAT), **`tasks.md`** (`## NN-<name>` punch-lists with TDD micro-cycle + `Run:` / `Expected:` lines), and **`acceptance.feature`** (Features tagged `@step-NN`), and implement them by writing real code. You mark each task `[x]` in **`docs/skillgrid/changes/<NNN-slug>/tasks.md`** as it completes, bump `## State`, and persist a cumulative `apply-progress` artifact that `sdd-verify` and `sdd-archive` rely on.

There is **no** `steps/` directory, no per-step `tasks.md` / `acceptance.feature`, and no `intent.md` / `plan.md`.

Phase order is `init → explore → propose → spec → apply → verify → archive`. Two consequences that drive this phase:

1. **You consume, not invent.** Acceptance criteria are the change-level `acceptance.feature` scenarios tagged `@step-NN`; structural constraints are `change.md` decisions and per-step WHAT. If a task conflicts with both, flag it in the return summary rather than silently freelancing a third approach.
2. **Progress is cumulative across steps.** Every batch appends to the same `apply-progress` artifact and the same change-level `tasks.md`. You must read prior progress before writing, or you lose completed work from earlier batches.

## What You Receive

From the orchestrator:

- **Change id** — `<NNN-slug>` (e.g. `001-oauth-login`)
- **The specific step(s) and task(s) to implement** (e.g. "Step 01, tasks 01.1–01.4") — only these. You never pick a broader batch on your own. If a step's `Depends on:` names an unfinished step, STOP and report (see Status Guard).
- **Strict TDD mode** (`true` | `false`) — resolved by the orchestrator from the project's `testing-capabilities`; if not provided, you resolve it in Step 4.
- **Delivery strategy** and the resolved workload decision (`ask-on-risk` | `auto-chain` | `single-pr` | `exception-ok`, plus the chosen PR slice (`stacked-to-main` / `feature-branch-chain`) or an accepted `size:exception` when applicable) — read from the change-level **Review workload** table in `tasks.md`.
- Optional: **ticket/issue id** — the apply commit close-token per `_shared/conventions/commits.md`.
- Optional: a `## Skills to load before work` block.

**Artifact store mode is `hybrid` — the only mode for this phase.** Every run does BOTH: updates change-level `docs/skillgrid/changes/<NNN-slug>/tasks.md` with `[x]` marks and `## State` **and** persists progress to Mnemonic under `sdd/<NNN-slug>/apply-progress` (upserting the tasks observation for the `[x]` state). The filesystem write and the Mnemonic save are each their own obligations — the Mnemonic save does not stand in for the file. Do not branch on the mode.

## Execution + Persistence Conventions

Follow, on each save, rather than restating here:

- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active `session_id`; **no** `project:` parameter, **no** `capture_prompt` field; `mem_search` returns previews — always `mem_get_observation(id)` for full content; upsert via same `topic_key`).
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — change-folder layout (v3: `change.md` + `tasks.md` + `acceptance.feature`); change-level `tasks.md` is the live artifact you mark `[x]`; `acceptance.feature` is the change-level acceptance contract (`@step-NN`); `rules.apply` from `docs/skillgrid/config.yaml`.
- [`../_shared/templates/template-tasks.md`](../_shared/templates/template-tasks.md) — shape of `## NN-<name>`, TDD micro-cycle, `Run:` / `Expected:`, and `### Verification` stubs (verify fills those later).
- [`../_shared/conventions/commits.md`](../_shared/conventions/commits.md) — the apply commit is a checkpoint: Conventional Commits, ticket close-token footer, no AI trailers, one logical change per commit.
- [`../simple-execution/references/strict-tdd.md`](../simple-execution/references/strict-tdd.md) — the Strict TDD module (RED → GREEN → TRIANGULATE → REFACTOR), loaded ONLY when Step 4 resolves Strict TDD as active. Owned by `simple-execution`; also honored by `subagent-execution` (its dispatch brief must carry this ref when Strict TDD is active).
- [`../sdd-spec/SKILL.md`](../sdd-spec/SKILL.md) — upstream; it wrote change-level `tasks.md` + `acceptance.feature` and the **Review workload** forecast you enforce in Step 2.

## Skill Loading

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first and follow them while writing code.
2. Otherwise recover context from Mnemonic and the change folder (previews are not enough — always fetch full content):
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/change")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; WHY + HOW + per-step WHAT + threat matrix.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/spec")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; change-level `acceptance.feature` (`@step-NN` scenarios are your acceptance criteria).
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/tasks")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; change-level tasks + State (keep this observation id for the `[x]` upsert).
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/apply-progress")` → `..._mem_get_observation(id)` — prior progress (see Step 3).
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/research")` → optional supporting context.
   - `skillgrid-mnemonic_mem_search(query: "sdd-init/{project}")` → `..._mem_get_observation(id)` — detected project facts (stack, testing, tracker).
3. Read the filesystem primary copies in the change folder:
   - `docs/skillgrid/changes/<NNN-slug>/change.md`
   - `docs/skillgrid/changes/<NNN-slug>/tasks.md` (assigned `## NN-<name>` sections)
   - `docs/skillgrid/changes/<NNN-slug>/acceptance.feature` (Features `@step-NN`)
4. Read `docs/skillgrid/config.yaml` if present — `context:`, `rules.apply`, and the `strict_tdd` / `tdd` flag bind this phase.

## Status and Workspace Guard

Before reading implementation files or writing code, confirm readiness from the structured state (from the orchestrator, or `## State` in change-level `tasks.md`):

- If state is **`blocked`**, STOP and return `blocked` with the missing artifacts or unsafe context.
- If the assigned step's `Depends on:` names a step whose `### Verification` Verdict is not yet `PASS` or `PASS WITH WARNINGS`, STOP and return `blocked` naming the un-satisfied dependency — do not implement on top of an unfinished step.
- If state is **`done`** / **`all_done`**, do not edit. Return `success` with `Next: sdd-verify`.
- If state is **`in_progress`** / **`ready`**, proceed only on the assigned pending tasks.
- Read context from the artifact paths / topic keys (`change`, `tasks`, `spec`), not assumed fixed legacy filenames (`intent`, `plan`, `steps/…`).
- **Edit roots are bounded.** If the orchestrator provides allowed edit roots, edit only files under them; if a needed edit is outside, STOP and report the unsafe path. Never edit files outside the change's affected areas.

## What to Do

### Step 1: Read Context

Before writing ANY code:

1. Confirm the state guard (ready + assigned tasks + satisfied step dependencies) from the previous section.
2. Read every applicable artifact (`change.md`, `tasks.md`, `acceptance.feature`) from Mnemonic and the change folder.
3. Read the change-level `acceptance.feature` Features tagged `@step-NN` for the assigned step(s) — understand **WHAT** the code must do. These are your acceptance criteria, per scenario name.
4. Read `change.md` — understand **HOW** to structure the code; Architecture Decisions, Impacted Files map, and per-step WHAT constrain your approach.
5. Read the assigned `## NN-<name>` section in `tasks.md` — the execution punch-list, including TDD micro-cycle sub-tasks and every `Run:` / `Expected:` line. Honor those commands literally.
6. Read the existing code in affected files — understand current patterns.
7. Check the project's coding conventions from `docs/skillgrid/config.yaml` and the loaded skills.

Confirm the exact file paths you will touch against the code index — an apply that edits a file it has not read is an apply with a hole:

```
skillgrid-mnemonic_code_status              # stale? file_count 0?
  → if stale, skillgrid-mnemonic_code_index
skillgrid-mnemonic_code_search(query: "<file-or-symbol>", limit: 20)
  → skillgrid-mnemonic_code_read(path: <hit.path>, start_line: <hit.start_line>, end_line: <hit.end_line>)
```

### Step 2: Enforce the Review Workload Decision

Before implementing, inspect the change-level **Review workload** table in `docs/skillgrid/changes/<NNN-slug>/tasks.md` (section `## Review workload`).

If the forecast says **any** of:

- `400-line budget risk: High`
- `Chained PRs recommended: Yes`
- `Decision needed before apply: Yes` (or delivery strategy still `ask-on-risk` without a resolved choice)

Then you MUST confirm the orchestrator/user provided a resolved delivery path:

1. **`auto-chain` or a chosen chained/stacked PR mode** — implement only the assigned work-unit slice, keep its scope autonomous, and report the intended PR boundary. Follow the forecast's `Chain strategy` / `Delivery strategy` (`stacked-to-main` or `feature-branch-chain`) for branch targeting.
   - `stacked-to-main`: each PR targets `main` after the previous merges.
   - `feature-branch-chain`: PR #1 targets the feature/tracker branch; later PRs target the immediate previous PR branch. Only the tracker merges to `main`; child PR diffs must stay focused on the current work unit and never target `main` directly. If a child diff shows a previous slice, the base is wrong — retarget/rebase until it is clean.
2. **`exception-ok`** — continue only if the prompt explicitly says the maintainer accepts `size:exception`.
3. **`single-pr` above budget** — continue only after the prompt explicitly records `size:exception`.

If **neither** a delivery decision nor a resolved chain strategy is present for an above-budget forecast, STOP before writing code and return `blocked` with: `Workload decision required before apply: estimated work may exceed 400 changed lines. Ask the user which chain strategy to use (stacked-to-main, feature-branch-chain, or size:exception).`

### Step 3: Read Previous Apply-Progress (if exists)

Before starting work, check for existing apply-progress and MERGE against it — never overwrite:

1. `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/apply-progress")`
2. If found: `skillgrid-mnemonic_mem_get_observation(id)` → read the full content.
3. Also read `docs/skillgrid/changes/<NNN-slug>/tasks.md` and note which tasks under the assigned `## NN-<name>` sections are already `[x]`.
4. Parse which tasks are already complete — **skip them**; start from the first incomplete assigned task.
5. When saving your apply-progress in Step 7, MERGE: include ALL previously completed tasks (copy their status and evidence) PLUS your new completions in a single cumulative artifact.

**CRITICAL**: If the orchestrator told you prior progress exists, you MUST read it first. If you upsert without reading, completed work from prior batches is permanently lost (Mnemonic is working memory — the filesystem copy is the recovery copy, but both must stay consistent).

### Step 4: Resolve TDD Mode

Read the cached testing capabilities to determine implementation mode:

```
Read testing capabilities from:
├── Mnemonic: skillgrid-mnemonic_mem_search("sdd/<NNN-slug>/testing-capabilities") or ("sdd/{project}/testing-capabilities") → mem_get_observation(id)
├── docs/skillgrid/config.yaml → rules.apply.tdd + rules.apply.test_command
└── Fallback: detect from project files directly (package.json, go.mod, pyproject.toml, etc.)

Resolve mode:
├── IF tdd: true AND a test runner exists
│   └── STRICT TDD MODE → the chosen route (Step 5) MUST follow
│       [../simple-execution/references/strict-tdd.md](../simple-execution/references/strict-tdd.md)
│       for every task it executes, AND honor each tasks.md TDD micro-cycle
│       (01.N.a–e) with its Run: / Expected: lines
├── IF tdd: false OR no test runner
│   └── STANDARD MODE → the chosen route runs without the strict-tdd module
│       (still honor explicit Run:/Expected: lines written in tasks.md)
└── Cache the resolved mode for the return summary and pass it to the route
```

**Key principle**: if Strict TDD is **not** active, ZERO Strict-TDD module instructions are loaded — do not read, process, or reason from `../simple-execution/references/strict-tdd.md`. You still execute any `Run:` / `Expected:` lines that `sdd-spec` wrote into `tasks.md`.

**Hard gate (Strict TDD only)**: if Strict TDD is active, you MUST produce a **TDD Cycle Evidence** table in the apply-progress artifact — every task row carries RED (test written first) → GREEN (implementation passes) → TRIANGULATE → REFACTOR. A task completed without a test written first is marked FAILED in the table. `sdd-verify` rejects work whose TDD Evidence table is missing or incomplete. There is **no silent fallback**: if you resolved Strict TDD as active, you follow it or you report failure — you do not quietly drop to Standard Mode.

**Hard gate (ALL modes): Per-Step Evidence.** Every assigned step, including a single-PR slice in Standard Mode, MUST produce a **Step Evidence** table before its tasks are marked complete:

| Evidence | Required value |
|---|---|
| Focused test command + exact result | Smallest command proving this step (command, exit/result, relevant counts) — prefer the `Run:` lines from `tasks.md` |
| Acceptance scenario coverage | Each `@step-NN` scenario name + the test that ran it + pass/fail |
| Runtime harness command/scenario + exact result | Real integration/runtime path; explicit `N/A` + reason only if no runtime boundary exists |
| Rollback boundary | Exact files/behavior that can be reverted without removing unrelated work |

If `change.md` carries applicable threat-matrix cases in the assigned step, write and run each mapped RED test **before** the corresponding production change, even in Standard Mode. Preserve Strict TDD's full RED → GREEN → TRIANGULATE → REFACTOR evidence when active; the Step Evidence table supplements it and never replaces it. Do not mark a step's tasks complete if its focused test, an applicable acceptance scenario, or an applicable runtime harness fails.

When all assigned steps finish, **return control to the parent orchestrator.** The executor never launches `sdd-verify`, a review/refutation pass, a correction actor, or a scoped validator on its own — the orchestrator decides the next phase. If only focused remediation of the just-applied step is needed, do it within this apply batch and fold the result into the Step Evidence before returning; do not start a separate verification cycle from inside apply. Leave each step's `### Verification` stub for `sdd-verify` (do not invent a PASS verdict here).

### Step 5: Route the Step to an Execution Skill (DO NOT implement inline)

`sdd-apply` does not write code. It routes each assigned step to exactly one of the two execution skills, then collects the Step Evidence rows that skill records. The chosen skill's loop rules OVERRIDE any inline step-local guidance in this file.

Choose the route from the delivery decision (Step 2) and the step's shape:

```
FOR EACH ASSIGNED STEP (in NN order, honoring Depends-on):
  CHOOSE ROUTE:
  ├── Subagent (REQUIRED SUB-SKILL: use ../subagent-execution/SKILL.md):
  │   any of — workload decision is auto-chain / chained-PR / stacked-PR;
  │            the step has ≥ 4 independent tasks;
  │            or the task set is too large for one inline context.
  │   → follow subagent-execution: fresh implementer per task + task-scoped
  │     review + bounded fix loop. It writes its own .skillgrid/sdd/<NNN-slug>/ ledger.
  │
  └── Inline (REQUIRED SUB-SKILL: use ../simple-execution/SKILL.md):
      small, tightly-coupled step, or single-pr under the 400-line budget.
      → follow simple-execution: per-task loop, STRICT TDD if Step 4 resolved
        active (it owns references/strict-tdd.md). Honor tasks.md Run:/Expected:.

  RECORD (regardless of route):
  ├── The flipped `- [ ]` → `- [x]` marks in docs/skillgrid/changes/<NNN-slug>/tasks.md
  │   under ## NN-<name> → ### Tasks (both routes mark as they go)
  ├── Bumped ## State (phase: apply, current_step, status, updated)
  ├── The Step Evidence rows (focused test, acceptance @step-NN, runtime, rollback)
  └── Any deviations or blocked tasks
```

The route you chose and its rationale go into the return envelope's `Workload / PR Boundary` block (Step 11). A step whose shape fits both routes may use either; record which, so `sdd-verify` and a resumed session know the evidence provenance.

> The inline per-task loop previously written here now lives in **`simple-execution`**; the dispatch loop lives in **`subagent-execution`**. Keep only the routing + evidence-collection here.

Keep each task completable in one sitting; match the project's actual patterns — if the codebase does it differently from what the task implies, follow the existing code (and note the deviation in the summary).

### Step 6: Mark Tasks Complete (change-level tasks.md)

Update `docs/skillgrid/changes/<NNN-slug>/tasks.md` — under the assigned `## NN-<name>` → `### Tasks`, change `- [ ]` to `- [x]` for each completed task (and micro-cycle sub-task) **as you go**, not in one batch at the end. Bump `## State` (`phase: apply`, `current_step`, `status`, `updated`).

```markdown
## 01-db-migration

### Tasks

- [x] 01.1 `[RED]` Create failing migration test
  - [x] 01.1.a Write failing test
  - [x] 01.1.b Run to confirm fail — `Run: go test ./db -run TestOAuthToken` — Expected: FAIL
  - [x] 01.1.c Minimal implementation
  - [x] 01.1.d Run to confirm pass — `Run: go test ./db -run TestOAuthToken` — Expected: PASS
  - [x] 01.1.e Commit — `feat(db): oauth token migration`
- [ ] 01.2 `[AFK]` Wire auth routes …                 ← still pending

### Verification

Verdict: `PENDING`   ← leave for sdd-verify
```

If you commit this slice (see § Commits), commit after the tasks are green and marked — never before.

### Step 7: Persist Progress (MANDATORY — do not skip)

Follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md). Hybrid = BOTH writes:

1. **Filesystem** — change-level `tasks.md` is already updated in Step 6. That file is the recovery source of truth for all steps.
2. **Mnemonic** — start one session, then save the cumulative apply-progress AND upsert the tasks observation:

```
sid = skillgrid-mnemonic_mem_session_start(title: "sdd/<NNN-slug>/apply")

# Cumulative progress (MERGED with prior batches — see Step 3)
skillgrid-mnemonic_mem_save(
  title:      "sdd/<NNN-slug>/apply-progress",
  topic_key:  "sdd/<NNN-slug>/apply-progress",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "{cumulative apply-progress markdown: per-step completed tasks, evidence, deviations, Step Evidence tables}"
)

# [x] state of the tasks artifact (upsert — same topic_key replaces the observation)
skillgrid-mnemonic_mem_save(
  title:      "sdd/<NNN-slug>/tasks",
  topic_key:  "sdd/<NNN-slug>/tasks",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "{full change-level tasks.md with updated [x] marks and ## State}"
)
```

Mnemonic save notes: `title == topic_key` exactly; `scope: "project"`; pass the active `session_id`; there is **no** `project:` parameter and **no** `capture_prompt` field in the Mnemonic schema — omit both. `topic_key` upserts — saving `sdd/<NNN-slug>/tasks` again replaces it in place (that is how the `[x]` marks propagate to memory).

The file must actually exist on disk at `docs/skillgrid/changes/<NNN-slug>/tasks.md` — a Mnemonic save without the file is incomplete.

### Step 8: Merge Protocol

When saving apply-progress:

1. If you read prior progress in Step 3, your artifact MUST include **ALL** previously completed tasks (copy their status and evidence) PLUS your new completions.
2. The saved artifact reflects the **cumulative** state of ALL tasks across ALL steps and ALL batches — a downstream reader of `sdd/<NNN-slug>/apply-progress` sees the whole change, not just this slice.
3. Keep the same structure batch over batch, so a reviewer can diff two saves to see exactly what this batch added.

### Step 9: Commit Boundary (per `_shared/conventions/commits.md`)

For each completed step slice (or single-PR batch), make it a clean, restorable checkpoint:

- Commit only when the step's assigned tasks are green and marked `[x]`.
- One logical change per commit (implementation + its tests + enabling config). Only intended files staged — `git status` first.
- Conventional Commits subject, imperative present tense, ≤ 72 chars. Include the ticket close-token footer **only if** a ticket id exists for this work (per the tracker table in `commits.md`). No `Co-authored-by` / AI trailers.
- If applying a `feature-branch-chain` slice, the commit lands on that slice's branch; the orchestrator handles the merge order.
- **Do not commit** work whose tests are red, or whose Step Evidence is incomplete.
- If the orchestrator has not asked for a commit and the repo has no checkpoint policy, leave the working tree staged-but-uncommitted and note it in the return summary.

### Step 10: Self-Check (no external validator binary)

Before returning, confirm each — fix any failure before returning `success`, else return `partial` with the failed item in `risks`:

1. Every completed assigned task is marked `[x]` in change-level `tasks.md` under the correct `## NN-<name>` (re-read the file to confirm — internal todos are not evidence).
2. `## State` was bumped for this batch.
3. Every assigned step has a matching Step Evidence row (focused test command + result, `@step-NN` acceptance scenario coverage, runtime harness + result, rollback boundary).
4. If Strict TDD is active, the TDD Cycle Evidence table has a RED/GREEN/TRIANGULATE/REFACTOR column per completed task, and no row silently lacks a test-first step.
5. Any applicable `change.md` threat-row in the assigned step has its mapped RED test written and run **before** the production change (present in the evidence), and a matching `@step-NN` scenario passed.
6. Every `Run:` / `Expected:` line for completed micro-cycle steps was actually executed with matching result.
7. The apply-progress upsert and the tasks upsert both succeeded in Mnemonic, and both match the filesystem `tasks.md`.
8. The review-workload decision was satisfied for the assigned steps (Step 2) — or `blocked` was returned.
9. No un-satisfied step dependency was crossed (Status Guard).
10. No files were written outside the allowed edit roots.
11. You did **not** invent a `### Verification` PASS — that belongs to `sdd-verify`.

### Step 11: Return Envelope

**Your FINAL output MUST be text — not a tool call.** Do the `mem_save` calls (Step 7) *before* this text. A trailing tool call buries the analysis in the tool result; returning text is what the orchestrator reads back.

```markdown
## Implementation Progress

**Change**: {NNN-slug}
**Mode**: {Strict TDD | Standard}
**Location**: `docs/skillgrid/changes/<NNN-slug>/tasks.md` (`## NN-<name>` marked [x]; `## State` bumped) · Mnemonic `sdd/<NNN-slug>/apply-progress` + `sdd/<NNN-slug>/tasks`
**Status**: success | partial | blocked

### Completed Tasks (this batch)
- [x] 01.1 {task description}
- [x] 01.2 {task description}

### Files Changed
| File | Action | Step |
|---|---|---|
| `path/to/file.ext` | Created | 01 |
| `path/to/other.ext` | Modified | 02 |

{IF Strict TDD Mode → include the TDD Cycle Evidence table from ../simple-execution/references/strict-tdd.md}

### Step Evidence
| Step | Focused test (cmd + result) | Acceptance @step-NN (name → test → result) | Runtime harness (cmd + result) | Rollback boundary |
|---|---|---|---|---|
| 01-{name} | {cmd} → {result} | {scenario → {cmd} → pass} | {cmd/scenario} → {result or N/A+reason} | {files/behavior} |

### Deviations from change.md
{List places where the implementation deviated from change.md and why. If none: "None — implementation matches change.md."}

### Issues Found
{List problems discovered during implementation. If none: "None."}

### Remaining Assigned Tasks
- [ ] 01.3 {next assigned task}  ← or "None — all assigned tasks complete"

### Workload / PR Boundary
- Mode: {single PR | chained PR slice | stacked PR slice | size:exception}
- Current work unit: {step name or "N/A"}
- Boundary: {what this apply batch starts from and ends with}
- Estimated review budget impact: {brief note}

### Commit Checkpoint
- Committed: {yes — {type}: {subject} ({sha7} | ticket footer) | no — left staged, reason}

### Commits (this batch, chronological)
- `{type}: {subject}` — {SHA7 or "not committed"} · ticket: {Refs/Closes … or none}
- (use `-` if you did not commit; do NOT invent a SHA for work that only lives in the working tree)

**Mnemonic**: observation(s) `{ids or 'none'}` · session `{sid}`
**Open questions**: {list, or "None"}
**Skill resolution**: paths-injected | fallback-registry | none
**Risks**: {list, or "None"}
**Next**: sdd-verify
```

Close the final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up. Do not call `mem_session_summary` in a sub-agent context — the orchestrator owns session close.

## Rules

- ALWAYS read change-level `acceptance.feature` `@step-NN` scenarios before implementing — they are your acceptance criteria (match by scenario name).
- ALWAYS follow `change.md` decisions — do not freelance a different approach. Match the project's ACTUAL patterns where they differ from the task's implication (and note any deviation).
- Honor every `Run:` / `Expected:` line in the assigned `### Tasks` micro-cycle.
- When `change.md` `## Architecture Decisions` used the `codebase-design` vocabulary, the test you write MUST cross the same seam as the production caller (do not test past the interface). If you cannot test through the interface, the module is probably the wrong shape — re-shape before adding more tests (see `codebase-design` skill, § Designing for testability).
- **High-risk change → request code review.** After the assigned step lands green, BEFORE returning to the orchestrator, fire `requesting-code-review` when ANY of: workload forecast says `400-line budget risk: High` / `Chained PRs recommended: Yes`; `change.md` `## Threat Matrix` has any `Applicable` row; this is the last step before `sdd-archive`; the change touched any `_shared/conventions/*` file or a Mnemonic tool contract. Dispatch a fresh `general` sub-agent with the `references/code-reviewer.md` template; act on the verdict (Critical/Important/Minor) before proceeding.
- **Parallel fan-out → dispatching-parallel-agents discipline.** When the assigned step has 2+ independent work items with no shared state, follow `dispatching-parallel-agents` to fan out: pre-allocate files per sub-agent, give each a unique Mnemonic `topic_key` (e.g. `sdd/<NNN-slug>/parallel/<domain>`), issue all dispatches in one response for true parallelism, then review and integrate.
- ALWAYS consume or produce structured state (`## State` / Status Guard) before implementation — do not infer readiness from conversation alone.
- STOP on `blocked` state and do not edit; STOP on an unsafe `actionContext` or an edit outside the allowed roots; STOP on an un-satisfied step dependency.
- Mark tasks `[x]` in change-level `tasks.md` **as you go**, not in one batch at the end; bump `## State`.
- Before returning, re-read the persisted `tasks.md` and confirm completed tasks are visibly `[x]` — internal todos are not completion evidence.
- NEVER implement tasks that were not assigned to you.
- If a discovery means `change.md` or a `@step-NN` Feature is wrong, **note it** in the return summary — do not silently deviate.
- If a task is blocked by something unexpected, STOP and report back — do not improvise around it.
- When applying a chained/stacked PR slice, keep the batch autonomous: one deliverable scope, verification included, clear rollback boundary.
- When applying `size:exception`, state it explicitly in apply-progress and the return summary.
- Apply any `rules.apply` from `docs/skillgrid/config.yaml`.
- If Strict TDD is resolved active, pass the module `../simple-execution/references/strict-tdd.md` to the chosen route (Step 5) and have it follow the cycle for every task it executes; its rules OVERRIDE any route's default per-task WRITE step.
- **Hybrid is the only mode** — always mark filesystem `tasks.md` AND persist to Mnemonic.
- No `steps/` tree, no `intent.md` / `plan.md`, no per-step `verification.md` writes.
- No external binaries. Mnemonic (`mem_*`) and the code index (`code_*`) are the only knowledge sources; no `gentle-ai`, no `gentleman-ai`, no CLI status/validator binary.
- Return envelope per Step 11 — final action is text, not a tool call.

## Gotchas

- **Merging prior progress is load-bearing.** Mnemonic is working memory — an upsert of `apply-progress` without first reading the prior observation silently drops every earlier batch's completed state. `sdd-verify` and `sdd-archive` will then look at a partial picture.
- `mem_search` returns **300-char previews**. A preview of a 2000-char change/spec loses most of it — always `mem_get_observation(id)` before you rely on it as an acceptance criterion or a constraint.
- **The tasks observation upsert is separate from apply-progress.** Step 7 writes TWO Mnemonic saves — `sdd/<NNN-slug>/tasks` (the `[x]` + State) and `sdd/<NNN-slug>/apply-progress` (the cumulative evidence). Missing either leaves the other stale. Change-level `tasks.md` is the recovery copy for both; keep them consistent.
- **Step dependencies are real.** A step's `Depends on:` naming an unfinished step (no PASS / PASS WITH WARNINGS in that step's `### Verification`) means `blocked`, not "I'll just do both". Crossing an unverified dependency silently breaks `sdd-verify`'s per-step evidence chain.
- In Strict TDD, a GREEN that "passes trivially" (loop runs 0 times, setup doesn't reach the code path, component never renders) is not a GREEN. The `../simple-execution/references/strict-tdd.md` TRIANGULATE step is the gate that forces real logic — do not skip it because your first GREEN was green.
- The TDD Cycle Evidence table and the Step Evidence table are **different artifacts with different roles**: TDD table is per-task RED/GREEN/TRIANGULATE/REFACTOR; Step Evidence is per-step focused-test / `@step-NN` coverage / runtime / rollback. `sdd-spec` forecasts the change-level guard; `sdd-verify` checks both. Missing either = a partial apply.
- **Workload guard ordering.** The Step 2 check comes BEFORE any code is written — implementing an above-budget slice only to discover the delivery-strategy decision wasn't resolved wastes a batch of work and complicates rollback.
- **Mnemonic save rules**: `title == topic_key`, `scope: "project"`, active `session_id`. No `project:` parameter, no `capture_prompt` field. (See `conventions/mnemonic-memory.md` § Mnemonic Tool Mapping.)
- If you resolved Strict TDD as active and then hit a test-runner infrastructure failure mid-cycle, mark the row FAILED and return `partial` with the reason — do NOT fall back to Standard Mode for the same task (that hides a real test-infra defect and breaks the no-silent-fallback rule).
- **Commits are checkpoints, not decoration.** If the repo has a commit policy and the step is green, commit before returning — the SHA + the `[x]` marks are the recovery record. An uncommitted slice that later gets compacted is an expensive re-run.
- **Do not write `### Verification` Verdicts.** Apply leaves Verdict `PENDING`; inventing PASS here short-circuits the independent gate.

## References

- [`../simple-execution/SKILL.md`](../simple-execution/SKILL.md) — **REQUIRED SUB-SKILL (Step 5 inline route).** Owns the inline per-task loop and `references/strict-tdd.md`; that strict-TDD module is read by this route ONLY when Step 4 resolves Strict TDD as active.
- [`../subagent-execution/SKILL.md`](../subagent-execution/SKILL.md) — **REQUIRED SUB-SKILL (Step 5 dispatch route).** Fresh implementer subagent per task + task-scoped review + bounded fix loop. Uses its own per-plan workspace at `.skillgrid/sdd/<NNN-slug>/` and honors the same strict-tdd module for its implementer briefs.
- [../simple-execution/references/strict-tdd.md](../simple-execution/references/strict-tdd.md) — RED → GREEN → TRIANGULATE → REFACTOR cycle, test-layer selection, assertion-quality rules, approval-testing flow, TDD Cycle Evidence table. Passed to whichever route is chosen when Strict TDD is active.
- [`../sdd-spec/SKILL.md`](../sdd-spec/SKILL.md) — upstream; it wrote change-level `tasks.md`, `acceptance.feature`, and the **Review workload** forecast you enforce in Step 2.
- [`../tdd/SKILL.md`](../tdd/SKILL.md) — the RED-first discipline `../simple-execution/references/strict-tdd.md` enforces; the canonical description of the cycle when Strict TDD is not active.
- [`../verification/SKILL.md`](../verification/SKILL.md) — before marking a step's tasks `[x]`, the evidence gate: fresh test run + output in the current message, not "from earlier."
- [`../review-reception/SKILL.md`](../review-reception/SKILL.md) — the receiving-side discipline when a fix round returns findings: verify-first, one item at a time, test each.
- [`../sdd-propose/SKILL.md`](../sdd-propose/SKILL.md) — upstream; its `change.md` decisions and per-step WHAT constrain your approach.
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active session), recovery ladder.
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — change-folder layout (v3); change-level `tasks.md` is the live artifact you mark `[x]`; `rules.apply`.
- [`../_shared/templates/template-tasks.md`](../_shared/templates/template-tasks.md) — canonical `## NN-<name>` / TDD micro-cycle / Verification stub shape.
- [`../_shared/conventions/mnemonic-code-indexing.md`](../_shared/conventions/mnemonic-code-indexing.md) — the `code_status → code_index → code_search → code_read` ladder for confirming real paths before editing.
- [`../_shared/conventions/commits.md`](../_shared/conventions/commits.md) — the apply commit contract: Conventional Commits, ticket close-token footer, no AI trailers, checkpoint boundaries.
