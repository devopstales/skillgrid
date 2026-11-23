#!/usr/bin/env node
// ticketing-pre-push.js — validate Refs/Closes footers on outgoing commits.
//
// Args (from git): <remote-name> <remote-url>; ref lines arrive on stdin as
// `<local-ref> <local-sha> <remote-ref> <remote-sha>`. Reads at most 100
// outgoing commits per ref (local `git log` only — never touches the network).
//
// Fail-closed (exit 1): a `Closes:`/`Fixes:`/`Resolves:` footer with no
// parseable Tracker ID — explicit close intent aimed at nothing is a real
// defect. A malformed `Refs:` is WARNING-only (refs are advisory).
// Fail-open (exit 0): ticketing disabled, no upstream info, deleted refs, or
// any git/stdin error.
//
// CLI-ready: called by checkpoint-state.js pre-push (the pre-push dispatcher).
'use strict';

const ticketing = require('./ticketing');

const root = ticketing.repoRoot();
if (!root) process.exit(0);

const cfg = ticketing.readTicketingConfig(root);
if (!cfg.enabled) process.exit(0);

let stdin = '';
try {
  stdin = require('node:fs').readFileSync(0, 'utf8');
} catch (_e) {
  process.exit(0);
}
const refLines = stdin.split('\n').map((l) => l.trim()).filter(Boolean);
if (refLines.length === 0) process.exit(0); // no ref info (e.g. --dry-run probe)

const ZERO = /^0+$/;
let bodies = [];
for (const line of refLines) {
  const parts = line.split(/\s+/);
  if (parts.length < 4) continue;
  const [, localSha, , remoteSha] = parts;
  if (ZERO.test(localSha)) continue; // ref deleted: nothing outgoing
  const range = ZERO.test(remoteSha) ? localSha : `${remoteSha}..${localSha}`;
  const r = ticketing.git(root, 'log', '--format=%B%x1e%x1e', '--max-count=100', range);
  if (r.status !== 0) continue; // fail-open per ref
  bodies.push(r.stdout || '');
}
if (bodies.length === 0) process.exit(0);

let badCloses = [];
let badRefs = [];
for (const body of bodies.join('').split('\x1e\x1e')) {
  for (const f of ticketing.invalidFooters(body)) {
    if (ticketing.isClosingKeyword(f.keyword)) badCloses.push(f);
    else badRefs.push(f);
  }
}

for (const f of badRefs) {
  process.stderr.write(`WARNING: outgoing commit has unparseable '${f.keyword}: ${f.value}' (no Tracker ID found).\n`);
}

if (badCloses.length === 0) process.exit(0);

const lines = [
  'FATAL: outgoing commit(s) close no parseable Tracker ID (ticketing).',
  '  A Closes/Fixes/Resolves footer with no ID closes nothing and orphans the audit trail.',
];
for (const f of badCloses) lines.push(`    - ${f.keyword}: ${f.value}`);
lines.push('RECOVERY: point the footer at a real ID (`Closes: TASK-069`, `Closes #42`, `Closes: PROJ-123`).');
lines.push('  Amend before pushing; `Refs:` (advisory) never blocks the push.');
process.stderr.write(`${lines.join('\n')}\n`);
process.exit(1);
