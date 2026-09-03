---
name: sdd-design
description: "Create the SDD plan (architecture, data flow, impacted files map, per-step WHAT, threat matrix) from the intent. Use when the orchestrator needs a plan.md between sdd-propose and sdd-tasks. Reads the intent, reads the actual code via the Mnemonic code index, defends each decision with a rationale, states what each step must deliver in WHAT terms, and fills an applicability-driven threat matrix. No external binaries — Mnemonic memory + code index only."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  author: devopstales
  version: "2.0"
  family: sdd
  part-of: skillgrid
  phase-order: "init → explore → propose → design → tasks → spec → apply → verify → archive"
  prev: [sdd-propose]
  next: [sdd-tasks]
  artifact: plan
  delegate_only: true
---

# sdd-design

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-design` sub-agent **unless** you loaded this skill directly through the `skill()` tool.

- **Sub-agent (primary)**: you were delegated here by the SDD orchestrator. Continue with the phase work below. Do not re-delegate. Do not call the `skill()` tool again.
- **Orchestrator (skill() loaded this directly)**: STOP. Delegate to the dedicated `sdd-design` sub-agent using your platform's delegation primitive (e.g. `task(...)`) instead of doing the work inline.

## Purpose

You are the PLAN phase. You take the intent (from `sdd-propose`) and the research (from `sdd-explore`) and produce a concrete `plan.md` that captures **HOW** the change will be implemented: architecture decisions, data flow, impacted-files map, and — because the new model has no separate RFC 2119 spec layer — **per-step WHAT**: for every step in the intent's Step Blueprint, what observable behavior that step must deliver. Your `plan.md` is what `sdd-tasks` decomposes into execution and what `sdd-spec` translates into Gherkin `acceptance.feature` scenarios.

You design the *shape* so `sdd-tasks` can break it into work and `sdd-spec` can state it as observable acceptance.

## What You Receive

From the orchestrator:

- **Change id** — `<NNN-slug>` (e.g. `001-oauth-login`). The folder already exists and holds `intent.md` (and usually `research.md`).
- **Artifact store mode** is `hybrid` — the only mode for this phase. Every run does BOTH: writes `docs/skillgrid/changes/<NNN-slug>/plan.md` **and** persists to Mnemonic under `sdd/<NNN-slug>/plan`.   Do not branch on the mode.
- Optional: **ticket/issue id** (for the eventual `sdd-apply` commit close-token per `_shared/conventions/commits.md`)
- Optional: a `## Skills to load before work` block

## Execution + Persistence Conventions

This skill does not restate them. Follow, on each save:

- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — Mnemonic save shape (`title == topic_key`, `scope: "project"`, active `session_id`; `mem_search` returns **previews only** — always `mem_get_observation(id)` for full content).
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — change-folder layout, write-before-reuse rule, `rules.plan` from `docs/skillgrid/config.yaml`.
- [`../_shared/conventions/mnemonic-code-indexing.md`](../_shared/conventions/mnemonic-code-indexing.md) — the `code_status → code_index → code_search → code_read` ladder you use to read the codebase. **No external binaries**: `gitnexus`, `gentle-ai`, `tree-sitter-cli`, and friends are not part of this phase.

## Skill Loading

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first.
2. Otherwise recover context from Mnemonic (previews are not enough — always fetch full content):
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/intent")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required input**; the Step Blueprint is your contract.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/research")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required input**.
   - `skillgrid-mnemonic_mem_search(query: "sdd-init/{project}")` → `skillgrid-mnemonic_mem_get_observation(id)` — detected project facts (stack, testing, tracker).
   - `skillgrid-mnemonic_mem_search(query: "skill-registry")` → `skillgrid-mnemonic_mem_get_observation(id)` — the skill index; if missing, fall back to the disk file `docs/agents/skill-registry.md`.
3. Read `docs/skillgrid/config.yaml` if present — `context:` and `rules.plan` bind this phase.
4. Read prior related changes from `docs/skillgrid/archive/NNN-slug/` (their `plan.md` / intent.md) where this plan must not contradict established behavior — the plan is a *delta*, not greenfield.
5. Load the `codebase-design` skill when the change introduces or restructures modules — its vocabulary (Module / Interface / Implementation / Seam / Adapter / Depth / Leverage / Locality) is mandatory in the `## Architecture Decisions` and `## Impacted Files Map` sections below.
6. Load the `glossary` skill before writing `plan.md` — run a close-term check for every new term you introduce and write `plan-glossary-reference.md` next to `plan.md` in the same change folder (always-on companion).

## What to Do

