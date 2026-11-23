#!/usr/bin/env node
// ticketing-guard.js — pre-commit completeness gate for staged tracker files.
//
// Backlog.md (`ticketing.type: backlogmd`) is file-based: the file IS the
// tracker, so a staged task file missing `type:`, `references:`, or a
// Definition of Done is a defect worth blocking — the same class of defect
// the ticketing skill's completeness gate rejects at publish time.
//
// FATAL (exit 1): staged `<backlogDir>/tasks/*.md` missing type, references,
// or DoD items. A missing Implementation Plan is WARNING-only: the `backlog`
// CLI itself creates plan-less triage tasks, so blocking on it would fight
// the CLI (see ticketing.js checkBacklogTask).
//
// Fail-open (exit 0): ticketing disabled, unknown/non-file tracker type
// (gh/glab/jira have no local state to validate — network checks never block
// pre-commit), missing config, or any git error.
//
// CLI-ready: called by checkpoint-state.js guard (the pre-commit dispatcher).
'use strict';

const ticketing = require('./ticketing');

const root = ticketing.repoRoot();
if (!root) process.exit(0); // not a git repo: nothing to guard

const cfg = ticketing.readTicketingConfig(root);
if (!cfg.enabled) process.exit(0);
if (cfg.type !== 'backlogmd') process.exit(0);

const files = ticketing.stagedFiles(root, `${cfg.backlogDir}/tasks`);
if (files.length === 0) process.exit(0);

let failed = [];
for (const f of files) {
  const content = ticketing.stagedContent(root, f);
  if (content === null) continue; // unreadable staged blob: fail-open
  const res = ticketing.checkBacklogTask(content);
  for (const w of res.warnings) {
    process.stderr.write(`WARNING: '${f}': ${w}\n`);
  }
  if (res.fatals.length > 0) failed.push({ file: f, fatals: res.fatals });
}

if (failed.length === 0) process.exit(0);

const lines = [
  'FATAL: staged backlog task(s) fail the completeness gate (ticketing).',
  '  The file is the tracker — a stub without type/references/DoD is a defect.',
];
for (const { file, fatals } of failed) {
  lines.push(`  ${file}:`);
  for (const x of fatals) lines.push(`    - ${x}`);
}
lines.push('RECOVERY: fix via the CLI so metadata stays consistent, then re-stage.');
lines.push('  backlog task edit <ID> --type feature --ref <path> --dod <item>');
lines.push('  (never hand-edit just to satisfy the guard; the CLI owns the format.)');
process.stderr.write(`${lines.join('\n')}\n`);
process.exit(1);
