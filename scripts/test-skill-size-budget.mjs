#!/usr/bin/env node
/**
 * test-skill-size-budget.mjs — fixture-driven tests for skill-size-budget.mjs.
 *
 * Usage: node scripts/test-skill-size-budget.mjs
 * Exit: 0 all pass, 1 any fail.
 */

import { spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, writeFileSync, rmSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const BUDGET = join(HERE, "skill-size-budget.mjs");

let passed = 0;
let failed = 0;
const failures = [];

function assert(name, cond, detail = "") {
  if (cond) {
    passed++;
    console.log(`  ok   ${name}`);
  } else {
    failed++;
    failures.push(name);
    console.log(`  FAIL ${name}${detail ? ` — ${detail}` : ""}`);
  }
}

function makeProject(skillFiles) {
  const root = mkdtempSync(join(tmpdir(), "sgsize-"));
  const skillsDir = join(root, ".agents", "skills");
  mkdirSync(skillsDir, { recursive: true });
  for (const [name, content] of Object.entries(skillFiles)) {
    const dir = join(skillsDir, name);
    mkdirSync(dir, { recursive: true });
    writeFileSync(join(dir, "SKILL.md"), content);
  }
  return { root, cleanup: () => rmSync(root, { recursive: true, force: true }) };
}

function runBudget(args) {
  const r = spawnSync("node", [BUDGET, ...args], { encoding: "utf8" });
  return { code: r.status ?? -1, stdout: r.stdout ?? "", stderr: r.stderr ?? "" };
}

function run() {
  console.log("skill-size-budget.mjs tests\n");

  // 1. No budget JSON → exit 0, "no budget file" message
  {
    const p = makeProject({ foo: "hello" });
    const r = runBudget(["check", p.root]);
    assert("no-budget: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    assert("no-budget: message", r.stdout.toLowerCase().includes("no budget"), r.stdout);
    p.cleanup();
  }

  // 2. Budget file, all under ceiling → exit 0
  {
    const p = makeProject({ foo: "a".repeat(1000), bar: "b".repeat(2000) });
    const budgetPath = join(p.root, "scripts", "skill-size-budget.json");
    mkdirSync(join(p.root, "scripts"), { recursive: true });
    writeFileSync(budgetPath, JSON.stringify({
      ceiling: { standard: 22000 },
      skills: { foo: 1000, bar: 2000 },
    }));
    const r = runBudget(["check", p.root]);
    assert("all-under: exit 0", r.code === 0, `got ${r.code}: ${r.stdout}`);
    assert("all-under: no overage", !r.stdout.includes("OVER"), r.stdout);
    p.cleanup();
  }

  // 3. Budget file, one over ceiling → exit 1, OVER line
  {
    const p = makeProject({ foo: "a".repeat(30000), bar: "b".repeat(1000) });
    const budgetPath = join(p.root, "scripts", "skill-size-budget.json");
    mkdirSync(join(p.root, "scripts"), { recursive: true });
    writeFileSync(budgetPath, JSON.stringify({
      ceiling: { standard: 22000 },
      skills: { foo: 1000, bar: 1000 },
    }));
    const r = runBudget(["check", p.root]);
    assert("over: exit 1", r.code === 1, `got ${r.code}: ${r.stdout}`);
    assert("over: OVER line for foo", r.stdout.includes("foo") && r.stdout.includes("OVER"), r.stdout);
    p.cleanup();
  }

  // 4. Budget file, skill not in budget → uses default ceiling
  {
    const p = makeProject({ unknown: "x".repeat(25000) });
    const budgetPath = join(p.root, "scripts", "skill-size-budget.json");
    mkdirSync(join(p.root, "scripts"), { recursive: true });
    writeFileSync(budgetPath, JSON.stringify({
      ceiling: { standard: 22000 },
      skills: {},
    }));
    const r = runBudget(["check", p.root]);
    assert("unknown-skill: exit 1", r.code === 1, `got ${r.code}: ${r.stdout}`);
    assert("unknown-skill: flagged", r.stdout.includes("unknown"), r.stdout);
    p.cleanup();
  }

  // 5. --write creates budget JSON
  {
    const p = makeProject({ foo: "a".repeat(5000), bar: "b".repeat(3000) });
    const r = runBudget(["check", "--write", p.root]);
    assert("write: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    const budgetPath = join(p.root, "scripts", "skill-size-budget.json");
    assert("write: file created", readFileSync(budgetPath, "utf8").includes("foo"));
    const budget = JSON.parse(readFileSync(budgetPath, "utf8"));
    assert("write: foo size recorded", budget.skills.foo === 5000, JSON.stringify(budget.skills));
    assert("write: bar size recorded", budget.skills.bar === 3000, JSON.stringify(budget.skills));
    p.cleanup();
  }

  // 6. --write with existing budget → updates sizes, keeps ceiling
  {
    const p = makeProject({ foo: "a".repeat(5000), bar: "b".repeat(3000) });
    const scriptsDir = join(p.root, "scripts");
    mkdirSync(scriptsDir, { recursive: true });
    const budgetPath = join(scriptsDir, "skill-size-budget.json");
    writeFileSync(budgetPath, JSON.stringify({
      ceiling: { standard: 22000, large: 38000 },
      skills: { foo: 4000 },
    }));
    const r = runBudget(["check", "--write", p.root]);
    assert("write-update: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    const budget = JSON.parse(readFileSync(budgetPath, "utf8"));
    assert("write-update: foo updated", budget.skills.foo === 5000, JSON.stringify(budget.skills));
    assert("write-update: bar added", budget.skills.bar === 3000, JSON.stringify(budget.skills));
    assert("write-update: ceiling preserved", budget.ceiling.large === 38000, JSON.stringify(budget.ceiling));
    p.cleanup();
  }

  // 7. Tiered ceilings: skill in large tier doesn't trigger standard ceiling
  {
    const p = makeProject({ big: "a".repeat(35000) });
    const budgetPath = join(p.root, "scripts", "skill-size-budget.json");
    mkdirSync(join(p.root, "scripts"), { recursive: true });
    writeFileSync(budgetPath, JSON.stringify({
      ceiling: { standard: 22000, large: 40000 },
      tiers: { big: "large" },
      skills: { big: 30000 },
    }));
    const r = runBudget(["check", p.root]);
    assert("tiered: exit 0 (large tier)", r.code === 0, `got ${r.code}: ${r.stdout}`);
    p.cleanup();
  }

  // 8. Tiered ceilings: skill in large tier still over large ceiling
  {
    const p = makeProject({ big: "a".repeat(45000) });
    const budgetPath = join(p.root, "scripts", "skill-size-budget.json");
    mkdirSync(join(p.root, "scripts"), { recursive: true });
    writeFileSync(budgetPath, JSON.stringify({
      ceiling: { standard: 22000, large: 40000 },
      tiers: { big: "large" },
      skills: { big: 30000 },
    }));
    const r = runBudget(["check", p.root]);
    assert("tiered-over: exit 1", r.code === 1, `got ${r.code}: ${r.stdout}`);
    assert("tiered-over: OVER for big", r.stdout.includes("big") && r.stdout.includes("OVER"), r.stdout);
    p.cleanup();
  }

  // Summary
  console.log(`\n${passed + failed} tests: ${passed} passed, ${failed} failed`);
  if (failures.length) {
    console.log("Failures:");
    for (const f of failures) console.log(`  - ${f}`);
  }
  process.exit(failed > 0 ? 1 : 0);
}

run();
