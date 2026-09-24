#!/usr/bin/env node
// check-lens-contract.mjs — machine-checkable completion contract for the
// parallel-code-review specialist lenses.
//
// "A rule asks; a hook guarantees." This is the guarantee for the review lens
// output contract. Each lens in .agents/skills/parallel-code-review/reviewers/
// promises a specific output shape (a JSON array with required keys, or a
// structured block). If a lens's declared output drifts from what this script
// expects, the check FAILS — a stale contract is a broken guarantee, the same
// class of bug gsd's `check:contract-drift` treats as a build failure.
//
// The single source of truth for the contract is this file's LENS_CONTRACTS
// table below. CONTRIBUTING.md names it as such. Change a lens's output shape
// and you MUST change the matching row here (or the check tells you to).
//
// Usage:
//   node scripts/check-lens-contract.mjs          # check every lens
//   node scripts/check-lens-contract.mjs --json   # machine-readable status
//
// Exit code: 0 if every lens matches its contract, 1 if any drift.
//
// # FUTURE CLI ---------------------------------------------------------------
// This script is the reference implementation. When skillgrid gains a CLI, this
// becomes a verb rather than a standalone script:
//
//   skillgrid check lens-contract            # same as this script
//   skillgrid check lens-contract --changed <ref>   # scope to lenses touched since <ref>
//
// The CLI would wrap this exact logic (extract the per-lens expectation, grep the
// lens file, compare) and add:
//   - `--changed <ref>`: run `git diff --name-only <ref> HEAD -- .agents/skills/
//     parallel-code-review/reviewers/` and check only the touched lenses, so a
//     PR that doesn't touch a lens never fails on it.
//   - machine-readable status on stdout (a JSON line per lens: name, expected,
//     found, ok) so CI can render per-lens results and a future `--fail-on-warn`.
//   - a `--list` that prints the contract table without checking.
//
// Until then, this script keeps working standalone — wire it into CI as a
// second check alongside test-hooks.mjs, and add a row to LENS_CONTRACTS for any
// new lens. The table, not the prose in each lens, is the contract.
// -----------------------------------------------------------------------------

import { readFileSync, existsSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const LENSES = path.join(ROOT, ".agents/skills/parallel-code-review/reviewers");
const jsonFlag = process.argv[2] === "--json";

// LENS_CONTRACTS: one row per lens, "filename|kind|required-keys-or-pattern".
//   kind = json  -> the lens must declare each of the comma-separated keys in a
//                   JSON finding object (grep for the literal key string).
//   kind = block -> the lens must declare each of the comma-separated block
//                   field markers (grep for the literal marker string).
// A lens file missing a required marker = DRIFT.
const LENS_CONTRACTS = [
  { fname: "security.md", kind: "json", spec: "location,issue,attack_or_exposure,fix" },
  { fname: "accessibility.md", kind: "json", spec: "location,issue,who_it_blocks,fix" },
  { fname: "performance.md", kind: "json", spec: "location,issue,cost_and_scale,fix" },
  { fname: "edge-case-hunter.md", kind: "json", spec: "location,trigger_condition,guard_snippet,potential_consequence,kind" },
  { fname: "verification-gap.md", kind: "block", spec: "Changed surface,Impacted consumer,Existing test evidence,Missing verification,Demonstration,Disposition,gap_type" },
  { fname: "red-team.md", kind: "json", spec: "location,issue,missed_by,consequence" },
];

let FAIL = 0;
const LINES = [];

// check_json <file> <comma-keys>
function check_json(file, keys) {
  const content = readFileSync(file, "utf8");
  let ok = 0;
  for (const k of keys.split(",")) {
    if (!content.includes(`"${k}"`)) {
      LINES.push(`DRIFT  ${path.basename(file)}: missing json key "${k}"`);
      ok = 1;
    }
  }
  // a json lens must actually emit a JSON array
  if (!content.includes("Return ONLY a valid JSON array")) {
    LINES.push(`DRIFT  ${path.basename(file)}: missing 'Return ONLY a valid JSON array'`);
    ok = 1;
  }
  return ok;
}

// check_block <file> <comma-markers>
function check_block(file, markers) {
  const content = readFileSync(file, "utf8");
  let ok = 0;
  for (const m of markers.split(",")) {
    if (!content.includes(m)) {
      LINES.push(`DRIFT  ${path.basename(file)}: missing block marker '${m}'`);
      ok = 1;
    }
  }
  return ok;
}

for (const row of LENS_CONTRACTS) {
  const { fname, kind, spec } = row;
  const file = path.join(LENSES, fname);
  if (!existsSync(file)) {
    LINES.push(`DRIFT  ${fname}: lens file not found`);
    FAIL = 1;
    continue;
  }
  let result;
  switch (kind) {
    case "json":
      result = check_json(file, spec);
      break;
    case "block":
      result = check_block(file, spec);
      break;
    default:
      LINES.push(`DRIFT  ${fname}: unknown contract kind '${kind}'`);
      FAIL = 1;
      continue;
  }
  if (result !== 0) FAIL = 1;
}

const LENS_COUNT = LENS_CONTRACTS.length;

if (FAIL === 0) {
  LINES.push(`OK     all ${LENS_COUNT} lens contracts match`);
}

if (jsonFlag) {
  for (const line of LINES) {
    console.log(JSON.stringify({ status: line }));
  }
} else {
  for (const line of LINES) console.log(line);
  console.log("\n========================================");
  if (FAIL === 0) {
    console.log("Lens contract: PASS");
  } else {
    console.log("Lens contract: FAIL — a lens drifted from its declared output contract.");
    console.log("Update scripts/check-lens-contract.mjs (LENS_CONTRACTS) or the lens file.");
  }
}

process.exit(FAIL === 0 ? 0 : 1);
