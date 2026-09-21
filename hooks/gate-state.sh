#!/usr/bin/env bash
# gate-state.sh — the gate state engine (bash port of unlazy's lib/gates.mjs).
#
# Parses a change's acceptance.feature `#### Gates` blocks and reports each
# G<n>'s state. skillgrid's format has NO checkbox (unlazy used `- [ ]`); a
# gate is met only when its CHECK command freshly exits 0 AND its output
# matches EXPECT — see test-driven-verification/SKILL.md ("the declared oracle,
# run fresh, is the oracle"). So the two modes:
#
#   --status    NON-executing. Report the ledger: which gates are runnable
#               (reported as `unmet (not run)`), which are manual, which
#               ABANDON, and whether any happy-path requirement is missing a
#               gate. Never runs a CHECK, writes nothing. Exit 0 always.
#   --reverify  Executing. Run every runnable CHECK, compare output to EXPECT,
#               report met / unmet / abandoned / manual / missing. This is the
#               fresh-evidence path the Stop hook uses.
#
# Scope: pass --spec <acceptance.feature>, or auto-discover. Discovery order:
#   1. an acceptance.feature path named in .skillgrid/sdd/checkpoint.json
#   2. an acceptance.feature touched in the current diff (git diff HEAD)
#   3. the most-recently-modified acceptance.feature under <specs_root>
# specs_root defaults to .skillgrid/specs (overridable via config bdd.specs_dir).
#
# Output (stdout), one line per gate / requirement, plus a summary:
#   G1=met   G2=unmet   G3=abandoned   G4=manual   REQ=<req>=missing
#   SUMMARY met=.. unmet=.. abandoned=.. manual=.. missing=..
# Exit code is 0 in both modes; gate-stop.sh interprets the states.
set -uo pipefail

MODE="status"
SPEC=""
SPECS_ROOT=""
TIMEOUT="${SKILLGRID_GATE_TIMEOUT:-120}"
ROOT=""

while [ $# -gt 0 ]; do
  case "$1" in
    --status)     MODE="status" ;;
    --reverify)   MODE="reverify" ;;
    --spec)       SPEC="${2:-}"; shift ;;
    --specs-root) SPECS_ROOT="${2:-}"; shift ;;
    --root)       ROOT="${2:-}"; shift ;;
    -h|--help)
      echo "usage: gate-state.sh [--status|--reverify] [--spec FILE] [--specs-root DIR] [--root DIR]"
      exit 0 ;;
    *) echo "gate-state: unknown option $1" >&2; exit 2 ;;
  esac
  shift
done

[ -n "$ROOT" ] || ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

# ---- config: bdd.specs_dir -------------------------------------------------
if [ -z "$SPECS_ROOT" ]; then
  cfg="$ROOT/.skillgrid/config.yaml"
  if [ -f "$cfg" ]; then
    v="$(awk '$0 ~ /^[[:space:]]*specs_dir:/ { sub(/^[^:]*:[[:space:]]*/,""); sub(/[[:space:]]*#.*$/,""); sub(/["'\''"]+/g,""); print; exit }' "$cfg" 2>/dev/null || true)"
    [ -n "$v" ] && SPECS_ROOT="$v"
  fi
  [ -n "$SPECS_ROOT" ] || SPECS_ROOT=".skillgrid/specs"
fi

# ---- spec discovery --------------------------------------------------------
if [ -z "$SPEC" ]; then
  ck="$ROOT/.skillgrid/sdd/checkpoint.json"
  if [ -f "$ck" ]; then
    SPEC="$(grep -oE '\.skillgrid/specs/[^"[:space:]]*/acceptance\.feature' "$ck" | head -1 || true)"
  fi
  if [ -z "$SPEC" ] || [ ! -f "$ROOT/$SPEC" ]; then
    SPEC="$(git -C "$ROOT" diff --name-only HEAD -- 2>/dev/null | grep -E '/acceptance\.feature$' | head -1 || true)"
  fi
  if [ -z "$SPEC" ] || [ ! -f "$ROOT/$SPEC" ]; then
    newest="$(find "$ROOT/$SPECS_ROOT" -name acceptance.feature -type f 2>/dev/null | xargs -I{} ls -1t {} 2>/dev/null | head -1 || true)"
    [ -n "$newest" ] && SPEC="${newest#"$ROOT"/}"
  fi
fi

if [ -z "$SPEC" ] || [ ! -f "$ROOT/$SPEC" ]; then
  echo "NO_SPEC"
  echo "SUMMARY met=0 unmet=0 abandoned=0 manual=0 missing=0"
  exit 0
fi
SPEC_ABS="$ROOT/$SPEC"

