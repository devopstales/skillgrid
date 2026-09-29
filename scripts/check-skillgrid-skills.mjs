#!/usr/bin/env node
// check-skillgrid-skills.mjs — skillgrid skill-hygiene guard.
//
// Enforces the invariants from _shared/rules/skill-anatomy.md so the
// 30+ skills stop drifting:
//   1. frontmatter: name == dir (kebab-case), description <= 400 chars,
//      license + metadata(version/part-of) present
//   2. required sections: ## When to Use (+ "When NOT to use"),
//      ## Common Rationalizations, ## Red Flags, ## Verification
//   3. per-file line budgets by tier (standard/heavy/orchestrator/reference)
//   4. cross-skill references use skillgrid:{name} or a _shared path — never a
//      raw .agents/skills/<other-skill>/ path into another skill's dir
//   5. hot-path budgets: the cumulative SKILL.md load of the canonical change
//      paths (blueprint / execution / qa / close) stays within a ratcheted
//      ceiling
//
// Exit 0 = clean, 1 = violations. Zero dependencies (Node >= 18).

import { readFileSync, readdirSync, existsSync, statSync } from "node:fs";
import { join, resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const SKILLS = join(ROOT, ".agents", "skills");

const KEBAB = /^[a-z0-9]+(-[a-z0-9]+)*$/;

// Per-file line budgets by tier. "standard" is the default. A skill earns a
// higher tier by explicit ratchet entry below — crossing the budget is a code
// smell, not a permission, so widening is deliberate and commented.
const BUDGET = {
  standard: 400,
  heavy: 500,
  heavyPlus: 560,
  orchestrator: 600,
  reference: 250,
};

// Ratcheted per-skill tier overrides (beyond the default "standard").
// Raise only with a one-line why; lower the moment the tail is pushed out.
const TIER_OVERRIDE = {
  // Orchestrator: carries an entire phase of control flow whose steps
  // cross-reference each other too densely to split.
  "subagent-execution": "orchestrator",
  // Heavy+: the plan-authoring spine — threat matrix, plan review, slicing
  // handoff, and the task-structure sample all cross-reference each other, so
  // the interlocking spine stays inline; only the deep-module tail is in
  // references/. Ratcheted from 555 (real size at lock); push tail out to drop
  // it back to "heavy" (500).
  "writing-blueprints": "heavyPlus",
  // Heavy: process + many references.
  "ship": "heavy",
  "brainstorming": "heavy",
  "structured-debugging": "heavy",
  "test-driven-development": "heavy",
  "qa": "heavy",
};

// Skills anatomy documents as skipping the Announce line.
const ANNOUNCE_SKIP = new Set([
  "using-skillgrid",
  "test-driven-development",
  "test-driven-verification",
]);

// Canonical change paths: the cumulative SKILL.md an agent loads for each.
// Budgets are ratcheted from the real sizes (measured at lock); warn at 90%.
const HOT_PATHS = {
  blueprint: ["writing-blueprints"],
  execution: ["subagent-execution", "simple-execution"],
  qa: ["qa", "test-driven-verification"],
  close: ["ship", "reflect"],
  // ceiling: ratchet (raise only with a comment).
  // warnAt:  90% of ceiling.
  "blueprint.ceiling": 600,
  "blueprint.warnAt": 540,
  "execution.ceiling": 750,
  "execution.warnAt": 675,
  "qa.ceiling": 560,
  "qa.warnAt": 504,
  "close.ceiling": 730,
  "close.warnAt": 657,
};

const errors = [];
const warns = [];

function err(f, msg) {
  errors.push(`${f}: ${msg}`);
}
function warn(f, msg) {
  warns.push(`${f}: ${msg}`);
}

function linesOf(p) {
  return readFileSync(p, "utf8").split("\n").length;
}

function frontmatterOf(text) {
  const m = text.match(/^---\n([\s\S]*?)\n---/);
  return m ? m[1] : null;
}

function parseDesc(fm) {
  const m = fm.match(/^description: *(.+)$/m);
  if (!m) return null;
  let v = m[1].trim();
  if ((v.startsWith('"') && v.endsWith('"')) || (v.startsWith("'") && v.endsWith("'"))) {
    v = v.slice(1, -1);
  }
  return v;
}

function parseName(fm) {
  const m = fm.match(/^name: *(.+)$/m);
  return m ? m[1].trim() : null;
}

// --- per-skill checks -------------------------------------------------------

const skillDirs = readdirSync(SKILLS, { withFileTypes: true })
  .filter((d) => d.isDirectory() && d.name !== "_shared" && existsSync(join(SKILLS, d.name, "SKILL.md")))
  .map((d) => d.name)
  .sort();

const lineCounts = {}; // skill name -> SKILL.md line count (for hot paths)

for (const name of skillDirs) {
  const f = join(SKILLS, name, "SKILL.md");
  const rel = `.agents/skills/${name}/SKILL.md`;
  const text = readFileSync(f, "utf8");
  const nLines = text.split("\n").length;
  lineCounts[name] = nLines;

  // 1. frontmatter
  const fm = frontmatterOf(text);
  if (!fm) {
    err(rel, "missing frontmatter (--- ... ---)");
  } else {
    const fmName = parseName(fm);
    if (fmName !== name) err(rel, `frontmatter name "${fmName}" != dir "${name}"`);
    if (!KEBAB.test(name)) err(rel, `dir "${name}" is not kebab-case`);
    const desc = parseDesc(fm);
    if (desc == null) err(rel, "missing description");
    else if (desc.length > 400) err(rel, `description is ${desc.length} chars (> 400)`);
    if (!/^\s*license: *MIT/m.test(fm)) err(rel, "missing license: MIT");
    if (!/^\s*version: *"?[\d.]+"?/m.test(fm)) err(rel, "missing metadata.version");
    if (!/^\s*part-of: *skillgrid/m.test(fm)) err(rel, "missing metadata.part-of: skillgrid");
  }

  // 2. required sections
  if (!/^## When to Use/m.test(text)) err(rel, "missing ## When to Use");
  if (!/When NOT to use/i.test(text)) err(rel, 'missing "When NOT to use"');
  if (!/^## Common Rationalizations/m.test(text)) err(rel, "missing ## Common Rationalizations");
  if (!/^## Red Flags/m.test(text)) err(rel, "missing ## Red Flags");
  if (!/^## Verification/m.test(text)) err(rel, "missing ## Verification");
  if (!ANNOUNCE_SKIP.has(name) && !/Announce at start/i.test(text)) {
    err(rel, "missing Announce line (or add to ANNOUNCE_SKIP)");
  }

  // 3. per-file line budget
  const tier = TIER_OVERRIDE[name] ?? "standard";
  const budget = BUDGET[tier];
  if (nLines > budget) {
    err(rel, `is ${nLines} lines (> ${tier} budget ${budget}) — push tail to references/`);
  }

  // 4. cross-skill refs: no raw path into another skill's dir
  const rawRef = text.match(/\.agents\/skills\/([a-z0-9-]+)\//g);
  if (rawRef) {
    const bad = [...new Set(rawRef.map((r) => r.replace(".agents/skills/", "").replace("/", "")))].filter(
      (other) => other !== name
    );
    if (bad.length) err(rel, `raw cross-skill ref into: ${bad.join(", ")} — use skillgrid:{name} or a _shared path`);
  }
}

// reference/ files: <= 250 lines each
for (const name of skillDirs) {
  const refs = join(SKILLS, name, "references");
  if (!existsSync(refs)) continue;
  for (const file of readdirSync(refs)) {
    if (!file.endsWith(".md")) continue;
    const p = join(refs, file);
    if (!statSync(p).isFile()) continue;
    const rel = `.agents/skills/${name}/references/${file}`;
    const n = linesOf(p);
    if (n > BUDGET.reference) err(rel, `is ${n} lines (> reference budget ${BUDGET.reference})`);
  }
}

// 5. hot-path budgets (cumulative SKILL.md load per canonical path)
for (const [path, skills] of Object.entries(HOT_PATHS)) {
  if (!Array.isArray(skills)) continue;
  const total = skills.reduce((sum, s) => sum + (lineCounts[s] ?? 0), 0);
  const ceiling = HOT_PATHS[`${path}.ceiling`];
  const warnAt = HOT_PATHS[`${path}.warnAt`];
  const label = `hot-path:${path} (${skills.join(" + ")})`;
  if (total > ceiling) err(label, `is ${total} lines (> ceiling ${ceiling}) — the path is too heavy to load at once`);
  else if (total > warnAt) warn(label, `is ${total} lines (> warn ${warnAt}, ceiling ${ceiling}) — ratchet pressure`);
}

// --- report -----------------------------------------------------------------

if (warns.length) {
  console.log("WARN");
  for (const w of warns) console.log("  " + w);
}
if (errors.length) {
  console.log("\nFAIL — skill hygiene violations:");
  for (const e of errors) console.log("  " + e);
  console.log(`\n${errors.length} error(s), ${warns.length} warning(s) across ${skillDirs.length} skills.`);
  process.exit(1);
}

console.log(`OK — ${skillDirs.length} skills, 0 errors, ${warns.length} warning(s).`);
process.exit(0);
