---
name: sdd-propose
description: "Create an SDD change proposal with intent, scope, and approach from an exploration analysis. Use when the orchestrator needs a proposal.md before design/spec phases."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  author: devopstales
  part-of: Skillgrid
  version: "1.0"
  family: sdd
  phase-order: "init → explore → propose → design → spec → tasks → apply → verify → archive"
  prev-phase: [sdd-explore]
  next-phase: [sdd-design, sdd-spec]
  artifact: proposal
  delegate_only: true
---

# sdd-propose

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-propose` sub-agent **unless** you loaded this skill directly through the `skill()` tool.

- **Sub-agent (primary)**: you were delegated here by the SDD orchestrator. Continue with the phase work below. Do not re-delegate. Do not call the `skill()` tool again.
- **Orchestrator (skill() loaded this directly)**: STOP. Delegate to the dedicated `sdd-propose` sub-agent using your platform's delegation primitive (e.g. `task(...)`) instead of doing the work inline.

## Purpose

You are the PROPOSAL phase. Your job is to take the exploration analysis (from `sdd-explore`) — or direct user input — and produce a structured `proposal.md` document inside the change folder. You shape the scope, capabilities, approach, risks, and rollback before anyone designs or specs.

## What You Receive

From the orchestrator:

- **Change name** (kebab-case, e.g. `add-dark-mode`)
- **Exploration analysis** (from `sdd-explore`) OR a direct user description
- **Artifact store mode** is `hybrid` — preferred (degraded modes allowed, see `../_shared/conventions/hybrid-degradation.md`). Every run does BOTH: writes `docs/skillgrid/changes/{change-name}/proposal.md` **and** persists to Mnemonic under `sdd/{change-name}/proposal`. Any store-mode token from the orchestrator is honored as `hybrid` here. Degrade explicitly with a `Degraded:` envelope line instead of failing silently.

## Skill Loading

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first.
2. Otherwise, recover context: `mem_search(query: "sdd/{change-name}/explore")` + `mem_get_observation(id)`, then `mem_search(query: "sdd-init/{project}")` + `mem_get_observation(id)` for detected project facts (stack, testing, tracker).
3. Read `docs/skillgrid/config.yaml` if present — it carries `rules` (including `rules.proposal`).
4. Read the relevant existing specs from `docs/skillgrid/specs/{domain}/spec.md` when filling the **Capabilities** section — you need real capability names.

## What to Do

### Step 0: Shape the Proposal (interactive mode only)

In interactive SDD mode, do not let the executor silently decide if the input is "clear enough." Run a **proposal question round** before finalizing — focus on business/product/PRD, **not** harness mechanics (test commands, PR shape, line budgets):

1. **Business problem** — pain, opportunity, or cost that justifies this change now
2. **Target users & situations** — who is affected, in which workflow, urgency
3. **Business rules** — policies, permissions, thresholds, compliance/domain invariants
4. **Product outcome** — what should feel/work/possible after
5. **Current-state gap** — what is wrong, missing, or inconsistent today
6. **Implications & impact** — teams, data, UX, support/operational processes
7. **Edge cases** — empty states, partial data, failures, migrations, conflicting needs
8. **Decision gaps** — unknowns that make the proposal ambiguous or over-broad
9. **Scope boundaries & non-goals** — what's in the first slice vs deferred
10. **Business risk / tradeoff** — downside that matters if the direction is wrong

Prefer 3–5 concrete questions per round, each carrying a recommended default so answering is cheap. GATE each round: post the cluster, then stop and wait — never ask and answer in the same breath, never roll into the next phase. Skip categories with nothing genuinely open; never manufacture questions. After answers, summarize resulting assumptions labeled `Verified / Inferred / Unknown` (Unknown carries the next-step check: code_search query or file to read) and ask: *correct anything, or another round?* Reflect a thin answer back once as the concrete choice it leaves open — never upgrade vagueness into a confident plan. If the user declines ("just write it"): name the guesses, ask only the 2–3 highest-leverage questions; everything unanswered ships as `TBD — needs validation` / `Assumed — <assumption>, confirm before execution`, and affected scope carries a `GOTCHA`. Trivial changes skip the round — one line (`Classification: trivial`) suffices.

Question 4 (Product outcome) must yield the Hypothesis right/wrong block; question 9 (Scope boundaries) must yield the MVP + door check. No hypothesis ships without a wrong condition.

The reusable `questioning` skill implements the shared clarification primitive (classify + design tree, frontier, rounds, recommendations, approval gate); invoke it when you need a deeper requirement-stress session before writing the proposal. If you cannot ask the user directly, embed a `## Proposal question round` section in the result with the questions and assumptions needing review.

