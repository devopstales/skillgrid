---
name: sdd-apply
description: "Implement SDD tasks from the step tree (steps/<NN-name>/) by writing real code, marking per-step tasks complete, and persisting apply-progress. Use to execute one or more assigned step tasks after sdd-spec and before sdd-verify — enforcing RED/GREEN/REFACTOR (Strict TDD) when the project requires it, the change-level review-workload/PR-split decision, and per-step work-unit evidence. Marks tasks [x] in each step's tasks.md AND persists to Mnemonic. Uses Mnemonic memory + code index; no external binaries."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  author: devopstales
  version: "2.0"
  family: sdd
  part-of: skillgrid
  phase-order: "init → explore → propose → design → tasks → spec → apply → verify → archive"
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

## Purpose

You are the APPLY phase — the only phase that writes production code. You take the assigned step(s) from the step tree — each `docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/` with its `tasks.md`, `acceptance.feature`, and the plan's per-step WHAT — and implement them by writing real code, following the acceptance (WHAT) and the plan (HOW) strictly. You mark each task `[x]` in **that step's** `tasks.md` as it is completed and persist a cumulative `apply-progress` artifact that `sdd-verify` (per step) and `sdd-archive` rely on.

Phase order is `… → tasks → spec → apply → verify → archive`. Two consequences that drive this phase:

1. **You consume, not invent.** Your acceptance criteria are the steps' `acceptance.feature` scenarios; your structural constraints are the plan's decisions. If a task conflicts with both, flag it in the return summary rather than silently freelancing a third approach.
2. **Progress is cumulative across steps.** Every batch appends to the same `apply-progress` artifact and the appropriate step's `tasks.md`. You must read prior progress before writing, or you lose completed work from earlier batches.

## What You Receive

From the orchestrator:

- **Change id** — `<NNN-slug>` (e.g. `001-oauth-login`)
- **The specific step(s) and task(s) to implement** (e.g. "Step 01, tasks 01.1–01.4") — only these. You never pick a broader batch on your own. If a step's `Depends on:` names an unfinished step, STOP and report (see Status Guard).
- **Strict TDD mode** (`true` | `false`) — resolved by the orchestrator from the project's `testing-capabilities`; if not provided, you resolve it in Step 4.
- **Delivery strategy** and the resolved workload decision (`ask-on-risk` | `auto-chain` | `single-pr` | `exception-ok`, plus the chosen PR slice (`stacked-to-main` / `feature-branch-chain`) or an accepted `size:exception` when applicable) — read from the change-level forecast in step 01's `tasks.md` (carried by reference in each step).
- Optional: **ticket/issue id** — the apply commit close-token per `_shared/conventions/commits.md`.
- Optional: a `## Skills to load before work` block.

**Artifact store mode is `hybrid` — the only mode for this phase.** Every run does BOTH: updates each assigned step's `steps/<NN-name>/tasks.md` with `[x]` marks **and** persists progress to Mnemonic under `sdd/<NNN-slug>/apply-progress` (upserting the tasks observation for the `[x]` state). A mode token of `openspec` / `engram-compat` / `none` from the orchestrator is honored as `hybrid` here. Do not branch on the mode.

## Execution + Persistence Conventions

Follow, on each save, rather than restating here:

- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active `session_id`; **no** `project:` parameter, **no** `capture_prompt` field; `mem_search` returns previews — always `mem_get_observation(id)` for full content; upsert via same `topic_key`).
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — change-folder layout; each step's `steps/<NN-name>/tasks.md` is the live artifact you mark `[x]`; `acceptance.feature` is the step's acceptance contract; `rules.apply` from `docs/skillgrid/config.yaml`; the `state.yaml` DAG state.
- [`../_shared/conventions/commits.md`](../_shared/conventions/commits.md) — the apply commit is a checkpoint: Conventional Commits, ticket close-token footer, no AI trailers, one logical change per commit.
- [`references/strict-tdd.md`](references/strict-tdd.md) — the Strict TDD module (RED → GREEN → TRIANGULATE → REFACTOR), loaded ONLY when Step 4 resolves Strict TDD as active.
- [`../sdd-tasks/SKILL.md`](../sdd-tasks/SKILL.md) — upstream; it created the step tree and the change-level `Review Workload Forecast` you enforce in Step 2.

