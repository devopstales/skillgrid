#!/usr/bin/env bash
# checkpoint-record.sh — Tier 1 operational checkpoint (log + handoff + event).
#
# Usage:
#   checkpoint-record.sh --change <change-id> --name <label> [options]
#
# Required:
#   --change <id>     OpenSpec / Skillgrid change id
#   --name <label>    Checkpoint name (e.g. before-apply, after-loop-3)
#
# Options:
#   --trigger <id>    Trigger id (default: same as --name)
#   --phase <phase>   SDD phase (apply, verify, archive, loop, handoff, …)
#   --slice <text>    Active slice or task line (short)
#   --evidence <text> One-line evidence summary
#   --prd <path>      PRD path (auto-detected from .skillgrid/prd when omitted)
#   --context <path>  Handoff path (default: .skillgrid/tasks/context_<change>.md)
#   --dry-run         Print actions without writing
#
# Exit: 0 on success, 1 usage error, 2 not a git repo, 3 write failure

set -euo pipefail

cd "$(dirname "$0")/../.." || exit 3

CHANGE=""
NAME=""
TRIGGER=""
PHASE=""
SLICE=""
EVIDENCE=""
PRD=""
CONTEXT=""
DRY_RUN=false

usage() {
  sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --change) CHANGE="${2:-}"; shift 2 ;;
    --name) NAME="${2:-}"; shift 2 ;;
    --trigger) TRIGGER="${2:-}"; shift 2 ;;
    --phase) PHASE="${2:-}"; shift 2 ;;
    --slice) SLICE="${2:-}"; shift 2 ;;
    --evidence) EVIDENCE="${2:-}"; shift 2 ;;
    --prd) PRD="${2:-}"; shift 2 ;;
    --context) CONTEXT="${2:-}"; shift 2 ;;
    --dry-run) DRY_RUN=true; shift ;;
    -h|--help) usage ;;
    *) echo "Unknown option: $1" >&2; usage ;;
  esac
done

[[ -n "$CHANGE" && -n "$NAME" ]] || usage

if ! git rev-parse --show-toplevel >/dev/null 2>&1; then
  echo "Error: run inside a git repository." >&2
  exit 2
fi

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

TRIGGER="${TRIGGER:-$NAME}"
CONTEXT="${CONTEXT:-.skillgrid/tasks/context_${CHANGE}.md}"
TASKS_DIR="${REPO_ROOT}/.skillgrid/tasks"
CHECKPOINT_LOG="${TASKS_DIR}/checkpoints.log"
EVENTS_FILE="${TASKS_DIR}/events/${CHANGE}.jsonl"

ISO_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
BRANCH="$(git branch --show-current 2>/dev/null || echo detached-head)"
SHA="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
DIRTY="no"
if [[ -n "$(git status --porcelain 2>/dev/null)" ]]; then
  DIRTY="yes"
fi

# --- helpers (defined before use) -------------------------------------------

find_prd_for_change() {
  local change_id="$1"
  local prd_root="${REPO_ROOT}/.skillgrid/prd"
  [[ -d "$prd_root" ]] || return 1
  local file
  while IFS= read -r -d '' file; do
    if grep -qiE "(change[[:space:]_-]*id|openspec/changes/${change_id}|changes/${change_id})" "$file" 2>/dev/null; then
      to_posix_rel "$file"
      return 0
    fi
  done < <(find "$prd_root" -maxdepth 1 -name 'PRD*.md' -print0 2>/dev/null)
  return 1
}

to_posix_rel() {
  local abs="$1"
  python3 -c 'import os,sys; print(os.path.relpath(sys.argv[1], sys.argv[2]).replace(os.sep, "/"))' "$abs" "$REPO_ROOT"
}

quote_evidence_for_log() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  printf '"%s"' "$value"
}

build_log_line() {
  local ev_quoted
  ev_quoted="$(quote_evidence_for_log "$EVIDENCE")"
  local line="${ISO_TIME} name=${NAME} trigger=${TRIGGER} branch=${BRANCH} sha=${SHA} dirty=${DIRTY} change=${CHANGE} context=${CONTEXT}"
  [[ -n "$PRD" ]] && line="${line} prd=${PRD}"
  [[ -n "$PHASE" ]] && line="${line} phase=${PHASE}"
  if [[ -n "$SLICE" ]]; then
    local slice_quoted
    slice_quoted="$(quote_evidence_for_log "$SLICE")"
    line="${line} slice=${slice_quoted}"
  fi
  line="${line} evidence=${ev_quoted}"
  printf '%s\n' "$line"
}

