#!/usr/bin/env node
// checkpoint-state.js — skillgrid guard-hooks entrypoint.
//
// The durable record lives in the commit (its [skillgrid-context] body block).
// Resume reads the session event stream, not a checkpoint file.
//
// CLI-ready: the repo's .git/hooks shims call this script. skillgrid-cli later
// rebinds the shims to its binary; the subcommand contract stays stable.
//
// Subcommands:
//   guard            pre-commit guards (delegates to precommit-guard.js)
//   guard-msg        validate a commit message file (conventional + no Co-Authored-By + tracker footer format)
//   prepare-msg      prepare-commit-msg injector: append `Refs: <id>` from SKILLGRID_TRACKER_REF (no-op otherwise, never blocks)
//   pre-push         validate Refs/Closes footers on outgoing commits (blocks only malformed Closes)
//   post-merge-check post-merge tracker consistency (advisory, never blocks)
//   post-check       post-commit deletion check (delegates to precommit-guard.js)
//
// (The snapshot/restore subcommands were removed in the session-events-layer
// consolidation: the durable record is the commit's [skillgrid-context] block
// and resume reads the session event stream. Both now report unknown
// subcommand.)
//
// Env: none.
'use strict';

const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const SUBCMD = process.argv[2] || '';
const REST = process.argv.slice(3);

const SCRIPT_DIR = __dirname;
const GUARD = path.join(SCRIPT_DIR, 'precommit-guard.js');

function runJs(file, ...args) {
  const r = spawnSync(process.execPath, [file, ...args], { stdio: 'inherit' });
  return r.status === null ? 1 : r.status;
}

// guard is a dispatcher: run every pre-commit guard in sequence. Each is a
// sibling in hooks/. The shims (and skillgrid-cli later) call only
// `guard` — adding a guard here is the one place to change.
function cmdGuard() {
  let rc = runJs(GUARD, 'guard'); // cwd-drift + protected-ref/detached
  if (rc !== 0) return process.exit(rc);
  rc = runJs(path.join(SCRIPT_DIR, 'precommit-zone-guard.js'));
  if (rc !== 0) return process.exit(rc);
  rc = runJs(path.join(SCRIPT_DIR, 'ticketing-guard.js')); // staged tracker files carry type/references/DoD
  if (rc !== 0) return process.exit(rc);
  rc = runJs(path.join(SCRIPT_DIR, 'precommit-ignore-guard.js'));
  if (rc !== 0) return process.exit(rc);
  rc = runJs(path.join(SCRIPT_DIR, 'precommit-go-vet.js')); // vet staged Go packages
  if (rc !== 0) return process.exit(rc);
  rc = runJs(path.join(SCRIPT_DIR, 'gate-lint.js')); // advisory: warn on malformed gate blocks
  process.exit(rc);
}

function cmdPostCheck() {
  const rc = runJs(GUARD, 'post-check');
  process.exit(rc === null ? 1 : rc);
}

function cmdGuardMsg() {
  const msgFile = REST[0] || '';
  if (!msgFile || !fs.existsSync(msgFile)) {
    process.stderr.write('guard-msg: missing message file\n');
    process.exit(2);
  }

  // Read only the first non-comment, non-empty line (the subject).
  const subject = fs
    .readFileSync(msgFile, 'utf8')
    .split('\n')
    .map((l) => l.replace(/\s+$/, ''))
    .find((l) => l.trim() !== '' && !/^\s*#/.test(l));

  // Conventional-commit subject regex (scope optional, ! optional).
  const re = /^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([a-z0-9._-]+\))?!?: .+/;
  if (!re.test(subject)) {
    process.stderr.write('FATAL: commit subject does not follow conventional-commits.\n');
    process.stderr.write(`  Got:      ${subject}\n`);
    process.stderr.write('  Expected: type(scope)?: description  (type in build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)\n');
    process.stderr.write("RECOVERY: rewrite the subject, e.g. 'feat(auth): add session refresh'.\n");
    process.exit(1);
  }

  const text = fs.readFileSync(msgFile, 'utf8');
  // No Co-Authored-By / Generated-By trailers anywhere in the message, in any
  // casing (Cursor writes "Co-authored-by:", git's canonical form is
  // "Co-Authored-By:").
  if (/^(Co-Authored-By|Generated-By):/im.test(text)) {
    process.stderr.write('FATAL: commit message carries a Co-Authored-By/Generated-By trailer.\n');
    process.stderr.write('  Skillgrid convention: no AI attribution in commits (see branch-pr).\n');
    process.stderr.write('RECOVERY: remove the trailer line and re-stage the commit message.\n');
    process.exit(1);
  }

  // Tracker footers (Refs:/Closes: from the ticketing lifecycle) are optional
  // but must reference a parseable Tracker ID when present. Absence passes.
  const ticketing = require('./ticketing');
  const bad = ticketing.invalidFooters(text);
  if (bad.length > 0) {
    process.stderr.write('FATAL: commit message has tracker footer(s) with no parseable Tracker ID.\n');
    for (const f of bad) process.stderr.write(`    - ${f.keyword}: ${f.value}\n`);
    process.stderr.write('  Expected an ID like TASK-069 (backlog), #42 (gh/glab), or PROJ-123 (jira).\n');
    process.stderr.write('RECOVERY: fix the footer, or drop it when the commit needs no tracker link.\n');
    process.exit(1);
  }
  process.exit(0);
}

// prepare-msg appends `Refs: <id>` from SKILLGRID_TRACKER_REF (exported by the
// agent from the active tasks.md Tracker ID) when the message has no tracker
// footer yet. No env, no-op. Never blocks: always exits 0.
function cmdPrepareMsg() {
  const msgFile = REST[0] || '';
  const ref = (process.env.SKILLGRID_TRACKER_REF || '').trim();
  if (!msgFile || !ref || !fs.existsSync(msgFile)) process.exit(0);
  const ticketing = require('./ticketing');
  const text = fs.readFileSync(msgFile, 'utf8');
  if (ticketing.parseFooters(text).length > 0) process.exit(0);
  const sep = text.endsWith('\n') ? '' : '\n';
  fs.writeFileSync(msgFile, `${text}${sep}\nRefs: ${ref}\n`);
  process.exit(0);
}

function cmdPrePush() {
  const rc = runJs(path.join(SCRIPT_DIR, 'ticketing-pre-push.js'), ...REST);
  process.exit(rc === null ? 0 : rc);
}

// post-merge-check is advisory by construction (ticketing-post-merge.js always
// exits 0); the `|| true` belt holds if node itself fails to spawn.
function cmdPostMergeCheck() {
  runJs(path.join(SCRIPT_DIR, 'ticketing-post-merge.js'), ...REST);
  process.exit(0);
}

switch (SUBCMD) {
  case 'guard':
    cmdGuard();
    break;
  case 'guard-msg':
    cmdGuardMsg();
    break;
  case 'prepare-msg':
    cmdPrepareMsg();
    break;
  case 'pre-push':
    cmdPrePush();
    break;
  case 'post-merge-check':
    cmdPostMergeCheck();
    break;
  case 'post-check':
    cmdPostCheck();
    break;
  default:
    process.stderr.write(`unknown subcommand: ${SUBCMD} (expected guard|guard-msg|prepare-msg|pre-push|post-merge-check|post-check)\n`);
    process.exit(2);
}
