#!/usr/bin/env node
/**
 * state-drift-check.mjs — read-only drift guard for .skillgrid/state.yaml.
 *
 * Compares state.yaml against the spec zone (.skillgrid/specs/) and reports
 * a named drift verdict. Does NOT own the write — pipeline skills keep
 * writing state.yaml directly; this guard owns the verification.
 *
 * Usage: node scripts/state-drift-check.mjs [project-root]
 *   project-root defaults to the current working directory.
 *
 * Exit codes:
 *   0 — clean (no drift)
 *   1 — drift detected (prints field/stale/derived table)
 *   2 — usage/parse error (state.yaml missing or unparseable)
 *
 * state/v2 (parallel development): state.yaml carries a `changes:` map,
 * one entry per in-flight change. The guard checks, for EVERY change dir
 * under specs/ and EVERY map entry:
 *   1 presence  — a specs/ dir without a map entry (or vice versa) is drift
 *   2 phase     — map phase ≠ phase derived from the dir's artifacts is drift
 *
 * Legacy v1 (single `pipeline.current_change`) still parses and is checked
 * against the newest specs/ dir, so pre-migration state files keep working.
 *
 * ADR: ASSUMPTIONS.md § ### ADR-0008 (Node.js + yaml package, read-only verifier)
 */

import { readFileSync, readdirSync, existsSync } from "node:fs";
import { join, resolve } from "node:path";
import { parse } from "yaml";

// --- Phase derivation from spec-zone artifacts ---
// Matches the Artifact File Paths table in sdd-structure.md.
// The deepest present artifact determines the phase.

function derivePhase(changeDir) {
  const briefing = existsSync(join(changeDir, "briefing.md"));
  const blueprint = existsSync(join(changeDir, "blueprint.md"));
  const tasks = existsSync(join(changeDir, "tasks.md"));
  const report = existsSync(join(changeDir, "report.md"));

  if (!briefing) return null;

  if (report) return "qa";
  if (tasks && hasExecutionSignal(changeDir)) return "execution";
  if (tasks) return "slicing";
  if (blueprint) return "blueprint";
  return "spec";
}

function hasExecutionSignal(changeDir) {
  const sddDir = join(resolve(changeDir, "..", ".."), "sdd");
  if (existsSync(sddDir)) {
    for (const entry of readdirSync(sddDir, { withFileTypes: true })) {
      if (!entry.isDirectory()) continue;
      const progress = join(sddDir, entry.name, "progress.md");
      if (existsSync(progress)) {
        const content = readFileSync(progress, "utf8");
        if (/\[x\]/i.test(content)) return true;
      }
    }
  }
  const tasksPath = join(changeDir, "tasks.md");
  if (existsSync(tasksPath)) {
    const content = readFileSync(tasksPath, "utf8");
    if (/\[x\]/i.test(content)) return true;
  }
  return false;
}

// --- Completion detection from report.md ---

// --- Scope derivation (verification-scope.md) ---
// A zero count is never a bare zero: it carries the scope of the input the
// derivation actually saw. Four atoms, worst-scope-wins when composed.
// Severity: UNREADABLE > UNSCOPED > TRUNCATED > COMPLETE.
const SCOPE_RANK = { COMPLETE: 0, TRUNCATED: 1, UNSCOPED: 2, UNREADABLE: 3 };

function worstScope(a, b) {
  return SCOPE_RANK[b] > SCOPE_RANK[a] ? b : a;
}

function specsDirScope(specsDir) {
  if (!existsSync(specsDir)) return "UNSCOPED";
  try {
    readdirSync(specsDir, { withFileTypes: true });
    return "COMPLETE";
  } catch {
    return "UNREADABLE";
  }
}

function changeDirScope(changeDir) {
  if (!existsSync(changeDir)) return "UNREADABLE";
  try {
    readdirSync(changeDir, { withFileTypes: true });
    return "COMPLETE";
  } catch {
    return "UNREADABLE";
  }
}

function isShippable(reportContent) {
  const verdictMatch = reportContent.match(/\*\*Verdict:\*\*\s*(\w+)/);
  if (!verdictMatch) return false;
  const verdict = verdictMatch[1].toUpperCase();
  if (verdict === "PASS" || verdict === "WAIVED") return true;
  if (verdict === "CONCERNS") {
    const overrideMatch = reportContent.match(
      /\*\*Human decision \(fill in\):\*\*\s*(.+)/
    );
    return overrideMatch && overrideMatch[1].trim() !== "";
  }
  return false;
}

// --- v2: the changes map ---
// Returns Map<changeSlug, { phase, status }> or null when absent.

function readChangesMap(state) {
  if (state && state.changes && typeof state.changes === "object") {
    return new Map(Object.entries(state.changes));
  }
  return null;
}

// --- Main ---

