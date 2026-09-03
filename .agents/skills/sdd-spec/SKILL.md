---
name: sdd-spec
description: "Author per-step Gherkin acceptance features (Feature/Scenario, Given/When/Then) for each step in the step tree, from the intent and plan. Use after sdd-tasks (which creates the step folders) and before sdd-apply. Translates every per-step WHAT bullet in the plan and every applicable plan threat-matrix row into an observable scenario in that step's acceptance.feature. Uses Mnemonic memory + code index; no external binaries."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  author: devopstales
  version: "2.0"
  family: sdd
  part-of: skillgrid
  phase-order: "init → explore → propose → design → tasks → spec → apply → verify → archive"
  prev: [sdd-propose, sdd-design, sdd-tasks]
  next: [sdd-apply]
  artifact: acceptance (per step)
  delegate_only: true
---

# sdd-spec

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-spec` sub-agent **unless** you loaded this skill directly through the `skill()` tool.

- **Sub-agent (primary)**: you were delegated here by the SDD orchestrator. Continue with the phase work below. Do not re-delegate. Do not call the `skill()` tool again.
- **Orchestrator (skill() loaded this directly)**: STOP. Delegate to the dedicated `sdd-spec` sub-agent using your platform's delegation primitive (e.g. `task(...)`) instead of doing the work inline.

## Purpose

You are the SPEC phase. You express **what the system must do** — for each step in the step tree — as a Gherkin `acceptance.feature` file: one `Feature` per step and 3+ observable `Scenario` lines (Given/When/Then) covering a happy path, an edge case, and a failure state. An `acceptance.feature` is a **WHAT** document: `plan.md` already holds the **HOW** (architecture, data flow, impacted files). If you see yourself writing "the handler calls `validate()` on line 42", you are in plan territory, not spec.

Phase order is `propose → design → tasks → spec`. **You run after `sdd-tasks`** because `acceptance.feature` lives inside `steps/<NN-name>/` — the tree must exist. Two consequences that drive this phase:

1. **The plan's per-step WHAT blocks are your primary input.** Each step in the plan has a "Step — What it delivers" block of 2–4 bullets. Your job is to turn those bullets (plus the intent's success criteria that map to that step) into concrete Given/When/Then scenarios in that step's `acceptance.feature`.
2. **The plan's threat-matrix applicable rows are a spec input.** Every applicable row names an owning step; ensure at least one scenario in **that step's** `acceptance.feature` covers the row's planned RED test. A plan-applicable row that no step's feature covers is a handoff gap; flag it in the envelope.

## What You Receive

From the orchestrator:

- **Change id** — `<NNN-slug>` (e.g. `001-oauth-login`). The folder exists with `intent.md`, `plan.md`, and `steps/<NN-name>/tasks.md` for every step.
- **Artifact store mode** is `hybrid` — the only mode for this phase. Every run does BOTH: writes each `steps/<NN-name>/acceptance.feature` **and** persists to Mnemonic under `sdd/<NNN-slug>/spec` (a single concatenated observation). A mode token of `openspec` / `engram-compat` / `none` from the orchestrator is honored as `hybrid` here. Do not branch on the mode.
- Optional: **ticket/issue id** (carry-through to `sdd-apply`'s commit close-token per `_shared/conventions/commits.md`; spec itself does not use it)
- Optional: a `## Skills to load before work` block

## Execution + Persistence Conventions

Follow, on each save, rather than restating here:

- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — Mnemonic save shape (`title == topic_key`, `scope: "project"`, active `session_id`; **no** `project:` parameter, **no** `capture_prompt` field; `mem_search` returns previews — always `mem_get_observation(id)` for full content).
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — change-folder layout, `acceptance.feature` placement, `rules.spec` from `docs/skillgrid/config.yaml`.
- [`references/acceptance-format.md`](references/acceptance-format.md) — the Gherkin shape, tagging rules, and scenario count floor (1 happy + 1 edge + 1 failure per step), threat-row coverage rules, and traceability rules.
- [`../_shared/conventions/mnemonic-code-indexing.md`](../_shared/conventions/mnemonic-code-indexing.md) — the `code_*` ladder, used only when you want to *verify* a scenario has code to test against (optional; a spec is a WHAT-document and does not require it).
- [`references/threat-matrix.md`](references/threat-matrix.md) — the boundary rows the plan filled in; the applicable ones feed this phase's per-step scenarios (local copy of `sdd-design`'s matrix for a self-contained skill).

