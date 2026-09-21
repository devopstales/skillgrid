#!/usr/bin/env bash
# test-sdd-workspace.sh — prove the per-plan SDD workspace scripts resolve
# isolated directories, keep git status clean, and fail loudly on bad input.
#
# Each case builds a throwaway git repo under a mktemp dir, drops in plan
# fixtures, runs sdd-workspace / task-brief / review-package, and asserts
# exit codes and file placement. No fixtures leak into the real repo; the
# temp root is torn down on exit.
#
# Usage:
#   bash scripts/test-sdd-workspace.sh          # run everything
#   bash scripts/test-sdd-workspace.sh workspace # run only cases tagged workspace
#
# Exit code: 0 if all pass, 1 if any fail.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE="$ROOT/scripts/sdd-workspace"
BRIEF="$ROOT/scripts/task-brief"
REVIEW="$ROOT/scripts/review-package"

FILTER="${1:-}"

PASS=0
FAIL=0
declare -a RESULTS

TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/skillgrid-sdd-workspace.XXXXXX")"
WORK="$TMP_ROOT/work"
# Hermetic git fixtures: the ambient global core.hooksPath (skillgrid guards
# refuse commits on protected branches) must not leak into throwaway repos.
NO_HOOKS="$TMP_ROOT/no-hooks"
mkdir -p "$NO_HOOKS"

cleanup() { rm -rf "$TMP_ROOT"; }
trap cleanup EXIT

# --- helpers -----------------------------------------------------------------

new_repo() {
  local name="$1"
  local path="$WORK/$name"
  rm -rf "$path"
  mkdir -p "$path"
  (
    cd "$path"
    git init -q -b sdd-test .
    git config user.email "sdd-test@skillgrid.local"
    git config user.name "SDD Test"
    git config commit.gpgsign false
    git config core.hooksPath "$NO_HOOKS"
  )
}

# write_plan <repo> <relpath> — writes a two-task plan fixture. Task 1's
# section embeds a fenced code block containing a fake "## Task 2" heading
# (a naive heading scan would misread it; the fence-aware extractor must not).
write_plan() {
  local repo="$1" rel="$2"
  mkdir -p "$WORK/$repo/$(dirname "$rel")"
  cat > "$WORK/$repo/$rel" <<'EOF'
# Plan: demo change

## Task 1 — first thing

Do the first thing.

```
## Task 2 — fake heading inside a fence, not a real task
```

## Task 2 — second thing

Do the second thing.
EOF
}

expect_rc() { # expect_rc <name> <want_rc> <actual_rc>
  local name="$1" want="$2" got="$3"
  if [ "$got" -eq "$want" ]; then
    PASS=$((PASS + 1)); RESULTS+=("PASS  $name")
  else
    FAIL=$((FAIL + 1)); RESULTS+=("FAIL  $name (want rc=$want, got rc=$got)")
  fi
}

expect_file() { # expect_file <name> <path>
  local name="$1" path="$2"
  if [ -f "$path" ]; then
    PASS=$((PASS + 1)); RESULTS+=("PASS  $name")
  else
    FAIL=$((FAIL + 1)); RESULTS+=("FAIL  $name (missing file: $path)")
  fi
}

# --- workspace: distinct dirs per plan ---------------------------------------

