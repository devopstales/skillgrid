#!/usr/bin/env node
// state-drift-check.mjs — READ-ONLY verifier: compares `.skillgrid/state.yaml`
// against the spec zone (`.skillgrid/change-specs/` + `.skillgrid/change-archive/`)
// and reports a named drift verdict. Pipeline skills own the write to
// state.yaml; this guard owns the verification (per _shared/rules/sdd-structure.md
// § "state.yaml drift guard"). A drift hit means a skill forgot to update
// state.yaml at a phase transition — the spec zone is the deeper truth.
//
// Usage: node state-drift-check.mjs [project-root]   (defaults to cwd)
//
// YAML parsing: ADR 0008 names Node + the `yaml` package. This script has
// zero hard dependencies (no package.json at repo root), so it dynamically
// imports `yaml` when resolvable and otherwise falls back to a purpose-built
// indentation parser that extracts ONLY the fields needed:
//   v2: changes.<slug>.phase (each map entry) + progress.completed_changes
//   v1 (legacy): pipeline.current_change + pipeline.current_phase
// The fallback reads 2-space-indented `key: value` scalars at known paths
// (quotes and inline comments stripped); unknown keys, nested maps, and
// list items are skipped. Multi-line quoted-string continuations are skipped
// the same way and can only add spurious deep path entries, never a shallow
// one — the extracted paths are at most 3 levels deep.
//
// The three checks (drift when):
//   1 phase      — the deepest present artifact in change-specs/<slug>/ derives
//                  a different phase than changes.<slug>.phase
//   2 presence   — a change-specs/<slug>/ dir exists without a changes:<slug>
//                  map entry, or a map entry names a dir that doesn't exist
//                  (bidirectional)
//   3 completion — a report.md with a shippable verdict (a Gate Decision
//                  section whose verdict is PASS or WAIVED, or CONCERNS with
//                  a recorded human override — a non-empty `## Human Override`
//                  section or an override line inside the Gate Decision
//                  section) exists in ANY change dir (both change-specs/ and
//                  change-archive/) while progress.completed_changes is 0
//
// Phase derivation (deepest artifact wins, per sdd-structure.md):
//   report.md present (any content)            → qa
//   tasks.md + execution signal                → execution
//     (a tasks.md line matching `^\s*-\s*\[x\]`, or
//      work/<slug>/progress.md containing `[x]`)
//   tasks.md only                              → slicing
//   blueprint.md                               → blueprint
//   briefing.md                                → spec
//   no derivable artifact                      → check 1 skipped for that entry
//                                                (noted in the SCOPE line)
//
// Legacy v1 state.yaml (no `changes:` map; `pipeline.current_change` +
// `pipeline.current_phase`): treated as a single-entry map for checks 1-2;
// check 3 skipped (no progress section).
//
// Output: a `DRIFT: clean` or `DRIFT: <n> hit(s)` verdict line; on drift, a
// `Check | Field | State says | Derived` table (one row per hit); ALWAYS a
// separate `SCOPE: <atom>` line describing the enumeration scope (per
// _shared/rules/verification-scope.md "Where It Applies").
//
// Exit codes: 0 clean, 1 drift found, 2 usage/parse error (absent or
// unreadable state.yaml, unrecognized schema).
// READ-ONLY: never writes anything (no state.yaml patch, no lock, no temp
// files outside /tmp).
import { existsSync, readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

function fail(msg) {
  console.error(`state-drift-check: ${msg}`);
  process.exit(2);
}

const args = process.argv.slice(2).filter(a => !a.startsWith("-"));
if (args.length > 1) fail(`unexpected argument(s): ${args.slice(1).join(" ")} (usage: node state-drift-check.mjs [project-root])`);
const root = args[0] || process.cwd();

const skillgridDir = join(root, ".skillgrid");
const statePath = join(skillgridDir, "state.yaml");
const specsDir = join(skillgridDir, "change-specs");
const archiveDir = join(skillgridDir, "change-archive");
const workDir = join(skillgridDir, "work");

// --- parse state.yaml (yaml package if resolvable, fallback otherwise) ----

let YAML = null;
try { YAML = (await import("yaml")).default; } catch { YAML = null; }

function stripInlineComment(value) {
  // strip a ` #` comment only when outside quotes
  let inS = false, inD = false;
  for (let i = 0; i < value.length; i++) {
    const c = value[i];
    if (c === "'" && !inD) inS = !inS;
    else if (c === '"' && !inS) inD = !inD;
    else if (c === "#" && !inS && !inD && i > 0 && /\s/.test(value[i - 1])) return value.slice(0, i);
  }
  return value;
}

function unquote(value) {
  const v = value.trim();
  if ((v.startsWith('"') && v.endsWith('"') && v.length >= 2) ||
      (v.startsWith("'") && v.endsWith("'") && v.length >= 2)) return v.slice(1, -1);
  return v;
}

// indentation parser: records scalar values by dotted path
function parseFallbackPaths(text) {
  const paths = {};
  const stack = []; // { indent, key }
  for (const rawLine of text.split(/\r?\n/)) {
    if (!rawLine.trim() || rawLine.trim().startsWith("#")) continue;
    const m = rawLine.match(/^(\s*)([^\s\-][^:]*):(?:\s+(.*))?$/);
    if (!m) continue; // list items, continuations, unknown shapes — skipped
    const indent = m[1].length;
    const key = m[2].trim();
    const value = m[3] !== undefined ? stripInlineComment(m[3]).trim() : "";
    while (stack.length && stack[stack.length - 1].indent >= indent) stack.pop();
    const dotted = [...stack.map(s => s.key), key].join(".");
    if (value !== "" && !(dotted in paths)) paths[dotted] = unquote(value);
    else if (value === "") paths[dotted] = undefined; // key present, no inline value
    stack.push({ indent, key });
  }
  return paths;
}

let rawText;
try { rawText = readFileSync(statePath, "utf8"); }
catch (e) { fail(`cannot read ${statePath}: ${e.message}`); }

let changes = null;      // { slug: phase | null }  (v2 map, or v1 single entry)
let completed = null;    // progress.completed_changes (int) or null
let legacy = false;

if (YAML) {
  let doc;
  try { doc = YAML.parse(rawText); }
  catch (e) { fail(`cannot parse ${statePath}: ${e.message}`); }
  if (doc && typeof doc === "object" && doc.changes && typeof doc.changes === "object") {
    changes = {};
    for (const [slug, entry] of Object.entries(doc.changes)) {
      changes[slug] = entry && typeof entry === "object" && typeof entry.phase === "string"
        ? entry.phase : null;
    }
    completed = doc.progress && Number.isInteger(doc.progress.completed_changes)
      ? doc.progress.completed_changes : null;
  } else if (doc && typeof doc === "object" && doc.pipeline && typeof doc.pipeline === "object") {
    legacy = true;
    changes = { [String(doc.pipeline.current_change)]: typeof doc.pipeline.current_phase === "string" ? doc.pipeline.current_phase : null };
  } else {
    fail("unrecognized state.yaml schema (expected a v2 `changes:` map or a v1 `pipeline:` section)");
  }
} else {
  const p = parseFallbackPaths(rawText);
  const hasChanges = Object.keys(p).some(k => k === "changes" || k.startsWith("changes."));
  if (hasChanges) {
    changes = {};
    for (const [k, v] of Object.entries(p)) {
      const e = k.match(/^changes\.([^.]*)$/);
      if (e) changes[e[1]] = null; // map entry (phase may be absent)
      const m = k.match(/^changes\.([^.]*)\.phase$/);
      if (m) changes[m[1]] = v;
      if (k === "progress.completed_changes" && v !== undefined) completed = parseInt(v, 10);
    }
  } else if (p["pipeline.current_change"]) {
    legacy = true;
    changes = { [p["pipeline.current_change"]]: p["pipeline.current_phase"] ?? null };
  } else {
    fail("unrecognized state.yaml schema (expected a v2 `changes:` map or a v1 `pipeline:` section)");
  }
}

// --- enumerate the spec zone ----------------------------------------------

function subdirs(dir) {
  if (!existsSync(dir)) return [];
  return readdirSync(dir, { withFileTypes: true })
    .filter(e => e.isDirectory() && !e.name.startsWith("."))
    .map(e => e.name).sort();
}

const specDirs = subdirs(specsDir);
const archiveDirs = subdirs(archiveDir);
const slugs = Object.keys(changes).sort();

// --- phase derivation (check 1) --------------------------------------------

function derivePhase(dir) {
  const has = f => existsSync(join(dir, f));
  if (has("report.md")) return "qa";
  if (has("tasks.md")) {
    const tasksText = readFileSync(join(dir, "tasks.md"), "utf8");
    if (/^\s*-\s*\[x\]/m.test(tasksText)) return "execution";
    const progress = join(workDir, dirName(dir), "progress.md");
    if (existsSync(progress) && /\[x\]/i.test(readFileSync(progress, "utf8"))) return "execution";
    return "slicing";
  }
  if (has("blueprint.md")) return "blueprint";
  if (has("briefing.md")) return "spec";
  return null;
}

function dirName(p) {
  const parts = p.split("/");
  return parts[parts.length - 1];
}

// --- shippable verdict (check 3) --------------------------------------------

function sectionText(text, heading) {
  const lines = text.split(/\r?\n/);
  const out = [];
  let inSec = false;
  for (const line of lines) {
    if (new RegExp(`^##\\s+${heading}\\b`, "i").test(line)) { inSec = true; continue; }
    if (inSec && /^##\s/.test(line)) break;
    if (inSec) out.push(line);
  }
  return out.join("\n");
}

function isShippableReport(text) {
  const gate = sectionText(text, "Gate Decision");
  if (!gate.trim()) return false;
  const v = gate.match(/\*\*Verdict:\*\*\s*\**\s*(PASS|CONCERNS|FAIL|WAIVED)/i);
  if (!v) return false;
  const verdict = v[1].toUpperCase();
  if (verdict === "PASS" || verdict === "WAIVED") return true;
  if (verdict !== "CONCERNS") return false;
  // recorded human override: a non-empty `## Human Override` section, or an
  // override line inside the Gate Decision section
  const ho = sectionText(text, "Human Override")
    .replace(/\*\*Override:\*\*/gi, "")
    .replace(/\*\*Waiver[^:]*:\*\*/gi, "")
    .replace(/[-*|]/g, "")
    .replace(/\s+/g, " ")
    .trim();
  if (ho && !/^none\b/i.test(ho)) return true;
  return /override/i.test(gate);
}

let shippableDir = null;
for (const [zone, dirs] of [["change-specs", specDirs], ["change-archive", archiveDirs]]) {
  for (const d of dirs) {
    const report = join(zone === "change-specs" ? specsDir : archiveDir, d, "report.md");
    if (existsSync(report) && isShippableReport(readFileSync(report, "utf8"))) {
      shippableDir = `${zone}/${d}`;
      break;
    }
  }
  if (shippableDir) break;
}

// --- run the checks -----------------------------------------------------------

const hits = [];
const undervivable = []; // dirs with no derivable artifact (check 1 skipped)

// check 2: presence (bidirectional)
for (const d of specDirs) {
  if (!(d in changes)) hits.push({ check: "presence", field: `changes:${d}`, state: "absent entry", derived: "dir present" });
}
for (const s of slugs) {
  if (!specDirs.includes(s)) hits.push({ check: "presence", field: `changes:${s}`, state: "entry present", derived: "dir absent" });
}

// check 1: phase (per entry with an existing dir)
for (const s of slugs) {
  if (!specDirs.includes(s)) continue;
  const derived = derivePhase(join(specsDir, s));
  if (derived === null) { undervivable.push(s); continue; }
  const stated = changes[s];
  if (stated !== derived) hits.push({ check: "phase", field: `changes.${s}.phase`, state: stated ?? "(missing)", derived });
}

// check 3: completion (v2 only)
let check3skipped = null;
if (legacy) check3skipped = "legacy v1 has no progress section";
else if (completed === null) check3skipped = "no progress.completed_changes";
else if (shippableDir && completed === 0) {
  hits.push({ check: "completion", field: "progress.completed_changes", state: "0", derived: `shippable report in ${shippableDir}` });
}

// --- emit ----------------------------------------------------------------------

console.log(hits.length === 0 ? "DRIFT: clean" : `DRIFT: ${hits.length} hit(s)`);
if (hits.length > 0) {
  console.log("");
  console.log("| Check | Field | State says | Derived |");
  console.log("|---|---|---|---|");
  for (const h of hits) console.log(`| ${h.check} | ${h.field} | ${h.state} | ${h.derived} |`);
}
const notes = [];
if (!existsSync(specsDir)) notes.push("change-specs absent (nothing to compare)");
if (legacy) notes.push("legacy v1 single-entry map");
if (undervivable.length) notes.push(`${undervivable.length} dir(s) without a derivable artifact (check 1 skipped): ${undervivable.join(", ")}`);
if (check3skipped) notes.push(`completion check skipped (${check3skipped})`);
const scope = `SCOPE: change-specs: ${specDirs.length} dir(s) enumerated, ${slugs.length} map entr${slugs.length === 1 ? "y" : "ies"} compared, ${archiveDirs.length} archive dir(s) scanned` + (notes.length ? `; ${notes.join("; ")}` : "");
console.log(scope);

process.exit(hits.length === 0 ? 0 : 1);
