#!/usr/bin/env node
// precommit-go-vet.js — vet the Go packages touched by the staged commit.
//
// A commit is only as sound as the vet of the code it ships. This guard runs
// `go vet` against the package dirs containing every staged .go file, so a
// type error or a vet warning introduced by THIS commit is caught before the
// commit lands, without vetting the whole module on a docs-only change.
//
// It finds each staged file's module root (the nearest ancestor with a go.mod),
// vetting per-module so monorepos with several go.mod files are handled, and
// resolves staged paths relative to that root (git paths are repo-root-relative).
//
// Additive + self-contained: it is one guard in the checkpoint-state.js `guard`
// dispatcher and only runs when the commit actually stages Go code and a `go`
// toolchain is available. A missing `go` binary or a commit with no .go files
// is a pass, never a failure — this guard must not wedge a non-Go or
// no-toolchain environment.
//
// CLI-ready: called by checkpoint-state.js guard (the pre-commit dispatcher).
'use strict';

const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

function git(...args) {
  return spawnSync('git', args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
}

const staged = git('diff', '--cached', '--name-only', '--diff-filter=ACMR').stdout
  .split('\n')
  .map((l) => l.trim())
  .filter(Boolean);

const goFiles = staged.filter((f) => f.endsWith('.go'));
if (goFiles.length === 0) process.exit(0); // nothing Go: let the other guards decide

const goBin = spawnSync('go', ['version'], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
if (goBin.error || goBin.status !== 0) {
  process.stderr.write('  [go-vet] no `go` toolchain on PATH — skipping go vet (pass).\n');
  process.exit(0);
}

const cwd = process.cwd();

// moduleRoot walks up from a repo-relative path to the nearest ancestor (as a
// cwd-absolute dir) that contains a go.mod, or null.
function moduleRoot(rel) {
  let abs = path.resolve(cwd, rel);
  for (;;) {
    if (fs.existsSync(path.join(abs, 'go.mod'))) return abs;
    const parent = path.dirname(abs);
    if (parent === abs) return null;
    abs = parent;
  }
}

// Group the staged package dirs by module root, then express each as a
// `./<relDir>/...` pattern relative to that root (or `./...` for the root).
const byModule = new Map(); // moduleRoot -> Set<cwd-absolute dir>
for (const f of goFiles) {
  const root = moduleRoot(f);
  if (!root) continue; // not under any go.mod: skip this file
  const dir = path.resolve(cwd, path.posix.dirname(f));
  if (!byModule.has(root)) byModule.set(root, new Set());
  byModule.get(root).add(dir);
}
if (byModule.size === 0) process.exit(0); // no staged Go file is under a go.mod

for (const [root, dirs] of byModule) {
  const patterns = [...dirs]
    .map((d) => path.relative(root, d))
    .filter((p) => p.length > 0 && p !== path.sep)
    .map((p) => `./${p.split(path.sep).join('/')}/...`);
  if (patterns.length === 0) patterns.push('./...');
  const args = ['vet', ...patterns];
  const r = spawnSync('go', args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'], cwd: root });
  if (r.status === 0) continue;

  const out = `${(r.stderr || '').trim()}\n${(r.stdout || '').trim()}`.replace(/\n{2,}/g, '\n').trim();
  const lines = [
    'FATAL: go vet failed on the Go packages staged in this commit.',
    'RECOVERY: fix the reported issues, re-stage, and commit again.',
    `  ran: cd ${root} && go vet ${patterns.join(' ')}`,
  ];
  if (out) lines.push('', out);
  process.stderr.write(`${lines.join('\n')}\n`);
  process.exit(1);
}
process.exit(0);
