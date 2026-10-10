#!/usr/bin/env node
// ticketing-post-merge.js — post-merge tracker consistency check (advisory).
//
// Merges can introduce duplicate Backlog task IDs (two branches creating the
// same next ID) — the file-based equivalent of what `bd` re-hydrates after a
// merge. Runs the tracker's read-only diagnosis and reports; always exits 0
// (git ignores post-merge exit codes anyway, and a down tracker must never
// alarm a merge).
//
// Backlogmd: `backlog doctor` (no flags = read-only diagnosis) with a 60s
// timeout. gh/glab/jira: nothing local to reconcile — exit 0.
//
// CLI-ready: called by checkpoint-state.js post-merge-check.
'use strict';

const { spawnSync } = require('node:child_process');
const ticketing = require('./ticketing');

const root = ticketing.repoRoot();
if (!root) process.exit(0);

const cfg = ticketing.readTicketingConfig(root);
if (!cfg.enabled) process.exit(0);
if (cfg.type !== 'backlogmd') process.exit(0);

const probe = spawnSync('backlog', ['--help'], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
if (probe.status !== 0) process.exit(0); // CLI missing/broken: fail-open

const r = spawnSync('backlog', ['doctor'], {
  cwd: root,
  encoding: 'utf8',
  stdio: ['ignore', 'pipe', 'pipe'],
  timeout: 60000,
});
if (r.error || r.status !== 0) {
  process.stderr.write('WARNING: tracker consistency check needs attention (ticketing).\n');
  const out = ((r.stdout || '') + (r.stderr || '')).trim().split('\n').slice(0, 15).join('\n');
  if (out) process.stderr.write(`${out}\n`);
  process.stderr.write('RECOVERY: run `backlog doctor` (add `--fix --yes` only after reviewing the preview).\n');
}
process.exit(0);
