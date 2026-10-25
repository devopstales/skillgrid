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
import { parse } from "yaml";
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
      patches.push({ field: `changes.${dir}`, from: null, to });
      continue;
    }
    // Phase drift → set to the derived phase.
    if (dp && entry.phase && entry.phase !== dp) {
      patches.push({ field: `changes.${dir}.phase`, from: String(entry.phase), to: dp });
    }
  }

  // 3: stale entries (a changes.<slug> with no specs/<slug> dir) → remove.
  for (const slug of Object.keys(changes)) {
    if (!activeDirs.includes(slug)) {
      patches.push({ field: `changes.${slug}`, from: JSON.stringify(changes[slug]), to: null });
    }
  }

  const clean = patches.length === 0;
  // The write is a surgical line-level edit of the raw YAML (see applyPatches),
  // NOT a parse/stringify round-trip — so comments and key order survive.
  const after = clean ? null : applyPatches(before, patches);

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

/**
 * Apply patches to the raw YAML text without a parse/stringify round-trip, so
 * comments, key order, and formatting survive. Handles the three patch types:
 *   - changes.<slug>.phase   → rewrite that slug's `phase:` line
 *   - changes.<slug> (add)   → append a `slug:` block under the changes map
 *   - changes.<slug> (remove)→ delete the slug's block from the changes map
 * This is intentionally conservative: it only touches the lines it must.
 */
function applyPatches(text, patches) {
  const lines = text.split("\n");

  // Find the `changes:` map boundary (top-level, no leading indent) and the
  // indent of its children (the first indented line after it).
  let changesIdx = -1;
  for (let i = 0; i < lines.length; i++) {
    if (/^changes:\s*$/.test(lines[i])) { changesIdx = i; break; }
  }
  if (changesIdx < 0) return text; // v1 / no changes map — leave untouched
  let childIndent = 2;
  for (let i = changesIdx + 1; i < lines.length; i++) {
    if (lines[i].trim() === "") continue;
    const m = lines[i].match(/^(\s+)/);
    childIndent = m ? m[1].length : 2;
    break;
  }
  // The changes map ends at the first non-empty line with indent <= 0 (a new
  // top-level key) or indent < childIndent.
  let changesEnd = lines.length;
  for (let i = changesIdx + 1; i < lines.length; i++) {
    if (lines[i].trim() === "") continue;
    const ind = (lines[i].match(/^(\s*)/)[0]).length;
    if (ind < childIndent) { changesEnd = i; break; }
  }

  // Locate each slug's block within [changesIdx+1, changesEnd).
  const slugBlock = (slug) => {
    const keyIndent = childIndent;
    const startRe = new RegExp(`^${" ".repeat(keyIndent)}${escapeRe(slug)}:\\s*$`);
    let start = -1;
    for (let i = changesIdx + 1; i < changesEnd; i++) {
      if (startRe.test(lines[i])) { start = i; break; }
    }
    if (start < 0) return null;
    // Block ends before the next line with indent <= keyIndent.
    let end = changesEnd;
    for (let i = start + 1; i < changesEnd; i++) {
      if (lines[i].trim() === "") continue;
      const ind = (lines[i].match(/^(\s*)/)[0]).length;
      if (ind <= keyIndent) { end = i; break; }
    }
    return { start, end };
  };

  // Single pass: rebuild the changes-map interior in order, preserving
  // comments/blank lines, skipping removed slugs, and rewriting a kept
  // slug's phase line when patched. This is the only mutation — no separate
  // pre-pass, so a block is emitted exactly once.
  const phasePatch = Object.fromEntries(
    patches.filter((p) => p.field.endsWith(".phase")).map((p) => [p.field.split(".").slice(1, -1).join("."), p.to])
  );
  const addPatch = Object.fromEntries(
    patches.filter((p) => p.to && typeof p.to === "object").map((p) => [p.field.split(".").slice(1).join("."), p.to])
  );
  const removeSet = new Set(
    patches.filter((p) => p.to === null).map((p) => p.field.split(".").slice(1).join("."))
  );

  const interior = [];
  for (let i = changesIdx + 1; i < changesEnd; i++) {
    const line = lines[i];
    const m = line.match(new RegExp(`^${" ".repeat(childIndent)}([^\\s:]+):\\s*$`));
    if (m && removeSet.has(m[1])) {
      // Skip this slug's entire block.
      const b = slugBlock(m[1]);
      if (b) i = b.end - 1;
      continue;
    }
    if (m && phasePatch[m[1]]) {
      // Emit the block, rewriting its phase line.
      const b = slugBlock(m[1]);
      for (let j = b.start; j < b.end; j++) {
        if (/^\s*phase:\s*/.test(lines[j])) {
          interior.push(`${" ".repeat(childIndent + 2)}phase: ${phasePatch[m[1]]}`);
        } else {
          interior.push(lines[j]);
        }
      }
      i = b.end - 1;
      continue;
    }
    if (m) {
      interior.push(...takeBlock(lines, i, changesEnd, childIndent));
      i += takeBlock(lines, i, changesEnd, childIndent).length - 1;
      continue;
    }
    interior.push(line); // comments / blank / non-slug lines preserved
  }
  // Append new entries (sorted for determinism).
  for (const slug of Object.keys(addPatch).sort()) {
    const v = addPatch[slug];
    interior.push(`${" ".repeat(childIndent)}${slug}:`);
    interior.push(`${" ".repeat(childIndent + 2)}phase: ${v.phase}`);
    interior.push(`${" ".repeat(childIndent + 2)}status: ${v.status}`);
  }

  return lines.slice(0, changesIdx + 1).concat(interior, lines.slice(changesEnd)).join("\n");
}

/** Return the lines of the block starting at `start` (a slug key line), up to
 *  the next line with indent <= `keyIndent`. */
function takeBlock(lines, start, changesEnd, keyIndent) {
  const out = [lines[start]];
  for (let i = start + 1; i < changesEnd; i++) {
    if (lines[i].trim() === "") { out.push(lines[i]); continue; }
    const ind = (lines[i].match(/^(\s*)/)[0]).length;
    if (ind <= keyIndent) break;
    out.push(lines[i]);
  }
  return out;
}

function escapeRe(s) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

main();
