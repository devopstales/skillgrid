#!/usr/bin/env node
// map-scope.mjs — compute the repo-mapping scope: which projectwiki
// architecture/ pages to (re)generate for this run, and at what cost.
//
// Deterministic layer of the repo-mapping rule
// (_shared/rules/repo-mapping.md). The agent decides whether to run a map;
// this script owns everything after: discovery, pruning, diffing against the
// manifest, post-order page list, gate verdict, and manifest bookkeeping.
// The scout (judgment layer) runs under _shared/rules/scout-contract.md.
//
// Usage:
//   node map-scope.mjs --base <sha> [--mode refresh|full] [--root <dir>] [--json]
//     --base    commit to diff against (the change's merge base). For a full
//               map on a populated manifest, pass the oldest mapped commit.
//     --mode    refresh (default): only directories whose files changed since
//               --base, plus their blueprint ancestors and missing pages.
//               full: every code directory in the tree.
//     --root    repo root (default: git toplevel from cwd).
//     --json    print the scope JSON even on usage errors (diagnostics).
//
//   node map-scope.mjs --merge <scout-returns.json> [--root <dir>]
//     Merge scout returns into the manifest: for each dir with a valid
//     return, head moves to the current HEAD and the return is cached.
//     A missing/failed dir keeps its old head (next refresh re-scouts it).
//
// Reads/writes: .skillgrid/projectwiki/map.json  (the manifest — the map's
// last-known state; atomic write; git-ignored via the .skillgitignore the
// script drops next to it)
//
// Output (stdout, one JSON object on scope):
//   {
//     "mode": "refresh" | "full",
//     "base": "<sha>",
//     "head": "<sha>",
//     "affected": ["<dir>", ...],        // code dirs whose files changed
//     "pages": [                         // post-order: children before parents
//       {"path": "<dir>", "type": "component" | "blueprint",
//        "reason": "changed" | "parent-of-changed" | "missing-page" | "full",
//        "changed_files": ["...", ...],  // component + changed only
//        "manifest_head": "<sha|null>"}
//     ],
//     "gate": "none" | "ask",            // ask when units > 15
//     "units": N,                        // parallel units = len(pages)
//     "concurrency_cap": 5,
//     "partitions": [["<dir>", ...], ...] // depth-1..3 partition roots
//   }
//
// On merge: {"merged": N, "kept": ["<dir>", ...], "head": "<sha>"}
//
// Exit codes: 0 = scope computed or merge applied (gate may be "ask"),
// 2 = usage / not a git repo / bad commit.
//
// Pruning (both modes): ignore symlinks; ignore .git, node_modules, dist,
// build, vendor, third_party, .skillgrid, .agents, docs, site; honor
// .gitignore + .egmignore globs (best effort — shell-style * ? [seq]); a
// "code directory" has a recognized source file directly, or a code
// directory below it; test-only directories (named tests/testing/testdata,
// or containing only test-named source files) are pruned.

import { existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync, renameSync } from "node:fs";
import { dirname, join } from "node:path";
import { execFileSync } from "node:child_process";

const CONCURRENCY_CAP = 5;
const GATE_THRESHOLD = 15;

const IGNORED_DIRS = new Set([
  ".git", ".hg", ".svn", "node_modules", "dist", "build", "vendor",
  "third_party", ".skillgrid", ".agents", ".pi", "docs", "site",
  "__pycache__", ".venv", "venv", ".next", ".cache", "target",
]);

const SOURCE_EXTS = new Set([
  ".c", ".cc", ".cpp", ".cxx", ".h", ".hh", ".hpp",
  ".rs", ".go", ".zig", ".lean", ".f", ".f90", ".f95", ".f03", ".f08", ".hs",
  ".java", ".kt", ".scala", ".cs", ".razor", ".cshtml", ".fs", ".vb",
  ".py", ".rb", ".php", ".lua", ".sh", ".bash", ".ps1", ".psm1", ".psd1",
  ".r", ".R", ".ex", ".exs",
  ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".swift", ".m", ".dart",
  ".sql", ".proto", ".graphql", ".gql", ".thrift", ".prisma",
  ".tf", ".hcl", ".cue",
]);

