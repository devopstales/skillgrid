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
//      raw .agents/skills/[<group>/]<other-skill>/ path into another skill's dir
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
  // 600: the plan-authoring spine — threat matrix, plan review, slicing
  // handoff, and the task-structure sample all cross-reference each other, so
  // the interlocking spine stays inline; only the deep-module tail is in
  // references/. Ratcheted from 555 → 560 → 573 (real size at lock); push
  // tail out to drop it back to "heavyPlus" (560).
  "writing-blueprints": "orchestrator",
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
  // Ratcheted 2026-09-30 from the pre-grouping sizes: blueprint 540→630
  // (writing-blueprints spine grew), execution 675→750 (subagent-execution
  // review loop), qa 504→690 (qa gate checklist), close 657→730 (ship docs
  // step). Each raise buys drift headroom, not growth permission.
  "blueprint.ceiling": 630,
  "blueprint.warnAt": 567,
  "execution.ceiling": 750,
  "execution.warnAt": 675,
  "qa.ceiling": 690,
  "qa.warnAt": 621,
  "close.ceiling": 740,
  "close.warnAt": 666,
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

// Skills live either flat (a dir with SKILL.md) or nested inside group dirs
// (a dir with DESCRIPTION.md, Hermes-style; groups may nest, e.g. craft/
// craft-refactor/). _shared is infra, not a skill. A skill is external when
// its frontmatter lacks `metadata.part-of: skillgrid` — external skills are
// inventoried (duplicate names, ref targets) but skip the skillgrid anatomy
// checks, since they are vendored from third-party repos.
const GROUP_DESC = "DESCRIPTION.md";
const skillEntries = []; // { name, group: string|null, external: boolean }
const groupNames = new Set();
function scan(dir, group) {
  for (const d of readdirSync(dir, { withFileTypes: true })) {
    if (!d.isDirectory() || d.name.startsWith(".")) continue;
    const p = join(dir, d.name);
    if (existsSync(join(p, "SKILL.md"))) {
      const text = readFileSync(join(p, "SKILL.md"), "utf8");
      const external = !/^\s*part-of: *skillgrid/m.test(frontmatterOf(text) ?? "");
      skillEntries.push({ name: d.name, group, external });
    } else if (existsSync(join(p, GROUP_DESC))) {
      groupNames.add(group ? `${group}/${d.name}` : d.name);
      scan(p, group ? `${group}/${d.name}` : d.name);
    }
  }
}
scan(SKILLS, null);
skillEntries.sort((a, b) => a.name.localeCompare(b.name));
const skillDirs = skillEntries.map((e) => e.name);
const skillFile = (e) =>
  e.group ? join(SKILLS, e.group, e.name, "SKILL.md") : join(SKILLS, e.name, "SKILL.md");
const skillRel = (e) =>
  e.group ? `.agents/skills/${e.group}/${e.name}/SKILL.md` : `.agents/skills/${e.name}/SKILL.md`;
const skillRefsDir = (e) =>
  e.group ? join(SKILLS, e.group, e.name, "references") : join(SKILLS, e.name, "references");
const skillRefsRel = (e, file) =>
  e.group
    ? `.agents/skills/${e.group}/${e.name}/references/${file}`
    : `.agents/skills/${e.name}/references/${file}`;

// Duplicate skill names are a load-order hazard: the first match wins silently.
const seen = new Map();
for (const e of skillEntries) {
  const key = e.name.toLowerCase();
  if (seen.has(key)) err(`${skillRel(e)}: duplicate skill name "${e.name}" (also ${seen.get(key)})`);
  else seen.set(key, skillRel(e));
}

const lineCounts = {}; // skill name -> SKILL.md line count (for hot paths)

for (const e of skillEntries) {
  const name = e.name;
  const f = skillFile(e);
  const rel = skillRel(e);
  const text = readFileSync(f, "utf8");
  const nLines = text.split("\n").length;
  lineCounts[name] = nLines;
  if (e.external || name === "_shared") continue;

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

  // 4. cross-skill refs: a raw path is allowed only into this skill's own
  // dir (scripts/, references/) or into _shared. Any raw path resolving to a
  // DIFFERENT skill's dir is a violation — use skillgrid:{name} instead.
  const ownRel = e.group ? `${e.group}/${name}/` : `${name}/`;
  const rawRefs = text.match(/\.agents\/skills\/[a-z0-9-]+(?:\/[a-z0-9-]+)*/g) ?? [];
  const bad = new Set();
  for (const r of rawRefs) {
    const segs = r.replace(/\.agents\/skills\//, "").replace(/\/$/, "").split("/");
    while (segs.length > 1 && groupNames.has(segs.join("/"))) segs = segs.slice(1);
    const target = segs.join("/");
    if (target === "_shared" || target.startsWith(ownRel)) continue;
    if (skillDirs.some((s) => s.toLowerCase() === segs[0].toLowerCase())) bad.add(segs[0]);
  }
  if (bad.size) {
    err(rel, `raw cross-skill ref into: ${[...bad].join(", ")} — use skillgrid:{name} or a _shared path`);
  }
}

// reference/ files: <= 250 lines each (skillgrid skills only)
for (const e of skillEntries) {
  if (e.external) continue;
  const refs = skillRefsDir(e);
  if (!existsSync(refs)) continue;
  for (const file of readdirSync(refs)) {
    if (!file.endsWith(".md")) continue;
    const p = join(refs, file);
    if (!statSync(p).isFile()) continue;
    const rel = skillRefsRel(e, file);
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
