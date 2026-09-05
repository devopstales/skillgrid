---
name: sdd-propose
description: Reserve NNN and write change.md (WHY + HOW, Architecture decisions, Research:/Prototype: links) from template-change.md; stop before code. Use when starting an SDD change, after optional sdd-explore / design-spike, or when use-skillgrid routes to propose.
disable-model-invocation: true
license: MIT
metadata:
  author: devopstales
  version: "4.0"
  part-of: skillgrid
---

# SDD Propose

Stage owner (v4). Reserve `NNN`, write **`change.md`**, stop before code. Architecture decisions live **in** `change.md` — promote to ADR later only when they outlive this change.

Layout: [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md).

## Hard Rules

- Instantiate [`../_shared/templates/template-change.md`](../_shared/templates/template-change.md) — do not invent a parallel structure.
- Reserve NNN **before** creating the change folder.
- If hard research or taste/UI/unknown API shape remains → **STOP** and require `sdd-explore` / `design-spike` first.
- Run **`questioning`** after concrete inputs (research/spike when used) — not before empty air.
- List `Research:` and `Prototype:` header paths when those artifacts exist.
- No production code. No default ADR write under `docs/adr/`.
- Hybrid: disk + Mnemonic `sdd/<NNN-slug>/change`.
- `force_ticket_creation` → invoke `issue-creation` for the change artifact.

## Workflow

```
[ ] 1. Gate: explore / spike required?
[ ] 2. Reserve NNN
[ ] 3. Questioning (after concrete inputs)
[ ] 4. Read code (code-index ladder)
[ ] 5. Write change.md from template
[ ] 6. Glossary + persist + envelope
```

### 1. Gate — explore / spike first?

| Signal | Action |
|---|---|
| External/rare docs, costly rediscovery, missing/stale research | STOP → **`sdd-explore`** → `research.md` |
| UI taste, unknown API shape, throwaway smoke needed | STOP → **`design-spike`** → list path as `Prototype:` |
| Bounded in-repo change with clear shape | Continue |

Announce the gate decision. Do not lock `change.md` while the gate is open.

### 2. Reserve NNN

Scan `docs/skillgrid/changes/`, `archive/`, Mnemonic `sdd/{project}/changelog`. Next = `max+1`, zero-pad 3. Id = `<NNN>-<slug>`. Append changelog line. Never reuse.

### 3. Questioning

Load **`questioning`**. Cover problem, users, rules, outcome, non-goals, edges, risks. Optional log: `interview.md`. Prefer revise later via user gate over guessing.

### 4. Read code

`code_status` → `code_index` if stale → `code_search` → `code_read` for every module you will touch. Load `codebase-design` when restructuring. Apply `rules.propose` from `config.yaml`.

### 5. Write change.md

1. READ the template; copy outline; fill placeholders.
2. Write `docs/skillgrid/changes/<NNN-slug>/change.md` (READ then UPDATE if exists).
3. Header: set **`Research:`** and **`Prototype:`** when present (else `none`).
4. Required: Goal, Out of scope/Non-Goals, Definition of Done, Problem, Testing strategy, Error handling, rollback, **Step Blueprint**, Technical approach, **Architecture decisions** (Choice / Alternatives / Rationale), Impacted files, per-step WHAT, **Threat matrix** ([references/threat-matrix.md](references/threat-matrix.md) — Applicable → owning step), Glossary footer.

Threat Applicable rows must propagate to `sdd-spec` as `[RED]` tasks.

### 6. Glossary + persist + envelope

- Fold terms into `## Glossary`; upsert `docs/skillgrid/glossary/{business,technical}.md` via **`glossary`**. No companion `*-glossary-reference.md`.
- `mem_session_start` → `mem_save` topic `sdd/<NNN-slug>/change` (full file). File must exist on disk.

```markdown
## Change Proposed
**Change**: {NNN-slug}
**Location**: docs/skillgrid/changes/<NNN-slug>/change.md
**Research / Prototype**: {paths or none}
**Status**: success | partial | blocked
**Step blueprint**: {N} · **Threat rows**: {K applicable}
**Next**: sdd-spec
```

## Gotchas

- Skipping explore/spike when the gate fires produces hollow `change.md` — stop early.
- Architecture in `change.md` only; archive does not auto-promote ADRs.
- Empty Step Blueprint is a handoff gap for `sdd-spec`.
- Former `sdd-design` / `intent.md` / `plan.md` are retired — new work is `change.md` only.
- `mem_search` previews are not enough — `mem_get_observation(id)`.

## References

- [`../_shared/templates/template-change.md`](../_shared/templates/template-change.md)
- [references/threat-matrix.md](references/threat-matrix.md)
- [`../questioning/SKILL.md`](../questioning/SKILL.md) · [`../sdd-explore/SKILL.md`](../sdd-explore/SKILL.md) · [`../design-spike/SKILL.md`](../design-spike/SKILL.md)
- [`../codebase-design/SKILL.md`](../codebase-design/SKILL.md) · [`../glossary/SKILL.md`](../glossary/SKILL.md)
- [`../sdd-spec/SKILL.md`](../sdd-spec/SKILL.md)