const TEST_NAME_RE = /^(_test\.|^test_|\.spec\.|Test\.$)/;
const TEST_DIR_NAMES = new Set(["tests", "testing", "testdata"]);

function fail(msg, code = 2) {
  console.error(msg);
  process.exit(code);
}

function git(root, ...args) {
  return execFileSync("git", args, { cwd: root, encoding: "utf8" }).trimEnd();
}

function isSourceFile(name) {
  const dot = name.lastIndexOf(".");
  if (dot < 0) return false;
  return SOURCE_EXTS.has(name.slice(dot));
}

function isTestFile(name) {
  return TEST_NAME_RE.test(name) && isSourceFile(name);
}

function globToRegExp(glob) {
  // Best-effort shell-style glob: * (no /), ? (no /), [seq], ** (any).
  let re = "";
  for (let i = 0; i < glob.length; i++) {
    const c = glob[i];
    if (c === "*") {
      if (glob[i + 1] === "*") { re += ".*"; i++; }
      else re += "[^/]*";
    } else if (c === "?") re += "[^/]";
    else if (c === "[") {
      const end = glob.indexOf("]", i + 1);
      re += end > i ? `[${glob.slice(i + 1, end)}]` : "\\[";
      i = end;
    } else re += c.replace(/[.+^${}()|\\]/g, "\\$&");
  }
  return new RegExp(`^${re}$`);
}

function loadIgnores(root) {
  const patterns = [];
  for (const f of [".gitignore", ".egmignore"]) {
    const p = join(root, f);
    if (!existsSync(p)) continue;
    for (const line of readFileSync(p, "utf8").split("\n")) {
      const g = line.trim();
      if (!g || g.startsWith("#")) continue;
      patterns.push(globToRegExp(g.startsWith("!") ? g.slice(1) : g));
    }
  }
  return (name) => patterns.some((re) => re.test(name));
}

// BFS discovery, level by level (never a recursive walk). Returns a map:
// dir (relative, "" = root) -> { files: [names], children: [dirs] }.
function discover(root) {
  const isIgnored = loadIgnores(root);
  const tree = new Map([["", { files: [], children: [] }]]);
  const queue = [""];
  while (queue.length) {
    const d = queue.shift();
    const node = tree.get(d);
    const abs = join(root, d);
    let entries;
    try {
      entries = readdirSync(abs, { withFileTypes: true });
    } catch {
      continue;
    }
    for (const e of entries) {
      if (e.name.startsWith(".")) continue;
      if (e.isSymbolicLink()) continue; // symlinks: ignore (no recursion)
      const rel = d ? `${d}/${e.name}` : e.name;
      if (e.isDirectory()) {
        if (IGNORED_DIRS.has(e.name) || isIgnored(rel)) continue;
        node.children.push(rel);
        tree.set(rel, { files: [], children: [] });
        queue.push(rel);
      } else if (e.isFile()) {
        if (!isIgnored(rel)) node.files.push(e.name);
      }
    }
  }
  return tree;
}

// Bottom-up prune: a dir survives if it (or any descendant) has a
// non-test source file, and it is not test-only itself.
function prune(tree) {
  const codeDirs = new Set();
  const postOrder = [];
  const walk = (d) => {
    const node = tree.get(d);
    for (const c of node.children) walk(c);
    postOrder.push(d);
  };
  walk("");
  // childHasCode(c) walks up from a direct child to any descendant code dir:
  // post-order guarantees every descendant of c is already decided.
  const childHasCode = (c) => {
    let cur = c;
    while (cur !== undefined) {
      if (codeDirs.has(cur)) return true;
      const i = cur.lastIndexOf("/");
      if (i < 0) cur = undefined;
      else cur = cur.slice(0, i);
    }
    return false;
  };
  for (const d of postOrder) {
    const node = tree.get(d);
    const ownSource = node.files.filter(isSourceFile);
    const isTestDir = TEST_DIR_NAMES.has(d.split("/").pop());
    const childCode = node.children.some(childHasCode);
    // keep if a non-test source file exists here (not in a test-named dir)
    // or anywhere below.
    const ownCode = !isTestDir && ownSource.some((f) => !isTestFile(f));
    if (ownCode || childCode) codeDirs.add(d);
  }
  return codeDirs;
}

