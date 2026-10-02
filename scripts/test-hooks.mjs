#!/usr/bin/env node
// test-hooks.mjs — exercise the skillgrid guard hooks against isolated throwaway
// git repos so the enforcement layer is proven, not assumed.
//
// "A rule asks; a hook guarantees." The hooks ARE the guarantee, so they get
// real test coverage: each case builds a temp repo, stages/commits a fixture,
// runs the hook (or the checkpoint-state.js subcommand that drives it), and
// asserts the exit code. No fixtures leak into the real repo; the temp repo is
// under an os.tmpdir() dir and torn down on exit.
//
// Usage:
//   node scripts/test-hooks.mjs          # run everything
//   node scripts/test-hooks.mjs guard-msg # run only cases tagged guard-msg
//
// Exit code: 0 if all pass, 1 if any fail.
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const HOOKS = path.join(ROOT, 'hooks');
const CHECKPOINT = path.join(HOOKS, 'checkpoint-state.js');
const ZONE = path.join(HOOKS, 'precommit-zone-guard.js');
const GUARD = path.join(HOOKS, 'precommit-guard.js');
const STOP_TESTS = path.join(HOOKS, 'stop-tests.js');
const IGNORE_GUARD = path.join(HOOKS, 'precommit-ignore-guard.js');
const GATE_LINT = path.join(HOOKS, 'gate-lint.js');

const FILTER = process.argv[2] || '';

let PASS = 0;
let FAIL = 0;
const RESULTS = [];

const TMP_ROOT = fs.mkdtempSync(path.join(os.tmpdir(), 'skillgrid-hooks.'));
const WORK = path.join(TMP_ROOT, 'work');
const STATE = path.join(TMP_ROOT, 'checkpoint.json');
fs.mkdirSync(WORK, { recursive: true });

process.on('exit', () => {
  try { fs.rmSync(TMP_ROOT, { recursive: true, force: true }); } catch { /* best effort */ }
});

// --- helpers -----------------------------------------------------------------

function sh(cmd, args, opts = {}) {
  const r = spawnSync(cmd, args, {
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
    ...opts,
  });
  if (r.error && r.status === null) process.exitCode = 127;
  return r;
}

function git(repo, ...args) {
  return sh('git', args, { cwd: repoPath(repo) });
}

function repoPath(name) {
  return path.join(WORK, name);
}

function newRepo(name) {
  const p = repoPath(name);
  fs.rmSync(p, { recursive: true, force: true });
  fs.mkdirSync(p, { recursive: true });
  sh('git', ['init', '-q', '-b', 'main', '.'], { cwd: p });
  sh('git', ['config', 'user.email', 'hook-test@skillgrid.local'], { cwd: p });
  sh('git', ['config', 'user.name', 'Hook Test'], { cwd: p });
  sh('git', ['config', 'commit.gpgsign', 'false'], { cwd: p });
}

function writeFile(repo, rel, content) {
  const p = path.join(repoPath(repo), rel);
  fs.mkdirSync(path.dirname(p), { recursive: true });
  fs.writeFileSync(p, content);
}

// Run `node <script> [args]` in a repo (or with no repo, plain cwd ROOT),
// capturing combined output.
function runHook(script, repo, args = [], env = {}) {
  const cwd = repo ? repoPath(repo) : ROOT;
  return sh(process.execPath, [script, ...args], { cwd, env: { ...process.env, ...env } });
}

function expectRC(name, want, got) {
  if (got === want) {
    PASS += 1;
    RESULTS.push(`PASS  ${name}`);
  } else {
    FAIL += 1;
    RESULTS.push(`FAIL  ${name} (want rc=${want}, got rc=${got})`);
  }
}

function expect(name, want, repo, script, args = [], env = {}) {
  const r = runHook(script, repo, args, env);
  expectRC(name, want, r.status === null ? 1 : r.status);
}

// --- guard-msg (commit-message validation) -----------------------------------
// Driven directly: no repo needed. The message file is a temp path.

function caseGuardMsg() {
  if (FILTER && FILTER !== 'guard-msg') return;

  const good = path.join(TMP_ROOT, 'good.msg');
  const bad1 = path.join(TMP_ROOT, 'bad1.msg');
  const bad2 = path.join(TMP_ROOT, 'bad2.msg');

  fs.writeFileSync(good, 'feat(auth): add session refresh\n');
  expectRC('guard-msg: conventional subject passes', 0,
    runHook(CHECKPOINT, null, ['guard-msg', good]).status);

  fs.writeFileSync(bad1, 'updated stuff\n');
  expectRC('guard-msg: non-conventional subject fails', 1,
    runHook(CHECKPOINT, null, ['guard-msg', bad1]).status);

  fs.writeFileSync(bad2, 'feat(auth): add session refresh\n\nCo-Authored-By: Cursor <cursor@cursor.com>\n');
  expectRC('guard-msg: Co-Authored-By trailer fails', 1,
    runHook(CHECKPOINT, null, ['guard-msg', bad2]).status);

  // Cursor's shell integration writes the trailer in lowercase; the guard
  // must reject every casing, not just the canonical one.
  const bad3 = path.join(TMP_ROOT, 'bad3.msg');
  fs.writeFileSync(bad3, 'feat(auth): add session refresh\n\nCo-authored-by: Cursor <cursoragent@cursor.com>\n');
  expectRC('guard-msg: lowercase Co-authored-by trailer fails', 1,
    runHook(CHECKPOINT, null, ['guard-msg', bad3]).status);
}

