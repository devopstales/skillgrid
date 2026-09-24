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
 * ADR: 04-adr-0008 (Node.js + yaml package, read-only verifier)
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

function hasFilledVerdict(reportContent) {
  const m = reportContent.match(/\*\*Verdict:\*\*\s*(\w+)/);
  return m && m[1] !== "" && m[1] !== "PASS / CONCERNS / FAIL / WAIVED";
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

// --- Main ---

function main() {
  const root = resolve(process.argv[2] || ".");
  const sgDir = join(root, ".skillgrid");
  const statePath = join(sgDir, "state.yaml");
  const specsDir = join(sgDir, "specs");
  const sddDir = join(sgDir, "sdd");

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

  const currentPhase = state?.pipeline?.current_phase || "";
  const currentChange = state?.pipeline?.current_change || "";
  const completedChanges = state?.progress?.completed_changes ?? 0;

  // Find active change dir(s)
  let activeDirs = [];
  if (existsSync(specsDir)) {
    activeDirs = readdirSync(specsDir, { withFileTypes: true })
      .filter((e) => e.isDirectory())
      .map((e) => e.name)
      .sort();
  }

  // No active change: if state is also clear, clean; if state names a change, drift
  if (activeDirs.length === 0) {
    if (currentChange === "" || currentChange === undefined) {
      console.log("DRIFT: none");
      process.exit(0);
    }
    // state names a change but no specs dir exists → drift
    const drifts = [
      {
        field: "current_change",
        stale: currentChange,
        derived: "(none — no active change in specs/)",
      },
    ];
    printDrift(drifts);
    process.exit(1);
  }

  // Determine the active change: if state names one, use it; else use newest
  const stateNamed = activeDirs.includes(currentChange);
  const activeChange = stateNamed ? currentChange : activeDirs[activeDirs.length - 1];
  const changeDir = join(specsDir, activeChange);

  const drifts = [];

  // Check 1: phase — only when there's an active change to derive from.
  // When current_change is empty (shipped/idle), the dirs under specs/ are
  // closed changes, not the active one — no phase to derive.
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

  // Check 2: change — only flags when current_change is non-empty and
  // doesn't match any dir under specs/. An empty current_change with dirs
  // present is the normal post-ship / idle state (no in-flight change).
  if (currentChange !== "" && !stateNamed) {
    drifts.push({
      field: "pipeline.current_change",
      stale: currentChange,
      derived: activeChange,
    });
  }

  // Check 3: completion — scan ALL active change dirs for shippable
  // verdicts. If any change has a shippable verdict (PASS, WAIVED, or
  // CONCERNS+override) and completed_changes is 0, that's drift: at least
  // one change has reached the ship-able state but the counter wasn't
  // incremented.
  for (const dir of activeDirs) {
    const dirPath = join(specsDir, dir);
    const reportPath = join(dirPath, "report.md");
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

  if (drifts.length === 0) {
    console.log("DRIFT: none");
    process.exit(0);
  }

  printDrift(drifts);
  process.exit(1);
}

function printDrift(drifts) {
  console.log("DRIFT DETECTED:");
  console.log("");
  console.log("| Field | Stale (state.yaml) | Derived (spec zone) |");
  console.log("|-------|-------------------|---------------------|");
  for (const d of drifts) {
    console.log(`| ${d.field} | ${d.stale} | ${d.derived} |`);
  }
  console.log("");
  console.log("Fix: update .skillgrid/state.yaml to match the spec-zone artifacts.");
}

main();
