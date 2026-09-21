#!/usr/bin/env bash
# precommit-ignore-guard.sh — block (and auto-fix) commits of generated/scratch
# paths that must never be tracked.
#
# Two invariants:
#   - acceptance-test-authoring: acceptance-tests/.extracted/ is "gitignored,
#     wiped and rebuilt on every run, never edited by hand." A committed
#     extraction is a real bug (a stale one keeps deleted capabilities running).
#   - isolated-workspace: the worktree dir (conventions.worktree_dir, default
#     .worktrees/) MUST be gitignored before use — an unignored worktree commits
#     the whole tree into the repo.
#
# On a hit: if the path is NOT yet ignored, auto-add it to .gitignore (the
# "auto-git" behavior) and unstage it from the commit, then still FAIL the
# commit so the agent sees the fix. If it IS ignored but somehow staged, just
# FAIL.
#
# CLI-ready: called by checkpoint-state.sh guard (the pre-commit dispatcher).
set -euo pipefail

root="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0

# read a key from .skillgrid/config.yaml (first match), fallback if absent.
# The key is a simple trailing key like `worktree_dir:` (no dots needed).
config_value() {
  local key="$1" fallback="$2" file="$root/.skillgrid/config.yaml" val
  [ -f "$file" ] || { echo "$fallback"; return; }
  val="$(awk -v k="^[[:space:]]*"k":" '
    $0 ~ k { sub(/^[^:]*:[[:space:]]*/, ""); sub(/[[:space:]]*#.*$/, ""); sub(/["'\'']/g, ""); print; exit }
  ' "$file" 2>/dev/null || true)"
  [ -n "$val" ] && echo "$val" || echo "$fallback"
}

wt_dir="$(config_value worktree_dir .worktrees)"

staged="$(git diff --cached --name-only --diff-filter=ACMR 2>/dev/null || true)"
[ -n "$staged" ] || exit 0

fix_needed=0
for f in $staged; do
  case "$f" in
    acceptance-tests/.extracted/*) pat="acceptance-tests/.extracted/" ;;
    "${wt_dir}"/*|worktrees/*|.worktrees/*) pat="${wt_dir}/" ;;
    *) continue ;;
  esac

  # Is the directory currently ignored?
  if git check-ignore -q "$f" 2>/dev/null; then
    echo "WARNING: '$f' is gitignored but staged. Unstaging it." >&2
    git reset -q -N -- "$f" 2>/dev/null || git reset -q HEAD -- "$f" 2>/dev/null || true
    fix_needed=1
    continue
  fi

  # Not ignored: auto-add the dir to .gitignore.
  gi="$root/.gitignore"
  touch "$gi"
  if ! grep -qxF "$pat" "$gi" 2>/dev/null; then
    {
      [ -s "$gi" ] && [ -n "$(tail -c1 "$gi")" ] && echo ""
      echo "$pat"
    } >> "$gi"
    echo "NOTICE: auto-added '$pat' to .gitignore." >&2
  fi
  # Unstage the generated path so it isn't committed this time.
  git reset -q HEAD -- "$f" 2>/dev/null || git rm --cached -q -- "$f" 2>/dev/null || true
  fix_needed=1
done

if [ "$fix_needed" = 1 ]; then
  {
    echo "FATAL: generated/scratch path(s) were staged."
    echo "  .extracted/ is rebuilt every run; the worktree dir must stay untracked."
    echo "  I added the missing pattern to .gitignore and unstaged the path(s)."
    echo "RECOVERY: re-stage your real files and commit again. Stage .gitignore too if you want to record the new ignore rule."
  } >&2
  exit 1
fi
exit 0
