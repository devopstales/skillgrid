#!/usr/bin/env node
/**
 * state-repair.mjs — explicit, reviewable repair of .skillgrid/state.yaml.
 *
 * The drift guard (state-drift-check.mjs) is read-only by design (ADR-0008:
 * "the guard owns verification, not the write"). When it reports drift, the
 * agent has hand-edited state.yaml — the failure mode the resume skill warns
 * about ("Never reset changes or progress to a clean value"). This script
 * performs that edit deterministically: it derives the correct values from the
 * spec zone, patches ONLY the drifted fields, and prints before/after. It
 * never rewrites the whole changes map or the progress block.
 *
 * It repairs the same three drift classes the guard detects (v2):
 *   1. phase drift   — changes.<slug>.phase ≠ derived phase → set to derived
 *   2. missing entry — a specs/ dir with no changes.<slug> entry → add it
 *                       (phase = derived, status = in-progress)
 *   3. stale entry   — a changes.<slug> entry with no specs/<slug> dir →
 *                       remove it (the dir is gone; the entry is dead weight)
 *
 * Usage:
 *   node scripts/state-repair.mjs [project-root] [--write]
 *
 *   --write   actually rewrite state.yaml (default is dry-run: report the
 *             patches, modify nothing). The write is explicit and reviewable —
 *             it prints the before/after of each patched field.
 *
 * Exit codes:
 *   0 — ran (dry-run always 0; --write 0 even if it patched, 0 if clean)
 *   2 — usage/parse error (state.yaml missing or unparseable)
 *
 * Output (stdout): a single JSON object:
 * {
 *   "patches": [ { "field": "changes.demo.phase", "from": "spec", "to": "slicing" }
 *                | { "field": "changes.demo", "from": null, "to": { "phase": "spec", "status": "in-progress" } }
 *                | { "field": "changes.demo", "from": "…", "to": null } ],
 *   "wrote": true | false,
 *   "clean": true | false,          // no patches
 *   "before": "<original yaml>",    // the full file, for the review
 *   "after":  "<repaired yaml>"     // null in dry-run
 * }
 * Deterministic: same state.yaml + same spec zone → same patches.
 */

import { readFileSync, writeFileSync, existsSync, readdirSync } from "node:fs";
import { join, resolve } from "node:path";
import { parse, stringify } from "yaml";
import { derivePhase as derivePhaseLib } from "./lib/skillgrid-parse.mjs";

function read(p) {
  try {
    return readFileSync(p, "utf8");
  } catch {
    return null;
  }
}

/** Derived phase for a change dir, or null when it has no briefing (unknown). */
function derivedPhase(changeDir) {
  const sddDir = join(resolve(changeDir, "..", ".."), "sdd");
  const { phase } = derivePhaseLib({ changeDir, sddDir });
  return phase === "unknown" ? null : phase;
}

function main() {
  const args = process.argv.slice(2);
  const write = args.includes("--write");
  const positional = args.filter((a) => !a.startsWith("--"));
  const root = resolve(positional[0] || ".");
  const statePath = join(root, ".skillgrid", "state.yaml");
  const specsDir = join(root, ".skillgrid", "specs");

  if (!existsSync(statePath)) {
    console.error(`ERROR: ${statePath} not found`);
    process.exit(2);
  }
  const before = read(statePath);
  let state;
  try {
    state = parse(before);
  } catch (e) {
    console.error(`ERROR: cannot parse ${statePath}: ${e.message}`);
    process.exit(2);
  }

  const changes = state?.changes && typeof state.changes === "object" ? state.changes : {};
  let activeDirs = [];
  if (existsSync(specsDir)) {
    activeDirs = readdirSync(specsDir, { withFileTypes: true })
      .filter((e) => e.isDirectory())
      .map((e) => e.name)
      .sort();
  }

  const patches = [];

  // 1 + 2: for every specs dir, ensure the entry exists with the right phase.
  for (const dir of activeDirs) {
    const dp = derivedPhase(join(specsDir, dir));
    const entry = changes[dir];
    if (!entry) {
      // Missing entry → add it (phase = derived, status = in-progress).
      const to = { phase: dp || "spec", status: "in-progress" };
      changes[dir] = to;
      patches.push({ field: `changes.${dir}`, from: null, to });
      continue;
    }
    // Phase drift → set to the derived phase.
    if (dp && entry.phase && entry.phase !== dp) {
      patches.push({ field: `changes.${dir}.phase`, from: String(entry.phase), to: dp });
      entry.phase = dp;
    }
  }

  // 3: stale entries (a changes.<slug> with no specs/<slug> dir) → remove.
  for (const slug of Object.keys(changes)) {
    if (!activeDirs.includes(slug)) {
      patches.push({ field: `changes.${slug}`, from: JSON.stringify(changes[slug]), to: null });
      delete changes[slug];
    }
  }

  // If the state has no changes map at all, leave it alone (v1 / legacy).
  if (state && !state.changes) {
    // Nothing to repair in a v1 state file; report clean.
  }

  const clean = patches.length === 0;
  const after = clean ? null : stringify(state, { lineWidth: 0 }).replace(/\r\n/g, "\n");

  if (write && !clean) {
    writeFileSync(statePath, after);
  }

  const result = {
    patches,
    wrote: write && !clean,
    clean,
    before,
    after: write && !clean ? after : null,
  };
  console.log(JSON.stringify(result, null, 2));
  process.exit(0);
}

main();