case_workspace() {
  [ "$FILTER" = "workspace" ] || [ -z "$FILTER" ] || return 0

  new_repo ws
  write_plan ws .skillgrid/specs/2026-09-21-alpha/plan-a.md
  write_plan ws .skillgrid/specs/2026-09-21-beta/plan-b.md
  ( cd "$WORK/ws" && git add -A && git commit -q -m "chore: fixture plans" )

  local da db rc
  da=$(cd "$WORK/ws" && bash "$WORKSPACE" .skillgrid/specs/2026-09-21-alpha/plan-a.md); rc=$?
  expect_rc "workspace: plan A resolves (rc 0)" 0 "$rc"
  db=$(cd "$WORK/ws" && bash "$WORKSPACE" .skillgrid/specs/2026-09-21-beta/plan-b.md); rc=$?
  expect_rc "workspace: plan B resolves (rc 0)" 0 "$rc"

  if [ "$da" != "$db" ] && [ -n "$da" ] && [ -n "$db" ]; then
    PASS=$((PASS + 1)); RESULTS+=("PASS  workspace: distinct plans resolve distinct dirs")
  else
    FAIL=$((FAIL + 1)); RESULTS+=("FAIL  workspace: distinct plans resolve distinct dirs (A=$da B=$db)")
  fi

  # resolve via git so symlinked TMPDIRs (macOS /var -> /private/var) compare equal
  local top
  top=$(cd "$WORK/ws" && git rev-parse --show-toplevel)
  case "$da" in
    "$top/.skillgrid/sdd/plan-a") PASS=$((PASS + 1)); RESULTS+=("PASS  workspace: dir is <root>/.skillgrid/sdd/<slug>") ;;
    *) FAIL=$((FAIL + 1)); RESULTS+=("FAIL  workspace: dir is <root>/.skillgrid/sdd/<slug> (got $da)") ;;
  esac

  if [ -d "$da" ] && [ -d "$db" ]; then
    PASS=$((PASS + 1)); RESULTS+=("PASS  workspace: both dirs created")
  else
    FAIL=$((FAIL + 1)); RESULTS+=("FAIL  workspace: both dirs created")
  fi

  # parent .gitignore keeps git status clean
  local gi
  gi="$(cat "$WORK/ws/.skillgrid/sdd/.gitignore")"
  if [ "$gi" = "*" ]; then
    PASS=$((PASS + 1)); RESULTS+=("PASS  workspace: parent .gitignore contains '*'")
  else
    FAIL=$((FAIL + 1)); RESULTS+=("FAIL  workspace: parent .gitignore contains '*' (got '$gi')")
  fi
  ( cd "$WORK/ws" && printf '# SDD ledger — plan: plan-a\n' > "$da/progress.md" && printf 'x\n' > "$da/task-1-brief.md" )
  local porcelain
  porcelain=$(cd "$WORK/ws" && git status --porcelain)
  if [ -z "$porcelain" ]; then
    PASS=$((PASS + 1)); RESULTS+=("PASS  workspace: git status clean with artifacts present")
  else
    FAIL=$((FAIL + 1)); RESULTS+=("FAIL  workspace: git status clean with artifacts present (got: $porcelain)")
  fi
}

# --- workspace: missing plan errors ------------------------------------------

case_missing() {
  [ "$FILTER" = "missing" ] || [ -z "$FILTER" ] || return 0

  new_repo miss
  ( cd "$WORK/miss" && printf 'a\n' > a.txt && git add -A && git commit -q -m "chore: init" )

  ( cd "$WORK/miss" && bash "$WORKSPACE" no-such-plan.md >/dev/null 2>&1 )
  expect_rc "workspace: missing plan file exits 2" 2 "$?"

  ( cd "$WORK/miss" && bash "$WORKSPACE" >/dev/null 2>&1 )
  expect_rc "workspace: missing arg exits 2" 2 "$?"

  ( cd "$WORK/miss" && bash "$BRIEF" no-such-plan.md 1 >/dev/null 2>&1 )
  expect_rc "task-brief: missing plan file exits 2" 2 "$?"

  ( cd "$WORK/miss" && bash "$REVIEW" no-such-plan.md HEAD HEAD >/dev/null 2>&1 )
  expect_rc "review-package: missing plan file exits 2" 2 "$?"

  # no directory may be created on the failure path
  if [ ! -e "$WORK/miss/.skillgrid/sdd" ]; then
    PASS=$((PASS + 1)); RESULTS+=("PASS  workspace: no dir created for missing plan")
  else
    FAIL=$((FAIL + 1)); RESULTS+=("FAIL  workspace: no dir created for missing plan")
  fi
}

# --- task-brief: extraction into the plan's own dir ---------------------------

