---
name: sdd-design
description: "Create the SDD technical design (architecture, data flow, file changes, testing strategy) from a proposal, before spec. Use when the orchestrator needs a design.md between sdd-propose and sdd-spec. Reads the proposal from Mnemonic, reads the actual code via the Mnemonic code index, defends each decision with a rationale, and fills an applicability-driven threat matrix. No external binaries — Mnemonic memory + code index only."
metadata:
  family: sdd
  phase-order: "propose → design → spec → tasks"
  prev: [sdd-propose]
  next: [sdd-spec]
  artifact: design
  delegate_only: true
---

# sdd-design

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-design` sub-agent **unless** you loaded this skill directly through the `skill()` tool.

- **Sub-agent (primary)**: you were delegated here by the SDD orchestrator. Continue with the phase work below. Do not re-delegate. Do not call the `skill()` tool again.
- **Orchestrator (skill() loaded this directly)**: STOP. Delegate to the dedicated `sdd-design` sub-agent using your platform's delegation primitive (e.g. `task(...)`) instead of doing the work inline.

## Purpose

You are the DESIGN phase. You take the proposal (and the spec if it already exists) and produce a concrete `design.md` that captures **HOW** the change will be implemented: architecture decisions, data flow, file changes, interfaces, and the testing strategy. You design the *shape* so `sdd-spec` can express it as requirements and `sdd-tasks` can break it into work.

Phase order is `propose → design → spec → tasks` — design runs **before** spec. The delta spec is expressed relative to what the design decided to build; if you skip or weaken the design, spec has nothing concrete to be a delta *against*.

## What You Receive

From the orchestrator:

- **Change name** (kebab-case, e.g. `add-dark-mode`)
- **Artifact store mode** is `hybrid` — the only mode for this phase. Every run does BOTH: writes `openspec/changes/{change-name}/design.md` **and** persists to Mnemonic under `sdd/{change-name}/design`. A mode token of `openspec` / `engram-compat` / `none` from the orchestrator is honored as `hybrid` here. Do not branch on the mode.
- Optional: **ticket/issue id** (for the eventual `sdd-apply` commit close-token per `_shared/conventions/commits.md`)
- Optional: a `## Skills to load before work` block

## Execution + Persistence Conventions

This skill does not restate them. Follow, on each save:

- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — Mnemonic save shape (`title == topic_key`, `scope: "project"`, active `session_id`; `mem_search` returns **previews only** — always `mem_get_observation(id)` for full content).
- [`../_shared/conventions/openspec.md`](../_shared/conventions/openspec.md) — change-folder layout, write-before-reuse rule, `rules.design` from `openspec/config.yaml`.
- [`../_shared/conventions/mnemonic-code-indexing.md`](../_shared/conventions/mnemonic-code-indexing.md) — the `code_status → code_index → code_search → code_read` ladder you use to read the codebase. **No external binaries**: `gitnexus`, `gentle-ai`, `tree-sitter-cli`, and friends are not part of this phase.

## Skill Loading

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first.
2. Otherwise recover context from Mnemonic (previews are not enough — always fetch full content):
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/proposal")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required input**.
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/spec")` → `..._mem_get_observation(id)` — **optional**; the spec may not exist yet if design runs first.
   - `skillgrid-mnemonic_mem_search(query: "sdd-init/{project}")` → `..._mem_get_observation(id)` — detected project facts (stack, testing, tracker).
   - `skillgrid-mnemonic_mem_search(query: "skill-registry")` → `..._mem_get_observation(id)` — the skill index; if it is missing, fall back to the disk file `docs/agents/skill-registry.md` at the repo root.
3. Read `openspec/config.yaml` if present — `context:` and `rules.design` bind this phase.
4. Read the relevant existing specs from `openspec/specs/{domain}/spec.md` where the design must not contradict them — the design is a *delta plan*, not a greenfield.

## What to Do