## Skill Loading

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first and follow them while writing code.
2. Otherwise recover context from Mnemonic and the change folder (previews are not enough — always fetch full content):
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/intent")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/plan")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; your structural constraints and per-step WHAT.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/spec")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; the per-step `acceptance.feature` scenarios are your acceptance criteria.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/tasks")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; the assigned steps and tasks (keep this observation id for the `[x]` upsert).
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/apply-progress")` → `..._mem_get_observation(id)` — prior progress (see Step 3).
   - `skillgrid-mnemonic_mem_search(query: "sdd-init/{project}")` → `..._mem_get_observation(id)` — detected project facts (stack, testing, tracker).
3. Read the filesystem primary copies in the change folder: `docs/skillgrid/changes/<NNN-slug>/plan.md`, the assigned `steps/<NN-name>/tasks.md`, and `steps/<NN-name>/acceptance.feature`.
4. Read `docs/skillgrid/config.yaml` if present — `context:`, `rules.apply`, and the `strict_tdd` flag bind this phase.

## Status and Workspace Guard

Before reading implementation files or writing code, confirm readiness from the structured state (from the orchestrator, or the `state.yaml` DAG state in the change folder):

- If state is **`blocked`**, STOP and return `blocked` with the missing artifacts or unsafe context.
- If the assigned step's `Depends on:` names a step whose own `verification.md` does not exist yet (or its verdict is not PASS), STOP and return `blocked` naming the un-satisfied dependency — do not implement on top of an unfinished step.
- If state is **`all_done`**, do not edit. Return `success` with `Next: sdd-verify`.
- If state is **`ready`**, proceed only on the assigned pending tasks.
- Read context from the artifact paths / topic keys, not assumed fixed filenames. In this SDD pipeline they map to intent, plan, per-step acceptance, and per-step tasks.
- **Edit roots are bounded.** If the orchestrator provides allowed edit roots, edit only files under them; if a needed edit is outside, STOP and report the unsafe path. Never edit files outside the change's affected areas.

## What to Do

### Step 1: Read Context

Before writing ANY code:

1. Confirm the state guard (ready + assigned tasks + satisfied step dependencies) from the previous section.
2. Read every applicable artifact (intent, plan, per-step acceptance, per-step tasks) from Mnemonic and the change folder.
3. Read the assigned step(s) `acceptance.feature` — understand **WHAT** the code must do. These are your acceptance criteria, per scenario name.
4. Read the plan — understand **HOW** to structure the code; the per-step WHAT block and Impacted Files map constrain your approach.
5. Read the assigned step's `tasks.md` — the execution punch-list, including which RED tests must land before their production code.
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

Before implementing, inspect the change-level **Review Workload Forecast** — it lives in step 01's `tasks.md` (carried by reference in the assigned step). Read it from the assigned step's `tasks.md` (or `01-…/tasks.md`).

If the forecast says **any** of:

- `400-line budget risk: High`
- `Chained PRs recommended: Yes`
- `Decision needed before apply: Yes`

Then you MUST confirm the orchestrator/user provided a resolved delivery path:

1. **`auto-chain` or a chosen chained/stacked PR mode** — implement only the assigned work-unit slice, keep its scope autonomous, and report the intended PR boundary. Follow the forecast's `Chain strategy` (`stacked-to-main` or `feature-branch-chain`) for branch targeting.
   - `stacked-to-main`: each PR targets `main` after the previous merges.
   - `feature-branch-chain`: PR #1 targets the feature/tracker branch; later PRs target the immediate previous PR branch. Only the tracker merges to `main`; child PR diffs must stay focused on the current work unit and never target `main` directly. If a child diff shows a previous slice, the base is wrong — retarget/rebase until it is clean.