function loadManifest(root) {
  const p = join(root, ".skillgrid", "projectwiki", "map.json");
  if (!existsSync(p)) return { head: null, pages: {} };
  try {
    const m = JSON.parse(readFileSync(p, "utf8"));
    return { head: m.head ?? null, pages: m.pages ?? {} };
  } catch {
    fail(`manifest corrupt: ${p}`);
  }
}

function saveManifest(root, manifest) {
  const dir = join(root, ".skillgrid", "projectwiki");
  mkdirSync(dir, { recursive: true });
  const p = join(dir, "map.json");
  const tmp = `${p}.tmp.${process.pid}`;
  writeFileSync(tmp, `${JSON.stringify(manifest, null, 2)}\n`);
  renameSync(tmp, p);
}

// Post-order: children before parents; "" (root) always last.
function postOrder(codeDirs) {
  const out = [...codeDirs].sort((a, b) => b.split("/").length - a.split("/").length || b.localeCompare(a));
  return out.filter((d) => d !== "");
}

function computeScope(root, base, mode) {
  const head = git(root, "rev-parse", "HEAD");
  const tree = discover(root);
  const codeDirs = prune(tree);
  const manifest = loadManifest(root);
  const allDirs = postOrder(codeDirs);

  const pages = [];
  if (mode === "full") {
    for (const d of allDirs) {
      const node = tree.get(d);
      const isComponent = node.files.some(isSourceFile);
      pages.push({
        path: d,
        type: isComponent ? "component" : "blueprint",
        reason: "full",
        manifest_head: manifest.pages[d]?.head ?? null,
      });
    }
    return finish(base, head, mode, [], pages, codeDirs);
  }
  // refresh: allDirs is already post-order (children before parents)

  // refresh: diff vs base
  let changed;
  try {
    changed = git(root, "diff", "--name-only", `${base}..HEAD`);
  } catch {
    fail(`cannot diff against base commit: ${base}`);
  }
  const changedFiles = changed ? changed.split("\n").filter(Boolean) : [];
  const affectedMap = new Map(); // dir -> changed source files
  for (const f of changedFiles) {
    if (!isSourceFile(f)) continue;
    const dir = dirname(f);
    const d = dir === "." ? "" : dir;
    // Affected dir is the file's own directory — page shapes are
    // per-directory, so the file's dir is re-scouted; "" (root file) marks
    // the root blueprint page.
    affectedMap.set(d, (affectedMap.get(d) ?? []).concat([f]));
  }
  const affected = [...affectedMap.keys()].filter((d) => d !== "");

  const changedSet = new Set(affected);
  // ancestors of changed dirs must re-render (blueprint parents)
  for (const d of affected) {
    let i = d.lastIndexOf("/");
    while (i >= 0) {
      const parent = d.slice(0, i);
      changedSet.add(parent);
      i = parent.lastIndexOf("/");
    }
  }

  for (const d of allDirs) {
    const node = tree.get(d);
    const isComponent = node.files.some(isSourceFile);
    const inManifest = manifest.pages[d];
    if (changedSet.has(d)) {
      pages.push({
        path: d,
        type: isComponent ? "component" : "blueprint",
        reason: affectedMap.has(d) ? "changed" : "parent-of-changed",
        changed_files: affectedMap.get(d) ?? [],
        manifest_head: inManifest?.head ?? null,
      });
    } else if (!inManifest) {
      pages.push({
        path: d,
        type: isComponent ? "component" : "blueprint",
        reason: "missing-page",
        manifest_head: null,
      });
    }
  }
  // post-order within the refresh set too (children before parents)
  pages.sort((a, b) => b.path.split("/").length - a.path.split("/").length || b.path.localeCompare(a.path));
  return finish(base, head, mode, affected, pages, codeDirs);
}

function finish(base, head, mode, affected, pages, allDirs) {
  const units = pages.length;
  const gate = units > GATE_THRESHOLD ? "ask" : "none";
  // Partitioning: depth 1 first; <3 nodes -> deeper (max 3); >5 -> largest
  // branches into 3-4 partitions + 1 fallback.
  const partitions = partition(allDirs);
  return {
    mode,
    base,
    head,
    affected,
    pages,
    gate,
    units,
    concurrency_cap: CONCURRENCY_CAP,
    partitions,
  };
}