### Step 1: Read the Code You Will Change (via the code index)

Never plan from memory or from the intent's prose alone. For each symbol, module, or integration point the intent names:

```
skillgrid-mnemonic_code_status              # stale? file_count 0?
  → if stale, skillgrid-mnemonic_code_index
skillgrid-mnemonic_code_search(query: "<symbol-or-phrase>", limit: 20)
  → for each hit you actually will touch,
skillgrid-mnemonic_code_read(path: <hit.path>, start_line: <hit.start_line>, end_line: <hit.end_line>)
```

Rules (from `conventions/mnemonic-code-indexing.md`):

- Search first, then read the matching slice. **Never read a whole file speculatively** — a plan that cites a file it never read is a plan with a hole.
- Prefer the code index over `rg`/`grep` when exploring unfamiliar areas; use `rg`/`grep` for exact-identifier lookups the index is a poor fit for.
- If the index is empty and `code_index` cannot run, fall back to reading the files directly — and say so in the envelope's `risks`.

Confirm before planning: entry points, module boundaries, existing patterns, dependency/interface contracts, and the test infrastructure the new work will plug into.

### Step 2: Per-step WHAT (replaces the old spec layer)

Because the new model has no RFC 2119 `specs/{domain}/spec.md`, **`plan.md` carries the per-step WHAT.** For **every step in the intent's Step Blueprint**, write a short "What this step delivers" block with observable, user-facing behavior — no file paths, no line numbers, no function names. These blocks are the raw material `sdd-spec` turns into Gherkin scenarios in that step's `acceptance.feature`.

