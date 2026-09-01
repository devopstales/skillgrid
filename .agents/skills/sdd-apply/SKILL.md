---
name: sdd-apply
description: "Implement SDD tasks from the specs and design by writing real code, marking tasks complete, and persisting apply-progress. Use to execute one or more assigned phase tasks after sdd-tasks and before sdd-verify — enforcing RED/GREEN/REFACTOR (Strict TDD) when the project requires it, the review-workload/chained-PR decision, and work-unit evidence. Marks tasks [x] in tasks.md AND persists to Mnemonic. Uses Mnemonic memory + code index; no external binaries."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  family: sdd
  phase-order: "propose → design → spec → tasks → apply"
  prev: [sdd-tasks]
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

You are the APPLY phase — the only phase that writes production code. You take the concrete tasks from `tasks.md`, and implement them by writing real code, following the specs (WHAT) and the design (HOW) strictly. You mark each task `[x]` as it is completed and persist a cumulative `apply-progress` artifact that `sdd-verify` and `sdd-archive` rely on.

Phase order is `propose → design → spec → tasks → apply`. You run last in the planning-to-implementation chain and hand a verified-ready change to `sdd-verify`. Two consequences that drive this phase:

1. **You consume, not invent.** Your acceptance criteria are the spec scenarios; your structural constraints are the design decisions. If a task conflicts with both, flag it in the return summary rather than silently freelancing a third approach.
2. **Progress is cumulative.** Every batch appends to the same `apply-progress` artifact and the same `tasks.md`. You must read prior progress before writing, or you lose completed work from earlier batches.

## What You Receive

From the orchestrator:

- **Change name** (kebab-case)
- **The specific task(s) to implement** (e.g. "Phase 1, tasks 1.1–1.3") — only these. You never pick a broader batch on your own.
- **Strict TDD mode** (`true` | `false`) — resolved by the orchestrator from the project's `testing-capabilities`; if not provided, you resolve it in Step 3.
- **Delivery strategy** and the resolved workload decision (`ask-on-risk` | `auto-chain` | `single-pr` | `exception-ok`, plus the chosen PR slice (`stacked-to-main` / `feature-branch-chain`) or an accepted `size:exception` when applicable).
- Optional: **ticket/issue id** — the apply commit close-token per `_shared/conventions/commits.md`.
- Optional: a `## Skills to load before work` block.

**Artifact store mode is `hybrid` — the only mode for this phase.** Every run does BOTH: updates `openspec/changes/{change-name}/tasks.md` with `[x]` marks **and** persists progress to Mnemonic under `sdd/{change-name}/apply-progress` (upserting the tasks observation for the `[x]` state). A mode token of `openspec` / `engram-compat` / `none` from the orchestrator is honored as `hybrid` here. Do not branch on the mode.

## Execution + Persistence Conventions

Follow, on each save, rather than restating here:

- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active `session_id`; **no** `project:` parameter, **no** `capture_prompt` field; `mem_search` returns previews — always `mem_get_observation(id)` for full content; upsert via same `topic_key`).
- [`../_shared/conventions/openspec.md`](../_shared/conventions/openspec.md) — change-folder layout; `tasks.md` is the live artifact you mark `[x]`; `rules.apply` from `openspec/config.yaml`; the `state.yaml` DAG state.
- [`../_shared/conventions/commits.md`](../_shared/conventions/commits.md) — the apply commit is a checkpoint: Conventional Commits, ticket close-token footer, no AI trailers, one logical change per commit.
- [`references/strict-tdd.md`](references/strict-tdd.md) — the Strict TDD module (RED → GREEN → TRIANGULATE → REFACTOR), loaded ONLY when Step 3 resolves Strict TDD as active.
- [`../sdd-tasks/SKILL.md`](../sdd-tasks/SKILL.md) — upstream; its `Review Workload Forecast` and the work-unit rows are the guard you enforce in Step 2a.

