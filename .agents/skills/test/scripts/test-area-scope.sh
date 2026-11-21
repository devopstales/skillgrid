#!/usr/bin/env bash
# Tests for scripts/area-scope. Builds throwaway change dirs and asserts the
# area-selection table classifies each change shape correctly.
set -euo pipefail

here=$(cd "$(dirname "$0")/.." && pwd)
scope="$here/scripts/area-scope"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

pass=0; fail=0
check() { # check <label> <haystack> <needle>
  if printf '%s' "$2" | grep -qF "$3"; then
    echo "  ok: $1"; pass=$((pass+1))
  else
    echo "  FAIL: $1 (missing: $3)"; fail=$((fail+1))
  fi
}
notcheck() { # notcheck <label> <haystack> <needle>
  if printf '%s' "$2" | grep -qF "$3"; then
    echo "  FAIL: $1 (unexpected: $3)"; fail=$((fail+1))
  else
    echo "  ok: $1"; pass=$((pass+1))
  fi
}

# --- usage error: missing change dir ----------------------------------------
out=$(bash "$scope" "$work/nope" 2>&1) || rc=$?
[ "${rc:-0}" = 2 ] && { echo "ok: usage error exit 2"; pass=$((pass+1)); } || { echo "FAIL: usage error (exit ${rc:-0})"; fail=$((fail+1)); }
check "usage error message" "$out" "no such change dir"

# --- behavior change, no URL -------------------------------------------------
c="$work/change-auth"; mkdir -p "$c"
printf 'Goal: add OAuth login\nTruth: user can view own profile\n' > "$c/briefing.md"
printf '### Scenario: successful login\n' > "$c/acceptance.feature"
printf 'Tier: T2\nAdd /login route, session middleware, token handling.\n' > "$c/blueprint.md"
printf 'Add login endpoint\n' > "$c/tasks.md"
out=$(bash "$scope" "$c")
check "behavior: intent RUN" "$out" "| Intent | RUN |"
check "behavior: functional RUN" "$out" "| Functional | RUN |"
check "behavior: code quality RUN" "$out" "| Code quality | RUN"
check "behavior: security trivy+owasp" "$out" "trivy + owasp"
check "behavior: review RUN" "$out" "| Review | RUN"
notcheck "behavior: no DAST without URL" "$out" "DAST available"
notcheck "behavior: no browser pass" "$out" "browser pass — "

# --- behavior change, WITH URL ----------------------------------------------
out=$(bash "$scope" "$c" http://localhost:3000)
check "url: browser pass selected" "$out" "browser pass — http://localhost:3000"
check "url: DAST available" "$out" "DAST available"

# --- refactor-only, no requirement -------------------------------------------
r="$work/change-refactor"; mkdir -p "$r"
printf 'Rename a to b\nMove helpers into shared module\n' > "$r/tasks.md"
out=$(bash "$scope" "$r")
check "refactor: intent N/A" "$out" "| Intent | N/A"
check "refactor: functional still RUN" "$out" "| Functional | RUN |"
check "refactor: security trivy only" "$out" "| Security | RUN (trivy) |"
check "refactor: review RUN (diff is the point)" "$out" "the diff is the point"

echo
echo "area-scope: $pass passed, $fail failed"
[ "$fail" = 0 ] || exit 1
