---
name: use-skillgrid
description: "Use when starting a conversation or beginning feature/change/bug/refactor work — routes Skillgrid SDD v4 (uninitialized → sdd-onboard/sdd-init; else optional explore/design-spike → propose → spec → user gate → apply ⇄ verify → archive). Triggers: skillgrid, SDD, start change, new feature, empty repo."
license: MIT
metadata:
  author: devopstales
  version: "2.0"
  family: sdd
  part-of: skillgrid
  role: orchestrator-entry
---

# use-skillgrid

Entry skill for Skillgrid SDD — counterpart to Superpowers' `using-superpowers`.
**Routes only.** Does not write `change.md` / `tasks.md` / product code.

Layout: [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md).
Phase order: `onboard → propose → spec → apply ⇄ verify → archive`.

<SUBAGENT-STOP>
If you were dispatched as a dedicated `sdd-*` sub-agent, ignore this skill and continue that phase.
</SUBAGENT-STOP>

## The Rule

**Before freestyle coding or inventing a change process**, invoke this skill when the request might be product/code change work.

1. Announce: `Using use-skillgrid to <route>`.
2. Run the checklist.
3. Load and follow the target skill exactly.

User instructions (`AGENTS.md`, “skip SDD”) override.

## Checklist

```
[ ] 1. Classify: change (feature|bug|refactor|app) | Q&A/lookup | spike-only
[ ] 2. Detect initialized? (config.yaml + AGENTS skillgrid sentinel)
[ ] 3. If NO  → sdd-onboard / sdd-init; stop until user validates
[ ] 4. If YES + change → optional explore / design-spike → sdd-propose (unless Resume)
[ ] 5. After sdd-spec → user gate (Implement | Revise) — never auto-apply
[ ] 6. Apply ⇄ verify (human QA findings re-enter apply) → sdd-archive
```

## Detection — initialized?

**Uninitialized** when **any** of:

1. No `docs/skillgrid/config.yaml`
2. No `<!-- skillgrid-sdd:start -->` … `<!-- skillgrid-sdd:end -->` in `AGENTS.md` / `CLAUDE.md` / `GEMINI.md`

Skill-registry / CONTEXT / CONSTRAINTS / `docs/adr/` are **not** init signals.

**Initialized** when `config.yaml` exists (prefer also the AGENTS block).

## Router

| Condition | First skill | Then |
|---|---|---|
| Uninitialized | `sdd-onboard` → `sdd-init` | After validate, if change stated → propose path |
| Initialized + change | optional `sdd-explore` / `design-spike` → `sdd-propose` | `sdd-spec` → gate → `sdd-apply` ⇄ `sdd-verify` → `sdd-archive` |
| Q&A / lookup | *(no pipeline)* | `mnemonic-memory` / code-index / `investigate` |
| Spike-only | `design-spike` | Promote to propose if user keeps findings |
| Mid-change | Resume from `tasks.md` `## State.phase` | verify findings may force apply |

```
use-skillgrid
    │
    ├─ Q&A → mem/code/investigate
    ├─ spike-only → design-spike
    ├─ uninitialized? → sdd-onboard → sdd-init → stop for validation
    ├─ change → [explore?] [design-spike?] → sdd-propose → sdd-spec → GATE
    │              ├─ Implement → apply ⇄ verify → archive
    │              └─ Revise → questioning / sdd-propose
    └─ resume State.phase
```

**Pre-propose gates:** hard research (external API, costly re-explore) → `sdd-explore`. Taste/UI/unknown shape → `design-spike` **before** locking `change.md`.

## User gate (mandatory)

After `sdd-spec` writes `tasks.md` + `acceptance.feature`:

1. **Implement** → `sdd-apply`
2. **Revise** → `questioning` and/or `sdd-propose`

Do not auto-apply.

## Resume

| `## State.phase` | Action |
|---|---|
| missing / onboard incomplete | `sdd-onboard` / `sdd-init` |
| propose / explore | `sdd-propose` (explore/spike first if still required) |
| spec | finish `sdd-spec` |
| apply | `sdd-apply` for unblocked work |
| verify | `sdd-verify` — human findings → set phase apply |
| archive | `sdd-archive` |

## Skill priority

1. `use-skillgrid` — route  
2. `sdd-*` workflow stage — execute  
3. General skills — as the stage loads them  

## Red flags

| Thought | Reality |
|---|---|
| "Repo has code, skip init" | Missing `config.yaml` ⇒ still onboard/init |
| "Jump to explore as first stage" | Explore is optional helper; prefer propose (pull explore when needed) |
| "Spec done, start coding" | User gate first |
| "Verify green → archive" | Need human QA accepted/waived too |
| "I'll invent a parallel process" | Route through this skill |

## References

- Plan: `docs/plan/01-workflow-new.md`
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md)
- Phase skills under `../sdd-*/SKILL.md`
