#!/usr/bin/env node
// test-map-scope.mjs — prove the repo-mapping scope computation:
// discovery + pruning, refresh diffing, post-order page list, gate verdict,
// partitioning, and manifest merge (merge moves head for good returns and
// keeps old head for failed ones).
//
// Hermetic: builds throwaway git repos under mkdtemp; torn down on exit.
//
// Usage: node test-map-scope.mjs
// Exit code: 0 if all pass, 1 if any fail.

import { execFileSync } from "node:child_process";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const script = join(here, "map-scope.mjs");

let pass = 0, fail = 0;
function check(name, cond, detail = "") {
  if (cond) { pass++; console.log(`ok   ${name}`); }
  else { fail++; console.log(`FAIL ${name}${detail ? ` — ${detail}` : ""}`); }
}

const work = mkdtempSync(join(tmpdir(), "skillgrid-map-scope-"));

function git(cwd, ...args) {
  // --no-verify: the temp repos inherit this repo's commit-msg hook
  // (conventional-commits), which a fixture subject like "initial" fails.
  return execFileSync("git", ["-c", "core.hooksPath=/dev/null", ...args], { cwd, encoding: "utf8" }).trimEnd();
}

function makeRepo(layout) {
  const root = join(work, `repo-${Date.now()}-${Math.random().toString(36).slice(2)}`);
  mkdirSync(root, { recursive: true });
  git(root, "init", "-q");
  git(root, "config", "user.email", "t@t");
  git(root, "config", "user.name", "t");
  for (const [rel, content] of Object.entries(layout)) {
    const p = join(root, rel);
    mkdirSync(dirname(p), { recursive: true });
    writeFileSync(p, content);
  }
  git(root, "add", "-A");
  git(root, "commit", "-q", "-m", "initial");
  return root;
}

function run(root, args) {
  try {
    const out = execFileSync("node", [script, ...args], { cwd: root, encoding: "utf8" });
    return { code: 0, json: JSON.parse(out), out };
  } catch (e) {
    return { code: e.status ?? 2, json: null, out: e.stdout?.toString() ?? "", err: e.stderr?.toString() ?? "" };
  }
}

function sha(root) {
  return git(root, "rev-parse", "HEAD");
}

// --- repo shape -------------------------------------------------------------
// root:
//   pkgA/a.py        (code)
//   pkgB/sub/b.py    (code, nested)
//   tests/t_a.py     (test-only dir — pruned)
//   docs/readme.md   (ignored dir)
//   node_modules/x.js (ignored dir)
//   cfg/app.json     (no source — pruned)
const LAYOUT = {
  "pkgA/a.py": "def alpha():\n    return 1\n",
  "pkgB/sub/b.py": "def beta():\n    return 2\n",
  "tests/t_a.py": "def test_alpha():\n    pass\n",
  "docs/readme.md": "# docs\n",
  "node_modules/x.js": "module.exports = 1;\n",
  "cfg/app.json": "{}",
};

// 1. full mode on a fresh manifest: exactly the two code dirs, post-order
{
  const root = makeRepo(LAYOUT);
  const base = sha(root);
  const r = run(root, ["--base", base, "--mode", "full"]);
  check("full: exit 0", r.code === 0, r.err);
  const paths = (r.json?.pages ?? []).map((p) => p.path);
  // pkgB is a blueprint page (no direct source, child has code)
  check("full: discovers pkgA + pkgB + pkgB/sub", JSON.stringify([...paths].sort()) === JSON.stringify(["pkgA", "pkgB", "pkgB/sub"].sort()), JSON.stringify(paths));
  check("full: prunes tests/docs/node_modules/cfg", !paths.some((p) => p.startsWith("tests") || p.startsWith("docs") || p.startsWith("node_modules") || p.startsWith("cfg")));
  const types = Object.fromEntries(r.json.pages.map((p) => [p.path, p.type]));
  check("full: pkgA is component", types["pkgA"] === "component");
  check("full: pkgB is blueprint", types["pkgB"] === "blueprint");
  check("full: pkgB/sub is component", types["pkgB/sub"] === "component");
  check("full: post-order (child before parent)", paths.indexOf("pkgB/sub") < paths.indexOf("pkgB"), JSON.stringify(paths));
  check("full: gate none (3 units)", r.json.gate === "none" && r.json.units === 3);
}

