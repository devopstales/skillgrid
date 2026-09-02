---
name: sdd-propose
description: "Create an SDD change intent (WHY & WHAT: business goals, UAT criteria, step blueprint) from a research analysis, and reserve the three-digit change number. Use when the orchestrator needs an intent.md before design/tasks phases."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  author: devopstales
  version: "2.0"
  family: sdd
  part-of: skillgrid
  phase-order: "init → explore → propose → design → tasks → spec → apply → verify → archive"
  prev: [sdd-explore]
  next: [sdd-design]
  artifact: intent
  delegate_only: true
---

# sdd-propose

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-propose` sub-agent **unless** you loaded this skill directly through the `skill()` tool.

- **Sub-agent (primary)**: you were delegated here by the SDD orchestrator. Continue with the phase work below. Do not re-delegate. Do not call the `skill()` tool again.
- **Orchestrator (skill() loaded this directly)**: STOP. Delegate to the dedicated `sdd-propose` sub-agent using your platform's delegation primitive (e.g. `task(...)`) instead of doing the work inline.

## Purpose

You are the INTENT phase. Your job is to take the research analysis (from `sdd-explore`) — or direct user input — and produce a structured `intent.md` inside the reserved change folder. You capture **WHY** we are doing this and **WHAT** success looks like at the user level, plus the **step blueprint**: the coarse, sequential steps this change will decompose into. You shape business goals, UAT criteria, scope, risks, and rollback before anyone plans, decomposes, or implements.

You also own **change-number reservation**: you choose the next free `NNN` and name the change folder before any artifact is written.

## What You Receive

From the orchestrator:

- **Change slug** (kebab-case, e.g. `oauth-login`) — the NNN number may be assigned by the orchestrator or left to you to reserve.
- **Research analysis** (from `sdd-explore`) OR a direct user description.
- **Artifact store mode** is `hybrid` — the only mode for this phase. Every run does BOTH: writes `docs/skillgrid/changes/<NNN-slug>/intent.md` **and** persists to Mnemonic under `sdd/<NNN-slug>/intent`. A mode token of `openspec` / `engram-compat` / `none` from the orchestrator is honored as `hybrid` here. Do not branch on the mode.

## Skill Loading

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first.
2. Otherwise, recover context: `mem_search(query: "sdd/<NNN-slug>/research")` + `mem_get_observation(id)`, then `mem_search(query: "sdd-init/{project}")` + `mem_get_observation(id)` for detected project facts (stack, testing, tracker).
3. Read `docs/skillgrid/config.yaml` if present — it carries `rules` (including `rules.intent`).
4. Read prior related changes from `docs/skillgrid/archive/NNN-slug/` when the slug or domain overlaps — their `intent.md` / `plan.md` are prior art.

## What to Do

### Step 0: Shape the Intent (interactive mode only)

In interactive SDD mode, do not let the executor silently decide if the input is "clear enough." Run an **intent question round** before finalizing — focus on business/product, **not** harness mechanics (test commands, PR shape, line budgets):

1. **Business problem** — pain, opportunity, or cost that justifies this change now
2. **Target users & situations** — who is affected, in which workflow, urgency
3. **Business rules** — policies, permissions, thresholds, compliance/domain invariants
4. **User-observable outcome** — what should feel/work/possible after
5. **Current-state gap** — what is wrong, missing, or inconsistent today
6. **Implications & impact** — teams, data, UX, support/operational processes
7. **Edge cases** — empty states, partial data, failures, migrations, conflicting needs
8. **Decision gaps** — unknowns that make the intent ambiguous or over-broad
9. **Scope boundaries & non-goals** — what's in the first slice vs deferred
10. **Business risk / tradeoff** — downside that matters if the direction is wrong

Prefer 3–5 concrete questions per round. After answers, summarize resulting assumptions and ask: *correct anything, or another round?*

The reusable `questioning` skill implements the shared clarification primitive (classify + design tree, frontier, rounds, recommendations, approval gate); invoke it when you need a deeper requirement-stress session before writing the intent. If you cannot ask the user directly, embed a `## Intent question round` section in the result with the questions and assumptions needing review.

### Step 1: Reserve the Change Number

This phase **owns the NNN number**. Resolve the next free one before creating the folder:

1. If the orchestrator supplied a number, use it — but still verify it is free.
2. Otherwise, scan existing numbers:
   - `ls docs/skillgrid/changes/ | grep -Eo '^([0-9]{3})' | sort -n | tail -1`
   - `ls docs/skillgrid/archive/ | grep -Eo '^([0-9]{3})' | sort -n | tail -1`
   - `mem_search(query: "sdd/{project}/changelog")` → `mem_get_observation(id)` for any archived entries not yet reflected on disk.
3. Take `max + 1`, zero-pad to 3 digits. If no changes exist, use `001`.
4. The full change id is `<NNN>-<slug>` (e.g. `001-oauth-login`). If the slug already collides (same NNN with a different slug), bump NNN until the pair is free.

Record the reservation in Mnemonic `sdd/{project}/changelog` as a one-line entry: `<NNN>-<slug>: <intent-title> (reserved by sdd-propose, {ISO date})`. This observation is extended, not replaced — each reservation is its own line; recover the full history with `mem_get_observation(id)`.

### Step 2: Create the Change Directory

Create the change folder (hybrid mode always writes the file):

```
docs/skillgrid/changes/<NNN-slug>/
└── intent.md
```

### Step 3: Write intent.md

