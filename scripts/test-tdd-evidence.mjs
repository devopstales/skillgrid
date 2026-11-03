#!/usr/bin/env node
/**
 * test-tdd-evidence.mjs — fixture-driven tests for scripts/tdd-evidence.mjs.
 *
 * Builds a temp git repo with real commits + a report.md TDD Evidence table,
 * runs the script, and asserts the verdict per row (OK / MISSING_RED /
 * MISSING_GREEN / STALE / N-A) and the summary line. The git checks are real:
 * `git cat-file -t <hash>` against actual commits.
 *
 * Usage: node scripts/test-tdd-evidence.mjs
 * Exit: 0 all pass, 1 any fail.
 */

import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync, readFileSync, mkdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname, isAbsolute } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const TDD = join(HERE, "tdd-evidence.mjs");

let passed = 0, failed = 0;
const failures = [];
function assert(name, cond, detail = "") {
  if (cond) { passed++; console.log(`  ok   ${name}`); }
  else { failed++; failures.push(name); console.log(`  FAIL ${name}${detail ? ` — ${detail}` : ""}`); }
}

function git(root, ...args) {
  return execFileSync("git", args, { cwd: root, encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] }).trim();
}

function makeRepo(files, commits = []) {
  const root = mkdtempSync(join(tmpdir(), "sgtdd-"));
  git(root, "init", "-q");
  git(root, "config", "user.email", "t@t");
  git(root, "config", "user.name", "t");
  const specDir = join(root, ".skillgrid", "specs", "demo-change");
  mkdirSync(specDir, { recursive: true });
  for (const [f, c] of Object.entries(files)) {
    const p = isAbsolute(f) ? f : join(root, f);
    mkdirSync(dirname(p), { recursive: true });
    writeFileSync(p, c);
  }
  git(root, "add", "-A");
  git(root, "commit", "-q", "-m", "initial");
  const hashes = [];
  for (const m of commits) {
    git(root, "commit", "-q", "--allow-empty", "-m", m);
    hashes.push(git(root, "rev-parse", "HEAD"));
  }
  return { root, hashes, cleanup: () => rmSync(root, { recursive: true, force: true }) };
}

function run(root, ...args) {
  try {
    const stdout = execFileSync("node", [TDD, ...args, root], { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] });
    return { code: 0, stdout };
  } catch (e) {
    return { code: e.status ?? -1, stdout: e.stdout ?? "", stderr: e.stderr ?? "" };
  }
}
function parse(out) { try { return JSON.parse(out); } catch { return null; } }

const REPORT_OK = `# Report

## TDD Evidence

| Ticket | RED commit | GREEN commit | Test file |
|--------|-----------|-------------|-----------|
| T1 | {RED} | {GREEN} | test_a.go |
| T2 | {RED} | {GREEN} | test_b.go |
`;

