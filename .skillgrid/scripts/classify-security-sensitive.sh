#!/usr/bin/env bash
# classify-security-sensitive.sh — Tag a change as security-sensitive from git diff.
#
# Usage:
#   classify-security-sensitive.sh --change <change-id> [--base <ref>]
#
# Writes .skillgrid/state/<change>/security_sensitive when matched.
# Exit 0 always (classification is advisory for verify; review may require heimdall).

set -euo pipefail

REPO_ROOT="${SKILLGRID_REPO_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}"
cd "$REPO_ROOT"

CHANGE=""
BASE_REF="${SKILLGRID_BASE_REF:-origin/main}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --change) CHANGE="$2"; shift 2 ;;
    --base) BASE_REF="$2"; shift 2 ;;
    -h|--help)
      echo "Usage: classify-security-sensitive.sh --change <id> [--base <ref>]"
      exit 0
      ;;
    *) echo "Unknown arg: $1" >&2; exit 2 ;;
  esac
done

[[ -n "$CHANGE" ]] || { echo "Missing --change" >&2; exit 2; }

STATE_DIR=".skillgrid/state/${CHANGE}"
mkdir -p "$STATE_DIR"
OUT="${STATE_DIR}/security_sensitive"
REASONS=()

if ! git rev-parse --verify "$BASE_REF" >/dev/null 2>&1; then
  BASE_REF="HEAD~1"
fi

MERGE_BASE="$(git merge-base HEAD "$BASE_REF" 2>/dev/null || echo "HEAD~20")"
FILES=()
while IFS= read -r line; do
  [[ -n "$line" ]] && FILES+=("$line")
done < <(git diff --name-only "$MERGE_BASE" HEAD 2>/dev/null || true)

if [[ ${#FILES[@]} -eq 0 ]]; then
  echo "security_sensitive=false (no diff files)"
  rm -f "$OUT"
  exit 0
fi

PATH_RE='auth|oauth|jwt|session|crypto|encrypt|password|secret|credential|permission|rbac|iam|webhook|upload|sql|migration|middleware|security|trivy|helm|terraform|dockerfile|\.env|policy|cors|csrf|xss|sanitize'
for f in "${FILES[@]}"; do
  case "$f" in
    .agents/skills/*|.agents/skills_back/*)
      # Hub skill tree: only count paths for this change's OpenSpec slice
      case "$f" in
        "openspec/changes/${CHANGE}/"*) ;;
        *) continue ;;
      esac
      ;;
  esac
  lower=$(echo "$f" | tr '[:upper:]' '[:lower:]')
  if echo "$lower" | grep -qE "$PATH_RE"; then
    REASONS+=("$f (path match)")
  fi
done

# Spec/design explicit security flag
for marker in "openspec/changes/${CHANGE}/spec.md" "openspec/changes/${CHANGE}/design.md"; do
  if [[ -f "$marker" ]] && grep -qiE 'security-sensitive|security sensitive|authz|authentication|authorization|encryption|secret' "$marker" 2>/dev/null; then
    REASONS+=("$marker (spec/design security language)")
  fi
done

if [[ ${#REASONS[@]} -gt 0 ]]; then
  {
    echo "security_sensitive=true"
    echo "classified_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo "base_ref=${BASE_REF}"
    echo "reasons:"
    printf '  - %s\n' "${REASONS[@]}"
  } >"$OUT"
  echo "security_sensitive=true (${#REASONS[@]} signals) → $OUT"
else
  rm -f "$OUT"
  echo "security_sensitive=false"
fi

exit 0