# One awk pass: emit TAB-joined records. Gate line -> "G\t<gid>\t<kind>\t<check>\t<expect>".
# A requirement that declared a happy-path scenario but carries no gate -> "M\t<req>".
# kind is RUN (has CHECK), MAN (no CHECK, not abandoned), ABANDON.
RECORDS="$(awk '
  function trim(s){ sub(/^[[:space:]]+/,"",s); sub(/[[:space:]]+$/,"",s); return s }
  function flush(  line){
    if (gid=="") return
    line = (abandoned?"ABANDON":(check!=""?"RUN":"MAN"))
    printf "G\t%s\t%s\t%s\t%s\n", gid, line, check, expect
    gid=""; check=""; expect=""; abandoned=0
  }
  /^[[:space:]]*```(gherkin)?[[:space:]]*$/ { inFence=!inFence; flush(); next }
  inFence { next }
  /^[[:space:]]*#/ { next }
  /^###[[:space:]]+Requirement:/ {
    flush()
    req=$0; sub(/^###[[:space:]]+Requirement:[[:space:]]*/,"",req)
    happy=0; ngates=0
    next
  }
  /^####[[:space:]]+Scenario:/ { if ($0 ~ /happy/) happy=1; next }
  /^G[0-9]+[A-Za-z]*:/ {
    flush()
    match($0, /^G[0-9]+[A-Za-z]*/)
    gid=substr($0, RSTART, RLENGTH)
    ngates++
    if ($0 ~ /^G[0-9]+[A-Za-z]*:[[:space:]]*ABANDON([[:space:]]|$)/) abandoned=1
    next
  }
  /^[[:space:]]*CHECK:/  { check=trim($0);  sub(/^[[:space:]]*CHECK:[[:space:]]*/,"",check) }
  /^[[:space:]]*EXPECT:/ { expect=trim($0); sub(/^[[:space:]]*EXPECT:[[:space:]]*/,"",expect) }
  END {
    flush()
    # emit missing-flag for the last requirement (and any prior) — but we only
    # know ngates at req end; track per-req and emit here for the last one only.
  }
' "$SPEC_ABS")"

# The END block above can only flag the final requirement. Re-emit a clean
# missing list with a second, simple pass keyed on requirement blocks.
MISSING="$(awk '
  function trim(s){ sub(/^[[:space:]]+/,"",s); sub(/[[:space:]]+$/,"",s); return s }
  /^###[[:space:]]+Requirement:/ { if (req!="") emit(); req=$0; sub(/^###[[:space:]]+Requirement:[[:space:]]*/,"",req); happy=0; ngates=0; next }
  /^####[[:space:]]+Scenario:/ { if ($0 ~ /happy/) happy=1; next }
  /^[[:space:]]*```(gherkin)?[[:space:]]*$/ { inFence=!inFence; next }
  inFence { next }
  /^G[0-9]+[A-Za-z]*:/ { ngates++; next }
  function emit(){ if (req!="" && happy==1 && ngates==0) print req }
  END { emit() }
' "$SPEC_ABS")"

# ---- CHECK execution -------------------------------------------------------
# Run a CHECK command, bounded by a timer when one is available (GNU/BSD
# coreutils). macOS ships neither `timeout` nor `gtimeout` by default, so we
# fall back to an unbounded `bash -c` — acceptable for a gate whose own
# command should be quick. Output (stdout+stderr) on stdout, exit code
# propagated.
run_check() {
  local cmd="$1"
  if command -v timeout >/dev/null 2>&1; then
    timeout "$TIMEOUT" bash -c "$cmd"
  elif command -v gtimeout >/dev/null 2>&1; then
    gtimeout "$TIMEOUT" bash -c "$cmd"
  else
    bash -c "$cmd"
  fi
}

# ---- EXPECT matching -------------------------------------------------------
# Default: literal substring. A value wrapped in /.../ is a POSIX ERE; a
# trailing 'i' flag is honored best-effort via grep -i.
expect_matches() {
  local output="$1" expect="$2"
  case "$expect" in
    /*/*)
      local body="${expect#/}"; body="${body%%/*}"; flags="${expect##*/}"
      local re="$body"; local use_i=0
      case "${flags#/}" in *i*) use_i=1 ;; esac
      if [ "$use_i" -eq 1 ]; then printf '%s' "$output" | grep -Ei -q -- "$re" 2>/dev/null
      else printf '%s' "$output" | grep -Eq -- "$re" 2>/dev/null; fi ;;
    *) printf '%s' "$output" | grep -Fq -- "$expect" 2>/dev/null ;;
  esac
}

# ---- classify --------------------------------------------------------------
met=0; unmet=0; abandoned=0; manual=0; missing=0
OUT=""

append() { OUT+="$1"$'\n'; }

# gate records
while IFS=$'\t' read -r tag gid kind check expect; do
  [ "$tag" = "G" ] || continue
  case "$kind" in
    ABANDON)
      STATE_ABANDON=1; abandoned=$((abandoned+1)); append "$gid=abandoned" ;;
    MAN)
      manual=$((manual+1)); append "$gid=manual" ;;
    RUN)
      if [ "$MODE" = "reverify" ] && [ -n "$check" ]; then
        if output="$(run_check "$check" 2>&1)"; then
          if [ -z "$expect" ] || expect_matches "$output" "$expect"; then
            met=$((met+1)); append "$gid=met"
          else
            unmet=$((unmet+1)); append "$gid=unmet"
          fi
        else
          unmet=$((unmet+1)); append "$gid=unmet"
        fi
      else
        unmet=$((unmet+1)); append "$gid=unmet (not run; --reverify to execute)"
      fi ;;
  esac
done <<< "$RECORDS"

# missing gates
while IFS= read -r req; do
  [ -n "$req" ] || continue
  missing=$((missing+1)); append "REQ=$req=missing"
done <<< "$MISSING"

printf '%s' "$OUT"
echo "SUMMARY met=$met unmet=$unmet abandoned=$abandoned manual=$manual missing=$missing"
exit 0
