#!/usr/bin/env node
/**
 * phase-router.mjs — deterministic phase router for the skillgrid SDD pipeline.
 *
 * Reads the spec-zone artifacts of one change and prints a single JSON object
 * with the derived phase, the next skill, the drift cross-check against
 * state.yaml, the execution-ledger position, and any blocked(...) reasons.
 *
 * This is the single source of truth for "where is this change, what's next."
 * `skillgrid:resume` and `skillgrid:using-skillgrid` call it and print the one
 * line; `state-drift-check.mjs` reuses the same derivation (via
 * scripts/lib/skillgrid-parse.mjs) so the two never disagree.
 *
 * Usage:
 *   node scripts/phase-router.mjs <change-dir | slug> [project-root]
 *     <change-dir | slug>  a spec dir name (e.g. 2026-10-24-hermes-integration)
 *                          or a path to the change dir. Also `all` to route
 *                          every in-flight change.
 *     [project-root]       defaults to the current working directory.
 *
 * Exit codes:
 *   0 — routed (one or more changes resolved; no usage error)
 *   1 — drift detected (a change's state.yaml phase != derived phase)
 *   2 — usage error (no change dir / not a skillgrid project)
 *
 * Output (stdout): a single JSON object (or {"changes":[...]} for `all`).
 * Deterministic: same files → same JSON, run to run.
 */

import { readFileSync, readdirSync, existsSync } from "node:fs";
import { join, resolve, isAbsolute } from "node:path";
import { parse } from "yaml";
import {
  derivePhase,
  worstScope,
  parseLedger,
  parseBlocked,
  parseStatusBanner,
  nextSkillFor,
  pathExists,
} from "./lib/skillgrid-parse.mjs";

function read(p) {
  try {
    return readFileSync(p, "utf8");
  } catch {
    return null;
  }
}

/** Locate a change dir from a slug or path, under either specs/ or xarchive/. */
function locateChange(root, arg) {
  const sg = join(root, ".skillgrid");
  if (isAbsolute(arg)) {
    const base = resolve(arg).split("/").pop();
    const inSpecs = pathExists(join(root, ".skillgrid", "specs", base));
    const inArchive = pathExists(join(root, ".skillgrid", "xarchive", base));
    if (inSpecs) return { dir: join(root, ".skillgrid", "specs", base), slug: base, archived: false };
    if (inArchive) return { dir: join(root, ".skillgrid", "xarchive", base), slug: base, archived: true };
    if (pathExists(arg)) return { dir: resolve(arg), slug: base, archived: false };
    return null;
  }
  const inSpecs = pathExists(join(sg, "specs", arg));
  const inArchive = pathExists(join(sg, "xarchive", arg));
  if (inSpecs) return { dir: join(sg, "specs", arg), slug: arg, archived: false };
  if (inArchive) return { dir: join(sg, "xarchive", arg), slug: arg, archived: true };
  return null;
}

function readState(root) {
  const p = join(root, ".skillgrid", "state.yaml");
  const text = read(p);
  if (text === null) return null;
  try {
    return parse(text);
  } catch {
    return null;
  }
}

/** Route one change dir → the JSON record. */
function routeChange(root, located, state) {
  const { dir, slug, archived } = located;
  const sddDir = join(root, ".skillgrid", "sdd");
  const { phase, marker } = derivePhase({ changeDir: dir, sddDir, archived });

  // Execution-ledger position (only meaningful in the execution phase, but
  // always computed when a ledger exists for the plan).
  let ledger = null;
  if (phase === "execution" || phase === "slicing") {
    const plan = join(sddDir, slug);
    const progress = join(plan, "progress.md");
    if (pathExists(progress)) {
      ledger = parseLedger(read(progress));
    }
  }

  // blocked(...) reasons: from the change's state.yaml entry, or the dir.
  let blocked = [];
  const entry = state?.changes?.[slug];
  if (entry) {
    for (const key of ["phase", "status", "blocked"]) {
      if (typeof entry[key] === "string") blocked = blocked.concat(parseBlocked(entry[key]));
    }
  }

  // Drift cross-check against state.yaml (the pointer).
  let stateYamlPhase = null;
  let drift = null;
  if (state && state.changes && typeof state.changes === "object") {
    stateYamlPhase = entry?.phase ?? null;
    if (stateYamlPhase && stateYamlPhase !== phase) {
      drift = { field: `changes.${slug}.phase`, stale: stateYamlPhase, derived: phase };
    }
    if (!entry) {
      drift = { field: "changes", stale: "(no entry)", derived: `${slug} present — add changes.${slug}` };
    }
  }

  const statusBanner = {
    briefing: parseStatusBanner(read(join(dir, "briefing.md"))),
    tasks: parseStatusBanner(read(join(dir, "tasks.md"))),
  };

  const record = {
    slug,
    archived,
    phase,
    next: nextSkillFor(phase),
    marker,
    state_yaml_phase: stateYamlPhase,
    drift: drift ?? null,
    next_recommended: blocked.length ? "resolve-blockers" : nextSkillFor(phase),
    blocked,
    ledger,
    status_banner: statusBanner,
  };

  // Scope: COMPLETE when the change dir is readable (it was — we listed it).
  record.scope = "COMPLETE";
  return record;
}

function main() {
  const args = process.argv.slice(2);
  if (args.length < 1) {
    console.error("Usage: phase-router.mjs <change-dir | slug | all> [project-root]");
    process.exit(2);
  }
  const arg = args[0];
  const root = resolve(args[1] || ".");

  const sg = join(root, ".skillgrid");
  if (!pathExists(sg) || !pathExists(join(sg, "specs"))) {
    console.error(`ERROR: ${sg}/specs not found (not a skillgrid project?)`);
    process.exit(2);
  }

  const state = readState(root);

  if (arg === "all") {
    const specsDir = join(sg, "specs");
    const slugs = readdirSync(specsDir, { withFileTypes: true })
      .filter((e) => e.isDirectory())
      .map((e) => e.name)
      .sort();
    const changes = [];
    for (const slug of slugs) {
      const located = locateChange(root, slug);
      if (located) changes.push(routeChange(root, located, state));
    }
    const drifted = changes.filter((c) => c.drift);
    console.log(JSON.stringify({ changes, drifted: drifted.length }, null, 2));
    process.exit(drifted.length ? 1 : 0);
    return;
  }

  const located = locateChange(root, arg);
  if (!located) {
    console.error(`ERROR: no change dir for '${arg}' under specs/ or xarchive/`);
    process.exit(2);
  }
  const record = routeChange(root, located, state);
  console.log(JSON.stringify(record, null, 2));
  process.exit(record.drift ? 1 : 0);
}

main();
