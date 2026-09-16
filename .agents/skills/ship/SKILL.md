---
name: ship
description: "Use when a change has passed qa and review and you need to integrate it to its base branch and close its change folder. The integration + archive-move step of the tail (qa → review → ship → reflect)."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
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

Ship is the close-out phase of the tail (`qa → review → ship → reflect`): it integrates verified work to its base branch, mechanically archives the change folder, and renders a GO / NO-GO ship decision with a mandatory rollback plan. It matters because shipping is where unverified work leaks into the base branch and the archive — the point of no return. The core principle: **do not merge, push, or move the folder until the test suite is green on the integrated tree.**

## When to Use

- When a change has passed QA/verification and is ready to be shipped/merged to its base branch.
- At the end of the skillgrid flow, before closing the change folder and handing off to `reflect`.

**When NOT to use:** mid-change before the gates pass — ship is the final phase, not a mid-task step.

## What You Receive

- **Change folder:** `.skillgrid/specs/YYYY-MM-DD-<topic>/` (briefing, blueprint, `tasks.md`, `qa-report.md`, review artifacts).
- **`qa-report.md` verdict** — the hard gate (PASS / CONCERNS / FAIL / WAIVED) and any human override.
- **`tasks.md` → `## Delivery Strategy`** — the four plain-text guard lines (`Decision needed before apply:`, `Chained PRs recommended:`, `Chain strategy:`, `400-line budget risk:`) and the work-unit table. `Chain strategy:` resolves the base branch.
- **Fast-track waiver class** (`trivial` / `small`) if present — drives the light variant.

## Phase Order

```
qa → review → ship → reflect
```

`prev-phase: [review]` · `next-phase: [reflect]` · `artifact: none` (the `report.md` is written by reflect in the archive).

## Status and Workspace Guard

Before any integration or move:

- If you are in a **read-only planning workspace** or the change folder is outside the allowed edit roots, STOP and report — do not merge or move across an unsafe boundary.
- If `qa-report.md` is `FAIL` or has an unresolved CRITICAL with no human override, STOP — return `blocked` with the reason. You cannot ship work the quality gate refused.

## Gates (all must pass before ANY mutation)

### QA Gate (hard — never overridable by a prompt claim)

- `qa-report.md` **verdict `PASS`** → proceed.
- **verdict `WAIVED`** → proceed; record the waiver (what was waived, by whom, risk accepted) in the Return Envelope (reflect persists it to `report.md`).
- **verdict `CONCERNS`** → **advisory**: surface the open items, the human owns the decision. Proceed only on an explicit human "ship it anyway" — record that override in the Return Envelope (reflect persists it to `report.md`).
- **verdict `FAIL`** or an unresolved **CRITICAL** → **`blocked`** (reason `qa-failed`). A launch-prompt claim "it's fixed in a later commit" does not clear it — a fresh passing `qa` run is required.

### Review Gate (advisory — never blocks on its own)

- No review verdict in `tasks.md` → record `review-waived`; **Step 9 runs the fan-out** on the diff to obtain one before rendering the GO/NO-GO (skip only if the change is trivial).
- Verdict `BACK-TO-APPLY` with unresolved items → surface as a WARNING, proceed (the human accepted this state).
- Verdict `REVIEW-PASS` → proceed (reuse; Step 9 does not re-review).

If a hard gate fails, **STOP and return `blocked`** with the failing gate named. Do not integrate, do not move.

## What to Do

### Step 1: Verify Tests on the Integrated Tree

Run the project's full test suite (`testing.runner` from `config.yaml`).

**If tests fail**, report the failures and stop — the menu comes after a green suite:

```
Tests failing (<N> failures). Must fix before shipping:

[show failures]
```

**If tests pass:** continue to Step 2.

### Step 2: Detect Environment

```bash
GIT_DIR=$(cd "$(git rev-parse --git-dir)" 2>/dev/null && pwd -P)
GIT_COMMON=$(cd "$(git rev-parse --git-common-dir)" 2>/dev/null && pwd -P)
# Capture now, while still inside the workspace — Step 5 changes directory
# before cleanup (Step 7) needs this value.
WORKTREE_PATH=$(git rev-parse --show-toplevel)
```

This determines which menu to show and how cleanup works:

