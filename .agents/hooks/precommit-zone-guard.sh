#!/usr/bin/env bash
# precommit-zone-guard.sh — enforce the BDD spec-zone rule at commit time.
#
# The rule (simple-execution, subagent-execution, acceptance-test-authoring, qa,
# writing-blueprints): edit `.skillgrid/specs/` OR code in a single commit —
# never both. The spec is the contract; commit it BEFORE the code that satisfies
# it. A commit that stages both a spec file and a source file defeats the
# spec-as-contract model, so this guard FAILs it.
#
# A commit is "mixed" when it stages at least one .skillgrid/specs/** path AND
# at least one non-spec path. A commit that is all-specs, or all-code, passes.
#
# CLI-ready: called by checkpoint-state.sh guard (the pre-commit dispatcher).
set -euo pipefail

SPEC_PREFIX=".skillgrid/specs/"
# Treat acceptance-test sources (not the extracted output) as spec zone too.
SPEC_EXTRAS="acceptance-tests/features/"

staged="$(git diff --cached --name-only --diff-filter=ACMR 2>/dev/null || true)"
[ -n "$staged" ] || exit 0   # nothing staged: let the other guards decide

has_spec=0
has_code=0
for f in $staged; do
  if [[ "$f" == "$SPEC_PREFIX"* || "$f" == "$SPEC_EXTRAS"* ]]; then
    has_spec=1
  else
    has_code=1
  fi
done

if [ "$has_spec" = 1 ] && [ "$has_code" = 1 ]; then
  {
    echo "FATAL: mixed spec-zone + code-zone commit (BDD zone rule)."
    echo "  This commit stages both .skillgrid/specs/** and code in one shot."
    echo "  The spec is the contract — commit it BEFORE the code that satisfies it."
    echo "RECOVERY: split into two commits."
    echo "  1) git add .skillgrid/specs/ && git commit -m 'docs(specs): ...'   # spec first"
    echo "  2) git add <code files> && git commit -m 'feat(...): ...'          # then code"
    echo "Mixed staged spec files:"
    for f in $staged; do
      [[ "$f" == "$SPEC_PREFIX"* || "$f" == "$SPEC_EXTRAS"* ]] && echo "    $f"
    done
  } >&2
  exit 1
fi
exit 0
