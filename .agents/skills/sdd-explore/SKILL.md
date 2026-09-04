---
name: sdd-explore
description: "Investigate the codebase and compare approaches before committing to a change. Use when an SDD change needs upfront exploration, or when the orchestrator delegates discovery before proposing."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  author: devopstales
  version: "2.0"
  family: sdd
  part-of: skillgrid
  phase-order: "init → explore → propose → design → tasks → spec → apply → verify → archive"
  prev: [sdd-init]
  next: [sdd-propose]
  artifact: research
  delegate_only: true
---

# sdd-explore

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-explore` sub-agent **unless** you loaded this skill directly through the `skill()` tool.

- **Sub-agent (primary)**: you were delegated here by the SDD orchestrator. Continue with the phase work below. Do not re-delegate. Do not call the `skill()` tool again.
- **Orchestrator (skill() loaded this directly)**: STOP. Delegate to the dedicated `sdd-explore` sub-agent using your platform's delegation primitive (e.g. `task(...)`) instead of doing the work inline.

## Purpose

You are the EXPLORATION phase. Your job is to investigate the codebase, think through the problem, and compare approaches — **before** anyone proposes, plans, or writes code. You are read-only: you survey the current state, weigh alternatives, and return a structured analysis. You only write one artifact, `research.md`, when a change name is provided.

## What You Receive

From the orchestrator:

- **Topic / feature** to explore (a requirement, bug, or refactor question)
- **Change name** — the reserved `NNN-slug` (e.g. `001-oauth-login`) or a slug seed for a standalone exploration. The orchestrator reserves the number via `sdd-propose`'s numbering rule when the change is already in flight; for a fresh standalone exploration the change may not exist yet.
- **Artifact store mode** is `hybrid` — the only mode for this phase. When a change name is given, every run does BOTH: writes `docs/skillgrid/changes/<NNN-slug>/research.md` **and** persists to Mnemonic under `sdd/<NNN-slug>/research`. The filesystem write and the Mnemonic save are each their own obligations — the Mnemonic save does not stand in for the file.   Do not branch on the mode. (For a standalone exploration with no change name, return the analysis envelope and skip the artifact write, as before.)

## Skill Loading (Section A equivalent)

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first.
2. Otherwise check for `SKILL: Load` instructions in your launch prompt and load those exact paths.
3. Otherwise, load project context: read `docs/skillgrid/config.yaml` and the `docs/skillgrid/archive/NNN-slug/` tree if present, and run `mem_search(query: "sdd-init/{project}")` then `mem_get_observation(id)` to recover detected project facts (stack, testing, tracker).
4. If nothing is available, proceed with this skill alone plus the raw codebase.

## What to Do

### Step 1: Acquire Project Context

- Run `code_status` to check code-index health. If stale, run `code_index` before searching.
- Recover prior context: `mem_context` first, then `mem_search(query: "sdd/<NNN-slug>/")` to find any existing research output for this change, and `mem_get_observation(id)` for full content.
- Read `docs/skillgrid/config.yaml` if present — it carries detected tech stack, testing capabilities, and per-phase `rules`.
- Scan `docs/skillgrid/archive/NNN-slug/` for prior changes in the same domain — their `research.md` / `plan.md` are prior art, not a substitute for a fresh investigation.

### Step 2: Understand the Request

Parse what the user wants to explore:

- New feature? Bug fix? Refactor? Unknown requirement?
- Which domain/capability does it touch?
- Is the request specific enough, or do you need to surface a clarification?

If the request is too vague to explore, stop and state what clarification is needed.

### Step 3: Investigate the Codebase

Use the Mnemonic code-indexing ladder — **never grep the whole repo raw**:

```
code_status  ->  code_index (if stale)  ->  code_search  ->  code_read
```

- `code_search` for the relevant concepts (multiple queries in one call).
- `code_read` the matching slices — entry points, modules, tests, config.
- Look for: current architecture & patterns, affected files/modules, existing behavior, existing tests and their gaps, dependencies and coupling.

```
INVESTIGATE:
├── Read entry points and key files
├── Search for related functionality (code_search)
├── Check existing tests (if any)
├── Look for patterns already in use
└── Identify dependencies and coupling
```

### Step 4: Analyze Options

If multiple approaches exist, compare with a consistent table:

| Approach | Pros | Cons | Complexity |
|----------|------|------|------------|
| Option A | ... | ... | Low/Med/High |
| Option B | ... | ... | Low/Med/High |

Quantify effort (Low/Med/High) and name tradeoffs explicitly.

### Step 5: Persist Artifact (when a change name is provided)

This step is **MANDATORY** when tied to a named change — do not skip it.

**Filesystem path** (follow [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md)):

```
docs/skillgrid/changes/<NNN-slug>/research.md
```

- Ensure the change folder exists (created by `sdd-propose` or this phase for a fresh change). If `sdd-propose` already ran, the folder exists.
- If `research.md` already exists, READ it first and UPDATE it (do not overwrite blindly).
- Apply any `rules.explore` from `docs/skillgrid/config.yaml` if present.

**Mnemonic** (follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md)):

```
skillgrid-mnemonic_mem_save(
  title:        "sdd/<NNN-slug>/research",
  topic_key:    "sdd/<NNN-slug>/research",
  type:         "architecture",
  scope:        "project",
  session_id:   "{sid}",   // from skillgrid-mnemonic_mem_session_start
  content:      "{full markdown content}"
)
```

- Start a session once: `sid = skillgrid-mnemonic_mem_session_start(title: "sdd/<NNN-slug>/research")`.
- `topic_key` enables upsert — saving again updates in place; do not create near-duplicates.
- Hybrid is the only mode for this phase: do the filesystem write and the Mnemonic save. The file must actually exist on disk at `docs/skillgrid/changes/<NNN-slug>/research.md` — a Mnemonic save without the file is incomplete.

### Step 6: Return Structured Analysis

Return EXACTLY this format to the orchestrator (and write the same content to `research.md` if persisting):

```markdown
## Research: {topic}

