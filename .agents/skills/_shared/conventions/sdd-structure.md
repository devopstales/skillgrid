# SDD Structure Convention (shared across all SDD skills)

This file is the single source of truth for the filesystem layout, artifact names, and phase order that every `sdd-*` skill reads or writes. If a skill disagrees with this file, this file wins. Change it here and update the skills, not the other way around.

## Phase Order (v3)

```
init → explore → propose → spec → apply → verify → archive
```

- **`sdd-propose`** absorbs former `sdd-design`. It reserves `NNN`, writes **`change.md`** (WHY + HOW + Step Blueprint + per-step WHAT + threat matrix).
- **`sdd-spec`** absorbs former `sdd-tasks`. It owns NN numbering and writes **`tasks.md`** (DAG state + all step punch-lists + verification stubs) and **`acceptance.feature`** (all Features tagged `@step-NN`) in one phase.
- **`sdd-apply`** marks checkboxes in `tasks.md` and updates `## State`.
- **`sdd-verify`** fills each step's `### Verification` block inside `tasks.md` — the per-step gate.
- **`sdd-archive`** gates on all step verdicts in `tasks.md`, then moves the entire change folder from `changes/<NNN>-<slug>/` to `archive/<NNN>-<slug>/`.

Retired standalone skills (redirect or remove when skills are updated): `sdd-design`, `sdd-tasks`.

## Numbering

- **Change**: 3-digit zero-padded `NNN` slug, e.g. `001-oauth-login`. `sdd-propose` reserves the next available number by scanning `docs/skillgrid/changes/NNN-*/` and `docs/skillgrid/archive/NNN-*/` + Mnemonic `sdd/{project}/changelog`, then writes the number into `change.md` and the folder name. Numbers are never reused.
- **Step**: 2-digit zero-padded `NN` within a change, e.g. `01-db-migration`. `sdd-spec` owns allocation; `sdd-apply` / `sdd-verify` never assign new steps after `tasks.md` is set.

## Directory Structure

```
docs/skillgrid/
├── config.yaml                   # Project-specific SDD config (stack, context, rules.*)
├── agents/                       # Skill registry, tracker, shared vocabulary
│   ├── skill-registry.md
│   ├── issue-tracker.md
│   ├── triage-labels.md
│   └── glossary/
│       ├── business.md
│       └── technical.md
├── changes/                      # Active development branch/context
│   └── <NNN-slug>/               # e.g. 001-oauth-login
│       ├── research.md           # SPIKE & FINDINGS — from sdd-explore
│       ├── change.md             # WHY + HOW — from sdd-propose (intent+design merged)
│       ├── tasks.md              # State + all steps + verify slots — from sdd-spec; updated by apply/verify
│       ├── acceptance.feature    # All step Features (@step-NN) — from sdd-spec
│       └── interview.md          # Optional — questioning rounds
└── archive/                      # HISTORICAL RECORD
    └── <NNN-slug>/               # Completed change moved here after successful gates
```

No `steps/` directory. No `intent.md`, `plan.md`, `state.yaml`, per-step `verification.md`, or `*-glossary-reference.md` companions.

## Artifact File Paths

| Skill | Phase | Creates / Updates | Path |
|---|---|---|---|
| sdd-init | 1 | Creates | `docs/skillgrid/config.yaml`, `docs/skillgrid/changes/`, `docs/skillgrid/archive/` |
| sdd-explore | 2 | Creates | `docs/skillgrid/changes/<NNN-slug>/research.md` |
| sdd-propose | 3 | Creates (reserves NNN) | `docs/skillgrid/changes/<NNN-slug>/change.md` |
| sdd-spec | 4 | Creates (owns NN; reads change) | `tasks.md` + `acceptance.feature` |
| sdd-apply | 5 | Updates | marks `tasks.md` checkboxes + `## State` |
| sdd-verify | 6 | Updates | fills `### Verification` per step in `tasks.md` |
| sdd-archive | 7 | Moves | `changes/<NNN-slug>/` → `archive/<NNN-slug>/` |

## Reading Artifacts

```
Research:   docs/skillgrid/changes/<NNN-slug>/research.md
Change:     docs/skillgrid/changes/<NNN-slug>/change.md
Tasks:      docs/skillgrid/changes/<NNN-slug>/tasks.md      (state + steps + verify)
Acceptance: docs/skillgrid/changes/<NNN-slug>/acceptance.feature
Config:     docs/skillgrid/config.yaml
```

## Writing Rules

