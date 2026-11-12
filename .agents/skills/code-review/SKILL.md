---
name: code-review
description: Use after completing a work unit, a major feature, or before merging to verify the diff. Topology-driven code review — 3-axis for small/low-risk diffs, core-five fan-out + conditional specialists for large/high-risk, red team last.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: superpowers:requesting-code-review + bmad:bmad-code-review + gstack:review
---

# Code Review

**Announce at start:** "I'm using the skillgrid:code-review skill to review this diff."

## Overview

Dispatch independent reviewer subagents to catch issues before they cascade. One
skill, one topology step: the fan-out is selected from the diff's size, risk,
and rigor tier, never from a choice between skills. Every lens is a fresh
subagent context that sees only the work product; the coordinator verifies each
finding at its location, renders one verdict per entry, and routes accepted
fixes into the `skillgrid:receiving-code-review` fix loop.

**Core principle:** Review early, review often. A review is evidence, and its
value is its independence from the producer.

**Rigor tier:** the tier floor per `_shared/planning/rigor-tiers.md` still
applies — T3 requires at least one reviewer on a model different from the one
that implemented the change (a model reviewing its own work carries its own
blind spots; the fresh-model reviewer is a recommendation the human can
decline, and the decline is recorded).

## When to Use

**Mandatory:**
- After each task in subagent-driven development
- After completing a major feature
- After `skillgrid:qa` renders a PASS or CONCERNS gate (QA is the quality
  gate; review is the standards/spec gate — they answer different questions)
- Before merge to main — review runs before `skillgrid:ship` (the
  integration + archive-move step)

**Optional but valuable:**
- When stuck (fresh perspective)
- Before refactoring (baseline check)
- After fixing a complex bug
- When you want independent eyes on a non-trivial diff — fresh context, no
  assumption that your reasoning holds

**When NOT to use:** T0 changes (no review, per
`_shared/planning/rigor-tiers.md`), and an empty diff (stop — there is nothing
to review).

## The Topology Step

The topology step replaces the old two-skill choice. Select the fan-out from
diff size / risk / rigor tier. Pin the diff first:

```bash
BASE_SHA=$(git merge-base <base-branch> HEAD)   # or the SHA the work started from
HEAD_SHA=$(git rev-parse HEAD)
git diff --stat $BASE_SHA..$HEAD_SHA
```

Capture the changed-line count **and the file types touched**. If the diff is
empty, stop — there is nothing to review. Then select:

- **<50 changed lines, low risk** → the 3-axis pass (Standards + Spec + Security).
- **≥50 changed lines, or high-risk** (auth, data migration, money, concurrency, public API) → the core five (Standards, Spec, Edge cases, Verification gaps, Security), plus conditional Accessibility (diff touches UI) / Performance (diff touches data access / rendering) specialists.
- **Red team always last** (reads the other specialists' merged findings); the tier floor per `_shared/planning/rigor-tiers.md` still applies (T3: at least one fresh-model reviewer).

The line-count floor gates the *core* five, not the domain specialists: a
small diff that touches UI or data access still earns the Accessibility or
Performance specialist on top of the 3-axis pass. The high-risk categories and
the trivial-skip rule live in `_shared/planning/rigor-tiers.md` (review
escalation threshold) — do not restate the numbers here. Say which lenses you
ran and why, including any you skipped for lack of surface.

## Shared Protocol

The three blocks below are the shared review protocol. They appear **exactly
once** in the repo's review surface — this is their home; every other review
surface (persona briefs, the ledger reference, the goldfish pipeline) cites
them, never restates them.

### Blindness Contract

The value of a review is its independence from the producer. Every lens is
dispatched as a **fresh subagent context** that sees only the work product —
the diff and the sources named for that lens — and never the coordinator's
session history, its rationale, or its self-assessment. That is what makes the
findings independent evidence rather than the implementer talking to itself.
Two rules keep it real:

- **No leaks.** Do not pass a reviewer your narrative, your "why I did it this
  way", your confidence, or a prior score/verdict on the same diff. Hand it
  the diff range plus the sources the template already names, and nothing else.
- **Independence grade.** Label each lens with an independence grade (mirrors
  the evaluator protocol; see `_shared/verification/calibration.md`):
  - **A** — fresh context, no shared history with the implementer (the normal case for a dispatched subagent).
  - **B** — fresh context but same model family / toolchain as the implementer.
  - **C** — the review happens in the implementer's own context (an inline
    self-review) or the implementer's prior verdict leaked in.

