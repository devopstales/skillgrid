#!/usr/bin/env bash
# Smoke tests for verify-before-review gate ordering.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
GATE="${ROOT}/.skillgrid/scripts/sdd-gate.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

setup_change() {
  local change="$1"
  mkdir -p "${TMP}/openspec/changes/${change}/specs/slice"
  mkdir -p "${TMP}/.skillgrid/tasks/research/${change}"
  mkdir -p "${TMP}/.skillgrid/state/${change}"
  cat > "${TMP}/openspec/changes/${change}/proposal.md" <<'EOF'
# Proposal
EOF
  cat > "${TMP}/openspec/changes/${change}/design.md" <<'EOF'
# Design
EOF
  cat > "${TMP}/openspec/changes/${change}/tasks.md" <<'EOF'
- [ ] Task one [Label: AFK]
EOF
  cat > "${TMP}/openspec/changes/${change}/specs/slice/spec.md" <<'EOF'
# Spec
EOF
}

run_gate() {
  local phase="$1"
  local change="$2"
  SKILLGRID_REPO_ROOT="$TMP" "$GATE" "$phase" --change "$change" --skip persona --skip persona_hardgates 2>/dev/null
}

assert_fail() {
  local phase="$1"
  local change="$2"
  if run_gate "$phase" "$change"; then
    echo "FAIL: expected gate ${phase} to fail for ${change}" >&2
    exit 1
  fi
}

assert_pass() {
  local phase="$1"
  local change="$2"
  if ! run_gate "$phase" "$change"; then
    echo "FAIL: expected gate ${phase} to pass for ${change}" >&2
    exit 1
  fi
}

CHANGE="test-verify-order"
setup_change "$CHANGE"

echo "== review blocked without verify =="
assert_fail review "$CHANGE"

echo "== review allowed after verify-report PASS =="
cat > "${TMP}/openspec/changes/${CHANGE}/verify-report.md" <<'EOF'
### Verdict
PASS
EOF
assert_pass review "$CHANGE"

echo "== archive blocked without review =="
assert_fail archive "$CHANGE"

echo "== archive allowed after review APPROVED =="
mkdir -p "${TMP}/openspec/changes/${CHANGE}/reviews"
cat > "${TMP}/openspec/changes/${CHANGE}/reviews/2026-01-01-review.md" <<'EOF'
Recommendation: APPROVED
EOF
assert_pass archive "$CHANGE"

echo "== verify precheck does not require review artifacts =="
CHANGE2="test-verify-only"
setup_change "$CHANGE2"
assert_pass verify "$CHANGE2"

echo "All sdd-gate ordering tests passed."
