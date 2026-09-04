---
name: finishing-a-development-branch
description: Use when an SDD change is complete (all `### Verification` verdicts in `tasks.md` are PASS or PASS WITH WARNINGS, all `[x]` marks in place) and you need to decide how to integrate the work. Verifies tests on the integrated tree, detects environment, presents the merge/PR/keep menu, and owns worktree cleanup.
license: MIT
metadata:
  author: skillgrid
  version: "1.0"
  source: derived from superpowers finishing-a-development-branch, adapted for skillgrid's worktree conventions
---

# Finishing a Development Branch

**Core principle:** verify tests → detect environment → present options → execute choice → clean up.

**Announce at start:** "I'm using the finishing-a-development-branch skill to complete this work."

In skillgrid, this skill is the **close-out companion** of `isolated-workspace`:

- `isolated-workspace` *created* the branch and proved a green baseline at the start.
- `sdd-apply` + `sdd-verify` produced the commits and `tasks.md` Verification verdicts.
- `finishing-a-development-branch` is the final integration step before `sdd-archive` mechanically moves the change folder.

## Step 1: Verify Tests on the Integrated Tree

Run the project's full test suite (`npm test` / `cargo test` / `pytest` / `go test ./...`).

**If tests fail**, report the failures and stop — the menu comes after a green suite:

```
Tests failing (<N> failures). Must fix before completing:

[Show failures]
```

**Why this matters:** the `sdd-verify` runs (verdicts in `tasks.md`) were against the branch's tree, not the tree you are about to integrate. A green run only proves the tree it ran on. Run the suite again on the *integrated* tree (post-merge, or on the branch you are about to push).

**If tests pass:** continue to Step 2.

## Step 2: Detect Environment

```bash
GIT_DIR=$(cd "$(git rev-parse --git-dir)" 2>/dev/null && pwd -P)
GIT_COMMON=$(cd "$(git rev-parse --git-common-dir)" 2>/dev/null && pwd -P)
# Capture now, while still inside the workspace — Step 5 changes directory
# before cleanup (Step 6) needs this value
WORKTREE_PATH=$(git rev-parse --show-toplevel)
```

This determines which menu to show and how cleanup works:

| State | Menu | Cleanup |
|-------|------|---------|
| `GIT_DIR == GIT_COMMON` (normal repo) | Standard 3 options | No worktree to clean up |
| `GIT_DIR != GIT_COMMON`, named branch | Standard 3 options | Provenance-based (see Step 6) |
| `GIT_DIR != GIT_COMMON`, detached HEAD | Reduced 2 options (no merge) | Externally managed — leave in place |

**Submodule guard:** before concluding "already in a worktree," verify you are not in a submodule:

```bash
git rev-parse --show-superproject-working-tree 2>/dev/null
```

If it prints a path, you are in a submodule — treat as a normal repo.

## Step 3: Determine Base Branch

The base branch is whatever this work forked from — usually the worktree's tracking branch, the `isolated-workspace` instruction, or the chain strategy in the change's `Review Workload Forecast`. If it is not already known, ask: "This branch split from `<your best guess>` — is that correct?"

Confirm before merging: merging into the wrong base is expensive to undo.

## Step 4: Present Options

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

Present the menu exactly as written — concise, with every option coming from the list above. Discarding the work happens only in response to an explicit request to throw the work away (see "If the user asks to discard the work" below). Wait for their answer; the integration decision is theirs.

## Step 5: Execute Choice

### Option 1: Merge Locally

```bash
# Get main repo root for CWD safety
MAIN_ROOT=$(git -C "$(git rev-parse --git-common-dir)/.." rev-parse --show-toplevel)
cd "$MAIN_ROOT"

# Merge first — verify success before removing anything
git checkout <base-branch>
git pull
git merge <feature-branch>

# Verify tests on merged result
<test command>
```

If tests fail on the merged result: stop, leave the worktree and branch in place, and investigate — nothing has been pushed, so the merge is local and recoverable.

Once the merged result is green: clean up the worktree (Step 6), then delete the branch:

```bash
git branch -d <feature-branch>
```

### Option 2: Push and Create PR

```bash
git push -u origin <feature-branch>
# From a detached HEAD, name the new branch on the remote:
# git push origin HEAD:refs/heads/<new-branch>
```

Then create the pull/merge request against `<base-branch>` with the forge's tooling — its CLI if one is available, or the creation URL most forges print when you push — following the repo's PR template and conventions if present, and report the URL to the user.

Keep the worktree — the user iterates on PR feedback there.

### Option 3: Keep As-Is

Report: "Keeping branch `<name>`. Worktree preserved at `<path>`."

### If the user asks to discard the work

This path exists only as a response to an explicit request to throw the work away. Confirm first:

```
This will permanently delete:
- Branch <name>
- All commits: <commit-list>
- Worktree at <path>

Type 'discard' to confirm.
```