2. **`exception-ok`** — continue only if the prompt explicitly says the maintainer accepts `size:exception`.
3. **`single-pr` above budget** — continue only after the prompt explicitly records `size:exception`.

If **neither** a delivery decision nor a resolved chain strategy is present for an above-budget forecast, STOP before writing code and return `blocked` with: `Workload decision required before apply: estimated work may exceed 400 changed lines. Ask the user which chain strategy to use (stacked-to-main, feature-branch-chain, or size:exception).`

### Step 3: Read Previous Apply-Progress (if exists)

Before starting work, check for existing apply-progress and MERGE against it — never overwrite:

1. `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/apply-progress")`
2. If found: `skillgrid-mnemonic_mem_get_observation(id)` → read the full content.
3. Also read each assigned step's `steps/<NN-name>/tasks.md` and note which tasks are already `[x]`.
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
│   └── STRICT TDD MODE → load and follow [references/strict-tdd.md](references/strict-tdd.md) INSTEAD of Step 5
├── IF tdd: false OR no test runner
│   └── STANDARD MODE → use Step 5 (the strict-tdd.md module is never read, never processed)
└── Cache the resolved mode for the return summary
```

**Key principle**: if Strict TDD is **not** active, ZERO TDD instructions are loaded — do not read, process, or reason from `references/strict-tdd.md`.

**Hard gate (Strict TDD only)**: if Strict TDD is active, you MUST produce a **TDD Cycle Evidence** table in the apply-progress artifact — every task row carries RED (test written first) → GREEN (implementation passes) → TRIANGULATE → REFACTOR. A task completed without a test written first is marked FAILED in the table. `sdd-verify` rejects work whose TDD Evidence table is missing or incomplete. There is **no silent fallback**: if you resolved Strict TDD as active, you follow it or you report failure — you do not quietly drop to Standard Mode.

**Hard gate (ALL modes): Per-Step Evidence.** Every assigned step, including a single-PR slice in Standard Mode, MUST produce a **Step Evidence** table before its tasks are marked complete:

| Evidence | Required value |
|---|---|
| Focused test command + exact result | Smallest command proving this step (command, exit/result, relevant counts) |
| Acceptance scenario coverage | Each `acceptance.feature` scenario name + the test that ran it + pass/fail |
| Runtime harness command/scenario + exact result | Real integration/runtime path; explicit `N/A` + reason only if no runtime boundary exists |
| Rollback boundary | Exact files/behavior that can be reverted without removing unrelated work |

If the plan carries applicable threat-matrix cases in the assigned step, write and run each mapped RED test **before** the corresponding production change, even in Standard Mode. Preserve Strict TDD's full RED → GREEN → TRIANGULATE → REFACTOR evidence when active; the Step Evidence table supplements it and never replaces it. Do not mark a step complete if its focused test, an applicable acceptance scenario, or an applicable runtime harness fails.

When all assigned steps finish, **return control to the parent orchestrator.** The executor never launches `sdd-verify`, a review/refutation pass, a correction actor, or a scoped validator on its own — the orchestrator decides the next phase. If only focused remediation of the just-applied step is needed, do it within this apply batch and fold the result into the Step Evidence before returning; do not start a separate verification cycle from inside apply.

### Step 5: Implement Tasks (Standard Workflow, per step)

Used when Strict TDD is **not** active (or between Strict-TDD tasks that are purely structural). For each assigned step, then each assigned task:

```
FOR EACH ASSIGNED STEP (in NN order, honoring Depends-on):
  FOR EACH ASSIGNED TASK in steps/<NN-name>/tasks.md:
  ├── Read the task description
  ├── Read the relevant acceptance.feature scenario (this IS the acceptance criterion — match by scenario name)
  ├── Read the relevant plan WHAT / decisions (these CONSTRAIN your approach)
  ├── Read existing code patterns in the affected files (match project style)
  ├── Write the code
  ├── Run the smallest test/command that proves this task (record it for the Step Evidence)
  ├── Mark the task complete — change `- [ ]` to `- [x]` in steps/<NN-name>/tasks.md IMMEDIATELY
  └── Note any issues or deviations