// --- precommit-zone-guard (BDD spec-zone XOR code-zone) ----------------------

function caseZone() {
  if (FILTER && FILTER !== 'zone') return;

  // spec-only commit: allow
  newRepo('zone-spec');
  writeFile('zone-spec', '.skillgrid/specs/t1/acceptance.feature', 'Feature: t1\n');
  git('zone-spec', 'add', '-A');
  expect('zone: spec-only commit allowed', 0, 'zone-spec', ZONE);

  // code-only commit: allow
  newRepo('zone-code');
  writeFile('zone-code', 'src/a.js', 'x=1\n');
  git('zone-code', 'add', '-A');
  expect('zone: code-only commit allowed', 0, 'zone-code', ZONE);

  // mixed spec + code: block
  newRepo('zone-mixed');
  writeFile('zone-mixed', '.skillgrid/specs/t1/acceptance.feature', 'Feature: t1\n');
  writeFile('zone-mixed', 'src/a.js', 'x=1\n');
  git('zone-mixed', 'add', '-A');
  expect('zone: mixed spec+code commit blocked', 1, 'zone-mixed', ZONE);

  // archive-only commit: allow (archive is the terminal state of spec artifacts)
  newRepo('zone-archive');
  writeFile('zone-archive', '.skillgrid/archive/t1/acceptance.feature', 'Feature: t1\n');
  git('zone-archive', 'add', '-A');
  expect('zone: archive-only commit allowed', 0, 'zone-archive', ZONE);

  // archive + specs commit: allow (both spec-zone)
  newRepo('zone-spec-archive');
  writeFile('zone-spec-archive', '.skillgrid/specs/t1/tasks.md', 'a\n');
  writeFile('zone-spec-archive', '.skillgrid/archive/t1/tasks.md', 'b\n');
  git('zone-spec-archive', 'add', '-A');
  expect('zone: specs+archive commit allowed', 0, 'zone-spec-archive', ZONE);

  // nothing staged: allow (guard defers to others)
  newRepo('zone-empty');
  expect('zone: nothing staged allowed', 0, 'zone-empty', ZONE);
}

// --- precommit-guard (protected-ref / detached) ------------------------------

function caseProtected() {
  if (FILTER && FILTER !== 'protected') return;

  // commit on protected branch 'main' with history: block
  newRepo('prot-main');
  writeFile('prot-main', 'a.txt', 'a\n');
  git('prot-main', 'add', '-A');
  git('prot-main', 'commit', '-q', '-m', 'chore: init');
  expect('protected: commit on main blocked', 1, 'prot-main', GUARD, ['guard']);

  // commit on a feature branch: allow
  newRepo('prot-feat');
  writeFile('prot-feat', 'a.txt', 'a\n');
  git('prot-feat', 'add', '-A');
  git('prot-feat', 'commit', '-q', '-m', 'chore: init');
  git('prot-feat', 'checkout', '-q', '-b', 'feature/x');
  expect('protected: commit on feature branch allowed', 0, 'prot-feat', GUARD, ['guard']);

  // detached HEAD (at a non-initial commit): block. (Detaching at the *init*
  // commit makes `--abbrev-ref HEAD` report "HEAD", which the guard allows as an
  // unborn-branch edge case — that is a separate, known nuance, not this path.)
  newRepo('prot-detached');
  writeFile('prot-detached', 'a.txt', 'a\n');
  git('prot-detached', 'add', '-A');
  git('prot-detached', 'commit', '-q', '-m', 'chore: init');
  writeFile('prot-detached', 'b.txt', 'b\n');
  git('prot-detached', 'add', '-A');
  git('prot-detached', 'commit', '-q', '-m', 'chore: second');
  git('prot-detached', 'checkout', '-q', '--detach', 'HEAD~1');
  expect('protected: detached HEAD blocked', 1, 'prot-detached', GUARD, ['guard']);
}

// --- stop-tests (Stop-phase test gate) ---------------------------------------

