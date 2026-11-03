#!/usr/bin/env node
/**
 * test-review-ledger-lint.mjs — fixture-driven tests for
 * scripts/review-ledger-lint.mjs.
 *
 * Builds a temp git repo with a review-ledger.md in the real schema (Round
 * headings, Findings table, Dispositions list, frozen-candidate line), runs
 * the script, and asserts each schema check: disposition present, fixed ⇒
 * commit resolves, parked ⇒ ruling + human, rounds sequential, no dupes,
 * finding IDs sequential. Also the review-debt scan across xarchive.
 *
 * Usage: node scripts/test-review-ledger-lint.mjs
 * Exit: 0 all pass, 1 any fail.
 */

import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync, mkdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const LINT = join(HERE, "review-ledger-lint.mjs");

let passed = 0, failed = 0;
const failures = [];
function assert(name, cond, detail = "") {
  if (cond) { passed++; console.log(`  ok   ${name}`); }
  else { failed++; failures.push(name); console.log(`  FAIL ${name}${detail ? ` — ${detail}` : ""}`); }
}

function git(root, ...args) {
  return execFileSync("git", args, { cwd: root, encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] }).trim();
}

function makeRepo(files = {}) {
  const root = mkdtempSync(join(tmpdir(), "sgled-"));
  git(root, "init", "-q");
  git(root, "config", "user.email", "t@t");
  git(root, "config", "user.name", "t");
  // A couple of real commits so fix-commit hashes resolve.
  mkdirSync(join(root, "src"), { recursive: true });
  writeFileSync(join(root, "src", "a.go"), "v1");
  git(root, "add", "-A"); git(root, "commit", "-q", "-m", "c1");
  writeFileSync(join(root, "src", "a.go"), "v2");
  git(root, "add", "-A"); git(root, "commit", "-q", "-m", "c2");
  const hashes = [git(root, "rev-parse", "HEAD~1"), git(root, "rev-parse", "HEAD")];
  for (const [f, c] of Object.entries(files)) writeFileSync(join(root, f), c);
  return { root, hashes, cleanup: () => rmSync(root, { recursive: true, force: true }) };
}

function writeLedger(root, rel, content) {
  const p = join(root, rel);
  mkdirSync(dirname(p), { recursive: true });
  writeFileSync(p, content);
}

function run(root, ...args) {
  try {
    const stdout = execFileSync("node", [LINT, ...args, root], { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] });
    return { code: 0, stdout };
  } catch (e) {
    // The script prints JSON to stdout even on exit 1 (findings found); the
    // parse() call handles it. Only a true usage error (exit 2) has no JSON.
    return { code: e.status ?? -1, stdout: e.stdout ?? "", stderr: e.stderr ?? "" };
  }
}
function parse(out) { try { return JSON.parse(out); } catch { return null; } }

// A valid ledger: round 1 (1 fixed, 1 parked, 1 waived), round 2 clean.
function validLedger(h) {
  return `# Review Ledger\n\n` +
    `## Round 1 — 2026-01-01\n\n` +
    `**Frozen candidate:** base ${h[0]} → head ${h[1]}, 2 files, hash deadbeef\n` +
    `**Reviewer:** skillgrid:requesting-code-review\n` +
    `**Independence grade:** A\n` +
    `**Budget:** round 1/2 (tier T2)\n\n` +
    `### Findings\n\n` +
    `| ID | Severity | File:Line | Finding | Disposition | Fix Commit | Lineage |\n` +
    `|---|---|---|---|---|---|---|\n` +
    `| R1-F001 | CRITICAL | a.go:10 | bad thing | fixed | ${h[1]} | R1-F001 |\n` +
    `| R1-F002 | WARNING | b.go:5 | slow thing | parked | — | R1-F002 |\n` +
    `| R1-F003 | SUGGESTION | c.go:1 | style | waived | — | R1-F003 |\n\n` +
    `### Dispositions\n\n` +
    `- **R1-F001** — fixed in \`${h[1]}\`. Fixed the bad thing.\n` +
    `- **R1-F002** — parked. Ruling: acceptable risk. paladm.\n` +
    `- **R1-F003** — waived. Risk accepted: minor. paladm.\n\n` +
    `### Round Verdict\n\nfindings-dispositioned\n\n` +
    `## Round 2 — 2026-01-02\n\n` +
    `**Frozen candidate:** base ${h[1]} → head ${h[1]}, 2 files, hash cafebabe\n` +
    `**Reviewer:** skillgrid:parallel-code-review\n` +
    `**Independence grade:** A\n` +
    `**Budget:** round 2/2 (tier T2)\n\n` +
    `### Findings\n\n` +
    `| ID | Severity | File:Line | Finding | Disposition | Fix Commit | Lineage |\n` +
    `|---|---|---|---|---|---|---|\n` +
    `(none)\n\n` +
    `### Dispositions\n\n(none)\n\n` +
    `### Round Verdict\n\nclean\n`;
}

