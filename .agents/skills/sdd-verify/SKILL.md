---
name: sdd-verify
description: Independent SDD quality gate — per-step verdicts with runtime proof, human QA plan, code review; on findings append tasks and return to apply; archive only when PASS/WARNINGS, no open tasks, and human QA accepted or waived. Use when apply finished a batch or State.phase is verify.
disable-model-invocation: true
license: MIT
metadata:
  author: devopstales
  version: "4.0"
  part-of: skillgrid
---

# SDD Verify

Stage owner (v4). Independent gate — judge, do not fix. Traceability lives in Evidence (no separate `sdd-trace` stage). Findings re-enter **apply**; archive only when both agent and human gates pass.

Layout: [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md).

## Hard Rules

- Fill `### Verification` inside change-level `tasks.md` only — never `steps/.../verification.md`.
- Static analysis alone is never verification — run real test/build commands.
- An `@step-NN` scenario is COMPLIANT only when a covering test **passed at runtime**.
- On human QA or review findings → **append tasks**, set `## State.phase` to **`apply`**, do **not** archive.
- Archive eligibility only when: every step PASS or PASS WITH WARNINGS, no open `- [ ]`, human QA accepted **or explicitly waived**.
- Hybrid: disk + Mnemonic `sdd/<NNN-slug>/verification`, upsert `…/tasks`, and `…/qa-plan`.

## Workflow

```
[ ] 1. Agent gate (verdicts + runtime proof + trace)
[ ] 2. Write human QA plan
[ ] 3. Code review as needed
[ ] 4. Findings → apply (do not archive)
[ ] 5. Archive eligibility check
```

### 1. Agent gate

Load `change.md`, `tasks.md`, `acceptance.feature`, `apply-progress`. Apply `rules.verify`. Call **`verification`** Iron Law.

For each step (NN order; honor `Depends on:` — predecessor must already be PASS/WARNINGS):

1. Incomplete tasks under that step → mark step `blocked`, skip full suite.
2. Map every `@step-NN` scenario → covering test + runtime result (COMPLIANT / PARTIAL / FAILING / UNTESTED).
3. Check `change.md` coherence + Global Constraints.
4. Run focused tests / build / coverage per config and `Run:` lines — record command, exit, counts.
5. If Strict TDD active, load [references/strict-tdd-verify.md](references/strict-tdd-verify.md) (else skip).
6. Fill `### Verification` Verdict + Evidence (`Run` / `Expected` / `Result`) per template. Statuses: [references/report-format.md](references/report-format.md).

Verdict: `FAIL` if any CRITICAL, unchecked task, violated Global Constraint, or required command non-zero; `PASS` only if clean; else `PASS WITH WARNINGS`.

Bump `## State` (`phase: verify`). Persist FAIL the same as PASS.

### 2. Write human QA plan

Write **`qa-plan.md`** beside the change artifacts **or** a `## QA plan` section in `tasks.md`:

- What a human should exercise (happy / edge / failure)
- Environments / data / accounts
- Pass / fail criteria and how to waive

Mnemonic: `sdd/<NNN-slug>/qa-plan`.

### 3. Code review

As needed: **`requesting-code-review`**. High-risk (threat Applicable, large/chained, shared conventions) → **`judgment-day`**. Process feedback with **`review-reception`** — verify-first, one item at a time.

### 4. Findings → apply (do not archive)

If human QA or review finds gaps:

1. Append concrete tasks under the right `## NN-<name>` (or a new follow-up step if scoped).
2. Set `## State.phase: apply`, `status: in_progress`.
3. Return envelope **Next: sdd-apply** — never `sdd-archive`.

### 5. Archive eligibility

Only when agent gates are PASS/WARNINGS for every step, no open tasks, and human QA accepted or waived → **Next: sdd-archive**. Else stay in verify/apply loop.

```markdown
## Verification Report
**Change**: {NNN-slug}
**Per-step verdicts**: …
**QA plan**: qa-plan.md | ## QA plan
**Review**: … | none
**Status**: success | partial | blocked
**Next**: sdd-archive | sdd-apply (findings) | remediation
```

## Gotchas

- Scenario totals must equal the `@step-NN` Feature count — never invent.
- A PASS with a CRITICAL finding is a FAIL.
- Do not archive on agent PASS alone while human QA is still open.
- `mem_search` previews lose scenario lists — `mem_get_observation(id)`.
- Coverage/lint are WARNING/SUGGESTION at worst — not CRITICAL by themselves.

## References

- [references/report-format.md](references/report-format.md) · [references/strict-tdd-verify.md](references/strict-tdd-verify.md)
- [`../_shared/templates/template-tasks.md`](../_shared/templates/template-tasks.md)
- [`../verification/SKILL.md`](../verification/SKILL.md) · [`../requesting-code-review/SKILL.md`](../requesting-code-review/SKILL.md)
- [`../review-reception/SKILL.md`](../review-reception/SKILL.md) · [`../judgment-day/SKILL.md`](../judgment-day/SKILL.md)
- [`../sdd-apply/SKILL.md`](../sdd-apply/SKILL.md) · [`../sdd-archive/SKILL.md`](../sdd-archive/SKILL.md)
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) · [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md)