function caseStop() {
  if (FILTER && FILTER !== 'stop') return;

  // no testing.runner configured -> allow (degrade to no-op)
  newRepo('stop-none');
  expect('stop: no runner configured allows', 0, 'stop-none', STOP_TESTS);

  // failing runner -> block (run inside the repo so git toplevel resolves)
  newRepo('stop-fail');
  expectRC('stop: failing runner blocks', 2,
    runHook(STOP_TESTS, 'stop-fail', [], { SKILLGRID_TEST_CMD: "bash -c 'exit 1'" }).status);

  // passing runner -> allow
  expectRC('stop: passing runner allows', 0,
    runHook(STOP_TESTS, 'stop-fail', [], { SKILLGRID_TEST_CMD: "bash -c 'exit 0'" }).status);
}

// --- precommit-ignore-guard (.extracted/ + worktree auto-git) -----------------

function caseIgnore() {
  if (FILTER && FILTER !== 'ignore') return;

  // staging an unignored .extracted/ path: block + auto-add to .gitignore
  newRepo('ign-extracted');
  writeFile('ign-extracted', 'acceptance-tests/.extracted/t.feature', 'gen\n');
  git('ign-extracted', 'add', '-A');
  expect('ignore: staged .extracted/ blocked', 1, 'ign-extracted', IGNORE_GUARD);

  const gi = path.join(repoPath('ign-extracted'), '.gitignore');
  const hasLine = fs.existsSync(gi) &&
    fs.readFileSync(gi, 'utf8').split('\n').some((l) => l === 'acceptance-tests/.extracted/');
  expectRC('ignore: auto-added .extracted/ to .gitignore', 0, hasLine ? 0 : 1);
}

// --- gate-lint (advisory: warnings only, always exit 0) ----------------------

function caseLint() {
  if (FILTER && FILTER !== 'lint') return;

  // a well-formed spec with a gate: lint runs, exit 0 (warnings only)
  newRepo('lint-good');
  writeFile('lint-good', '.skillgrid/specs/t1/acceptance.feature',
    ['# Cap: t1', '## Requirement: thing', '### Scenario: happy path', 'G1:',
     '  CHECK: echo 5', '  EXPECT: 5', ''].join('\n'));
  expect('lint: well-formed gate exits 0', 0, 'lint-good', GATE_LINT);

  // a malformed gate (CHECK without EXPECT): still exit 0 (advisory), but warns
  newRepo('lint-bad');
  writeFile('lint-bad', '.skillgrid/specs/t1/acceptance.feature',
    ['# Cap: t1', '## Requirement: thing', '### Scenario: happy path', 'G1:',
     '  CHECK: echo 5', ''].join('\n'));
  expect('lint: malformed gate still exits 0 (advisory)', 0, 'lint-bad', GATE_LINT);

  // and it must actually warn
  const out = `${runHook(GATE_LINT, 'lint-bad').stdout || ''}\n${runHook(GATE_LINT, 'lint-bad').stderr || ''}`;
  expectRC('lint: malformed gate emits a warning', 0, out.includes('missing EXPECT') ? 0 : 1);
}

// --- snapshot/restore (removed — must report unknown subcommand) -------------
// TICKET-07 (old-surfaces-removed): the checkpoint file is gone; resume reads
// the session event stream. Both subcommands fail closed with exit 2 and an
// unknown-subcommand message, and no checkpoint file is written.

function caseSnapshot() {
  if (FILTER && FILTER !== 'snapshot') return;

  newRepo('snap');
  writeFile('snap', 'a.txt', 'a\n');
  git('snap', 'add', '-A');
  git('snap', 'commit', '-q', '-m', 'feat(x): add a');

  const snapR = runHook(CHECKPOINT, 'snap', ['snapshot']);
  expectRC('snapshot: removed subcommand exits 2', 2, snapR.status === null ? 1 : snapR.status);
  const snapOut = `${snapR.stdout || ''}${snapR.stderr || ''}`;
  expectRC('snapshot: reports unknown subcommand', 0,
    snapOut.includes('unknown subcommand') ? 0 : 1);

  const restR = runHook(CHECKPOINT, 'snap', ['restore']);
  expectRC('restore: removed subcommand exits 2', 2, restR.status === null ? 1 : restR.status);
  const restOut = `${restR.stdout || ''}${restR.stderr || ''}`;
  expectRC('restore: reports unknown subcommand', 0,
    restOut.includes('unknown subcommand') ? 0 : 1);

  expectRC('snapshot: no checkpoint file written', 0,
    fs.existsSync(STATE) ? 1 : 0);
}

// --- run all (respect filter) ------------------------------------------------

caseGuardMsg();
caseZone();
caseProtected();
caseStop();
caseIgnore();
caseLint();
caseSnapshot();

// --- report ------------------------------------------------------------------

process.stdout.write('\n');
for (const line of RESULTS) process.stdout.write(`${line}\n`);
process.stdout.write('\n========================================\n');
process.stdout.write(`Results: ${PASS} passed, ${FAIL} failed\n`);
process.exit(FAIL === 0 ? 0 : 1);
