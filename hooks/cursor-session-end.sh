#!/usr/bin/env bash
# cursor-session-end.sh / stop — fail-open teardown (parity with OpenCode session.idle).

set -u
input=$(cat || true)

SID=$(printf '%s' "$input" | node -e '
  let s=""; process.stdin.on("data",d=>s+=d); process.stdin.on("end",()=>{
    try { const j=JSON.parse(s||"{}"); process.stdout.write(String(j.session_id||j.conversation_id||"")); }
    catch { process.stdout.write(""); }
  });
' 2>/dev/null || true)

DIR=$(printf '%s' "$input" | node -e '
  let s=""; process.stdin.on("data",d=>s+=d); process.stdin.on("end",()=>{
    try {
      const j=JSON.parse(s||"{}");
      const roots=Array.isArray(j.workspace_roots)?j.workspace_roots:[];
      process.stdout.write(String(j.cwd||roots[0]||""));
    } catch { process.stdout.write(""); }
  });
' 2>/dev/null || true)

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

if command -v skillgrid >/dev/null 2>&1; then
  skillgrid compact --dir "${DIR:-$PWD}" --session "$SID" >/dev/null 2>&1 || true
fi

echo '{}'
exit 0
