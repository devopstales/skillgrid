#!/usr/bin/env node
// stop.js — agent harness hook (Stop event). Runs the project test gate before the
// agent may end its turn. The harness feeds session JSON on stdin; we consume
// and discard it. This is an AGENT hook, not a git hook — installed to
// ~/.skillgrid/git-hooks/ by `skillgrid install`. CLI-ready: skillgrid-cli
// rebinds it later.
'use strict';

const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

// drain stdin so the harness doesn't see a broken pipe
try {
  fs.readFileSync(0);
} catch {}

const SCRIPT_DIR = __dirname;
const r = spawnSync(process.execPath, [path.join(SCRIPT_DIR, '..', 'hooks', 'stop-tests.js')], {
  stdio: 'inherit',
});
process.exit(r.status === null ? 1 : r.status);
