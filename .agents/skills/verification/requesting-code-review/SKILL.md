---
name: requesting-code-review
description: Use when completing tasks, implementing major features, or before merging to verify work meets requirements
license: MIT
metadata:
  author: devopstales
  version: "1.1"
  part-of: skillgrid
  based_on: superpowers:requesting-code-review
---

# Requesting Code Review

**Announce at start:** "I'm using the skillgrid:requesting-code-review skill to get this reviewed."

Dispatch a code reviewer subagent to catch issues before they cascade. The reviewer gets precisely crafted context for evaluation — never your session's history.

**Core principle:** Review early, review often.

**Rigor tier:** at T2 (Beta) this three-axis pass is the default review (per `_shared/conventions/rigor-tiers.md`). At T3 (GA) use `skillgrid:parallel-code-review` instead, with at least one reviewer running on a model different from the one that implemented.

**Blindness contract:** the value of a review is its independence from the producer. All three reviewers are dispatched as **fresh subagent contexts** that see only the work product — the diff and the named standards/spec/security sources — and never the coordinator's session history, its rationale, or its self-assessment. That is what makes the findings independent evidence rather than the implementer talking to itself. Two rules keep it real:

- **No leaks.** Do not pass a reviewer your narrative, your "why I did it this way", your confidence, or a prior score/verdict on the same diff. Hand it the diff range plus the standards or spec sources (the contract the template already names) and nothing else.
- **Independence grade.** Label each review with an independence grade (mirrors the evaluator protocol; see `_shared/rules/calibration.md`):
  - **A** — fresh context, no shared history with the implementer (the normal case for a dispatched subagent).
  - **B** — fresh context but same model family / toolchain as the implementer.
  - **C** — the review happens in the implementer's own context (an inline self-review) or the implementer's prior verdict leaked in.

  A **Grade C** review is **diagnostic only**: it can guide a fix, but it cannot stand as the "independently reviewed" evidence that `ship` or a merge requires. If the runtime cannot isolate a fresh context (no subagent dispatch available, or you reviewed inline), record the review as Grade C and say so — never present it as independent. At T2 the default three-axis dispatch is Grade A/B; at T3 `skillgrid:parallel-code-review` adds a cross-model reviewer to reach a clean Grade A.

## Overview

Dispatch code reviewer subagents to catch issues before they cascade — a second pair of eyes that sees only the work product, never your session's history. This skill dispatches three axes in parallel (standards + spec + security) so a change is checked against the repo's conventions, what was actually asked, and exploitable security issues, then triages the combined findings.

## When to Use

- After completing a work unit (a task, a feature, a bug fix) and before proceeding to the next one
- Before merging to main, so the diff is reviewed before `skillgrid:ship`
- When you want independent eyes on a non-trivial diff — fresh context, no assumption that your reasoning holds
- When you're stuck on your own change and need a fresh perspective

