#!/usr/bin/env node
// test-hooks.mjs — exercise the skillgrid v2 guard hooks against isolated
// throwaway git repos so the enforcement layer is proven, not assumed.
//
// "A rule asks; a hook guarantees." The hooks ARE the guarantee, so they get
// real test coverage: each case builds a temp repo (mktemp dir, torn down on
// exit), stages/commits a fixture, runs the hook (or the checkpoint-state.js
// subcommand that drives it), and asserts the exit code + stderr/stdout
// substrings. No fixtures leak into the real repo.
//
// Port of v1 scripts/test-hooks.sh, adapted to v2's guard set:
//   guard-msg (conventional subject, AI trailers, tracker footers)
//   commit-msg hook (AI-trailer strip step, then guard-msg)
//   precommit-zone-guard (BDD spec-zone XOR code-zone)
//   precommit-guard (protected-ref / detached HEAD, env overrides)
//   checkpoint-state.js `guard` dispatch (the pre-commit chain)
//   precommit-ignore-guard (.extracted/ + worktree auto-gitignore)
//   gate-lint (advisory: warnings only, always exit 0)
//   gate-state (--status non-executing / --reverify executing)
//   stop-tests (Stop-phase test gate: red exit 2, green exit 0)
//   ticketing-guard (staged backlog task completeness, fail-open)
//   prepare-commit-msg (SKILLGRID_TRACKER_REF `Refs:` injector)
//   pre-push (ticketing footer check: malformed Closes blocks, Refs warns)
//   pre-commit orchestrator (full git commit through the installed shims)
//
// Usage:
//   node scripts/test-hooks.mjs            # run everything
//   node scripts/test-hooks.mjs <filter>   # run only cases whose name contains <filter>
//
// Exit code: 0 if all pass, 1 if any fail. Plain node, no dependencies.
// Add a case here when you change a hook — a hook with no test is an
// unproven guarantee.
'use strict';

