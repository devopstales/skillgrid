#!/usr/bin/env node
// precommit-zone-guard.js — enforce the BDD spec-zone rule at commit time.
//
// The rule (simple-execution, subagent-execution, acceptance-test-authoring, qa,
// writing-blueprints): edit `.skillgrid/specs/` OR code in a single commit —
// never both. The spec is the contract; commit it BEFORE the code that satisfies
// it. A commit that stages both a spec file and a source file defeats the
// spec-as-contract model, so this guard FAILs it.
//
// A commit is "mixed" when it stages at least one spec-zone path
// (.skillgrid/specs/** or .skillgrid/archive/** — archive is the terminal state
// of spec artifacts) AND at least one non-spec path. A commit that is
// all-specs, or all-code, passes.
//
// CLI-ready: called by checkpoint-state.js guard (the pre-commit dispatcher).
'use strict';

const { spawnSync } = require('node:child_process');

function git(...args) {
  return spawnSync('git', args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
}

const SPEC_PREFIX = '.skillgrid/specs/';
const SPEC_ARCHIVE_PREFIX = '.skillgrid/archive/';
// Treat acceptance-test sources (not the extracted output) as spec zone too.
const SPEC_EXTRAS = 'acceptance-tests/features/';

const staged = git('diff', '--cached', '--name-only', '--diff-filter=ACMR').stdout
  .split('\n')
  .filter(Boolean);
if (staged.length === 0) process.exit(0); // nothing staged: let the other guards decide

const isSpec = (f) => f.startsWith(SPEC_PREFIX) || f.startsWith(SPEC_ARCHIVE_PREFIX) || f.startsWith(SPEC_EXTRAS);

const hasSpec = staged.some(isSpec);
const hasCode = staged.some((f) => !isSpec(f));

if (hasSpec && hasCode) {
  const lines = [
    'FATAL: mixed spec-zone + code-zone commit (BDD zone rule).',
    '  This commit stages both .skillgrid/specs/** and code in one shot.',
    '  The spec is the contract — commit it BEFORE the code that satisfies it.',
    'RECOVERY: split into two commits.',
    "  1) git add .skillgrid/specs/ && git commit -m 'docs(specs): ...'   # spec first",
    "  2) git add <code files> && git commit -m 'feat(...): ...'          # then code",
    'Mixed staged spec files:',
  ];
  for (const f of staged) {
    if (isSpec(f)) lines.push(`    ${f}`);
  }
  process.stderr.write(`${lines.join('\n')}\n`);
  process.exit(1);
}
process.exit(0);
