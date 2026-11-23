---
name: git-discipline
description: Use when a phase transition touches git state that another phase depends on — a spec-zone change must reach the base before execution starts, the execution branch must merge back before archive, a fan-out integration must land before the next wave, a tracker PR must exist before the next slice. Enforces the state table and exact phrasing at each gate.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: intent-driven-template:openspec-git-discipline 1.0 (adapted to skillgrid's phase model)
  related_skills: [execute, ship, slicing, brainstorm, blueprint]
---

## When to Use

This skill fires at **phase-transition gates** — the moments where the
state of git determines whether the next phase can proceed. It is **not**
about commit format or commit timing (those live in
`_shared/work-unit-commits/README.md`); it is about **phase-ordering
invariants**: the spec must reach the base before code starts, code must
reach the base before the change is archived, one phase's git state is a
precondition for the next phase's work.

**Announce at start:** "I'm using the skillgrid:git-discipline skill to
check the git state before this phase transition."

The gates below are **hard** — the skill halts at a failed gate, names the
missing state, and uses the prescribed phrasing. The gates are not
suggestions and do not degrade to warnings.

## The State Table

Each row is a **gate**: a precondition the skill checks before the
transition proceeds. The check is deterministic (read git); the action is
halt + phrasing.

| Gate | Moment | Check | Action on fail |
|---|---|---|---|
| G1: spec-reached-base | Before `execute` Step 0 dispatches Task 1 | The change's spec-zone artifacts (`briefing.md`, `blueprint.md`, `acceptance.feature`, `tasks.md`, `adr.md`) are committed on the base branch (the change's base per `chain.strategy` — usually `main`), not only on a feature branch or worktree | Halt + phrasing |
| G2: no-mixed-uncommitted | Before `execute` commits a task | No staged file is both in `<specs_root>` (default `.skillgrid/change-specs/`) and outside it. A `tasks.md` progress-mark commit is allowed to stage only `tasks.md` (zone-neutral) | Halt + phrasing |
| G3: on-per-agent-branch | Before any `git commit` in an execution worktree | Current branch matches the per-agent allow-list (`agent-*`, `worktree-agent-*`, `worktree-wf_*`) — enforced by the `pre-commit` hook | FATAL (hook fires; this gate is the prose mirror) |
| G4: spec-committed-before-code | Before a task commit that touches both a spec file and a code file | The spec file is already committed (not staged for the same commit) — the spec is the contract, the code satisfies it | Halt + phrasing |
| G5: fan-out-integrated | Before the next wave in `execute` fan-out mode | Every previous wave's worktrees have been merged into the parent branch and the parent branch's tests pass; the fan-out ledger row for each integrated worktree reads `status: done` | Halt + phrasing |
| G6: impl-reached-base | Before `ship` Step 6 moves the change folder to `change-archive/` | The change's execution branch is merged into the base branch (not just present in a worktree); the base branch's full test suite passes | Halt + phrasing |
| G7: archive-from-base | At `ship` Step 6 | The shell's current branch is the base branch (not a feature branch, not a worktree branch, not detached HEAD) | Halt + phrasing |
| G8: spec-zone-committed | Before `ship` Step 6 | All spec-zone files under `<specs_root>/<topic>/` are committed (the zone is clean); the change's `report.md` QA verdict is filled | Halt + phrasing |
| G9: tracker-pr-exists | Before the next slice in a chained-PR strategy | The forge API confirms the previous slice's PR is open (for `feature-branch-chain`, the previous slice's PR target is the immediate parent branch, not the base) | Halt + phrasing |
| G10: no-phantom-merge | Before `ship` Step 6 cleanup deletes a worktree branch | The branch HEAD equals the base HEAD (no added changes) — per the no-added-changes rule in `_shared/work-unit-commits/README.md` | Halt + phrasing |

## Required Checks

Each gate names the deterministic check it runs. The checks are
read-only; they do not modify git state.

### G1: spec-reached-base

```bash
BASE=$(git remote show origin | sed -n "s/.*HEAD branch: //p")
# or the change's base per chain.strategy
git diff --name-only "$BASE" -- .skillgrid/change-specs/<topic>/
```

If the diff is non-empty, the spec zone has uncommitted changes that have
not reached the base. If the spec files exist on a worktree branch but not
on the base:

> I should not start execution yet because the change's spec-zone
> artifacts have not reached `<base>`. The spec must be committed on the
> base before execution begins, so the execution branch has a stable
> contract to satisfy. Commit and push the spec-zone artifacts to
> `<base>` first, then I can dispatch Task 1.

### G2: no-mixed-uncommitted

```bash
STAGED=$(git diff --cached --name-only)
SPEC_STAGED=$(echo "$STAGED" | grep -E '^\.skillgrid/change-specs/')
CODE_STAGED=$(echo "$STAGED" | grep -vE '^\.skillgrid/change-specs/' | grep -vE '^\.skillgrid/projectwiki/' | grep -vE '^\.skillgrid/work/')
```

If both are non-empty, the commit is mixed.

> I should not commit this yet because the staged set mixes spec-zone
> files (`<list>`) and code-zone files (`<list>`). The zone rule in
> `_shared/work-unit-commits` blocks a mixed commit: commit the spec
> files first, then the code files. A `tasks.md` progress-mark commit is
> zone-neutral and may be committed with either.

### G4: spec-committed-before-code

```bash
# For each task that touches a spec file:
git log -1 --format=%H -- .skillgrid/change-specs/<topic>/acceptance.feature
# must return a SHA that predates the code commit about to be made
```

If the spec file's last commit is newer than the most recent code commit
on the current branch, the spec is committed and the gate passes. If not:

> I should not commit the code yet because the spec file it satisfies
> (`<path>`) is not yet committed on this branch. Commit the spec
> change first — the spec is the contract, the code satisfies it —
> then commit the code.

### G5: fan-out-integrated

```bash
# For each previous-wave worktree branch:
git merge-base --is-ancestor <worktree-branch> <parent-branch>
# and read the fan-out ledger row: status must be `done`
```

If a previous-wave branch is not an ancestor of the parent, or the
ledger row is not `done`:

> I should not start the next wave yet because the previous wave's
> integration is incomplete: `<branch>` is not merged into
> `<parent>` (or the fan-out ledger row for `<worktree>` is not
> `done`). Integrate the previous wave first — merge, run the full
> suite, update the ledger — then dispatch the next wave.

### G6: impl-reached-base

```bash
git merge-base --is-ancestor <execution-branch> <base>
# and the base branch's test suite passes
```

If the execution branch is not an ancestor of the base:

> I should not archive this change yet because the execution branch
> `<branch>` has not been merged into `<base>`. Merging is the
> integration; a worktree with green tests is not a merge. Merge the
> branch to `<base>` and run the full suite on the base, then archive.

### G7: archive-from-base

```bash
CURRENT=$(git branch --show-current)
[ "$CURRENT" = "$BASE" ]
```

If the shell is not on the base:

> I should not archive from here because archive runs from the base
> branch (`<base>`) and the shell is on `<current>`. Switch to
> `<base>` and re-run the archive step.

### G8: spec-zone-committed

```bash
git diff --name-only -- .skillgrid/change-specs/<topic>/
git diff --cached --name-only -- .skillgrid/change-specs/<topic>/
# both must be empty
```

If the spec zone has uncommitted changes:

> I should not archive yet because the change's spec zone
> (`.skillgrid/change-specs/<topic>/`) has uncommitted files: `<list>`.
> Commit the spec-zone changes (or revert them with a ruling) before
> the archive move.

### G9: tracker-pr-exists

```bash
# via the forge API (gh pr list / glab mr list):
gh pr list --head <prev-slice-branch> --state open
# must return at least one PR
```

If no open PR exists for the previous slice:

> I should not start the next slice yet because the previous slice's
> PR (`<prev-slice-branch>`) is not open on the forge. In a
> `<chain-strategy>` chain, each slice's PR must exist before the
> next slice starts — the next slice targets the previous slice's
> branch (for `feature-branch-chain`) and the chain breaks if the
> previous PR is missing. Open the previous slice's PR first.

### G10: no-phantom-merge

```bash
git rev-parse <branch>
git rev-parse <base>
# equal, OR git merge-base --is-ancestor <branch> <base> AND git diff --quiet <base>...<branch>
```

If the branch would add changes to the base:

> I should not delete `<branch>` yet because it has unmerged changes
> relative to `<base>`: `<diff-stat>`. The no-added-changes rule in
> `_shared/work-unit-commits` keeps an unmerged branch as the
> rollback boundary. Merge it to `<base>` first, or keep the branch
> and note the reason in the ship report.