## Skill Loading

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first.
2. Otherwise recover inputs from Mnemonic (previews are not enough — always fetch full content):
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/intent")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; success criteria and Step Blueprint.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/plan")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; per-step WHAT blocks and threat-matrix applicable rows.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/tasks")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; the step tree and any step the tasks phase flagged.
   - `skillgrid-mnemonic_mem_search(query: "sdd-init/{project}")` → `..._mem_get_observation(id)` — detected project facts (stack, testing, tracker).
3. Read the filesystem primary copies in the change folder: `docs/skillgrid/changes/<NNN-slug>/intent.md`, `plan.md`, and the `steps/` tree (list the NN folders and read each `tasks.md` — you must write one `acceptance.feature` into each folder).
4. Read `docs/skillgrid/config.yaml` if present — `context:` and `rules.spec` bind this phase.

## What to Do

### Step 1: Walk the step tree

List the step folders in dependency order (NN ascending):

```
ls docs/skillgrid/changes/<NNN-slug>/steps/
```

For every step folder you MUST produce exactly one `steps/<NN-name>/acceptance.feature`. If `sdd-tasks` created a step folder and you do not write an `acceptance.feature` into it, flag it in `risks` — that step will `blocked` at verify time because `sdd-verify` requires the file.

### Step 2: Write each step's acceptance.feature

Follow the format in [`references/acceptance-format.md`](references/acceptance-format.md). The one rule that matters most:

> **An `acceptance.feature` is WHAT, not HOW.** No file paths, function names, or line numbers in Given/When/Then lines. `sdd-apply` writes the implementation; your scenarios are its acceptance contract.

Per step:

```gherkin
# <NN>-<name> acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/<NNN-slug>/)

Feature: <one-line capability for this step>
  As a <role>
  I want <capability>
  So that <value>

  @happy @p0
  Scenario: <happy path — from a plan WHAT bullet>
    Given <precondition>
    When  <action>
    Then  <observable outcome>

  @edge
  Scenario: <edge case — from a plan WHAT bullet or an intent edge-criterion>
    Given <precondition>
    When  <action>
    Then  <outcome at the boundary>

  @failure @p1
  Scenario: <failure / rejection / rollback — from WHAT or threat-row>
    Given <precondition>
    When  <action>
    Then  <expected error / status / fallback>
```

Rules (from `acceptance-format.md`):
- One `Feature` per file.
- ≥ 3 scenarios per step: one `@happy`, one `@edge`, one `@failure`. Omitting one requires a `#` comment with the reason.
- Tags `@happy` / `@edge` / `@failure` / `@p0` / `@p1` / `@security` are the selection contract for `sdd-verify`.
- Scenario names unique within the file; `sdd-apply`'s test task and `sdd-verify`'s compliance table both reference them literally.
- Write the `@happy` and `@edge` scenarios first (from the plan's WHAT bullets), then the `@failure` (from a threat-row if one maps to this step, else the WHAT edge/failure bullet).

### Step 3: Carry the Plan's Applicable Threat Rows into Scenarios

Read the plan's `## Threat Matrix`. For **each row marked `Applicable`**, the plan named an owning step. In **that step's** `acceptance.feature`, ensure at least one scenario covers the row's concrete case (GIVEN/WHEN/THEN matching the plan's expected safe/failure behavior).

- If a scenario covers a row, that is enough — do not force a dedicated `# Required RED tests` section.
- If a plan-applicable row has no covering scenario in its owning step, add one — or flag the gap in the envelope `risks` (do not silently drop it).
- `N/A` rows need no scenario and no action.

### Step 4: Self-Check (no external validator binary)

Before returning, confirm each — fix any failure before returning `success`, otherwise return `partial` with the failed item in `risks`:

