# SDD Structure Convention (shared across all SDD skills)

This file is the single source of truth for the filesystem layout, artifact names, and phase order that every `sdd-*` skill reads or writes. If a skill disagrees with this file, this file wins. Change it here and update the skills, not the other way around.

## Phase Order

```
init → explore → propose → design → tasks → spec → apply → verify → archive
```

- `sdd-tasks` runs **before** `sdd-spec` because `acceptance.feature` lives inside each step folder (`steps/<NN>-<name>/acceptance.feature`) — the step tree must exist before any acceptance is written.
- `sdd-spec` consumes `plan.md` and populates each step's `acceptance.feature`.
- `sdd-apply` marks `steps/<NN>-<name>/tasks.md` lines `[x]` as it goes and persists per-step apply progress.
- `sdd-verify` writes one `verification.md` **per step** — the per-step gate.
- `sdd-archive` gates on all steps' `verification.md`, then moves the entire change folder from `changes/<NNN>-<slug>/` to `archive/<NNN>-<slug>/`.

## Numbering

- **Change**: 3-digit zero-padded `NNN` slug, e.g. `001-oauth-login`. `sdd-propose` reserves the next available number by scanning `docs/skillgrid/changes/NNN-*/` and `docs/skillgrid/archive/NNN-*/` + Mnemonic `sdd/{project}/changelog`, then writes the number into `intent.md` and the folder name. Numbers are never reused.
- **Step**: 2-digit zero-padded `NN` within a change, e.g. `01-db-migration`. `sdd-tasks` owns allocation; `sdd-apply`/`sdd-verify` never assign new steps after the tree is set.

## Directory Structure

```
docs/skillgrid/
├── config.yaml                   # Project-specific SDD config (stack, context, rules.*)
├── changes/                      # Active development branch/context
│   └── <NNN-slug>/               # e.g. 001-oauth-login
│       ├── intent.md             # WHY & WHAT  — from sdd-propose
│       ├── research.md           # SPIKE & FINDINGS — from sdd-explore
│       ├── plan.md               # HOW: architecture + impacted files — from sdd-design
│       ├── state.yaml            # DAG state (survives compaction) — orchestrator
│       └── steps/                # SUBTASKS: sequential execution phases
│           └── <NN>-<step-name>/ # e.g. 01-db-migration
│               ├── tasks.md          # Execution punch-list — sdd-tasks, [x]-marked by sdd-apply
│               ├── acceptance.feature# E2E UAT in Gherkin — sdd-spec
│               └── verification.md   # Per-step PASS/FAIL gate — sdd-verify
└── archive/                      # HISTORICAL RECORD
    └── <NNN-slug>/               # Completed change moved here after successful gates
```

## Artifact File Paths

| Skill | Phase | Creates / Reads | Path |
|---|---|---|---|
| orchestrator | — | Creates/Updates | `docs/skillgrid/changes/<NNN-slug>/state.yaml` |
| sdd-init | 1 | Creates | `docs/skillgrid/config.yaml`, `docs/skillgrid/changes/`, `docs/skillgrid/archive/` |
| sdd-explore | 2 | Creates | `docs/skillgrid/changes/<NNN-slug>/research.md` |
| sdd-propose | 3 | Creates (reserves NNN) | `docs/skillgrid/changes/<NNN-slug>/intent.md` |
| sdd-design | 4 | Creates (reads intent) | `docs/skillgrid/changes/<NNN-slug>/plan.md` |
| sdd-tasks | 5 | Creates (reads plan; owns step tree) | `docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/tasks.md` + step folders |
| sdd-spec | 6 | Creates (reads intent + plan + tasks) | `docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/acceptance.feature` |
| sdd-apply | 7 | Updates (step tree) | marks `steps/<NN-name>/tasks.md` `[x]` |
| sdd-verify | 8 | Creates (per step) | `docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/verification.md` |
| sdd-archive | 9 | Moves | `changes/<NNN-slug>/` → `archive/<NNN-slug>/` |

## Reading Artifacts

```
Research:   docs/skillgrid/changes/<NNN-slug>/research.md
Intent:     docs/skillgrid/changes/<NNN-slug>/intent.md
Plan:       docs/skillgrid/changes/<NNN-slug>/plan.md
Steps:      docs/skillgrid/changes/<NNN-slug>/steps/         (NN-listed per step)
Tasks:      docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/tasks.md
Acceptance: docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/acceptance.feature
Verify:     docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/verification.md
Config:     docs/skillgrid/config.yaml
State:      docs/skillgrid/changes/<NNN-slug>/state.yaml
```

## Writing Rules

- Always create the change directory before writing artifacts; reserve the NNN number **before** any folder is created.
- If a file already exists, READ it first and UPDATE it (don't overwrite blindly).
- If the change directory already exists with artifacts, the change is being CONTINUED.
- Use `docs/skillgrid/config.yaml` `rules.*` for project-specific per-phase constraints.

## Acceptance Format (Gherkin / BDD)

Each `acceptance.feature` is a **Gherkin file** describing the end-to-end behavior of its step — not the internal HOW. Format:

```gherkin
# <NN>-<name> step acceptance
# Source: docs/skillgrid/changes/<NNN-slug>/intent.md + plan.md

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
  Scenario: <edge case / failure>
    Given <precondition>
    When  <action>
    Then  <expected failure or fallback>
```

Rules:
- One `Feature` per step file. Scenarios are user-observable, not implementation-shaped.
- Use `@happy` / `@edge` / `@security` tags so the runner can select.
- `sdd-apply`'s test task references a specific scenario name (not "cover the acceptance").
- `sdd-verify` maps each scenario to a test execution — a scenario without a passing run is a `FAIL` for the step.

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
  intent:
    - Include rollback plan for risky changes
    - Include success criteria (measurable)
  plan:
    - Document architecture decisions with rationale (Choice/Alternatives/Rationale)
    - Carry the threat matrix applicable rows forward into per-step WHAT
  tasks:
    - Group by step (2-digit NN), use hierarchical numbering within a step
    - Keep each task completable in one sitting
  spec:
    - Use Gherkin Feature/Scenario per step
    - Cover a happy path and at least one edge case per step
  apply:
    guidelines:
      - Follow existing code patterns
      - Mark tasks [x] as you go (per step)
    tdd: false            # true enables RED-GREEN-REFACTOR per task
    test_command: ""
  verify:
    test_command: ""
    build_command: ""
    coverage_threshold: 0
  archive:
    - Require all steps to have a passing verification.md before move
```

## Archive Structure

On successful gates, the change folder **moves** (not copies):

```
docs/skillgrid/changes/<NNN-slug>/  ──►  docs/skillgrid/archive/<NNN-slug>/
```

The NNN slug is preserved. No date prefix — the NNN number already carries the history. The archive is an AUDIT TRAIL — never delete or modify archived changes.
