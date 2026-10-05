#!/usr/bin/env bash
# opencode-session-start.sh — fail-open Mnemonic bootstrap for OpenCode session.created.
# Env: OPENCODE_SESSION_ID, OPENCODE_PROJECT_DIR. SKILLGRID_AGENT defaults to opencode
# (Kilo sets kilo). Stdout is not a session prompt.

set -u
cat >/dev/null || true

AGENT="${SKILLGRID_AGENT:-opencode}"
SID="${OPENCODE_SESSION_ID:-}"
DIR="${OPENCODE_PROJECT_DIR:-$PWD}"
BASE="${SKILLGRID_MNEMONIC_HTTP_URL:-http://127.0.0.1:7438}"

curl -sf --max-time 1 "$BASE/health" >/dev/null 2>&1 || \
  (command -v skillgrid >/dev/null 2>&1 && \
   nohup skillgrid serve >/dev/null 2>&1 & disown 2>/dev/null) || true

HERE="$(cd "$(dirname "$0")" && pwd)"
CAPTURE=""
if [ -f "$HOME/.skillgrid/hooks/tool-call-capture.js" ]; then
  CAPTURE="$HOME/.skillgrid/hooks/tool-call-capture.js"
elif [ -f "$HERE/tool-call-capture.js" ]; then
  CAPTURE="$HERE/tool-call-capture.js"
fi
if [ -n "$SID" ] && [ -n "$CAPTURE" ]; then
  SKILLGRID_AGENT="$AGENT" OPENCODE_PROJECT_DIR="$DIR" \
    node "$CAPTURE" register </dev/null >/dev/null 2>&1 || true
fi

if command -v skillgrid >/dev/null 2>&1; then
  skillgrid prime --dir "$DIR" --session "$SID" >/dev/null 2>&1 || true
fi

if [ -n "$SID" ]; then
  node -e '
    const sid = process.argv[1];
    const dir = process.argv[2] || ".";
    const base = process.argv[3];
    const agent = process.argv[4] || "opencode";
    fetch(base + "/sessions", {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ id: sid, directory: dir, agent }),
      signal: AbortSignal.timeout(2000),
    }).then(() => process.exit(0)).catch(() => process.exit(0));
  ' "$SID" "$DIR" "$BASE" "$AGENT" >/dev/null 2>&1 || true
fi
exit 0