Format (one block per step, matching the intent's Step Blueprint IDs exactly):

```markdown
### Step 01-<step-slug> — What it delivers
- As a {role}: I can {observable action} and see {observable outcome}
- Given {precondition}, when {action}, then {observable behavior}
- Edge: given {boundary / failure precondition}, when {action}, then {observed fallback}
```

Keep each block to 2–4 bullets. A block that names a file path or a function is HOW, not WHAT — move it to Architecture or Impacted Files.

### Step 3: Applicability-Driven Threat Matrix

Read the applicability trigger list in [references/threat-matrix.md](references/threat-matrix.md). It lists the boundaries where a plan can silently break (routing, shell/subprocess, VCS/PR automation, executable-file classification, process integration — **plus** skillgrid-specific rows: a new/modified Mnemonic tool contract, and a change to any `_shared/conventions/*` file).

- If **any** row is applicable, include the full matrix in `plan.md`, mark each row `Applicable` or explicit `N/A: reason`, and name the expected safe behavior, failure behavior, and the step that owns the RED test for every applicable row.
- If **none** apply (a pure ref, doc, or additive-feature change away from every listed boundary), record the matrix as not applicable in one line — do not manufacture `N/A` rows.

**Applicable rows are plan requirements.** They MUST propagate unchanged into `sdd-tasks` (as RED-test tasks ordered before their production task) and then `sdd-spec` (as a scenario in the owning step's `acceptance.feature`). An explicit `N/A` row requires no test and no scenario, but must carry a reason a reviewer can challenge.

### Step 4: Write plan.md

Create the file in the change folder (hybrid mode always writes it):

```
docs/skillgrid/changes/<NNN-slug>/
├── intent.md
├── research.md    (usually, from sdd-explore)
├── plan.md        ← you create this
└── steps/         (comes later, from sdd-tasks)
```

If a `plan.md` already exists in the change folder, **READ it first and UPDATE it** — do not overwrite a prior plan's valid content.

#### Companion glossary reference (always-on)

Per the `glossary` skill, also write `plan-glossary-reference.md` in the same change folder. Format:

```markdown
# Glossary Reference

| Term | Source Glossary | Context |
| --- | --- | --- |
| <Term> | `docs/skillgrid/glossary/technical.md` | <Short context for how the plan uses this term.> |
```

If no glossary terms are used, write `No glossary terms referenced.` on one line. Do not copy definitions into the companion file.

#### Plan document format

```markdown
# Plan: {NNN-slug — Change Title}

## Technical Approach
{Concise overall strategy. How it maps to the intent's success criteria. Reference prior changes (archive/NNN-slug/plan.md) it must not contradict.}

## Architecture Decisions

Use the `codebase-design` vocabulary: each Decision block names the **Module** and its **Interface**, places the **Seam**, names the **Adapter** (when a seam is real — at least two concrete adapters), and assesses **Depth** (is the interface small relative to the behaviour it exposes?). Apply the deletion test, the test-surface rule, and the seam discipline from the `codebase-design` skill.

### Decision: {title}
**Module / Interface / Seam / Adapter / Depth**: {one line each — vocabulary comes from codebase-design}
**Choice**: {what we chose}
**Alternatives considered**: {what we rejected, briefly}
**Rationale**: {why this over the others}

## Data Flow
{How data moves through the system for this change. ASCII diagram when it helps:}

    Component A ──▶ Component B ──▶ Component C
         │                              │
         └──────── Store ───────────────┘

## Impacted Files Map
| File | Action | Step | Description |
|------|--------|------|-------------|
| `path/to/new-file` | Create | 01 | {what it does} |
| `path/to/existing` | Modify | 02 | {what changes and why} |
| `path/to/old` | Delete | 03 | {why} |

The **Step** column assigns each file to the step that owns its change. `sdd-tasks` uses it so the file appears in exactly one step's `tasks.md`.

## Step WHAT
{One block per intent Step Blueprint entry — see Step 2 format above.}

## Interfaces / Contracts
{New interfaces, API contracts, types, data structures. Code blocks in the project's language, only where non-obvious.}

## Mnemonic Integration
{Only when relevant. New `sdd/<NNN-slug>/…` topic keys this plan anticipates; any change to existing Mnemonic contracts (a new `mem_save` shape, a `code_search` caller, a new code-index expectation); any new tool surface.}
{If the plan does NOT touch the memory surface, write "No Mnemonic contract changes." on one line.}

## Threat Matrix
{Applicability matrix from references/threat-matrix.md — every row marked Applicable or N/A:reason — or the single line "N/A — no routing, shell, subprocess, VCS/PR, executable-classification, process-integration, Mnemonic-tool-contract, or shared-convention boundary."}

## Migration / Rollout
{Data migration, feature flags, phased rollout — or "No migration required."}

## Open Questions
- [ ] {unresolved technical question, if any}
```

### Step 5: Persist Artifact

**MANDATORY — do not skip.** Follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md). Hybrid = BOTH writes:

1. **Filesystem** — `docs/skillgrid/changes/<NNN-slug>/plan.md` (already written in Step 4).
2. **Mnemonic** — start one session, then save the same content:

```
sid = skillgrid-mnemonic_mem_session_start(title: "sdd/<NNN-slug>/plan")

skillgrid-mnemonic_mem_save(
  title:      "sdd/<NNN-slug>/plan",
  topic_key:  "sdd/<NNN-slug>/plan",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "{full plan markdown}"
)
```

`topic_key` upserts — re-running the phase replaces the observation in place, it does not duplicate. Mnemonic save notes: `title == topic_key` exactly; `scope: "project"`; pass the active `session_id`; there is **no** `project:` parameter and **no** `capture_prompt` field in the Mnemonic schema — omit both.

Do not branch on mode — `hybrid` is the only mode for this phase.

### Step 6: Self-Check (no external validator binary)

In place of an admission validator, before you return, confirm each:

1. Every `Impacted Files Map` row has a matching `code_read` in this session (or an explicit `code index unavailable` risk).
2. Every applicable threat-matrix row is assigned to the step that owns its RED test (Step 3). "A test for the edge case" is not an assignment — name the step.
3. Every step in the intent's Step Blueprint has a non-empty `## Step WHAT` block (Step 2). A step with no WHAT block is a handoff gap for `sdd-spec`.
4. Every `Architecture Decisions` block has all three of Choice / Alternatives / Rationale.
5. No `Mnemonic Integration` line claims a contract delta you did not also name in `_shared/conventions/…` (the plan is the *proposal* of the change; the convention file update is the *implementation* in a later phase — do not promise it here).
6. Word count of the artifact body is within the Size Budget below.

If any check fails, fix it before returning `success`. Return `partial` with the failed item described in `risks` if you cannot fix it now.

### Step 7: Return Envelope

**Your FINAL output MUST be text — not a tool call.** If you still need a `mem_save`, do it *before* this text (Step 5 already did). Returning text as your last action is what the orchestrator reads back; a trailing tool call buries the analysis in the tool result.

```markdown
## Plan Created
**Change**: {NNN-slug}
**Location**: `docs/skillgrid/changes/<NNN-slug>/plan.md` · Mnemonic `sdd/<NNN-slug>/plan` (hybrid)

**Status**: success | partial | blocked
**Executive summary**: 1–3 sentences.
**Key decisions**: {N documented, with one-line pointers}
**Files affected**: {A create / B modify / C delete}
**Step WHAT blocks**: {K steps, each covered} | {steps missing: …}
**Threat matrix**: {K applicable rows → owner step per row} | {not applicable}
**Open questions**: {list, or "None"}
**Skill resolution**: paths-injected | fallback-registry | none
**Risks**: {list, or "None"}
**Next**: sdd-tasks
```

Close your final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up (per `mnemonic-memory.md` § Session Close Protocol). Do not close with `mem_session_summary` in a sub-agent context — that's for top-level agents; the orchestrator owns session close.

## Rules

- Always read the actual code (via the code index) before planning — never from the intent's prose alone.
- Every decision MUST have a rationale (the "why").
- Every intent step MUST have a non-empty `## Step WHAT` block (Step 2). These blocks are the contract `sdd-spec` turns into Gherkin scenarios.
- Concrete file paths, not abstractions, in `## Impacted Files Map`.
- Every file row MUST be assigned to exactly one step — the `Step` column is the decomposition contract with `sdd-tasks`.
- Use the project's ACTUAL patterns — if the codebase does it differently from what you'd recommend, follow the existing pattern unless this change specifically addresses it (and say so in the decision rationale).
- Keep ASCII diagrams simple — clarity over beauty.
- Applicable threat-matrix rows are requirements: they MUST have a named owning step and propagate to `sdd-tasks` (RED-test task) and `sdd-spec` (scenario in that step's `acceptance.feature`). `N/A` rows MUST carry a reason.
- Apply any `rules.plan` from `docs/skillgrid/config.yaml`.
- If a question BLOCKS the plan, say so in `Open Questions` and return `partial` — a plan that guesses is not a plan.
- **Size budget**: the plan artifact body MUST be **under 850 words** (not counting fenced code diagrams). Architecture decisions as `Choice / Alternatives / Rationale` triples; code snippets only for non-obvious patterns.
- No external binaries. Mnemonic (`mem_*`) and code index (`code_*`) are the only knowledge sources. `gitnexus`, `gentle-ai`, `tree-sitter`, `npx <pkg>`, and similar are out of scope for this phase.
- Return envelope per the shape in Step 7 — final action is text, not a tool call.

## Gotchas

- `mem_search` returns **300-char previews**. Never use a preview as source material — `mem_get_observation(id)` is the only full-content path. Skipping this silently degrades the plan (a 300-char preview of a 2000-char intent loses most of it).
- The **Step WHAT block is a new obligation in v2 of this skill.** In the old model, the spec layer carried WHAT. There is no more spec layer — if you do not write per-step WHAT here, `sdd-spec` has nothing to turn into Gherkin and `sdd-apply` has nothing to test against.
- An applicability row marked `N/A` with a **vague** reason ("none", "out of scope") is a plan decision the reviewer cannot audit. Write the *boundary* you checked and why it is not in this change.
- **Mnemonic save rules**: `title == topic_key`, `scope: "project"`. No `project:` parameter, no `capture_prompt`. (See `conventions/mnemonic-memory.md` § Mnemonic Tool Mapping.)
- The registry lives at `docs/agents/skill-registry.md` (disk) *and* as a Mnemonic observation (`skill-registry`). Do not invent a third location.
- If the code index is `stale: true`, run `code_index` before `code_search`. Searching a stale index finds last-session's code, not this change's.
- **The `Impacted Files Map` Step column is the decomposition seam.** Every file MUST appear in exactly one step. A file with no step is a task with a hole; a file in two steps is a double-count in `sdd-apply`'s Work Unit Evidence.
- Do not commit from this phase. Plan is the *what* and *why*; `sdd-apply` commits the *do*. Commit conventions are `_shared/conventions/commits.md` and the close-token footer applies to the apply commit, not the plan.

## References

- [references/threat-matrix.md](references/threat-matrix.md) — applicability-driven threat matrix; load when the plan touches routing, shell/subprocess, VCS/PR automation, executable-file classification, process integration, **or** Mnemonic tool contracts, **or** any `_shared/conventions/*` file.
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape, session protocol, recovery ladder.
- [`../_shared/conventions/mnemonic-code-indexing.md`](../_shared/conventions/mnemonic-code-indexing.md) — the `code_status → code_index → code_search → code_read` ladder.
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — change-folder layout and phase order.
- [`../_shared/conventions/commits.md`](../_shared/conventions/commits.md) — commit contract (relevant to the downstream `sdd-apply` commit, not this phase).
- [`../sdd-propose/SKILL.md`](../sdd-propose/SKILL.md) — the upstream phase; its intent and Step Blueprint are this plan's contract.
- [`../sdd-tasks/SKILL.md`](../sdd-tasks/SKILL.md) — the downstream phase this plan feeds.
