#!/usr/bin/env node
// pre-commit.js — thin shim over checkpoint-state.js guard.
// Lives in git-hooks/ (the hook entrypoints); the implementation lives in
// hooks/. Installed (copied) to ~/.skillgrid/git-hooks/ by `skillgrid install`,
// next to a copy of hooks/, so the relative path below resolves in both places.
// CLI-ready: skillgrid-cli rebinds this shim to its binary.
'use strict';

const { spawnSync } = require('node:child_process');
const path = require('node:path');

const SCRIPT_DIR = __dirname;
const r = spawnSync(process.execPath, [path.join(SCRIPT_DIR, '..', 'hooks', 'checkpoint-state.js'), 'guard'], {
  stdio: 'inherit',
});
process.exit(r.status === null ? 1 : r.status);
