# Archive Report — <topic>

> Change: `.skillgrid/archive/YYYY-MM-DD-<topic>/` (moved from `.skillgrid/specs/`)
> Generated: <ISO 8601 timestamp>
> Status: success / blocked
> Report reflects FINAL state at close (per Final-State Authority), not intermediate snapshots.

## Final-State Facts

**Shipped:** <what actually shipped>
**Base branch:** <base-branch> · **Chain strategy:** <strategy>
**Integration:** <merged @ <commit> | PR <url> | kept branch <name>> (from ship-report.md)

## Gates

| Gate | Result |
|------|--------|
| Ship gate | ✅ success + `diff -r` empty (or: blocked — {reason}) |
| QA gate | ✅ {PASS \| WAIVED \| CONCERNS — human override recorded} |
| Verdict gate (advisory) | {accepted \| accepted-with-open-items \| rejected} — recorded, not enforced |

## Lineage (observation IDs)

Every artifact read this close, for traceability:

- blueprint: {id}
- tasks: {id}
- ship-report: {id}
- qa-report: {id}
- research / findings / ADRs: {ids or "none"}

## Move Evidence (from ship-report.md)

**From:** `.skillgrid/specs/YYYY-MM-DD-<topic>/` → **To:** `.skillgrid/archive/YYYY-MM-DD-<topic>/`
**`diff -r` readback:** <empty → PASS | verbatim output → FAIL>

## Overrides / Waivers / Contradictions

- <QA waiver, human CONCERNS override, size-exception, or "None">
- <Unrankable contradiction: both statements + sources + when written — or "None">

## Verdict (advisory)

**Accepted?** accepted | accepted-with-open-items | rejected
**Open items → next change:** <list or "None">
**Follow-through:** <prior open items addressed / still open, or "None">