// 1. All commits resolve + test files exist → all OK
{
  const p = makeRepo({ "report.md": REPORT_OK, "test_a.go": "x", "test_b.go": "x" }, ["red1", "green1", "red2", "green2"]);
  const report = REPORT_OK.replace("{RED}", p.hashes[0]).replace("{GREEN}", p.hashes[1]).replace("{RED}", p.hashes[2]).replace("{GREEN}", p.hashes[3]);
  writeFileSync(join(p.root, ".skillgrid", "specs", "demo-change", "report.md"), report);
  git(p.root, "add", "-A"); git(p.root, "commit", "-q", "-m", "report");
  const r = run(p.root, "demo-change");
  const j = parse(r.stdout);
  assert("all-ok: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("all-ok: 2 rows", j?.rows?.length === 2, r.stdout);
  assert("all-ok: both OK", j?.rows?.every((row) => row.verdict === "OK"), JSON.stringify(j?.rows));
  assert("all-ok: summary 2/2", j?.summary === "2/2 tasks have complete TDD evidence", r.stdout);
  p.cleanup();
}

// 2. A RED commit that does not exist → MISSING_RED
{
  const p = makeRepo({ "test_a.go": "x" }, ["g"]);
  const report = `## TDD Evidence\n\n| Ticket | RED commit | GREEN commit | Test file |\n|---|---|---|---|\n| T1 | deadbeef000000000000000000000000000000000 | ${p.hashes[0]} | test_a.go |\n`;
  writeFileSync(join(p.root, ".skillgrid", "specs", "demo-change", "report.md"), report);
  git(p.root, "add", "-A"); git(p.root, "commit", "-q", "-m", "report");
  const r = run(p.root, "demo-change");
  const j = parse(r.stdout);
  assert("missing-red: row verdict MISSING_RED", j?.rows?.[0]?.verdict === "MISSING_RED", JSON.stringify(j?.rows));
  assert("missing-red: summary 0/1", j?.summary === "0/1 tasks have complete TDD evidence", r.stdout);
  p.cleanup();
}

// 3. Test file absent on disk → STALE
{
  const p = makeRepo({ "report.md": "" }, ["r", "g"]);
  const report = `## TDD Evidence\n\n| Ticket | RED commit | GREEN commit | Test file |\n|---|---|---|---|\n| T1 | ${p.hashes[0]} | ${p.hashes[1]} | missing_test.go |\n`;
  writeFileSync(join(p.root, ".skillgrid", "specs", "demo-change", "report.md"), report);
  git(p.root, "add", "-A"); git(p.root, "commit", "-q", "-m", "report");
  const r = run(p.root, "demo-change");
  const j = parse(r.stdout);
  assert("stale: row verdict STALE (test file missing)", j?.rows?.[0]?.verdict === "STALE", JSON.stringify(j?.rows));
  p.cleanup();
}

// 4. N/A RED (Python, no test framework) → OK (the cell is absent but
//    acceptable; the ticket still has valid GREEN + test evidence)
{
  const p = makeRepo({ "report.md": "" }, ["g"]);
  const report = `## TDD Evidence\n\n| Ticket | RED commit | GREEN commit | Test file |\n|---|---|---|---|\n| T1 | N/A (Python, no test framework) | ${p.hashes[0]} | provider.py |\n`;
  writeFileSync(join(p.root, ".skillgrid", "specs", "demo-change", "report.md"), report);
  writeFileSync(join(p.root, "provider.py"), "x");
  git(p.root, "add", "-A"); git(p.root, "commit", "-q", "-m", "report");
  const r = run(p.root, "demo-change");
  const j = parse(r.stdout);
  assert("na-red: row verdict OK (N/A cell acceptable)", j?.rows?.[0]?.verdict === "OK", JSON.stringify(j?.rows));
  assert("na-red: summary 1/1", j?.summary === "1/1 tasks have complete TDD evidence", r.stdout);
  p.cleanup();
}

// 5. No TDD Evidence section → exit 2 (or scope UNSCOPED)
{
  const p = makeRepo({ "report.md": "# Report\n\n## Other\n\nno tdd here\n" }, []);
  const r = run(p.root, "demo-change");
  const j = parse(r.stdout);
  assert("no-section: exit 2 or 0 with 0 rows", (r.code === 2 || (r.code === 0 && (j?.rows?.length === 0 || j?.rows === null))), `code ${r.code}: ${r.stdout}`);
  p.cleanup();
}

// 5b. --run-test re-runs the named test; non-zero → MISSING_GREEN, ENOENT → FAILED
{
  const p = makeRepo({ "t.go": "x" }, ["r", "g"]);
  const report = `## TDD Evidence\n\n| Ticket | RED commit | GREEN commit | Test file |\n|---|---|---|---|\n| T1 | ${p.hashes[0]} | ${p.hashes[1]} | t.go |\n`;
  writeFileSync(join(p.root, ".skillgrid", "specs", "demo-change", "report.md"), report);
  git(p.root, "add", "-A"); git(p.root, "commit", "-q", "-m", "report");

  // PASS: a command that exits 0.
  let r = run(p.root, "demo-change", "--run-test=exit 0");
  let j = parse(r.stdout);
  assert("retest-pass: verdict OK", j?.rows?.[0]?.verdict === "OK", JSON.stringify(j?.rows));
  assert("retest-pass: retest PASS", j?.rows?.[0]?.retest === "PASS", JSON.stringify(j?.rows));

  // FAIL: a command that exits 1 → the GREEN cell no longer passes.
  r = run(p.root, "demo-change", "--run-test=exit 1");
  j = parse(r.stdout);
  assert("retest-fail: verdict MISSING_GREEN", j?.rows?.[0]?.verdict === "MISSING_GREEN", JSON.stringify(j?.rows));
  assert("retest-fail: retest FAIL", j?.rows?.[0]?.retest === "FAIL", JSON.stringify(j?.rows));

  // ERROR: a command that cannot execute (ENOENT) → FAILED (infrastructure).
  r = run(p.root, "demo-change", "--run-test=/nonexistent/binary-xyz");
  j = parse(r.stdout);
  assert("retest-error: verdict FAILED", j?.rows?.[0]?.verdict === "FAILED", JSON.stringify(j?.rows));
  assert("retest-error: retest ERROR", j?.rows?.[0]?.retest === "ERROR", JSON.stringify(j?.rows));
  p.cleanup();
}

// 6. Determinism: run twice identical
{
  const p = makeRepo({ "report.md": "" }, ["r", "g"]);
  const report = `## TDD Evidence\n\n| Ticket | RED commit | GREEN commit | Test file |\n|---|---|---|---|\n| T1 | ${p.hashes[0]} | ${p.hashes[1]} | t.go |\n`;
  writeFileSync(join(p.root, ".skillgrid", "specs", "demo-change", "report.md"), report);
  writeFileSync(join(p.root, "t.go"), "x");
  git(p.root, "add", "-A"); git(p.root, "commit", "-q", "-m", "report");
  const a = run(p.root, "demo-change").stdout;
  const b = run(p.root, "demo-change").stdout;
  assert("determinism: identical", a === b && a.length > 0);
  p.cleanup();
}

function readReport(root) {
  return readFileSync(join(root, ".skillgrid", "specs", "demo-change", "report.md"), "utf8");
}

console.log(`\n${passed + failed} tests: ${passed} passed, ${failed} failed`);
if (failures.length) { console.log("Failures:"); for (const f of failures) console.log(`  - ${f}`); }
process.exit(failed > 0 ? 1 : 0);
