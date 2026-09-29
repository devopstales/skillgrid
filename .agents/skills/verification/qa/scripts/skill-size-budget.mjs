#!/usr/bin/env node
/**
 * skill-size-budget.mjs — byte-budget ratchet for SKILL.md files.
 *
 * Prevents unbounded growth of skill files. Each skill has a ceiling
 * (per-tier); the budget JSON records the last-known size so a skill
 * can only grow by a small delta per change (ratchet).
 *
 * Usage:
 *   node .agents/skills/verification/qa/scripts/skill-size-budget.mjs check [--write] [project-root]
 *
 * Exit codes:
 *   0 — all skills within ceiling (or no budget file)
 *   1 — one or more skills over their ceiling
 *   2 — usage error
 *
 * Budget JSON: .agents/skills/_shared/skill-size-budget.json
 * {
 *   "ceiling": { "standard": 22000, "large": 38000, "xl": 40000 },
 *   "tiers": { "subagent-execution": "xl", "brainstorming": "large" },
 *   "skills": { "brainstorming": 36285, "qa": 26632 }
 * }
 */

import { readFileSync, writeFileSync, readdirSync, existsSync, statSync, mkdirSync } from "node:fs";
import { join, resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const DEFAULT_CEILING = 22000;

function findSkillsDir(root) {
  return join(root, ".agents", "skills");
}

function listSkillFiles(skillsDir) {
  const results = [];
  if (!existsSync(skillsDir)) return results;
  for (const entry of readdirSync(skillsDir, { withFileTypes: true })) {
    if (!entry.isDirectory() || entry.name.startsWith(".")) continue;
    if (entry.name === "_shared") continue;
    const skillFile = join(skillsDir, entry.name, "SKILL.md");
    if (existsSync(skillFile)) {
      results.push({ name: entry.name, size: statSync(skillFile).size });
      continue;
    }
    // Group dir (Hermes-style: a DESCRIPTION.md with skill subdirs) — recurse one level.
    for (const sub of readdirSync(join(skillsDir, entry.name), { withFileTypes: true })) {
      if (!sub.isDirectory()) continue;
      const subFile = join(skillsDir, entry.name, sub.name, "SKILL.md");
      if (!existsSync(subFile)) continue;
      results.push({ name: sub.name, size: statSync(subFile).size });
    }
  }
  return results;
}

function loadBudget(root) {
  const budgetPath = join(root, ".agents", "skills", "_shared", "skill-size-budget.json");
  if (!existsSync(budgetPath)) return null;
  try {
    return JSON.parse(readFileSync(budgetPath, "utf8"));
  } catch {
    return null;
  }
}

function getCeiling(budget, skillName) {
  if (!budget) return DEFAULT_CEILING;
  const tier = budget.tiers?.[skillName];
  if (tier && budget.ceiling?.[tier]) return budget.ceiling[tier];
  return budget.ceiling?.standard ?? DEFAULT_CEILING;
}

function cmdCheck(root, doWrite) {
  const skillsDir = findSkillsDir(root);
  const skills = listSkillFiles(skillsDir);
  if (skills.length === 0) {
    console.log("No SKILL.md files found.");
    return 0;
  }

  const budget = loadBudget(root);
  const budgetPath = join(root, ".agents", "skills", "_shared", "skill-size-budget.json");

  if (!budget) {
    if (doWrite) {
      mkdirSync(dirname(budgetPath), { recursive: true });
      const newBudget = buildBudget(skills);
      writeFileSync(budgetPath, JSON.stringify(newBudget, null, 2) + "\n");
      console.log(`Wrote budget: ${budgetPath}`);
      for (const s of skills) {
        console.log(`  ${s.name}: ${s.size} bytes`);
      }
      return 0;
    }
    console.log("No budget file found. Run with --write to create one.");
    for (const s of skills) {
      console.log(`  ${s.name}: ${s.size} bytes`);
    }
    return 0;
  }

  if (doWrite) {
    for (const s of skills) {
      budget.skills[s.name] = s.size;
    }
    mkdirSync(dirname(budgetPath), { recursive: true });
    writeFileSync(budgetPath, JSON.stringify(budget, null, 2) + "\n");
    console.log(`Updated budget: ${budgetPath}`);
    return 0;
  }

  const overages = [];
  for (const s of skills) {
    const ceiling = getCeiling(budget, s.name);
    if (s.size > ceiling) {
      overages.push({ name: s.name, size: s.size, ceiling, delta: s.size - ceiling });
    }
  }

  if (overages.length === 0) {
    console.log(`All ${skills.length} skills within ceiling.`);
    return 0;
  }

  console.log("SKILL.md SIZE OVERAGE:");
  console.log("");
  console.log("| Skill | Size | Ceiling | Over By |");
  console.log("|-------|------|---------|---------|");
  for (const o of overages) {
    console.log(`| ${o.name} | ${o.size} | ${o.ceiling} | +${o.delta} |`);
  }
  console.log("");
  console.log("Trim the skill content or raise the ceiling in .agents/skills/_shared/skill-size-budget.json.");
  return 1;
}

function buildBudget(skills) {
  const skillsMap = {};
  for (const s of skills) {
    skillsMap[s.name] = s.size;
  }
  return {
    ceiling: { standard: 22000, large: 38000, xl: 40000 },
    tiers: {},
    skills: skillsMap,
  };
}

function main() {
  const args = process.argv.slice(2);
  if (args.length < 1) {
    console.error("Usage: skill-size-budget.mjs check [--write] [project-root]");
    process.exit(2);
  }

  const cmd = args[0];
  if (cmd !== "check") {
    console.error(`Unknown command: ${cmd}`);
    process.exit(2);
  }

  const rest = args.slice(1);
  const doWrite = rest.includes("--write");
  const pathArgs = rest.filter((a) => a !== "--write");
  const root = resolve(pathArgs[0] || ".");

  const code = cmdCheck(root, doWrite);
  process.exit(code);
}

main();
