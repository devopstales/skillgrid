#!/usr/bin/env node
/**
 * windows.mjs — deterministic broken-windows ledger (WINDOWS.md) manager.
 *
 * The broken-windows ledger is a cross-change defect register: open
 * WARNING/SUGGESTION findings not fixed before archive. The qa skill currently
 * appends each open finding with the next sequential `W<NNN>` by reading the
 * last row's number — a hand-edit that two concurrent qa runs can race. This
 * script owns the numbering + append so it is atomic and reviewable.
 *
 * Subcommands:
 *   check   — validate the existing WINDOWS.md: monotonic W<NNN> IDs (no gaps,
 *             no dupes), severity in {WARNING, SUGGESTION}, status in the
 *             allowed set. Prints the next available W<NNN>.
 *   append  <report.md path> — read the report's open WARNING/SUGGESTION
 *             findings, compute the next W<NNN> atomically, and append rows to
 *             WINDOWS.md (creating it if absent).
 *
 * Usage:
 *   node scripts/windows.mjs check [project-root]
 *   node scripts/windows.mjs append <report.md> [project-root]
 *
 * Exit codes:
 *   0 — check clean (or append succeeded)
 *   1 — check found schema violations (dupes / gaps / bad severity)
 *   2 — usage error (bad subcommand, report.md missing)
 *
 * Output (stdout): a single JSON object:
 *   check:  { "windows": N, "nextWindow": "W00N", "findings": [ {id, rule, detail} ],
 *             "ok": bool, "scope": "COMPLETE" | "UNSCOPED" }
 *   append: { "appended": N, "nextWindow": "W00N", "rows": [ {window, severity, finding} ],
 *             "ok": bool, "scope": "COMPLETE" }
 * Deterministic: same inputs → same output (append is idempotent for a given
 * report.md — it appends the open findings once).
 */

import { readFileSync, writeFileSync, existsSync, mkdirSync } from "node:fs";
import { join, resolve, dirname } from "node:path";

const WINDOWS_RE = /^\|\s*(W\d{3})\s*\|/;
const SEVERITIES = new Set(["WARNING", "SUGGESTION"]);
const STATUSES = new Set(["open", "closed", "waived"]);

function read(p) {
  try {
    return readFileSync(p, "utf8");
  } catch {
    return null;
  }
}

function windowsPath(root) {
  return join(root, ".skillgrid", "WINDOWS.md");
}

/** Parse WINDOWS.md → rows [{id, num, change, severity, finding, status}]. */
function parseWindows(text) {
  const rows = [];
  if (!text) return rows;
  for (const line of text.split("\n")) {
    const m = line.trim().match(WINDOWS_RE);
    if (!m) continue;
    const cells = line.trim().split("|").slice(1, -1).map((c) => c.trim());
    if (cells.length < 7) continue;
    rows.push({
      id: m[1],
      num: parseInt(m[1].slice(1), 10),
      change: cells[1],
      severity: cells[2],
      finding: cells[3],
      status: (cells[6] || "").toLowerCase(),
    });
  }
  return rows;
}

/** Next sequential W<NNN> after the rows (max num + 1, min W001). */
function nextWindow(rows) {
  const max = rows.reduce((m, r) => Math.max(m, r.num), 0);
  return `W${String(max + 1).padStart(3, "0")}`;
}

/** Validate a WINDOWS.md's schema. Returns { rows, findings, next }. */
function checkWindows(text) {
  const rows = parseWindows(text);
  const findings = [];
  const seen = new Map(); // id → count
  for (const r of rows) {
    if (seen.has(r.id)) {
      findings.push({ id: r.id, rule: "duplicate", detail: `${r.id} appears more than once` });
    }
    seen.set(r.id, (seen.get(r.id) || 0) + 1);
    if (!SEVERITIES.has(r.severity.toUpperCase())) {
      findings.push({ id: r.id, rule: "bad-severity", detail: `severity '${r.severity}' not in WARNING/SUGGESTION` });
    }
  }
  // Monotonic: no gaps (W001, W002, …). Only report when there are ≥ 2 rows.
  if (rows.length >= 2) {
    const nums = rows.map((r) => r.num).sort((a, b) => a - b);
    if (nums[0] !== 1) {
      findings.push({ id: `W${String(nums[0]).padStart(3, "0")}`, rule: "gap", detail: `first window is W${String(nums[0]).padStart(3, "0")}, expected W001` });
    }
    for (let i = 1; i < nums.length; i++) {
      if (nums[i] !== nums[i - 1] + 1) {
        findings.push({ id: `W${String(nums[i]).padStart(3, "0")}`, rule: "gap", detail: `expected W${String(nums[i - 1] + 1).padStart(3, "0")} before W${String(nums[i]).padStart(3, "0")}` });
      }
    }
  }
  return { rows, findings, next: nextWindow(rows) };
}

