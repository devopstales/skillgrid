#!/usr/bin/env node
// test-state-lock.mjs — exercise the advisory session lock against a throwaway
// project root: fresh acquire, same-change refresh, different-change conflict
// (advisory, exit 1), stale overwrite (default 2h threshold and the
// $SKILLGRID_LOCK_STALE_HOURS override), release (always exit 0), check JSON
// output, and corrupt-lock handling. No sleeps: staleness is tested with
// backdated timestamps and tiny threshold overrides.
//
// Usage:
//   node scripts/test-state-lock.mjs            # run everything
//   node scripts/test-state-lock.mjs <filter>   # run only cases whose name contains <filter>
//
// Exit code: 0 if all pass, 1 if any fail. Plain node, no dependencies.
'use strict';

import { execFileSync } from 'node:child_process';
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const HERE = dirname(fileURLToPath(import.meta.url));
const LOCK = join(HERE, 'state-lock.mjs');

const filter = process.argv[2] || '';
let pass = 0, fail = 0;
const work = mkdtempSync(join(tmpdir(), 'skillgrid-state-lock-'));

function run(args, env = {}) {
  try {
    const out = execFileSync('node', [LOCK, ...args], { encoding: 'utf8', env: { ...process.env, ...env } });
    return { rc: 0, out };
  } catch (e) {
    return { rc: e.status ?? 1, out: `${e.stdout || ''}${e.stderr || ''}` };
  }
}

function check(label, cond, detail = '') {
  if (cond) { console.log(`  ok: ${label}`); pass++; }
  else { console.log(`  FAIL: ${label}${detail ? ` (${detail})` : ''}`); fail++; }
}

function readLock(root) {
  const p = join(root, '.skillgrid', '.lock');
  if (!existsSync(p)) return null;
  try { return JSON.parse(readFileSync(p, 'utf8')); }
  catch { return null; }
}

function backdate(root, hours) {
  const p = join(root, '.skillgrid', '.lock');
  let lock;
  try { lock = JSON.parse(readFileSync(p, 'utf8')); }
  catch (e) { throw new Error(`backdate: cannot parse lock at ${p}: ${e.message}`); }
  lock.acquired_at = new Date(Date.now() - hours * 3600 * 1000).toISOString();
  writeFileSync(p, JSON.stringify(lock, null, 2) + '\n');
}

function tcase(name, fn) {
  if (filter && !name.includes(filter)) return;
  const root = join(work, name);
  mkdirSync(join(root, '.skillgrid'), { recursive: true });
  console.log(name);
  try { fn(root); }
  catch (e) { console.log(`  FAIL: threw: ${e.message}`); fail++; }
}

