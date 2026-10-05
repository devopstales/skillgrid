---
name: ship
description: "Use when a change has passed qa and review and you need to integrate it to its base branch and close its change folder. The integration and archive-move step after review."
license: MIT
metadata:
  author: devopstales
  version: "1.1"
  part-of: skillgrid
---

# Ship

**Announce at start:** "I'm using the skillgrid:ship skill to integrate and close this change."

You are the **INTEGRATION + ARCHIVE-MOVE** phase — the close-out step after `qa` and `review`, before the terminal `reflect`. You do two things:

1. **Integrate the work** to its base branch (merge / PR / keep) — proven green on the *integrated* tree.
2. **Close the change folder** — move `.skillgrid/specs/YYYY-MM-DD-<topic>/` to `.skillgrid/archive/YYYY-MM-DD-<topic>/` with a mechanical, verifiable move.

The ship decision (GO / NO-GO + rollback plan) and the final `report.md` are written by `skillgrid:reflect` in the archived folder. Ship renders the decision in its Return Envelope; reflect persists it.

**Iron law:**

```
DO NOT MERGE, PUSH, OR MOVE THE FOLDER UNTIL THE TEST SUITE IS GREEN ON THE INTEGRATED TREE
```

A green run only proves the tree it ran on. `qa` ran against the feature tree — run the suite again on the tree you are about to integrate.

## Overview

Ship integrates verified work to its base branch, mechanically archives the change folder, and renders a GO / NO-GO ship decision with a mandatory rollback plan. It matters because shipping is where unverified work leaks into the base branch and the archive — the point of no return. The core principle: **do not merge, push, or move the folder until the test suite is green on the integrated tree.** The full phase order is in `_shared/rules/sdd-structure.md`.

## When to Use

- When a change has passed QA/verification and is ready to be shipped/merged to its base branch.
- At the end of the skillgrid flow, before closing the change folder and handing off to `reflect`.

**When NOT to use:** mid-change before the gates pass — ship is the final phase, not a mid-task step.

## What You Receive

- **Change folder:** `.skillgrid/specs/YYYY-MM-DD-<topic>/` (briefing, blueprint, `tasks.md`, `report.md` with the QA half complete, `review.md` — the committed code-review audit record).
- **`report.md` → `## Gate Decision` verdict** — the hard gate (PASS / CONCERNS / FAIL / WAIVED) and any human override.
- **`review.md` → `## Verdict` floor** — the review gate (met / met-with-fixes / not met, the floor across both axes) + the independence grade; `not met` or a Grade C axis blocks the same way an unresolved CRITICAL does.
- **`tasks.md` → `## Delivery Strategy`** — the four plain-text guard lines (`Decision needed before apply:`, `Chained PRs recommended:`, `Chain strategy:`, `400-line budget risk:`) and the work-unit table. `Chain strategy:` resolves the base branch.
- **Fast-track waiver class** (`trivial` / `small`) if present — drives the light variant.

## Phase Order

After `receiving-code-review`, before `reflect`. The full chain is in `_shared/rules/sdd-structure.md`.

`prev-phase: [receiving-code-review]` · `next-phase: [reflect]` · `artifact: none` (the `report.md` is written by reflect in the archive).

## Status and Workspace Guard

Before any integration or move:

- If you are in a **read-only planning workspace** or the change folder is outside the allowed edit roots, STOP and report — do not merge or move across an unsafe boundary.
- If `report.md` → `## Gate Decision` is `FAIL` or has an unresolved CRITICAL with no human override, STOP — return `blocked` with the reason. You cannot ship work the quality gate refused.

## Gates (all must pass before ANY mutation)

### QA Gate (hard — never overridable by a prompt claim)

Read `report.md` → `## Gate Decision` → `**Verdict:**` line, **PRE-MOVE** (the file is still in `specs/`):