```

Keep each task completable in one sitting; match the project's actual patterns — if the codebase does it differently from what the task implies, follow the existing code (and note the deviation in the summary).

### Step 6: Mark Tasks Complete (per step)

Update each assigned step's `steps/<NN-name>/tasks.md` — change `- [ ]` to `- [x]` for each completed task **as you go**, not in one batch at the end:

```markdown
# Tasks: 001-oauth-login — Step 01-db-migration

> Goal: {…}
> Depends on: none

## Execution

- [x] 01.1 Create `db/migrations/001_oauth_token.go`          ← done
- [x] 01.2 Add up/down to `db/migrations/001_oauth_token.go`   ← done
- [ ] 01.3 Wire auth routes to `internal/server/server.go`      ← still pending
```

If you commit this slice (see § Commits), commit after the tasks are green and marked — never before.

### Step 7: Persist Progress (MANDATORY — do not skip)

Follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md). Hybrid = BOTH writes:

1. **Filesystem** — each assigned step's `tasks.md` is already updated in Step 6. The step's `tasks.md` is the recovery source of truth for that step.
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
  content:    "{full step-tree task lists with updated [x] marks, concatenated by ## Step {NN}-{name}}"
)
```

Mnemonic save notes: `title == topic_key` exactly; `scope: "project"`; pass the active `session_id`; there is **no** `project:` parameter and **no** `capture_prompt` field in the Mnemonic schema — omit both. `topic_key` upserts — saving `sdd/<NNN-slug>/tasks` again replaces it in place (that is how the `[x]` marks propagate to memory).

### Step 8: Merge Protocol

When saving apply-progress:

1. If you read prior progress in Step 3, your artifact MUST include **ALL** previously completed tasks (copy their status and evidence) PLUS your new completions.
2. The saved artifact reflects the **cumulative** state of ALL tasks across ALL steps and ALL batches — a downstream reader of `sdd/<NNN-slug>/apply-progress` sees the whole change, not just this slice.
3. Keep the same structure batch over batch, so a reviewer can diff two saves to see exactly what this batch added.

### Step 9: Commit Boundary (per `_shared/conventions/commits.md`)

For each completed step slice (or single-PR batch), make it a clean, restorable checkpoint:

- Commit only when the step is green and its tasks are marked `[x]`.
- One logical change per commit (implementation + its tests + enabling config). Only intended files staged — `git status` first.
- Conventional Commits subject, imperative present tense, ≤ 72 chars. Include the ticket close-token footer **only if** a ticket id exists for this work (per the tracker table in `commits.md`). No `Co-authored-by` / AI trailers.
- If applying a `feature-branch-chain` slice, the commit lands on that slice's branch; the orchestrator handles the merge order.
- **Do not commit** work whose tests are red, or whose Step Evidence is incomplete.
- If the orchestrator has not asked for a commit and the repo has no checkpoint policy, leave the working tree staged-but-uncommitted and note it in the return summary.

### Step 10: Self-Check (no external validator binary)

Before returning, confirm each — fix any failure before returning `success`, else return `partial` with the failed item in `risks`:

1. Every completed assigned task is marked `[x]` in its step's `tasks.md` (re-read the file to confirm — internal todos are not evidence).
2. Every assigned step has a matching Step Evidence row (focused test command + result, acceptance scenario coverage, runtime harness + result, rollback boundary).
3. If Strict TDD is active, the TDD Cycle Evidence table has a RED/GREEN/TRIANGULATE/REFACTOR column per completed task, and no row silently lacks a test-first step.
4. Any applicable plan threat-row in the assigned step has its mapped RED test written and run **before** the production change (present in the evidence), and a matching `acceptance.feature` scenario passed.
5. The apply-progress upsert and the tasks upsert both succeeded in Mnemonic, and both match the filesystem `tasks.md`.
6. The review-workload decision was satisfied for the assigned steps (Step 2) — or `blocked` was returned.
7. No un-satisfied step dependency was crossed (Status Guard).
8. No files were written outside the allowed edit roots.

### Step 11: Return Envelope

**Your FINAL output MUST be text — not a tool call.** Do the `mem_save` calls (Step 7) *before* this text. A trailing tool call buries the analysis in the tool result; returning text is what the orchestrator reads back.

```markdown
## Implementation Progress

**Change**: {NNN-slug}
**Mode**: {Strict TDD | Standard}
**Location**: `docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/tasks.md` (marked [x]) · Mnemonic `sdd/<NNN-slug>/apply-progress` + `sdd/<NNN-slug>/tasks`
**Status**: success | partial | blocked

### Completed Tasks (this batch)
- [x] 01.1 {task description}
- [x] 01.2 {task description}

### Files Changed
| File | Action | Step |
|---|---|---|
| `path/to/file.ext` | Created | 01 |
| `path/to/other.ext` | Modified | 02 |

{IF Strict TDD Mode → include the TDD Cycle Evidence table from references/strict-tdd.md}

### Step Evidence
| Step | Focused test (cmd + result) | Acceptance scenarios (name → test → result) | Runtime harness (cmd + result) | Rollback boundary |
|---|---|---|---|---|
| 01-{name} | {cmd} → {result} | {scenario → {cmd} → pass} | {cmd/scenario} → {result or N/A+reason} | {files/behavior} |

### Deviations from Plan
{List places where the implementation deviated from plan.md and why. If none: "None — implementation matches plan."}

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

- ALWAYS read the step's `acceptance.feature` before implementing — the scenarios are your acceptance criteria (match by scenario name).
- ALWAYS follow the plan's decisions — do not freelance a different approach. Match the project's ACTUAL patterns where they differ from the task's implication (and note any deviation).
- ALWAYS consume or produce structured state (Step guard) before implementation — do not infer readiness from conversation alone.
- STOP on `blocked` state and do not edit; STOP on an unsafe `actionContext` or an edit outside the allowed roots; STOP on an un-satisfied step dependency.
- Mark tasks `[x]` in each step's `tasks.md` **as you go**, not in one batch at the end.
- Before returning, re-read the persisted `tasks.md` and confirm completed tasks are visibly `[x]` — internal todos are not completion evidence.
- NEVER implement tasks that were not assigned to you.
- If a discovery means the plan or a step's acceptance is wrong, **note it** in the return summary — do not silently deviate.
- If a task is blocked by something unexpected, STOP and report back — do not improvise around it.
- When applying a chained/stacked PR slice, keep the batch autonomous: one deliverable scope, verification included, clear rollback boundary.
- When applying `size:exception`, state it explicitly in apply-progress and the return summary.
- Apply any `rules.apply` from `docs/skillgrid/config.yaml`.
- If Strict TDD is resolved active, load `references/strict-tdd.md` and follow its cycle INSTEAD of Step 5; its rules OVERRIDE Step 5 entirely.
- **Hybrid is the only mode** — always mark each step's filesystem `tasks.md` AND persist to Mnemonic; never branch on `openspec` / `engram-compat` / `none`.
- No external binaries. Mnemonic (`mem_*`) and the code index (`code_*`) are the only knowledge sources; no `gentle-ai`, no `gentleman-ai`, no CLI status/validator binary.
- Return envelope per Step 11 — final action is text, not a tool call.

## Gotchas

- **Merging prior progress is load-bearing.** Mnemonic is working memory — an upsert of `apply-progress` without first reading the prior observation silently drops every earlier batch's completed state. `sdd-verify` and `sdd-archive` will then look at a partial picture.
- `mem_search` returns **300-char previews**. A preview of a 2000-char plan/spec loses most of it — always `mem_get_observation(id)` before you rely on it as an acceptance criterion or a constraint.
- **The tasks observation upsert is separate from apply-progress.** Step 7 writes TWO Mnemonic saves — `sdd/<NNN-slug>/tasks` (the `[x]` state) and `sdd/<NNN-slug>/apply-progress` (the cumulative evidence). Missing either leaves the other stale. The step `tasks.md` files are the recovery copy for both; keep them consistent.
- **Step dependencies are real.** A step's `Depends on:` naming an unfinished step means `blocked`, not "I'll just do both". Crossing an unverified dependency silently breaks `sdd-verify`'s per-step evidence chain.
- In Strict TDD, a GREEN that "passes trivially" (loop runs 0 times, setup doesn't reach the code path, component never renders) is not a GREEN. The `references/strict-tdd.md` TRIANGULATE step is the gate that forces real logic — do not skip it because your first GREEN was green.
- The TDD Cycle Evidence table and the Step Evidence table are **different artifacts with different roles**: TDD table is per-task RED/GREEN/TRIANGULATE/REFACTOR; Step Evidence is per-step focused-test/acceptance-coverage/runtime/rollback. `sdd-tasks` forecasts the change-level guard; `sdd-verify` checks both. Missing either = a partial apply.
- **Workload guard ordering.** The Step 2 check comes BEFORE any code is written — implementing an above-budget slice only to discover the delivery-strategy decision wasn't resolved wastes a batch of work and complicates rollback.
- **Mnemonic ≠ Engram.** No `project:` parameter, no `capture_prompt`. `title == topic_key`, `scope: "project"`, active `session_id`. (See `conventions/mnemonic-memory.md` § Mnemonic Tool Mapping.)
- If you resolved Strict TDD as active and then hit a test-runner infrastructure failure mid-cycle, mark the row FAILED and return `partial` with the reason — do NOT fall back to Standard Mode for the same task (that hides a real test-infra defect and breaks the no-silent-fallback rule).
- **Commits are checkpoints, not decoration.** If the repo has a commit policy and the step is green, commit before returning — the SHA + the `[x]` marks are the recovery record. An uncommitted slice that later gets compacted is an expensive re-run.

## References

- [references/strict-tdd.md](references/strict-tdd.md) — RED → GREEN → TRIANGULATE → REFACTOR cycle, test-layer selection, assertion-quality rules, approval-testing flow, TDD Cycle Evidence table. Load only when Step 4 resolves Strict TDD as active.
- [`../sdd-tasks/SKILL.md`](../sdd-tasks/SKILL.md) — upstream; it created the step tree, per-step `tasks.md`, and the change-level `Review Workload Forecast` you enforce in Step 2.
- [`../sdd-spec/SKILL.md`](../sdd-spec/SKILL.md) — upstream; its per-step `acceptance.feature` scenarios are your acceptance criteria in every task.
- [`../sdd-design/SKILL.md`](../sdd-design/SKILL.md) — upstream; its decisions and per-step WHAT constrain your approach.
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active session), recovery ladder.
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — change-folder layout; each step's `tasks.md` is the live artifact you mark `[x]`; `state.yaml`; `rules.apply`.
- [`../_shared/conventions/mnemonic-code-indexing.md`](../_shared/conventions/mnemonic-code-indexing.md) — the `code_status → code_index → code_search → code_read` ladder for confirming real paths before editing.
- [`../_shared/conventions/commits.md`](../_shared/conventions/commits.md) — the apply commit contract: Conventional Commits, ticket close-token footer, no AI trailers, checkpoint boundaries.