## Skill Loading

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first and follow them while writing code.
2. Otherwise recover context from Mnemonic and the change folder (previews are not enough — always fetch full content):
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/proposal")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**.
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/design")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; your structural constraints.
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/spec")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; your acceptance criteria.
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/tasks")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; the assigned tasks (keep this observation id for the `[x]` upsert).
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/apply-progress")` → `..._mem_get_observation(id)` — prior progress (see Step 2b).
   - `skillgrid-mnemonic_mem_search(query: "sdd-init/{project}")` → `..._mem_get_observation(id)` — detected project facts (stack, testing, tracker).
3. Read the filesystem primary copies in the change folder: `openspec/changes/{change-name}/tasks.md`, `design.md`, and `specs/{domain}/spec.md`.
4. Read `openspec/config.yaml` if present — `context:`, `rules.apply`, and the `strict_tdd` flag bind this phase.

## Status and Workspace Guard

Before reading implementation files or writing code, confirm readiness from the structured state (from the orchestrator, or the `state.yaml` DAG state in the change folder):

- If state is **`blocked`**, STOP and return `blocked` with the missing artifacts or unsafe context.
- If state is **`all_done`**, do not edit. Return `success` with `Next: sdd-verify` (or `sdd-archive` if verify already passed).
- If state is **`ready`**, proceed only on the assigned pending tasks.
- Read context from the artifact paths / topic keys, not assumed fixed filenames. In this SDD pipeline they normally map to proposal, specs, design, and tasks.
- **Edit roots are bounded.** If the orchestrator provides allowed edit roots, edit only files under them; if a needed edit is outside, STOP and report the unsafe path. Never edit files outside the change's affected areas.

## What to Do

### Step 1: Read Context

Before writing ANY code:

1. Confirm the state guard (ready + assigned tasks) from the previous section.
2. Read every applicable artifact (proposal, design, spec, tasks) from Mnemonic and the change folder.
3. Read the specs — understand **WHAT** the code must do. These are your acceptance criteria.
4. Read the design — understand **HOW** to structure the code. These constrain your approach.
5. Read the existing code in affected files — understand current patterns.
6. Check the project's coding conventions from `openspec/config.yaml` and the loaded skills.

Confirm the exact file paths you will touch against the code index — an apply that edits a file it has not read is an apply with a hole:

```
skillgrid-mnemonic_code_status              # stale? file_count 0?
  → if stale, skillgrid-mnemonic_code_index
skillgrid-mnemonic_code_search(query: "<file-or-symbol>", limit: 20)
  → skillgrid-mnemonic_code_read(path: <hit.path>, start_line: <hit.start_line>, end_line: <hit.end_line>)
```

### Step 2: Enforce the Review Workload Decision

Before implementing, inspect the tasks artifact's `Review Workload Forecast` (from `sdd-tasks`).

If the forecast says **any** of:

- `400-line budget risk: High`
- `Chained PRs recommended: Yes`
- `Decision needed before apply: Yes`

Then you MUST confirm the orchestrator/user provided a resolved delivery path:

1. **`auto-chain` or a chosen chained/stacked PR mode** — implement only the assigned work-unit slice, keep its scope autonomous, and report the intended PR boundary. Follow the tasks artifact's `Chain strategy` (`stacked-to-main` or `feature-branch-chain`) for branch targeting.
   - `stacked-to-main`: each PR targets `main` after the previous merges.
   - `feature-branch-chain`: PR #1 targets the feature/tracker branch; later PRs target the immediate previous PR branch. Only the tracker merges to `main`; child PR diffs must stay focused on the current work unit and never target `main` directly. If a child diff shows a previous slice, the base is wrong — retarget/rebase until it is clean.
2. **`exception-ok`** — continue only if the prompt explicitly says the maintainer accepts `size:exception`.
3. **`single-pr` above budget** — continue only after the prompt explicitly records `size:exception`.

If **neither** a delivery decision nor a resolved chain strategy is present for an above-budget forecast, STOP before writing code and return `blocked` with: `Workload decision required before apply: estimated work may exceed 400 changed lines. Ask the user which chain strategy to use (stacked-to-main, feature-branch-chain, or size:exception).`