| State | Menu | Cleanup |
|-------|------|---------|
| `GIT_DIR == GIT_COMMON` (normal repo) | Standard 3 options | No worktree to clean up |
| `GIT_DIR != GIT_COMMON`, named branch | Standard 3 options | Provenance-based (Step 7) |
| `GIT_DIR != GIT_COMMON`, detached HEAD | Reduced 2 options (no merge) | Externally managed — leave in place |

**Submodule guard:** before concluding "already in a worktree," verify you are not in a submodule:

```bash
git rev-parse --show-superproject-working-tree 2>/dev/null
```

If it prints a path, you are in a submodule — treat as a normal repo.

### Step 3: Determine Base Branch

The base branch is whatever this work forked from. Resolve it from `tasks.md` `## Delivery Strategy` → **`Chain strategy:`**:

| `Chain strategy:` | Base branch |
|---|---|
| `stacked-to-main` | `main` (each PR merges to main in order) |
| `feature-branch-chain` | The work-unit table's **base boundary** for the final work unit (PR #1 base = main/tracker; PR #2 base = PR #1 branch; …). Ship integrates the **tracker** (last) unit to its base boundary. |
| `size-exception` | The single PR's target (usually `main`) — record the accepted exception. |
| `pending` | Not decided — **ask** the user which chain strategy / base to use before proceeding. |

If it is still not known, ask: "This branch split from `<your best guess>` — is that correct?"

**Confirm before merging: merging into the wrong base is expensive to undo.**

### Step 4: Present Options

**Normal repo and named-branch worktree — present exactly these 3 options:**

```
Implementation complete. What would you like to do?

1. Merge back to <base-branch> locally
2. Push and create a Pull Request
3. Keep the branch as-is (I'll handle it later)

Which option?
```

**Detached HEAD — present exactly these 2 options:**

```
Implementation complete. You're on a detached HEAD (externally managed workspace).

1. Push as new branch and create a Pull Request
2. Keep as-is (I'll handle it later)

Which option?
```

Present the menu exactly as written. Discarding the work happens only in response to an explicit request to throw the work away (below). **The integration decision is the user's** — wait for the answer.

> **Fast-track variant (`trivial` / `small`):** the menu still appears, but the expected answer is merge (or PR); there is no PR-body generation step and the reflect `report.md` is the light form (one verdict line + the move readback). No release mechanics, no docs check — in any variant.

### Step 5: Execute Choice

**Option 1 — Merge Locally:**

```bash
MAIN_ROOT=$(git -C "$(git rev-parse --git-common-dir)/.." rev-parse --show-toplevel)
cd "$MAIN_ROOT"

# Merge first — verify success before removing anything.
git checkout <base-branch>
git pull
git merge <feature-branch>

# Verify tests on the merged result.
<test command>
```

If tests fail on the merged result: stop, leave the worktree and branch in place, and investigate — nothing has been pushed, so the merge is local and recoverable.

Once the merged result is green: clean up the worktree (Step 7), then delete the branch:

```bash
git branch -d <feature-branch>
```

**Option 2 — Push and Create PR:**

```bash
git push -u origin <feature-branch>
# From a detached HEAD, name the new branch on the remote:
# git push origin HEAD:refs/heads/<new-branch>
```

Then create the pull/merge request against `<base-branch>` with the forge's tooling (its CLI if available, or the creation URL most forges print on push), following the repo's PR template if present, and report the URL. The PR body is generated from the planning artifacts: the change's `briefing.md` (goal) + `blueprint.md` (approach) + `tasks.md` (what shipped) + `qa-report.md` (gate + evidence). **Keep the worktree** — the user iterates on PR feedback there.

**Option 3 — Keep As-Is:**

Report: "Keeping branch `<name>`. Worktree preserved at `<path>`."

**If the user asks to discard the work** (explicit request only). Confirm first:

```
This will permanently delete:
- Branch <name>
- All commits: <commit-list>
- Worktree at <path>

Type 'discard' to confirm.
```

Wait for that exact confirmation. Then clean up the worktree (Step 7) and force-delete the branch:

```bash
git branch -D <feature-branch>
```

### Step 6: Capture Ship Context (for reflect)

