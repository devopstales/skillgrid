#!/usr/bin/env node
// requirement-coverage.mjs — plan-check coverage gate for spec folders.
//
// Verifies that every in-scope requirement in briefing.md has a row in
// blueprint.md's `## Validation` matrix. A planner that drops a requirement
// from the matrix is silently descopeing it — this checker names the gap
// before slicing publishes.
//
// Usage:
//   node requirement-coverage.mjs <change-dir>
//   <change-dir> contains briefing.md and blueprint.md
//
// Contract (per _shared/craft/deterministic-boundary.md "agentic script
// contract"): JSON on stdout, diagnostics on stderr, exit 0 = pass,
// exit 1 = findings (an uncovered requirement), exit 2 = usage error.
// Read-only — never writes.
//
// Requirement heading forms accepted (briefing.md), in priority order:
//   1. `#### Req <N> — <title> [(locked)]`      (Release/2 numbered form)
//   2. `#### <N>. <title> [(locked)]`            (numbered heading form)
//   3. `N. **[label]:**`                         (template list form, numbering)
// "(locked)" in the heading marks a locked requirement — uncovered locked
// reqs are reported as "locked" findings, uncovered non-locked reqs as
// "coverage" findings. Either kind fails the check. A heading that says
// "not locked" (or "no lock") is treated as non-locked.

import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

function fail2(msg) {
  console.error(`usage: requirement-coverage.mjs <change-dir> [--skip-satifies]`);
  console.error(msg);
  process.exit(2);
}

const args = process.argv.slice(2);
const changeDir = args.find((a) => !a.startsWith("--"));
const skipSatifies = args.includes("--skip-satifies");
if (!changeDir) fail2("missing <change-dir>");

const briefingPath = join(changeDir, "briefing.md");
const blueprintPath = join(changeDir, "blueprint.md");
if (!existsSync(briefingPath)) fail2(`no briefing.md in ${changeDir}`);
if (!existsSync(blueprintPath)) fail2(`no blueprint.md in ${changeDir}`);

const briefing = readFileSync(briefingPath, "utf8");
const blueprint = readFileSync(blueprintPath, "utf8");

// --- extract requirement IDs from briefing.md --------------------------------

function extractRequirements(text) {
  const reqs = new Map(); // id -> { id, title, locked }
  const seen = new Set();
  for (const raw of text.split("\n")) {
    const line = raw.trim();
    let m = line.match(/^####\s+Req\s+(\d+)\s+—?\s*(.*)$/);
    if (!m) m = line.match(/^####\s+(\d+)\.\s+(.*)$/);
    if (!m) m = line.match(/^(\d+)\.\s+\*\*[^*]+:\*\*/);
    if (!m) continue;
    const id = m[1];
    const title = (m[2] || "").replace(/\(locked\)\s*$/i, "").trim();
    const locked = /locked/i.test(line) && !/no[\s-]*lock/i.test(line);
    if (!seen.has(id)) {
      seen.add(id);
      reqs.set(id, { id, title, locked });
    }
  }
  return reqs;
}

// --- extract `## Validation` row IDs from blueprint.md ------------------------

function extractValidationRows(text) {
  const ids = new Set();
  let inSection = false;
  for (const raw of text.split("\n")) {
    const line = raw.trim();
    if (line.startsWith("## Validation")) {
      inSection = true;
      continue;
    }
    if (inSection && line.startsWith("## ")) break; // next H2 ends the section
    if (!inSection || !line.startsWith("|")) continue;
    let cells = line.split("|");
    if (cells[0] === "") cells = cells.slice(1);
    if (cells[cells.length - 1] === "") cells = cells.slice(0, -1);
    const id = cells[0].trim();
    if (/^\d+$/.test(id)) ids.add(id); // header row ("Req") and legend rows are skipped
  }
  return ids;
}

// --- extract scenario names from acceptance.feature ---------------------------

function extractScenarios(text) {
  const names = new Set();
  for (const raw of text.split("\n")) {
    const line = raw.trim();
    // Gherkin scenario headings: `Scenario: name`, `Scenario Outline: name`,
    // and the indented `#### Scenario: name` form some briefings emit.
    const m = line.match(/^(?:#{1,6}\s+)?Scenario(?:\s+Outline)?:\s*(.+)$/i);
    if (m) {
      const name = m[1].replace(/\(locked\)\s*$/i, "").trim();
      if (name) names.add(name);
    }
  }
  return names;
}

// --- extract SATISFIES names from tasks.md -------------------------------------

function extractSatifies(text) {
  const names = new Set();
  for (const raw of text.split("\n")) {
    const line = raw;
    const m = line.match(/\*\*SATISFIES:\*\*\s*(.*)$/);
    if (!m) continue;
    for (const part of m[1].split(",")) {
      const name = part.trim().replace(/^`|`$/g, "");
      if (name) names.add(name);
    }
  }
  return names;
}

// --- check -------------------------------------------------------------------

const reqs = extractRequirements(briefing);
const rows = extractValidationRows(blueprint);

const findings = [];
if (reqs.size === 0) {
  findings.push({
    kind: "parse",
    subject: "briefing.md",
    message: "no numbered requirements found (#### Req N / #### N. / N. **label:**) — coverage check skipped",
  });
}
for (const r of reqs.values()) {
  if (!rows.has(r.id)) {
    findings.push({
      kind: r.locked ? "locked" : "coverage",
      subject: r.id,
      message: `requirement ${r.id}${r.locked ? " (locked)" : ""} has no row in blueprint.md's ## Validation table`,
    });
  }
}

// --- SATISFIES <-> acceptance.feature cross-check -----------------------------
// The traceability oracle: every scenario a ticket claims to satisfy must
// exist in acceptance.feature, and every scenario must be claimed by at least
// one ticket. Either direction dangling means the acceptance spec and the
// plan have drifted apart. Skipped when tasks.md or acceptance.feature is
// absent (a blueprint without slicing yet), or with --skip-satifies.

const tasksPath = join(changeDir, "tasks.md");
const acceptancePath = join(changeDir, "acceptance.feature");
let satifiesChecked = false;
if (!skipSatifies && existsSync(tasksPath) && existsSync(acceptancePath)) {
  satifiesChecked = true;
  const satifies = extractSatifies(readFileSync(tasksPath, "utf8"));
  const scenarios = extractScenarios(readFileSync(acceptancePath, "utf8"));
  for (const name of [...satifies].sort()) {
    if (!scenarios.has(name)) {
      findings.push({
        kind: "satifies-dangling",
        subject: name,
        message: `tasks.md SATISFIES "${name}" but acceptance.feature has no scenario by that name`,
      });
    }
  }
  for (const name of [...scenarios].sort()) {
    if (!satifies.has(name)) {
      findings.push({
        kind: "scenario-unclaimed",
        subject: name,
        message: `acceptance.feature scenario "${name}" is not claimed by any tasks.md SATISFIES`,
      });
    }
  }
}

const result = {
  status: findings.length ? "fail" : "pass",
  requirements: reqs.size,
  covered: rows.size,
  satifies_checked: satifiesChecked,
  findings,
};
console.log(JSON.stringify(result, null, 2));
process.exit(findings.length ? 1 : 0);