update_handoff_section() {
  local handoff_abs="${REPO_ROOT}/${CONTEXT#./}"
  [[ -f "$handoff_abs" ]] || {
    echo "WARN: handoff file not found (skipped): ${CONTEXT}" >&2
    return 0
  }

  local section
  section="$(cat <<EOF
## Last checkpoint

- \`${NAME}\` — \`${SHA}\` @ ${ISO_TIME} (trigger: \`${TRIGGER}\`)
- Log: \`.skillgrid/tasks/checkpoints.log\`
EOF
)"

  if ! command -v python3 >/dev/null 2>&1; then
    echo "" >> "$handoff_abs"
    printf '%s\n' "$section" >> "$handoff_abs"
    return 0
  fi

  python3 - "$handoff_abs" "$section" <<'PY'
import re
import sys

path, section = sys.argv[1], sys.argv[2].strip() + "\n"
text = open(path, encoding="utf-8").read()
pattern = re.compile(r"## Last checkpoint\n.*?(?=\n## |\Z)", re.DOTALL)
if pattern.search(text):
    text = pattern.sub(section + "\n", text, count=1)
else:
    text = text.rstrip() + "\n\n" + section + "\n"
open(path, "w", encoding="utf-8").write(text)
PY
}

append_jsonl_event() {
  mkdir -p "${TASKS_DIR}/events"
  PHASE="${PHASE:-checkpoint}" \
  CHANGE="$CHANGE" \
  ISO_TIME="$ISO_TIME" \
  TRIGGER="$TRIGGER" \
  NAME="$NAME" \
  SHA="$SHA" \
  BRANCH="$BRANCH" \
  DIRTY="$DIRTY" \
  CONTEXT="$CONTEXT" \
  PRD="$PRD" \
  EVIDENCE="$EVIDENCE" \
  python3 - "$EVENTS_FILE" <<'PY'
import json
import os
import sys

path = sys.argv[1]
entry = {
    "time": os.environ["ISO_TIME"],
    "changeId": os.environ["CHANGE"],
    "node": "checkpoint",
    "phase": os.environ.get("PHASE") or "checkpoint",
    "status": "completed",
    "checkpoint": os.environ["NAME"],
    "trigger": os.environ["TRIGGER"],
    "sha": os.environ["SHA"],
    "branch": os.environ["BRANCH"],
    "dirty": os.environ["DIRTY"],
    "artifacts": [".skillgrid/tasks/checkpoints.log", os.environ["CONTEXT"]],
}
prd = os.environ.get("PRD", "").strip()
if prd:
    entry["prd"] = prd
evidence = os.environ.get("EVIDENCE", "").strip()
entry["summary"] = (
    f"checkpoint {entry['trigger']}: {evidence[:200]}"
    if evidence
    else f"checkpoint {entry['trigger']}: {entry['checkpoint']}"
)
with open(path, "a", encoding="utf-8") as handle:
    handle.write(json.dumps(entry, ensure_ascii=False) + "\n")
PY
}

# --- resolve fields ---------------------------------------------------------

if [[ -z "$PRD" ]]; then
  PRD="$(find_prd_for_change "$CHANGE" || true)"
fi

if [[ -z "$EVIDENCE" ]]; then
  EVIDENCE="checkpoint ${TRIGGER} on ${BRANCH}@${SHA}"
fi

# --- main -------------------------------------------------------------------

LOG_LINE="$(build_log_line)"

if "$DRY_RUN"; then
  echo "DRY RUN — would append to ${CHECKPOINT_LOG}:"
  echo "$LOG_LINE"
  echo "DRY RUN — would update ${CONTEXT}"
  echo "DRY RUN — would append to ${EVENTS_FILE}"
  exit 0
fi

mkdir -p "$TASKS_DIR"
echo "$LOG_LINE" >> "$CHECKPOINT_LOG"
update_handoff_section
append_jsonl_event

# Per-change phase state for sdd-gate.sh (verify before review, review before archive)
STATE_DIR="${REPO_ROOT}/.skillgrid/state/${CHANGE}"
case "${TRIGGER}" in
  verify-pass)
    mkdir -p "$STATE_DIR"
    printf 'passed\n' > "${STATE_DIR}/verification_status"
    ;;
  review-pass|review-approved)
    mkdir -p "$STATE_DIR"
    printf 'approved\n' > "${STATE_DIR}/review_status"
    ;;
esac

echo "Checkpoint recorded: ${NAME} (${SHA}) → ${CHECKPOINT_LOG}"
