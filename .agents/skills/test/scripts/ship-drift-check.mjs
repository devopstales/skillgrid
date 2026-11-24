#!/usr/bin/env node
// ship-drift-check.mjs — structure drift check: surfaces files changed outside
// the anticipated set at QA time, before integration, while the spec is still
// the contract (per _shared/rules/sdd-structure.md § "Ship drift").
//
// The anticipated set is the union of tasks.md's `Files:` column and
// blueprint.md's Global Constraints paths. That resolution is the CALLER's
// job (the `test` skill); the script receives the set via --anticipated and
// does not read tasks.md or blueprint.md itself.
//
// Usage:
//   node ship-drift-check.mjs check <base-ref> --anticipated <path>...
//        [--exclude <glob>...] [project-root]
//     <base-ref>       git ref to diff against (base vs working tree —
//                      uncommitted changes are included; that is the point at
//                      QA time)
//     --anticipated    path prefixes (dirs or files); a changed file is
//                      anticipated if its path equals or is under one of them.
//                      May be repeated or space-separated.
//     --exclude        extra glob to exclude from the changed set. May be
//                      repeated or space-separated.
//     [project-root]   repo to run git in (defaults to cwd)
//
// Default excludes (always applied): *.lock, *.sum, package-lock.json,
// yarn.lock, pnpm-lock.yaml.
// Glob semantics: `*` = any run except `/`, `?` = one char except `/`,
// `**` = any run including `/`. A pattern containing no `/` is matched
// against the file's basename at any depth; a pattern containing `/` is
// matched against the full path.
//
// Git failure (unknown ref, not a repo) → exit 2 with git's stderr.
//
// Output: `DRIFT: clean` or `DRIFT: <n> unexpected file(s)` + the list of
// unexpected paths (each prefixed `  - `) + a `SCOPE: <atom>` line describing
// the enumeration scope (per _shared/rules/verification-scope.md "Where It
// Applies").
//
// Exit codes: 0 clean, 1 drift found, 2 usage/git error (unknown flag,
// missing <base-ref>, missing --anticipated, git failure).
// READ-ONLY: runs git diff only; writes nothing.
import { execFileSync } from "node:child_process";
import { existsSync } from "node:fs";
import { join } from "node:path";

function usage(msg) {
  if (msg) console.error(`ship-drift-check: ${msg}`);
  console.error("usage: node ship-drift-check.mjs check <base-ref> --anticipated <path>... [--exclude <glob>...] [project-root]");
  process.exit(2);
}

function gitError(e) {
  const detail = (e.stderr || "").trim();
  console.error(`ship-drift-check: git diff failed: ${detail || e.message}`);
  if (detail) console.error(detail);
  process.exit(2);
}

// --- argument parsing ---------------------------------------------------------

const argv = process.argv.slice(2);
if (argv.length === 0 || argv[0] !== "check") usage(argv.length === 0 ? "missing subcommand (expected `check`)" : `unknown subcommand: ${argv[0]}`);

let baseRef = null;
const positional = [];
const anticipated = [];
const excludes = [];
for (let i = 1; i < argv.length; i++) {
  const a = argv[i];
  if (a === "--anticipated") {
    let got = false;
    while (i + 1 < argv.length && !argv[i + 1].startsWith("--")) { anticipated.push(argv[++i]); got = true; }
    if (!got) usage("--anticipated needs at least one path");
  } else if (a === "--exclude") {
    let got = false;
    while (i + 1 < argv.length && !argv[i + 1].startsWith("--")) { excludes.push(argv[++i]); got = true; }
    if (!got) usage("--exclude needs at least one glob");
  } else if (a.startsWith("--")) {
    usage(`unknown flag: ${a}`);
  } else {
    positional.push(a);
  }
}
if (positional.length === 0) usage("missing <base-ref>");
if (positional.length > 2) usage(`unexpected argument(s): ${positional.slice(2).join(" ")}`);
baseRef = positional[0];
const root = positional[1] || process.cwd();
if (anticipated.length === 0) usage("missing --anticipated <path>...");
if (!existsSync(root)) usage(`no such project root: ${root}`);

// --- glob matching --------------------------------------------------------------

function globToRegex(g) {
  let re = "";
  for (let i = 0; i < g.length; i++) {
    const c = g[i];
    if (c === "*") {
      if (g[i + 1] === "*") { re += ".*"; i++; if (g[i + 1] === "/") i++; }
      else re += "[^/]*";
    } else if (c === "?") {
      re += "[^/]";
    } else {
      re += c.replace(/[.+^${}()|[\]\\]/g, "\\$&");
    }
  }
  return new RegExp(`^${re}$`);
}

function matchesExclude(p, globs) {
  return globs.some(g => {
    const re = globToRegex(g);
    return g.includes("/") ? re.test(p) : re.test(p.split("/").pop());
  });
}

// --- changed files ----------------------------------------------------------------

let diff;
try {
  diff = execFileSync("git", ["diff", "--name-only", baseRef], { cwd: root, encoding: "utf8" });
} catch (e) { gitError(e); }
const changed = diff.split(/\r?\n/).map(s => s.trim().replace(/^\.\//, "")).filter(Boolean);

const defaultExcludes = ["*.lock", "*.sum", "package-lock.json", "yarn.lock", "pnpm-lock.yaml"];
const allExcludes = [...defaultExcludes, ...excludes];
const anticipatedNorm = anticipated.map(p => p.replace(/^\.\//, "").replace(/\/+$/, "")).filter(Boolean);

function isAnticipated(p) {
  return anticipatedNorm.some(a => p === a || p.startsWith(`${a}/`));
}

const unexpected = changed.filter(p => !isAnticipated(p) && !matchesExclude(p, allExcludes));

// --- emit ---------------------------------------------------------------------------

console.log(unexpected.length === 0 ? "DRIFT: clean" : `DRIFT: ${unexpected.length} unexpected file(s)`);
if (unexpected.length > 0) for (const p of unexpected) console.log(`  - ${p}`);
console.log(`SCOPE: git diff ${baseRef}: ${changed.length} file(s) enumerated, ${anticipatedNorm.length} anticipated root(s), ${allExcludes.length} excludes`);

process.exit(unexpected.length === 0 ? 0 : 1);