Wait for that exact confirmation. When it arrives:

```bash
MAIN_ROOT=$(git -C "$(git rev-parse --git-common-dir)/.." rev-parse --show-toplevel)
cd "$MAIN_ROOT"
```

Then clean up the worktree (Step 6) and force-delete the branch:

```bash
git branch -D <feature-branch>
```

## Step 6: Cleanup Workspace

**Runs for Option 1 and confirmed discards.** Options 2 and 3 always preserve the worktree. Both callers have already changed directory to the main repo root — worktree removal must run from outside the worktree — and use the `GIT_DIR` / `GIT_COMMON` / `WORKTREE_PATH` values captured in Step 2, from before that directory change.

**If `GIT_DIR == GIT_COMMON`:** normal repo, no worktree to clean up. Done.

**If `WORKTREE_PATH` is under `.worktrees/` or `worktrees/`:** the workspace's `isolated-workspace` step (or the runtime's native worktree tool) created this worktree — we own cleanup:

```bash
git worktree remove "$WORKTREE_PATH"
git worktree prune  # Self-healing: clean up any stale registrations
```

**If removal is refused** (`contains modified or untracked files`): the worktree holds files that exist nowhere else — uncommitted plans, notes, or scratch work. **Never `--force` on your own initiative.** Show the user what is at stake and ask:

```bash
git -C "$WORKTREE_PATH" status --porcelain -uall
```

```
Worktree removal refused — these files were never committed:

<file list>

1. Commit them to <branch> before cleanup
2. Move them into <main repo root>
3. Delete them (unrecoverable)

Which?
```

Carry out the choice, then remove the worktree.

**Otherwise:** the host environment owns this workspace — leave it in place. If your platform provides a workspace-exit tool, use it.

## Quick Reference

| Option | Merge | Push | Keep Worktree | Cleanup Branch |
|--------|-------|------|---------------|----------------|
| 1. Merge locally | yes | - | - | yes |
| 2. Create PR | - | yes | yes | - |
| 3. Keep as-is | - | - | yes | - |
| Discard (explicit request only) | - | - | - | yes (force) |

## Common Rationalizations

| Excuse | Reality |
|---|---|
| "Tests passed earlier this session" | Run the suite on the tree you are about to integrate. A green run only proves the tree it ran on. |
| "They obviously want it merged" | Integration is the user's decision. Present the menu and wait. |
| "They seem done with this feature — I'll offer to discard it" | The menu is complete as written. Discard happens only when the user asks for it in so many words. |
| "'Yeah, get rid of it' counts as confirmation" | Only the typed word `discard` authorizes deletion. |
| "The PR is up, so the worktree is clutter now" | PR feedback gets fixed in that worktree. It stays until the work lands. |
| "This other worktree looks stale — I'll clean it too" | Clean up only worktrees under `.worktrees/` or `worktrees/`. Everything else belongs to the host. |
| "Removal refused — `--force` is just finishing the cleanup" | The refusal means files exist only in that worktree. `--force` destroys them permanently. Show the user and ask. |
| "The merged-result failure is probably flaky" | A failing merged result stops everything. Branch and worktree stay put while you investigate. |
| "The base branch is obviously main" | Confirm the fork point or ask. Merging into the wrong base is expensive to undo. |
| "The push was rejected — force-push will fix it" | A rejected push means the remote moved. Investigate; force-push only on the user's explicit request. |

## Integration with skillgrid

- **`sdd-archive`** is the close-out *artifact* move (`docs/skillgrid/changes/<NNN-slug>/` → `archive/`). This skill is the close-out *integration* step (merge / PR / keep) that runs *before* archive. Order: `sdd-verify` (`tasks.md` verdicts) → `requesting-code-review` (high-risk) → **this skill** → `sdd-archive`.
- **`isolated-workspace`** is the up-front step that creates the branch and proves the baseline. This skill is its mirror at the end — the workspace came from there, the cleanup returns there.
- **For PR-backed changes**, the tracker convention lives in `_shared/issue-tracker/`; use the right CLI for the PR creation step (e.g. `gh pr create`, `glab mr create`).
- **For SDD worktrees created by the runtime's native worktree tool** (not `git worktree add`), prefer the runtime's cleanup primitive over `git worktree remove` — manual removal creates phantom state the harness cannot see.

## References

- [../isolated-workspace/SKILL.md](../isolated-workspace/SKILL.md) — the up-front mirror of this skill; owns creation, the green baseline, and the `.worktrees/` ownership contract.
- [../sdd-archive/SKILL.md](../sdd-archive/SKILL.md) — the close-out artifact move that runs *after* this skill.
- [../_shared/conventions/commits.md](../_shared/conventions/commits.md) — commit hygiene enforced by this skill's reviewers and the merge/PR step.
- [../_shared/issue-tracker/](../_shared/issue-tracker/) — tracker CLI choice for the PR-creation step.