// 2. refresh: no changes since base -> no pages (all in manifest? no — fresh
//    manifest means missing-page). Use merge first to populate, then no-change
//    refresh -> empty.
{
  const root = makeRepo(LAYOUT);
  const base = sha(root);
  // populate manifest via merge
  const returns = [
    { path: "pkgA", interface: ["[a.py:alpha]"], invariants: ["None"], side_effects: ["None"], summary: "pkgA" },
    { path: "pkgB", interface: [], invariants: ["None"], side_effects: ["None"], summary: "pkgB (blueprint)" },
    { path: "pkgB/sub", interface: ["[b.py:beta]"], invariants: ["None"], side_effects: ["None"], summary: "pkgB/sub" },
  ];
  const rf = join(root, "returns.json");
  writeFileSync(rf, JSON.stringify({ subtree_index: returns }));
  const m = run(root, ["--merge", rf]);
  check("merge: exit 0", m.code === 0, m.err);
  check("merge: merged 3", m.json?.merged === 3, JSON.stringify(m.json));
  check("merge: kept empty", Array.isArray(m.json?.kept) && m.json.kept.length === 0);
  let manifest = null;
  let manifestErr = null;
  try {
    manifest = JSON.parse(readFileSync(join(root, ".skillgrid/projectwiki/map.json"), "utf8"));
  } catch (e) {
    manifestErr = e.message;
  }
  check("merge: manifest readable", manifest !== null, manifestErr);
  if (manifest) {
    check("merge: manifest head = HEAD", manifest.head === sha(root));
    check("merge: cached interface", manifest.pages.pkgA?.interface?.[0] === "[a.py:alpha]");
  }

  // no-change refresh -> no pages
  const r2 = run(root, ["--base", base, "--mode", "refresh"]);
  check("refresh no-change: exit 0", r2.code === 0, r2.err);
  check("refresh no-change: 0 pages (all in manifest)", r2.json?.pages?.length === 0, JSON.stringify(r2.json?.pages));

  // change one file in pkgA -> pkgA re-scouted only
  writeFileSync(join(root, "pkgA/a.py"), "def alpha():\n    return 2\n\ndef gamma():\n    return 3\n");
  git(root, "add", "-A");
  git(root, "commit", "-q", "-m", "touch pkgA");
  const r3 = run(root, ["--base", base, "--mode", "refresh"]);
  check("refresh changed: exit 0", r3.code === 0, r3.err);
  const p3 = r3.json?.pages ?? [];
  const paths3 = p3.map((p) => p.path);
  // pkgA is its own blueprint (no code children), so it is both the changed
  // component and the parent — reason "changed" wins over "parent-of-changed".
  check("refresh changed: only pkgA", JSON.stringify(paths3) === JSON.stringify(["pkgA"]), JSON.stringify(paths3));
  check("refresh changed: pkgA reason changed", p3.find((p) => p.path === "pkgA")?.reason === "changed");
  check("refresh changed: changed_files lists a.py", p3.find((p) => p.path === "pkgA")?.changed_files?.includes("pkgA/a.py"));

  // nested change: pkgB/sub -> both pkgB/sub (component) and pkgB (blueprint
  // ancestor) appear, child before parent
  writeFileSync(join(root, "pkgB/sub/b.py"), "def beta():\n    return 9\n");
  git(root, "add", "-A");
  git(root, "commit", "-q", "-m", "touch pkgB/sub");
  const r4 = run(root, ["--base", base, "--mode", "refresh"]);
  const p4 = r4.json?.pages ?? [];
  const paths4 = p4.map((p) => p.path);
  check("refresh nested: pkgB/sub + pkgB + pkgA listed", JSON.stringify(paths4) === JSON.stringify(["pkgB/sub", "pkgB", "pkgA"]), JSON.stringify(paths4));
  check("refresh nested: pkgB/sub reason changed", p4.find((p) => p.path === "pkgB/sub")?.reason === "changed");
  check("refresh nested: pkgA reason changed (cumulative base)", p4.find((p) => p.path === "pkgA")?.reason === "changed");
  check("refresh nested: pkgB is blueprint", p4.find((p) => p.path === "pkgB")?.type === "blueprint");
  check("refresh nested: pkgB reason parent-of-changed", p4.find((p) => p.path === "pkgB")?.reason === "parent-of-changed");
  check("refresh nested: child before parent", paths4.indexOf("pkgB/sub") < paths4.indexOf("pkgB"));
}