### Step 1: Read the Code You Will Change (via the code index)

Never design from memory or from the proposal's prose alone. For each symbol, module, or integration point the proposal names:

```
skillgrid-mnemonic_code_status              # stale? file_count 0?
  → if stale, skillgrid-mnemonic_code_index
skillgrid-mnemonic_code_search(query: "<symbol-or-phrase>", limit: 20)
  → for each hit you actually will touch,
skillgrid-mnemonic_code_read(path: <hit.path>, start_line: <hit.start_line>, end_line: <hit.end_line>)
```

Rules (from `conventions/mnemonic-code-indexing.md`):

- Search first, then read the matching slice. **Never read a whole file speculatively** — a design that cites a file it never read is a design with a hole.
- Prefer the code index over `rg`/`grep` when exploring unfamiliar areas; use `rg`/`grep` for exact-identifier lookups the index is a poor fit for.
- If the index is empty and `code_index` cannot run (e.g. no MCP and no git repo), fall back to reading the files directly — and say so in the envelope's `risks`.

Confirm before designing: entry points, module boundaries, existing patterns, dependency/interface contracts, and the test infrastructure the new work will plug into.

### Step 2: Applicability-Driven Threat Matrix

Read the applicability trigger list in [references/threat-matrix.md](references/threat-matrix.md). It lists the boundaries where a design can silently break (routing, shell/subprocess, VCS/PR automation, executable-file classification, process integration — **plus** skillgrid-specific rows: a new/modified Mnemonic tool contract, and a change to any `_shared/conventions/*` file).

- If **any** row is applicable, include the full matrix in `design.md`, mark each row `Applicable` or explicit `N/A: reason`, and name the expected safe behavior, failure behavior, and the planned RED test for every applicable row.
- If **none** apply (a pure ref, doc, or additive-feature change away from every listed boundary), record the matrix as not applicable in one line — do not manufacture `N/A` rows.

**Applicable rows are design requirements.** They MUST propagate unchanged **into `sdd-spec` first** — each becomes a Given/When/Then scenario (or a `## Required RED tests` entry) in the delta spec — and only then into `sdd-tasks` as RED tests before any production code. The spec is the enforcement boundary: a design row that the spec drops is a handoff gap. An explicit `N/A` row requires no scenario and no task, but must carry a reason a reviewer can challenge.

### Step 3: Write design.md

Create the file in the change folder (hybrid mode always writes it):

```
openspec/changes/{change-name}/
├── proposal.md
├── design.md              ← you create this
└── (specs/ comes later, from sdd-spec)
```

If a `design.md` already exists in the change folder, **READ it first and UPDATE it** — do not overwrite a prior design's valid content.

#### Design document format

```markdown
# Design: {Change Title}

## Technical Approach
{Concise overall strategy. How it maps to the proposal's approach. Reference the specs it must not contradict.}

## Architecture Decisions

### Decision: {title}
**Choice**: {what we chose}
**Alternatives considered**: {what we rejected, briefly}
**Rationale**: {why this over the others}

## Data Flow
{How data moves through the system for this change. ASCII diagram when it helps:}

    Component A ──▶ Component B ──▶ Component C
         │                              │
         └──────── Store ───────────────┘

## File Changes
| File | Action | Description |
|------|--------|-------------|
| `path/to/new-file` | Create | {what it does} |
| `path/to/existing` | Modify | {what changes and why} |
| `path/to/old` | Delete | {why} |

## Interfaces / Contracts
{New interfaces, API contracts, types, data structures. Code blocks in the project's language, only where non-obvious.}

## Mnemonic Integration
{Only when relevant. New `sdd/{change}/…` topic keys this design anticipates; any change to existing Mnemonic contracts (a new `mem_save` shape, a `code_search` caller, a new code-index expectation); any new tool surface.}
{If the design does NOT touch the memory surface, write "No Mnemonic contract changes." on one line.}

## Testing Strategy
| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | {what} | {how} |
| Integration | {what} | {how} |
| E2E | {what} | {how} |

## Threat Matrix
{Applicability matrix from references/threat-matrix.md — every row marked Applicable or N/A:reason — or the single line "N/A — no routing, shell, subprocess, VCS/PR, executable-classification, process-integration, Mnemonic-tool-contract, or shared-convention boundary."}

## Migration / Rollout
{Data migration, feature flags, phased rollout — or "No migration required."}

## Open Questions
- [ ] {unresolved technical question, if any}
```