1. Every step folder under `steps/` has exactly one `acceptance.feature`.
2. Every `acceptance.feature` has a `Feature:` and ≥ 3 `Scenario:` lines (`@happy`, `@edge`, `@failure`).
3. Every `Scenario` line has Given, When, and at least one Then (Gherkin well-formed).
4. Every plan per-step WHAT bullet is covered by ≥ 1 scenario in that step's file.
5. Every applicable plan threat-row has a covering scenario in the owning step's file (Step 3).
6. No scenario names a file path, function name, or line number (WHAT not HOW).
7. Scenario names are unique within each file.
8. Filesystem `steps/*` `acceptance.feature` set and the Mnemonic concatenated content are **consistent**.

### Step 5: Persist Artifact (hybrid — MANDATORY, do not skip)

Follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md). Hybrid = BOTH writes:

1. **Filesystem** — one `steps/<NN-name>/acceptance.feature` per step.
2. **Mnemonic** — start one session, then one save per change (concatenate steps so a single `mem_get_observation` id retrieves the whole spec):

```
sid = skillgrid-mnemonic_mem_session_start(title: "sdd/<NNN-slug>/spec")

skillgrid-mnemonic_mem_save(
  title:      "sdd/<NNN-slug>/spec",
  topic_key:  "sdd/<NNN-slug>/spec",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "## Step 01-{name}\n\n{full 01 acceptance.feature}\n\n## Step 02-{name}\n\n{full 02 acceptance.feature}\n..."
)
```

One observation per change (concatenated with `## Step {NN}-{name}` headers) keeps the pipeline consistent with `sdd/<NNN-slug>/intent` and `sdd/<NNN-slug>/plan`. `topic_key` upserts — re-running the phase replaces the observation in place. Mnemonic save notes: `title == topic_key` exactly; `scope: "project"`; pass the active `session_id`; there is **no** `project:` parameter and **no** `capture_prompt` field in the Mnemonic schema — omit both.

Do not branch on mode — `hybrid` is the only mode for this phase.

### Step 6: Return Envelope

**Your FINAL output MUST be text — not a tool call.** Do the `mem_save` (Step 5) *before* this text. A trailing tool call buries the analysis in the tool result; returning text is what the orchestrator reads back.

```markdown
## Step Acceptance Created
**Change**: {NNN-slug}
**Status**: success | partial | blocked
**Executive summary**: 1–3 sentences.

### Acceptance coverage
| Step | Feature | Scenarios (H/E/F) | Threat rows covered |
|---|---|---|---|
| 01-{name} | {feature line} | {n}/{n}/{n} | {k applicable → k covered} |
| 02-{name} | {feature line} | {n}/{n}/{n} | {k applicable → k covered} |

**Coverage**: happy {covered|missing} · edge {covered|missing} · failure {covered|missing}
**Plan threat-matrix handoff**: {K applicable rows, all covered} | {list the gaps} | {not applicable}
**Open questions**: {list, or "None"}
**Skill resolution**: paths-injected | fallback-registry | none
**Risks**: {list, or "None"}
**Next**: sdd-apply
```

Close the final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up (per `mnemonic-memory.md` § Session Close Protocol). Do not call `mem_session_summary` here — that is a top-level-agent concern; the orchestrator owns session close.

## Execution Handoff (present after the acceptance is written)

This is the last planning skill before implementation. End by asking your human partner to choose the next move. Present the choice concisely and wait for a decision — do not auto-advance into implementation.

> **For agentic workers:** REQUIRED SUB-SKILL — present exactly these two options and let your human partner pick:
>
> 1. **Implement this plan** — `sdd-apply` (the dispatcher) → use **`subagent-execution`** (recommended: fresh subagent per task with a review gate) or **`simple-execution`** (inline) to implement the step tree task-by-task. Steps use the checkbox (`- [ ]`) syntax in `steps/<NN-name>/tasks.md` for tracking; `sdd-verify` then gates each step.
> 2. **Propose a new idea** — if the step tree or its acceptance isn't what you actually want, go back to **`sdd-propose`** (optionally via the `questioning` skill to clarify intent) before any code is written.
>
> Which one?

Default to option 1 only when the human partner confirms the acceptance covers the real requirement. Any doubt → option 2. Either way, record the chosen path in the `Next:` line of the return envelope (update it to `sdd-apply` or `sdd-propose` depending on the answer) and in the `## Key Learnings` so a resumed session knows the handoff state.

