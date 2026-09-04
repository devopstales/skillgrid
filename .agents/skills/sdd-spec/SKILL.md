---
name: sdd-spec
description: "Own NN allocation; write change-level tasks.md and acceptance.feature from change.md by instantiating templates. Absorbs former sdd-tasks. Use after sdd-propose and before sdd-apply."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  author: devopstales
  version: "3.0"
  family: sdd
  part-of: skillgrid
  phase-order: "init → explore → propose → spec → apply → verify → archive"
  prev: [sdd-propose]
  next: [sdd-apply]
  artifact: tasks+acceptance
  delegate_only: true
---

# sdd-spec

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-spec` sub-agent **unless** you loaded this skill directly through the `skill()` tool.

- **Sub-agent (primary)**: you were delegated here by the SDD orchestrator. Continue with the phase work below. Do not re-delegate. Do not call the `skill()` tool again.
- **Orchestrator (skill() loaded this directly)**: STOP. Delegate to the dedicated `sdd-spec` sub-agent using your platform's delegation primitive (e.g. `task(...)`) instead of doing the work inline.

## Purpose

You are the SPEC phase (v3): **punch-lists + Gherkin in one phase**. You own **NN numbering**, write change-level **`tasks.md`** and **`acceptance.feature`**, and persist hybrid to Mnemonic.

This skill **absorbs former `sdd-tasks`**. Do not call `sdd-tasks` (retired stub). Do not create a `steps/` directory.

Order inside this phase:

1. Allocate steps + write `tasks.md` (from template)
2. Write `acceptance.feature` (from template)

Layout: [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md).

## What You Receive

- **Change id** `<NNN-slug>` — folder has `change.md` (and usually `research.md`).
- Hybrid only: write both files on disk **and** Mnemonic.
- Optional: delivery strategy (`ask-on-risk` | `auto-chain` | `single-pr` | `exception-ok`); ticket id; skills block.
- `force_ticket_creation` true → `issue-creation` for `tasks.md`.

## Conventions

- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md)
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — `rules.spec`
- [`../_shared/templates/template-tasks.md`](../_shared/templates/template-tasks.md)
- [`../_shared/templates/template-acceptance.feature`](../_shared/templates/template-acceptance.feature)
- [`references/acceptance-format.md`](references/acceptance-format.md)
- [`references/threat-matrix.md`](references/threat-matrix.md)

## Skill Loading

1. Injected skills, else recover Mnemonic: `sdd/<NNN-slug>/change` (required), `sdd/<NNN-slug>/research`, `sdd-init/{project}`.
2. Read filesystem: `docs/skillgrid/changes/<NNN-slug>/change.md`.
3. Read `docs/skillgrid/config.yaml` → `rules.spec` (legacy: `rules.tasks` / `rules.spec`).

## What to Do

### Step 1: Own NN allocation

From `change.md` Step Blueprint, allocate every `NN-<name>`. Never renumber after `tasks.md` exists. Vertical-slice rules: each step demoable alone; encode `Depends on:`; expand-contract for wide refactors.

### Step 2: Write tasks.md from template

1. READ [`../_shared/templates/template-tasks.md`](../_shared/templates/template-tasks.md).
2. Fill: Goal / Out of scope / Non-Goals / Definition of Done; **Global Constraints** (verbatim from change.md Error handling + Non-Goals + stack rules); State; Step map; Review workload; one `## NN-<name>` per blueprint entry.
3. Per step: Goal, Out of scope, DoD, Files, Interfaces, Tasks, empty Verification, Commit hint.
4. Applicable threat rows → `[RED]` tasks **before** production `[AFK]` tasks; each RED uses TDD micro-cycle (`a–e`) with `Run:` / `Expected: FAIL|PASS`.
5. Assign every Impacted Files row to exactly one step.
6. Write `docs/skillgrid/changes/<NNN-slug>/tasks.md`.

### Step 3: Write acceptance.feature from template

1. READ [`../_shared/templates/template-acceptance.feature`](../_shared/templates/template-acceptance.feature) + [`references/acceptance-format.md`](references/acceptance-format.md).
2. One change-level file; one `Feature` per step tagged `@step-NN`.
3. ≥1 `@happy` + `@edge` + `@failure` per step; every per-step WHAT bullet → scenario; every applicable threat row → scenario in owning step.
4. WHAT not HOW — no file paths/function names in Gherkin.
5. Write `docs/skillgrid/changes/<NNN-slug>/acceptance.feature`.

### Step 4: Self-check

- Every Blueprint step has a `## NN` section and an `@step-NN` Feature.
- Every `[RED]` has Run/Expected lines; Global Constraints present.
- Filesystem and Mnemonic content consistent.

### Step 5: Persist (hybrid, mandatory)

```
sid = mem_session_start(title: "sdd/<NNN-slug>/spec")
mem_save(topic_key: "sdd/<NNN-slug>/tasks", content: full tasks.md, …)
mem_save(topic_key: "sdd/<NNN-slug>/spec", content: full acceptance.feature, …)
```

### Step 6: Return envelope + handoff

```markdown
## Spec Created
**Change**: {NNN-slug}
**Status**: success | partial | blocked
**tasks.md**: N steps · Global Constraints {ok}
**acceptance.feature**: N Features · H/E/F coverage
**Threat handoff**: {K applicable → covered}
**Next**: sdd-apply | sdd-propose
```

Present choice: **Implement** (`sdd-apply`) or **Revise** (`sdd-propose`). Wait for human decision.

## Rules

- Instantiate both templates; no `steps/` tree; no per-step `verification.md`.
- Apply owns marking `[x]`; verify fills `### Verification`.
- Scenario names unique and referenceable from tasks.md verify lines.
- Apply `rules.spec` from config.

## Gotchas

- Do not call retired `sdd-tasks` / `sdd-design`.
- Missing `change.md` → run `sdd-propose`, do not invent acceptance from research alone.
- Omitting `@edge`/`@failure` blocks verify.

## References

- [`../_shared/templates/template-tasks.md`](../_shared/templates/template-tasks.md)
- [`../_shared/templates/template-acceptance.feature`](../_shared/templates/template-acceptance.feature)
- [`../sdd-propose/SKILL.md`](../sdd-propose/SKILL.md)
- [`../sdd-apply/SKILL.md`](../sdd-apply/SKILL.md)
