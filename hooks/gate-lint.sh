#!/usr/bin/env bash
# gate-lint.sh — advisory static linter for acceptance.feature gate blocks.
#
# Mirrors the useful subset of unlazy's gate-lint.mjs: flags structural problems
# an agent can fix in the spec before committing, WITHOUT executing anything.
# This is a linter, not a gate: it emits warnings and exits 0. It rides the
# pre-commit dispatcher (checkpoint-state.sh guard) so a malformed gate is
# surfaced at commit time, not at Stop time.
#
# Flags (per gate / requirement):
#   - a runnable gate missing CHECK or EXPECT (one present, one absent)
#   - a CHECK present but blank, or an EXPECT present but blank
#   - an EXPECT that looks like a tautology (contains only a digit/word that
#     also appears verbatim in the CHECK echo — heuristic, low-confidence)
#   - an ABANDON with a blank reason
#   - a requirement that declares a happy-path scenario but carries no gate
#     (a traceability gap — the same check gate-state reports as `missing`)
#   - a gate id that does not match G<n> (or G<n><letter>)
#
# Exit code: always 0 (warnings only). No spec / no gates -> silent.
set -uo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0

# Target: an explicit --spec, else any acceptance.feature in the staged diff,
# else the most-recently-modified one under the specs root.
SPEC=""
while [ $# -gt 0 ]; do case "$1" in --spec) SPEC="${2:-}"; shift;; esac; shift; done
SPECS_ROOT=".skillgrid/specs"
cfg="$ROOT/.skillgrid/config.yaml"
if [ -f "$cfg" ]; then
  v="$(awk '$0 ~ /^[[:space:]]*specs_dir:/ { sub(/^[^:]*:[[:space:]]*/,""); sub(/[[:space:]]*#.*$/,""); sub(/["'\''"]+/g,""); print; exit }' "$cfg" 2>/dev/null || true)"
  [ -n "$v" ] && SPECS_ROOT="$v"
fi
if [ -z "$SPEC" ]; then
  SPEC="$(git -C "$ROOT" diff --cached --name-only 2>/dev/null | grep -E '/acceptance\.feature$' | head -1 || true)"
fi
if [ -z "$SPEC" ] || [ ! -f "$ROOT/$SPEC" ]; then
  SPEC="$(git -C "$ROOT" diff --name-only HEAD 2>/dev/null | grep -E '/acceptance\.feature$' | head -1 || true)"
fi
if [ -z "$SPEC" ] || [ ! -f "$ROOT/$SPEC" ]; then
  newest="$(find "$ROOT/$SPECS_ROOT" -name acceptance.feature -type f 2>/dev/null | xargs -I{} ls -1t {} 2>/dev/null | head -1 || true)"
  [ -n "$newest" ] && SPEC="${newest#"$ROOT"/}"
fi
[ -n "$SPEC" ] && [ -f "$ROOT/$SPEC" ] || exit 0
SPEC_ABS="$ROOT/$SPEC"

# Single awk pass: parse gates (id/kind/check/expect/abandon-reason) and
# per-requirement happy+gatecount, emitting "WARN<TAB>message" lines.
awk '
  function trim(s){ sub(/^[[:space:]]+/,"",s); sub(/[[:space:]]+$/,"",s); return s }
  function flushgate(  msg){
    if (gid=="") return
    if (gid !~ /^G[0-9]+[A-Za-z]*$/) msg("gate id must match G<n> (line: " gid ")")
    if (abandoned) {
      # reason is the text after "ABANDON" on the G<n>: line
      if (reason ~ /^[[:space:]]*$/) msg("ABANDON " gid " needs a non-blank reason")
    } else if (hasCheck != hasExpect) {
      msg("gate " gid " is runnable but missing " (hasCheck?"EXPECT":"CHECK"))
    } else if (hasCheck && (check ~ /^[[:space:]]*$/ || expect ~ /^[[:space:]]*$/)) {
      msg("gate " gid " has a blank CHECK or EXPECT")
    }
    if (hasExpect && expect ~ /^[[:space:]]*[0-9]+[[:space:]]*$/ && check ~ expect) {
      msg("gate " gid " EXPECT is a bare number that also appears in CHECK (tautology?)")
    }
    gid=""; check=""; expect=""; abandoned=0; reason=""; hasCheck=0; hasExpect=0
  }
  function flushreq(){
    if (req!="" && happy==1 && ngates==0) msg("requirement \"" req "\" declares a happy-path scenario but has no G<n> gate (traceability gap)")
  }
  function msg(m){ printf "WARN\t%s\n", m }
  /^###[[:space:]]+Requirement:/ { flushgate(); flushreq(); req=$0; sub(/^###[[:space:]]+Requirement:[[:space:]]*/,"",req); happy=0; ngates=0; next }
  /^####[[:space:]]+Scenario:/ { if ($0 ~ /happy/) happy=1; next }
  /^[[:space:]]*```(gherkin)?[[:space:]]*$/ { inFence=!inFence; next }
  inFence { next }
  /^[[:space:]]*#/ { next }
  /^G[0-9]+[A-Za-z]*:/ {
    flushgate()
    match($0, /^G[0-9]+[A-Za-z]*/); gid=substr($0, RSTART, RLENGTH)
    ngates++
    rest=$0; sub(/^G[0-9]+[A-Za-z]*:[[:space:]]*/,"",rest)
    if (rest ~ /^ABANDON([[:space:]]|$)/) { abandoned=1; reason=rest; sub(/^ABANDON[[:space:]]*/,"",reason) }
    next
  }
  /^[[:space:]]*CHECK:/  { check=trim($0); sub(/^[[:space:]]*CHECK:[[:space:]]*/,"",check); hasCheck=1 }
  /^[[:space:]]*EXPECT:/ { expect=trim($0); sub(/^[[:space:]]*EXPECT:[[:space:]]*/,"",expect); hasExpect=1 }
  END { flushgate(); flushreq() }
' "$SPEC_ABS" | sed 's/^WARN\t//' | sed "s|^|$SPEC: |"
exit 0
