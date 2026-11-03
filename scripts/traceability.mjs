#!/usr/bin/env node
/**
 * traceability.mjs — deterministic traceability matrix generator for the QA gate.
 *
 * The QA skill (Step 5) must count the ACTUAL scenarios in acceptance.feature
 * ("never invent totals") and map each to a covering test. This script does
 * that enumeration + counting deterministically so the total is exact by
 * construction. The agent then only judges PARTIAL vs UNTESTED where the match
 * is ambiguous.
 *
 * What it reads:
 *   - <change-dir>/acceptance.feature  → the scenarios (the traceability oracle)
 *   - <change-dir>/tasks.md            → each ticket's SATISFIES line (which
 *     scenario each ticket claims to cover)
 *   - --test-output=<file>             (optional) test runner output, for
 *     `--- PASS:` / `--- FAIL:` / `ok ` lines, to mark scenarios that actually
 *     ran and passed
 *
 * Per-scenario status:
 *   COMPLIANT — a covering test exists and passed at runtime
 *   FAILING   — a covering test exists and failed
 *   UNTESTED  — no covering test exists (a traceability gap)
 *   (PARTIAL is a judgment the agent makes where a test covers only part —
 *    this script marks it UNTESTED/COMPLIANT and the agent refines.)
 *
 * Usage:
 *   node scripts/traceability.mjs <slug> [--test-output=<file>] [project-root]
 *
 * Exit codes:
 *   0 — matrix produced (exit reflects scope, not findings — the QA skill
 *       renders the verdict; a non-zero scenario count is data, not an error)
 *   2 — usage error (no acceptance.feature / not a skillgrid project)
 *
 * Output (stdout): a single JSON object:
 * {
 *   "summary": { "total", "requirements", "compliant", "failing", "untested" },
 *   "scenarios": [ { "name", "requirement", "status", "satisfies_refs": [...],
 *                    "test": { "name", "passed" } | null } ],
 *   "scenarios": "N/N"   // exact count, the QA report's `scenarios: N/N`
 *   "scope": "COMPLETE" | "UNREADABLE"
 * }
 * Deterministic: same files → same JSON, run to run.
 */

import { readFileSync, existsSync } from "node:fs";
import { join, resolve } from "node:path";
import { parseScenarios, normalizeName, pathExists } from "./lib/skillgrid-parse.mjs";

function read(p) {
  try {
    return readFileSync(p, "utf8");
  } catch {
    return null;
  }
}

/** Parse test runner output into { testName → passed } for `--- PASS/FAIL`. */
function parseTestOutput(text) {
  const result = new Map();
  if (!text) return result;
  for (const line of text.split("\n")) {
    let m = line.match(/^--- (PASS|FAIL):\s*(\S+)/);
    if (m) {
      result.set(m[2], m[1] === "PASS");
      continue;
    }
    // Go: a bare `ok` / `FAIL` package line doesn't name a test; skip.
  }
  return result;
}

/** Match a scenario name to a test name (normalized, substring either way). */
function matchTest(scenarioName, testMap) {
  const sn = normalizeName(scenarioName);
  // A test "covers" a scenario if one name contains the other, ignoring the
  // `happy path —` / `edge —` / `failure —` prefix on the scenario side (those
  // are classification, not behavior). Compare both the spaced form and the
  // space-stripped form so camelCase Go test names (TestThingWorks) match.
  const scenarioCore = sn.replace(/^(happy path|edge|failure)(?:\s+|$)/, "").trim();
  const coreFlat = scenarioCore.replace(/\s+/g, "");
  for (const [testName, passed] of testMap) {
    const tn = normalizeName(testName);
    const tnFlat = tn.replace(/\s+/g, "");
    if (tn.includes(scenarioCore) || scenarioCore.includes(tn) ||
        tnFlat.includes(coreFlat) || coreFlat.includes(tnFlat)) {
      return { name: testName, passed };
    }
  }
  return null;
}

/** Parse each ticket's SATISFIES line from tasks.md → scenarioName → [ticketIds]. */
function parseSatisfiesRefMap(text, scenarioNames) {
  const refs = new Map(); // normalized scenario name → [ticket]
  for (const s of scenarioNames) refs.set(normalizeName(s.name), []);
  if (!text) return refs;
  let currentTicket = null;
  for (const line of text.split("\n")) {
    const t = line.match(/^###\s+(TICKET-\d+|Task\s+\d+|[A-Za-z0-9-]+)\b/i);
    if (t) {
      currentTicket = t[1].trim();
      continue;
    }
    const sat = line.match(/\*\*SATISFIES:?\*\*\s*(.+)/i);
    if (sat && currentTicket) {
      // SATISFIES is `;`- or `,`-separated scenario names (sometimes with
      // backticks or prose). Match each against the known scenario set.
      const candidates = sat[1].split(/[;,]/).map((c) => c.replace(/`/g, "").trim());
      for (const cand of candidates) {
        const nc = normalizeName(cand);
        // Exact or substring match against a known scenario.
        for (const [key, tickets] of refs) {
          if (key === nc || key.includes(nc) || nc.includes(key)) {
            if (!tickets.includes(currentTicket)) tickets.push(currentTicket);
          }
        }
      }
    }
  }
  return refs;
}

function main() {
  const args = process.argv.slice(2);
  const testOutArg = args.find((a) => a.startsWith("--test-output="));
  // The slug is the first non-flag arg; the project root is an optional trailing
  // positional (a path with a slash, or `.`).
  const positional = args.filter((a) => !a.startsWith("--"));
  const slug = positional[0];
  const rootArg = positional.length > 1 ? positional[positional.length - 1] : null;
  if (!slug) {
    console.error("Usage: traceability.mjs <slug> [--test-output=<file>] [project-root]");
    process.exit(2);
  }
  const root = resolve(rootArg || ".");

  const specDir = join(root, ".skillgrid", "specs", slug);
  const featurePath = join(specDir, "acceptance.feature");
  if (!pathExists(specDir)) {
    console.error(`ERROR: no spec dir for '${slug}'`);
    process.exit(2);
  }
  const feature = read(featurePath);
  if (feature === null) {
    console.error(`ERROR: ${featurePath} not found`);
    process.exit(2);
  }

  const scenarios = parseScenarios(feature);
  const tasks = read(join(specDir, "tasks.md"));
  const refs = parseSatisfiesRefMap(tasks, scenarios);
  const testText = testOutArg ? read(testOutArg.slice("--test-output=".length)) : null;
  const testMap = parseTestOutput(testText);

  const out = [];
  const counts = { compliant: 0, failing: 0, untested: 0 };
  for (const s of scenarios) {
    const test = matchTest(s.name, testMap);
    let status;
    if (test) status = test.passed ? "COMPLIANT" : "FAILING";
    else status = "UNTESTED";
    counts[status.toLowerCase()] = (counts[status.toLowerCase()] || 0) + 1;
    out.push({
      name: s.name,
      requirement: s.requirement,
      status,
      satisfies_refs: refs.get(normalizeName(s.name)) || [],
      test: test || null,
    });
  }

  const requirements = new Set(scenarios.map((s) => s.requirement).filter(Boolean));
  const result = {
    slug,
    summary: {
      total: scenarios.length,
      requirements: requirements.size,
      compliant: counts.compliant,
      failing: counts.failing,
      untested: counts.untested,
    },
    scenarios: out,
    "scenarios: total": `${scenarios.length}/${scenarios.length}`,
    scope: "COMPLETE",
  };
  console.log(JSON.stringify(result, null, 2));
  process.exit(0);
}

main();
