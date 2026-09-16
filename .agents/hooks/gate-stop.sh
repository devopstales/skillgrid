#!/usr/bin/env bash
# gate-stop.sh — the gate-level Stop agent hook (port of unlazy's stop-hook.mjs).
#
# The suite-level stop-tests.sh only proves the whole test suite is green.
# This proves the stronger thing: every G<n> in the change's acceptance.feature
# is met, abandoned, or manual. "A rule asks; a hook guarantees" — the
# test-driven-verification completion rule ("no done-claim while any happy-path
# gate is unmet or abandoned") becomes a guarantee here: the agent cannot end
# its turn while a non-abandoned gate is unmet or a happy-path requirement is
# missing its gate.
#
# Decision (on the Stop event), after a fresh --reverify of the change's spec:
#   - any gate unmet OR any missing happy-path gate  -> BLOCK (exit 2), list them
#   - only abandoned gates remain                     -> ALLOW with HANDOFF note
#   - all met / manual / abandoned                    -> ALLOW
#   - no spec discoverable                            -> ALLOW (degrade, like stop-tests)
#
# Loop guard (unlazy's MAX_BLOCKS): if the agent is stopped repeatedly with the
# SAME resolved gate state and makes no progress, release after MAX_BLOCKS to
# avoid trapping a session on a genuinely-impossible gate. The guard keys on a
# hash of the resolved state (met/unmet/abandoned/missing), NOT raw spec bytes —
# a comment edit must not re-arm it. State lives in .skillgrid/sdd/gate-stop-state.json.
#
# This is an AGENT harness hook (Stop event), not a git hook. The harness feeds
# session JSON on stdin; we read session_id for the guard key. Install with
# install-hooks.sh --with-stop. CLI-ready: skillgrid-cli rebinds it later.
set -uo pipefail

MAX_BLOCKS="${SKILLGRID_GATE_MAX_BLOCKS:-6}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATE_STATE="$SCRIPT_DIR/gate-state.sh"

# Drain stdin (harness JSON) but capture session_id if present.
SESSION_ID=""
if [ -t 0 ]; then :; else
  INPUT="$(cat 2>/dev/null || true)"
  SESSION_ID="$(printf '%s' "$INPUT" | grep -oE '"session_id"[[:space:]]*:[[:space:]]*"[^"]+"' | head -1 | sed -E 's/.*"session_id"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/' || true)"
fi
[ -n "$SESSION_ID" ] || SESSION_ID="anonymous"

ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
STATE_FILE="$ROOT/.skillgrid/sdd/gate-stop-state.json"

# 24-char sha256 without a hard dependency (shasum on macOS, sha256sum elsewhere).
# Named hash24: bash's built-in `sha256` command would otherwise shadow a
# function of that name and receive the piped stdin instead of an argument.
hash24() {
  if command -v shasum >/dev/null 2>&1; then printf '%s' "$1" | shasum -a 256 | cut -c1-24
  else printf '%s' "$1" | sha256sum | cut -c1-24; fi
}

allow() { [ -n "${1:-}" ] && echo "$1" >&2; exit 0; }
block() { echo "${1:-}" >&2; exit 2; }

# ---- fresh evidence --------------------------------------------------------
REPORT="$(bash "$GATE_STATE" --reverify --root "$ROOT" 2>/dev/null || true)"
SUMMARY_LINE="$(printf '%s\n' "$REPORT" | grep -E '^SUMMARY ' | tail -1 || true)"

if [ -z "$SUMMARY_LINE" ] || printf '%s' "$REPORT" | grep -q '^NO_SPEC'; then
  allow ""   # no spec to verify against — degrade, don't trap
fi

met=0; unmet=0; abandoned=0; manual=0; missing=0
eval "$(printf '%s' "$SUMMARY_LINE" | sed -E 's/^SUMMARY //; s/ / /g')"

# Outstanding = unmet gates + missing happy-path requirements. Parse the
# per-gate lines, not the summary, so the block message names them.
unmet_list="$(printf '%s\n' "$REPORT" | grep -E '^[^=]+=unmet' | sed -E 's/=unmet.*//' || true)"
missing_list="$(printf '%s\n' "$REPORT" | grep -E '^REQ=.*=missing' | sed -E 's/^REQ=//; s/=missing//' || true)"

# ---- clean state: reset the guard, allow (with handoff if abandoned) -------
if [ "$unmet" -eq 0 ] && [ "$missing" -eq 0 ]; then
  # clear this session's guard entry
  if [ -f "$STATE_FILE" ]; then
    rm -f "$STATE_FILE" 2>/dev/null || true
  fi
  if [ "$abandoned" -gt 0 ]; then
    allow "gate-stop: HANDOFF REQUIRED — $abandoned abandoned gate(s). Surface them in the completion report (never a pass)."
  fi
  allow ""
fi

outstanding="$(printf '%s\n%s' "$unmet_list" "$missing_list" | sed '/^$/d')"
out_count=$((unmet + missing))
if [ -z "$outstanding" ]; then out_count=0; fi

# ---- loop guard: key on resolved state, not raw bytes ----------------------
# The guard accumulates per (session, repo) while the resolved gate state is
# UNCHANGED, and resets to 1 the moment the state changes (i.e. the agent made
# progress — a gate flipped to met, a new gate appeared, etc.). We therefore
# store BOTH the session-scoped block count and the progress hash it was
# measured against.
session_key="$(hash24 "$SESSION_ID$ROOT")"
progress_key="$(hash24 "$(printf '%s' "$REPORT" | grep -E '^(G[0-9]+[A-Za-z]*=|REQ=)' | sort)")"

blocks=0
prev_progress=""
if [ -f "$STATE_FILE" ]; then
  # pull the stored entry for this session_key
  entry="$(sed -nE 's/.*"'"$session_key"'"[[:space:]]*:[[:space:]]*\{[^}]*"hash"[[:space:]]*:[[:space:]]*"([a-f0-9]{24})"[^}]*"blocks"[[:space:]]*:[[:space:]]*([0-9]+).*/\1 \2/p' "$STATE_FILE" | head -1 || true)"
  if [ -n "$entry" ]; then
    prev_progress="${entry%% *}"
    blocks="${entry##* }"
  fi
fi
# reset if the resolved state changed (progress), else accumulate
if [ -n "$prev_progress" ] && [ "$prev_progress" != "$progress_key" ]; then blocks=0; fi
[ -n "$blocks" ] || blocks=0
blocks=$((blocks + 1))

mkdir -p "$(dirname "$STATE_FILE")"
printf '{\n  "schema": "skillgrid/gate-stop/v1",\n  "sessions": {\n    "%s": { "hash": "%s", "blocks": %d, "updated": "%s" }\n  }\n}\n' \
  "$session_key" "$progress_key" "$blocks" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$STATE_FILE"

list="$outstanding"
nlines="$(printf '%s\n' "$list" | grep -c . || true)"
if [ "$nlines" -gt 5 ]; then list="$(printf '%s\n' "$list" | head -5) (+$((nlines-5)) more)"; fi

if [ "$blocks" -gt "$MAX_BLOCKS" ]; then
  allow "gate-stop: releasing after $MAX_BLOCKS blocks without gate progress; $out_count item(s) remain ($list). Use ABANDON <id> <reason> only when a gate is genuinely impossible — it surfaces as a handoff, never a pass."
fi

block "gate-stop: $out_count gate/requirement item(s) need work before you may stop: $list. Run 'bash .agents/hooks/gate-state.sh --reverify' to see fresh evidence. ABANDON <id> <reason> is terminal and non-successful — surface it as a handoff, never a pass."