### Current State
{How the system works today relevant to this topic}

### Affected Areas
- `path/to/file.ext` — {why it's affected}
- `path/to/other.ext` — {why it's affected}

### Approaches
1. **{Approach name}** — {brief description}
   - Pros: {list}
   - Cons: {list}
   - Effort: {Low/Medium/High}

2. **{Approach name}** — {brief description}
   - Pros: {list}
   - Cons: {list}
   - Effort: {Low/Medium/High}

### Recommendation
{Your recommended approach and why}

### Risks
- {Risk 1}
- {Risk 2}

### Ready for Proposal
{Yes/No — and what the orchestrator should tell the user}
```

## Return Envelope

Your FINAL output MUST be text (the envelope), not a trailing tool call. Do any `mem_save` calls first, then respond with text.

```markdown
**Status**: success | partial | blocked
**Summary**: 1-3 sentence summary of what was done
**Artifacts**: Mnemonic `sdd/<NNN-slug>/research` | `docs/skillgrid/changes/<NNN-slug>/research.md`
**Next**: sdd-propose (if ready) or sdd-propose-interactive / user-clarification
**Risks**: {risks discovered, or "None"}
```

## Rules

- The ONLY file you MAY create is `research.md` inside the change folder (if a change name is given). You may not modify existing code or files elsewhere.
- ALWAYS read real code — never guess about the codebase.
- Keep the analysis CONCISE; the orchestrator needs a summary, not a novel.
- If you can't find enough information, say so clearly.
- If the request is too vague, state what clarification is needed.
- Use the code-indexing ladder (`code_status` → `code_index` → `code_search` → `code_read`); do not raw-grep the whole repo.
- Recovery: `mem_search` returns previews only — always call `mem_get_observation(id)` for full content before relying on a prior artifact.
- At session end: call `mem_session_summary` then `mem_session_end`.

## Gotchas

- `mem_search` returns 300-char previews. Never use a preview as source material — always `mem_get_observation(id)` for full content. Skipping this produces wrong output.
- Mnemonic topic keys are namespaced per change: `sdd/<NNN-slug>/research`. Misspell the slug segment and later phases search into the void.
- Do not create the change directory with `mkdir -p docs/skillgrid/changes/...` blindly — first confirm the slug and check the change isn't already being continued. The NNN number is reserved by `sdd-propose`; you do not allocate it.
- The code index may be stale on a fresh checkout. If `code_status` reports stale, run `code_index` before `code_search` — an unindexed repo returns irrelevant or no results.
- Do not confuse this phase with proposal: you ANALYZE options here, you do not yet choose an approach as a commitment. Recommendation ≠ commitment.

## References

- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — change-folder layout, phase order, `research.md` placement.
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape, session protocol, recovery ladder.
- [`../_shared/conventions/mnemonic-code-indexing.md`](../_shared/conventions/mnemonic-code-indexing.md) — the `code_status → code_index → code_search → code_read` ladder.
- [`../sdd-propose/SKILL.md`](../sdd-propose/SKILL.md) — the downstream phase that consumes your `research.md`.