try {
  tcase('acquire-fresh', (root) => {
    const r = run(['acquire', 'change-a', 'session-1', root]);
    check('exit 0', r.rc === 0, `rc=${r.rc}`);
    const lock = readLock(root);
    check('lock file written', lock !== null);
    check('change recorded', lock && lock.change === 'change-a');
    check('session_id recorded', lock && lock.session_id === 'session-1');
    check('acquired_at is ISO', lock && !Number.isNaN(Date.parse(lock.acquired_at)));
  });

  tcase('acquire-same-refresh', (root) => {
    run(['acquire', 'change-a', 'session-1', root]);
    const first = readLock(root).acquired_at;
    // let the timestamp move on; then refresh under the same change
    run(['acquire', 'change-a', 'session-2', root]);
    const lock = readLock(root);
    check('exit path kept working', lock !== null);
    check('session_id refreshed', lock.session_id === 'session-2');
    check('acquired_at not older', Date.parse(lock.acquired_at) >= Date.parse(first));
  });

  tcase('acquire-different-conflict', (root) => {
    run(['acquire', 'change-a', 'session-1', root]);
    const r = run(['acquire', 'change-b', 'session-2', root]);
    check('exit 1', r.rc === 1, `rc=${r.rc}`);
    check('names the locked change', r.out.includes('change-a'));
    check('advisory message', /advisory/i.test(r.out));
    check('lock unchanged', readLock(root).change === 'change-a');
  });

  tcase('stale-backdated-overwrite', (root) => {
    run(['acquire', 'change-a', 'session-1', root]);
    backdate(root, 3); // 3h > default 2h threshold
    const r = run(['acquire', 'change-b', 'session-2', root]);
    check('exit 0', r.rc === 0, `rc=${r.rc}`);
    check('stale warning', /stale lock from change-a/i.test(r.out));
    check('lock overwritten', readLock(root).change === 'change-b');
  });

  tcase('stale-env-override', (root) => {
    run(['acquire', 'change-a', 'session-1', root]);
    // fresh lock, but a tiny threshold makes even seconds stale
    const r = run(['acquire', 'change-b', 'session-2', root], { SKILLGRID_LOCK_STALE_HOURS: '0' });
    check('exit 0 with threshold 0', r.rc === 0, `rc=${r.rc}`);
    check('stale warning', /stale/i.test(r.out));
    check('lock overwritten', readLock(root).change === 'change-b');
    // and a large threshold keeps the fresh lock active
    run(['acquire', 'change-c', 'session-3', root]);
    const r2 = run(['acquire', 'change-d', 'session-4', root], { SKILLGRID_LOCK_STALE_HOURS: '100' });
    check('large threshold → conflict exit 1', r2.rc === 1, `rc=${r2.rc}`);
  });

  tcase('release', (root) => {
    run(['acquire', 'change-a', 'session-1', root]);
    let r = run(['release', 'change-a', root]);
    check('release with lock exit 0', r.rc === 0);
    check('lock removed', !existsSync(join(root, '.skillgrid', '.lock')));
    r = run(['release', 'change-a', root]);
    check('release without lock still exit 0', r.rc === 0);
  });

  tcase('check-status', (root) => {
    let r = run(['check', root]);
    check('no lock → {"locked": false}', r.rc === 0 && r.out.includes('{"locked": false}'), r.out);

    run(['acquire', 'change-a', 'session-1', root]);
    r = run(['check', root]);
    let j = null;
    try { j = JSON.parse(r.out); } catch { /* asserted below */ }
    check('locked true', r.rc === 0 && j && j.locked === true, r.out);
    check('lock fields echoed', j && j.change === 'change-a' && j.session_id === 'session-1' && typeof j.acquired_at === 'string');
    check('same topic → no conflict key', j && j.conflict === undefined);

    r = run(['check', 'change-a', root]);
    j = JSON.parse(r.out);
    check('matching topic → no conflict', j.conflict !== true);

    r = run(['check', 'change-b', root]);
    j = JSON.parse(r.out);
    check('different topic → conflict true', j.conflict === true);
    check('check never blocks', r.rc === 0);
  });

  tcase('corrupt-lock', (root) => {
    writeFileSync(join(root, '.skillgrid', '.lock'), 'not-json{');
    let r = run(['check', root]);
    check('check reports corrupt', r.rc === 0 && r.out.includes('{"locked": false, "corrupt": true}'), r.out);

    r = run(['acquire', 'change-a', 'session-1', root]);
    check('acquire overwrites corrupt', r.rc === 0 && /stale/i.test(r.out), r.out);
    const lock = readLock(root);
    check('valid lock after acquire', lock !== null && lock.change === 'change-a');

    // release also handles a corrupt file
    writeFileSync(join(root, '.skillgrid', '.lock'), '{{{');
    r = run(['release', 'change-a', root]);
    check('release corrupt exit 0', r.rc === 0);
    check('corrupt file removed', !existsSync(join(root, '.skillgrid', '.lock')));
  });

  tcase('usage-errors', (root) => {
    let r = run([], root);
    check('no command exit 2', r.rc === 2, `rc=${r.rc}`);
    r = run(['bogus'], root);
    check('unknown command exit 2', r.rc === 2, `rc=${r.rc}`);
    r = run(['acquire'], root);
    check('acquire without topic exit 2', r.rc === 2, `rc=${r.rc}`);
    r = run(['release'], root);
    check('release without topic exit 2', r.rc === 2, `rc=${r.rc}`);
    r = run(['check', root, 'extra']);
    check('too many args exit 2', r.rc === 2, `rc=${r.rc}`);
  });
} finally {
  rmSync(work, { recursive: true, force: true });
}

console.log('');
console.log(`state-lock: ${pass} passed, ${fail} failed`);
process.exit(fail === 0 ? 0 : 1);
