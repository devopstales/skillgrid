#!/usr/bin/env bash
# test-hooks.sh — exercise the skillgrid guard hooks against isolated throwaway
# git repos so the enforcement layer is proven, not assumed.
#
# "A rule asks; a hook guarantees." The hooks ARE the guarantee, so they get
# real test coverage: each case builds a temp repo, stages/commits a fixture,
# runs the hook (or the checkpoint-state.sh subcommand that drives it), and
# asserts the exit code. No fixtures leak into the real repo; the temp repo is
# under a mktemp dir and torn down on exit.
#
# Usage:
#   bash scripts/test-hooks.sh          # run everything
#   bash scripts/test-hooks.sh guard-msg # run only cases tagged guard-msg
#
# Exit code: 0 if all pass, 1 if any fail.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOOKS="$ROOT/.agents/hooks"
CHECKPOINT="$HOOKS/checkpoint-state.sh"
ZONE="$HOOKS/precommit-zone-guard.sh"
GUARD="$HOOKS/precommit-guard.sh"
STOP_TESTS="$HOOKS/stop-tests.sh"
IGNORE_GUARD="$HOOKS/precommit-ignore-guard.sh"
GATE_LINT="$HOOKS/gate-lint.sh"

FILTER="${1:-}"

PASS=0
FAIL=0
declare -a RESULTS

TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/skillgrid-hooks.XXXXXX")"
WORK="$TMP_ROOT/work"
STATE="$TMP_ROOT/checkpoint.json"

cleanup() { rm -rf "$TMP_ROOT"; }
trap cleanup EXIT

# --- helpers -----------------------------------------------------------------

new_repo() {
  local name
  name="$1"
  local path
  path="$WORK/$name"
  rm -rf "$path"
  mkdir -p "$path"
  (
    cd "$path"
    git init -q -b main .
    git config user.email "hook-test@skillgrid.local"
    git config user.name "Hook Test"
    git config commit.gpgsign false
  )
}

cd_repo() { (cd "$WORK/$1" && "$@" >/dev/null 2>&1); }

run() { # run <repo> "<cmd string>"  -> sets RC
  local repo="$1"; shift
  ( cd "$WORK/$repo" && bash -c "$1" ) >/dev/null 2>&1
  RC=$?
}

expect() { # expect <name> <want_rc> <repo> "<cmd string>"
  local name want repo
  name="$1"; want="$2"; repo="$3"
  run "$repo" "$4"
  if [ "$RC" -eq "$want" ]; then
    PASS=$((PASS + 1)); RESULTS+=("PASS  $name")
  else
    FAIL=$((FAIL + 1)); RESULTS+=("FAIL  $name (want rc=$want, got rc=$RC)")
  fi
}

expect_rc() { # expect_rc <name> <want_rc> <actual_rc>
  local name want got
  name="$1"; want="$2"; got="$3"
  if [ "$got" -eq "$want" ]; then
    PASS=$((PASS + 1)); RESULTS+=("PASS  $name")
  else
    FAIL=$((FAIL + 1)); RESULTS+=("FAIL  $name (want rc=$want, got rc=$got)")
  fi
}

# --- guard-msg (commit-message validation) -----------------------------------
# Driven directly: no repo needed. The message file is a temp path.

case_guard_msg() {
  [ "$FILTER" = "guard-msg" ] || [ -z "$FILTER" ] || return 0

  local good bad1 bad2
  good="$TMP_ROOT/good.msg"; bad1="$TMP_ROOT/bad1.msg"; bad2="$TMP_ROOT/bad2.msg"

  printf 'feat(auth): add session refresh\n' > "$good"
  ( bash "$CHECKPOINT" guard-msg "$good" >/dev/null 2>&1 )
  expect_rc "guard-msg: conventional subject passes" 0 "$?"

  printf 'updated stuff\n' > "$bad1"
  ( bash "$CHECKPOINT" guard-msg "$bad1" >/dev/null 2>&1 )
  expect_rc "guard-msg: non-conventional subject fails" 1 "$?"

  printf 'feat(auth): add session refresh\n\nCo-Authored-By: Cursor <cursor@cursor.com>\n' > "$bad2"
  ( bash "$CHECKPOINT" guard-msg "$bad2" >/dev/null 2>&1 )
  expect_rc "guard-msg: Co-Authored-By trailer fails" 1 "$?"
}