// 3. merge keeps old head for failed scouts
{
  const root = makeRepo(LAYOUT);
  writeFileSync(join(root, "pkgA/a.py"), "def alpha():\n    return 5\n");
  git(root, "add", "-A");
  git(root, "commit", "-q", "-m", "change");
  const returns = [
    { path: "pkgA", interface: ["[a.py:alpha]"], invariants: ["None"], side_effects: ["None"], summary: "ok" },
    { path: "pkgB/sub", status: "failed" },
  ];
  const rf = join(root, "returns.json");
  writeFileSync(rf, JSON.stringify({ subtree_index: returns }));
  const m = run(root, ["--merge", rf]);
  check("merge failed: merged 1 kept 1", m.json?.merged === 1 && m.json?.kept?.length === 1 && m.json.kept[0] === "pkgB/sub", JSON.stringify(m.json));
  let manifest = null;
  let manifestErr = null;
  try {
    manifest = JSON.parse(readFileSync(join(root, ".skillgrid/projectwiki/map.json"), "utf8"));
  } catch (e) {
    manifestErr = e.message;
  }
  check("merge failed: manifest readable", manifest !== null, manifestErr);
  if (manifest) {
    check("merge failed: pkgB/sub has no entry (old state kept)", !manifest.pages["pkgB/sub"]);
  }
}

// 4. gate: >15 units -> ask
{
  const layout = {};
  for (let i = 0; i < 16; i++) layout[`p${i}/f.py`] = `def f${i}():\n    return ${i}\n`;
  const root = makeRepo(layout);
  const base = sha(root);
  const r = run(root, ["--base", base, "--mode", "full"]);
  check("gate: 16 units -> ask", r.json?.gate === "ask" && r.json?.units === 16, JSON.stringify({ gate: r.json?.gate, units: r.json?.units }));
  check("gate: concurrency cap 5", r.json?.concurrency_cap === 5);
}

// 5. partitioning: 8 siblings at depth 1 -> 4 big + fallback marker
{
  const layout = {};
  for (let i = 0; i < 8; i++) layout[`p${i}/f.py`] = `def f${i}():\n    return ${i}\n`;
  const root = makeRepo(layout);
  const base = sha(root);
  const r = run(root, ["--base", base, "--mode", "full"]);
  const parts = r.json?.partitions ?? [];
  check("partition: 8 siblings -> 4 + fallback", parts.length === 5 && parts[4] === "/*fallback*/", JSON.stringify(parts));
}

// 6. usage errors exit 2
{
  const root = makeRepo(LAYOUT);
  const r = run(root, ["--mode", "full"]);
  check("usage: missing base exits 2", r.code === 2);
  const r2 = run(root, ["--base", "not-a-sha"]);
  check("usage: bad base exits 2", r2.code === 2);
}

rmSync(work, { recursive: true, force: true });
console.log("");
console.log(`map-scope: ${pass} passed, ${fail} failed`);
process.exit(fail === 0 ? 0 : 1);