### Step 4: Persist Artifact

**MANDATORY — do not skip.** Follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md). Hybrid = BOTH writes:

1. **Filesystem** — `openspec/changes/{change-name}/design.md` (already written in Step 3).
2. **Mnemonic** — start one session, then save the same content:

  ```
  sid = skillgrid-mnemonic_mem_session_start(title: "sdd/{change-name}/design")

  skillgrid-mnemonic_mem_save(
    title:      "sdd/{change-name}/design",
    topic_key:  "sdd/{change-name}/design",
    type:       "architecture",
    scope:      "project",
    session_id: "{sid}",
    content:    "{full design markdown}"
  )
  ```

  `topic_key` upserts — re-running the phase replaces the observation in place, it does not duplicate. Mnemonic save notes: `title == topic_key` exactly; `scope: "project"`; pass the active `session_id`; there is **no** `project:` parameter and **no** `capture_prompt` field in the Mnemonic schema — omit both.

Do not branch on mode — `hybrid` is the only mode for this phase.

Mnemonic save notes (from `mnemonic-memory.md`): `title == topic_key` exactly; `scope: "project"`; pass the active `session_id`; there is **no** `project:` parameter and **no** `capture_prompt` field in the Mnemonic schema — omit both.

### Step 5: Self-Check (no external validator binary)

In place of an admission validator, before you return, confirm each:

1. Every `File Changes` row has a matching `code_read` in this session (or an explicit `code index unavailable` risk).
2. Every applicable threat-matrix row has a `Planned RED test` concrete enough for `sdd-spec` to turn into a scenario (and `sdd-tasks` into a task). "A test for the edge case" is not a test.
3. Every `Architecture Decisions` block has all three of Choice / Alternatives / Rationale.
4. No `Mnemonic Integration` line claims a contract delta you did not also name in `_shared/conventions/…` (the design is the *proposal* of the change; the convention file update is the *implementation* in a later phase — do not promise it here).
5. Word count of the artifact body is within the Size Budget below.

If any check fails, fix it before returning `success`. Return `partial` with the failed item described in `risks` if you cannot fix it now.

### Step 6: Return Envelope

**Your FINAL output MUST be text — not a tool call.** If you still need a `mem_save`, do it *before* this text (Step 4 already did). Returning text as your last action is what the orchestrator reads back; a trailing tool call buries the analysis in the tool result.

```markdown
## Design Created
**Change**: {change-name}
**Location**: `openspec/changes/{change-name}/design.md` · Mnemonic `sdd/{change-name}/design` (hybrid)

**Status**: success | partial | blocked
**Executive summary**: 1–3 sentences.
**Key decisions**: {N documented, with one-line pointers}
**Files affected**: {A create / B modify / C delete}
**Threat matrix**: {K applicable rows → required RED tests} | {not applicable}
**Testing strategy**: {unit/integration/e2e one-liner}
**Mnemonic**: observation `{id or 'none in mode X'}` · session `{sid}`
**Open questions**: {list, or "None"}
**Skill resolution**: paths-injected | fallback-registry | none
**Risks**: {list, or "None"}
**Next**: sdd-spec
```

Close your final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up (per `mnemonic-memory.md` § Session Close Protocol). Do not close with `mem_session_summary` in a sub-agent context — that's for top-level agents; the orchestrator owns session close.

## Rules

