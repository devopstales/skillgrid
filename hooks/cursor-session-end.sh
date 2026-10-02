#!/usr/bin/env bash
# cursor-session-end.sh / stop — fail-open teardown (parity with OpenCode session.idle).

set -u
cat >/dev/null || true

run_hook() {
  local name="$1"
  if [ -f "$HOME/.skillgrid/hooks/$name" ]; then
    node "$HOME/.skillgrid/hooks/$name" >/dev/null 2>&1 || true
  elif [ -f "$(dirname "$0")/$name" ]; then
    node "$(dirname "$0")/$name" >/dev/null 2>&1 || true
  fi
}

run_hook stop-tests.js
run_hook gate-stop.js

echo '{}'
exit 0