### Step 1: Acquire Context

- Recover any prior exploration: `mem_search(query: "sdd/{change-name}/explore")` → `mem_get_observation(id)` for full content. (Do not rely on search previews.)
- Read `docs/skillgrid/config.yaml` and `docs/skillgrid/specs/{domain}/spec.md` if they exist — needed for the **Capabilities** contract.
- Check the code index for affected modules if the exploration analysis is thin.

### Step 2: Create the Change Directory

Create the change folder (hybrid mode always writes the file):

```
docs/skillgrid/changes/{change-name}/
└── proposal.md
```

### Step 3: Write proposal.md

```markdown
# Proposal: {Change Title}

## Intent
{What problem are we solving? Why now? Be specific about the user need or tech debt.}

## Scope

### In Scope
- {Concrete deliverable 1}
- {Concrete deliverable 2}
- {Concrete deliverable 3}

### Out of Scope
- {What we are explicitly NOT doing}
- {Related future work, deferred}

## Capabilities
> CONTRACT with the sdd-spec phase: these names tell spec exactly which spec files to create or update. Research `docs/skillgrid/specs/` first.

### New Capabilities
<!-- Each becomes a new `docs/skillgrid/specs/<name>/spec.md`. Use kebab-case. Leave empty if none. -->
- `<capability-name>`: <brief description>

### Modified Capabilities
<!-- Existing capabilities whose REQUIREMENTS change. Use existing spec names. Leave empty if none. -->
- `<existing-capability-name>`: <what requirement is changing>

## Approach
{High-level technical approach. Reference the recommended approach from exploration if available. Intent only — never decide engineering here (stack/versions, data model, security boundaries, testing arch, error handling, project structure); flag any such call as deferred to design.}

## Hypothesis
{We believe [change] will cause [these users] to [do Y], resulting in [outcome]. RIGHT if [signal] within [timeframe]. WRONG if [counter-signal]. No hypothesis ships without a wrong condition. Trivial: one line.}

## MVP
{Thinnest end-to-end slice proving the hypothesis right or wrong.}

## Door check
{Two-way → build; one-way → spike first with decision rule.}

## Classification
{`trivial | standard | risky` per `../_shared/conventions/task-classification.md`. Optional for trivial — one line suffices.}

## Fast-Track Waiver
{Only for `trivial`/`small` per `../_shared/conventions/fast-track.md`: Class, Skips, Reason, Approved by. Omit this section for standard/risky changes — no waiver, full pipeline.}

## Verification floor
{`L1–L4` per `../_shared/conventions/verification-ladder.md` (trivial→L1 or omit, standard→L2, risky→L3/L4).}

## Trust Boundaries
{Actors, privileges, untrusted inputs, external systems, secrets, sensitive data — or one line: `No new trust boundary`. See `../_shared/conventions/trust-boundaries.md`.}

## Security Assumptions
{Stop-and-ask ambiguities (auth model, trust boundary, retention, secret handling) — or `None`. Never silently pick.}

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `path/to/area` | New/Modified/Removed | {What changes}

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| {Risk description} | Low/Med/High | {How we mitigate}

## Rollback Plan
{How to revert if something goes wrong. Be specific.}

## Dependencies
- {External dependency or prerequisite, if any}

## Success Criteria
- [ ] {How do we know this change succeeded?}
- [ ] {Measurable outcome}
```

If a `proposal.md` already exists in the change folder, READ it first and UPDATE it — do not overwrite blindly.

### Step 4: Persist Artifact