function main() {
  const root = resolve(process.argv[2] || ".");
  const sgDir = join(root, ".skillgrid");
  const statePath = join(sgDir, "state.yaml");
  const specsDir = join(sgDir, "specs");

  // Parse state.yaml
  if (!existsSync(statePath)) {
    console.error(`ERROR: ${statePath} not found`);
    process.exit(2);
  }

  let state;
  try {
    state = parse(readFileSync(statePath, "utf8"));
  } catch (e) {
    console.error(`ERROR: cannot parse ${statePath}: ${e.message}`);
    process.exit(2);
  }

  const completedChanges = state?.progress?.completed_changes ?? 0;
  const changesMap = readChangesMap(state);

  // Enumerate change dirs under specs/
  let activeDirs = [];
  if (existsSync(specsDir)) {
    activeDirs = readdirSync(specsDir, { withFileTypes: true })
      .filter((e) => e.isDirectory())
      .map((e) => e.name)
      .sort();
  }

  const drifts = [];
  let scope = specsDirScope(specsDir);

  if (changesMap) {
    // --- v2: bidirectional presence + per-change phase ---
    for (const dir of activeDirs) {
      scope = worstScope(scope, changeDirScope(join(specsDir, dir)));
      if (!changesMap.has(dir)) {
        drifts.push({
          field: "changes",
          stale: `(no entry)`,
          derived: `${dir} present in specs/ — add changes.${dir}`,
        });
        continue;
      }
      const derivedPhase = derivePhase(join(specsDir, dir));
      const entry = changesMap.get(dir);
      if (derivedPhase !== null && entry?.phase && entry.phase !== derivedPhase) {
        drifts.push({
          field: `changes.${dir}.phase`,
          stale: String(entry.phase),
          derived: derivedPhase,
        });
      }
    }
    for (const slug of [...changesMap.keys()].sort()) {
      if (!activeDirs.includes(slug)) {
        drifts.push({
          field: "changes",
          stale: slug,
          derived: `(no specs/${slug} dir — remove the entry or restore the dir)`,
        });
      }
    }

    // Check 3 (completion) — v2: any shippable change whose entry is still
    // mapped (not shipped) while completed_changes is 0.
    for (const dir of activeDirs) {
      const reportPath = join(specsDir, dir, "report.md");
      if (!existsSync(reportPath)) continue;
      const reportContent = readFileSync(reportPath, "utf8");
      if (isShippable(reportContent) && completedChanges < 1) {
        drifts.push({
          field: "progress.completed_changes",
          stale: String(completedChanges),
          derived: `≥ 1 (shippable verdict for ${dir}, not yet counted)`,
        });
        break;
      }
    }
  } else {
    // --- Legacy v1: single pipeline.current_change ---
    const currentPhase = state?.pipeline?.current_phase || "";
    const currentChange = state?.pipeline?.current_change || "";

    if (activeDirs.length === 0) {
      if (currentChange !== "" && currentChange !== undefined) {
        drifts.push({
          field: "pipeline.current_change",
          stale: currentChange,
          derived: "(none — no active change in specs/)",
        });
      }
    } else {
      const stateNamed = activeDirs.includes(currentChange);
      const activeChange = stateNamed
        ? currentChange
        : activeDirs[activeDirs.length - 1];
      const changeDir = join(specsDir, activeChange);
      scope = worstScope(scope, changeDirScope(changeDir));

      if (currentChange !== "" && stateNamed) {
        const derivedPhase = derivePhase(changeDir);
        if (derivedPhase !== null && currentPhase !== derivedPhase) {
          drifts.push({
            field: "pipeline.current_phase",
            stale: currentPhase || "(empty)",
            derived: derivedPhase,
          });
        }
      }
      if (currentChange !== "" && !stateNamed) {
        drifts.push({
          field: "pipeline.current_change",
          stale: currentChange,
          derived: activeChange,
        });
      }
      for (const dir of activeDirs) {
        const reportPath = join(specsDir, dir, "report.md");
        if (!existsSync(reportPath)) continue;
        const reportContent = readFileSync(reportPath, "utf8");
        if (isShippable(reportContent) && completedChanges < 1) {
          drifts.push({
            field: "progress.completed_changes",
            stale: String(completedChanges),
            derived: `≥ 1 (shippable verdict for ${dir}, not yet counted)`,
          });
          break;
        }
      }
    }
  }

  if (drifts.length === 0) {
    console.log("DRIFT: none");
    console.log(`SCOPE: ${scope}`);
    process.exit(0);
  }

  printDrift(drifts, scope);
  process.exit(1);
}

function printDrift(drifts, scope) {
  console.log("DRIFT DETECTED:");
  console.log("");
  console.log("| Field | Stale (state.yaml) | Derived (spec zone) |");
  console.log("|-------|-------------------|---------------------|");
  for (const d of drifts) {
    console.log(`| ${d.field} | ${d.stale} | ${d.derived} |`);
  }
  console.log("");
  console.log(`SCOPE: ${scope}`);
  console.log(
    "Fix: update .skillgrid/state.yaml to match the spec-zone artifacts (changes map in v2)."
  );
}

main();
