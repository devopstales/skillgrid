#!/usr/bin/env bash
# opencode-tool-capture.sh — tool.after.* → tool-call-capture.js.
# Fail-open. Skips when OPENCODE_SESSION_ID is empty.

set -u
AGENT="${SKILLGRID_AGENT:-opencode}"
SID="${OPENCODE_SESSION_ID:-}"
HERE="$(cd "$(dirname "$0")" && pwd)"
if [ -z "$SID" ]; then
  cat >/dev/null || true
  exit 0
fi

CAPTURE=""
if [ -f "$HOME/.skillgrid/hooks/tool-call-capture.js" ]; then
  CAPTURE="$HOME/.skillgrid/hooks/tool-call-capture.js"
elif [ -f "$HERE/tool-call-capture.js" ]; then
  CAPTURE="$HERE/tool-call-capture.js"
fi
if [ -z "$CAPTURE" ]; then
  cat >/dev/null || true
  exit 0
fi

SKILLGRID_AGENT="$AGENT" node "$CAPTURE" >/dev/null 2>&1 || true
exit 0
