#!/usr/bin/env bash
# run-truecourse-review.sh — TrueCourse diff analyze + list for sdd-review Stage B.
#
# Usage:
#   .skillgrid/scripts/run-truecourse-review.sh --change <change-id> [--llm] [--full] [--no-llm]
#
# Writes:
#   .agents/reviews/<change>/truecourse-analyze.txt
#   .agents/reviews/<change>/truecourse-violations.txt
#   .agents/reviews/<change>/truecourse-summary.json (parsed counts when possible)
#
# Requires: Node.js >= 20, npx. Baseline: .truecourse/LATEST.json (commit on main).
# Upstream: https://github.com/truecourse-ai/truecourse

set -euo pipefail

REPO_ROOT="${SKILLGRID_REPO_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}"
cd "$REPO_ROOT"

CHANGE=""
MODE="diff"
LLM_FLAG="--no-llm"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --change) CHANGE="$2"; shift 2 ;;
    --llm) LLM_FLAG="--llm"; shift ;;
    --no-llm) LLM_FLAG="--no-llm"; shift ;;
    --full) MODE="full"; shift ;;
    --diff) MODE="diff"; shift ;;
    -h|--help)
      echo "Usage: run-truecourse-review.sh --change <id> [--llm|--no-llm] [--full|--diff]"
      exit 0
      ;;
    *) echo "Unknown arg: $1" >&2; exit 2 ;;
  esac
done

[[ -n "$CHANGE" ]] || { echo "Missing --change" >&2; exit 2; }

if ! command -v npx >/dev/null 2>&1; then
  echo "ERROR: npx not found (Node.js >= 20 required)" >&2
  exit 1
fi

OUT_DIR=".agents/reviews/${CHANGE}"
mkdir -p "$OUT_DIR"

if [[ ! -f ".truecourse/LATEST.json" ]]; then
  echo "WARN: .truecourse/LATEST.json missing — run on main first:" >&2
  echo "  npx -y truecourse analyze --no-llm && git add .truecourse/LATEST.json .truecourse/config.json" >&2
fi

ANALYZE_OUT="${OUT_DIR}/truecourse-analyze.txt"
LIST_OUT="${OUT_DIR}/truecourse-violations.txt"
SUMMARY="${OUT_DIR}/truecourse-summary.json"

if [[ "$MODE" == "diff" ]]; then
  echo "Running: npx -y truecourse analyze --diff ${LLM_FLAG} ..."
  npx -y truecourse analyze --diff ${LLM_FLAG} 2>&1 | tee "$ANALYZE_OUT"
  echo "Running: npx -y truecourse list --diff --limit 50 ..."
  npx -y truecourse list --diff --limit 50 2>&1 | tee "$LIST_OUT"
else
  echo "Running: npx -y truecourse analyze ${LLM_FLAG} ..."
  npx -y truecourse analyze ${LLM_FLAG} 2>&1 | tee "$ANALYZE_OUT"
  echo "Running: npx -y truecourse list --limit 50 ..."
  npx -y truecourse list --limit 50 2>&1 | tee "$LIST_OUT"
fi

# Best-effort summary line extraction for gates
new_issues=0
resolved=0
if grep -qE 'Summary:' "$ANALYZE_OUT" 2>/dev/null; then
  line=$(grep -E 'Summary:' "$ANALYZE_OUT" | tail -1 || true)
  ni=$(echo "$line" | sed -n 's/.*Summary:[[:space:]]*\([0-9]*\) new.*/\1/p')
  rs=$(echo "$line" | sed -n 's/.*\([0-9]*\) resolved.*/\1/p')
  [[ -n "$ni" ]] && new_issues=$ni
  [[ -n "$rs" ]] && resolved=$rs
fi

baseline_status="missing"
[[ -f .truecourse/LATEST.json ]] && baseline_status="present"
llm_json=false
[[ "$LLM_FLAG" == "--llm" ]] && llm_json=true

cat >"$SUMMARY" <<EOF
{
  "change": "${CHANGE}",
  "mode": "${MODE}",
  "llm": ${llm_json},
  "artifacts": {
    "analyze": "${ANALYZE_OUT}",
    "violations": "${LIST_OUT}"
  },
  "baseline": "${baseline_status}",
  "new_issues": ${new_issues},
  "resolved": ${resolved}
}
EOF

echo "TrueCourse review artifacts written to ${OUT_DIR}/"
exit 0
