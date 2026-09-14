#!/usr/bin/env bash
# stop-tests.sh — the Stop-phase test gate (colemedin's stop_tests_must_pass).
#
# "A rule asks; a hook guarantees." The execution skills say "commit after the
# gate is green" and "run the full suite once before committing." This is the
# only invariant where the *agent* can claim done while tests are red — and that
# is a real incident. So when the agent tries to STOP (end its turn), this hook
# runs the project's test command and blocks the stop, handing back the failures.
#
# This is an AGENT harness hook (a Stop event), not a git hook — it needs the
# test runner, so it is installed separately and only when a runner is
# configured. See install-hooks.sh (--with-stop).
#
# Reads the test command from .skillgrid/config.yaml `testing.runner`. If no
# runner is configured (or the file is absent) it allows the stop (degrades to
# no-op) rather than blocking on nothing.
#
# Override the command with SKILLGRID_TEST_CMD for testing / non-standard stacks.
set -euo pipefail

root="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0

config_value() {
  local key="$1" fallback="$2" file="$root/.skillgrid/config.yaml" val
  [ -f "$file" ] || { echo "$fallback"; return; }
  val="$(awk -v k="^[[:space:]]*"k":" '
    $0 ~ k { sub(/^[^:]*:[[:space:]]*/, ""); sub(/[[:space:]]*#.*$/, ""); sub(/["'\'']/g, ""); print; exit }
  ' "$file" 2>/dev/null || true)"
  [ -n "$val" ] && echo "$val" || echo "$fallback"
}

test_cmd="${SKILLGRID_TEST_CMD:-$(config_value runner "")}"
[ -n "$test_cmd" ] || { echo "stop-tests: no testing.runner configured — allowing stop." >&2; exit 0; }

echo "stop-tests: running '$test_cmd' before the agent may stop..." >&2
if ! bash -c "$test_cmd" 2>&1 | tail -40 >&2; then
  {
    echo "STOP BLOCKED: tests are failing (ran: $test_cmd)."
    echo "  Fix the failures, or if they are pre-existing/out-of-scope, state that"
    echo "  explicitly and re-attempt the stop. A red suite is not a clean stop."
  } >&2
  exit 2   # non-zero blocks the stop in the harness
fi
exit 0