# --- precommit-zone-guard (BDD spec-zone XOR code-zone) ----------------------

case_zone() {
  [ "$FILTER" = "zone" ] || [ -z "$FILTER" ] || return 0

  # spec-only commit: allow
  new_repo zone-spec
  ( cd "$WORK/zone-spec" && mkdir -p .skillgrid/specs/t1 && printf 'Feature: t1\n' > .skillgrid/specs/t1/acceptance.feature && git add -A )
  expect "zone: spec-only commit allowed" 0 zone-spec "bash '$ZONE'"

  # code-only commit: allow
  new_repo zone-code
  ( cd "$WORK/zone-code" && mkdir -p src && printf 'x=1\n' > src/a.js && git add -A )
  expect "zone: code-only commit allowed" 0 zone-code "bash '$ZONE'"

  # mixed spec + code: block
  new_repo zone-mixed
  ( cd "$WORK/zone-mixed" && mkdir -p .skillgrid/specs/t1 src && printf 'Feature: t1\n' > .skillgrid/specs/t1/acceptance.feature && printf 'x=1\n' > src/a.js && git add -A )
  expect "zone: mixed spec+code commit blocked" 1 zone-mixed "bash '$ZONE'"

  # nothing staged: allow (guard defers to others)
  new_repo zone-empty
  expect "zone: nothing staged allowed" 0 zone-empty "bash '$ZONE'"
}

# --- precommit-guard (protected-ref / detached) ------------------------------

case_protected() {
  [ "$FILTER" = "protected" ] || [ -z "$FILTER" ] || return 0

  # commit on protected branch 'main' with history: block
  new_repo prot-main
  ( cd "$WORK/prot-main" && printf 'a\n' > a.txt && git add -A && git commit -q -m "chore: init" )
  expect "protected: commit on 'main' blocked" 1 prot-main "bash '$GUARD' guard"

  # commit on a feature branch: allow
  new_repo prot-feat
  ( cd "$WORK/prot-feat" && printf 'a\n' > a.txt && git add -A && git commit -q -m "chore: init" && git checkout -q -b feature/x )
  expect "protected: commit on feature branch allowed" 0 prot-feat "bash '$GUARD' guard"

  # detached HEAD (at a non-initial commit): block. (Detaching at the *init*
  # commit makes `--abbrev-ref HEAD` report "HEAD", which the guard allows as an
  # unborn-branch edge case — that is a separate, known nuance, not this path.)
  new_repo prot-detached
  ( cd "$WORK/prot-detached" && printf 'a\n' > a.txt && git add -A && git commit -q -m "chore: init" && printf 'b\n' > b.txt && git add -A && git commit -q -m "chore: second" && git checkout -q --detach HEAD~1 )
  expect "protected: detached HEAD blocked" 1 prot-detached "bash '$GUARD' guard"
}

# --- stop-tests (Stop-phase test gate) ---------------------------------------

case_stop() {
  [ "$FILTER" = "stop" ] || [ -z "$FILTER" ] || return 0

  # no testing.runner configured -> allow (degrade to no-op)
  new_repo stop-none
  expect "stop: no runner configured allows" 0 stop-none "bash '$STOP_TESTS'"

  # failing runner -> block (run inside the repo so git toplevel resolves)
  new_repo stop-fail
  ( cd "$WORK/stop-fail" && SKILLGRID_TEST_CMD="bash -c 'exit 1'" bash "$STOP_TESTS" >/dev/null 2>&1 )
  expect_rc "stop: failing runner blocks" 2 "$?"

  # passing runner -> allow
  ( cd "$WORK/stop-fail" && SKILLGRID_TEST_CMD="bash -c 'exit 0'" bash "$STOP_TESTS" >/dev/null 2>&1 )
  expect_rc "stop: passing runner allows" 0 "$?"
}

