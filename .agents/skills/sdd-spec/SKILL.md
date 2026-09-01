---
name: sdd-spec
description: "Write SDD delta or full specs (requirements + Given/When/Then scenarios) from the proposal and the design. Use after sdd-design, before sdd-tasks. Maps each capability in the proposal's Capabilities contract to a spec file, copies the ENTIRE existing requirement before editing it (MODIFIED is replace semantics), and turns every applicable threat-matrix row in the design into a spec scenario. Uses Mnemonic memory + code index; no external binaries."
metadata:
  family: sdd
  phase-order: "propose → design → spec → tasks"
  prev: [sdd-propose, sdd-design]
  next: [sdd-tasks]
  artifact: spec
  delegate_only: true
---

# sdd-spec

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-spec` sub-agent **unless** you loaded this skill directly through the `skill()` tool.

- **Sub-agent (primary)**: you were delegated here by the SDD orchestrator. Continue with the phase work below. Do not re-delegate. Do not call the `skill()` tool again.
- **Orchestrator (skill() loaded this directly)**: STOP. Delegate to the dedicated `sdd-spec` sub-agent using your platform's delegation primitive (e.g. `task(...)`) instead of doing the work inline.

## Purpose

You are the SPEC phase. You express **what the system must do** — requirements with RFC 2119 strength (`MUST`/`SHALL`/`SHOULD`/`MAY`) and Given/When/Then scenarios — for every capability the proposal and design named. A spec is a **WHAT** document: `design.md` already holds the **HOW** (architecture, data flow, file changes). If you see yourself writing "the handler calls `validate()` on line 42", you are in design territory, not spec.

Phase order is `propose → design → spec → tasks`. Design runs **before** spec. Two consequences that drive this phase:

1. The design's **threat-matrix applicable rows are a spec input.** Every applicable row MUST appear as at least one scenario in some spec — that is where "a requirement" becomes "a testable requirement" one phase *earlier* than it would if you waited for `sdd-tasks`. A design carrying an applicable row that the spec does not reflect is a broken handoff; flag it in the envelope.
2. The proposal's **Capabilities section is the contract with this phase.** It mechanically names which spec files to create (New → full spec) or update (Modified → delta spec). Do not infer domains.

## What You Receive

From the orchestrator:

- **Change name** (kebab-case)
- **Artifact store mode** is `hybrid` — the only mode for this phase. Every run does BOTH: writes `specs/` files to `openspec/changes/{change-name}/` **and** persists to Mnemonic under `sdd/{change-name}/spec`. A mode token of `openspec` / `engram-compat` / `none` from the orchestrator is honored as `hybrid` here. Do not branch on the mode.
- Optional: **ticket/issue id** (carry-through to `sdd-apply`'s commit close-token per `_shared/conventions/commits.md`; spec itself does not use it)
- Optional: a `## Skills to load before work` block

## Execution + Persistence Conventions

Follow, on each save, rather than restating here:

- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — Mnemonic save shape (`title == topic_key`, `scope: "project"`, active `session_id`; **no** `project:` parameter, **no** `capture_prompt` field; `mem_search` returns previews — always `mem_get_observation(id)` for full content).
- [`../_shared/conventions/openspec.md`](../_shared/conventions/openspec.md) — change-folder layout, delta-spec section semantics, `rules.specs` from `openspec/config.yaml`.
- [`../_shared/conventions/mnemonic-code-indexing.md`](../_shared/conventions/mnemonic-code-indexing.md) — the `code_*` ladder, used only when you want to *verify* a scenario has code to test against (optional; a spec is a WHAT-document and does not require it).
- [`references/threat-matrix.md`](references/threat-matrix.md) — the boundary rows the design filled in; the applicable ones feed this phase's Required RED scenarios (local copy of `sdd-design`'s matrix for a self-contained skill).

## Skill Loading

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first.
2. Otherwise recover inputs from Mnemonic (previews are not enough — always fetch full content):
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/proposal")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; the **Capabilities** section is your primary contract.
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/design")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; the design's threat-matrix applicable rows feed this phase.
   - `skillgrid-mnemonic_mem_search(query: "sdd-init/{project}")` → `..._mem_get_observation(id)` — detected project facts (stack, testing, tracker).
3. Read `openspec/config.yaml` if present — `context:` and `rules.specs` bind this phase.
4. For every **Modified** capability, read the existing main spec `openspec/specs/{domain}/spec.md` — you cannot write a correct MODIFIED block against a requirement you have not read. For every **New** capability, confirm that file does NOT yet exist (a "New" capability whose spec already exists is a proposal bug — flag in `risks`).

## What to Do

### Step 1: Map Capabilities → Spec Files

Mechanically, no inference. From the proposal's **Capabilities** section:

```
FOR EACH entry under "New Capabilities":
  → write openspec/changes/{change-name}/specs/<capability>/spec.md
    AS A FULL SPEC (## Purpose + ## Requirements), not a delta.
    Reason: there is no existing behavior to be a delta against.

FOR EACH entry under "Modified Capabilities":
  → read openspec/specs/<capability>/spec.md  (REQUIRED)
  → write openspec/changes/{change-name}/specs/<capability>/spec.md
    AS A DELTA SPEC (## ADDED / ## MODIFIED / ## REMOVED / ## RENAMED).
```

Write **New** specs before **Modified** ones so ADDED blocks are unambiguous. If the proposal has **no** Capabilities section (older format), fall back to inferring from the proposal's **Affected Areas** — and note in the envelope `risks` that `sdd-propose` should have filled Capabilities.

### Step 2: Write the Specs

**Full spec** (New capability) and **delta spec** (Modified capability) formats are in [`references/delta-spec-format.md`](references/delta-spec-format.md). The one rule that matters most:

> **MODIFIED is REPLACE semantics, not PATCH semantics.**
> Copy the **ENTIRE** existing requirement block — name, body, and **every scenario** — from `openspec/specs/{domain}/spec.md`, paste it under `## MODIFIED Requirements`, then edit the copy. `sdd-archive` replaces the main-spec requirement with your MODIFIED block byte-for-byte; any scenario you did not copy is **gone** the moment archive runs.

Requirements carry RFC 2119 keywords. Scenarios are Given/When/Then. Every requirement has at least one scenario, covering a happy path **and** an edge case (or failure state). Keep scenarios testable — someone should be able to write an automated test directly from one.

### Step 3: Carry the Design's Applicable Threat Rows

Read the design's `## Threat Matrix`. For **each row marked `Applicable`**, ensure at least one scenario in the spec set covers it. This is the point of design-before-spec: the design's "required RED tests" become concrete GIVEN/WHEN/THEN here, not later in `sdd-tasks`.

- If a spec covers a row, that is enough — do not force a dedicated `## Required RED tests` section.
- If a design-applicable row has no covering scenario, add one under the domain it belongs to, or flag the gap in the envelope `risks` (do not silently drop it).
- `N/A` rows need no scenario and no action.

### Step 4: REMOVED and RENAMED Discipline

- **REMOVED**: every block MUST include `(Reason: …)`. Include `(Migration: …)` (or `Migration: None`) when consumers, persisted data, docs, or tests are affected.
- **RENAMED**: state both names in the heading — `### Requirement: {old} → {new}` — and include a `(Migration: …)` for references, tests, and docs that still point at the old name.
- Adding **new** behavior alongside existing behavior → use `## ADDED`, not `## MODIFIED`.

### Step 5: Persist Artifact

**MANDATORY — do not skip.** Follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md). Hybrid = BOTH writes:

1. **Filesystem** — write each domain file `openspec/changes/{change-name}/specs/{domain}/spec.md`.
2. **Mnemonic** — start one session, then one save per change (concatenate domains so a single `mem_get_observation` id retrieves the whole spec):

  ```
  sid = skillgrid-mnemonic_mem_session_start(title: "sdd/{change-name}/spec")

  skillgrid-mnemonic_mem_save(
    title:      "sdd/{change-name}/spec",
    topic_key:  "sdd/{change-name}/spec",
    type:       "architecture",
    scope:      "project",
    session_id: "{sid}",
    content:    "## {domain-1}\n\n…full spec…\n\n## {domain-2}\n\n…full spec…"
  )
  ```

  One observation per change (concatenated with `## {domain}` headers) keeps the pipeline consistent with `sdd/{change}/proposal` and `sdd/{change}/design`. `topic_key` upserts — re-running the phase replaces the observation in place. Mnemonic save notes: `title == topic_key` exactly; `scope: "project"`; pass the active `session_id`; there is **no** `project:` parameter and **no** `capture_prompt` field in the Mnemonic schema — omit both.

Do not branch on mode — `hybrid` is the only mode for this phase.

### Step 6: Self-Check (no external validator binary)

Before returning, confirm each — fix any failure before returning `success`, otherwise return `partial` with the failed item in `risks`:

1. Every requirement has ≥ 1 scenario.
2. Every RFC 2119 keyword in a requirement body is **uppercase** (`MUST`, `SHALL`, `SHOULD`, `MAY` — lowercase `must`/`should` is not a keyword).
3. Every `## MODIFIED` block contains **all** scenarios from the existing main spec that the delta does not explicitly remove.
4. Every `## REMOVED` block has a `(Reason: …)`.
5. Every `## RENAMED` heading states `{old} → {new}` explicitly.
6. Every applicable design threat-row has a covering scenario (Step 3).
7. A "New" capability has no pre-existing main spec; a "Modified" capability's main spec exists and was read.
8. Spec body is within the size budget.

