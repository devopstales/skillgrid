#!/usr/bin/env node
// state-lock.mjs — advisory session lock for `.skillgrid/state.yaml` writers.
//
// "It does not block — it warns." With parallel development, the lock surfaces
// concurrent sessions touching the same change; different changes proceed
// independently (per _shared/rules/sdd-structure.md § "Session lock").
//
// Lock file: <project-root>/.skillgrid/.lock (JSON):
//   { "change": "<topic>", "session_id": "<id>", "acquired_at": "<ISO8601>" }
//
// Usage:
//   node scripts/state-lock.mjs acquire <topic> [session-id] [project-root]
//   node scripts/state-lock.mjs release <topic> [project-root]
//   node scripts/state-lock.mjs check [topic] [project-root]
//   (project-root defaults to cwd; session-id defaults to
//    $SKILLGRID_SESSION_ID or "unknown"; for `check`, a single positional
//    that is an existing directory is the project root)
//
// acquire:
//   no lock file            → write it, exit 0
//   lock for the SAME change → refresh (acquired_at + session_id), exit 0
//   lock for a DIFFERENT change:
//     acquired_at older than the stale threshold (default 2 hours; override
//     with $SKILLGRID_LOCK_STALE_HOURS for tests) → stale: print WARNING and
//     overwrite, exit 0
//     otherwise → print the conflict (the lock is ADVISORY), exit 1
//   writes atomically (tmp file in the same dir + rename)
//
// release: remove the lock file if present (regardless of owner — advisory);
//   ALWAYS exit 0.
//
// check: print lock status as JSON to stdout and exit 0 (never blocks):
//   {"locked": false} — no lock file
//   {"locked": false, "corrupt": true} — lock file present but unparseable
//   the lock object + "locked": true; if topic given and the lock is for a
//   different change, "conflict": true is added.
//
// A corrupt lock JSON is treated as stale: acquire overwrites it with a
// WARNING, check reports it as corrupt. The script never crashes on it.
//
// Exit codes: 0 acquired/refreshed/stale-overwritten/released/status-printed,
// 1 active lock for a different change (advisory conflict), 2 usage error.
'use strict';

import { existsSync, mkdirSync, readFileSync, renameSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

function usage(msg) {
  if (msg) console.error(`state-lock: ${msg}`);
  console.error('usage: node scripts/state-lock.mjs <acquire|release|check> [args] [project-root]');
  process.exit(2);
}

const argv = process.argv.slice(2);
const cmd = argv[0];
const rest = argv.slice(1);
if (!cmd || !['acquire', 'release', 'check'].includes(cmd)) usage(cmd ? `unknown command: ${cmd}` : 'missing command');

// split positional args: the LAST one is project-root only for commands where
// a root position exists (acquire: 3rd, release: 2nd, check: 2nd)
let topic = null, sessionId = null, root = process.cwd();
const MAX_TOPIC_ONLY = { acquire: 3, release: 2, check: 2 }[cmd];
const pos = rest.filter(a => !a.startsWith('--'));
const flags = rest.filter(a => a.startsWith('--'));
if (flags.length > 0) usage(`unexpected flag(s): ${flags.join(' ')}`);
if (pos.length > MAX_TOPIC_ONLY) usage(`too many arguments for ${cmd}`);
topic = pos[0];
if (cmd === 'acquire') {
  sessionId = pos[1] || process.env.SKILLGRID_SESSION_ID || 'unknown';
  root = pos[2] || process.cwd();
} else if (cmd === 'release') {
  root = pos[1] || process.cwd();
} else { // check
  // `check [topic] [project-root]`: a single positional that is an existing
  // directory is the project root (so `check /path/to/project` works);
  // otherwise it is the topic.
  if (pos.length === 1 && existsSync(pos[0]) && statSync(pos[0]).isDirectory()) {
    root = pos[0];
    topic = null;
  } else {
    topic = pos[0];
    root = pos[1] || process.cwd();
  }
}
if (!topic && cmd !== 'check') usage(`missing <topic> for ${cmd}`);
if (root !== process.cwd() && !existsSync(root)) usage(`no such project root: ${root}`);

const skillgridDir = join(root, '.skillgrid');
const lockPath = join(skillgridDir, '.lock');

const STALE_HOURS = Number.parseFloat(process.env.SKILLGRID_LOCK_STALE_HOURS);
const staleThresholdMs = (Number.isFinite(STALE_HOURS) && STALE_HOURS >= 0 ? STALE_HOURS : 2) * 3600 * 1000;

function ageHours(iso) {
  const t = Date.parse(iso);
  if (Number.isNaN(t)) return Infinity;
  return Math.max(0, (Date.now() - t) / (3600 * 1000));
}

function ageLabel(hours) {
  if (!Number.isFinite(hours)) return 'unknown';
  if (hours < 1) return `${Math.max(1, Math.round(hours * 60))}m`;
  return `${Math.round(hours * 10) / 10}h`;
}

function readLock() {
  // returns { status: 'absent' | 'ok' | 'corrupt', lock }
  if (!existsSync(lockPath)) return { status: 'absent', lock: null };
  try {
    const lock = JSON.parse(readFileSync(lockPath, 'utf8'));
    if (lock && typeof lock === 'object' && typeof lock.change === 'string') return { status: 'ok', lock };
    return { status: 'corrupt', lock: null };
  } catch {
    return { status: 'corrupt', lock: null };
  }
}

function writeLockAtomic(lock) {
  mkdirSync(skillgridDir, { recursive: true });
  const tmp = `${lockPath}.tmp.${process.pid}`;
  writeFileSync(tmp, JSON.stringify(lock, null, 2) + '\n');
  renameSync(tmp, lockPath);
}

if (cmd === 'check') {
  const { status, lock } = readLock();
  if (status === 'absent') console.log('{"locked": false}');
  else if (status === 'corrupt') console.log('{"locked": false, "corrupt": true}');
  else {
    const out = { locked: true, ...lock };
    if (topic && lock.change !== topic) out.conflict = true;
    console.log(JSON.stringify(out));
  }
  process.exit(0);
}

if (cmd === 'release') {
  if (existsSync(lockPath)) rmSync(lockPath);
  process.exit(0); // advisory: always succeeds
}

// cmd === 'acquire'
const { status, lock } = readLock();
const now = new Date().toISOString();

if (status === 'absent') {
  writeLockAtomic({ change: topic, session_id: sessionId, acquired_at: now });
  process.exit(0);
}

if (status === 'corrupt') {
  console.log(`WARNING: corrupt lock at ${lockPath} — treating as stale, overwriting (advisory)`);
  writeLockAtomic({ change: topic, session_id: sessionId, acquired_at: now });
  process.exit(0);
}

if (lock.change === topic) {
  writeLockAtomic({ change: topic, session_id: sessionId, acquired_at: now });
  process.exit(0);
}

const age = ageHours(lock.acquired_at);
if (age * 3600 * 1000 > staleThresholdMs) {
  console.log(`WARNING: stale lock from ${lock.change} (acquired ${ageLabel(age)} ago, session ${lock.session_id || 'unknown'}) — overwriting (advisory)`);
  writeLockAtomic({ change: topic, session_id: sessionId, acquired_at: now });
  process.exit(0);
}

console.log(`conflict: active lock on ${lock.change} by session ${lock.session_id || 'unknown'} (acquired ${ageLabel(age)} ago); the lock is ADVISORY — release it with \`state-lock release ${lock.change}\` or proceed if you own ${lock.change}`);
process.exit(1);
