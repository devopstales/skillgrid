# Tasks: <NNN-slug>

> **STATUS:** `in-progress` | `complete` (<YYYY-MM-DD>) — <N>/<M> steps PASS
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-driven-development (or simple-execution) to implement step-by-step. Steps use checkbox (`- [ ]`) syntax.

**Goal:** <one sentence — must match `## Goal` and `change.md`>

**Architecture:** <2–3 sentences pointing at change.md decisions — do not restate the full design>

**Tech Stack:** <key technologies for this change>

**Spec:** `docs/skillgrid/changes/<NNN-slug>/change.md`

**Acceptance:** `docs/skillgrid/changes/<NNN-slug>/acceptance.feature` (`@step-NN`)

---

## Goal

<One clear sentence: what this change must achieve when all steps are done. Trace to change.md Definition of Done.>

## Out of scope / Non-Goals

- <Explicit exclusion — do not implement in this change>
- <Deferred / adjacent work that belongs to another NNN-slug or later step>
- <Hard non-goal agents must not "helpfully" expand into>

## Definition of Done

Change is done only when **all** of the following are true:

- [ ] Every success criterion / DoD checkbox in `change.md` is met
- [ ] Every `@step-NN` Feature in `acceptance.feature` has passing scenarios
- [ ] Every step below has Verdict `PASS` or `PASS WITH WARNINGS`
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] No **Global Constraint** violated
- [ ] Rollback path in `change.md` is still valid (or N/A documented)
- [ ] `## State` status is `done` (set at archive gate)

## Global Constraints

Copy verbatim from `change.md` (Error handling + Non-Goals + stack rules). Every step inherits these — do not restate per step.

- <e.g. Base path / module root>
- <e.g. Missing optional dep → warn+continue, do not fail>
- <e.g. No Windows support in this change>
- <e.g. JSONC merge must preserve comments>

---

## State

```yaml
phase: spec          # spec | apply | verify | archive
current_step: 01-<name>
status: in_progress  # in_progress | blocked | done
updated: <ISO-8601>
```

## Step map

| NN | Step | Tag | Blocked by | Acceptance |
|----|------|-----|------------|------------|
| 01 | `<name>` | `@step-01` | — | Feature tagged `@step-01` |
| 02 | `<name>` | `@step-02` | 01 | Feature tagged `@step-02` |

## Review workload (change-level)

| Field | Value |
|-------|-------|
| Estimated changed lines (change) | <n> |
| 400-line budget risk | Low \| Medium \| High |
| Chained PRs recommended | Yes \| No |
| Delivery strategy | ask-on-risk \| auto-chain \| single-pr \| exception-ok |

---

## 01-<name>

### Goal

<What this step alone must deliver — from Step Blueprint / per-step WHAT. Demoable or verifiable on its own.>

### Out of scope / Non-Goals

- <What this step must not touch (belongs to a later NN or another change)>
- <Horizontal work deferred to a later vertical slice>

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Produces contracts listed under Interfaces are available to dependents
- [ ] No Global Constraint violated

> Depends on: none

**Files:**
- Create: `<path>`
- Modify: `<path>`
- Test: `<path>`

**Interfaces:**
- Consumes: <signatures / contracts from earlier steps — or none>
- Produces: <names/types later steps rely on>

### Tasks

<!-- Each [RED] item uses the TDD micro-cycle. Inline test/impl snippets are optional ("when helpful"); Run/Expected lines are mandatory. -->

- [ ] 01.1 `[RED]` <failing test for WHAT / threat row>
  - [ ] 01.1.a Write failing test
  - [ ] 01.1.b Run to confirm fail — `Run: <command>` — Expected: FAIL
  - [ ] 01.1.c Minimal implementation
  - [ ] 01.1.d Run to confirm pass — `Run: <command>` — Expected: PASS
  - [ ] 01.1.e Commit — `feat(<scope>): <subject>`
- [ ] 01.2 `[AFK]` <edge / failure coverage mapped to Scenario> — `Run: <command>` — Expected: PASS
<!-- Applicable threat-matrix rows MUST appear as [RED] tasks before production tasks. -->

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `<cmd>` | PASS | | |
| Acceptance `@step-01` / `@p0` | `<cmd>` | PASS | | |
| Runtime harness | `<cmd>` | PASS | | |
| Rollback boundary | `<cmd or manual>` | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(<scope>): <step goal one-liner>`

---

## 02-<name>

### Goal

<What this step alone must deliver.>

### Out of scope / Non-Goals

- <Deferred to later NN or another change>

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-<name>

**Files:**
- Create: `<path>`
- Modify: `<path>`
- Test: `<path>`

**Interfaces:**
- Consumes: <from 01>
- Produces: <for later steps>

### Tasks

- [ ] 02.1 `[RED]` <…>
  - [ ] 02.1.a Write failing test
  - [ ] 02.1.b Run to confirm fail — `Run: <command>` — Expected: FAIL
  - [ ] 02.1.c Minimal implementation
  - [ ] 02.1.d Run to confirm pass — `Run: <command>` — Expected: PASS
  - [ ] 02.1.e Commit — `feat(<scope>): <subject>`
- [ ] 02.2 `[AFK]` <…> — `Run: <command>` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `<cmd>` | PASS | | |
| Acceptance `@step-02` / `@p0` | `<cmd>` | PASS | | |
| Runtime harness | `<cmd>` | PASS | | |
| Rollback boundary | `<cmd or manual>` | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(<scope>): <step goal one-liner>`

---

<!-- Duplicate ## NN-<name> sections for every Step Blueprint row. Never renumber after creation. -->

## Archive gate checklist

- [ ] Change-level **Definition of Done** fully checked
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated
- [ ] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [ ] STATUS banner updated to `complete`