# --- precommit-ignore-guard (.extracted/ + worktree auto-git) -----------------

case_ignore() {
  [ "$FILTER" = "ignore" ] || [ -z "$FILTER" ] || return 0

  # staging an unignored .extracted/ path: block + auto-add to .gitignore
  new_repo ign-extracted
  ( cd "$WORK/ign-extracted" && mkdir -p acceptance-tests/.extracted && printf 'gen\n' > acceptance-tests/.extracted/t.feature && git add -A )
  expect "ignore: staged .extracted/ blocked" 1 ign-extracted "bash '$IGNORE_GUARD'"
  ( cd "$WORK/ign-extracted" && grep -qxF "acceptance-tests/.extracted/" .gitignore 2>/dev/null )
  expect_rc "ignore: auto-added .extracted/ to .gitignore" 0 "$?"
}

# --- gate-lint (advisory: warnings only, always exit 0) ----------------------

case_lint() {
  [ "$FILTER" = "lint" ] || [ -z "$FILTER" ] || return 0

  # a well-formed spec with a gate: lint runs, exit 0 (warnings only)
  new_repo lint-good
  ( cd "$WORK/lint-good" && mkdir -p .skillgrid/specs/t1 && cat > .skillgrid/specs/t1/acceptance.feature <<'EOF'
# Cap: t1
## Requirement: thing
### Scenario: happy path
G1:
  CHECK: echo 5
  EXPECT: 5
EOF
  )
  expect "lint: well-formed gate exits 0" 0 lint-good "bash '$GATE_LINT'"

  # a malformed gate (CHECK without EXPECT): still exit 0 (advisory), but warns
  new_repo lint-bad
  ( cd "$WORK/lint-bad" && mkdir -p .skillgrid/specs/t1 && cat > .skillgrid/specs/t1/acceptance.feature <<'EOF'
# Cap: t1
## Requirement: thing
### Scenario: happy path
G1:
  CHECK: echo 5
EOF
  )
  expect "lint: malformed gate still exits 0 (advisory)" 0 lint-bad "bash '$GATE_LINT'"
  # and it must actually warn
  ( cd "$WORK/lint-bad" && bash "$GATE_LINT" 2>&1 | grep -q "missing EXPECT" )
  expect_rc "lint: malformed gate emits a warning" 0 "$?"
}

# --- snapshot (checkpoint.json derivation) -----------------------------------

case_snapshot() {
  [ "$FILTER" = "snapshot" ] || [ -z "$FILTER" ] || return 0

  new_repo snap
  ( cd "$WORK/snap" && printf 'a\n' > a.txt && git add -A && git commit -q -m "feat(x): add a" )
  export SKILLGRID_CHECKPOINT_JSON="$STATE"
  ( cd "$WORK/snap" && bash "$CHECKPOINT" snapshot >/dev/null 2>&1 )
  expect_rc "snapshot: writes checkpoint.json" 0 "$?"
  [ -f "$STATE" ] && grep -q '"branch"' "$STATE"
  expect_rc "snapshot: checkpoint.json has branch field" 0 "$?"
  unset SKILLGRID_CHECKPOINT_JSON
}

# --- run all (respect filter) ------------------------------------------------

case_guard_msg
case_zone
case_protected
case_stop
case_ignore
case_lint
case_snapshot

# --- report ------------------------------------------------------------------

printf '\n'
for line in "${RESULTS[@]}"; do printf '%s\n' "$line"; done
printf '\n========================================\n'
printf 'Results: %d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
