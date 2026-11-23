#!/usr/bin/env node
// precommit-ignore-guard.js — block (and auto-fix) commits of generated/scratch
// paths that must never be tracked.
//
// Two invariants:
//   - acceptance-test-authoring: acceptance-tests/.extracted/ is "gitignored,
//     wiped and rebuilt on every run, never edited by hand." A committed
//     extraction is a real bug (a stale one keeps deleted capabilities running).
//   - isolated-workspace: the worktree dir (conventions.worktree_dir, default
//     .worktrees/) MUST be gitignored before use — an unignored worktree commits
//     the whole tree into the repo.
//
// On a hit: if the path is NOT yet ignored, auto-add it to .gitignore (the
// "auto-git" behavior) and unstage it from the commit, then still FAIL the
// commit so the agent sees the fix. If it IS ignored but somehow staged, just
// FAIL.
//
// CLI-ready: called by checkpoint-state.js guard (the pre-commit dispatcher).
'use strict';

const { spawnSync } = require('node:child_process');
const fs = require('node:fs');

function git(...args) {
  return spawnSync('git', args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
}

const root = git('rev-parse', '--show-toplevel').stdout.trim();
if (!root) process.exit(0);

// read a key from .skillgrid/config.yaml (first match), fallback if absent.
// The key is a simple trailing key like `worktree_dir:` (no dots needed).
function configValue(key, fallback) {
  const file = `${root}/.skillgrid/config.yaml`;
  if (!fs.existsSync(file)) return fallback;
  for (const line of fs.readFileSync(file, 'utf8').split('\n')) {
    const m = line.match(new RegExp(`^[\\s]*${key}:`));
    if (m) {
      let v = line.replace(/^[^:]*:[\s]*/, '').replace(/[\s]*#.*$/, '').replace(/["']/g, '');
      if (v) return v;
    }
  }
  return fallback;
}

const wtDir = configValue('worktree_dir', '.worktrees');

const staged = git('diff', '--cached', '--name-only', '--diff-filter=ACMR').stdout
  .split('\n')
  .filter(Boolean);
if (staged.length === 0) process.exit(0);

let fixNeeded = false;
for (const f of staged) {
  let pat = null;
  if (f.startsWith('acceptance-tests/.extracted/')) {
    pat = 'acceptance-tests/.extracted/';
  } else if (f.startsWith(`${wtDir}/`) || f.startsWith('worktrees/') || f.startsWith('.worktrees/')) {
    pat = `${wtDir}/`;
  } else {
    continue;
  }

  // Is the directory currently ignored?
  if (git('check-ignore', '-q', f).status === 0) {
    process.stderr.write(`WARNING: '${f}' is gitignored but staged. Unstaging it.\n`);
    let r = git('reset', '-q', '-N', '--', f);
    if (r.status !== 0) r = git('reset', '-q', 'HEAD', '--', f);
    fixNeeded = true;
    continue;
  }

  // Not ignored: auto-add the dir to .gitignore.
  const gi = `${root}/.gitignore`;
  if (!fs.existsSync(gi)) fs.writeFileSync(gi, '');
  const current = fs.readFileSync(gi, 'utf8');
  if (!current.split('\n').includes(pat)) {
    let prefix = '';
    if (current.length > 0 && !current.endsWith('\n')) prefix = '\n';
    fs.appendFileSync(gi, `${prefix}${pat}\n`);
    process.stderr.write(`NOTICE: auto-added '${pat}' to .gitignore.\n`);
  }
  // Unstage the generated path so it isn't committed this time.
  if (git('reset', '-q', 'HEAD', '--', f).status !== 0) {
    git('rm', '--cached', '-q', '--', f);
  }
  fixNeeded = true;
}

if (fixNeeded) {
  process.stderr.write(
    [
      'FATAL: generated/scratch path(s) were staged.',
      '  .extracted/ is rebuilt every run; the worktree dir must stay untracked.',
      '  I added the missing pattern to .gitignore and unstaged the path(s).',
      'RECOVERY: re-stage your real files and commit again. Stage .gitignore too if you want to record the new ignore rule.',
      '',
    ].join('\n')
  );
  process.exit(1);
}
process.exit(0);
