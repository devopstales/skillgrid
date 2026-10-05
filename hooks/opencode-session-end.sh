#!/usr/bin/env bash
# opencode-session-end.sh — OpenCode session.idle teardown (compact + usage).
# Fail-open. stop-tests and gate-stop stay on the Cursor stop hook: session.idle
# cannot block the agent, and those gates (go test ./...) exceed the hook timeout.
# The checkpoint prompt is skillgrid-checkpoint.ts, not this script.

set -u
AGENT="${SKILLGRID_AGENT:-opencode}"
DIR="${OPENCODE_PROJECT_DIR:-$PWD}"
SID="${OPENCODE_SESSION_ID:-}"
HERE="$(cd "$(dirname "$0")" && pwd)"
# One JSON payload, then the runtime closes stdin. Do not wait if it stays open.
read -t 1 -r _ || true

resolve() {
  local name="$1"
  if [ -f "$HOME/.skillgrid/hooks/$name" ]; then
    printf '%s' "$HOME/.skillgrid/hooks/$name"
  elif [ -f "$HERE/$name" ]; then
    printf '%s' "$HERE/$name"
  fi
}

if command -v skillgrid >/dev/null 2>&1; then
  skillgrid compact --dir "$DIR" --session "$SID" >/dev/null 2>&1 || true
fi

CAPTURE="$(resolve tool-call-capture.js)"
if [ -n "$SID" ] && [ -n "$CAPTURE" ]; then
  SKILLGRID_AGENT="$AGENT" OPENCODE_PROJECT_DIR="$DIR" \
    node "$CAPTURE" usage </dev/null >/dev/null 2>&1 || true
fi
exit 0
