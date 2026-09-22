#!/usr/bin/env bash
# checkpoint-state.sh — skillgrid guard-hooks entrypoint.
#
# The durable record lives in the commit (its [skillgrid-context] body block).
# Resume reads the session event stream, not a checkpoint file.
#
# CLI-ready: the repo's .git/hooks shims call this script. skillgrid-cli later
# rebinds the shims to its binary; the subcommand contract stays stable.
#
# Subcommands:
#   guard        pre-commit guards (delegates to precommit-guard.sh)
#   guard-msg    validate a commit message file (conventional + no Co-Authored-By)
#   post-check   post-commit deletion check (delegates to precommit-guard.sh)
#
# (The snapshot/restore subcommands were removed in the session-events-layer
# consolidation: the durable record is the commit's [skillgrid-context] block
# and resume reads the session event stream. Both now report unknown
# subcommand.)
#
# Env: none.
set -euo pipefail

SUBCMD="${1:-}"
shift || true

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GUARD="$SCRIPT_DIR/precommit-guard.sh"

# guard is a dispatcher: run every pre-commit guard in sequence. Each is a
# sibling in hooks/. The shims (and skillgrid-cli later) call only
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
  guard)       cmd_guard ;;
  guard-msg)   cmd_guard_msg "$@" ;;
  post-check)  cmd_post_check ;;
  *)
    echo "unknown subcommand: $SUBCMD (expected guard|guard-msg|post-check)" >&2
    exit 2
    ;;
esac