### Step 3: Read Previous Apply-Progress (if exists)

Before starting work, check for existing apply-progress and MERGE against it — never overwrite:

1. `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/apply-progress")`
2. If found: `skillgrid-mnemonic_mem_get_observation(id)` → read the full content.
3. Also read `openspec/changes/{change-name}/tasks.md` and note which tasks are already `[x]`.
4. Parse which tasks are already complete — **skip them**; start from the first incomplete assigned task.
5. When saving your apply-progress in Step 6, MERGE: include ALL previously completed tasks (copy their status and evidence) PLUS your new completions in a single cumulative artifact.

**CRITICAL**: If the orchestrator told you prior progress exists, you MUST read it first. If you upsert without reading, completed work from prior batches is permanently lost (Mnemonic is working memory — the filesystem copy is the recovery copy, but both must stay consistent).

### Step 4: Resolve TDD Mode

Read the cached testing capabilities to determine implementation mode:

```
Read testing capabilities from:
├── Mnemonic: skillgrid-mnemonic_mem_search("sdd/{project}/testing-capabilities") → mem_get_observation(id)
├── openspec/config.yaml → rules.apply.strict_tdd + test_command
└── Fallback: detect from project files directly (package.json, go.mod, pyproject.toml, etc.)

Resolve mode:
├── IF strict_tdd: true AND a test runner exists
│   └── STRICT TDD MODE → load and follow [references/strict-tdd.md](references/strict-tdd.md) INSTEAD of Step 5
├── IF strict_tdd: false OR no test runner
│   └── STANDARD MODE → use Step 5 (the strict-tdd.md module is never read, never processed)
└── Cache the resolved mode for the return summary
```

**Key principle**: if Strict TDD is **not** active, ZERO TDD instructions are loaded — do not read, process, or reason from `references/strict-tdd.md`.

**Hard gate (Strict TDD only)**: if Strict TDD is active, you MUST produce a **TDD Cycle Evidence** table in the apply-progress artifact — every task row carries RED (test written first) → GREEN (implementation passes) → TRIANGULATE → REFACTOR. A task completed without a test written first is marked FAILED in the table. `sdd-verify` rejects work whose TDD Evidence table is missing or incomplete. There is **no silent fallback**: if you resolved Strict TDD as active, you follow it or you report failure — you do not quietly drop to Standard Mode.

**Hard gate (ALL modes): Work Unit Evidence.** Every assigned work unit, including a single-PR slice in Standard Mode, MUST produce a **Work Unit Evidence** table before its tasks are marked complete:

| Evidence | Required value |
|---|---|
| Focused test command + exact result | Smallest command proving this unit (command, exit/result, relevant counts) |
| Runtime harness command/scenario + exact result | Real integration/runtime path; explicit `N/A` + reason only if no runtime boundary exists |
| Rollback boundary | Exact files/behavior that can be reverted without removing unrelated work |

If the design/tasks carry applicable threat-matrix cases, write and run each mapped RED test **before** the corresponding production change, even in Standard Mode. Preserve Strict TDD's full RED → GREEN → TRIANGULATE → REFACTOR evidence when active; the Work Unit Evidence table supplements it and never replaces it. Do not mark a work unit complete if its focused test or an applicable runtime harness fails.

When all assigned work units finish, **return control to the parent orchestrator.** The executor never launches `sdd-verify`, a review/refutation pass, a correction actor, or a scoped validator on its own — the orchestrator decides the next phase. If only focused remediation of the just-applied slice is needed, do it within this apply batch and fold the result into the Work Unit Evidence before returning; do not start a separate verification cycle from inside apply.

### Step 5: Implement Tasks (Standard Workflow)

Used when Strict TDD is **not** active (or between Strict-TDD tasks that are purely structural). For each assigned task:

```
FOR EACH TASK:
├── Read the task description
├── Read the relevant spec scenarios (these ARE your acceptance criteria)
├── Read the relevant design decisions (these CONSTRAIN your approach)
├── Read existing code patterns in the affected files (match project style)
├── Write the code
├── Run the smallest test/command that proves this task (record it for the Work Unit Evidence)
├── Mark the task complete — change `- [ ]` to `- [x]` in tasks.md IMMEDIATELY
└── Note any issues or deviations
```

Keep each task completable in one sitting; match the project's actual patterns — if the codebase does it differently from what the task implies, follow the existing code (and note the deviation in the summary).

### Step 6: Mark Tasks Complete

Update `openspec/changes/{change-name}/tasks.md` — change `- [ ]` to `- [x]` for each completed task **as you go**, not in one batch at the end:

```markdown
## Phase 1: Foundation

- [x] 1.1 Create `internal/auth/middleware.go` with JWT validation
- [x] 1.2 Add `AuthConfig` struct to `internal/config/config.go`
- [ ] 1.3 Add auth routes to `internal/server/server.go`  ← still pending
```

If you commit this slice (see § Commits), commit after the tasks are green and marked — never before.

### Step 7: Persist Progress (MANDATORY — do not skip)

Follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md). Hybrid = BOTH writes:

1. **Filesystem** — `tasks.md` is already updated in Step 6. The change folder's `tasks.md` is the recovery source of truth.
2. **Mnemonic** — start one session, then save the cumulative apply-progress AND upsert the tasks observation:

```
sid = skillgrid-mnemonic_mem_session_start(title: "sdd/{change-name}/apply")

# Cumulative progress (MERGED with prior batches — see Step 3/9)
skillgrid-mnemonic_mem_save(
  title:      "sdd/{change-name}/apply-progress",
  topic_key:  "sdd/{change-name}/apply-progress",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "{cumulative apply-progress markdown: all completed tasks, evidence, deviations, work-unit tables}"
)

# [x] state of the tasks artifact (upsert — same topic_key replaces the observation)
skillgrid-mnemonic_mem_save(
  title:      "sdd/{change-name}/tasks",
  topic_key:  "sdd/{change-name}/tasks",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "{full tasks.md markdown with updated [x] marks}"
)
```

Mnemonic save notes: `title == topic_key` exactly; `scope: "project"`; pass the active `session_id`; there is **no** `project:` parameter and **no** `capture_prompt` field in the Mnemonic schema — omit both. `topic_key` upserts — saving `sdd/{change-name}/tasks` again replaces it in place (that is how the `[x]` marks propagate to memory).

### Step 8: Merge Protocol

When saving apply-progress:

1. If you read prior progress in Step 3, your artifact MUST include **ALL** previously completed tasks (copy their status and evidence) PLUS your new completions.
2. The saved artifact reflects the **cumulative** state of ALL tasks across ALL batches — a downstream reader of `sdd/{change-name}/apply-progress` sees the whole change, not just this slice.
3. Keep the same structure batch over batch, so a reviewer can diff two saves to see exactly what this batch added.

### Step 9: Commit Boundary (per `_shared/conventions/commits.md`)

For each completed work-unit slice (or single-PR batch), make it a clean, restorable checkpoint:

- Commit only when the slice is green and its tasks are marked `[x]`.
- One logical change per commit (implementation + its tests + enabling config). Only intended files staged — `git status` first.
- Conventional Commits subject, imperative present tense, ≤ 72 chars. Include the ticket close-token footer **only if** a ticket id exists for this work (per the tracker table in `commits.md`). No `Co-authored-by` / AI trailers.
- If applying a `feature-branch-chain` slice, the commit lands on that slice's branch; the orchestrator handles the merge order.
- **Do not commit** work whose tests are red, or whose Work Unit Evidence is incomplete.
- If the orchestrator has not asked for a commit and the repo has no checkpoint policy, leave the working tree staged-but-uncommitted and note it in the return summary.

### Step 10: Self-Check (no external validator binary)

Before returning, confirm each — fix any failure before returning `success`, else return `partial` with the failed item in `risks`:

1. Every completed task is marked `[x]` in `tasks.md` (re-read the file to confirm — internal todos are not evidence).
2. Every assigned task has a matching Work Unit Evidence row (focused test command + result, runtime harness + result, rollback boundary).
3. If Strict TDD is active, the TDD Cycle Evidence table has a RED/GREEN/TRIANGULATE/REFACTOR column per completed task, and no row silently lacks a test-first step.
4. Any applicable design threat-row has its mapped RED test written and run **before** the production change (present in the evidence).
5. The apply-progress upsert and the tasks upsert both succeeded in Mnemonic, and both match the filesystem `tasks.md`.
6. The review-workload decision was satisfied for the assigned slice (Step 2) — or `blocked` was returned.
7. No files were written outside the allowed edit roots.

### Step 11: Return Envelope

**Your FINAL output MUST be text — not a tool call.** Do the `mem_save` calls (Step 7) *before* this text. A trailing tool call buries the analysis in the tool result; returning text is what the orchestrator reads back.

```markdown
## Implementation Progress

**Change**: {change-name}
**Mode**: {Strict TDD | Standard}
**Location**: `openspec/changes/{change-name}/tasks.md` (marked [x]) · Mnemonic `sdd/{change-name}/apply-progress` + `sdd/{change-name}/tasks`
**Status**: success | partial | blocked

### Completed Tasks (this batch)
- [x] {task 1.1 description}
- [x] {task 1.2 description}

### Files Changed
| File | Action | What Was Done |
|------|--------|---------------|
| `path/to/file.ext` | Created | {brief description} |
| `path/to/other.ext` | Modified | {brief description} |

{IF Strict TDD Mode → include the TDD Cycle Evidence table from references/strict-tdd.md}

### Work Unit Evidence
| Unit | Focused test (cmd + result) | Runtime harness (cmd + result) | Rollback boundary |
|------|------------------------------|---------------------------------|-------------------|
| {unit name} | {cmd} → {result} | {cmd/scenario} → {result or N/A+reason} | {files/behavior} |

### Deviations from Design
{List places where the implementation deviated from design.md and why. If none: "None — implementation matches design."}

### Issues Found
{List problems discovered during implementation. If none: "None."}

### Remaining Assigned Tasks
- [ ] {next assigned task}  ← or "None — all assigned tasks complete"

### Workload / PR Boundary
- Mode: {single PR | chained PR slice | stacked PR slice | size:exception}
- Current work unit: {unit name or "N/A"}
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

- ALWAYS read the specs before implementing — the spec scenarios are your acceptance criteria.
- ALWAYS follow the design decisions — do not freelance a different approach. Match the project's ACTUAL patterns where they differ from the task's implication (and note any deviation).
- ALWAYS consume or produce structured state (Step guard) before implementation — do not infer readiness from conversation alone.
- STOP on `blocked` state and do not edit; STOP on an unsafe `actionContext` or an edit outside the allowed roots.
- Mark tasks `[x]` in `tasks.md` **as you go**, not in one batch at the end.
- Before returning, re-read the persisted `tasks.md` and confirm completed tasks are visibly `[x]` — internal todos are not completion evidence.
- NEVER implement tasks that were not assigned to you.
- If a discovery means design or spec is wrong, **note it** in the return summary — do not silently deviate.
- If a task is blocked by something unexpected, STOP and report back — do not improvise around it.
- When applying a chained/stacked PR slice, keep the batch autonomous: one deliverable scope, verification included, clear rollback boundary.
- When applying `size:exception`, state it explicitly in apply-progress and the return summary.
- Apply any `rules.apply` from `openspec/config.yaml`.
- If Strict TDD is resolved active, load `references/strict-tdd.md` and follow its cycle INSTEAD of Step 5; its rules OVERRIDE Step 5 entirely.
- **Hybrid is the only mode** — always mark the filesystem `tasks.md` AND persist to Mnemonic; never branch on `openspec` / `engram-compat` / `none`.
- No external binaries. Mnemonic (`mem_*`) and the code index (`code_*`) are the only knowledge sources; no `gentle-ai`, no `gentleman-ai`, no `sdd-phase-common.md`, no CLI status/validator binary.
- Return envelope per Step 11 — final action is text, not a tool call.

## Gotchas

- **Merging prior progress is load-bearing.** Mnemonic is working memory — an upsert of `apply-progress` without first reading the prior observation silently drops every earlier batch's completed state. `sdd-verify` and `sdd-archive` will then look at a partial picture.
- `mem_search` returns **300-char previews**. A preview of a 2000-char design/spec loses most of it — always `mem_get_observation(id)` before you rely on it as an acceptance criterion or a constraint.
- **The tasks observation upsert is separate from apply-progress.** Step 7 writes TWO Mnemonic saves — `sdd/{change-name}/tasks` (the `[x]` state) and `sdd/{change-name}/apply-progress` (the cumulative evidence). Missing either leaves the other stale. The filesystem `tasks.md` is the recovery copy for both; keep them byte-consistent.
- In Strict TDD, a GREEN that "passes trivially" (loop runs 0 times, setup doesn't reach the code path, component never renders) is not a GREEN. The `references/strict-tdd.md` TRIANGULATE step is the gate that forces real logic — do not skip it because your first GREEN was green.
- The TDD Cycle Evidence table and the Work Unit Evidence table are **different artifacts with different roles**: TDD table is per-task RED/GREEN/TRIANGULATE/REFACTOR; Work Unit Evidence is per-unit focused-test/runtime-harness/rollback. `sdd-tasks` forecasts the latter; `sdd-verify` checks both. Missing either = a partial apply.
- **Workload guard ordering.** The Step 2 check comes BEFORE any code is written — implementing an above-budget slice only to discover the delivery-strategy decision wasn't resolved wastes a batch of work and complicates rollback.
- **Mnemonic ≠ Engram.** No `project:` parameter, no `capture_prompt`. `title == topic_key`, `scope: "project"`, active `session_id`. (See `conventions/mnemonic-memory.md` § Mnemonic Tool Mapping.)
- If you resolved Strict TDD as active and then hit a test-runner infrastructure failure mid-cycle, mark the row FAILED and return `partial` with the reason — do NOT fall back to Standard Mode for the same task (that hides a real test-infra defect and breaks the no-silent-fallback rule).
- **Commits are checkpoints, not decoration.** If the repo has a commit policy and the slice is green, commit before returning — the SHA + the `[x]` marks are the recovery record. An uncommitted slice that later gets compacted is an expensive re-run.

## References

- [references/strict-tdd.md](references/strict-tdd.md) — RED → GREEN → TRIANGULATE → REFACTOR cycle, test-layer selection, assertion-quality rules, approval-testing flow, TDD Cycle Evidence table. Load only when Step 4 resolves Strict TDD as active.
- [`../sdd-tasks/SKILL.md`](../sdd-tasks/SKILL.md) — upstream; its `Review Workload Forecast`, chain strategy, and work-unit rows are the guard you enforce in Step 2.
- [`../sdd-spec/SKILL.md`](../sdd-spec/SKILL.md) — upstream; its scenarios are your acceptance criteria in every task.
- [`../sdd-design/SKILL.md`](../sdd-design/SKILL.md) — upstream; its decisions constrain your approach.
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active session), recovery ladder.
- [`../_shared/conventions/openspec.md`](../_shared/conventions/openspec.md) — change-folder layout; `tasks.md` is the live artifact you mark `[x]`; `state.yaml`; `rules.apply`.
- [`../_shared/conventions/mnemonic-code-indexing.md`](../_shared/conventions/mnemonic-code-indexing.md) — the `code_status → code_index → code_search → code_read` ladder for confirming real paths before editing.
- [`../_shared/conventions/commits.md`](../_shared/conventions/commits.md) — the apply commit contract: Conventional Commits, ticket close-token footer, no AI trailers, checkpoint boundaries.
