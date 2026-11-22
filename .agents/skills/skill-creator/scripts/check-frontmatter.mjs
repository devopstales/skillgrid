#!/usr/bin/env node
/**
 * check-frontmatter.mjs — SKILL.md frontmatter contract validator.
 *
 * Validates every SKILL.md's YAML frontmatter against the exact field
 * set and order defined in _shared/craft/skill-anatomy.md:
 *   - `name` present, kebab-case, equals the directory name
 *   - `description` present, non-empty, <= 1024 chars
 *   - `license` present
 *   - `metadata.author`, `metadata.version` (quoted semver), `metadata.part-of: skillgrid`
 *   - field order: name, description, license, metadata
 *   - if `disable-model-invocation` is present, it is boolean
 *
 * This is a CI lint: it catches the drift the size budget cannot — a
 * frontmatter that no longer matches the anatomy contract.
 *
 * Usage:
 *   node .agents/skills/skill-creator/scripts/check-frontmatter.mjs check [project-root]
 *
 * Exit codes:
 *   0 — all SKILL.md frontmatter valid (or none found)
 *   1 — one or more frontmatter violations
 *   2 — usage error
 */

import { readFileSync, readdirSync, existsSync } from "node:fs";
import { join, resolve } from "node:path";

const KEBAB_RE = /^[a-z0-9]+(-[a-z0-9]+)*$/;
// Repo convention is 2-part MAJOR.MINOR (quoted), per skill-anatomy.md.
const VERSION_RE = /^\d+\.\d+$/;
const MAX_DESC = 1024;
// _shared/ is a non-skill reference bundle: exempt from name==directory and
// from license (it carries no per-skill license of its own).
const SHARED_DIR = "_shared";

// Required top-level fields in the exact order the anatomy contract fixes.
// `metadata` is an object; the rest are scalars.
const REQUIRED_ORDER = ["name", "description", "license", "metadata"];
// Optional fields allowed in the frontmatter (order not enforced, presence
// is). Anything else is a violation.
//   - effort: the top-level effort-budget signal (per _shared/craft/effort-budgets.md)
//   - based_on: also legal inside metadata; allowed at top level for legacy
const OPTIONAL_FIELDS = new Set([
  "disable-model-invocation",
  "user-invocable",
  "effort",
  "based_on",
]);

function findSkillsDir(root) {
  return join(root, ".agents", "skills");
}

// Skills are FLAT: .agents/skills/<skill-name>/SKILL.md (one level deep).
function listSkillFiles(skillsDir) {
  const results = [];
  if (!existsSync(skillsDir)) return results;
  for (const entry of readdirSync(skillsDir, { withFileTypes: true })) {
    if (!entry.isDirectory() || entry.name.startsWith(".")) continue;
    const skillFile = join(skillsDir, entry.name, "SKILL.md");
    if (existsSync(skillFile)) {
      results.push({ name: entry.name, file: skillFile });
    }
  }
  return results;
}

// Split the leading YAML frontmatter block (between the first two `---`
// lines) from the body. Returns null when there is no frontmatter block.
function splitFrontmatter(text) {
  const lines = text.split("\n");
  if (lines[0] !== "---") return null;
  let end = -1;
  for (let i = 1; i < lines.length; i++) {
    if (lines[i] === "---") {
      end = i;
      break;
    }
  }
  if (end === -1) return null;
  return lines.slice(1, end);
}

// Minimal top-level key parser. Returns { keys: [names in order], values:
// {key: rawValueString} } where rawValueString is the inline value after
// `key:` (empty for block-style values like `metadata:`). Indented lines are
// ignored (they belong to the preceding key).
function parseTopLevel(fmLines) {
  const keys = [];
  const values = {};
  for (const line of fmLines) {
    if (line.startsWith(" ") || line.startsWith("\t") || line.trim() === "") continue;
    const m = line.match(/^([A-Za-z0-9_-]+):\s?(.*)$/);
    if (!m) continue;
    keys.push(m[1]);
    values[m[1]] = m[2];
  }
  return { keys, values };
}

// Unquote a YAML scalar for comparison (strips matching surrounding quotes).
function unquote(s) {
  const t = s.trim();
  if (
    (t.startsWith('"') && t.endsWith('"') && t.length >= 2) ||
    (t.startsWith("'") && t.endsWith("'") && t.length >= 2)
  ) {
    return t.slice(1, -1);
  }
  return t;
}

// A top-level key is a violation if it is neither required nor optional.
// For `metadata`, the object is validated separately by the caller.
function isKnownField(key) {
  return REQUIRED_ORDER.includes(key) || OPTIONAL_FIELDS.has(key);
}