function partition(codeDirs) {
  const byDepth = new Map();
  for (const d of codeDirs) {
    const depth = d.split("/").length; // root's direct children = 1
    if (depth > 3) continue;
    if (!byDepth.has(depth)) byDepth.set(depth, []);
    byDepth.get(depth).push(d);
  }
  let depth = 1;
  let roots;
  while (depth <= 3) {
    const nodes = byDepth.get(depth) ?? [];
    // only dirs that are actually in the mapped set
    const mapped = nodes.filter((n) => codeDirs.has(n));
    if (mapped.length >= 3 && mapped.length <= 5) { roots = mapped; break; }
    if (mapped.length > 5) {
      // group largest branches into 3-4 partitions + fallback
      const sizes = mapped.map((n) => ({ n, size: countDescendants(codeDirs, n) }));
      sizes.sort((a, b) => b.size - a.size);
      const big = sizes.slice(0, 4).map((s) => s.n);
      const rest = sizes.slice(4).map((s) => s.n);
      roots = [...big];
      if (rest.length) {
        // fallback partition: the smallest of `big` absorbs none; use its
        // parent? no — rest are siblings, so the partition root is the common
        // parent (depth-1 ancestor), which may already be in `big`. Mark by
        // returning a synthetic marker.
        roots.push("/*fallback*/");
      }
      break;
    }
    depth++;
  }
  if (!roots || roots.length < 3) roots = (byDepth.get(1) ?? []).filter((n) => codeDirs.has(n));
  return roots.length ? roots : [""];
}

function countDescendants(codeDirs, prefix) {
  let n = 0;
  for (const d of codeDirs) {
    if (d === prefix || d.startsWith(`${prefix}/`)) n++;
  }
  return n;
}

function mergeScoutReturns(root, returnsFile) {
  const manifest = loadManifest(root);
  const head = git(root, "rev-parse", "HEAD");
  let data;
  try {
    data = JSON.parse(readFileSync(returnsFile, "utf8"));
  } catch (e) {
    fail(`cannot read scout returns: ${e.message}`);
  }
  const returns = data.subtree_index ?? data;
  if (!Array.isArray(returns)) fail("scout returns must be an array or {subtree_index: [...]}");
  let merged = 0;
  const kept = [];
  for (const r of returns) {
    if (!r?.path) continue;
    const d = r.path.replace(/^\.\//, "");
    if (r.status === "failed" || !Array.isArray(r.interface)) {
      kept.push(d); // keep old head — next refresh re-scouts
      continue;
    }
    manifest.pages[d] = {
      head,
      summary: r.summary ?? "",
      interface: r.interface,
      invariants: r.invariants ?? ["None"],
      side_effects: r.side_effects ?? ["None"],
    };
    merged++;
  }
  manifest.head = head;
  saveManifest(root, manifest);
  console.log(JSON.stringify({ merged, kept, head }));
}

function parseArgs(argv) {
  const args = { mode: "refresh" };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--mode") args.mode = argv[++i];
    else if (a === "--base") args.base = argv[++i];
    else if (a === "--root") args.root = argv[++i];
    else if (a === "--merge") args.merge = argv[++i];
    else if (a === "--json") args.json = true;
    else if (a === "--help" || a === "-h") args.help = true;
  }
  return args;
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  if (args.help) {
    console.error("usage: node map-scope.mjs --base <sha> [--mode refresh|full] [--root <dir>]");
    console.error("       node map-scope.mjs --merge <scout-returns.json> [--root <dir>]");
    process.exit(2);
  }
  const root = args.root ?? git(process.cwd(), "rev-parse", "--show-toplevel");
  try {
    git(root, "rev-parse", "--git-dir");
  } catch {
    fail(`not a git working tree: ${root}`);
  }
  if (args.merge) {
    mergeScoutReturns(root, args.merge);
    return;
  }
  if (!args.base) fail("missing --base <sha> (scope mode)");
  if (!["refresh", "full"].includes(args.mode)) fail(`bad --mode: ${args.mode}`);
  try {
    git(root, "rev-parse", `${args.base}^{commit}`);
  } catch {
    fail(`bad base commit: ${args.base}`);
  }
  const scope = computeScope(root, args.base, args.mode);
  console.log(JSON.stringify(scope, null, 2));
}

main();
