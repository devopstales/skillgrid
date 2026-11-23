#!/usr/bin/env node
// precommit-guard.js — commit-time safety assertions for skillgrid work units.
//
// Called by checkpoint-state.js (subcommands `guard` and `post-check`) from the
// repo's .git/hooks/pre-commit shim. Exits non-zero (with a RECOVERY line) on
// any guard failure so the hook blocks the commit.
//
// CLI-ready: the hooks are one-line shims over checkpoint-state.js; skillgrid-cli
// rebinds the shims to its binary later. These assertions are portable and have
// no dependency on a specific harness.
//
// Usage:
//   precommit-guard.js guard        # pre-commit: cwd-drift + protected-ref/detached
//   precommit-guard.js post-check   # post-commit (run manually after committing):
//                                   #   warn on unexpected file deletions in HEAD
//
// Env:
//   SKILLGRID_AGENT_BRANCH_REGEX  positive allow-list for per-agent branches
//                                 (default: ^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$)
//   SKILLGRID_PROTECTED_BRANCHES  deny-list (default: main master develop trunk release/)
'use strict';

const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

function git(...args) {
  return spawnSync('git', args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
}

const SUBCMD = process.argv[2] || 'guard';

const isWorktree = () => fs.existsSync('.git') && fs.statSync('.git').isFile();

// --- cwd-drift assertion (worktree mode) -------------------------------------
// A prior Bash call may have `cd`'d out of the worktree into the main repo. When
// that happens `.git` is a directory (main repo) not a file, so `is_worktree`
// would silently skip every worktree guard. Capture the spawn-time toplevel via
// a sentinel on the first check, then verify on every subsequent check.
function checkCwdDrift() {
  const wtGitDir = git('rev-parse', '--git-dir').stdout.trim();
  if (!wtGitDir) return;
  const m = wtGitDir.match(/^(.*)\.git\/worktrees\/(.*)$/);
  if (!m) return;
  const gitBase = `${m[1].replace(/\/+$/, '')}.git/worktrees/${m[2]}`;
  const sentinel = path.join(gitBase, 'skillgrid-spawn-toplevel');
  if (!fs.existsSync(sentinel)) {
    const r = git('rev-parse', '--show-toplevel');
    if (r.status === 0) {
      fs.writeFileSync(sentinel, r.stdout);
    }
  }
  const expectedTl = fs.existsSync(sentinel) ? fs.readFileSync(sentinel, 'utf8').trim() : '';
  const actualTl = git('rev-parse', '--show-toplevel').stdout.trim();
  if (expectedTl && actualTl !== expectedTl) {
    process.stderr.write('FATAL: cwd drifted from spawn-time worktree root.\n');
    process.stderr.write(`  Spawn-time: ${expectedTl}\n`);
    process.stderr.write(`  Current:    ${actualTl}\n`);
    process.stderr.write(`RECOVERY: cd "${expectedTl}" before staging, then re-run the commit.\n`);
    process.exit(1);
  }
}

// --- protected-ref / detached-HEAD assertion (all repos) -----------------------
// Never commit on a protected ref or a detached HEAD. Applies to every repo
// (matches the "never commit on main without consent" rule); only the per-agent
// allow-list at the end is worktree-specific. We do NOT self-recover via
// `git update-ref` — surface it as a blocker instead.
function checkProtectedRef() {
  const sym = git('symbolic-ref', '--quiet', 'HEAD');
  const headRef = sym.status === 0 ? sym.stdout.trim() : 'DETACHED';
  // On an unborn branch (initial commit) `--abbrev-ref HEAD` errors; fall back
  // to the branch name from symbolic-ref, then to "HEAD".
  let actualBranch = git('rev-parse', '--abbrev-ref', 'HEAD').stdout.trim();
  if (!actualBranch) {
    actualBranch = headRef.replace(/^refs\/heads\//, '');
    if (actualBranch === 'HEAD') actualBranch = 'HEAD';
  }

  // First commit of the repo: no HEAD yet. Allow it — the protected-ref rule
  // bites once history exists (you should be on a per-agent branch by then).
  if (actualBranch === 'HEAD') return;

  if (headRef === 'DETACHED') {
    process.stderr.write('FATAL: refusing to commit — worktree HEAD is detached (expected a per-agent branch).\n');
    process.stderr.write('RECOVERY: create/checkout a per-agent branch, e.g. git checkout -b agent-<id>.\n');
    process.exit(1);
  }

  const protectedList = (process.env.SKILLGRID_PROTECTED_BRANCHES || 'main master develop trunk release/').split(/\s+/).filter(Boolean);
  for (const p of protectedList) {
    if (actualBranch === p) {
      process.stderr.write(`FATAL: refusing to commit — HEAD is on protected branch '${actualBranch}'.\n`);
      process.stderr.write(`RECOVERY: commit on a per-agent branch, not '${actualBranch}'.\n`);
      process.exit(1);
    }
  }

  // Positive allow-list: in a worktree the branch must be a per-agent branch.
  // Only enforced in worktrees (a normal repo commits on its feature branch).
  if (isWorktree()) {
    const allow = process.env.SKILLGRID_AGENT_BRANCH_REGEX || '^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$';
    if (!new RegExp(allow).test(actualBranch)) {
      process.stderr.write(`FATAL: refusing to commit — worktree HEAD '${actualBranch}' is not a per-agent branch.\n`);
      process.stderr.write('  Allowed: agent-*, worktree-agent-*, worktree-wf_* (override: SKILLGRID_AGENT_BRANCH_REGEX).\n');
      process.stderr.write('RECOVERY: checkout a per-agent branch, e.g. git checkout -b agent-<id>.\n');
      process.exit(1);
    }
  }
}

// --- post-commit deletion check -----------------------------------------------
// After a commit, verify it did not accidentally delete tracked files. Intentional
// deletions are expected and the caller documents them; here we only WARN (a
// deletion in a work-unit commit is worth a glance, not a hard block).
function postCheckDeletions() {
  const deletions = git('diff', '--diff-filter=D', '--name-only', 'HEAD~1', 'HEAD').stdout.trim();
  if (deletions) {
    const short = git('rev-parse', '--short', 'HEAD').stdout.trim();
    process.stderr.write(`WARNING: commit ${short} includes file deletions:\n`);
    for (const d of deletions.split('\n').filter(Boolean)) {
      process.stderr.write(`  - ${d}\n`);
    }
    process.stderr.write("Intentional? Document them in the commit's [skillgrid-context] block; otherwise revert and fix.\n");
  }
}

switch (SUBCMD) {
  case 'guard':
    // cwd-drift is worktree-only; protected-ref/detached applies to all repos.
    if (isWorktree()) checkCwdDrift();
    checkProtectedRef();
    break;
  case 'post-check':
    postCheckDeletions();
    break;
  default:
    process.stderr.write(`unknown subcommand: ${SUBCMD} (expected guard|post-check)\n`);
    process.exit(2);
}
process.exit(0);
