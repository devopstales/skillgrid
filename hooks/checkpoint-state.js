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
//   guard        pre-commit guards (delegates to precommit-guard.js)
//   guard-msg    validate a commit message file (conventional + no Co-Authored-By)
//   post-check   post-commit deletion check (delegates to precommit-guard.js)
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
  process.exit(0);
}

switch (SUBCMD) {
  case 'guard':
    cmdGuard();
    break;
  case 'guard-msg':
    cmdGuardMsg();
    break;
  case 'post-check':
    cmdPostCheck();
    break;
  default:
    process.stderr.write(`unknown subcommand: ${SUBCMD} (expected guard|guard-msg|post-check)\n`);
    process.exit(2);
}