## Red Flags

These are the failure modes the skill is designed to catch. When any is
detected, halt and use the corresponding phrasing above.

- **Applying a spec that exists only on the current worktree/branch** —
  the spec is not on the base; G1 fails.
- **A commit that stages both a spec file and a code file** — the zone
  rule is violated; G2 fails.
- **A `tasks.md` progress-mark commit that stages a code file alongside
  it** — `tasks.md` is zone-neutral, but the code file is not; G2
  fails (only `tasks.md` may be staged).
- **Starting the next fan-out wave before the previous wave is merged** —
  G5 fails; the integration order breaks.
- **Archiving from a feature branch** — G7 fails; the archive would move
  the folder while the shell is off-base.
- **Archiving while the spec zone has uncommitted changes** — G8 fails;
  the archive would carry an uncommitted spec into the archive folder.
- **The next slice of a chain starting before the previous slice's PR is
  open** — G9 fails; the chain target is missing.
- **Deleting a worktree branch that would add changes to the base** —
  G10 fails; the rollback boundary is destroyed.
- **Treating a worktree's green tests as proof the change reached the
  base** — a worktree is a checkout, not a merge; G6 fails.
- **Auto-committing or auto-merging without the user's explicit
  approval** — the skill checks state, it does not mutate it; the
  mutation is the user's (or the `ship` skill's, at its named step).

## Anti-Patterns

| Anti-pattern | Symptom | Fix |
|---|---|---|
| G1 as a suggestion | "The spec is on the worktree, let's start anyway" | G1 is a halt, not a warning. The spec must reach the base before Task 1 dispatches. |
| G2 "just this once" | "The code change is tiny, I'll stage it with the spec" | The zone rule is binary. Split the commit: spec first, code second. |
| G5 "the tests passed" | "The previous wave's worktree has green tests, we're fine" | Green tests in a worktree are not a merge. G5 checks ancestry + ledger status, not the test output. |
| G6 "worktree = merged" | "The worktree is up to date with the base" | A worktree is a checkout. G6 checks `merge-base --is-ancestor` from the branch to the base. |
| G7 "I'm on the right branch" (the shell says otherwise) | The agent believes it is on the base; `git branch --show-current` says otherwise | G7 reads the shell's actual branch. The agent's belief is not the check. |
| G9 "the PR will be there soon" | "I'll open the PR right after I start the next slice" | The chain breaks if the next slice's base is missing. G9 checks the forge, not the plan. |
| G10 "it's a small branch" | "The branch is 3 lines, just delete it" | The no-added-changes rule does not have a size exemption. 3 lines unmerged is 3 lines lost on revert. |

## How to measure it

Per `_shared/craft/measurement.md`.

| | Indicator | Data source | Direction |
|---|-----------|-------------|-----------|
| Leading | Count of `git-discipline` halt events per executed change (each halt is a ledger line) | `.skillgrid/work/<change>/progress.md` halt lines | should fall |
| Lagging | Count of post-archive discoveries where a spec/code zone mix or an unmerged branch is found after the move | `change-archive/<change>/report.md` retro half, "Final-State Facts" section | should fall |

## Verification

- [ ] The gate that fired was named in the halt message (G1–G10)
- [ ] The deterministic check (the bash command in the Required Checks
      section) was actually run and its output is in the ledger
- [ ] The phrasing matches the gate's prescribed wording (or the skill
      states why a deviation was needed)
- [ ] No gate was "waived" without a named human and a recorded ruling in
      the ledger
- [ ] The skill did not mutate git state — it checked, halted, and
      phrased; the mutation is the user's or a downstream skill's

## References

- Commit format, atomic sizing, when-to-commit, the
  `[skillgrid-context]` block: `_shared/work-unit-commits/README.md`
- The `pre-commit` zone guard (the machine-enforcement of G2):
  `_shared/work-unit-commits/README.md` → The two layers → hooks
- The no-added-changes rule (the machine-enforcement of G10):
  `_shared/work-unit-commits/README.md` → Worktree removal + branch
  deletion
- The chain strategy (G9's context): `_shared/rules/chain-strategy.md`
- The phase order (which phase is "next"): `_shared/rules/sdd-structure.md`
- The user gate contract (how the halt is presented):
  `_shared/rules/user-gates.md`
