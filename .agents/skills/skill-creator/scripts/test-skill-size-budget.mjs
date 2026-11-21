#!/usr/bin/env node
// test-skill-size-budget.mjs — fixture tests for skill-size-budget.mjs.
//
// Usage: node test-skill-size-budget.mjs
// Exit: 0 all pass, 1 any fail.
//
// The script under test is the one shipped beside this test (the source of
// truth for the ratchet); it reads its budget from
// <root>/.agents/skills/_shared/skill-size-budget.json, so each case builds a
// throwaway skill tree under mkdtemp.

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

// makeProject({ skills: { name: "content" }, budget: {json} }) — a throwaway
// skill tree: .agents/skills/<name>/SKILL.md per skill, optional _shared budget.
function makeProject({ skills = {}, budget = null }) {
  const root = mkdtempSync(join(tmpdir(), "sgsize-"));
  const skillsDir = join(root, ".agents", "skills");
  for (const [name, content] of Object.entries(skills)) {
    const dir = join(skillsDir, name);
    mkdirSync(dir, { recursive: true });
    writeFileSync(join(dir, "SKILL.md"), content);
  }
  if (budget) {
    const shared = join(skillsDir, "_shared");
    mkdirSync(shared, { recursive: true });
    writeFileSync(join(shared, "skill-size-budget.json"), JSON.stringify(budget, null, 2));
  }
  return { root, cleanup: () => rmSync(root, { recursive: true, force: true }) };
}

function runBudget(args) {
  const r = spawnSync("node", [BUDGET, ...args], { encoding: "utf8" });
  return { code: r.status ?? -1, stdout: r.stdout ?? "", stderr: r.stderr ?? "" };
}

// --- 1. no budget file -> exit 0, lists skills ---------------------------------
{
  const p = makeProject({ skills: { foo: "hello" } });
  const r = runBudget(["check", p.root]);
  assert("no-budget: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("no-budget: lists the skill", r.stdout.includes("foo: 5 bytes"), r.stdout);
  p.cleanup();
}

// --- 2. within ceiling -> exit 0 ----------------------------------------------
{
  const p = makeProject({
    skills: { ok: "x".repeat(1000) },
    budget: {
      ceiling: { standard: 22000, large: 38000, xl: 40000 },
      tiers: {},
      skills: { ok: 1000 },
    },
  });
  const r = runBudget(["check", p.root]);
  assert("within-ceiling: exit 0", r.code === 0, `got ${r.code}: ${r.stdout}${r.stderr}`);
  assert("within-ceiling: reports all within", r.stdout.includes("within ceiling"), r.stdout);
  p.cleanup();
}

// --- 3. over standard ceiling -> exit 1, names the skill -----------------------
{
  const over = "x".repeat(22001);
  const p = makeProject({
    skills: { big: over },
    budget: {
      ceiling: { standard: 22000, large: 38000, xl: 40000 },
      tiers: {},
      skills: { big: 1000 },
    },
  });
  const r = runBudget(["check", p.root]);
  assert("over-standard: exit 1", r.code === 1, `got ${r.code}: ${r.stdout}${r.stderr}`);
  assert("over-standard: names the skill", r.stdout.includes("| big |"), r.stdout);
  assert("over-standard: shows the delta", r.stdout.includes("+1"), r.stdout);
  p.cleanup();
}

// --- 4. tier raises the ceiling -> same size passes ----------------------------
{
  const content = "x".repeat(22001); // over standard, under large
  const p = makeProject({
    skills: { heavy: content },
    budget: {
      ceiling: { standard: 22000, large: 38000, xl: 40000 },
      tiers: { heavy: "large" },
      skills: { heavy: 22001 },
    },
  });
  const r = runBudget(["check", p.root]);
  assert("tiered: exit 0 under its tier ceiling", r.code === 0, `got ${r.code}: ${r.stdout}${r.stderr}`);
  p.cleanup();
}

// --- 5. --write ratchets the recorded size -------------------------------------
{
  const before = "x".repeat(1000);
  const p = makeProject({
    skills: { grow: before },
    budget: {
      ceiling: { standard: 22000, large: 38000, xl: 40000 },
      tiers: {},
      skills: { grow: 1000 },
    },
  });
  const r = runBudget(["check", "--write", p.root]);
  assert("--write: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  const after = JSON.parse(readFileSync(join(p.root, ".agents", "skills", "_shared", "skill-size-budget.json"), "utf8"));
  assert("--write: ratchets to actual size", after.skills.grow === 1000 && after.tiers && after.ceiling, JSON.stringify(after));
  p.cleanup();
}

// --- 6. usage errors -> exit 2 ---------------------------------------------------
{
  let r = runBudget([]);
  assert("usage: no args exits 2", r.code === 2, `got ${r.code}`);
  r = runBudget(["frobnicate"]);
  assert("usage: unknown command exits 2", r.code === 2, `got ${r.code}`);
}

console.log(`\nskill-size-budget tests: ${passed} passed, ${failed} failed`);
if (failed > 0) {
  console.error(failures.join("\n"));
  process.exit(1);
}
