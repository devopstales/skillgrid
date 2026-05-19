---
name: sdd-tasks
description: >
  Break down a change into an implementation task checklist.
  Trigger: When the orchestrator launches you to create or update the task breakdown for a change.
license: Apache-2.0
metadata:
  author: devopstales
  version: "1.0"
---

## Purpose

You are a sub-agent responsible for creating the TASK BREAKDOWN. You take the proposal, specs, and design, then produce a `tasks.md` with concrete, actionable implementation steps organized by phase.

For **small same-session work** without a formal change artifact, use **`micro-plan`** (short steps + exit criteria) instead of this skill.

## Norse persona invocations (coordinator)

When orchestrating `/sdd-tasks` (or tasks step inside brainstorm), dispatch per `skills/_shared/sdd-persona-delegation.md`:

| Required | Persona | Capability |
| --- | --- | --- |
| yes | thor | execution-feasibility |
| no | bragi | structured-artifacts |
| no | tyr | tasks-readiness |

## What You Receive

From the orchestrator:
- Change name
- Artifact store mode (`engram | openspec | hybrid | none`)

## Execution and Persistence Contract

- If mode is `engram`:

  **CRITICAL: `mem_search` returns 300-char PREVIEWS, not full content. You MUST call `mem_get_observation(id)` for EVERY artifact. If you skip this, you will work with incomplete data and produce wrong tasks.**

  **STEP A — SEARCH** (get IDs only — content is truncated):

  **Run all artifact searches in parallel** — call all mem_search calls simultaneously in a single response, then all mem_get_observation calls simultaneously in the next response. Do NOT search sequentially.

  1. `mem_search(query: "sdd/{change-name}/proposal", project: "{project}")` → save ID
  2. `mem_search(query: "sdd/{change-name}/spec", project: "{project}")` → save ID
  3. `mem_search(query: "sdd/{change-name}/design", project: "{project}")` → save ID

  **STEP B — RETRIEVE FULL CONTENT** (mandatory for each):

  **Run all retrieval calls in parallel** — call all mem_get_observation calls simultaneously in a single response.

  4. `mem_get_observation(id: {proposal_id})` → full proposal (REQUIRED)
  5. `mem_get_observation(id: {spec_id})` → full spec (REQUIRED)
  6. `mem_get_observation(id: {design_id})` → full design (REQUIRED)

  **DO NOT use search previews as source material.**

  **Save your artifact**:
  ```
  mem_save(
    title: "sdd/{change-name}/tasks",
    topic_key: "sdd/{change-name}/tasks",
    type: "architecture",
    project: "{project}",
    content: "{your full tasks markdown}"
  )
  ```
  `topic_key` enables upserts — saving again updates, not duplicates. (Read `skills/_shared/sdd-phase-common.md`.)

  (See `skills/_shared/engram-convention.md` for full naming conventions.)
- If mode is `openspec`: Read and follow `skills/_shared/openspec-convention.md`.
- If mode is `hybrid`: Follow BOTH conventions — persist to Engram AND write `tasks.md` to filesystem. Retrieve dependencies from Engram (primary) with filesystem fallback.
- If mode is `none`: Return result only. Never create or modify project files.

## What to Do

### Step 1: Load Skills

The orchestrator provides your skill path in the launch prompt. Load it now. If no path was provided, proceed without additional skills.

> Read `skills/_shared/sdd-phase-common.md` for the engram upsert note and return envelope format.

### Step 2: Vertical Slice Validation (Context Budget Gate)

**Goal:** Guarantee that every vertical slice in the PRD fits a fresh agent’s cognitive window before we write detailed tasks.

**Input:** PRD file (see `What You Receive`).

**Actions:**

1. Open the PRD and locate the `#### Implementation tasks` list. Each top‑level checkbox item is a vertical slice.
2. For **each slice**, do the following:

 a. **Estimate file count** – Use the PRD’s `#### Codebase touchpoints` plus your codebase knowledge to assign a realistic number of files/modules the slice would touch.
 b. **Determine safety** – Compare the count against the threshold:
    - Default threshold: **5 files**.  
      If `.skillgrid/config.json` contains a key `contextBudgetThreshold`, use that value instead.
    - If file count ≤ threshold → **Context budget: safe**
    - If file count > threshold → **Context budget: RISK**
 c. **Split trigger** – If the slice is marked `RISK`, propose a concrete decomposition into two or more smaller slices that each stay within the safe threshold. Write the split suggestion as a bullet list.
 d. **Preserve labels** – Keep the original `[AFK]` or `[HITL]` label. Do **not** change it based on the budget. The `sdd-apply` gate will handle the override later.

3. **Rewrite the slice list** in the PRD. Replace the raw `Implementation tasks` with an enriched version that follows this exact format:

```markdown
#### Implementation tasks

- [ ] `[AFK]` **Basic email/password login**
  - Goal: User can register, verify email, login and receive access token.
  - Files: 3
  - Context budget: safe
  - Split trigger: None

- [ ] `[AFK]` **Session timeout and token refresh**
  - Goal: Access tokens expire after 15 min; silent refresh using refresh token.
  - Files: 2
  - Context budget: safe
  - Split trigger: None
```

If you split an oversized slice, replace the original single slice with the new smaller ones and adjust the count accordingly. Update the PRD’s `#### Decomposition` section to reflect the new boundaries.

    Report to user (inside your return summary, or immediately if interactive): number of slices, how many are safe vs. RISK, and any splits performed.

Only after this step is complete and the PRD is updated may you proceed to the actual task breakdown.

### Step 3: Break Down Each Slice into Tasks

**Now, for each validated vertical slice** (as they appear in the updated `Implementation tasks` list):

1. Read the slice's goal, acceptance criteria, and any technical constraints.
2. Produce a concrete, ordered list of implementation steps (one line per step).
3. Tag each step with `[Slice: <name>] [Label: AFK|HITL] [Reason: <why this label applies>] [Budget: safe|RISK]`.
4. If the slice is `[HITL]`, make the very first step a human‑decision point.
5. Group steps under slice headings (`### Slice: <name>`).

**Save the tasks file:**

- **IF mode is `openspec` or `hybrid`:** Create the file right now:


```
openspec/changes/{change-name}/
├── proposal.md
├── specs/
├── design.md
└── tasks.md               ← You create this
```

- **IF mode is `engram` or `none`:** Do NOT create any `openspec/` directories or files. Compose the tasks content in memory — you will persist it in Step 4.

### Vertical Slice Rules

Each slice is a **tracer bullet** — a thin vertical slice that cuts through ALL integration layers end-to-end, NOT a horizontal slice of one layer.

- Each slice delivers a narrow but COMPLETE path through every layer (schema, API, UI, tests)
- A completed slice is demoable or verifiable on its own
- Prefer many thin slices over few thick ones
- Slices may be 'HITL' or 'AFK'. HITL slices require human interaction (architectural decision, design review). AFK slices can be implemented and merged without human interaction.

### Vertical Slice Rules

Each slice is a **tracer bullet** — a thin vertical slice that cuts through ALL integration layers end-to-end, NOT a horizontal slice of one layer.

- Each slice delivers a narrow but COMPLETE path through every layer (schema, API, UI, tests)
- A completed slice is demoable or verifiable on its own
- Prefer many thin slices over few thick ones
- Slices may be 'HITL' or 'AFK'. HITL slices require human interaction (architectural decision, design review). AFK slices can be implemented and merged without human interaction.

### Task Writing Rules

Each task MUST be:

| Criteria | Example ✅ | Anti-example ❌ |
|----------|-----------|----------------|
| **Specific** | "Create `internal/auth/middleware.go` with JWT validation" | "Add auth" |
| **Actionable** | "Add `ValidateToken()` method to `AuthService`" | "Handle tokens" |
| **Verifiable** | "Test: `POST /login` returns 401 without token" | "Make sure it works" |
| **Small** | One file or one logical unit of work | "Implement the feature" |


#### Task Format

Each task must be tagged with:
```
[Slice: <slice-name>] [Label: AFK|HITL] [Reason: <brief gate rationale>] [Budget: safe|RISK]
```

If the slice is [HITL], make the very first step a human decision point, e.g.:

```
- [ ] DECISION: Choose between Redis or JWT‑only session storage.  [Slice: session] [Label: HITL] [Reason: architecture decision required] [Budget: safe]
```

Group the steps by slice in the output. You may use nested headings like `### Slice: <name>` and then list the steps. Keep steps ordered inside each slice.

#### Task File Format

```markdown
# Tasks: {Change Title}

## Vertical Slice 1: <Slice Name> (Label: AFK, Budget: safe)

- [ ] 1.1 Create `internal/auth/middleware.go` with JWT validation. [Slice: <name>] [Label: AFK] [Reason: clear scoped implementation] [Budget: safe]
- [ ] 1.2 Add `POST /auth/register` endpoint. [Slice: <name>] [Label: AFK] [Reason: API behavior fully specified] [Budget: safe]
- [ ] 1.3 Write unit tests for `AuthService.Login()`. [Slice: <name>] [Label: AFK] [Reason: deterministic verification step] [Budget: safe]

## Vertical Slice 2: <Slice Name> (Label: AFK, Budget: safe)

- [ ] 2.1 Add login rate limiting middleware. [Slice: <name>] [Label: AFK] [Reason: bounded middleware change] [Budget: safe]
- [ ] 2.2 Update swagger docs. [Slice: <name>] [Label: AFK] [Reason: documentation-only update] [Budget: safe]
```

