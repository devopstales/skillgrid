#!/usr/bin/env node
// sync-ide-assets.mjs — keep IDE mirrors in sync with the canonical .github assets.
//
// The canonical prompts live in .github/prompts/ and the canonical agents in
// .github/agents/. Some harnesses (Cursor, Copilot, ...) consume per-IDE copies
// under .<harness>/; those copies are mirrors, not sources. This script:
//
//   --check   (CI) verify the canonical set is intact and that any IDE mirror
//             present on disk is in sync with it. Exits non-zero on drift.
//             No IDE mirror exists yet -> check the canonical set only (green).
//
//   --sync    (default) regenerate any present IDE mirrors from the canonical
//             assets, and print what would change if a mirror were added.
//
// Usage:
//   node scripts/sync-ide-assets.mjs --check     # CI
//   node scripts/sync-ide-assets.mjs             # local sync

import { execFileSync } from 'node:child_process';
import { cpSync, existsSync, mkdirSync, readFileSync, readdirSync, statSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const PROMPTS_DIR = path.join(ROOT, '.github/prompts');
const AGENTS_DIR = path.join(ROOT, '.github/agents');

const MODE = process.argv[2] ?? '--check';

// IDE mirror dirs: <dir> maps prompts to <dir>/prompts, agents to <dir>/agents.
// Extend this list when a new IDE copy lands in the repo.
const IDE_MIRRORS = ['.cursor', '.copilot'];

let drift = false;

function fail(msg) {
  console.error(`FAIL: ${msg}`);
  process.exit(1);
}

function listMd(dir, exclude = new Set()) {
  if (!existsSync(dir)) return [];
  return readdirSync(dir)
    .filter((f) => f.endsWith('.md') && !exclude.has(f))
    .sort();
}

function promptHasTitle(file) {
  const lines = readFileSync(file, 'utf8').split('\n').slice(0, 5);
  return lines.some((l) => /^description:/.test(l) || /^#/.test(l));
}

function checkCanonical() {
  if (!existsSync(PROMPTS_DIR)) fail('canonical prompts dir missing: .github/prompts');
  if (!existsSync(AGENTS_DIR)) fail('canonical agents dir missing: .github/agents');

  const nPrompts = listMd(PROMPTS_DIR).length;
  const nAgents = listMd(AGENTS_DIR, new Set(['README.md'])).length;

  if (nPrompts < 1) fail(`no prompt files in ${PROMPTS_DIR}`);
  if (nAgents < 1) fail(`no agent files in ${AGENTS_DIR}`);

  // Every prompt must be non-empty and carry a title: either a frontmatter
  // `description:` (the canonical .github/prompts format) or a `#` heading.
  for (const f of listMd(PROMPTS_DIR)) {
    const file = path.join(PROMPTS_DIR, f);
    if (statSync(file).size === 0) fail(`empty prompt: ${file}`);
    if (!promptHasTitle(file)) fail(`prompt missing title: ${file}`);
  }

  console.log(`canonical OK: ${nPrompts} prompts, ${nAgents} agents`);
}

// compare one IDE mirror (if present) against canonical; report drift
function checkMirror(ide) {
  const dir = path.join(ROOT, ide);
  if (!existsSync(dir)) {
    console.log(`mirror ${ide}: not present (nothing to check)`);
    return;
  }

  for (const sub of ['prompts', 'agents']) {
    const src = path.join(ROOT, '.github', sub);
    const dst = path.join(dir, sub);
    if (!existsSync(src)) continue;
    if (existsSync(dst)) {
      let inSync = true;
      try {
        execFileSync('diff', ['-rq', src, dst], { stdio: 'ignore' });
      } catch {
        inSync = false;
      }
      if (inSync) {
        console.log(`mirror ${ide}/${sub}: in sync`);
      } else {
        console.error(`mirror ${ide}/${sub}: DRIFT (run --sync)`);
        drift = true;
      }
    } else {
      console.error(`mirror ${ide}: ${sub}/ missing (run --sync to create)`);
      drift = true;
    }
  }
}

function syncOne(ide) {
  const dir = path.join(ROOT, ide);
  mkdirSync(path.join(dir, 'prompts'), { recursive: true });
  mkdirSync(path.join(dir, 'agents'), { recursive: true });
  // Mirror the canonical .md assets (skip the agents README — it's hub docs).
  for (const f of listMd(PROMPTS_DIR)) {
    cpSync(path.join(PROMPTS_DIR, f), path.join(dir, 'prompts', f), { force: true });
  }
  for (const f of listMd(AGENTS_DIR, new Set(['README.md']))) {
    cpSync(path.join(AGENTS_DIR, f), path.join(dir, 'agents', f), { force: true });
  }
  console.log(`synced ${ide}`);
}

switch (MODE) {
  case '--check':
    checkCanonical();
    for (const ide of IDE_MIRRORS) checkMirror(ide);
    if (drift) fail('IDE mirror drift detected');
    console.log('sync-check OK');
    break;
  case '--sync':
    checkCanonical();
    for (const ide of IDE_MIRRORS) {
      if (!existsSync(path.join(ROOT, ide))) {
        console.log(`${ide}: not present, skipping (add the dir to mirror)`);
        continue;
      }
      syncOne(ide);
    }
    console.log('sync done');
    break;
  default:
    console.error('usage: sync-ide-assets.mjs [--check|--sync]');
    process.exit(2);
}