function validateSkill({ name, file }) {
  const issues = [];
  const text = readFileSync(file, "utf8");
  const fmLines = splitFrontmatter(text);
  if (!fmLines) {
    return ["no `---` frontmatter block"];
  }

  const { keys, values } = parseTopLevel(fmLines);

  const isShared = name === SHARED_DIR;

  // 1. Required fields present, in order.
  const required = REQUIRED_ORDER.filter((k) => !(isShared && k === "license"));
  const present = new Set(keys);
  for (const req of required) {
    if (!present.has(req)) issues.push(`missing required field \`${req}\``);
  }
  // Order check: walk the required fields that are present and confirm they
  // appear as a contiguous, in-order subsequence of `keys`.
  const requiredPresent = keys.filter((k) => REQUIRED_ORDER.includes(k));
  const expectedOrder = REQUIRED_ORDER.filter((k) => present.has(k));
  for (let i = 0; i < expectedOrder.length; i++) {
    if (requiredPresent[i] !== expectedOrder[i]) {
      issues.push(
        `field order wrong: found \`${requiredPresent[i]}\` where expected \`${expectedOrder[i]}\``
      );
    }
  }

  // 2. Unknown fields.
  for (const k of keys) {
    if (!isKnownField(k)) issues.push(`unknown field \`${k}\``);
  }

  // 3. name = directory name, kebab-case. _shared is a non-skill bundle and
  // is exempt from the name==directory rule (its `name: shared` is fine).
  const rawName = values.name;
  if (rawName !== undefined) {
    const nm = unquote(rawName);
    if (!KEBAB_RE.test(nm)) issues.push(`name \`${nm}\` is not kebab-case`);
    if (!isShared && nm !== name)
      issues.push(`name \`${nm}\` != directory \`${name}\``);
  }

  // 4. description non-empty and <= 1024 chars.
  if (values.description !== undefined) {
    const desc = unquote(values.description);
    if (desc.length === 0) issues.push("description is empty");
    else if (desc.length > MAX_DESC)
      issues.push(`description ${desc.length} chars > ${MAX_DESC}`);
  }

  // 5. metadata.version quoted semver; metadata.part-of = skillgrid.
  if (values.metadata !== undefined) {
    // Inline `metadata: {…}` is not our contract form; require a block.
    if (values.metadata.trim() !== "") {
      issues.push("metadata must be a block (no inline value)");
    }
    // Re-scan the metadata sub-block for version / part-of.
    const meta = parseMetaBlock(fmLines);
    if (meta.version !== undefined) {
      // version must be quoted in source; capture raw.
      if (meta.versionRaw !== undefined && meta.versionRaw.trim() !== "") {
        const quoted =
          meta.versionRaw.trim().startsWith('"') || meta.versionRaw.trim().startsWith("'");
        if (!quoted) issues.push("metadata.version is not quoted");
      }
      const v = unquote(meta.version);
      if (!VERSION_RE.test(v))
        issues.push(`metadata.version \`${v}\` is not MAJOR.MINOR`);
    } else {
      issues.push("missing metadata.version");
    }
    if (meta["part-of"] !== undefined) {
      const pof = unquote(meta["part-of"]);
      if (pof !== "skillgrid") issues.push(`metadata.part-of is \`${pof}\`, expected skillgrid`);
    } else {
      issues.push("missing metadata.part-of");
    }
  }

  // 6. disable-model-invocation, when present, must be boolean.
  if (values["disable-model-invocation"] !== undefined) {
    const dimv = unquote(values["disable-model-invocation"]);
    if (dimv !== "true" && dimv !== "false")
      issues.push(`disable-model-invocation \`${dimv}\` is not boolean`);
  }

  return issues;
}

// Parse the `metadata:` sub-block into { version, versionRaw, "part-of", ... }.
function parseMetaBlock(fmLines) {
  const out = {};
  let inMeta = false;
  for (const line of fmLines) {
    if (/^metadata:\s*$/.test(line)) {
      inMeta = true;
      continue;
    }
    if (!inMeta) continue;
    // An unindented key ends the metadata block.
    if (!line.startsWith(" ") && !line.startsWith("\t") && line.trim() !== "") {
      break;
    }
    const m = line.match(/^\s+([A-Za-z0-9_-]+):\s?(.*)$/);
    if (!m) continue;
    out[m[1]] = m[2];
    if (m[1] === "version") out.versionRaw = m[2];
  }
  return out;
}

function cmdCheck(root) {
  const skillsDir = findSkillsDir(root);
  const skills = listSkillFiles(skillsDir);
  if (skills.length === 0) {
    console.log("No SKILL.md files found.");
    return 0;
  }

  const failures = [];
  for (const s of skills) {
    const issues = validateSkill(s);
    if (issues.length > 0) failures.push({ name: s.name, issues });
  }

  if (failures.length === 0) {
    console.log(`All ${skills.length} SKILL.md frontmatter valid.`);
    return 0;
  }

  console.log("FRONTMATTER VIOLATIONS:");
  console.log("");
  console.log("| Skill | Issue |");
  console.log("|-------|-------|");
  for (const f of failures) {
    for (const issue of f.issues) {
      console.log(`| ${f.name} | ${issue} |`);
    }
  }
  console.log("");
  console.log(
    `Fix per the contract in .agents/skills/_shared/craft/skill-anatomy.md. ${failures.length} skill(s) affected.`
  );
  return 1;
}

function main() {
  const args = process.argv.slice(2);
  if (args.length < 1) {
    console.error("Usage: check-frontmatter.mjs check [project-root]");
    process.exit(2);
  }
  const cmd = args[0];
  const root = resolve(args[1] || ".");
  if (cmd === "check") {
    process.exit(cmdCheck(root));
  }
  console.error(`Unknown command: ${cmd}`);
  process.exit(2);
}

main();
