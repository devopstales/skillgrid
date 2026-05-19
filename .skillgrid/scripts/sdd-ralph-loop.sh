#!/usr/bin/env bash
# sdd-ralph-loop.sh — AFK driver for the SDD Ralph loop (/sdd-loop one iteration per call).
#
# Pattern: https://github.com/snarktank/ralph — fresh agent per iteration; memory via
# tasks.md, git, and .skillgrid/tasks/research/<change>/ralph-loop-state.md
#
# Usage:
#   .skillgrid/scripts/sdd-ralph-loop.sh <change-name> [max-iterations]
#
# Environment:
#   SDD_RALPH_AGENT   Agent CLI: claude | opencode | cursor (default: claude)
#   SDD_RALPH_DRY_RUN If set, print iterations without invoking agent
#
# Exit: 0 when <promise>COMPLETE</promise> seen or all tasks done; 1 on error

set -euo pipefail

CHANGE="${1:-}"
MAX="${2:-10}"
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$REPO_ROOT"

if [[ -z "$CHANGE" ]]; then
  echo "Usage: $0 <change-name> [max-iterations]" >&2
  exit 1
fi

TASKS="openspec/changes/${CHANGE}/tasks.md"
if [[ ! -f "$TASKS" ]]; then
  echo "ERROR: missing ${TASKS}" >&2
  exit 1
fi

AGENT="${SDD_RALPH_AGENT:-claude}"
PROGRESS_DIR=".skillgrid/tasks/research/${CHANGE}"
mkdir -p "$PROGRESS_DIR"
STATE="${PROGRESS_DIR}/ralph-loop-state.md"
PROGRESS="${PROGRESS_DIR}/progress.txt"
touch "$PROGRESS" "$STATE"

read -r -d '' LOOP_PROMPT <<EOF || true
You are running ONE iteration of the SDD Ralph loop for change "${CHANGE}".

Read and follow the workflow at .agents/workflows/sdd-loop.md (or /sdd-loop command) exactly.

Rules:
- Do NOT implement code yourself — delegate exactly ONE AFK task to /sdd-apply with explicit single-task scope.
- Do NOT start another iteration in this session — stop after reflect.
- If all AFK tasks are complete, output <promise>COMPLETE</promise> on its own line.

Context files: @${TASKS} @${STATE} @${PROGRESS}
EOF

run_agent() {
  local prompt="$1"
  case "$AGENT" in
    claude)
      if command -v claude &>/dev/null; then
        claude --permission-mode acceptEdits -p "$prompt"
      else
        echo "ERROR: claude CLI not found (install or set SDD_RALPH_AGENT)" >&2
        return 1
      fi
      ;;
    opencode)
      if command -v opencode &>/dev/null; then
        opencode run "$prompt"
      else
        echo "ERROR: opencode CLI not found" >&2
        return 1
      fi
      ;;
    cursor)
      if command -v cursor &>/dev/null; then
        cursor agent -p "$prompt"
      else
        echo "ERROR: cursor CLI not found" >&2
        return 1
      fi
      ;;
    *)
      echo "ERROR: unknown SDD_RALPH_AGENT=${AGENT} (use claude|opencode|cursor)" >&2
      return 1
      ;;
  esac
}

echo "[sdd-ralph-loop] change=${CHANGE} max=${MAX} agent=${AGENT}" >&2

for ((i = 1; i <= MAX; i++)); do
  echo "" >&2
  echo "=== Ralph iteration ${i}/${MAX} ===" >&2

  if [[ -n "${SDD_RALPH_DRY_RUN:-}" ]]; then
    echo "[DRY-RUN] would invoke ${AGENT} with sdd-loop prompt" >&2
    break
  fi

  result="$(run_agent "$LOOP_PROMPT" 2>&1)" || true
  echo "$result"

  if [[ "$result" == *"<promise>COMPLETE</promise>"* ]] || [[ "$result" == *"{COMPLETE}"* ]]; then
    echo "[sdd-ralph-loop] PRD complete after ${i} iteration(s)." >&2
    exit 0
  fi
done

echo "[sdd-ralph-loop] Stopped after ${MAX} iterations (no completion sigil)." >&2
echo "Resume with: /sdd-loop ${CHANGE}  or  $0 ${CHANGE} ${MAX}" >&2
exit 0