**When NOT to use:** For a diff that meets the review escalation threshold (per `_shared/conventions/rigor-tiers.md`) — use `skillgrid:parallel-code-review` instead. For the moment the *mandatory/optional* triggers below are met, see [When to Request Review](#when-to-request-review). For a quick "does this look right" sanity pass on a trivial change, this three-axis dispatch is overkill.

## When to Request Review

**Mandatory:**
- After each task in subagent-driven development
- After completing major feature
- After `skillgrid:qa` renders a PASS or CONCERNS gate (QA is the quality gate; review is the standards/spec gate — they answer different questions)
- Before merge to main — review runs before `skillgrid:ship` (the integration + archive-move step)

**Optional but valuable:**
- When stuck (fresh perspective)
- Before refactoring (baseline check)
- After fixing complex bug

**Escalate to fan-out:** for a diff at the review escalation threshold (per
`_shared/conventions/rigor-tiers.md`), use
`skillgrid:parallel-code-review` instead — it dispatches several specialist
reviewers in parallel. This single-pass two-axis review is the lightweight
default.

## How to Request

**1. Get git SHAs:**
```bash
BASE_SHA=$(git rev-parse HEAD~1)  # or origin/main
HEAD_SHA=$(git rev-parse HEAD)
```

**2. Dispatch three reviewers in parallel:** 

Three axes, three `general-purpose` subagents, run concurrently (do not pollute
each other's context):

**Standards reviewer** — template at [references/code-reviewer.md](references/code-reviewer.md)

Does the code follow this repo's documented standards? The two-axis contract
and the smell baseline are canonically defined in
`_shared/conventions/code-standards.md` ("Reviewing"); this template inlines
them for the reviewer. Pass:
- the diff range (`{BASE_SHA}..{HEAD_SHA}`)
- the standards sources: the terms files (`.skillgrid/artifacts/01-business-terms.md` /
  `02-technical-terms.md`, ubiquitous language), the in-force ADRs from the change's ADR Review Manifest
  (`.skillgrid/specs/YYYY-MM-DD-<topic>/adr.md`, or `conventions.artifacts`), plus
  the template's smell baseline. Don't dump the whole ADR folder — only the
  in-force ADRs the manifest names for this change.
- brief: cite each violated standard (file + rule) and each baseline smell
  (name it, quote the hunk). Repo standards (terms/ADRs) override the
  baseline. Skip anything tooling (lint) already enforces.

**Spec reviewer** — template at [references/spec-reviewer.md](references/spec-reviewer.md)

Does the code implement what was asked? Pass:
- the diff range
- the spec sources: the change's `acceptance.feature`
  (`.skillgrid/specs/<id>/acceptance.feature`), the blueprint
  (`.skillgrid/specs/YYYY-MM-DD-<topic>/blueprint.md`), and the originating
  spec if the blueprint names one
- brief (BDD is always on): per scenario, does the diff make each `SATISFIES`
  scenario pass? Is TDD Evidence's RED→GREEN real, and did the spec zone rule
  hold (specs committed before code)? Any requirement missing or partial? Any
  behavior the spec never asked for (scope creep)? A scenario claimed green
  but whose RED evidence is missing or whose code path doesn't match the
  scenario's Given/When/Then is a Critical finding.

**Security reviewer** — template at [references/security-reviewer.md](references/security-reviewer.md)

Does the diff contain exploitable security issues? Pass:
- the diff range
- the security standards: the `skillgrid:owasp-security` skill
  (`.agents/skills/verification/owasp-security/SKILL.md`) and its `reference/`
  files. The template inlines the 5-step workflow, the triage rubric, and the
  reporting format, so the reviewer needs no second hop.
- brief: run the 5-step workflow (map entry points → load references → sweep →
  triage → report). For each finding, cite the CWE + OWASP/LLM/ASI category,
  the concrete entry-point-to-sink path, and a Confidence (Confirmed | Likely |
  Needs verification). Drop candidates that fail the four-part triage rubric;
  say plainly when nothing exploitable was found.

**3. Aggregate (do NOT merge or rerank) and persist `review.md`:**

Present the three reports under `## Standards`, `## Spec`, and `## Security`
headings, side by side, verbatim or lightly cleaned. End with one line per
axis: finding count + worst issue *within* that axis. Never pick a single
cross-axis winner — that's the reranking the separation exists to prevent. A
change can be spec-perfect but violate every convention, beautifully written
but implement the wrong thing, or conventionally clean and spec-complete but
one SQLi away from a data breach; the three axes are deliberately separate.

Then **write the durable audit record** — `.skillgrid/specs/<topic>/review.md`
— from [templates/review.md](templates/review.md), next to `report.md` (which
owns the QA gate; `review.md` owns this review). It records the three axes
side by side, the explicit "What Important means" threshold, the capped Minor
findings, each axis's **independence grade** (the no-leaks proof), and the
floor verdict. **Commit `review.md` in the spec zone** before continuing
code-zone work (spec-zone rule). It survives the session — a review that
lives only in the transcript is not an audit record, and a Grade C review
that never got written down is even harder to audit later.

**4. Triage and fix findings:**
Apply `skillgrid:receiving-code-review` triage rules to the combined findings
from all three axes — sort (fix now / defer / human look / noise), fix the
in-scope set one at a time with tests, log the rest, validate, and loop until
clean or capped per `_shared/conventions/rigor-tiers.md` (fix loop cap).

**5. Update ticket status (if tracked):**
- If `ticketing.enabled: true` AND the reviewed work corresponds to a ticket in `tasks.md` with a tracker ID: set ticket status → `done`
- If all tickets in the epic are `done`: set epic status → `done`
- Use the tracker adapter from `skillgrid:ticketing` (Backlog.md CLI, `gh`, `glab`, `jira`)

**6. Update `state.yaml`:** set `pipeline.current_phase: review`.

## Example

```
[Just completed Task 2: Add verification function]

You: Let me request code review before proceeding.

BASE_SHA=$(git log --oneline | grep "Task 1" | head -1 | awk '{print $1}')
HEAD_SHA=$(git rev-parse HEAD)

[Dispatch three reviewers in parallel]
  Standards (references/code-reviewer.md): glossary + ADRs + smell baseline
  Spec (references/spec-reviewer.md): acceptance.feature + blueprint
  Security (references/security-reviewer.md): OWASP 5-step workflow + triage rubric

[Standards returns]
  Strengths: Naming inside the glossary, real tests
  Issues: Important: verifyIndex() drifts from glossary "Validate"
          Minor: Possible Duplicated Code (retry block in 2 files)
  Assessment: Standards met with fixes

[Spec returns]
  Scenarios: verifies-index-integrity → PASS,
             repairs-corrupt-entry → MISSING (no test, no code)
  Scope Creep: none
  Assessment: Spec met partially

[Security returns]
  [High] SQLi in /search (CWE-89, OWASP A05:2025) — Confirmed
  [Medium] Open redirect in /login?next= (CWE-601, OWASP A01:2025) — Likely
  Assessment: 2 findings: 0 Critical, 1 High, 1 Medium

You: [triage all three axes]
  [Fix glossary drift + add the missing scenario + parameterize the query + allowlist redirect]
[Continue to Task 3]
```

## How to measure it

Per `_shared/conventions/measurement.md`.

| | Indicator | Data source | Direction |
|---|-----------|-------------|-----------|
| Leading | Time from PR open to first review `review.md` commit | Git history | should fall |
| Lagging | Defects caught pre-merge (in `review.md`) vs. escaping to an incident | `review.md` findings + incident records | caught/rising, escaped/falling |

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "I'll just review the diff myself instead of dispatching a reviewer" | You're the coordinator — reviewing the diff inline burns the context window you need to keep driving the work. Dispatch a reviewer subagent: the diff and the evaluation live in its context, and only the findings come back to you. An inline self-review is Grade C — diagnostic, not independent evidence. |
| "The reviewer needs my whole session history to understand the change" | Hand it precisely crafted context, never your session's history. That keeps the reviewer on the work product, not your thought process. |
| "The reviewer is a fresh subagent, so its grade is obviously A" | Fresh context is necessary but not sufficient. If it shares the implementer's model family or toolchain it's Grade B; if your prior verdict or rationale leaked in it's Grade C (diagnostic only). Name the grade before you treat it as independent. |

## Red Flags

**Never:**
- Skip review because "it's simple"
- Ignore Critical issues
- Proceed with unfixed Important issues
- Argue with valid technical feedback

**If reviewer wrong:**
- Push back with technical reasoning
- Show code/tests that prove it works
- Request clarification

## Verification

- [ ] The three reviewers were actually dispatched (standards + spec + security, in parallel) and all three returned reports — not assumed
- [ ] Each reviewer's request included the diff range (`{BASE_SHA}..{HEAD_SHA}`) plus the context it needs (terms/ADRs for standards, `acceptance.feature`/blueprint for spec, OWASP skill + references for security) — and **nothing from your session narrative or a prior verdict** (blindness contract)
- [ ] Each review is labeled with an independence grade (A/B/C); any Grade C review is reported as diagnostic only, not as independent evidence for ship/merge
- [ ] The request names the specific risk areas to focus on (violated standards, scenarios that must pass, exploitable security issues) — per the "How to Request" briefs
- [ ] The three reports were presented under `## Standards`, `## Spec`, and `## Security` headings side by side, NOT merged or reranked
- [ ] `review.md` was written to `.skillgrid/specs/<topic>/review.md` and committed (spec zone) — the durable audit record with all three axes side by side, the "What Important means" threshold, capped Minor findings, each axis's independence grade, and the floor verdict
- [ ] The combined findings were triaged and in-scope issues fixed with tests — not left as "reviewed, good to go"
- [ ] The review was actually received (a report exists for each axis) before proceeding to the next task or merge

See templates at: [references/code-reviewer.md](references/code-reviewer.md) (Standards), [references/spec-reviewer.md](references/spec-reviewer.md) (Spec), and [references/security-reviewer.md](references/security-reviewer.md) (Security).