### Step 7: Return Envelope

**Your FINAL output MUST be text — not a tool call.** If you still need a `mem_save`, do it *before* this text (Step 5 already did). A trailing tool call buries the analysis in the tool result; returning text is what the orchestrator reads back.

```markdown
## Specs Created
**Change**: {change-name}
**Status**: success | partial | blocked
**Executive summary**: 1–3 sentences.

**Specs written**
| Domain | Type | Requirements (A/M/R) | Scenarios |
|--------|------|----------------------|-----------|
| {domain} | New | {n} | {n} |
| {domain} | Delta | {a}/{m}/{r} | {n} |

**Coverage**: happy {covered|missing} · edge {covered|missing} · error {covered|missing}
**Threat-matrix handoff**: {K applicable rows, all covered} | {list the gaps} | {not applicable}
**Open questions**: {list, or "None"}
**Skill resolution**: paths-injected | fallback-registry | none
**Risks**: {list, or "None"}
**Next**: sdd-tasks
```

Close the final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up (per `mnemonic-memory.md` § Session Close Protocol). Do not call `mem_session_summary` here — that is a top-level-agent concern; the orchestrator owns session close.

## Rules

- Use Given/When/Then for every scenario; RFC 2119 for every requirement.
- **NEW capability → full spec; MODIFIED capability → delta spec.** No exceptions.
- **MODIFIED = copy the ENTIRE requirement block, then edit.** A partial MODIFIED block loses scenarios at archive time — this is the number-one spec defect.
- Every requirement ≥ 1 scenario, covering a happy path and an edge case.
- Spec is WHAT, not HOW — no file paths, line numbers, or function names in requirement text.
- Every applicable design threat-row maps to a spec scenario (Step 3).
- REMOVED requires `(Reason: …)`; RENAMED requires the `{old} → {new}` heading.
- Apply any `rules.specs` from `openspec/config.yaml`.
- **Size budget**: spec artifact body **under 650 words** (not counting scenario bodies). Prefer requirement tables over narrative. A scenario is 3–5 lines max.
- No external binaries. Mnemonic (`mem_*`) and, if you choose, the code index (`code_*`) are the only knowledge sources. No `openspec-cli`, no `gentle-ai` spec validator, no grammar binary.
- Return envelope per Step 7 — final action is text, not a tool call.

## Gotchas

- `mem_search` returns **300-char previews.** A preview of a 2000-char design or proposal loses most of it — always `mem_get_observation(id)`.
- Writing a MODIFIED block with "the changed scenarios only" is the classic trap. `sdd-archive` does a replacement, not a merge — un-copied scenarios vanish on archive.
- **Adding behavior without changing existing behavior → `## ADDED`, not `## MODIFIED`.** Reach for MODIFIED and you are likely to drop scenarios.
- A "New" capability whose main spec already exists (or a "Modified" capability with none) is a proposal bug, not a spec bug — flag it in `risks` and let `sdd-propose` fix the Capabilities list.
- **Mnemonic ≠ Engram.** No `project:` parameter, no `capture_prompt`. `title == topic_key`, `scope: "project"`, active `session_id`. (See `conventions/mnemonic-memory.md` § Mnemonic Tool Mapping.)
- The Mnemonic artifact is **one observation per change** (`sdd/{change}/spec`, domains concatenated) — consistent with the proposal and design saves. Do not split it into one observation per domain; recovery is one `mem_get_observation` id.
- Design-before-spec means the design is already here. If `design.md` is missing, `sdd-design` is the phase to run next — do not write specs from a proposal alone, or you lose the threat-row handoff (Step 3).
- Do not commit from this phase. Spec is WHAT; `sdd-apply` commits the DO.

## References

- [references/delta-spec-format.md](references/delta-spec-format.md) — the ADDED/MODIFIED/REMOVED/RENAMED shape, the full-spec shape, the copy-full-then-edit workflow, and the RFC 2119 quick reference.
- [`../sdd-design/SKILL.md`](../sdd-design/SKILL.md) — the upstream phase; its `## Threat Matrix` applicable rows feed Step 3.
- [`../sdd-propose/SKILL.md`](../sdd-propose/SKILL.md) — the proposal's Capabilities section is the contract this phase maps against.
- [`references/threat-matrix.md`](references/threat-matrix.md) — the boundary rows the design may have marked applicable (local copy of `sdd-design`'s matrix).
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape, session protocol, recovery ladder.
- [`../_shared/conventions/openspec.md`](../_shared/conventions/openspec.md) — change-folder layout and delta-spec section semantics.
