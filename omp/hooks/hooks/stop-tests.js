#!/usr/bin/env node
// stop-tests.js — the Stop-phase test gate (colemedin's stop_tests_must_pass).
//
// "A rule asks; a hook guarantees." The execution skills say "commit after the
// gate is green" and "run the full suite once before committing." This is the
// only invariant where the *agent* can claim done while tests are red — and that
// is a real incident. So when the agent tries to STOP (end its turn), this hook
// runs the project's test command and blocks the stop, handing back the failures.
//
// This is an AGENT harness hook (a Stop event), not a git hook — it needs the
// test runner, so it is wired separately and only when a runner is
// configured (staged to ~/.skillgrid/git-hooks/ by `skillgrid install`).
//
// Reads the test command from .skillgrid/config.yaml `testing.runner`. If no
// runner is configured (or the file is absent) it allows the stop (degrades to
// no-op) rather than blocking on nothing.
//
// Override the command with SKILLGRID_TEST_CMD for testing / non-standard stacks.
'use strict';

const { spawnSync } = require('node:child_process');
const fs = require('node:fs');

function git(...args) {
  return spawnSync('git', args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
}

const root = git('rev-parse', '--show-toplevel').stdout.trim();
if (!root) process.exit(0);

function configValue(key, fallback) {
  const file = `${root}/.skillgrid/config.yaml`;
  if (!fs.existsSync(file)) return fallback;
  for (const line of fs.readFileSync(file, 'utf8').split('\n')) {
    const m = line.match(new RegExp(`^[\\s]*${key}:`));
    if (m) {
      const v = line.replace(/^[^:]*:[\s]*/, '').replace(/[\s]*#.*$/, '').replace(/["']/g, '');
      if (v) return v;
    }
  }
  return fallback;
}

const testCmd = process.env.SKILLGRID_TEST_CMD || configValue('runner', '');
if (!testCmd) {
  process.stderr.write('stop-tests: no testing.runner configured — allowing stop.\n');
  process.exit(0);
}

process.stderr.write(`stop-tests: running '${testCmd}' before the agent may stop...\n`);
const r = spawnSync('bash', ['-c', testCmd], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
const out = `${r.stdout || ''}${r.stderr || ''}`;
// Echo only the last 40 lines to stderr (mirrors `| tail -40 >&2`).
const lines = out.split('\n');
process.stderr.write(`${lines.slice(-40).join('\n')}\n`);
if (r.status !== 0) {
  process.stderr.write(
    [
      `STOP BLOCKED: tests are failing (ran: ${testCmd}).`,
      '  Fix the failures, or if they are pre-existing/out-of-scope, state that',
      '  explicitly and re-attempt the stop. A red suite is not a clean stop.',
      '',
    ].join('\n')
  );
  process.exit(2); // non-zero blocks the stop in the harness
}
process.exit(0);
