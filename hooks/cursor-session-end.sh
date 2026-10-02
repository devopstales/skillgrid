#!/usr/bin/env bash
# cursor-session-end.sh — sessionEnd teardown + stop checkpoint follow-up.
# Fail-open always. sessionEnd never emits a follow-up; stop may.

set -u
EVENT="${1:-}"
input=$(cat || true)

if [ -z "$EVENT" ]; then
  EVENT=$(printf '%s' "$input" | node -e '
    let s=""; process.stdin.on("data",d=>s+=d); process.stdin.on("end",()=>{
      try {
        const j=JSON.parse(s||"{}");
        process.stdout.write(String(j.hook_event_name||""));
      } catch { process.stdout.write(""); }
    });
  ' 2>/dev/null || true)
fi

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

CAPTURE="$HOME/.skillgrid/hooks/tool-call-capture.js"
[ -f "$CAPTURE" ] || CAPTURE="$(dirname "$0")/tool-call-capture.js"

if [ -f "$CAPTURE" ]; then
  printf '%s' "$input" | SKILLGRID_AGENT=cursor node "$CAPTURE" usage >/dev/null 2>&1 || true
fi

if command -v skillgrid >/dev/null 2>&1; then
  skillgrid compact --dir "${DIR:-$PWD}" --session "$SID" >/dev/null 2>&1 || true
fi

case "$EVENT" in
  stop)
    if [ -f "$CAPTURE" ]; then
      OUT="$(printf '%s' "$input" | SKILLGRID_AGENT=cursor node "$CAPTURE" checkpoint 2>/dev/null)" || OUT=""
      case "$OUT" in
        *'"followup_message"'*) printf '%s\n' "$OUT" ;;
        *) echo '{}' ;;
      esac
    else
      echo '{}'
    fi
    ;;
  *)
    echo '{}'
    ;;
esac
exit 0
