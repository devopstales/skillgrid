---
name: sdd-propose
description: "Create the SDD change.md (WHY + HOW: Goal/DoD, Step Blueprint, architecture, threat matrix) from research, reserve NNN, instantiate template-change.md. Absorbs former sdd-design. Use after sdd-explore and before sdd-spec."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  author: devopstales
  version: "3.0"
  family: sdd
  part-of: skillgrid
  phase-order: "init → explore → propose → spec → apply → verify → archive"
  prev: [sdd-explore]
  next: [sdd-spec]
  artifact: change
  delegate_only: true
---

# sdd-propose

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-propose` sub-agent **unless** you loaded this skill directly through the `skill()` tool.

- **Sub-agent (primary)**: you were delegated here by the SDD orchestrator. Continue with the phase work below. Do not re-delegate. Do not call the `skill()` tool again.
- **Orchestrator (skill() loaded this directly)**: STOP. Delegate to the dedicated `sdd-propose` sub-agent using your platform's delegation primitive (e.g. `task(...)`) instead of doing the work inline.

## Purpose

You are the PROPOSE phase (v3): **WHY + HOW in one artifact**. You reserve `NNN`, write **`change.md`** by instantiating [`../_shared/templates/template-change.md`](../_shared/templates/template-change.md), and persist hybrid to Mnemonic `sdd/<NNN-slug>/change`.

This skill **absorbs former `sdd-design`**. Do not call `sdd-design` (retired stub).

Layout contract: [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md).

## What You Receive

- **Change slug** (kebab-case) — NNN may be supplied or reserved here.
- **Research** from `sdd-explore` OR direct user description.
- **Artifact store mode** is `hybrid` only: write `docs/skillgrid/changes/<NNN-slug>/change.md` **and** Mnemonic `sdd/<NNN-slug>/change`.
- Optional: ticket id; `## Skills to load before work`.
- `force_ticket_creation` true → invoke `issue-creation` for the `change.md` artifact.

## Conventions

- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md)
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — `rules.propose`
- [`../_shared/conventions/mnemonic-code-indexing.md`](../_shared/conventions/mnemonic-code-indexing.md)
- [`../_shared/templates/template-change.md`](../_shared/templates/template-change.md) — **mandatory outline**
- [`references/threat-matrix.md`](references/threat-matrix.md)

## Skill Loading

1. Injected skills block first, else recover: `sdd/<NNN-slug>/research`, `sdd-init/{project}`, `skill-registry` (fallback `docs/skillgrid/agents/skill-registry.md`).
2. Read `docs/skillgrid/config.yaml` → `rules.propose` (legacy alias: `rules.intent` / `rules.plan` if present).
3. Prior art: `docs/skillgrid/archive/<NNN-slug>/change.md` (legacy read-only: `plan.md` / `intent.md`).
4. Load `codebase-design` when restructuring modules.
5. Load `glossary` — fold terms into `change.md` `## Glossary`; upsert `docs/skillgrid/agents/glossary/{business,technical}.md`. **No companion `*-glossary-reference.md`.**

## What to Do

### Step 0: Classify (path ratchet)

- **Spike** → recommend `design-spike`, not a full change.
- **Bounded** → existing flow in repo; short design in chat; approval; single `01-` step OK.
- **Architectural** → full pipeline: explore → **propose** → **spec** → apply → verify → archive.

Announce classification. Every path ends with user approval of the change before apply.

### Step 0.5: Question round (interactive)

Business/product only (not harness). Prefer `questioning` skill. Cover problem, users, rules, outcome, gap, impact, edges, decision gaps, non-goals, risk.

### Step 1: Reserve NNN

Scan `docs/skillgrid/changes/`, `archive/`, Mnemonic `sdd/{project}/changelog`. `max+1`, zero-pad 3. Id = `<NNN>-<slug>`. Append changelog line.

### Step 2: Read code (code-index ladder)

`code_status` → `code_index` if stale → `code_search` → `code_read` for every symbol/module you will touch. Never plan from prose alone.

### Step 3: Write change.md from template

1. **READ** [`../_shared/templates/template-change.md`](../_shared/templates/template-change.md).
2. Copy outline verbatim; fill placeholders. Do not invent a parallel structure.
3. Write `docs/skillgrid/changes/<NNN-slug>/change.md`.
4. If file exists, READ then UPDATE.

Required content (template sections):

- STATUS, Goal / Architecture / Tech header
- Goal, Out of scope / Non-Goals, Definition of Done
- Problem, users, rules, in scope, risks & rollback
- Error handling, Testing strategy
- Step Blueprint (NN + primary package)
- Technical approach, Architecture decisions (Choice/Alternatives/Rationale + codebase-design vocabulary)
- Data flow, File layout (optional), Impacted files map (Step column)
- Per-step WHAT (Goal / Out of scope / DoD + WHAT bullets)
- Threat matrix ([references/threat-matrix.md](references/threat-matrix.md) — Applicable → owning step)
- Migration, open questions, Glossary footer, author self-review

**Threat rows:** Applicable → must propagate to `sdd-spec` as `[RED]` tasks + `@step-NN` scenarios.

### Step 4: Glossary close-term

Run glossary close-term check. Upsert glossary files. Fill `## Glossary` in `change.md`. No companion reference files.

### Step 5: Persist (hybrid, mandatory)

```
sid = mem_session_start(title: "sdd/<NNN-slug>/change")
mem_save(title/topic_key: "sdd/<NNN-slug>/change", type: architecture, scope: project, session_id: sid, content: full change.md)
# append sdd/{project}/changelog reservation line
```

File must exist on disk.

### Step 6: Return envelope

```markdown
## Change Proposed
**Change**: {NNN-slug}
**Location**: docs/skillgrid/changes/<NNN-slug>/change.md · Mnemonic sdd/<NNN-slug>/change
**Status**: success | partial | blocked
**Summary**: …
**Step blueprint**: {N steps}
**Threat rows**: {K applicable}
**Risk Level**: Low | Medium | High
**Next**: sdd-spec
```

## Rules

- ALWAYS instantiate `template-change.md`; ALWAYS reserve NNN before folder create.
- EVERY change has Goal, Out of scope/Non-Goals, Definition of Done, rollback, Testing strategy, Error handling.
- Step Blueprint is the contract with `sdd-spec` — never leave as empty placeholder.
- Apply `rules.propose` from config.yaml.
- No companion glossary files.
- Recovery: always `mem_get_observation` after `mem_search`.

## Gotchas

- Former `sdd-design` is retired — do not write `plan.md` or `intent.md`.
- Legacy archive may still have `intent.md`/`plan.md`; new work uses `change.md` only.
- Step Blueprint drives NN allocation in `sdd-spec`; leaving it blank is a handoff gap.

## References

- [`../_shared/templates/template-change.md`](../_shared/templates/template-change.md)
- [`references/threat-matrix.md`](references/threat-matrix.md)
- [`../sdd-spec/SKILL.md`](../sdd-spec/SKILL.md) — next
- [`../codebase-design/SKILL.md`](../codebase-design/SKILL.md)
- [`../glossary/SKILL.md`](../glossary/SKILL.md)