Record the following in your working context (it goes into reflect's `report.md`):

- Base branch + chain strategy + work-unit table reference.
- Integration test evidence (the green run on the integrated tree).
- PR/merge outcome (URL, or merge commit, or "kept").
- Worktree state (cleaned up / preserved / host-managed).
- Gate results (QA verdict + any waiver/override; review status).

These are passed to `skillgrid:reflect` via the Return Envelope.

### Step 7: Mechanical Move to Archive (LAST step)

The change folder moves from `specs/` (active) to a top-level `archive/` (closed historical record). This is a **mechanical filesystem operation** — file content MUST NEVER pass through the model's Read/Write path. The only acceptable move is a native shell command (`git mv` / `mv`), verified by a structural `diff -r` readback.

```bash
# Run as ONE shell transaction so the EXIT trap stays active.
# The snapshot is recursive and MUST be created BEFORE the move.
snapshot_root="$(mktemp -d "${TMPDIR:-/tmp}/ship.move.XXXXXX")"
trap 'rm -rf -- "$snapshot_root"' EXIT
cp -R ".skillgrid/specs/YYYY-MM-DD-<topic>" "$snapshot_root/source"

# Mechanical move (MANDATORY): git mv when tracked, mv otherwise.
mkdir -p .skillgrid/archive
if ! git mv ".skillgrid/specs/YYYY-MM-DD-<topic>" ".skillgrid/archive/YYYY-MM-DD-<topic>"; then
  mv ".skillgrid/specs/YYYY-MM-DD-<topic>" ".skillgrid/archive/YYYY-MM-DD-<topic>" || exit $?
fi

# The source must be gone before comparing the archived tree to its snapshot.
if [ -e ".skillgrid/specs/YYYY-MM-DD-<topic>" ] || [ -L ".skillgrid/specs/YYYY-MM-DD-<topic>" ]; then
  printf 'archive move left the source directory in place\n' >&2; exit 1
fi

# MANDATORY readback: only an empty diff passes.
diff -r "$snapshot_root/source" ".skillgrid/archive/YYYY-MM-DD-<topic>"
diff_status=$?
[ "$diff_status" -ne 0 ] && exit "$diff_status"
```

Use **today's date in ISO format** (`YYYY-MM-DD`) — it is already the folder's name prefix. Compare the archived folder against the **pre-move** recursive snapshot — do not substitute a model readback, a staged tree, or the post-move source. Any non-empty `diff -r` output or non-zero status is truncation, alteration, or an operational failure — it **FAILS** the phase.

> **If shell access is unavailable**, STOP and report `blocked` with reason `shell access required for mechanical archive move is unavailable` — do **not** fall back to Read/Write moving.

### Step 8: Persist to Mnemonic (hybrid)

The filesystem move is already done (Step 7); persist the ship context as an index/backup so reflect can resume if interrupted.

```
mem_save(
  title:      "ship — YYYY-MM-DD-<topic>",
  topic_key:  "skillgrid/YYYY-MM-DD-<topic>/ship",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "{ship context: base branch, chain strategy, integration test evidence, PR/merge outcome, worktree state, gate results, diff -r readback}"
)
```

> If `mnemonic.enabled` is `false`, the in-repo artifacts + git history are the sole record — skip this step (degrade explicitly, never fail silently).

### Step 9: Render the Ship Decision

The integration and the move are mechanical; the **decision** is the artifact. Synthesize the gates into a single **GO / NO-GO**, and write the **rollback plan** — both go into the Return Envelope (reflect persists them to `report.md` in the archive).

**Gather the review signal first.** If the change is **non-trivial** and the review gate above recorded `review-waived` (no prior `parallel-code-review` verdict in `tasks.md`), run **skillgrid:parallel-code-review** on the diff now, before rendering the verdict — the ship decision should rest on a fresh fan-out, not an absence of one. Its output (`REVIEW-PASS` / `BACK-TO-APPLY` + Critical/Warn counts) feeds Verdict rules 2 and 3. Two cases skip the fan-out:

- **Trivial change** (≤ 2 files, < 50 lines, and touches no auth / payments / data migration / config) → no fan-out; the decision rests on the QA gate + a one-line rollback. This is the explicit **skip threshold**.
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

If the change is **trivial** (≤ 2 files, < 50 lines, and touches no auth / payments / data migration / config), the rollback plan may be one line: "revert the single commit." The full plan is required otherwise.

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
**Next**: reflect — writes `report.md` (integration + retro + archive summary) + session close
```

Close the final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up. Do **not** call `mem_session_summary` in a sub-agent context — `reflect` owns session close.

## Rules

- You integrate and close **verified** work. A `qa-report` `FAIL` or unresolved CRITICAL blocks ship — no prompt override; require a fresh passing `qa` run.
- **Do not merge, push, or move until tests are green on the integrated tree.** A green run only proves the tree it ran on.
- **The move is mechanical.** `cp -R`/`git mv`/`mv` via the shell only, NEVER model Read/Write — and a `diff -r` readback (empty = only passing evidence, verbatim in the result) after it. A skipped or non-empty `diff -r` FAILS the phase.
- **The move is the LAST step** — everything that must be in the archive (qa-report, review artifacts) is already written when the folder leaves `specs/`. The `report.md` is written by reflect *after* the move, in the archive.
- **The archive is an audit trail** — `.skillgrid/archive/` is created if absent; archived changes are never deleted or modified after the move.
- The integration decision (merge / PR / keep / discard) is the user's — present the menu and wait. Discard happens only on the typed word `discard`.
- No release mechanics (no VERSION/CHANGELOG bump), no docs check (no Diataxis coverage) — in any variant.
- If shell access is unavailable, STOP and report `blocked` — do not fall back to Read/Write moving.
- Return envelope per Step 9 — final action is text, not a tool call. `reflect` is next.

## Gotchas

- **"Tests passed earlier this session."** Run the suite on the tree you are about to integrate. A green run only proves the tree it ran on.
- **"The base branch is obviously main."** Resolve it from `Chain strategy:` + the work-unit table, or confirm/ask. Merging into the wrong base is expensive to undo. For `feature-branch-chain`, ship integrates the *tracker* (last) unit to *its* base boundary — not necessarily `main`.
- **`Chain strategy: pending`** means the base is undecided — ask before Step 4, don't guess.
- **Reflect writes `report.md` in the archive** — ship does not write a report file; it captures context for the Return Envelope, and reflect persists it.
- **Moving before the source is gone-check** gives a meaningless empty diff. The `diff -r` compares the *pre-move snapshot* to the archived folder, after confirming the source path no longer exists.
- **`--force` on a refused worktree removal** destroys files that exist only in that worktree. Show the user `git -C "$WORKTREE_PATH" status --porcelain -uall` and ask; never `--force` on your own initiative.
- **Clean up only worktrees under `.worktrees/` or `worktrees/`.** Everything else belongs to the host.
- **`CONCERNS` is not a block.** It is advisory — surface open items, the human decides. But `FAIL` / unresolved CRITICAL is a hard block.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "The gates are probably fine, ship it." | A `FAIL` or unresolved CRITICAL in `qa-report.md` is a hard block — a launch-prompt claim "it's fixed in a later commit" does not clear it. A fresh passing `qa` run is required. |
| "I'll skip the workspace guard, it's fast." | In a read-only planning workspace or with the change folder outside the allowed edit roots, STOP and report — do not merge or move across an unsafe boundary. |
| "I'll ship without the final review." | If the review gate recorded `review-waived` and the change is non-trivial, Step 9 runs the fan-out on the diff before rendering the GO/NO-GO. Skipping it means the verdict rests on an absence of signal. |
| "Tests passed earlier this session, that's enough." | A green run only proves the tree it ran on. `qa` ran against the feature tree — run the suite again on the tree you are about to integrate. |
| "I'll write the report in the archive after the move." | That's what reflect does — ship captures context in the Return Envelope, reflect writes `report.md` in the archive. Ship writes no report file. |

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

- [`../qa/SKILL.md`](../qa/SKILL.md) — upstream; its `qa-report.md` verdict drives the QA Gate.
- [`../slicing/SKILL.md`](../slicing/SKILL.md) — upstream; the `## Delivery Strategy` guard lines + work-unit table resolve the base branch.
- [`../reflect/SKILL.md`](../reflect/SKILL.md) — next; reads the moved folder from `.skillgrid/archive/`, writes `report.md`, owns session close.
- [`../_shared/conventions/fast-track.md`](../_shared/conventions/fast-track.md) — the light-variant waiver this phase honors.
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — the `specs/` → `archive/` layout, phase order, and Mnemonic slots.
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active `session_id`).