A **Grade C** review is **diagnostic only**: it can guide a fix, but it cannot
stand as the "independently reviewed" evidence that `ship` or a merge
requires. If the runtime cannot isolate a fresh context (no subagent dispatch
available, or you reviewed inline), record the review as Grade C and say so —
never present it as independent. At T3 the fresh-model reviewer reaches a
clean Grade A.

### Verdict Table (six values)

The coordinator renders exactly one verdict per triaged entry:

| Verdict | Meaning | Ledger severity (record boundary) |
|---------|---------|-----------------------------------|
| `high` | The bad outcome is real and intolerable (bug, data loss, security break, broken feature) | `CRITICAL` |
| `medium` | Real, tolerable — should be fixed before merge | `WARNING` |
| `low` | Real, cosmetic or negligible | `SUGGESTION` |
| `false` | You checked; the bad outcome does not happen here. Say what disproves it. | — (drop the entry) |
| `maybe-false` | You couldn't tell. Say what would settle it. | — (human look) |
| `could-not-verify` | The check is **not inferable from the diff alone** — it needs a held-out test, a runtime, or state the diff doesn't show. Never a confident pass. Emit the concrete "write a held-out test" action that would settle it. | — (human look / held-out test) |

Severity normalizes at the record boundary: raw findings carry this reviewer
vocabulary; the ledger entry (per `_shared/craft/review-ledger.md`) carries
`CRITICAL` / `WARNING` / `SUGGESTION`.

### Resilience Rule

A failed lens is logged by name: if a lens fails, times out, or returns empty,
record its name and continue with the survivors. If **all** lenses fail or
return empty, the review is **"possibly incomplete", never "clean"** — report
that it may be incomplete and list which lenses didn't run.

## Dispatch

Launch **every selected lens in a single message** (multiple subagent calls)
so they run in parallel, each with fresh context and no bias from the others.
Use a `general-purpose` subagent per lens. Each lens's prompt is: the template
path in this skill's `reviewers/` directory, the diff scope
(`$BASE_SHA..$HEAD_SHA` passed as a **path**, not inlined text — each lens
reads it), and the Blindness Contract. Read `.skillgrid/config.yaml` first:
use `testing.runner` for any verification the triage step needs, and the
terms/ADR paths (`conventions.glossary`, the change's ADR Review Manifest at
`.skillgrid/change-specs/YYYY-MM-DD-<topic>/adr.md`) for the Standards lens.
Don't dump the whole ADR folder — only the in-force ADRs the manifest names
for this change.

The two-axis contract (Standards vs Spec, never cross-reranked) and the smell
baseline are canonically defined in `_shared/rules/code-standards.md`
("Reviewing"); the templates instantiate them.

| Lens | Template | Lens focus | Selected when |
|------|----------|------------|---------------|
| Standards | [reviewers/code-reviewer.md](reviewers/code-reviewer.md) | glossary, in-force ADRs, smell baseline | always (3-axis pass + core five) |
| Spec | [reviewers/spec-reviewer.md](reviewers/spec-reviewer.md) | per-scenario BDD, missing/partial, scope creep | always (3-axis pass + core five) |
| Security | [reviewers/security.md](reviewers/security.md) | injection, authz, exposure, secrets, dep risk (OWASP 5-step) | always (3-axis pass + core five) |
| Edge cases | [reviewers/edge-case-hunter.md](reviewers/edge-case-hunter.md) | unhandled branches, boundaries, deletion regressions | core five |
| Verification gaps | [reviewers/verification-gap.md](reviewers/verification-gap.md) | does verification actually catch a break? | core five |
| Accessibility | [reviewers/accessibility.md](reviewers/accessibility.md) | keyboard, semantics, forms, ARIA, contrast | diff touches UI |
| Performance | [reviewers/performance.md](reviewers/performance.md) | N+1, hot paths, network, caching, rendering | diff touches data access / rendering |
| Red team | [reviewers/red-team.md](reviewers/red-team.md) | what the others missed | always, runs last |