- **verdict `PASS`** → proceed.
- **verdict `WAIVED`** → proceed; record the waiver (what was waived, by whom, risk accepted) in the Return Envelope (reflect persists it in the report's retro half).
- **verdict `CONCERNS`** → **advisory**: surface the open items, the human owns the decision. Proceed only on an explicit human "ship it anyway" — record that override in the Return Envelope (reflect persists it in the report's retro half).
- **verdict `FAIL`** or an unresolved **CRITICAL** → **`blocked`** (reason `qa-failed`). A launch-prompt claim "it's fixed in a later commit" does not clear it — a fresh passing `qa` run is required.

### Review Gate (advisory — never blocks on its own)

- No review verdict in `tasks.md` → record `review-waived`; **Step 9 runs the fan-out** on the diff to obtain one before rendering the GO/NO-GO (skip only if the change is trivial).
- Verdict `BACK-TO-APPLY` with unresolved items → surface as a WARNING, proceed (the human accepted this state).
- Verdict `REVIEW-PASS` → proceed (reuse; Step 9 does not re-review).

If a hard gate fails, **STOP and return `blocked`** with the failing gate named. Do not integrate, do not move.

## What to Do

Integration steps (integrated-tree tests, environment, base branch, the menu, archive move, reconcile, mnemonic): [integrate.md](references/integrate.md).

### Step 9: Render the Ship Decision

The integration and the move are mechanical; the **decision** is the artifact. Synthesize the gates into a single **GO / NO-GO**, and write the **rollback plan** — both go into the Return Envelope (reflect persists them to `report.md` in the archive).

**Gather the review signal first.** If the change is **non-trivial** and the review gate above recorded `review-waived` (no prior `parallel-code-review` verdict in `tasks.md`), run **skillgrid:parallel-code-review** on the diff now, before rendering the verdict — the ship decision should rest on a fresh fan-out, not an absence of one. Its output (`REVIEW-PASS` / `BACK-TO-APPLY` + Critical/Warn counts) feeds Verdict rules 2 and 3. Two cases skip the fan-out:

- **Trivial change** (the trivial-skip rule per `_shared/planning/rigor-tiers.md`, review escalation threshold) → no fan-out; the decision rests on the QA gate + a one-line rollback. This is the explicit **skip threshold**.
- **Prior verdict exists** (`REVIEW-PASS` or an accepted `BACK-TO-APPLY` already in `tasks.md`) → reuse it; don't re-review.

Fan-out selection, per-file lenses, and the red-team pass all follow `parallel-code-review/SKILL.md` — ship does not re-implement them. Record the fan-out result in `tasks.md` so the verdict below cites a real signal.

**Verdict rules (apply in order; the first that matches wins):**

1. **QA `FAIL`, or an unresolved CRITICAL** → **NO-GO**. Nothing downstream overrides a hard gate.
2. **Any Critical review finding** → **NO-GO**, unless the human explicitly accepted the risk (record the acceptance).
3. **QA `CONCERNS` or a `BACK-TO-APPLY` review verdict** → **NO-GO** by default; the human can override to GO. Record the override and what was accepted.
4. **Everything else (QA `PASS`/`WAIVED`, review clean or waived)** → **GO**, with the rollback plan attached.

**Rollback plan (mandatory before any GO).** Name the trigger conditions and the procedure:

- **Merge already landed** → `git revert <merge-commit> -m 1` (or `git reset --hard <pre-merge> ` only if nothing else has landed on the base). Verify tests green after the revert.
- **PR open, not merged** → close the PR; the branch stays until the work lands or is re-based.
- **Branch kept** → the branch is the rollback (it never left the repo).
- **Folder already moved to `archive/`** → the move is the *last* step, so a NO-GO discovered before the move leaves it untouched. A NO-GO discovered *after* the move is corrected in the next change's `reflect`, not by un-archiving.

If the change is **trivial** (per the trivial-skip rule in `_shared/planning/rigor-tiers.md`, review escalation threshold), the rollback plan may be one line: "revert the single commit." The full plan is required otherwise.

### Step 10: Return Envelope

**Your FINAL output MUST be text — not a tool call.** Do the `mem_save` (Step 8) *before* this text. Reflect reads this envelope to write `report.md`.

```markdown
## Shipped

**Change**: YYYY-MM-DD-<topic>
**Base branch**: {base-branch} · **Chain strategy**: {strategy}
**Integration**: {merged to {base} @ {commit} | PR {url} | kept branch {name}}
**Archived**: `.skillgrid/archive/YYYY-MM-DD-<topic>/` · Mnemonic `skillgrid/YYYY-MM-DD-<topic>/ship`
**Status**: success | blocked

### Ship Decision
**Verdict**: {GO | NO-GO}
**Basis**: {QA {verdict} + review {verdict} + {human override, if any}}
**Rollback plan**: {trigger → procedure, per the verdict's integration state}

### Gates
| Gate | Result |
|------|--------|
| QA gate | ✅ {PASS \| WAIVED \| CONCERNS — human override recorded} |
| Review gate (advisory) | ✅ {REVIEW-PASS \| waived \| BACK-TO-APPLY notes} |

### Integration Test Evidence
- {test command} on integrated tree: {N passed, M failed} → {green}

### Mechanical Move
- Pre-move snapshot → `git mv`/`mv` → `.skillgrid/archive/YYYY-MM-DD-<topic>/`
- `diff -r` readback: {empty → ✅ | verbatim output → FAIL}

### Overrides / Waivers
{QA waiver, human CONCERNS override, size-exception — or "None"}

**Mnemonic**: observation `{id or 'none'}` · session `{sid}`
**Open questions**: {list, or "None"}
**Open decision debt**: {assumed blueprints owed ratification — or "None"}
**Next**: reflect — completes `report.md`'s retro half in the archive (integration + retro + archive summary) + session close
```

**Update `state.yaml`:** append the shipped `<topic>` to `progress.completed_changes`, and clear `pipeline.current_change` + `pipeline.current_phase` (the pipeline is now idle, awaiting the next change or `reflect`).

**Release session lock:** run `node scripts/state-lock.mjs release <topic>`. Always exit 0 (no-op if the lock doesn't name this change).

Close the final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up. Do **not** call `mem_session_summary` in a sub-agent context — `reflect` owns session close.

## Rules

- You integrate and close **verified** work. A `FAIL` in `report.md` → `## Gate Decision` or an unresolved CRITICAL blocks ship — no prompt override; require a fresh passing `qa` run.
- **Do not merge, push, or move until tests are green on the integrated tree.** A green run only proves the tree it ran on.
- **The move is mechanical.** `cp -R`/`git mv`/`mv` via the shell only, NEVER model Read/Write — and a `diff -r` readback (empty = only passing evidence, verbatim in the result) after it. A skipped or non-empty `diff -r` FAILS the phase.
- **The move is the LAST step** — everything that must be in the archive (the QA half of `report.md`, review artifacts) is already written when the folder leaves `specs/`, so the `diff -r` readback is exactly empty. Reflect completes `report.md`'s retro half (from `## Final-State Facts` onward) in place *after* the move, in the archive.
- **The archive is an audit trail** — `.skillgrid/archive/` is created if absent; archived changes are never deleted or modified after the move, **except** the retro sections of `report.md` that reflect completes in place after the move (the QA half is pre-move and included in the `diff -r` readback; the retro half is appended after it).
- The integration decision (merge / PR / keep / discard) is the user's — present the menu and wait. Discard happens only on the typed word `discard`.
- No release mechanics (no VERSION/CHANGELOG bump), no docs check (no Diataxis coverage) — in any variant. The human-facing record (PR body, changelog, release note, postmortem) is written by `skillgrid:document`, which ship invokes for the PR body when the user chooses the PR option (and offers the other types where applicable).
- **Decision debt ships, it doesn't clear.** An `ASSUMED` blueprint flag is carried into the Return Envelope and the archive — never removed by shipping.
- If shell access is unavailable, STOP and report `blocked` — do not fall back to Read/Write moving.
- Return envelope per Step 9 — final action is text, not a tool call. `reflect` is next.

## Gotchas

- **"Tests passed earlier this session."** Run the suite on the tree you are about to integrate. A green run only proves the tree it ran on.
- **"The base branch is obviously main."** Resolve it from `Chain strategy:` + the work-unit table, or confirm/ask. Merging into the wrong base is expensive to undo. For `feature-branch-chain`, ship integrates the *tracker* (last) unit to *its* base boundary — not necessarily `main`.
- **`Chain strategy: pending`** means the base is undecided — ask before Step 4, don't guess.
- **Reflect completes `report.md` in the archive** — ship does not write the report; it captures context for the Return Envelope, and reflect completes the retro half in the archive in place.
- **Moving before the source is gone-check** gives a meaningless empty diff. The `diff -r` compares the *pre-move snapshot* to the archived folder, after confirming the source path no longer exists.
- **`--force` on a refused worktree removal** destroys files that exist only in that worktree. Show the user `git -C "$WORKTREE_PATH" status --porcelain -uall` and ask; never `--force` on your own initiative.
- **Clean up only worktrees under `.worktrees/` or `worktrees/`.** Everything else belongs to the host.
- **`CONCERNS` is not a block.** It is advisory — surface open items, the human decides. But `FAIL` / unresolved CRITICAL is a hard block.

## How to measure it

Per `_shared/craft/measurement.md`.

| | Indicator | Data source | Direction |
|---|-----------|-------------|-----------|
| Leading | Time from `report.md` `## Verdict` to merge commit | Git history | should fall |
| Lagging | Rollbacks or reverts within N days of the merge | Git history (revert commits) | should fall |

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "The gates are probably fine, ship it." | A `FAIL` or unresolved CRITICAL in `report.md` → `## Gate Decision` is a hard block — a launch-prompt claim "it's fixed in a later commit" does not clear it. A fresh passing `qa` run is required. |
| "I'll skip the workspace guard, it's fast." | In a read-only planning workspace or with the change folder outside the allowed edit roots, STOP and report — do not merge or move across an unsafe boundary. |
| "I'll ship without the final review." | If the review gate recorded `review-waived` and the change is non-trivial, Step 9 runs the fan-out on the diff before rendering the GO/NO-GO. Skipping it means the verdict rests on an absence of signal. |
| "Tests passed earlier this session, that's enough." | A green run only proves the tree it ran on. `qa` ran against the feature tree — run the suite again on the tree you are about to integrate. |
| "I'll write the report in the archive after the move." | That's what reflect does — ship captures context in the Return Envelope, reflect completes `report.md`'s retro half in the archive in place. Ship writes no report. |

## Red Flags

- A merge, push, or folder move happens before a fresh green test run on the integrated tree.
- The Return Envelope is missing the ship context (base branch, integration evidence, gate results) that reflect needs.
- The QA gate shows `FAIL` or an unresolved CRITICAL but the phase proceeds instead of returning `blocked`.
- The workspace guard was skipped in a read-only planning workspace or across an unsafe boundary.
- The archive move used model Read/Write instead of `git mv`/`mv`, or the `diff -r` readback is non-empty or absent.
- The ship decision renders a GO with no rollback plan.

## Verification

- [ ] All gates in `Gates (all must pass before ANY mutation)` show PASS (or a recorded waiver/override).
- [ ] Workspace/status guard confirms clean state — not a read-only workspace, change folder inside allowed edit roots, no `FAIL`/unresolved CRITICAL.
- [ ] The test suite is green on the integrated tree (fresh run, not the `qa` run).
- [ ] The archive move is mechanical (`git mv`/`mv`) with an empty `diff -r` readback against the pre-move snapshot.
- [ ] The change folder is at `.skillgrid/archive/YYYY-MM-DD-<topic>/` and the source is gone.
- [ ] The Return Envelope is rendered with a GO/NO-GO verdict, basis, rollback plan, and ship context (base branch, integration evidence, gate results) for reflect.

## References

- [integrate.md](references/integrate.md) — integrated-tree tests, the ship menu, the archive move, reconcile, and mnemonic.
- [`../../verification/qa/SKILL.md`](../../verification/qa/SKILL.md) — upstream; it wrote the QA half of `report.md`, whose `## Gate Decision` verdict drives the QA Gate (read pre-move).
- [`../../planning/slicing/SKILL.md`](../../planning/slicing/SKILL.md) — upstream; the `## Delivery Strategy` guard lines + work-unit table resolve the base branch.
- [`../reflect/SKILL.md`](../reflect/SKILL.md) — next; reads the moved folder from `.skillgrid/archive/`, completes `report.md`'s retro half in place, owns session close.
- [`../../_shared/planning/fast-track.md`](../../_shared/planning/fast-track.md) — the light-variant waiver this phase honors.
- [`../../_shared/rules/sdd-structure.md`](../../_shared/rules/sdd-structure.md) — the `specs/` → `archive/` layout, phase order, and Mnemonic slots.
- [`../../_shared/rules/mnemonic-memory.md`](../../_shared/rules/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active `session_id`).
