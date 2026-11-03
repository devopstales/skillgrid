#!/usr/bin/env node
/**
 * test-windows.mjs — fixture-driven tests for scripts/windows.mjs.
 *
 * The broken-windows ledger (WINDOWS.md) is a cross-change defect register.
 * `qa` appends open WARNING/SUGGESTION findings with the next sequential
 * W<NNN>; `check` validates monotonic IDs, no dupes, and valid columns.
 *
 * Usage: node scripts/test-windows.mjs
 * Exit: 0 all pass, 1 any fail.
 */

import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync, mkdirSync, rmSync, readFileSync, existsSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const WIN = join(HERE, "windows.mjs");

let passed = 0, failed = 0;
const failures = [];
function assert(name, cond, detail = "") {
  if (cond) { passed++; console.log(`  ok   ${name}`); }
  else { failed++; failures.push(name); console.log(`  FAIL ${name}${detail ? ` — ${detail}` : ""}`); }
}

function makeRoot(files = {}) {
  const root = mkdtempSync(join(tmpdir(), "sgwin-"));
  for (const [f, c] of Object.entries(files)) {
    const p = join(root, f);
    mkdirSync(dirname(p), { recursive: true });
    writeFileSync(p, c);
  }
  return { root, cleanup: () => rmSync(root, { recursive: true, force: true }) };
}

function run(root, ...args) {
  try {
    const stdout = execFileSync("node", [WIN, ...args, root], { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] });
    return { code: 0, stdout };
  } catch (e) {
    return { code: e.status ?? -1, stdout: e.stdout ?? "", stderr: e.stderr ?? "" };
  }
}
function parse(out) { try { return JSON.parse(out); } catch { return null; } }

// A WINDOWS.md with W001-W003 (the last number is W003).
const WINDOWS = `# Broken Windows\n\n| Window | Change | Severity | Finding | Waived By | Date | Status |\n|--------|--------|----------|---------|-----------|------|--------|\n| W001 | ch1 | WARNING | finding one | qa-auto | 2026-01-01 | open |\n| W002 | ch1 | SUGGESTION | finding two | qa-auto | 2026-01-01 | closed |\n| W003 | ch2 | WARNING | finding three | qa-auto | 2026-01-02 | open |\n`;

// A report.md with open WARNING/SUGGESTION findings (the "Findings" table).
const REPORT = `# QA Report\n\n## Findings\n\n| # | Severity | Finding | Disposition |\n|---|----------|---------|-------------|\n| 1 | WARNING | a real gap | open |\n| 2 | SUGGESTION | a suggestion | open |\n| 3 | CRITICAL | a critical (not a window) | fixed |\n`;

// 1. check on a valid WINDOWS.md → exit 0, no findings
{
  const p = makeRoot({ ".skillgrid/WINDOWS.md": WINDOWS });
  const r = run(p.root, "check");
  const j = parse(r.stdout);
  assert("check-valid: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("check-valid: 0 findings", j && j.findings.length === 0, r.stdout);
  assert("check-valid: next W004", j && j.nextWindow === "W004", r.stdout);
  p.cleanup();
}

// 2. append reads report.md's open rows, appends W004 + W005
{
  const p = makeRoot({ ".skillgrid/WINDOWS.md": WINDOWS, "report.md": REPORT });
  const r = run(p.root, "append", "report.md");
  const j = parse(r.stdout);
  assert("append: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("append: appended 2", j && j.appended === 2, r.stdout);
  assert("append: first W004", j && j.rows && j.rows[0] && j.rows[0].window === "W004", JSON.stringify(j?.rows));
  assert("append: second W005", j && j.rows && j.rows[1] && j.rows[1].window === "W005", JSON.stringify(j?.rows));
  // The file now has W004 + W005 appended.
  const content = readFileSync(join(p.root, ".skillgrid", "WINDOWS.md"), "utf8");
  assert("append: file has W004", content.includes("| W004 |"), content);
  assert("append: file has W005", content.includes("| W005 |"), content);
  assert("append: file has no W006", !content.includes("| W006 |"), content);
  p.cleanup();
}

// 3. CRITICAL findings are NOT windows (only WARNING/SUGGESTION)
{
  const p = makeRoot({ ".skillgrid/WINDOWS.md": WINDOWS, "report.md": REPORT });
  const r = run(p.root, "append", "report.md");
  const j = parse(r.stdout);
  // Only the WARNING + SUGGESTION rows (2), not the CRITICAL.
  assert("append: only WARNING/SUGGESTION (2 rows)", j && j.appended === 2, r.stdout);
  p.cleanup();
}

// 4. check detects a duplicate W id
{
  const dup = WINDOWS.replace("| W003 |", "| W002 |"); // W002 appears twice
  const p = makeRoot({ ".skillgrid/WINDOWS.md": dup });
  const r = run(p.root, "check");
  const j = parse(r.stdout);
  assert("check-dup: has finding", j && j.findings.length > 0, r.stdout);
  assert("check-dup: names the dup", j && j.findings.some((f) => /W002/.test(f.detail || f.id || "")), JSON.stringify(j?.findings));
  p.cleanup();
}

// 5. check detects a non-monotonic gap (W001 then W003, no W002)
{
  const gap = `# Broken Windows\n\n| Window | Change | Severity | Finding | Waived By | Date | Status |\n|--------|--------|----------|---------|-----------|------|--------|\n| W001 | ch1 | WARNING | one | qa-auto | 2026-01-01 | open |\n| W003 | ch1 | WARNING | three | qa-auto | 2026-01-01 | open |\n`;
  const p = makeRoot({ ".skillgrid/WINDOWS.md": gap });
  const r = run(p.root, "check");
  const j = parse(r.stdout);
  assert("check-gap: has finding", j && j.findings.some((f) => /gap/i.test(f.rule || f.detail || "")), JSON.stringify(j?.findings));
  p.cleanup();
}

// 6. check on a bad severity (not WARNING/SUGGESTION) → finding
{
  const badSev = WINDOWS.replace("| W001 | ch1 | WARNING |", "| W001 | ch1 | CRITICAL |");
  const p = makeRoot({ ".skillgrid/WINDOWS.md": badSev });
  const r = run(p.root, "check");
  const j = parse(r.stdout);
  assert("check-badsev: has finding", j && j.findings.some((f) => /severity/i.test(f.rule || f.detail || "")), JSON.stringify(j?.findings));
  p.cleanup();
}

// 7. No WINDOWS.md → check exit 0 (empty ledger, next is W001), append creates it
{
  const p = makeRoot({ "report.md": REPORT });
  const r = run(p.root, "check");
  const j = parse(r.stdout);
  assert("no-file: check exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("no-file: next W001", j && j.nextWindow === "W001", r.stdout);
  // Append creates the file.
  const ra = run(p.root, "append", "report.md");
  assert("no-file: append exit 0", ra.code === 0, ra.stderr);
  assert("no-file: file created", existsSync(join(p.root, ".skillgrid", "WINDOWS.md")), "WINDOWS.md not created");
  p.cleanup();
}

// 8. Determinism: check run twice identical
{
  const p = makeRoot({ ".skillgrid/WINDOWS.md": WINDOWS });
  const a = run(p.root, "check").stdout;
  const b = run(p.root, "check").stdout;
  assert("determinism: identical", a === b && a.length > 0);
  p.cleanup();
}

console.log(`\n${passed + failed} tests: ${passed} passed, ${failed} failed`);
if (failures.length) { console.log("Failures:"); for (const f of failures) console.log(`  - ${f}`); }
process.exit(failed > 0 ? 1 : 0);