```markdown
# Intent: {NNN-slug — Change Title}

## Business Problem
{What problem are we solving? Why now? Be specific about the user need or tech debt.}

## Target Users & Situations
- {Who is affected, in which workflow, with what urgency}
- {Second persona/situation if any}

## Business Rules
- {Policy / permission / threshold / domain invariant this change must respect}
- {Second rule if any}

## Success Criteria (UAT-level)
Measurable, user-observable outcomes. These become the acceptance contract `sdd-spec` will translate into per-step Gherkin scenarios:

- [ ] {User can ... / system must ... — observable, not implementation-shaped}
- [ ] {Second criterion}
- [ ] {Edge / failure behavior the user can observe}

## Scope

### In Scope
- {Concrete deliverable 1}
- {Concrete deliverable 2}

### Out of Scope
- {What we are explicitly NOT doing}
- {Related future work, deferred}

## Step Blueprint
> CONTRACT with the sdd-tasks phase: these names tell tasks exactly which step folders to create under `steps/`. Use `<NN>-<step-name>` kebab-case IDs (NN = 2-digit sequential, step-name = verb-noun slug). Leave empty if the change is a single step.

- `01-<step-slug>`: {one-line goal for this step}
- `02-<step-slug>`: {one-line goal}
- `03-<step-slug>`: {one-line goal}

## Affected Areas
| Area | Impact | Description |
|------|--------|-------------|
| `path/to/area` | New / Modified / Removed | {What changes} |

## Risks
| Risk | Likelihood | Mitigation |
|------|------------|------------|
| {Risk description} | Low / Med / High | {How we mitigate} |

## Rollback Plan
{How to revert if something goes wrong. Be specific.}

## Dependencies
- {External dependency or prerequisite, if any}
```

If an `intent.md` already exists in the change folder, READ it first and UPDATE it — do not overwrite blindly.

### Step 4: Persist Artifact

This step is **MANDATORY** — do not skip it.

**Filesystem path** (follow [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md)):

```
docs/skillgrid/changes/<NNN-slug>/intent.md
```

- Always create the change folder before writing (Step 1 reserved the NNN; create the folder here).
- If the file already exists, READ then UPDATE (merge, preserve valid prior content).

**Mnemonic** (follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md)):

```
sid = skillgrid-mnemonic_mem_session_start(title: "sdd/<NNN-slug>/intent")

skillgrid-mnemonic_mem_save(
  title:        "sdd/<NNN-slug>/intent",
  topic_key:    "sdd/<NNN-slug>/intent",
  type:         "architecture",
  scope:        "project",
  session_id:   "{sid}",
  content:      "{full markdown content}"
)

# Append the reservation line to the project changelog (upsert the whole history)
changelog = mem_get_observation( mem_search("sdd/{project}/changelog").id )  # if exists
skillgrid-mnemonic_mem_save(
  title:        "sdd/{project}/changelog",
  topic_key:    "sdd/{project}/changelog",
  type:         "config",
  scope:        "project",
  session_id:   "{sid}",
  content:      "{existing changelog}\n{NNN}-{slug}: {intent title} (reserved by sdd-propose, {ISO date})"
)
```

- `topic_key` enables upsert — saving again updates in place; do not create near-duplicates.
- Hybrid is the only mode for this phase: do the filesystem write (Step 3) and the Mnemonic saves; do not branch on `openspec` / `engram-compat` / `none`.

### Step 5: Return Summary

Return to the orchestrator:

```markdown
## Intention Set
**Change**: {NNN-slug} (number reserved by this run)
**Location**: `docs/skillgrid/changes/<NNN-slug>/intent.md` | Mnemonic `sdd/<NNN-slug>/intent`

**Status**: success | partial | blocked
**Summary**: 1-2 sentence summary of the intent
**Business problem**: {one-line business problem}
**Scope**: {N in, M out}
**Step blueprint**: {N steps planned: 01-…, 02-…, …}
**Risk Level**: Low / Medium / High
**Changelog**: appended `sdd/{project}/changelog` (observation {id})
**Next**: sdd-design
```

## Rules

- ALWAYS create `intent.md` (hybrid mode — the only mode for this phase).
- ALWAYS reserve the change number before creating the folder (Step 1).
- Every intent MUST have a rollback plan.
- Every intent MUST have measurable success criteria — these become the acceptance contract.
- The **Step Blueprint** is the contract with `sdd-tasks` — always fill it. If nothing decomposes into more than one step, write a single `01-…` entry; do not leave it as a template placeholder.
- Use concrete file paths in **Affected Areas** when possible.
- Apply any `rules.intent` from `docs/skillgrid/config.yaml`.
- **Size budget**: the intent artifact MUST be under 500 words. Use bullets and tables over prose.
- Recovery: `mem_search` returns 300-char previews only — always `mem_get_observation(id)` for full content before relying on it.
- At session end: call `mem_session_summary` then `mem_session_end`.

## Gotchas

- `mem_search` returns 300-char previews. Never use a preview as source material — always call `mem_get_observation(id)` for full content.
- The **Step Blueprint** is what drives `sdd-tasks` step allocation. Leaving it as a template placeholder leaves `sdd-tasks` guessing the step count — that is a handoff gap, not a task defect.
- "Out of Scope" is as important as "In Scope" — it prevents scope creep in later phases.
- In interactive mode, the question round must stay on business/product questions, not delivery mechanics. The user is the domain expert, not the delivery configurator.
- If a prior `intent.md` exists and you UPDATE it, preserve any content the user hand-approved in earlier rounds — only revise the sections the new input affects.
- **NNN reservation is idempotent per slug pair.** If `001-oauth-login` already exists in `changes/` or `archive/`, the next NNN must be used — do not collide. Check both folders and the Mnemonic changelog before reserving.