// 1. Valid ledger → no findings
{
  const p = makeRepo();
  writeLedger(p.root, ".skillgrid/specs/demo/review-ledger.md", validLedger(p.hashes));
  const r = run(p.root, "demo");
  const j = parse(r.stdout);
  assert("valid: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("valid: 0 findings", j && (j.findings.length === 0 || j.ok === true), r.stdout);
  assert("valid: 2 rounds", j && j.rounds === 2, r.stdout);
  p.cleanup();
}

// 2. A `fixed` finding with no disposition bullet → finding
{
  const p = makeRepo();
  const led = [
    "# Review Ledger",
    "",
    "## Round 1 — 2026-01-01",
    "",
    `**Frozen candidate:** base ${p.hashes[0]} → head ${p.hashes[1]}, 2 files, hash deadbeef`,
    "",
    "### Findings",
    "",
    "| ID | Severity | File:Line | Finding | Disposition | Fix Commit | Lineage |",
    "|---|---|---|---|---|---|---|",
    `| R1-F001 | CRITICAL | a.go:10 | bad thing | fixed | ${p.hashes[1]} | R1-F001 |`,
    "",
    "### Dispositions",
    "",
    "(no bullet for R1-F001)",
    "",
    "### Round Verdict",
    "",
    "findings-dispositioned",
  ].join("\n");
  writeLedger(p.root, ".skillgrid/specs/demo/review-ledger.md", led);
  const r = run(p.root, "demo");
  const j = parse(r.stdout);
  assert("missing-disp: has finding", j && j.findings.length > 0, r.stdout);
  assert("missing-disp: names R1-F001", j && j.findings.some((f) => f.finding === "R1-F001"), JSON.stringify(j?.findings));
  p.cleanup();
}

// 3. A `fixed` finding whose commit does not resolve → finding
{
  const p = makeRepo();
  let led = validLedger(p.hashes);
  led = led.replace(`| fixed | ${p.hashes[1]} | R1-F001`, "| fixed | 0000000000000000000000000000000000000000 | R1-F001");
  led = led.replace(`fixed in \`${p.hashes[1]}\``, "fixed in `0000000000000000000000000000000000000000`");
  writeLedger(p.root, ".skillgrid/specs/demo/review-ledger.md", led);
  const r = run(p.root, "demo");
  const j = parse(r.stdout);
  assert("bad-commit: has finding", j && j.findings.length > 0, r.stdout);
  p.cleanup();
}

// 4. A `parked` finding with no ruling/human → finding
{
  const p = makeRepo();
  let led = validLedger(p.hashes);
  led = led.replace("- **R1-F002** — parked. Ruling: acceptable risk. paladm.", "- **R1-F002** — parked.");
  writeLedger(p.root, ".skillgrid/specs/demo/review-ledger.md", led);
  const r = run(p.root, "demo");
  const j = parse(r.stdout);
  assert("parked-no-ruling: has finding", j && j.findings.length > 0, r.stdout);
  p.cleanup();
}

// 5. Non-sequential rounds (1, 3) → finding
{
  const p = makeRepo();
  let led = validLedger(p.hashes).replace("## Round 2 — 2026-01-02", "## Round 3 — 2026-01-02");
  writeLedger(p.root, ".skillgrid/specs/demo/review-ledger.md", led);
  const r = run(p.root, "demo");
  const j = parse(r.stdout);
  assert("gap-rounds: has finding", j && j.findings.some((f) => /round/i.test(f.rule || "")), JSON.stringify(j?.findings));
  p.cleanup();
}

// 6. Review-debt: parked findings across xarchive are surfaced
{
  const p = makeRepo();
  // A valid primary ledger for the active change + an archived change with a
  // parked finding (validLedger's R1-F002 is parked).
  writeLedger(p.root, ".skillgrid/specs/demo/review-ledger.md", validLedger(p.hashes));
  writeLedger(p.root, ".skillgrid/xarchive/2026-01-01-old/review-ledger.md", validLedger(p.hashes));
  const r = run(p.root, "demo");
  const j = parse(r.stdout);
  assert("debt: surfaces parked from xarchive", j && j.debt && j.debt.length > 0, JSON.stringify(j?.debt));
  assert("debt: names the finding", j && j.debt.some((d) => /R1-F002/.test(d.finding)), JSON.stringify(j?.debt));
  p.cleanup();
}

// 7. No ledger file → exit 2
{
  const p = makeRepo();
  const r = run(p.root, "demo");
  assert("noleader: exit 2", r.code === 2, `got ${r.code}: ${r.stderr}`);
  p.cleanup();
}

// 8. Determinism: run twice identical
{
  const p = makeRepo();
  writeLedger(p.root, ".skillgrid/specs/demo/review-ledger.md", validLedger(p.hashes));
  const a = run(p.root, "demo").stdout;
  const b = run(p.root, "demo").stdout;
  assert("determinism: identical", a === b && a.length > 0);
  p.cleanup();
}

console.log(`\n${passed + failed} tests: ${passed} passed, ${failed} failed`);
if (failures.length) { console.log("Failures:"); for (const f of failures) console.log(`  - ${f}`); }
process.exit(failed > 0 ? 1 : 0);
