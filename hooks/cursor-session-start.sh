#!/usr/bin/env bash
# cursor-session-start.sh — fail-open Mnemonic bootstrap for Cursor sessionStart.
# Stdin: Cursor sessionStart JSON. Stdout: { env?, additional_context? }.

set -u
input=$(cat || true)

SID=$(printf '%s' "$input" | node -e '
  let s=""; process.stdin.on("data",d=>s+=d); process.stdin.on("end",()=>{
    try { const j=JSON.parse(s||"{}"); process.stdout.write(String(j.session_id||j.conversation_id||"")); }
    catch { process.stdout.write(""); }
  });
' 2>/dev/null || true)

BASE="${SKILLGRID_MNEMONIC_HTTP_URL:-http://127.0.0.1:7438}"
curl -sf --max-time 1 "$BASE/health" >/dev/null 2>&1 || \
  (command -v skillgrid >/dev/null 2>&1 && \
   nohup skillgrid serve >/dev/null 2>&1 & disown 2>/dev/null) || true

# Prefer staged hooks, then repo-local session-start.js beside this script.
HOOK=""
if [ -f "$HOME/.skillgrid/hooks/session-start.js" ]; then
  HOOK="$HOME/.skillgrid/hooks/session-start.js"
elif [ -f "$(dirname "$0")/session-start.js" ]; then
  HOOK="$(dirname "$0")/session-start.js"
fi
if [ -n "$HOOK" ]; then
  node "$HOOK" >/dev/null 2>&1 || true
fi

CONTEXT='Mnemonic is active. Start with mem_session_start (title), save decisions/bugfixes via mem_save, and close with mem_session_summary + mem_session_end. Prefer mem_context / mem_search before re-deriving prior work.'

node -e '
  const sid = process.argv[1] || "";
  const ctx = process.argv[2] || "";
  const out = { additional_context: ctx };
  if (sid) out.env = { SKILLGRID_CURSOR_SESSION_ID: sid, SKILLGRID_AGENT: "cursor" };
  process.stdout.write(JSON.stringify(out) + "\n");
' "$SID" "$CONTEXT" 2>/dev/null || echo '{}'
exit 0