- Always create the change directory before writing artifacts; reserve the NNN number **before** any folder is created.
- If a file already exists, READ it first and UPDATE it (don't overwrite blindly).
- If the change directory already exists with artifacts, the change is being CONTINUED.
- Use `docs/skillgrid/config.yaml` `rules.*` for project-specific per-phase constraints.
- Glossary terms: define/reuse via `docs/skillgrid/agents/glossary/{business,technical}.md`; fold first-use definitions or a short `## Glossary` footer into the main artifact. **Do not** create companion `*-glossary-reference.md` files.
- **Templates (mandatory):** instantiate artifacts from `.agents/skills/_shared/templates/` — read the template first, copy its outline, fill placeholders. Do not invent a parallel structure.
  - `sdd-propose` → [`templates/template-change.md`](../templates/template-change.md) → `change.md`
  - `sdd-spec` → [`templates/template-tasks.md`](../templates/template-tasks.md) → `tasks.md`
  - `sdd-spec` → [`templates/template-acceptance.feature`](../templates/template-acceptance.feature) → `acceptance.feature`
  - Index: [`templates/README.md`](../templates/README.md)

## `change.md` Shape

Single document (propose = former intent + design). **Canonical blank:** `templates/template-change.md`. Required sections match that template:

1. STATUS + Goal / Architecture / Tech stack header
2. **Goal / Out of scope / Non-Goals / Definition of Done** (mandatory — checkbox DoD)
3. Problem / users / rules / in scope / rollback
4. **Error handling** + **Testing strategy** (mandatory)
5. Step Blueprint (NN table with Goal + primary package — contract for spec)
6. Technical approach + architecture decisions (Choice / Alternatives / Rationale)
7. Data flow + optional File layout tree + Impacted Files Map (Step column)
8. Per-step WHAT (each step states Goal, Out of scope, Definition of Done + WHAT bullets)
9. Threat matrix (Applicable → owning step)
10. Migration / open questions
11. Glossary footer + author self-review

## `tasks.md` Shape

Combines former `state.yaml` + per-step tasks + per-step verification. **Canonical blank:** `templates/template-tasks.md`.

Required structure (see template for full scaffold):

```markdown
# Tasks: <NNN-slug>
# STATUS banner; Goal / Architecture / Tech Stack / Spec header
## Goal / Out of scope / Non-Goals / Definition of Done
## Global Constraints          # verbatim from change.md — mandatory
## State
## Step map
## Review workload
## NN-<name>                   # Goal, Out of scope, DoD, Files, Interfaces,
                               # Tasks (TDD micro-cycle + Run/Expected), Verification, Commit
## Archive gate checklist
```

Rules:
- Every Step Blueprint entry gets exactly one `## NN-<name>` section.
- Applicable threat-matrix rows become `[RED]` tasks ordered before their production (`[AFK]`) tasks.
- Each `[RED]` task uses the TDD micro-cycle (write fail → prove FAIL → impl → prove PASS → commit).
- Every verify line uses `Run: <command>` — `Expected: FAIL|PASS`.
- Assign every file in the Impacted Files map to exactly one step; encode deps as `Depends on:`.
- `sdd-apply` marks `[x]` and bumps `## State`.
- `sdd-verify` fills `### Verification`; a scenario without a passing run is `FAIL` for that step.
- Archive gate: no unchecked tasks; every step `PASS` or `PASS WITH WARNINGS`; Global Constraints held; `## State` reflects done.

## Acceptance Format (Gherkin / BDD)

One change-level `acceptance.feature` — all steps. **Canonical blank:** `templates/template-acceptance.feature`. Each Feature is tagged `@step-NN`:

```gherkin
# Source: docs/skillgrid/changes/<NNN-slug>/change.md

@step-01
Feature: <one-line capability for this step>
  As a <role>
  I want <capability>
  So that <value>

  @happy
  Scenario: <happy path>
    Given <precondition>
    When  <action>
    Then  <observable outcome>

  @edge
  Scenario: <edge case>
    Given <precondition>
    When  <action>
    Then  <expected failure or fallback>

  @failure
  Scenario: <failure state>
    Given <precondition>
    When  <action>
    Then  <expected error or recovery>
```

Rules:
- One `Feature` per step, tagged `@step-NN`. Scenarios are user-observable, not implementation-shaped.
- Use `@happy` / `@edge` / `@failure` / `@security` so the runner can select.
- ≥ 1 happy + 1 edge + 1 failure per step; every change.md per-step WHAT bullet → a scenario; every applicable threat row → a scenario in its owning step.
- `sdd-apply`'s test task references a specific scenario name (not "cover the acceptance").
- `sdd-verify` maps scenarios by `@step-NN` to the matching `## NN` section in `tasks.md`.

## Config File Reference

```yaml
# docs/skillgrid/config.yaml
schema: skillgrid-sdd/v1

context: |
  Tech stack: {detected}
  Architecture: {detected}
  Testing: {detected}
  Style: {detected}

rules:
  explore: []
  propose:
    - Include rollback plan for risky changes
    - Include success criteria (measurable)
    - Document architecture decisions with rationale (Choice/Alternatives/Rationale)
    - Carry the threat matrix applicable rows forward into per-step WHAT
  spec:
    - Own NN allocation; one ## section per Step Blueprint entry in tasks.md
    - Group tasks by step; keep each task completable in one sitting
    - One change-level acceptance.feature with @step-NN Features
    - Cover a happy path and at least one edge + failure per step
  apply:
    guidelines:
      - Follow existing code patterns
      - Mark tasks [x] as you go (in tasks.md)
    tdd: false            # true enables RED-GREEN-REFACTOR per task
    test_command: ""
  verify:
    test_command: ""
    build_command: ""
    coverage_threshold: 0
  archive:
    - Require every step in tasks.md to have PASS or PASS WITH WARNINGS before move
```

## Archive Structure

On successful gates, the change folder **moves** (not copies):

```
docs/skillgrid/changes/<NNN-slug>/  ──►  docs/skillgrid/archive/<NNN-slug>/
```

The NNN slug is preserved. No date prefix — the NNN number already carries the history. The archive is an AUDIT TRAIL — never delete or modify archived changes.

## Legacy layout (pre-v3)

Changes authored under the old model (`intent.md` + `plan.md` + `steps/<NN>/…` + companions) remain valid historical artifacts. New changes use v3 only. Do not renumber or rewrite archived trees unless explicitly migrating.
