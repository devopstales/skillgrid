#!/usr/bin/env bash
# check-lens-contract.sh — machine-checkable completion contract for the
# parallel-code-review specialist lenses.
#
# "A rule asks; a hook guarantees." This is the guarantee for the review lens
# output contract. Each lens in .agents/skills/parallel-code-review/reviewers/
# promises a specific output shape (a JSON array with required keys, or a
# structured block). If a lens's declared output drifts from what this script
# expects, the check FAILS — a stale contract is a broken guarantee, the same
# class of bug gsd's `check:contract-drift` treats as a build failure.
#
# The single source of truth for the contract is this file's LENS_CONTRACTS
# table below. CONTRIBUTING.md names it as such. Change a lens's output shape
# and you MUST change the matching row here (or the check tells you to).
#
# Usage:
#   bash scripts/check-lens-contract.sh          # check every lens
#   bash scripts/check-lens-contract.sh --json   # machine-readable status
#
# Exit code: 0 if every lens matches its contract, 1 if any drift.
#
# # FUTURE CLI ---------------------------------------------------------------
# This script is the reference implementation. When skillgrid gains a CLI, this
# becomes a verb rather than a standalone script:
#
#   skillgrid check lens-contract            # same as this script
#   skillgrid check lens-contract --changed <ref>   # scope to lenses touched since <ref>
#
# The CLI would wrap this exact logic (extract the per-lens expectation, grep the
# lens file, compare) and add:
#   - `--changed <ref>`: run `git diff --name-only <ref> HEAD -- .agents/skills/
#     parallel-code-review/reviewers/` and check only the touched lenses, so a
#     PR that doesn't touch a lens never fails on it.
#   - machine-readable status on stdout (a JSON line per lens: name, expected,
#     found, ok) so CI can render per-lens results and a future `--fail-on-warn`.
#   - a `--list` that prints the contract table without checking.
#
# Until then, this script keeps working standalone — wire it into CI as a
# second check alongside test-hooks.sh, and add a row to LENS_CONTRACTS for any
# new lens. The table, not the prose in each lens, is the contract.
# -----------------------------------------------------------------------------
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LENSES="$ROOT/.agents/skills/parallel-code-review/reviewers"
JSON="${1:-}"

# LENS_CONTRACTS: one row per lens, "filename|kind|required-keys-or-pattern".
#   kind = json  -> the lens must declare each of the comma-separated keys in a
#                   JSON finding object (grep for the literal key string).
#   kind = block -> the lens must declare each of the comma-separated block
#                   field markers (grep for the literal marker string).
# A lens file missing a required marker = DRIFT.
LENS_CONTRACTS="
security.md|json|location,issue,attack_or_exposure,fix
accessibility.md|json|location,issue,who_it_blocks,fix
performance.md|json|location,issue,cost_and_scale,fix
edge-case-hunter.md|json|location,trigger_condition,guard_snippet,potential_consequence,kind
verification-gap.md|block|Changed surface,Impacted consumer,Existing test evidence,Missing verification,Demonstration,Disposition,gap_type
red-team.md|json|location,issue,missed_by,consequence
"

FAIL=0
LINES=()

# check_json <file> <comma-keys>
check_json() {
  local file="$1" keys="$2" k
  local ok=0
  IFS=',' read -r -a arr <<< "$keys"
  for k in "${arr[@]}"; do
    if ! grep -q "\"${k}\"" "$file"; then
      LINES+=("DRIFT  $(basename "$file"): missing json key \"${k}\"")
      ok=1
    fi
  done
  # a json lens must actually emit a JSON array
  if ! grep -q 'Return ONLY a valid JSON array' "$file"; then
    LINES+=("DRIFT  $(basename "$file"): missing 'Return ONLY a valid JSON array'")
    ok=1
  fi
  return "$ok"
}

# check_block <file> <comma-markers>
check_block() {
  local file="$1" markers="$2" m
  local ok=0
  IFS=',' read -r -a arr <<< "$markers"
  for m in "${arr[@]}"; do
    if ! grep -qF "$m" "$file"; then
      LINES+=("DRIFT  $(basename "$file"): missing block marker '$m'")
      ok=1
    fi
  done
  return "$ok"
}

while IFS='|' read -r fname kind spec; do
  [ -z "${fname// /}" ] && continue
  file="$LENSES/$fname"
  if [ ! -f "$file" ]; then
    LINES+=("DRIFT  ${fname}: lens file not found")
    FAIL=1
    continue
  fi
  case "$kind" in
    json)  check_json  "$file" "$spec" ;;
    block) check_block "$file" "$spec" ;;
    *) LINES+=("DRIFT  ${fname}: unknown contract kind '$kind'"); FAIL=1; continue ;;
  esac
  [ $? -ne 0 ] && FAIL=1
done <<< "$LENS_CONTRACTS"

LENS_COUNT=0
while IFS='|' read -r fname kind spec; do
  [ -z "${fname// /}" ] && continue
  LENS_COUNT=$((LENS_COUNT + 1))
done <<< "$LENS_CONTRACTS"

if [ "$FAIL" -eq 0 ]; then
  LINES+=("OK     all ${LENS_COUNT} lens contracts match")
fi

if [ "$JSON" = "--json" ]; then
  for line in "${LINES[@]}"; do
    printf '{"status":"%s"}\n' "$line"
  done
else
  for line in "${LINES[@]}"; do printf '%s\n' "$line"; done
  printf '\n========================================\n'
  if [ "$FAIL" -eq 0 ]; then
    printf 'Lens contract: PASS\n'
  else
    printf 'Lens contract: FAIL — a lens drifted from its declared output contract.\n'
    printf 'Update scripts/check-lens-contract.sh (LENS_CONTRACTS) or the lens file.\n'
  fi
fi

[ "$FAIL" -eq 0 ] && exit 0 || exit 1
