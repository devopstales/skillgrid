#!/usr/bin/env node
// rule-metrics.mjs — deterministic line / rule / byte counts for distillation tiers.
//
// Usage:
//   node rule-metrics.mjs <full.md> <mini.md> <nano.md>
//
// A "rule" is a top-level markdown list item (`- ` / `* ` / `1. `) that is not a
// task checkbox and does not sit under a "Final checklist" heading. This is the
// reproducible counting convention for the release table — it does NOT judge
// whether a rule is good, only how many there are.
//
// Exit codes: 0 = all three files read and measured, 2 = usage or read error.

import { readFileSync } from 'node:fs';

const args = process.argv.slice(2);
if (args.length !== 3) {
  console.error('usage: node rule-metrics.mjs <full.md> <mini.md> <nano.md>');
  process.exit(2);
}
const [full, mini, nano] = args;

const CHECKLIST = /^final checklist$/i;

function measure(path) {
  const raw = readFileSync(path, 'utf8');
  const lines = raw.split('\n');
  let bytes = Buffer.byteLength(raw, 'utf8');
  let rules = 0;
  let inChecklist = false;
  for (const line of lines) {
    const h = line.match(/^#{1,6}\s+(.*)$/);
    if (h) {
      inChecklist = CHECKLIST.test(h[1].trim());
      continue;
    }
    if (inChecklist) continue;
    const item = line.match(/^\s*(?:[-*]|\d+\.)\s+\S/);
    if (item && !/^\s*(?:[-*]|\d+\.)\s+\[[ xX]\]/.test(line)) rules += 1;
  }
  return { lines: lines.length, bytes, rules };
}

let rows;
try {
  rows = [
    ['full', measure(full)],
    ['mini', measure(mini)],
    ['nano', measure(nano)],
  ];
} catch (err) {
  console.error(`ERROR: ${err.message}`);
  process.exit(2);
}

console.log('tier  lines  bytes  rules');
for (const [tier, m] of rows) {
  console.log(
    `${tier.padEnd(6)} ${String(m.lines).padStart(5)}  ${String(m.bytes).padStart(5)}  ${String(m.rules).padStart(4)}`,
  );
}