/** Parse report.md's open WARNING/SUGGESTION findings. */
function parseOpenFindings(reportText) {
  const out = [];
  if (!reportText) return out;
  for (const line of reportText.split("\n")) {
    if (!line.trim().startsWith("|")) continue;
    const cells = line.trim().split("|").slice(1, -1).map((c) => c.trim());
    if (cells.length < 4) continue;
    // The Findings table: | # | Severity | Finding | Disposition |
    // (column 1 is the row number, column 2 is severity). Skip header/separator.
    if (/^#/i.test(cells[0]) || /^\d+$/.test(cells[0]) === false) continue;
    if (/^[-: ]+$/.test(cells[0])) continue;
    const severity = cells[1].toUpperCase();
    const finding = cells[2];
    const disposition = (cells[3] || "").toLowerCase();
    if (SEVERITIES.has(severity) && disposition === "open") {
      out.push({ severity, finding });
    }
  }
  return out;
}

const WINDOWS_HEADER = `# Broken Windows

> Cross-change defect register. Advisory findings (WARNING/SUGGESTION) not
> fixed before archive. Auto-appended by \`skillgrid:qa\` at gate-render time.

| Window | Change | Severity | Finding | Waived By | Date | Status |
|--------|--------|----------|---------|-----------|------|--------|
`;

function main() {
  const args = process.argv.slice(2);
  const sub = args[0];
  if (!sub || (sub !== "check" && sub !== "append")) {
    console.error("Usage: windows.mjs <check|append [report.md]> [project-root]");
    process.exit(2);
  }
  const positional = args.slice(1).filter((a) => !a.startsWith("--"));
  const rootArg = positional[positional.length - 1] || ".";
  const root = resolve(rootArg);
  const wPath = windowsPath(root);
  const existing = read(wPath);

  if (sub === "check") {
    const scope = existing === null ? "UNSCOPED" : "COMPLETE";
    const { rows, findings, next } = checkWindows(existing);
    const result = {
      windows: rows.length,
      nextWindow: next,
      findings,
      ok: findings.length === 0,
      scope,
    };
    console.log(JSON.stringify(result, null, 2));
    process.exit(findings.length > 0 ? 1 : 0);
  }

  // append
  const reportArg = positional[0];
  if (!reportArg) {
    console.error("Usage: windows.mjs append <report.md> [project-root]");
    process.exit(2);
  }
  // The report path is relative to the project root (or absolute).
  const reportPath = resolve(root, reportArg);
  const reportText = read(reportPath);
  if (reportText === null) {
    console.error(`ERROR: report not found at ${reportPath}`);
    process.exit(2);
  }
  const open = parseOpenFindings(reportText);
  const { rows, next } = checkWindows(existing);
  // Assign the next sequential numbers.
  let n = rows.reduce((m, r) => Math.max(m, r.num), 0);
  const newRows = open.map((f) => {
    n += 1;
    return { window: `W${String(n).padStart(3, "0")}`, severity: f.severity, finding: f.finding };
  });
  // Append to the file (create it if absent).
  const date = new Date().toISOString().slice(0, 10);
  const change = "qa-auto";
  const lines = newRows
    .map((r) => `| ${r.window} | ${change} | ${r.severity} | ${r.finding} | qa-auto | ${date} | open |`)
    .join("\n");
  let out;
  if (existing === null) {
    out = WINDOWS_HEADER + lines + "\n";
  } else {
    out = existing.replace(/\n*$/, "\n") + lines + "\n";
  }
  mkdirSync(dirname(wPath), { recursive: true });
  writeFileSync(wPath, out);
  const result = {
    appended: newRows.length,
    nextWindow: newRows.length ? `W${String(n + 1).padStart(3, "0")}` : next,
    rows: newRows,
    ok: true,
    scope: "COMPLETE",
  };
  console.log(JSON.stringify(result, null, 2));
  process.exit(0);
}

main();
