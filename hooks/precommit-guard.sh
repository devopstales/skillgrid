#!/usr/bin/env bash
# precommit-guard.sh — commit-time safety assertions for skillgrid work units.
#
# Called by checkpoint-state.sh (subcommands `guard` and `post-check`) from the
# repo's .git/hooks/pre-commit shim. Exits non-zero (with a RECOVERY line) on
# any guard failure so the hook blocks the commit.
#
# CLI-ready: the hooks are one-line shims over checkpoint-state.sh; skillgrid-cli
# rebinds the shims to its binary later. These assertions are portable and have
# no dependency on a specific harness.
#
# Usage:
#   precommit-guard.sh guard        # pre-commit: cwd-drift + protected-ref/detached
#   precommit-guard.sh post-check   # post-commit (run manually after committing):
#                                   #   warn on unexpected file deletions in HEAD
#
# Env:
#   SKILLGRID_AGENT_BRANCH_REGEX  positive allow-list for per-agent branches
#                                 (default: ^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$)
#   SKILLGRID_PROTECTED_BRANCHES  deny-list (default: main master develop trunk release/)
set -euo pipefail

SUBCMD="${1:-guard}"

is_worktree() { [ -f .git ]; }   # worktree: .git is a file pointing at the main repo

# --- cwd-drift assertion (worktree mode) -------------------------------------
# A prior Bash call may have `cd`'d out of the worktree into the main repo. When
# that happens `.git` is a directory (main repo) not a file, so `is_worktree`
# would silently skip every worktree guard. Capture the spawn-time toplevel via
# a sentinel on the first check, then verify on every subsequent check.
check_cwd_drift() {
  local wt_git_dir
  wt_git_dir="$(git rev-parse --git-dir 2>/dev/null)" || return 0
  case "$wt_git_dir" in
    *.git/worktrees/*)
      local sentinel expected_tl actual_tl
      sentinel="$wt_git_dir/skillgrid-spawn-toplevel"
      [ -f "$sentinel" ] || git rev-parse --show-toplevel > "$sentinel" 2>/dev/null
      expected_tl="$(cat "$sentinel" 2>/dev/null)"
      actual_tl="$(git rev-parse --show-toplevel 2>/dev/null)"
      if [ -n "$expected_tl" ] && [ "$actual_tl" != "$expected_tl" ]; then
        echo "FATAL: cwd drifted from spawn-time worktree root." >&2
        echo "  Spawn-time: $expected_tl" >&2
        echo "  Current:    $actual_tl" >&2
        echo "RECOVERY: cd \"$expected_tl\" before staging, then re-run the commit." >&2
        exit 1
      fi
      ;;
  esac
}

# --- protected-ref / detached-HEAD assertion (all repos) -----------------------
# Never commit on a protected ref or a detached HEAD. Applies to every repo
# (matches the "never commit on main without consent" rule); only the per-agent
# allow-list at the end is worktree-specific. We do NOT self-recover via
# `git update-ref` — surface it as a blocker instead.
check_protected_ref() {
  local head_ref actual_branch
  head_ref="$(git symbolic-ref --quiet HEAD || echo DETACHED)"
  # On an unborn branch (initial commit) `--abbrev-ref HEAD` errors under
  # set -e; fall back to the branch name from symbolic-ref, then to "HEAD".
  actual_branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)"
  if [ -z "$actual_branch" ]; then
    actual_branch="${head_ref##refs/heads/}"
    [ "$actual_branch" = "HEAD" ] && actual_branch="HEAD"
  fi

  # First commit of the repo: no HEAD yet. Allow it — the protected-ref rule
  # bites once history exists (you should be on a per-agent branch by then).
  if [ "$actual_branch" = "HEAD" ]; then
    return 0
  fi

  if [ "$head_ref" = "DETACHED" ]; then
    echo "FATAL: refusing to commit — worktree HEAD is detached (expected a per-agent branch)." >&2
    echo "RECOVERY: create/checkout a per-agent branch, e.g. git checkout -b agent-<id>." >&2
    exit 1
  fi

  local protected="${SKILLGRID_PROTECTED_BRANCHES:-main master develop trunk release/}"
  for p in $protected; do
    if [ "$actual_branch" = "$p" ]; then
      echo "FATAL: refusing to commit — HEAD is on protected branch '$actual_branch'." >&2
      echo "RECOVERY: commit on a per-agent branch, not '$actual_branch'." >&2
      exit 1
    fi
  done

  # Positive allow-list: in a worktree the branch must be a per-agent branch.
  # Only enforced in worktrees (a normal repo commits on its feature branch).
  if is_worktree; then
    local allow="${SKILLGRID_AGENT_BRANCH_REGEX:-^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$}"
    if ! echo "$actual_branch" | grep -Eq "$allow"; then
      echo "FATAL: refusing to commit — worktree HEAD '$actual_branch' is not a per-agent branch." >&2
      echo "  Allowed: agent-*, worktree-agent-*, worktree-wf_* (override: SKILLGRID_AGENT_BRANCH_REGEX)." >&2
      echo "RECOVERY: checkout a per-agent branch, e.g. git checkout -b agent-<id>." >&2
      exit 1
    fi
  fi
}

# --- post-commit deletion check -----------------------------------------------
# After a commit, verify it did not accidentally delete tracked files. Intentional
# deletions are expected and the caller documents them; here we only WARN (a
# deletion in a work-unit commit is worth a glance, not a hard block).
post_check_deletions() {
  local deletions
  deletions="$(git diff --diff-filter=D --name-only HEAD~1 HEAD 2>/dev/null || true)"
  if [ -n "$deletions" ]; then
    echo "WARNING: commit $(git rev-parse --short HEAD) includes file deletions:" >&2
    echo "$deletions" | sed 's/^/  - /' >&2
    echo "Intentional? Document them in the commit's [skillgrid-context] block; otherwise revert and fix." >&2
  fi
}

case "$SUBCMD" in
  guard)
    # `if` (not `&&`) so a non-worktree repo doesn't trip set -e at the
    # last command of the case body.
    # cwd-drift is worktree-only; protected-ref/detached applies to all repos.
    if is_worktree; then check_cwd_drift; fi
    check_protected_ref
    ;;
  post-check)
    post_check_deletions
    ;;
  *)
    echo "unknown subcommand: $SUBCMD (expected guard|post-check)" >&2
    exit 2
    ;;
esac
