#!/usr/bin/env bash
# cursor-tool-capture.sh — thin Cursor postToolUse wrapper → tool-call-capture.js
# Fail-open. Always prints "{}" for Cursor.

set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
CAPTURE=""
if [ -f "$HOME/.skillgrid/hooks/tool-call-capture.js" ]; then
  CAPTURE="$HOME/.skillgrid/hooks/tool-call-capture.js"
elif [ -f "$HERE/tool-call-capture.js" ]; then
  CAPTURE="$HERE/tool-call-capture.js"
fi

if [ -z "$CAPTURE" ]; then
  echo '{}'
  exit 0
fi

export SKILLGRID_AGENT="${SKILLGRID_AGENT:-cursor}"
# Drain stdin into the shared capture script; ignore its exit / stdout.
node "$CAPTURE" >/dev/null 2>&1 || true
echo '{}'
exit 0
