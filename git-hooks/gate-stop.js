#!/usr/bin/env node
// gate-stop.js — agent harness hook (Stop event). Blocks the agent from ending its
// turn while a G<n> in the change's acceptance.feature is unmet or a happy-path
// requirement is missing its gate (fresh --reverify evidence). Allows when all
// are met, or only abandoned (emits a HANDOFF note). Loop-guard releases after
// 6 no-progress blocks. AGENT hook, not a git hook — installed to
// ~/.skillgrid/git-hooks/ by `skillgrid install`. CLI-ready: skillgrid-cli
// rebinds it later.
'use strict';

const { spawnSync } = require('node:child_process');
const path = require('node:path');

// Do NOT drain stdin: gate-stop.js reads the harness JSON to extract session_id
// (for the loop-guard key) and is a no-op if stdin is absent.
const SCRIPT_DIR = __dirname;
const r = spawnSync(process.execPath, [path.join(SCRIPT_DIR, '..', 'hooks', 'gate-stop.js')], {
  stdio: 'inherit',
});
process.exit(r.status === null ? 1 : r.status);
