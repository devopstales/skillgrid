# Report — <topic>

> Change: `.skillgrid/archive/YYYY-MM-DD-<topic>/` (moved from `.skillgrid/specs/`)
> Generated: <ISO 8601 timestamp>
> Status: success / blocked
> Variant: full / fast-track (trivial|small)
> Report reflects FINAL state at close (per Final-State Authority), not intermediate snapshots.

## Final-State Facts

**Shipped:** <what actually shipped>
**Base branch:** <base-branch> · **Chain strategy:** <strategy>
**Integration:** <merged @ <commit> | PR <url> | kept branch <name>> (from ship context)

## Gates

| Gate | Result |
|------|--------|
| Ship gate | ✅ success + `diff -r` empty (or: blocked — {reason}) |
| QA gate | ✅ {PASS \| WAIVED \| CONCERNS — human override recorded} |
| Verdict gate (advisory) | {accepted \| accepted-with-open-items \| rejected} — recorded, not enforced |

## Decisions

**Every entry MUST cite a source** — a file:line, a commit, a ticket ID, or a scenario name. No source = undiagnosed, not a learning.

| Decision | Tradeoff | Why | Source |
|----------|----------|-----|--------|
| <choice made> | <what was given up> | <why it won> | <file:line / commit / ADR / ticket> |

## Lessons

| Lesson | Root Cause | Do Differently | Source |
|--------|-----------|----------------|--------|
| <what to do differently> | <why it went wrong — root, not symptom> | <the concrete change> | <file:line / commit / ticket> |

## Patterns

| Pattern | Reuse | Source |
|---------|-------|--------|
| <named reusable approach / convention> | <when a future change should apply it> | <file:line / commit / ticket> |

## Surprises

| Surprise | Signal | Source |
|----------|--------|--------|
| <non-obvious gotcha / edge case / behavior> | <what the evidence shows> | <file:line / commit / scenario> |

## Acceptance Verdict

**Verdict:** accepted | accepted-with-open-items | rejected

**Grounding:** <goal from briefing.md + the qa-report.md Goal-Backward / Traceability evidence that supports the verdict>

**Reasoning:** <2–3 sentences: what drove the verdict>

## Open Items (→ next change)

- <item + path to resolution> (or "None")

## Prior-Change Follow-Through

| Prior open item (from previous archived change) | Addressed by this change? | Evidence |
|--------------------------------------------------|---------------------------|----------|
| <item> | yes / no / partially | <commit / file:line / scenario, or "still open"> |

## Move Evidence (from ship context)

**From:** `.skillgrid/specs/YYYY-MM-DD-<topic>/` → **To:** `.skillgrid/archive/YYYY-MM-DD-<topic>/`
**`diff -r` readback:** <empty → PASS | verbatim output → FAIL>

## Overrides / Waivers / Contradictions

- <QA waiver, human CONCERNS override, size-exception, or "None">
- <Unrankable contradiction: both statements + sources + when written — or "None">

## Lineage (observation IDs)

Every artifact read this close, for traceability:

- briefing: {id}
- blueprint: {id}
- tasks: {id}
- ship: {id}
- qa-report: {id}
- research / findings / ADRs: {ids or "none"}