**The red team runs last:** dispatch it after the other lenses return, and
hand it their merged findings so it hunts for what they missed, not what they
found.

**Each lens returns machine-parseable findings** (JSON, per the template) so
the triage step can normalize them uniformly. A lens that finds nothing
returns an empty list — that is a valid, clean result for that lens.

## Triage

Do this yourself — the lenses file findings, you render the verdict. They
lack the full context to grade severity.

1. **Normalize.** Pull every finding from every lens into one list. Each
   carries: `location` (file:line), the finding, its **source** (lens name),
   and the lens's own evidence.
2. **Deduplicate by fingerprint** (`path:line:category`). Findings from two
   or more lenses pointing at the same defect collapse into one entry, tagged
   `MULTI-REVIEWER CONFIRMED (a + b)`. Multi-confirmation is a strong signal
   the finding is real.
3. **Group by root cause.** Two findings belong in one entry only when the
   same defect produced both — same location alone is not enough, and neither
   is a shared fix. The entry's severity is the highest of its members;
   `source` is the contributing lenses joined with `+`.
4. **Verify each entry at its location.** Read the cited `file:line` and read
   beyond it — follow callers, upstream guards, the surrounding code — until
   you can answer: does the bad outcome actually occur? Judge whether the
   problem is real, not whether the proposed fix is plausible. (Edge-case and
   verification-gap findings arrive with their own traced evidence; trust that
   evidence as filed and confirm the cited location matches.)
5. **Render exactly one verdict per entry** from the six-value table above.

**Reject** (drop) any entry that is `false`; any `low` whose fix adds more
complexity than the harm it removes; or any entry whose only fix is to edit
the spec under review. `could-not-verify` entries are **never dropped** — they
route to a human look / held-out test, because a silent pass on a check you
couldn't infer is the failure mode this skill exists to catch.

## Output Contract

**1. Report the round.** Present the lens reports side by side, verbatim or
lightly cleaned, each under its own heading, ending each with the finding
count + worst issue *within* that lens. Never pick a single cross-lens winner
— that's the reranking the separation exists to prevent. A change can be
spec-perfect but violate every convention, beautifully written but implement
the wrong thing, or conventionally clean and spec-complete but one SQLi away
from a data breach. Then the deduplicated/grouped entries with their verdicts
and source tags, and what was fixed vs deferred vs needs a human look. If any
lens failed to run, say so before the verdict.

**2. Write the audit record.** Fill [templates/review.md](templates/review.md)
and write it to `.skillgrid/change-specs/<topic>/review.md`, next to
`report.md` (which owns the QA gate; `review.md` owns this review). It
records the lenses side by side, the explicit "What Important means"
threshold, the capped Minor findings, each lens's **independence grade** (the
no-leaks proof), and the floor verdict. **Commit it in the spec zone** before
continuing code-zone work — a review that lives only in the transcript is not
an audit record, and a Grade C review that never got written down is even
harder to audit later.

