# Ship Report — <topic>

> Change: `.skillgrid/archive/YYYY-MM-DD-<topic>/` (moved from `.skillgrid/specs/`)
> Generated: <ISO 8601 timestamp>
> Status: success / blocked
> Variant: full / fast-track (trivial|small)

## Base Branch & Delivery Strategy

**Base branch:** <base-branch>
**Chain strategy:** <stacked-to-main | feature-branch-chain | size-exception | pending>
**Work unit (final/tracker):** <unit name + its base boundary, from tasks.md work-unit table>

## QA Gate

**Verdict:** <PASS | WAIVED | CONCERNS | FAIL>
**Override:** <None | human "ship it anyway" (name + date) | waiver: what + who + risk>
**Open items (if CONCERNS):**
- <item + path to resolution>

## Review Gate (advisory)

**Verdict:** <REVIEW-PASS | waived | BACK-TO-APPLY (unresolved: list)>

## Ship Decision

**Verdict:** <GO | NO-GO>
**Basis:** <QA <verdict> + review <verdict> + <human override, if any>>
**Rollback plan:**
- <trigger condition> → <procedure, e.g. `git revert <merge-commit> -m 1`>
- <…>

## Integration Test Evidence

**Command:** `<testing.runner>`
**Ran on:** <integrated tree — branch/commit>
**Result:** <N passed, M failed> → <green | red>

## Integration Outcome

**Choice:** <merged to <base> @ <commit> | PR <url> | kept branch <name>>

**PR body source (if PR):** <briefing.md goal + blueprint.md approach + tasks.md scope + qa-report.md gate>

## Worktree

**State:** <cleaned up @ <path> | preserved @ <path> | host-managed | normal repo (no worktree)>

## Mechanical Move to Archive

**From:** `.skillgrid/specs/YYYY-MM-DD-<topic>/`
**To:** `.skillgrid/archive/YYYY-MM-DD-<topic>/`
**Method:** <git mv | mv>
**`diff -r` readback:** <empty → PASS | verbatim output → FAIL>

## Overrides / Waivers

<QA waiver, human CONCERNS override, size-exception, or "None">