This step is **MANDATORY** — do not skip it.

**Filesystem path** (follow [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md)):

```
docs/skillgrid/changes/{change-name}/proposal.md
```

- Always create the change folder before writing.
- If the file already exists, READ then UPDATE (merge, preserve valid prior content).

**Mnemonic** (follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md)):

```
skillgrid-mnemonic_mem_save(
  title:        "sdd/{change-name}/proposal",
  topic_key:    "sdd/{change-name}/proposal",
  type:         "architecture",
  scope:        "project",
  session_id:   "{sid}",   // from skillgrid-mnemonic_mem_session_start
  content:      "{full markdown content}"
)
```

- Start a session once: `sid = skillgrid-mnemonic_mem_session_start(title: "sdd/{change-name}/proposal")`.
- `topic_key` enables upsert — saving again updates in place; do not create near-duplicates.
- Hybrid is preferred: do the filesystem write (Step 2/3) and the Mnemonic save; declare any degraded store in the envelope.

### Step 5: Return Summary

Return to the orchestrator:

```markdown
**Status**: success | partial | blocked
**Summary**: 1-2 sentence summary of the proposal
**Location**: `docs/skillgrid/changes/{change-name}/proposal.md` | Mnemonic `sdd/{change-name}/proposal`
**Intent**: {one-line intent}
**Scope**: {N in, M out}
**Approach**: {one-line approach}
**Hypothesis**: {right-signal / wrong-signal} · **MVP**: {thinnest slice} · **Door**: {two-way → build | one-way → spike}
**Classification**: {trivial | standard | risky} · **Verification floor**: {L1–L4 or omitted for trivial}
**Risk Level**: Low/Medium/High
**Assumptions (Inferred)**: {list, or "None"}
**Residual Risk**: {what remains unproven, blocked, or risky}
**Next**: sdd-design
```

## Rules

- ALWAYS create `proposal.md` (hybrid mode — preferred (degraded modes allowed, see `../_shared/conventions/hybrid-degradation.md`)).
- Every proposal MUST have a rollback plan.
- Every proposal MUST have success criteria.
- The **Capabilities** section is the contract with `sdd-spec` — always fill it. Research `docs/skillgrid/specs/` for real capability names. If nothing changes at the spec level, write "None" under both sub-sections — do not leave template placeholders.
- Use concrete file paths in **Affected Areas** when possible.
- Choose the narrowest viable change; tie-break lighter for local edits, stricter if blast radius or uncertainty is material (`../_shared/conventions/task-classification.md`). For `trivial`/`small`, record the `## Fast-Track Waiver` (see `../_shared/conventions/fast-track.md`) — downstream phases honor it, never re-derive it.
- Minimum secure design: no extra configurability, fallback paths, or optional insecure modes unless explicitly required (`../_shared/conventions/trust-boundaries.md`).
- Label assumptions `Verified / Inferred / Unknown`; never present inference as fact.
- Intent only in propose; engineering decisions belong to design/spec — flag divergence instead of silently deciding.
- Later phases inherit propose/design calls; if a phase must break one, flag it in Open Questions rather than silently diverging.
- Apply any `rules.proposal` from `docs/skillgrid/config.yaml`.
- **Size budget**: the proposal artifact MUST be under 450 words. Use bullets and tables over prose.
- Recovery: `mem_search` returns 300-char previews only — always `mem_get_observation(id)` for full content before relying on it.
- At session end: call `mem_session_summary` then `mem_session_end`.

## Gotchas

- `mem_search` returns 300-char previews. Never use a preview as source material — always call `mem_get_observation(id)` for full content. Skipping this produces wrong proposals.
- The **Capabilities** section is what drives `sdd-spec` file creation. Leaving it as a template placeholder causes spec files to be misnamed or missed entirely.
- "Out of Scope" is as important as "In Scope" — it prevents scope creep in later phases.
- In interactive mode, the question round must stay on business/product questions, not delivery mechanics. The user is the domain expert, not the delivery configurator.
- If a prior proposal exists and you UPDATE it, preserve any content the user hand-approved in earlier rounds — only revise the sections the new input affects.
