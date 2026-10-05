#!/usr/bin/env bash
# opencode-policy.sh — tool.before.* → tool-call-capture.js policy (ADR-0021).
# Exit 2 blocks the tool. A missing script, missing node, or any other error allows it.

set -u
AGENT="${SKILLGRID_AGENT:-opencode}"
HERE="$(cd "$(dirname "$0")" && pwd)"
CAPTURE=""
if [ -f "$HOME/.skillgrid/hooks/tool-call-capture.js" ]; then
  CAPTURE="$HOME/.skillgrid/hooks/tool-call-capture.js"
elif [ -f "$HERE/tool-call-capture.js" ]; then
  CAPTURE="$HERE/tool-call-capture.js"
fi
if [ -z "$CAPTURE" ] || ! command -v node >/dev/null 2>&1; then
  cat >/dev/null || true
  exit 0
fi

SKILLGRID_AGENT="$AGENT" node "$CAPTURE" policy
status=$?
if [ "$status" -eq 2 ]; then
  exit 2
fi
exit 0