case_brief() {
  [ "$FILTER" = "brief" ] || [ -z "$FILTER" ] || return 0

  new_repo brief
  write_plan brief .skillgrid/specs/2026-09-21-alpha/plan-a.md
  write_plan brief .skillgrid/specs/2026-09-21-beta/plan-b.md
  ( cd "$WORK/brief" && git add -A && git commit -q -m "chore: fixture plans" )

  ( cd "$WORK/brief" && bash "$BRIEF" .skillgrid/specs/2026-09-21-alpha/plan-a.md 1 >/dev/null 2>&1 )
  expect_rc "task-brief: existing task exits 0" 0 "$?"
  expect_file "task-brief: default outfile lands in plan A's dir" \
    "$WORK/brief/.skillgrid/sdd/plan-a/task-1-brief.md"

  if grep -q "Do the first thing" "$WORK/brief/.skillgrid/sdd/plan-a/task-1-brief.md" \
    && grep -q "fake heading inside a fence" "$WORK/brief/.skillgrid/sdd/plan-a/task-1-brief.md" \
    && ! grep -q "Do the second thing" "$WORK/brief/.skillgrid/sdd/plan-a/task-1-brief.md"; then
    PASS=$((PASS + 1)); RESULTS+=("PASS  task-brief: extracts only Task 1, fence-aware")
  else
    FAIL=$((FAIL + 1)); RESULTS+=("FAIL  task-brief: extracts only Task 1, fence-aware")
  fi

  # plan B's brief must not leak into plan A's dir
  ( cd "$WORK/brief" && bash "$BRIEF" .skillgrid/specs/2026-09-21-beta/plan-b.md 2 >/dev/null 2>&1 )
  expect_rc "task-brief: plan B task exits 0" 0 "$?"
  if [ -f "$WORK/brief/.skillgrid/sdd/plan-b/task-2-brief.md" ] \
    && [ ! -f "$WORK/brief/.skillgrid/sdd/plan-a/task-2-brief.md" ]; then
    PASS=$((PASS + 1)); RESULTS+=("PASS  task-brief: plan B brief stays in plan B dir")
  else
    FAIL=$((FAIL + 1)); RESULTS+=("FAIL  task-brief: plan B brief stays in plan B dir")
  fi

  ( cd "$WORK/brief" && bash "$BRIEF" .skillgrid/specs/2026-09-21-alpha/plan-a.md 99 >/dev/null 2>&1 )
  expect_rc "task-brief: missing task exits 3" 3 "$?"
}

# --- review-package: diff bundle into the plan's own dir ----------------------

case_review() {
  [ "$FILTER" = "review" ] || [ -z "$FILTER" ] || return 0

  new_repo review
  write_plan review .skillgrid/specs/2026-09-21-alpha/plan-a.md
  ( cd "$WORK/review" && git add -A && git commit -q -m "chore: fixture plan" )
  local base head
  base=$(cd "$WORK/review" && git rev-parse HEAD)
  ( cd "$WORK/review" && printf 'v2\n' > feat.txt && git add -A && git commit -q -m "feat(x): add feat" )
  head=$(cd "$WORK/review" && git rev-parse HEAD)

  ( cd "$WORK/review" && bash "$REVIEW" .skillgrid/specs/2026-09-21-alpha/plan-a.md "$base" "$head" >/dev/null 2>&1 )
  expect_rc "review-package: valid range exits 0" 0 "$?"

  local short_base short_head want
  short_base=$(cd "$WORK/review" && git rev-parse --short "$base")
  short_head=$(cd "$WORK/review" && git rev-parse --short "$head")
  want="$WORK/review/.skillgrid/sdd/plan-a/review-${short_base}..${short_head}.diff"
  expect_file "review-package: default outfile lands in plan's dir" "$want"

  if grep -q "## Commits" "$want" && grep -q "## Files changed" "$want" \
    && grep -q "## Diff" "$want" && grep -q "feat.txt" "$want"; then
    PASS=$((PASS + 1)); RESULTS+=("PASS  review-package: bundle holds commits, stat, and diff")
  else
    FAIL=$((FAIL + 1)); RESULTS+=("FAIL  review-package: bundle holds commits, stat, and diff")
  fi

  ( cd "$WORK/review" && bash "$REVIEW" .skillgrid/specs/2026-09-21-alpha/plan-a.md deadbeef "$head" >/dev/null 2>&1 )
  expect_rc "review-package: bad BASE exits 2" 2 "$?"

  ( cd "$WORK/review" && bash "$REVIEW" .skillgrid/specs/2026-09-21-alpha/plan-a.md "$base" deadbeef >/dev/null 2>&1 )
  expect_rc "review-package: bad HEAD exits 2" 2 "$?"
}

# --- ledger convention documented ---------------------------------------------

case_ledger() {
  [ "$FILTER" = "ledger" ] || [ -z "$FILTER" ] || return 0

  if grep -qF '# SDD ledger — plan: <plan-file-path>' "$WORKSPACE"; then
    PASS=$((PASS + 1)); RESULTS+=("PASS  ledger: convention documented in sdd-workspace header")
  else
    FAIL=$((FAIL + 1)); RESULTS+=("FAIL  ledger: convention documented in sdd-workspace header")
  fi
}

# --- run all (respect filter) ------------------------------------------------

case_workspace
case_missing
case_brief
case_review
case_ledger

# --- report ------------------------------------------------------------------

printf '\n'
for line in "${RESULTS[@]}"; do printf '%s\n' "$line"; done
printf '\n========================================\n'
printf 'Results: %d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
