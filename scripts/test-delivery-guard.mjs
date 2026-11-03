#!/usr/bin/env node
/**
 * test-delivery-guard.mjs — fixture-driven tests for scripts/delivery-guard.mjs.
 *
 * The delivery guard is four plain-text lines in tasks.md that three skills
 * (subagent-execution, ship, qa-gate) each regex independently. A reworded
 * line silently breaks the guard. This script is the single parser: it reads
 * the four lines and emits a structured decision the skills reuse.
 *
 * The four lines:
 *   Decision needed before apply: <Yes|No>
 *   Chained PRs recommended: <Yes|No>
 *   Chain strategy: <ref|name>
 *   400-line budget risk: <Low|Medium|High>
 *
 * Usage: node scripts/test-delivery-guard.mjs
 * Exit: 0 all pass, 1 any fail.
 */

import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync, mkdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const GUARD = join(HERE, "delivery-guard.mjs");

let passed = 0, failed = 0;
const failures = [];
function assert(name, cond, detail = "") {
  if (cond) { passed++; console.log(`  ok   ${name}`); }
  else { failed++; failures.push(name); console.log(`  FAIL ${name}${detail ? ` — ${detail}` : ""}`); }
}

function makeSpec(files) {
  const root = mkdtempSync(join(tmpdir(), "sgdg-"));
  const dir = join(root, ".skillgrid", "specs", "demo");
  mkdirSync(dir, { recursive: true });
  for (const [f, c] of Object.entries(files)) writeFileSync(join(dir, f), c);
  return { root, cleanup: () => rmSync(root, { recursive: true, force: true }) };
}

function run(root, slug) {
  try {
    const stdout = execFileSync("node", [GUARD, slug, root], { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] });
    return { code: 0, stdout };
  } catch (e) {
    return { code: e.status ?? -1, stdout: e.stdout ?? "", stderr: e.stderr ?? "" };
  }
}
function parse(out) { try { return JSON.parse(out); } catch { return null; } }

// A clean tasks.md: no decision needed, no chained PRs, low risk.
const TASKS_CLEAN = `# Tasks\n\nSome prose.\n\nDecision needed before apply: No\nChained PRs recommended: No\nChain strategy: none\n400-line budget risk: Low\n`;

// A risky tasks.md: decision needed, chained PRs yes, high risk.
const TASKS_RISKY = `# Tasks\n\n| 400-line budget risk | High |\n| Chained PRs recommended | Yes |\n\nDecision needed before apply: Yes — needs a human on the DBA sign-off\nChained PRs recommended: Yes\nChain strategy: size-exception\n400-line budget risk: High\n`;

// 1. Clean guard → not blocked, all fields parsed
{
  const p = makeSpec({ "tasks.md": TASKS_CLEAN });
  const r = run(p.root, "demo");
  const j = parse(r.stdout);
  assert("clean: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("clean: decisionNeeded false", j && j.decisionNeeded === false, r.stdout);
  assert("clean: chainedPRs false", j && j.chainedPRs === false, r.stdout);
  assert("clean: chainStrategy none", j && j.chainStrategy === "none", r.stdout);
  assert("clean: budgetRisk Low", j && j.budgetRisk === "Low", r.stdout);
  assert("clean: not blocked", j && j.blocked === null, r.stdout);
  p.cleanup();
}

// 2. Risky guard: decision needed → blocked with the reason
{
  const p = makeSpec({ "tasks.md": TASKS_RISKY });
  const r = run(p.root, "demo");
  const j = parse(r.stdout);
  assert("risky: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("risky: decisionNeeded true", j && j.decisionNeeded === true, r.stdout);
  assert("risky: chainedPRs true", j && j.chainedPRs === true, r.stdout);
  assert("risky: chainStrategy size-exception", j && j.chainStrategy === "size-exception", r.stdout);
  assert("risky: budgetRisk High", j && j.budgetRisk === "High", r.stdout);
  assert("risky: blocked (reason named)", j && typeof j.blocked === "string" && j.blocked.length > 0, r.stdout);
  p.cleanup();
}

// 3. High risk + No chained PRs → inconsistent (the contract says High ⇒ Yes)
{
  const tasks = TASKS_RISKY.replace("Chained PRs recommended: Yes", "Chained PRs recommended: No");
  const p = makeSpec({ "tasks.md": tasks });
  const r = run(p.root, "demo");
  const j = parse(r.stdout);
  // High risk with chainedPRs No is a contract violation → flagged.
  assert("inconsistent: has a consistency issue", j && (j.consistency === "inconsistent" || (j.blocked || "").includes("risk")), JSON.stringify(j));
  p.cleanup();
}

// 4. Missing guard lines → not blocked, fields null, scope UNSCOPED
{
  const p = makeSpec({ "tasks.md": "# Tasks\n\nJust prose, no guard lines.\n" });
  const r = run(p.root, "demo");
  const j = parse(r.stdout);
  assert("missing: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("missing: decisionNeeded null", j && j.decisionNeeded === null, r.stdout);
  assert("missing: not blocked", j && j.blocked === null, r.stdout);
  p.cleanup();
}

// 5. No tasks.md → exit 2
{
  const p = makeSpec({});
  const r = run(p.root, "demo");
  assert("notasks: exit 2", r.code === 2, `got ${r.code}: ${r.stderr}`);
  p.cleanup();
}

// 6. Determinism: run twice identical
{
  const p = makeSpec({ "tasks.md": TASKS_RISKY });
  const a = run(p.root, "demo").stdout;
  const b = run(p.root, "demo").stdout;
  assert("determinism: identical", a === b && a.length > 0);
  p.cleanup();
}

console.log(`\n${passed + failed} tests: ${passed} passed, ${failed} failed`);
if (failures.length) { console.log("Failures:"); for (const f of failures) console.log(`  - ${f}`); }
process.exit(failed > 0 ? 1 : 0);
