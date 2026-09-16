---
name: requesting-code-review
description: Use when completing tasks, implementing major features, or before merging to verify work meets requirements
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: superpowers:requesting-code-review
---

# Requesting Code Review

**Announce at start:** "I'm using the skillgrid:requesting-code-review skill to get this reviewed."

Dispatch a code reviewer subagent to catch issues before they cascade. The reviewer gets precisely crafted context for evaluation — never your session's history.

**Core principle:** Review early, review often.

## Overview

Dispatch a code reviewer subagent to catch issues before they cascade — a second pair of eyes that sees only the work product, never your session's history. This skill dispatches two axes in parallel (standards + spec) so a change is checked against both the repo's conventions and what was actually asked, then triages the combined findings.

## When to Use

- After completing a work unit (a task, a feature, a bug fix) and before proceeding to the next one
- Before merging to main, so the diff is reviewed before `skillgrid:ship`
- When you want independent eyes on a non-trivial diff — fresh context, no assumption that your reasoning holds
- When you're stuck on your own change and need a fresh perspective

**When NOT to use:** For a large (50+ changed lines) or high-risk diff (auth, data migration, money, concurrency, public API) — use `skillgrid:parallel-code-review` instead. For the moment the *mandatory/optional* triggers below are met, see [When to Request Review](#when-to-request-review). For a quick "does this look right" sanity pass on a trivial change, this two-axis dispatch is overkill.

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

**Escalate to fan-out:** for a large (50+ changed lines) or high-risk diff
(auth, data migration, money, concurrency, public API), use
`skillgrid:parallel-code-review` instead — it dispatches several specialist
reviewers in parallel. This single-pass two-axis review is the lightweight
default.

## How to Request

**1. Get git SHAs:**
```bash
BASE_SHA=$(git rev-parse HEAD~1)  # or origin/main
HEAD_SHA=$(git rev-parse HEAD)
```

**2. Dispatch two reviewers in parallel:**

Two axes, two `general-purpose` subagents, run concurrently (do not pollute
each other's context):

**Standards reviewer** — template at [references/code-reviewer.md](references/code-reviewer.md)

Does the code follow this repo's documented standards? Pass:
- the diff range (`{BASE_SHA}..{HEAD_SHA}`)
- the standards sources: the glossary (`.skillgrid/glossary/`, ubiquitous
  language), the in-force ADRs from the change's ADR Review Manifest
  (`.skillgrid/specs/YYYY-MM-DD-<topic>/adr.md`, or `conventions.adr`), plus
  the template's smell baseline. Don't dump the whole ADR folder — only the
  in-force ADRs the manifest names for this change.
- brief: cite each violated standard (file + rule) and each baseline smell
  (name it, quote the hunk). Repo standards (glossary/ADRs) override the
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

**3. Aggregate (do NOT merge or rerank):**

Present the two reports under `## Standards` and `## Spec` headings, side by
side, verbatim or lightly cleaned. End with one line per axis: finding count
+ worst issue *within* that axis. Never pick a single cross-axis winner —
that's the reranking the separation exists to prevent. A change can be
spec-perfect but violate every convention, or beautifully written but
implement the wrong thing; the two axes are deliberately separate.

**4. Triage and fix findings:**
Apply `skillgrid:receiving-code-review` triage rules to the combined findings
from both axes — sort (fix now / defer / human look / noise), fix the
in-scope set one at a time with tests, log the rest, validate, and loop until
clean or capped at 3 rounds.

**5. Update ticket status (if tracked):**
- If `ticketing.enabled: true` AND the reviewed work corresponds to a ticket in `tasks.md` with a tracker ID: set ticket status → `done`
- If all tickets in the epic are `done`: set epic status → `done`
- Use the tracker adapter from `skillgrid:ticketing` (Backlog.md CLI, `gh`, `glab`, `jira`)

## Example

```
[Just completed Task 2: Add verification function]

You: Let me request code review before proceeding.

BASE_SHA=$(git log --oneline | grep "Task 1" | head -1 | awk '{print $1}')
HEAD_SHA=$(git rev-parse HEAD)

[Dispatch two reviewers in parallel]
  Standards (references/code-reviewer.md): glossary + ADRs + smell baseline
  Spec (references/spec-reviewer.md): acceptance.feature + blueprint

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

You: [triage both axes] [Fix glossary drift + add the missing scenario]
[Continue to Task 3]
```

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "I'll just review the diff myself instead of dispatching a reviewer" | You're the coordinator — reviewing the diff inline burns the context window you need to keep driving the work. Dispatch a reviewer subagent: the diff and the evaluation live in its context, and only the findings come back to you. |
| "The reviewer needs my whole session history to understand the change" | Hand it precisely crafted context, never your session's history. That keeps the reviewer on the work product, not your thought process. |

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

- [ ] The two reviewers were actually dispatched (standards + spec, in parallel) and both returned reports — not assumed
- [ ] Each reviewer's request included the diff range (`{BASE_SHA}..{HEAD_SHA}`) plus the context it needs (glossary/ADRs for standards, `acceptance.feature`/blueprint for spec)
- [ ] The request names the specific risk areas to focus on (violated standards, scenarios that must pass) — per the "How to Request" briefs
- [ ] The two reports were presented under `## Standards` and `## Spec` headings side by side, NOT merged or reranked
- [ ] The combined findings were triaged and in-scope issues fixed with tests — not left as "reviewed, good to go"
- [ ] The review was actually received (a report exists for each axis) before proceeding to the next task or merge

See templates at: [references/code-reviewer.md](references/code-reviewer.md) (Standards) and [references/spec-reviewer.md](references/spec-reviewer.md) (Spec).