Important: Tasks are always listed under their vertical slice; do not mix slices inside a flat phase grouping. If a slice needs internal phasing (e.g., foundation, core, test), you may add sub‑headings like `#### Foundation`, `#### Core` within the slice, but the top‑level grouping remains the slice.

### Slice‑Internal Task Ordering

Within a single vertical slice, order steps logically so that dependencies are satisfied:

| Order | What to do |
|-------|------------|
| 1 | New types, interfaces, config changes the rest of the slice needs. |
| 2 | Core logic and behaviour. |
| 3 | Integration and wiring. |
| 4 | Tests (unit, integration, spec‑based). |
| 5 | Cleanup / documentation (if needed). |

If a slice is small enough, you may collapse some of these levels into a single flat list. Always keep the steps sequential and bounded by the slice.

### Step 3.5: Enforce TDD and Granularity Validation (Mandatory)

Before persisting the task list, run validations:

**Validation A — TDD enforcement check** (if project uses TDD):
- Every implementation task must be preceded or accompanied by a corresponding test task
- No implementation task appears without explicit test-first indicator
- Each test task follows RED phase pattern: write failing test, capture expected failure

**Validation B — Granularity check**:
- Count expected file touches per task (from "Files:" section)
- If any task touches >3 files → flag for split
- If any task duration estimate >5 minutes → flag for split

**Validation C — No-placeholders scan**:
- Search for: TODO, TBD, "implement later", "fill in", "appropriate", "proper", "etc"
- Any placeholder → FAIL validation

**Validation D — Completeness check**:
- Every task has: exact file paths OR code blocks OR exact commands
- No vagueness: "handle errors" → "throw ValidationError with message"
- No references to undefined types/functions

Use the label validation script:
```bash
.skillgrid/scripts/validate-task-labels.sh
```

Fix ALL failures before persisting. If unable to resolve, escalate to human with clear rationale.

### Step 4: Persist Artifact

**This step is MANDATORY — do NOT skip it.**

If mode is `engram`:
```
mem_save(
  title: "sdd/{change-name}/tasks",
  topic_key: "sdd/{change-name}/tasks",
  type: "architecture",
  project: "{project}",
  content: "{your full tasks markdown from Step 3}"
)
```

If mode is `openspec` or `hybrid`: the file was already written in Step 3.

If mode is `hybrid`: also call `mem_save` as above (write to BOTH backends).

If you skip this step, the next phase (sdd-apply) will NOT be able to find your tasks and the pipeline BREAKS.

### Step 5: Return Summary

Return to the orchestrator:

```markdown
## Tasks Created

**Change**: {change-name}
**Location**: `openspec/changes/{change-name}/tasks.md` (openspec/hybrid) | Engram `sdd/{change-name}/tasks` (engram) | inline (none)

### Breakdown
| Phase | Tasks | Focus |
|-------|-------|-------|
| Phase 1 | {N} | {Phase name} |
| Phase 2 | {N} | {Phase name} |
| Phase 3 | {N} | {Phase name} |
| Total | {N} | |

### Implementation Order
{Brief description of the recommended order and why}

### Next Step
Ready for implementation (sdd-apply).
```

## Rules

- ALWAYS reference concrete file paths in tasks
- Tasks MUST be ordered by dependency — Phase 1 tasks shouldn't depend on Phase 2
- Testing tasks should reference specific scenarios from the specs
- Each task should be completable in ONE session (if a task feels too big, split it)
- Use hierarchical numbering: 1.1, 1.2, 2.1, 2.2, etc.
- NEVER include vague tasks like "implement feature" or "add tests"
- ALWAYS include `[Label: ...]` and `[Reason: ...]` tags on every actionable checkbox task line so automated label gating can validate the file
- Apply any `rules.tasks` from `openspec/config.yaml`
- **TDD requirement:** If project uses TDD (detected from config or presence of test infra), every implementation task MUST be paired with explicit test-first tasks in RED-GREEN-REFACTOR sequence:
  - RED task: write failing test, capture expected failure output
  - GREEN task: minimal implementation, make test pass
  - REFACTOR task: extract helpers, rename, remove duplication
  - No GREEN task without preceding RED task in same slice
- **Size budget:** Tasks artifact must be under 530 words. Each task: 1-2 lines max. Use checklist format, not paragraphs.
- **Completeness contract:** No placeholders (TBD/TODO/"later"/"appropriate"/"proper"/"etc"). Every code block must be complete and executable as written. Every command must have expected output. Every file modification must specify exact line ranges or full file content.
- **Return:** the standard SDD envelope per `skills/_shared/sdd-return-envelope.md` (see `skills/_shared/sdd-phase-common.md`)
