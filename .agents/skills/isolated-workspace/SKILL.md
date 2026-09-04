---
name: isolated-workspace
description: Use when starting feature work that needs isolation from current workspace or before executing implementation plans - ensures an isolated workspace exists via native tools or git worktree fallback
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
---

# isolated-workspace

**Core principle:** if the user wants isolation, help them build it — detect what exists, prefer a native tool, fall back to manual git, and verify a clean baseline before handing off. If the user does not want isolation (or has already declined), work in place. This is a *preference the skill helps honor*, not a gate the skill enforces.

Use this skill **when the user asks for isolation**, when the plan's chain strategy requires it, when the current branch is dirty and the work must not land on it, or when the work is risky enough that an isolated branch makes rollback trivial. Otherwise, do not use it — do not create a worktree for a small one-line edit.

When invoked:

1. Detect existing isolation first (do not recreate if you are already in one).
2. Prefer a native worktree tool when the runtime has one.
3. Fall back to `git worktree add`, then verify the target directory is ignored, then create.
4. Install dependencies and verify a clean baseline **in the workspace you ended up in** (worktree or in-place — either way, prove the tests pass before starting work).

## Step 0 — Detect existing isolation

Before creating anything, check whether you are already in an isolated workspace:

```bash
GIT_DIR=$(cd "$(git rev-parse --git-dir)" 2>/dev/null && pwd -P)
GIT_COMMON=$(cd "$(git rev-parse --git-common-dir)" 2>/dev/null && pwd -P)
BRANCH=$(git branch --show-current)
```

**Submodule guard.** `GIT_DIR != GIT_COMMON` is also true inside a git submodule. Before concluding "already in a worktree," verify you are not in a submodule:

```bash
# If this prints a path, you are in a submodule, not a worktree — treat as a normal repo.
git rev-parse --show-superproject-working-tree 2>/dev/null
```

- **`GIT_DIR != GIT_COMMON` (and not a submodule):** already in a linked worktree. Skip to Step 2. Do **not** create another. Report state:
  - On a branch: "Already in isolated workspace at `<path>` on branch `<name>`."
  - Detached HEAD: "Already in isolated workspace at `<path>` (detached HEAD, externally managed). Branch creation needed at finish time."
- **`GIT_DIR == GIT_COMMON` (or a submodule):** normal checkout. If the user's instructions already declare a worktree preference, honor it without asking. Otherwise ask consent:

  > "Would you like me to set up an isolated worktree? It protects your current branch from changes."

  If the user declines, work in place and skip to Step 2.

## Step 1 — Create the isolated workspace

Try, in this order. Stop at the first that works.

### 1a. Native worktree tool (preferred)

If your runtime offers a worktree primitive — e.g. an `agent_manager` worktree session, an `EnterWorktree`-style tool, a `/worktree` command, or a `--worktree` flag — use it and skip to Step 2.

Native tools own directory placement, branch creation, and cleanup. Running `git worktree add` when a native tool exists creates **phantom state the harness cannot see or manage** — the #1 mistake in this skill. Only proceed to 1b when no native tool is available.

### 1b. Git worktree fallback

Only if 1a does not apply.

**Directory selection** (explicit user preference always wins):

1. A declared worktree directory in your instructions — use it.
2. An existing project-local worktree dir (`ls -d .worktrees` first, then `worktrees`). If both exist, `.worktrees` wins.
3. Otherwise default to `.worktrees/` at the project root.

**Safety check (project-local dir only) — REQUIRED:**

```bash
git check-ignore -q .worktrees 2>/dev/null || git check-ignore -q worktrees 2>/dev/null
```

If **not** ignored: add the directory to `.gitignore`, commit it, then continue. An un-ignored worktree directory lets worktree contents get committed into the repository.

**Create:**

```bash
path="$LOCATION/$BRANCH_NAME"
git worktree add "$path" -b "$BRANCH_NAME"
cd "$path"
```

For an SDD change, name the branch after the change so the isolation is traceable: e.g. `feat/<NNN-slug>` or the chain-strategy branch from the `Review Workload Forecast` in `tasks.md`.

**Sandbox fallback:** if `git worktree add` fails with a permission error, tell the user the sandbox blocked worktree creation and you will work in the current directory instead; then run setup and the baseline in place.

## Step 2 — Project setup

Install dependencies for the detected stack:

```bash
[ -f package.json ]    && npm install
[ -f Cargo.toml ]      && cargo build
[ -f requirements.txt ] && pip install -r requirements.txt
[ -f pyproject.toml ]  && python -m poetry install 2>/dev/null
[ -f go.mod ]          && go mod download
```

Run only the lines that match project files present.

## Step 3 — Verify a clean baseline

Run the project's test suite before any work:

```bash
# npm test | cargo test | pytest | go test ./...  (whichever the project uses)
```

- **Failing:** report the failures and ask whether to proceed (proceed means "the baseline is already red — my changes may be blamed for pre-existing failures") or to first investigate via the `debugging` skill.
- **Passing:** report ready.

Report which kind of workspace you are in:

```
# if you created a worktree:
Worktree ready at <full-path> on branch <name>
Baseline: <N> tests passing, 0 failures
Ready to implement <feature-name>.

# if you are working in place (by preference or after decline):
Working in place on branch <name> (no worktree created)
Baseline: <N> tests passing, 0 failures
Ready to implement <feature-name>.
```

## Where it fits in SDD

- **Only when the change needs it.** A chained/stacked PR strategy or a dirty shared branch makes an isolated worktree worthwhile; a small change does not. This is a judgment the skill supports, not a rule it imposes.
- When you do isolate, record the workspace in Mnemonic so a resumed session finds it: `mem_save(title: "sdd/<NNN-slug>/workspace", topic_key: "sdd/<NNN-slug>/workspace", type: "config", scope: "project", content: "path: ...\nbranch: ...\nbaseline: <N> tests @ <sha7>")`.
- At finish: `sdd-archive` or the execution route (`simple-execution` / `subagent-execution`) handles merge and cleanup — this skill only **creates** the isolation (when appropriate) and proves the baseline.

## Finish mirror (hand off to `finishing-a-development-branch`)

This skill is the **up-front** half of worktree lifecycle. The **close-out** half lives in `finishing-a-development-branch`. When the SDD change reaches the close-out step (all `### Verification` verdicts in `tasks.md` are PASS or PASS WITH WARNINGS, all `[x]` marks in place):

- `finishing-a-development-branch` re-runs the env detection (`GIT_DIR` / `GIT_COMMON` / `WORKTREE_PATH` — capture **before** it changes directory for cleanup).
- Cleanup ownership: worktrees under `.worktrees/` or `worktrees/` are *ours* to remove; everything else belongs to the host.
- **Tests-on-the-integrated-tree** is mandatory, not ceremonial — a green run only proves the tree it ran on. The `sdd-verify` runs (verdicts in `tasks.md`) were against the branch's tree, not the merged result.
- Never `--force` on removal; never `discard` on anything but the typed word.

## Rules

- **Check before creating.** If you will make a worktree, confirm you are not already in one — do not stack a worktree over a worktree. (This is about the *creation step*, not about forcing a worktree.)
- **Prefer a native tool when the runtime has one** — it owns placement, branching, and cleanup; manual `git worktree add` creates state the harness can't see.
- **Before a manual create into a project-local dir, check it's ignored** — an un-ignored local dir is a data-loss path.
- **Prove the baseline is green in the workspace you ended up in** (worktree or in-place) before handing off, or the user must explicitly accept a red baseline.
- **This skill does not merge or push** — it prepares a workspace; it does not move history.

## Gotchas

- **Submodules fool the check.** `GIT_DIR != GIT_COMMON` is true in a submodule too — run the `--show-superproject-working-tree` guard before concluding you are in a worktree.
- **`git worktree add` under a native tool creates phantom state.** The harness then cannot see the branch, merge it, or clean it up. Prefer the native tool.
- **An un-ignored local worktree dir is a data-loss path** (contents get committed). `git check-ignore` before every manual create.
- **A red baseline makes every later failure ambiguous.** Run it now; if it is red and the user proceeds anyway, say so in every later report so nobody blames new work.
- **Detached HEAD / externally-managed worktrees** need a branch created at finish time — note it in the report so the finish step does not lose the work.

## References

- [../simple-execution/SKILL.md](../simple-execution/SKILL.md) / [../subagent-execution/SKILL.md](../subagent-execution/SKILL.md) — the execution routes that run after this skill establishes the branch and clean up at finish.
- [../_shared/conventions/commits.md](../_shared/conventions/commits.md) — the commit contract the isolated branch's commits follow.
