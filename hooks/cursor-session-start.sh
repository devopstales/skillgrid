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

DIR=$(printf '%s' "$input" | node -e '
  let s=""; process.stdin.on("data",d=>s+=d); process.stdin.on("end",()=>{
    try {
      const j=JSON.parse(s||"{}");
      const roots=Array.isArray(j.workspace_roots)?j.workspace_roots:[];
      process.stdout.write(String(j.cwd||roots[0]||""));
    } catch { process.stdout.write(""); }
  });
' 2>/dev/null || true)

PRIME=""
if command -v skillgrid >/dev/null 2>&1; then
  PRIME=$(skillgrid prime --dir "${DIR:-.}" --session "$SID" 2>/dev/null || true)
fi
if [ -z "$PRIME" ]; then
  PRIME='Mnemonic is active. Start with mem_session_start (title), save decisions/bugfixes via mem_save, and close with mem_session_summary + mem_session_end. Prefer mem_context / mem_search before re-deriving prior work. Call code_explore before rg.'
fi

node -e '
  const sid = process.argv[1] || "";
  const ctx = process.argv[2] || "";
  const out = { additional_context: ctx };
  const env = {};
  if (sid) {
    env.SKILLGRID_CURSOR_SESSION_ID = sid;
    env.SKILLGRID_AGENT = "cursor";
  }
  const m = ctx.match(/^project: (.+)$/m);
  if (m) env.MNEMONIC_PROJECT = m[1].trim();
  if (Object.keys(env).length) out.env = env;
  process.stdout.write(JSON.stringify(out) + "\n");
' "$SID" "$PRIME" 2>/dev/null || echo '{}'
exit 0
