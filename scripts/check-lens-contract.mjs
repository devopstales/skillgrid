#!/usr/bin/env node
// check-lens-contract.mjs — machine-checkable completion contract for the
// review lenses in .agents/skills/test/references/reviewers/.
//
// "A rule asks; a hook guarantees." This is the guarantee for the review lens
// output contract. Each lens promises a specific prompt shape (Inputs, an
// Output format, running under the Blindness Contract) and the core-five
// lenses must carry the `could-not-verify` verdict into their prompt. If a
// lens file drifts from what this script expects, the check FAILS — a stale
// contract is a broken guarantee.
//
// Port of v1 scripts/check-lens-contract.sh (the LENS_CONTRACTS table
// discipline: this table is the single source of truth for the contract —
// change a lens's shape and you MUST change the matching row here).
//
// Usage:
//   node scripts/check-lens-contract.mjs          # check every lens
//   node scripts/check-lens-contract.mjs --list   # print the contract table
//   node scripts/check-lens-contract.mjs --json   # machine-readable status
//
// Exit code: 0 if every lens matches its contract, 1 if any drift (or a
// lens file is missing). Plain node, no dependencies.
'use strict';

import { existsSync, readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const HERE = dirname(fileURLToPath(import.meta.url));
const LENSES = join(HERE, '..', '.agents/skills', 'test', 'references', 'reviewers');

// --- the contract -----------------------------------------------------------------
//
// Markers (grepped literally against the lens file):
//   inputs     — an `Inputs` section: the lens names what it reads (the diff
//                range, the sources it is given) and that it reads them itself.
//   output     — an `Output` section: the declared output format the
//                coordinator's triage step consumes.
//   blindness  — the lens names the Blindness Contract it runs under (fresh
//                context, no leaks, independence grade).
//   could-not-verify — core-five lenses only: the prompt must carry the
//                `could-not-verify` verdict (never a confident pass when the
//                check is not inferable from the diff alone).
//   persona    — security names its persona path.
//   rest       — spec references api-design.md (REST surface review).

const COMMON = [
  { name: 'Inputs section', re: /\*\*Inputs:?\*\*|^#{1,6}[ \t]*Inputs\b/m },
  { name: 'Output section', re: /\*\*Output:?\*\*|^#{1,6}[ \t]*Output\b/m },
  { name: 'Blindness Contract mention', re: /Blindness Contract/ },
];
const CNV = { name: 'could-not-verify mention', re: /could-not-verify/ };

// LENS_CONTRACTS: one row per lens. Core five = Standards, Spec, Security,
// Edge cases, Verification gaps (per test/references/review-and-fix-loop.md
// topology step); accessibility/performance are conditional specialists,
// red team runs last.
const LENS_CONTRACTS = [
  { file: 'standards.md', markers: [...COMMON, CNV] },
  {
    file: 'spec.md',
    markers: [
      ...COMMON,
      CNV,
      { name: 'api-design.md REST reference', re: /api-design\.md/ },
    ],
  },
  {
    file: 'security.md',
    markers: [
      ...COMMON,
      CNV,
      { name: 'security-sentinel persona path', re: /\.agents\/agents\/security-sentinel\.md/ },
    ],
  },
  { file: 'edge-cases.md', markers: [...COMMON, CNV] },
  { file: 'verification-gap.md', markers: [...COMMON, CNV] },
  { file: 'accessibility.md', markers: [...COMMON] },
  { file: 'performance.md', markers: [...COMMON] },
  { file: 'red-team.md', markers: [...COMMON] },
];

// --- check ---------------------------------------------------------------------------

const mode = process.argv[2] || '';
if (mode === '--list') {
  for (const row of LENS_CONTRACTS) {
    console.log(`${row.file} | ${row.markers.map((m) => m.name).join(', ')}`);
  }
  process.exit(0);
}

const results = [];
let fail = 0;

for (const row of LENS_CONTRACTS) {
  const path = join(LENSES, row.file);
  const drift = [];
  if (!existsSync(path)) {
    drift.push('missing lens file');
  } else {
    const text = readFileSync(path, 'utf8');
    for (const m of row.markers) {
      if (!m.re.test(text)) drift.push(`missing marker: ${m.name}`);
    }
  }
  results.push({ file: row.file, expected: row.markers.map((m) => m.name), drift, ok: drift.length === 0 });
  if (drift.length > 0) {
    fail += 1;
    for (const d of drift) console.log(`DRIFT  ${row.file}: ${d}`);
  }
}

if (fail === 0) {
  console.log(`OK     all ${results.length} lens contracts match`);
}

if (mode === '--json') {
  console.log(JSON.stringify(results, null, 2));
} else if (fail > 0) {
  console.log('');
  console.log(`${fail} lens file(s) drifted from LENS_CONTRACTS — fix the lens or update the table.`);
}
process.exit(fail === 0 ? 0 : 1);
