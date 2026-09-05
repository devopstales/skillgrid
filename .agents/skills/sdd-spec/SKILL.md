---
name: sdd-spec
description: Own NN allocation; write tasks.md (mandatory blocking/depends DAG) and acceptance.feature from templates, inherit threats as [RED], then stop for the user gate. Use when change.md is ready for punch-lists and Gherkin — after sdd-propose, before apply; never auto-apply.
disable-model-invocation: true
license: MIT
metadata:
  author: devopstales
  version: "4.0"
  part-of: skillgrid
---

# SDD Spec

Stage owner (v4). Punch-lists + Gherkin in one phase. Own **NN** numbering; write change-level **`tasks.md`** and **`acceptance.feature`**; then **STOP for user gate** (Implement | Revise). Do not call `sdd-apply`.

Layout: [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md).

## Hard Rules

- Instantiate both templates — no `steps/` tree, no per-step `verification.md`.
- **Every** step MUST declare `Depends on: <NN or none>` (kanban DAG for parallel apply).
- Applicable threat rows from `change.md` → `[RED]` tasks **before** production `[AFK]` tasks.
- Missing/ambiguous requirements → prefer revise via `sdd-propose` / `questioning`, not invent.
- `force_ticket_creation` → `issue-creation` for `tasks.md`; Backlog tickets must pass the **Backlog completeness gate** (type, references, DoD, Implementation Plan) — no thin stubs.
- Hybrid: disk + Mnemonic `sdd/<NNN-slug>/tasks` and `sdd/<NNN-slug>/spec`.

## Workflow

```
[ ] 1. Load change.md
[ ] 2. Own NN allocation
[ ] 3. Write tasks.md (blocking DAG)
[ ] 4. Write acceptance.feature
[ ] 5. Self-check + persist
[ ] 6. STOP — user gate
```

### 1. Load change.md

Required: `docs/skillgrid/changes/<NNN-slug>/change.md` (Mnemonic `sdd/<NNN-slug>/change`). Also read `research.md` / Prototype path if listed. Apply `rules.spec` from `config.yaml`. If `change.md` missing → run `sdd-propose`, do not invent from research alone.

### 2. Own NN allocation

From Step Blueprint allocate every `NN-<name>`. Never renumber after `tasks.md` exists. Vertical slices: each step demoable alone; expand-contract for wide refactors.

### 3. Write tasks.md

1. READ [`../_shared/templates/template-tasks.md`](../_shared/templates/template-tasks.md).
2. Fill Goal / Out of scope / DoD; **Global Constraints** (verbatim from change Error handling + Non-Goals + stack rules); `## State` (`phase: spec`); Step map; Review workload; one `## NN-<name>` per blueprint entry.
3. Per step: Goal, Out of scope, DoD, Files, Interfaces, Tasks, empty Verification stub, Commit hint.
4. **Mandatory** under each step: `Depends on: <NN or none>` — also fill Step map **Blocked by**.
5. Applicable threats → `[RED]` with TDD micro-cycle (`a–e`) and `Run:` / `Expected: FAIL|PASS`.
6. Assign every Impacted Files row to exactly one step.
7. Write `docs/skillgrid/changes/<NNN-slug>/tasks.md`.

### 4. Write acceptance.feature

1. READ [`../_shared/templates/template-acceptance.feature`](../_shared/templates/template-acceptance.feature) + [references/acceptance-format.md](references/acceptance-format.md).
2. One change-level file; one `Feature` per step tagged `@step-NN`.
3. ≥1 `@happy` + `@edge` + `@failure` per step; every WHAT bullet → scenario; every applicable threat → scenario in owning step.
4. WHAT not HOW — no file paths/function names in Gherkin.
5. Write `docs/skillgrid/changes/<NNN-slug>/acceptance.feature`.

### 5. Self-check + persist

- Every Blueprint step has `## NN` + `@step-NN` Feature + explicit `Depends on:`.
- Every `[RED]` has Run/Expected; Global Constraints present.
- `mem_session_start` → save `sdd/<NNN-slug>/tasks` and `sdd/<NNN-slug>/spec`.

### 6. STOP — user gate

```markdown
## Spec Created
**Change**: {NNN-slug}
**Status**: success | partial | blocked
**tasks.md**: N steps · blocking edges {ok} · Global Constraints {ok}
**acceptance.feature**: N Features · H/E/F coverage
**Threat handoff**: {K applicable → covered}
**Next**: USER GATE — choose Implement (sdd-apply) | Revise (questioning / sdd-propose)
```

Wait for human decision. Never auto-apply.

## Gotchas

- Missing `Depends on:` breaks parallel apply — treat as incomplete spec.
- Omitting `@edge`/`@failure` blocks verify.
- Do not invent a PASS under `### Verification` — apply leaves PENDING; verify fills it.
- Retired: `sdd-tasks`, `sdd-design`, `steps/` tree.

## References

- [`../_shared/templates/template-tasks.md`](../_shared/templates/template-tasks.md)
- [`../_shared/templates/template-acceptance.feature`](../_shared/templates/template-acceptance.feature)
- [references/acceptance-format.md](references/acceptance-format.md) · [references/threat-matrix.md](references/threat-matrix.md)
- [`../sdd-propose/SKILL.md`](../sdd-propose/SKILL.md) · [`../sdd-apply/SKILL.md`](../sdd-apply/SKILL.md)
- [`../glossary/SKILL.md`](../glossary/SKILL.md) · [`../questioning/SKILL.md`](../questioning/SKILL.md)