- Always read the actual code (via the code index) before designing — never from the proposal's prose alone.
- Every decision MUST have a rationale (the "why").
- Concrete file paths, not abstractions, in `## File Changes`.
- Use the project's ACTUAL patterns — if the codebase does it differently from what you'd recommend, follow the existing pattern unless this change specifically addresses it (and say so in the decision rationale).
- Keep ASCII diagrams simple — clarity over beauty.
- Applicable threat-matrix rows are requirements: they MUST propagate unchanged into `sdd-spec` (as scenarios) and then `sdd-tasks` (as RED tests). `N/A` rows MUST carry a reason.
- Apply any `rules.design` from `openspec/config.yaml`.
- If a question BLOCKS the design, say so in `Open Questions` and return `partial` — a design that guesses is not a design.
- **Size budget**: the design artifact body MUST be **under 800 words** (not counting fenced code diagrams). Architecture decisions as `Choice / Alternatives / Rationale` triples; code snippets only for non-obvious patterns.
- No external binaries. Mnemonic (`mem_*`) and code index (`code_*`) are the only knowledge sources. `gitnexus`, `gentle-ai`, `tree-sitter`, `npx <pkg>`, and similar are out of scope for this phase.
- Return envelope per the shape in Step 6 — final action is text, not a tool call.

## Gotchas

- `mem_search` returns **300-char previews**. Never use a preview as source material — `mem_get_observation(id)` is the only full-content path. Skipping this silently degrades the design (a 300-char preview of a 2000-char proposal loses most of it).
- An applicability row marked `N/A` with a **vague** reason ("none", "out of scope") is a design decision the reviewer cannot audit. Write the *boundary* you checked and why it is not in this change.
- "Applicable rows must propagate unchanged" is the whole point of the matrix — through `sdd-spec` to `sdd-tasks`. If a `sdd-tasks` run re-derives those RED tests from prose instead of inheriting the spec's scenarios, the design was carrying them, not the spec.
- **Mnemonic ≠ Engram.** No `project:` parameter, no `capture_prompt`. `title == topic_key`. `scope: "project"`. (See `conventions/mnemonic-memory.md` § Mnemonic Tool Mapping.)
- The registry lives at `docs/agents/skill-registry.md` (disk) *and* as a Mnemonic observation (`skill-registry`). Do not invent a third location (`.atl/skill-registry.md`, `~/.skillgrid/registry.md`, etc.).
- If the code index is `stale: true`, run `code_index` before `code_search`. Searching a stale index finds last-session's code, not this change's.
- **Design-before-spec**: if `proposal.md` exists but `design.md` does not, this skill runs. If `design.md` exists but `spec.md` does not, spec runs next. Do not write `specs/` here — that is `sdd-spec`'s job, and it consumes this `design.md`.
- Do not commit from this phase. Design is the *what* and *why*; `sdd-apply` commits the *do*. Commit conventions are `_shared/conventions/commits.md` and the close-token footer applies to the apply commit, not the design.

## References

- [references/threat-matrix.md](references/threat-matrix.md) — applicability-driven threat matrix; load when the design touches routing, shell/subprocess, VCS/PR automation, executable-file classification, process integration, **or** Mnemonic tool contracts, **or** any `_shared/conventions/*` file.
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape, session protocol, recovery ladder.
- [`../_shared/conventions/mnemonic-code-indexing.md`](../_shared/conventions/mnemonic-code-indexing.md) — the `code_status → code_index → code_search → code_read` ladder.
- [`../_shared/conventions/openspec.md`](../_shared/conventions/openspec.md) — change-folder layout and phase order (`propose → design → spec → tasks`).
- [`../_shared/conventions/commits.md`](../_shared/conventions/commits.md) — commit contract (relevant to the downstream `sdd-apply` commit, not this phase).
- [`../sdd-propose/SKILL.md`](../sdd-propose/SKILL.md) — the upstream phase; read for the shape of the proposal this design is answering.
