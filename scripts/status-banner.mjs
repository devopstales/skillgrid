#!/usr/bin/env node
/**
 * status-banner.mjs — enforces the sdd-structure.md § STATUS Banner contract.
 *
 * The banner `> **STATUS:** \`<value>\` (YYYY-MM-DD)` is a machine-readable
 * contract (the /plans endpoint parses it) but nothing enforced it. This guard
 * checks, for every spec dir:
 *   - briefing.md carries a banner (required by the contract)
 *   - tasks.md carries a banner once slicing has produced it
 *   - the value is in the allowed set
 *   - the value is consistent with the phase derived from the artifacts
 *
 * Per `_shared/craft/deterministic-boundary.md`: the value is a pure function
 * of (file type, derived phase), so the check is deterministic.
 *
 * Usage:
 *   node scripts/status-banner.mjs check [project-root]
 *   node scripts/status-banner.mjs write <slug> [project-root]
 *
 * Exit codes:
 *   0 — clean (no banner problems)
 *   1 — banner problems found (prints the findings)
 *   2 — usage/parse error
 */

import { readdirSync, existsSync, readFileSync, writeFileSync } from "node:fs";
import { join, resolve } from "node:path";
import {
  parseStatusBanner,
  STATUS_VALUES,
  derivePhase,
  pathExists,
} from "./lib/skillgrid-parse.mjs";

// The phase a banner value is "consistent with." A banner is a coarse
// lifecycle label; map each allowed value to the phases where it is correct.
const VALUE_PHASES = {
  draft: new Set(["spec", "blueprint", "unknown"]),
  sliced: new Set(["slicing"]),
  "in-progress": new Set(["execution", "qa", "review"]),
  complete: new Set(["review", "complete"]),
  shipped: new Set(["complete"]),
  // side states: no phase-consistency check (explicit overrides)
  stalled: null,
  blocked: null,
  revised: null,
  superseded: null,
};

function listSpecs(root) {
  const specsDir = join(root, ".skillgrid", "specs");
  if (!pathExists(specsDir)) return [];
  return readdirSync(specsDir, { withFileTypes: true })
    .filter((e) => e.isDirectory())
    .map((e) => e.name)
    .sort();
}

function read(p) {
  try {
    return readFileSync(p, "utf8");
  } catch {
    return null;
  }
}

/** Check one change dir. Returns a findings array (empty = clean). */
function checkChange(root, slug) {
  const dir = join(root, ".skillgrid", "specs", slug);
  const findings = [];
  const sddDir = join(root, ".skillgrid", "sdd");
  const { phase } = derivePhase({ changeDir: dir, sddDir });

  const briefing = read(join(dir, "briefing.md"));
  if (briefing === null) {
    // No briefing → not a real change dir; nothing to check.
    return findings;
  }

  // briefing.md MUST carry a banner.
  const bBanner = parseStatusBanner(briefing);
  if (!bBanner) {
    findings.push(`briefing.md: missing STATUS banner (add \`> **STATUS:** \`draft\` (YYYY-MM-DD)\` on line 3)`);
  } else if (!STATUS_VALUES.has(bBanner.value)) {
    findings.push(`briefing.md: STATUS value \`${bBanner.value}\` not in allowed set`);
  } else {
    const okPhases = VALUE_PHASES[bBanner.value];
    if (okPhases && !okPhases.has(phase)) {
      findings.push(`briefing.md: STATUS \`${bBanner.value}\` inconsistent with derived phase \`${phase}\``);
    }
  }

  // tasks.md MUST carry a banner once it exists.
  const tasks = read(join(dir, "tasks.md"));
  if (tasks !== null) {
    const tBanner = parseStatusBanner(tasks);
    if (!tBanner) {
      findings.push(`tasks.md: missing STATUS banner`);
    } else if (!STATUS_VALUES.has(tBanner.value)) {
      findings.push(`tasks.md: STATUS value \`${tBanner.value}\` not in allowed set`);
    } else {
      const okPhases = VALUE_PHASES[tBanner.value];
      if (okPhases && !okPhases.has(phase)) {
        findings.push(`tasks.md: STATUS \`${tBanner.value}\` inconsistent with derived phase \`${phase}\``);
      }
    }
  }

  return findings;
}

/** The banner value the phase implies (for `write`). */
function bannerValueForPhase(phase) {
  switch (phase) {
    case "slicing": return "sliced";
    case "execution":
    case "qa":
    case "review": return "in-progress";
    case "complete": return "shipped";
    default: return "draft";
  }
}

/** Rewrite the banner line in a file to the given value+date. Returns the new content. */
function setBanner(content, value, date) {
  const lines = content.split("\n");
  const newLine = `> **STATUS:** \`${value}\` (${date})`;
  // Find an existing banner line and replace it; else insert after line 2 (index 1)
  // so it lands on line 3 per the contract.
  const idx = lines.findIndex((l) => l.match(/^>\s*\*\*STATUS:\*\*/));
  if (idx >= 0) {
    lines[idx] = newLine;
  } else {
    lines.splice(2, 0, newLine);
  }
  return lines.join("\n");
}

function main() {
  const args = process.argv.slice(2);
  const cmd = args[0];
  if (!cmd || (cmd !== "check" && cmd !== "write")) {
    console.error("Usage: status-banner.mjs <check|write [slug]> [project-root]");
    process.exit(2);
  }
  const rest = args.slice(1).filter((a) => a !== "write");
  const slugArg = cmd === "write" ? args[1] : null;
  const root = resolve(rest[rest.length - 1] || ".");

  if (cmd === "check") {
    const all = listSpecs(root);
    let found = 0;
    for (const slug of all) {
      for (const f of checkChange(root, slug)) {
        console.log(`${slug}: ${f}`);
        found++;
      }
    }
    if (found === 0) {
      console.log("STATUS BANNER: clean");
      process.exit(0);
    }
    console.log(`\n${found} STATUS banner finding(s).`);
    process.exit(1);
  }

  // write <slug>
  if (!slugArg) {
    console.error("Usage: status-banner.mjs write <slug> [project-root]");
    process.exit(2);
  }
  const dir = join(root, ".skillgrid", "specs", slugArg);
  if (!pathExists(dir)) {
    console.error(`ERROR: no spec dir for '${slugArg}'`);
    process.exit(2);
  }
  const sddDir = join(root, ".skillgrid", "sdd");
  const { phase } = derivePhase({ changeDir: dir, sddDir });
  const value = bannerValueForPhase(phase);
  const date = new Date().toISOString().slice(0, 10);

  for (const file of ["briefing.md", "tasks.md"]) {
    const p = join(dir, file);
    const content = read(p);
    if (content === null) continue;
    writeFileSync(p, setBanner(content, value, date));
    console.log(`wrote ${file}: ${value} (${date})`);
  }
  process.exit(0);
}

main();
