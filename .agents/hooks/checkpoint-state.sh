#!/usr/bin/env bash
# checkpoint-state.sh — skillgrid checkpoint CLI entrypoint (model B).
#
# The durable record lives in the commit (its [skillgrid-context] body block).
# The resume handle is .skillgrid/sdd/checkpoint.json, which is DERIVED from
# `git log -1` + the last [skillgrid-context] block so it can never drift from
# history. A fresh session reads it to resume; nothing is hand-maintained.
#
# CLI-ready: the repo's .git/hooks shims call this script. skillgrid-cli later
# rebinds the shims to its binary; the subcommand contract stays stable.
#
# Subcommands:
#   snapshot     derive + write .skillgrid/sdd/checkpoint.json from the last commit
#   restore      print the current resume state (for a fresh session)
#   guard        pre-commit guards (delegates to precommit-guard.sh)
#   guard-msg    validate a commit message file (conventional + no Co-Authored-By)
#   post-check   post-commit deletion check (delegates to precommit-guard.sh)
#
# Env:
#   SKILLGRID_CHECKPOINT_JSON  override the state file path
#   (defaults to <repo-root>/.skillgrid/sdd/checkpoint.json)
set -euo pipefail

SUBCMD="${1:-snapshot}"
shift || true

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GUARD="$SCRIPT_DIR/precommit-guard.sh"

repo_root() { git rev-parse --show-toplevel 2>/dev/null; }

state_file() {
  if [ -n "${SKILLGRID_CHECKPOINT_JSON:-}" ]; then
    echo "$SKILLGRID_CHECKPOINT_JSON"
  else
    local root; root="$(repo_root)"
    echo "$root/.skillgrid/sdd/checkpoint.json"
  fi
}

# Pull a field out of the last commit's [skillgrid-context] block.
# $1 = field name (Decisions/Remaining/Tried/Task/Commit)
context_field() {
  local field="$1" line
  line="$(git log -1 --pretty=%B 2>/dev/null | sed -n "/^\[skillgrid-context\]/,/\[\/skillgrid-context\]/p" | grep -E "^${field}:" || true)"
  # strip the "Field:" prefix, keep the rest
  printf '%s' "$line" | sed -E "s/^${field}:[[:space:]]*//"
}

now_iso() { date -u +%Y-%m-%dT%H:%M:%SZ; }

# Build the JSON body as a single-quoted-safe string. We keep values minimal and
# escape double-quotes/backslashes so the output is valid JSON without jq.
json_escape() {
  printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g' | tr '\t' ' '
}

cmd_snapshot() {
  local root; root="$(repo_root)"
  [ -n "$root" ] || { echo "FATAL: not a git repository." >&2; exit 1; }
  local sf; sf="$(state_file)"
  mkdir -p "$(dirname "$sf")"

  local branch ts commit short subject
  branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
  ts="$(now_iso)"
  commit="$(git rev-parse HEAD 2>/dev/null || echo "")"
  short="$(git rev-parse --short HEAD 2>/dev/null || echo "")"
  subject="$(git log -1 --pretty=%s 2>/dev/null || echo "")"

  local decisions remaining tried task
  decisions="$(json_escape "$(context_field Decisions)")"
  remaining="$(json_escape "$(context_field Remaining)")"
  tried="$(json_escape "$(context_field Tried)")"
  task="$(json_escape "$(context_field Task)")"

  # completed_tasks: tasks named in the block are unknown to the script; the
  # skill fills this from its todo list. We record the last commit's task label.
  local completed
  completed="$(json_escape "$task")"

  cat > "$sf" <<EOF
{
  "schema": "skillgrid/checkpoint/v1",
  "updated": "$ts",
  "branch": "$branch",
  "last_commit": "$commit",
  "last_commit_short": "$short",
  "last_commit_subject": "$(json_escape "$subject")",
  "current_task": "$task",
  "completed_tasks": "$completed",
  "decisions": "$decisions",
  "remaining": "$remaining",
  "tried": "$tried"
}
EOF
  echo "CHECKPOINT_WRITTEN $sf"
}

cmd_restore() {
  local sf; sf="$(state_file)"
  if [ ! -f "$sf" ]; then
    echo "NO_CHECKPOINT $sf"
    return 0
  fi
  # Print the state for a fresh session to consume.
  cat "$sf"
  echo
  # Plus the git truth the file may predate, so resume is grounded.
  echo "--- GIT STATE ---"
  echo "branch: $(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
  echo "HEAD:   $(git rev-parse --short HEAD 2>/dev/null || echo none) $(git log -1 --pretty=%s 2>/dev/null || true)"
  git status --short 2>/dev/null || true
}

# guard is a dispatcher: run every pre-commit guard in sequence. Each is a
# sibling in .agents/hooks/. The shims (and skillgrid-cli later) call only
# `guard` — adding a guard here is the one place to change.
cmd_guard() {
  bash "$GUARD" guard                 # cwd-drift + protected-ref/detached
  bash "$SCRIPT_DIR/precommit-zone-guard.sh"
  bash "$SCRIPT_DIR/precommit-ignore-guard.sh"
  bash "$SCRIPT_DIR/gate-lint.sh"     # advisory: warn on malformed gate blocks
}
cmd_post_check() { bash "$GUARD" post-check; }

cmd_guard_msg() {
  local msg_file="${1:-}"
  [ -n "$msg_file" ] && [ -f "$msg_file" ] || { echo "guard-msg: missing message file" >&2; exit 2; }

  # Read only the first non-comment, non-empty line (the subject).
  local subject
  subject="$(sed -n '/^[[:space:]]*#/!p' "$msg_file" | sed '/^[[:space:]]*$/d' | head -1)"

  # Conventional-commit subject regex (scope optional, ! optional).
  local re='^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([a-z0-9._-]+\))?!?: .+'
  if ! echo "$subject" | grep -Eq "$re"; then
    echo "FATAL: commit subject does not follow conventional-commits." >&2
    echo "  Got:      $subject" >&2
    echo "  Expected: type(scope)?: description  (type in build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)" >&2
    echo "RECOVERY: rewrite the subject, e.g. 'feat(auth): add session refresh'." >&2
    exit 1
  fi

  # No Co-Authored-By / AI attribution trailers anywhere in the message.
  if grep -Eiq '^(Co-Authored-By|Generated-By|AI-Model|Signed-off-by).*[Aa]uthored|[Aa]uthor.*[Aa]i\b|Co-Authored-By' "$msg_file"; then
    # Be specific: flag the exact trailer.
    if grep -Eiq '^(Co-Authored-By|Generated-By):' "$msg_file"; then
      echo "FATAL: commit message carries a Co-Authored-By/Generated-By trailer." >&2
      echo "  Skillgrid convention: no AI attribution in commits (see branch-pr)." >&2
      echo "RECOVERY: remove the trailer line and re-stage the commit message." >&2
      exit 1
    fi
  fi
}

case "$SUBCMD" in
  snapshot)    cmd_snapshot ;;
  restore)     cmd_restore ;;
  guard)       cmd_guard ;;
  guard-msg)   cmd_guard_msg "$@" ;;
  post-check)  cmd_post_check ;;
  *)
    echo "unknown subcommand: $SUBCMD (expected snapshot|restore|guard|guard-msg|post-check)" >&2
    exit 2
    ;;
esac