**3. Route into the fix loop.** Route every surviving entry into
`skillgrid:receiving-code-review`'s triage buckets — **fix now** (real,
in-scope) / **defer** (real, later) / **human look** (needs manual
inspection) / **noise** (won't-fix). The mapping: `high`/`medium` that are
in-scope → fix now; `high`/`medium` that are pre-existing or out of scope →
defer; anything you can't confirm → human look; `low`/`maybe-false` you chose
to drop → noise. Then run the receiving-code-review fix loop: fix the
in-scope set one at a time with tests, log the rest, validate, and loop until
clean or capped per `_shared/planning/rigor-tiers.md` (fix loop cap). Accepted
fixes route to `skillgrid:receiving-code-review` — the fix loop keeps its
role; this skill renders the verdict, it never writes production fixes.

**4. Record the review signal.** In the change's `tasks.md` `## Review`,
record exactly one verdict: **`REVIEW-PASS`** (no unresolved `high`/`medium`
entries) or **`BACK-TO-APPLY`** (unresolved entries itemized with
`file:line` + evidence), plus the finding counts —
`Critical N · Warning N · Suggestion N`. This is the contract
`skillgrid:ship` consumes: its Verdict rules read the verdict and the counts,
and a prior `REVIEW-PASS` (or an accepted `BACK-TO-APPLY`) is reused, not
re-reviewed.

**5. Record a ledger round.** Per `_shared/craft/review-ledger.md`: freeze the
candidate, then append a round entry to the change's `review-ledger.md` with
the findings table (severities normalized to `CRITICAL` / `WARNING` /
`SUGGESTION`), dispositions (fixed / parked / waived), and lineage. Commit the
ledger with the review round.

**6. Update ticket status (if tracked):** if `ticketing.enabled: true` AND the
reviewed work corresponds to a ticket in `tasks.md` with a tracker ID: set the
ticket status → `done`; if all tickets in the epic are `done`: set the epic
status → `done`. Use the tracker adapter from `skillgrid:ticketing`
(Backlog.md CLI, `gh`, `glab`, `jira`).

**7. Update `state.yaml`:** set `changes.<topic>.phase: review`.

## Example

```
[Just completed Task 2: Add verification function]

You: Let me request code review before proceeding.

BASE_SHA=$(git merge-base main HEAD)
HEAD_SHA=$(git rev-parse HEAD)
git diff --stat → 412 changed lines, touches data access

[Topology: ≥50 changed lines + data access → core five + Performance, red team last]

[Dispatch six lenses in parallel: Standards, Spec, Security, Edge cases,
 Verification gaps, Performance — each fresh, each gets the diff range + its
 template's sources + the Blindness Contract, nothing from the session]

[Verification gaps returns]
  [MEDIUM] Index repair untested at the corruption boundary — held-out test
  would settle it (could-not-verify candidate)

[Security returns]
  [HIGH] SQLi in /search (CWE-89, OWASP A05:2025) — Confirmed

[Red team runs last with the merged findings]
  [MEDIUM] The two findings share a root cause — both hit the same
  unvalidated input path

You: [verify each entry at its location, render one verdict per entry]
  [Route into the fix loop via skillgrid:receiving-code-review; fix with tests]
  [Write review.md (spec zone), record the ledger round, record REVIEW-PASS +
   counts in tasks.md]
[Continue to Task 3]
```

## How to measure it

Per `_shared/craft/measurement.md`.

| | Indicator | Data source | Direction |
|---|-----------|-------------|-----------|
| Leading | Time from PR open to first review `review.md` commit | Git history | should fall |
| Lagging | Defects caught pre-merge (in `review.md`) vs. escaping to an incident | `review.md` findings + incident records | caught/rising, escaped/falling |

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "I'll just review the diff myself instead of dispatching" | You're the coordinator — reviewing the diff inline burns the context window you need to keep driving the work. Dispatch a lens subagent: the diff and the evaluation live in its context, and only the findings come back to you. An inline self-review is Grade C — diagnostic, not independent evidence. |
| "The reviewer needs my whole session history to understand the change" | Hand it precisely crafted context, never your session's history. That keeps the reviewer on the work product, not your thought process. |
| "The reviewer is a fresh subagent, so its grade is obviously A" | Fresh context is necessary but not sufficient. If it shares the implementer's model family or toolchain it's Grade B; if your prior verdict or rationale leaked in it's Grade C (diagnostic only). Name the grade before you treat it as independent. |
| "One good reviewer is enough; fan-out is overkill" | For a large or risky diff, one reviewer's context is the bottleneck — it can't hold the whole diff *and* every lens at full depth. Lenses run in parallel, each fresh; the diff lives in their context, only findings come back. |
| "I'll grade severity as I read the findings" | The lens that saw the hunk doesn't have your full context. You verify at the location and render the verdict; the lens only files the candidate. |
| "Two lenses flagged the same line, so it's twice as bad" | No. Dedup it into one entry, tag it MULTI-REVIEWER CONFIRMED, and grade it once. Confirmation raises your confidence it's real; it doesn't double the severity. |
| "The red team is redundant if the others ran" | The red team reads the *merged* findings and hunts the gaps — cross-cutting and integration-boundary issues that per-lens checklists don't cover. It runs last, with the others' output in hand. |

## Red Flags

**Never:**
- Skip review because "it's simple"
- Ignore Critical issues
- Proceed with unfixed Important issues
- Argue with valid technical feedback
- Claim a clean review when a lens failed or returned empty
- Let a lens's self-assigned severity be the final verdict
- Merge two findings just because they share a file or a fix
- Skip the red team on a high-risk diff because "the others looked fine"
- Re-grade a verified verification-gap finding — trust its filed evidence,
  confirm the location
- Confidently pass a check you can't infer from the diff — route it to
  `could-not-verify` and name the held-out test that would settle it

**If a lens is wrong:**
- Push back with technical reasoning
- Show code/tests that prove it works
- Request clarification

## Verification

- [ ] The topology step ran: the diff was pinned and sized (`git diff --stat`), the fan-out selected from size/risk/tier, and the selection is named in the report — including any lens skipped for lack of surface
- [ ] Every selected lens actually ran and returned findings (an empty list counts; a failed/timed-out lens is logged by name)
- [ ] Each lens's request included the diff range plus the context its template names — and **nothing from your session narrative or a prior verdict** (Blindness Contract)
- [ ] Each review is labeled with an independence grade (A/B/C); any Grade C review is reported as diagnostic only, not as independent evidence for ship/merge
- [ ] Findings from all lenses were collected and normalized into one list — not just one lens's output
- [ ] Every entry was verified at its cited `file:line` and rendered exactly one of the six verdicts; none was dropped on a silent pass
- [ ] The reports were presented side by side under per-lens headings, NOT merged or reranked
- [ ] `review.md` was written to `.skillgrid/change-specs/<topic>/review.md` and committed (spec zone) — the durable audit record with the lenses side by side, the "What Important means" threshold, capped Minor findings, each lens's independence grade, and the floor verdict
- [ ] The review signal is recorded in `tasks.md`: one verdict (`REVIEW-PASS` / `BACK-TO-APPLY`) + `Critical N · Warning N · Suggestion N` counts
- [ ] A ledger round entry was appended per `_shared/craft/review-ledger.md` (frozen candidate, findings table, dispositions) and committed with the review round
- [ ] The surviving entries were routed into a `skillgrid:receiving-code-review` bucket and the in-scope set fixed with tests — not left as "reviewed, good to go"
- [ ] No "clean review" was claimed while any selected lens failed or returned empty

## References

- `reviewers/` — one template per lens: `code-reviewer.md` (Standards),
  `spec-reviewer.md` (Spec), `security.md` (Security, OWASP 5-step workflow +
  triage rubric), `edge-case-hunter.md`, `verification-gap.md`,
  `accessibility.md`, `performance.md`, `red-team.md`
- [templates/review.md](templates/review.md) — the audit record this skill
  writes into the change's spec zone
- `_shared/planning/rigor-tiers.md` — tier floors, review escalation
  threshold, review budget, fix loop cap
- `_shared/craft/review-ledger.md` — the ledger round contract
- `_shared/verification/calibration.md` — the independence-grade protocol
- `_shared/verification/floor.md` — the floor verdict rule
- `_shared/rules/code-standards.md` — the two-axis contract + smell baseline
  (canonical)
- `skillgrid:receiving-code-review` — the fix loop this skill routes into
- `skillgrid:qa` — the quality gate; answers a different question
