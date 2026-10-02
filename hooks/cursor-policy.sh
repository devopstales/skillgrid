#!/usr/bin/env bash
# cursor-policy.sh — Cursor beforeShellExecution / beforeMCPExecution /
# beforeReadFile hook → tool-call-capture.js policy (ADR-0021).
# Fail-open: prints {"permission":"allow"} unless the policy blocks.

set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
CAPTURE=""
if [ -f "$HOME/.skillgrid/hooks/tool-call-capture.js" ]; then
  CAPTURE="$HOME/.skillgrid/hooks/tool-call-capture.js"
elif [ -f "$HERE/tool-call-capture.js" ]; then
  CAPTURE="$HERE/tool-call-capture.js"
fi

ALLOW='{"permission":"allow"}'
if [ -z "$CAPTURE" ] || ! command -v node >/dev/null 2>&1; then
  cat >/dev/null
  echo "$ALLOW"
  exit 0
fi

OUT="$(SKILLGRID_AGENT=cursor node "$CAPTURE" policy 2>/dev/null)" || OUT=""
case "$OUT" in
  *'"permission"'*) printf '%s\n' "$OUT" ;;
  *) echo "$ALLOW" ;;
esac
exit 0