import { spawnSync } from 'node:child_process';
import {
  chmodSync,
  cpSync,
  mkdirSync,
  mkdtempSync,
  readdirSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const HERE = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(HERE, '..');
const HOOKS = join(ROOT, 'git', 'hooks');
const OMPHOOKS = join(ROOT, 'omp', 'hooks', 'hooks');

const FILTER = process.argv[2] || '';

const TMP = mkdtempSync(join(tmpdir(), 'skillgrid-hooks-'));
const WORK = join(TMP, 'work');
mkdirSync(WORK, { recursive: true });
process.on('exit', () => { try { rmSync(TMP, { recursive: true, force: true }); } catch { /* best effort */ } });

// --- run helpers ---------------------------------------------------------------

function run(file, args, { cwd, env = {}, input } = {}) {
  const r = spawnSync(file, args, {
    cwd,
    env: { ...process.env, ...env },
    input,
    encoding: 'utf8',
  });
  return { code: r.status === null ? -1 : r.status, out: r.stdout || '', err: r.stderr || '' };
}

const nodeFile = (file, args, opts) => run(process.execPath, [file, ...args], opts);
const hook = (rel, args, opts) => nodeFile(join(HOOKS, rel), args, opts);
const ompHook = (rel, args, opts) => nodeFile(join(OMPHOOKS, rel), args, opts);
// Pin core.hooksPath to the throwaway repo's own .git/hooks so the developer's
// global hooksPath (e.g. an installed ~/.skillgrid/git-hooks) can neither
// shadow the orchestrator cases nor block the plain commits the fixtures need.
const git = (repo, ...args) =>
  run('git', ['-c', `core.hooksPath=${join(repo, '.git', 'hooks')}`, ...args], { cwd: repo });

function newRepo(name) {
  const p = join(WORK, name);
  rmSync(p, { recursive: true, force: true });
  mkdirSync(p, { recursive: true });
  git(p, 'init', '-q', '-b', 'main');
  git(p, 'config', 'user.email', 'hook-test@skillgrid.local');
  git(p, 'config', 'user.name', 'Hook Test');
  git(p, 'config', 'commit.gpgsign', 'false');
  return p;
}

// First commit on main (the protected-ref rule bites once history exists).
function initRepo(repo) {
  writeFileSync(join(repo, 'seed.txt'), 'seed\n');
  git(repo, 'add', '-A');
  git(repo, 'commit', '-q', '-m', 'chore: init');
}

function file(repo, rel, content) {
  const p = join(repo, rel);
  mkdirSync(dirname(p), { recursive: true });
  writeFileSync(p, content);
  return rel;
}

const stage = (repo, ...rels) => git(repo, 'add', '-A', '--', ...rels);
const onBranch = (repo, b) => git(repo, 'checkout', '-q', '-b', b);

// Install the git-hooks layout into a temp repo: all of git/hooks/ copied into
// .git/hooks/ (the shims resolve checkpoint-state.js as a sibling, so the flat
// copy mirrors the installed layout ~/.skillgrid/git-hooks/ + ~/.skillgrid/hooks/).
function installGitHooks(repo) {
  const dest = join(repo, '.git', 'hooks');
  cpSync(HOOKS, dest, { recursive: true });
  for (const f of readdirSync(dest)) {
    if (f.endsWith('~')) continue;
    chmodSync(join(dest, f), 0o755);
  }
}

// --- fixtures ------------------------------------------------------------------

const MSG_GOOD = 'feat(auth): add session refresh\n';
const MSG_BAD_SUBJECT = 'updated stuff\n';
const MSG_HUMAN_COAUTHOR = 'feat(auth): add session refresh\n\nCo-Authored-By: Jane <jane@example.com>\n';
const MSG_CURSOR = 'feat(auth): add session refresh\n\nCo-Authored-By: Cursor <cursor@cursor.com>\n';
const MSG_BAD_FOOTER = 'feat(auth): add session refresh\n\nRefs: not-a-ticket\n';
const MSG_GOOD_FOOTER = 'feat(auth): add session refresh\n\nRefs: TASK-069\n';

const SPEC_PATH = '.skillgrid/change-specs/t1/acceptance.feature';
// `echo ok`/`ok` keeps the gate out of gate-lint's bare-number tautology
// heuristic (a numeric EXPECT that appears in its own CHECK echo is flagged).
const SPEC_GOOD = '### Requirement: thing\n#### Scenario: happy path\nG1:\n  CHECK: echo ok\n  EXPECT: ok\n';
const SPEC_MISSING_EXPECT = '### Requirement: thing\n#### Scenario: happy path\nG1:\n  CHECK: echo 5\n';
const SPEC_NO_GATE = '### Requirement: thing\n#### Scenario: happy path\n- the thing works\n';
const SPEC_G1_PASS_G2_FAIL =
  '### Requirement: thing\n#### Scenario: happy path\nG1:\n  CHECK: echo 5\n  EXPECT: 5\nG2:\n  CHECK: echo 5\n  EXPECT: 6\n';

const BACKLOG_BAD = '.backlog/tasks/TASK-001-stub.md';
const BACKLOG_GOOD = '.backlog/tasks/TASK-002-real.md';
const BAD_TASK = '# TASK-001 stub\n';
const GOOD_TASK =
  '---\ntype: feature\nreferences:\n  - hooks/ticketing.js\n---\n\n# TASK-002 example\n\n' +
  '## Definition of Done\n<!-- DOD:BEGIN -->\n- [ ] the guard has a test\n<!-- DOD:END -->\n';

const RED_TEST = 'process.stderr.write("red suite: deliberate failure");\nprocess.exit(1);\n';
const GREEN_TEST = 'console.log("green suite: ok");\n';

// --- case framework --------------------------------------------------------------

const results = [];
let pass = 0;
let fail = 0;

function check(name, fn) {
  if (FILTER && !name.toLowerCase().includes(FILTER.toLowerCase())) return;
  try {
    fn();
    pass += 1;
    results.push(`PASS  ${name}`);
  } catch (e) {
    fail += 1;
    results.push(`FAIL  ${name} — ${e.message}`);
  }
}

function assert(cond, detail) {
  if (!cond) throw new Error(detail);
}
function assertRc(r, want, ctx) {
  assert(r.code === want, `${ctx}: want rc=${want}, got rc=${r.code} (stderr: ${r.err.split('\n')[0]})`);
}
function assertErr(r, re, ctx) {
  assert(re.test(r.err), `${ctx}: stderr missing ${re} (stderr: ${r.err.split('\n').slice(0, 3).join(' | ')})`);
}
function assertOut(r, re, ctx) {
  assert(re.test(r.out), `${ctx}: stdout missing ${re} (stdout: ${r.out.split('\n').slice(0, 3).join(' | ')})`);
}

// --- guard-msg (checkpoint-state.js, driven directly) -----------------------------

check('guard-msg: conventional subject passes', () => {
  const f = join(TMP, 'gm-good.msg');
  writeFileSync(f, MSG_GOOD);
  const r = hook('checkpoint-state.js', ['guard-msg', f]);
  assertRc(r, 0, 'guard-msg good');
});

check('guard-msg: non-conventional subject fails', () => {
  const f = join(TMP, 'gm-bad.msg');
  writeFileSync(f, MSG_BAD_SUBJECT);
  const r = hook('checkpoint-state.js', ['guard-msg', f]);
  assertRc(r, 1, 'guard-msg bad subject');
  assertErr(r, /FATAL: commit subject/, 'guard-msg bad subject');
});

check('guard-msg: Co-Authored-By trailer fails', () => {
  const f = join(TMP, 'gm-coauth.msg');
  writeFileSync(f, MSG_HUMAN_COAUTHOR);
  const r = hook('checkpoint-state.js', ['guard-msg', f]);
  assertRc(r, 1, 'guard-msg coauthor');
  assertErr(r, /Co-Authored-By/, 'guard-msg coauthor');
});

check('guard-msg: malformed tracker footer fails', () => {
  const f = join(TMP, 'gm-badfooter.msg');
  writeFileSync(f, MSG_BAD_FOOTER);
  const r = hook('checkpoint-state.js', ['guard-msg', f]);
  assertRc(r, 1, 'guard-msg bad footer');
  assertErr(r, /tracker footer/, 'guard-msg bad footer');
});

check('guard-msg: valid Refs footer passes', () => {
  const f = join(TMP, 'gm-goodfooter.msg');
  writeFileSync(f, MSG_GOOD_FOOTER);
  const r = hook('checkpoint-state.js', ['guard-msg', f]);
  assertRc(r, 0, 'guard-msg good footer');
});

// --- commit-msg hook (AI-trailer strip step, then guard-msg) -----------------------

check('commit-msg: good subject passes end-to-end', () => {
  const f = join(TMP, 'cm-good.msg');
  writeFileSync(f, MSG_GOOD);
  const r = run('sh', [join(HOOKS, 'commit-msg'), f]);
  assertRc(r, 0, 'commit-msg good');
});

check('commit-msg: bad subject fails end-to-end', () => {
  const f = join(TMP, 'cm-bad.msg');
  writeFileSync(f, MSG_BAD_SUBJECT);
  const r = run('sh', [join(HOOKS, 'commit-msg'), f]);
  assertRc(r, 1, 'commit-msg bad subject');
});

check('commit-msg: AI Co-Authored-By (Cursor) stripped, then guard passes', () => {
  const f = join(TMP, 'cm-cursor.msg');
  writeFileSync(f, MSG_CURSOR);
  const r = run('sh', [join(HOOKS, 'commit-msg'), f]);
  assertRc(r, 0, 'commit-msg cursor strip');
  assert(!/Cursor/.test(readFileSync(f, 'utf8')), 'commit-msg cursor: trailer still in file after strip');
});

check('commit-msg: human Co-Authored-By survives strip, guard blocks', () => {
  const f = join(TMP, 'cm-human.msg');
  writeFileSync(f, MSG_HUMAN_COAUTHOR);
  const r = run('sh', [join(HOOKS, 'commit-msg'), f]);
  assertRc(r, 1, 'commit-msg human coauthor');
  assert(/jane@example\.com/.test(readFileSync(f, 'utf8')), 'commit-msg human: trailer should survive the strip');
});

// --- precommit-zone-guard (BDD spec-zone XOR code-zone) ---------------------------

check('zone: spec-only commit allowed', () => {
  const repo = newRepo('zone-spec');
  stage(repo, file(repo, SPEC_PATH, SPEC_GOOD));
  const r = hook('precommit-zone-guard.js', [], { cwd: repo });
  assertRc(r, 0, 'zone spec-only');
});

check('zone: code-only commit allowed', () => {
  const repo = newRepo('zone-code');
  stage(repo, file(repo, 'src/a.js', 'x=1;\n'));
  const r = hook('precommit-zone-guard.js', [], { cwd: repo });
  assertRc(r, 0, 'zone code-only');
});

check('zone: mixed spec+code commit blocked', () => {
  const repo = newRepo('zone-mixed');
  stage(repo, file(repo, SPEC_PATH, SPEC_GOOD), file(repo, 'src/a.js', 'x=1;\n'));
  const r = hook('precommit-zone-guard.js', [], { cwd: repo });
  assertRc(r, 1, 'zone mixed');
  assertErr(r, /FATAL: mixed spec-zone/, 'zone mixed');
});

check('zone: nothing staged allowed', () => {
  const repo = newRepo('zone-empty');
  const r = hook('precommit-zone-guard.js', [], { cwd: repo });
  assertRc(r, 0, 'zone empty');
});

// --- precommit-guard (protected-ref / detached HEAD) ------------------------------

check('protected: commit on main blocked', () => {
  const repo = newRepo('prot-main');
  initRepo(repo);
  stage(repo, file(repo, 'a.txt', 'a\n'));
  const r = hook('precommit-guard.js', ['guard'], { cwd: repo });
  assertRc(r, 1, 'protected main');
  assertErr(r, /protected branch 'main'/, 'protected main');
});

check('protected: commit on feature branch allowed', () => {
  const repo = newRepo('prot-feat');
  initRepo(repo);
  onBranch(repo, 'feat/x');
  stage(repo, file(repo, 'a.txt', 'a\n'));
  const r = hook('precommit-guard.js', ['guard'], { cwd: repo });
  assertRc(r, 0, 'protected feature');
});

check('protected: detached HEAD at a real commit is blocked', () => {
  const repo = newRepo('prot-detached');
  initRepo(repo);
  git(repo, 'commit', '-q', '--allow-empty', '-m', 'chore: second');
  git(repo, 'checkout', '-q', '--detach', 'HEAD~1');
  stage(repo, file(repo, 'b.txt', 'b\n'));
  const r = hook('precommit-guard.js', ['guard'], { cwd: repo });
  // The guard consults `symbolic-ref --quiet HEAD` first: it fails only when
  // HEAD is detached, so the detached case now hits the FATAL branch instead
  // of the unborn-branch allowance (which used to swallow it — latent bug
  // pinned 2026-11-24, matching v1's bash guard which blocked via symbolic-ref).
  assertRc(r, 1, 'protected detached');
  assertErr(r, /detached/, 'protected detached');
});

check('protected: env override SKILLGRID_PROTECTED_BRANCHES lifts main', () => {
  const repo = newRepo('prot-env');
  initRepo(repo);
  stage(repo, file(repo, 'a.txt', 'a\n'));
  const r = hook('precommit-guard.js', ['guard'], { cwd: repo, env: { SKILLGRID_PROTECTED_BRANCHES: 'trunk' } });
  assertRc(r, 0, 'protected env override');
});

// --- checkpoint-state.js `guard` dispatch ------------------------------------------

check('guard dispatch: mixed spec+code fails via zone guard', () => {
  const repo = newRepo('disp-mixed');
  initRepo(repo);
  onBranch(repo, 'feat/x');
  stage(repo, file(repo, SPEC_PATH, SPEC_GOOD), file(repo, 'src/a.js', 'x=1;\n'));
  const r = hook('checkpoint-state.js', ['guard'], { cwd: repo });
  assertRc(r, 1, 'dispatch mixed');
  assertErr(r, /mixed spec-zone/, 'dispatch mixed');
});

check('guard dispatch: clean feature-branch commit passes all guards', () => {
  const repo = newRepo('disp-clean');
  initRepo(repo);
  onBranch(repo, 'feat/x');
  stage(repo, file(repo, 'src/a.js', 'x=1;\n'));
  const r = hook('checkpoint-state.js', ['guard'], { cwd: repo });
  assertRc(r, 0, 'dispatch clean');
});

check('guard dispatch: staged malformed backlog task fails via ticketing-guard', () => {
  const repo = newRepo('disp-backlog');
  initRepo(repo);
  onBranch(repo, 'feat/x');
  stage(repo, file(repo, BACKLOG_BAD, BAD_TASK));
  const r = hook('checkpoint-state.js', ['guard'], { cwd: repo });
  assertRc(r, 1, 'dispatch backlog');
  assertErr(r, /completeness gate/, 'dispatch backlog');
});

check('guard dispatch: malformed gate is advisory (gate-lint exit 0)', () => {
  const repo = newRepo('disp-lint');
  initRepo(repo);
  onBranch(repo, 'feat/x');
  stage(repo, file(repo, SPEC_PATH, SPEC_MISSING_EXPECT));
  const r = hook('checkpoint-state.js', ['guard'], { cwd: repo });
  assertRc(r, 0, 'dispatch lint advisory');
  assertOut(r, /missing EXPECT/, 'dispatch lint advisory');
});

// --- precommit-ignore-guard (.extracted/ + worktree auto-gitignore) -----------------

check('ignore: staged acceptance-tests/.extracted/ blocked', () => {
  const repo = newRepo('ign-extracted');
  const rel = file(repo, 'acceptance-tests/.extracted/t.feature', 'gen\n');
  stage(repo, rel);
  const r = hook('precommit-ignore-guard.js', [], { cwd: repo });
  assertRc(r, 1, 'ignore extracted');
  assertErr(r, /FATAL: generated\/scratch/, 'ignore extracted');
});

check('ignore: auto-added .extracted/ to .gitignore and unstaged', () => {
  const repo = newRepo('ign-extracted2');
  const rel = file(repo, 'acceptance-tests/.extracted/t.feature', 'gen\n');
  stage(repo, rel);
  const r = hook('precommit-ignore-guard.js', [], { cwd: repo });
  assertRc(r, 1, 'ignore extracted auto-git');
  const gi = readFileSync(join(repo, '.gitignore'), 'utf8');
  assert(gi.split('\n').includes('acceptance-tests/.extracted/'), 'ignore extracted: .gitignore missing pattern');
  const stagedNow = git(repo, 'diff', '--cached', '--name-only').out;
  assert(!stagedNow.includes(rel), 'ignore extracted: file still staged after guard');
});

check('ignore: staged worktree dir blocked + auto-ignored', () => {
  const repo = newRepo('ign-worktrees');
  const rel = file(repo, '.worktrees/wt1/inner.txt', 'w\n');
  stage(repo, rel);
  const r = hook('precommit-ignore-guard.js', [], { cwd: repo });
  assertRc(r, 1, 'ignore worktrees');
  const gi = readFileSync(join(repo, '.gitignore'), 'utf8');
  assert(gi.split('\n').includes('.worktrees/'), 'ignore worktrees: .gitignore missing pattern');
});

// --- gate-lint (advisory: warnings only, always exit 0) -----------------------------

check('gate-lint: gate missing EXPECT warns, exits 0', () => {
  const repo = newRepo('lint-bad');
  file(repo, SPEC_PATH, SPEC_MISSING_EXPECT);
  const r = hook('gate-lint.js', [], { cwd: repo });
  assertRc(r, 0, 'lint missing expect');
  assertOut(r, /missing EXPECT/, 'lint missing expect');
});

check('gate-lint: well-formed gate exits 0 silent', () => {
  const repo = newRepo('lint-good');
  file(repo, SPEC_PATH, SPEC_GOOD);
  const r = hook('gate-lint.js', [], { cwd: repo });
  assertRc(r, 0, 'lint good');
  assert(!r.out.trim(), `lint good: expected no warnings, got: ${r.out.trim()}`);
});

check('gate-lint: happy-path requirement without gate warns (traceability)', () => {
  const repo = newRepo('lint-nogate');
  file(repo, SPEC_PATH, SPEC_NO_GATE);
  const r = hook('gate-lint.js', [], { cwd: repo });
  assertRc(r, 0, 'lint nogate');
  assertOut(r, /traceability gap/, 'lint nogate');
});

// --- gate-state (--status non-executing / --reverify executing) ----------------------

check('gate-state: --status reports runnable gate as unmet (not run)', () => {
  const repo = newRepo('gs-status');
  file(repo, SPEC_PATH, SPEC_GOOD);
  const r = ompHook('gate-state.js', ['--status', '--spec', SPEC_PATH], { cwd: repo });
  assertRc(r, 0, 'gate-state status');
  assertOut(r, /G1=unmet \(not run/, 'gate-state status');
});

check('gate-state: --reverify passes met gate, fails unmet gate', () => {
  const repo = newRepo('gs-reverify');
  file(repo, SPEC_PATH, SPEC_G1_PASS_G2_FAIL);
  const r = ompHook('gate-state.js', ['--reverify', '--spec', SPEC_PATH], { cwd: repo });
  assertRc(r, 0, 'gate-state reverify');
  assertOut(r, /^G1=met$/m, 'gate-state reverify G1');
  assertOut(r, /^G2=unmet$/m, 'gate-state reverify G2');
  assertOut(r, /SUMMARY met=1 unmet=1/, 'gate-state reverify summary');
});

// --- stop-tests (Stop-phase test gate) -------------------------------------------------

check('stop-tests: no runner configured allows the stop', () => {
  const repo = newRepo('stop-none');
  const r = ompHook('stop-tests.js', [], { cwd: repo });
  assertRc(r, 0, 'stop none');
  assertErr(r, /no testing\.runner/, 'stop none');
});

check('stop-tests: red suite blocks the stop (exit 2)', () => {
  const repo = newRepo('stop-red');
  file(repo, 'tests/red.mjs', RED_TEST);
  const r = ompHook('stop-tests.js', [], { cwd: repo, env: { SKILLGRID_TEST_CMD: 'node tests/red.mjs' } });
  assertRc(r, 2, 'stop red');
  assertErr(r, /STOP BLOCKED/, 'stop red');
});

check('stop-tests: green suite allows the stop', () => {
  const repo = newRepo('stop-green');
  file(repo, 'tests/green.mjs', GREEN_TEST);
  const r = ompHook('stop-tests.js', [], { cwd: repo, env: { SKILLGRID_TEST_CMD: 'node tests/green.mjs' } });
  assertRc(r, 0, 'stop green');
});

// --- ticketing-guard (staged backlog task completeness, fail-open) --------------------

check('ticketing-guard: malformed staged backlog task FATAL', () => {
  const repo = newRepo('tg-bad');
  stage(repo, file(repo, BACKLOG_BAD, BAD_TASK));
  const r = hook('ticketing-guard.js', [], { cwd: repo });
  assertRc(r, 1, 'ticketing bad task');
  assertErr(r, /completeness gate/, 'ticketing bad task');
});

check('ticketing-guard: valid task passes (missing plan is WARNING-only)', () => {
  const repo = newRepo('tg-good');
  stage(repo, file(repo, BACKLOG_GOOD, GOOD_TASK));
  const r = hook('ticketing-guard.js', [], { cwd: repo });
  assertRc(r, 0, 'ticketing good task');
});

check('ticketing-guard: fail-open when ticketing disabled (env)', () => {
  const repo = newRepo('tg-disabled');
  stage(repo, file(repo, BACKLOG_BAD, BAD_TASK));
  const r = hook('ticketing-guard.js', [], { cwd: repo, env: { SKILLGRID_TICKETING_ENABLED: '0' } });
  assertRc(r, 0, 'ticketing disabled');
});

check('ticketing-guard: fail-open for non-file tracker type (env)', () => {
  const repo = newRepo('tg-gh');
  stage(repo, file(repo, BACKLOG_BAD, BAD_TASK));
  const r = hook('ticketing-guard.js', [], { cwd: repo, env: { SKILLGRID_TICKETING_TYPE: 'gh' } });
  assertRc(r, 0, 'ticketing gh type');
});

// --- prepare-commit-msg (SKILLGRID_TRACKER_REF `Refs:` injector) ------------------------

check('prepare-commit-msg: SKILLGRID_TRACKER_REF appends Refs line', () => {
  const f = join(TMP, 'pcm-new.msg');
  writeFileSync(f, MSG_GOOD);
  const r = run('sh', [join(HOOKS, 'prepare-commit-msg'), f], { env: { SKILLGRID_TRACKER_REF: 'TASK-069' } });
  assertRc(r, 0, 'prepare-commit-msg append');
  assert(/^Refs: TASK-069$/m.test(readFileSync(f, 'utf8')), 'prepare-commit-msg: Refs line not appended');
});

check('prepare-commit-msg: no-op without env', () => {
  const f = join(TMP, 'pcm-noenv.msg');
  writeFileSync(f, MSG_GOOD);
  const r = run('sh', [join(HOOKS, 'prepare-commit-msg'), f]);
  assertRc(r, 0, 'prepare-commit-msg noenv');
  assert(readFileSync(f, 'utf8') === MSG_GOOD, 'prepare-commit-msg: file modified without env');
});

check('prepare-commit-msg: no-op when a footer is already present', () => {
  const f = join(TMP, 'pcm-has.msg');
  writeFileSync(f, MSG_GOOD_FOOTER);
  const r = run('sh', [join(HOOKS, 'prepare-commit-msg'), f], { env: { SKILLGRID_TRACKER_REF: 'TASK-777' } });
  assertRc(r, 0, 'prepare-commit-msg dup');
  const text = readFileSync(f, 'utf8');
  assert((text.match(/^Refs:/gm) || []).length === 1, 'prepare-commit-msg: duplicate Refs line');
  assert(!/TASK-777/.test(text), 'prepare-commit-msg: should not append over an existing footer');
});

// --- pre-push (ticketing footer check on outgoing commits) -------------------------------

const ZERO_SHA = '0000000000000000000000000000000000000000';

function pushCase(name, commitMsg, want, errRe, ctx) {
  check(name, () => {
    const repo = newRepo(`pp-${ctx}`);
    initRepo(repo);
    git(repo, 'commit', '-q', '--allow-empty', '-m', commitMsg);
    const sha = git(repo, 'rev-parse', 'HEAD').out.trim();
    const input = `refs/heads/main ${sha} refs/heads/main ${ZERO_SHA}\n`;
    const r = hook('checkpoint-state.js', ['pre-push'], { cwd: repo, input });
    assertRc(r, want, `pre-push ${ctx}`);
    if (errRe) assertErr(r, errRe, `pre-push ${ctx}`);
  });
}

pushCase('pre-push: malformed Closes footer blocks the push', 'fix(x): done\n\nCloses: no-id\n', 1, /close no parseable/, 'closes-bad');
pushCase('pre-push: valid Closes footer passes', 'fix(x): done\n\nCloses: TASK-069\n', 0, null, 'closes-good');
pushCase('pre-push: malformed Refs footer only warns (exit 0)', 'docs(x): note\n\nRefs: nope\n', 0, /WARNING/, 'refs-bad');

// --- pre-commit orchestrator (full git commit through the installed shims) ------------------

check('orchestrator: git commit blocked on mixed spec+code commit', () => {
  const repo = newRepo('orch-mixed');
  initRepo(repo);
  installGitHooks(repo);
  onBranch(repo, 'feat/x');
  stage(repo, file(repo, SPEC_PATH, SPEC_GOOD), file(repo, 'src/a.js', 'x=1;\n'));
  const r = git(repo, 'commit', '-m', 'feat(x): both zones');
  assert(r.code !== 0, `orchestrator mixed: commit should fail, got rc=${r.code}`);
  assertErr(r, /mixed spec-zone/, 'orchestrator mixed');
});

check('orchestrator: clean commit lands with AI trailer stripped', () => {
  const repo = newRepo('orch-clean');
  initRepo(repo);
  installGitHooks(repo);
  onBranch(repo, 'feat/x');
  stage(repo, file(repo, 'src/a.js', 'x=1;\n'));
  const r = git(repo, 'commit', '-m', 'feat(x): add a', '-m', 'Co-Authored-By: Cursor <cursor@cursor.com>');
  assertRc(r, 0, 'orchestrator clean');
  const log = git(repo, 'log', '-1', '--format=%B').out;
  assert(!/Co-Authored-By/.test(log), `orchestrator clean: AI trailer survived: ${log.trim()}`);
  assert(/feat\(x\): add a/.test(log), 'orchestrator clean: subject missing from commit');
});

// --- report ---------------------------------------------------------------------------

console.log('');
for (const line of results) console.log(line);
console.log('');
console.log('========================================');
console.log(`Results: ${pass} passed, ${fail} failed`);
process.exit(fail === 0 ? 0 : 1);