## Rules

- Use Gherkin `Feature`/`Scenario`/Given/When/Then for every step file.
- **One `acceptance.feature` per step folder.** No exceptions.
- Every scenario ≥ 1 Given, ≥ 1 When, ≥ 1 Then.
- Every step ≥ 3 scenarios: happy, edge, failure. Omission requires a `#` comment with the reason.
- Spec is WHAT, not HOW — no file paths, line numbers, function names, or internal symbols in Given/When/Then text.
- Every applicable plan threat-row maps to a scenario in the **owning step** (Step 3).
- Scenario names are unique within a file and referenceable literally by `sdd-apply` and `sdd-verify`.
- Tags `@happy` / `@edge` / `@failure` / `@p0` / `@p1` / `@security` are the selection contract for `sdd-verify`'s test command.
- Apply any `rules.spec` from `docs/skillgrid/config.yaml`.
- **Size budget**: each step's `acceptance.feature` body **under 120 words** (not counting `#` comments or `Feature:` header lines). 3 scenarios × ~3 Given/When/Then lines × ~8 words = ~72 words typical.
- No external binaries. Mnemonic (`mem_*`) and, if you choose, the code index (`code_*`) are the only knowledge sources. No `openspec-cli`, no grammar binary, no `gherkin-lint` binary.
- Return envelope per Step 6 — final action is text, not a tool call.

## Gotchas

- `mem_search` returns **300-char previews.** A preview of a 2000-char plan loses most of its WHAT bullets — always `mem_get_observation(id)`.
- **Writing an `acceptance.feature` with only the happy scenario is the classic v2 trap.** The step will `blocked` at verify time because the `@edge` / `@failure` coverage is missing. The floor is 3 scenarios per step — exception only with a `#` comment.
- **Scenario names that name a WHAT bullet verbatim are a handoff gap for `sdd-apply`.** `sdd-apply`'s test task must reference a *concrete* scenario name — "write a test for the WHAT bullet 'as a user I can log in'" is not a name. Give each scenario a short, unique, stable name.
- **A step without an `acceptance.feature` blocks that step at `sdd-verify` and the whole change at `sdd-archive`.** Do not skip a step folder on the theory that "it is trivial" — the trivial one is usually the one whose edge case breaks production.
- **Mnemonic ≠ Engram.** No `project:` parameter, no `capture_prompt`. `title == topic_key`, `scope: "project"`, active `session_id`. (See `conventions/mnemonic-memory.md` § Mnemonic Tool Mapping.)
- The Mnemonic artifact is **one observation per change** (`sdd/<NNN-slug>/spec`, steps concatenated with `## Step {NN}-{name}` headers) — consistent with the intent and plan saves. Do not split it into one observation per step; recovery is one `mem_get_observation` id.
- Design-before-spec means both the intent and the plan are already here. If either is missing, the correct phase to run is `sdd-design` (not this one) — do not write acceptance from an intent alone; you lose the per-step WHAT and the threat-row handoff (Step 3).
- Do not commit from this phase. Spec is WHAT; `sdd-apply` commits the DO.

## References

- [references/acceptance-format.md](references/acceptance-format.md) — the Gherkin shape, tagging, scenario-floor rule, threat-row coverage rules, and traceability.
- [`../sdd-design/SKILL.md`](../sdd-design/SKILL.md) — upstream; its per-step WHAT blocks and `## Threat Matrix` applicable rows feed Steps 2–3.
- [`../sdd-propose/SKILL.md`](../sdd-propose/SKILL.md) — the intent's **Step Blueprint** and **Success Criteria** (UAT) are the acceptance contract this phase translates into per-step scenarios.
- [`../sdd-tasks/SKILL.md`](../sdd-tasks/SKILL.md) — upstream; it created the `steps/<NN-name>/` tree you fill.
- [`references/threat-matrix.md`](references/threat-matrix.md) — the boundary rows the plan may have marked applicable (local copy of `sdd-design`'s matrix).
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape, session protocol, recovery ladder.
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — change-folder layout, `acceptance.feature` placement, and `rules.spec`.
