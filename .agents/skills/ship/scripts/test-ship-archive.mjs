#!/usr/bin/env node
// test-ship-archive.mjs — prove the archive move: mechanical git mv, empty
// diff -r readback, source gone, state.yaml closed out, and the failure
// paths (usage, already-archived) exit correctly.
//
// Each case builds a throwaway git repo under a mktemp dir, drops in a change
// fixture, runs ship-archive.mjs, and asserts exit codes + filesystem state.
// No fixtures leak into the real repo; the temp root is torn down on exit.
//
// Usage:
//   node .agents/skills/ship/scripts/test-ship-archive.mjs          # run everything
//   node .agents/skills/ship/scripts/test-ship-archive.mjs move      # run only cases tagged move
//
// Exit code: 0 if all pass, 1 if any fail.
import { execFileSync } from "node:child_process";
import { cpSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const SCRIPTS_DIR = dirname(fileURLToPath(import.meta.url));
const ARCHIVE = join(SCRIPTS_DIR, "ship-archive.mjs");
const NODE = process.execPath;
const FILTER = process.argv[2] || "";

let PASS = 0, FAIL = 0;
const RESULTS = [];

const TMP_ROOT = mkdtempSync(join(tmpdir(), "skillgrid-ship-archive-"));
const WORK = join(TMP_ROOT, "work");
const NO_HOOKS = join(TMP_ROOT, "no-hooks");
mkdirSync(NO_HOOKS, { recursive: true });

function git(cwd, args) {
  return execFileSync("git", args, { encoding: "utf8", cwd, stdio: ["ignore", "pipe", "pipe"] });
}

function runArchive(cwd, topic) {
  try {
    const out = execFileSync(NODE, [ARCHIVE, topic], { encoding: "utf8", cwd, stdio: ["ignore", "pipe", "pipe"] });
    return { rc: 0, out };
  } catch (e) {
    return { rc: e.status ?? 1, out: `${e.stdout || ""}${e.stderr || ""}` };
  }
}

// --- helpers -------------------------------------------------------------------

function newRepo(name) {
  const path = join(WORK, name);
  rmSync(path, { recursive: true, force: true });
  mkdirSync(path, { recursive: true });
  git(path, ["init", "-q", "-b", "ship-test", "."]);
  git(path, ["config", "user.email", "ship-test@skillgrid.local"]);
  git(path, ["config", "user.name", "Ship Test"]);
  git(path, ["config", "commit.gpgsign", "false"]);
  git(path, ["config", "core.hooksPath", NO_HOOKS]);
  return path;
}

// write_change <repo> <topic> — a committed change folder with a state.yaml
function writeChange(repo, topic) {
  const dir = join(WORK, repo, ".skillgrid", "change-specs", topic);
  mkdirSync(dir, { recursive: true });
  writeFileSync(join(dir, "briefing.md"), "# Briefing\n\n> **STATUS:** `shipped` (2026-01-01)\n");
  writeFileSync(join(dir, "blueprint.md"), "# Blueprint\n");
  writeFileSync(join(WORK, repo, ".skillgrid", "state.yaml"),
    `schema: skillgrid/state/v2\nchanges:\n  ${topic}:\n    phase: "review"\n    status: in_progress\nprogress:\n  completed_changes: 0\n  blocked_changes: 0\n`);
  git(join(WORK, repo), ["add", "-A"]);
  git(join(WORK, repo), ["commit", "-q", "-m", "chore: fixture change"]);
}

function expectRC(name, want, got) {
  if (got === want) { PASS++; RESULTS.push(`PASS  ${name}`); }
  else { FAIL++; RESULTS.push(`FAIL  ${name} (want rc=${want}, got rc=${got})`); }
}
function expectFile(name, path) {
  if (existsSync(path)) { PASS++; RESULTS.push(`PASS  ${name}`); }
  else { FAIL++; RESULTS.push(`FAIL  ${name} (missing file: ${path})`); }
}
function expectGone(name, path) {
  if (!existsSync(path)) { PASS++; RESULTS.push(`PASS  ${name}`); }
  else { FAIL++; RESULTS.push(`FAIL  ${name} (still exists: ${path})`); }
}
function expectGrep(name, path, needle, invert = false) {
  const hit = existsSync(path) && readFileSync(path, "utf8").includes(needle);
  if (hit !== invert) { PASS++; RESULTS.push(`PASS  ${name}`); }
  else { FAIL++; RESULTS.push(`FAIL  ${name} (${invert ? "unexpected" : "missing"}: ${needle})`); }
}

// --- move: clean archive ---------------------------------------------------------

function caseMove() {
  if (FILTER && FILTER !== "move") return;

  const repo = newRepo("clean");
  writeChange("clean", "2026-01-01-alpha");

  const r = runArchive(repo, "2026-01-01-alpha");
  expectRC("move: clean archive exits 0", 0, r.rc);

  expectFile("move: archived briefing exists",
    join(repo, ".skillgrid", "change-archive", "2026-01-01-alpha", "briefing.md"));
  expectFile("move: archived blueprint exists",
    join(repo, ".skillgrid", "change-archive", "2026-01-01-alpha", "blueprint.md"));
  expectGone("move: source dir is gone",
    join(repo, ".skillgrid", "change-specs", "2026-01-01-alpha"));

  // JSON contract: moved=true, readback empty, used_git_mv=true
  if (r.out.includes('"moved": true') && r.out.includes('"readback": ""') && r.out.includes('"used_git_mv": true')) {
    PASS++; RESULTS.push("PASS  move: JSON reports moved+empty readback+git mv");
  } else {
    FAIL++; RESULTS.push(`FAIL  move: JSON reports moved+empty readback+git mv (got: ${r.out})`);
  }

  expectGrep("move: state.yaml entry removed", join(repo, ".skillgrid", "state.yaml"), "2026-01-01-alpha", true);
  expectGrep("move: completed_changes incremented", join(repo, ".skillgrid", "state.yaml"), "completed_changes: 1");
}

// --- move: content integrity (readback catches a divergent archive) ---------------

function caseContent() {
  if (FILTER && FILTER !== "content") return;

  // the readback is internal (snapshot taken inside the script); the
  // externally observable contract is: archived content == source content,
  // byte for byte, verified against an independent pre-move copy.
  const repo = newRepo("content");
  writeChange("content", "2026-02-02-beta");
  const copyDir = join(WORK, "content-copy", ".skillgrid", "change-specs", "2026-02-02-beta");
  mkdirSync(copyDir, { recursive: true });
  cpSync(join(repo, ".skillgrid", "change-specs", "2026-02-02-beta"), copyDir, { recursive: true });

  const r = runArchive(repo, "2026-02-02-beta");
  expectRC("content: archive exits 0", 0, r.rc);

  let identical = false;
  try {
    const diff = execFileSync("diff", ["-r", copyDir, join(repo, ".skillgrid", "change-archive", "2026-02-02-beta")], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
    identical = diff.trim() === "";
  } catch {
    identical = false;
  }
  if (identical) { PASS++; RESULTS.push("PASS  content: archived bytes match the pre-move copy"); }
  else { FAIL++; RESULTS.push("FAIL  content: archived bytes match the pre-move copy"); }
}

// --- usage: missing topic / no change-specs ---------------------------------------

function caseUsage() {
  if (FILTER && FILTER !== "usage") return;

  const repo = newRepo("usage");
  let r = runArchive(repo, "no-such-topic");
  expectRC("usage: missing topic exits 2", 2, r.rc);
  r = runUsageNoArg(repo);
  expectRC("usage: missing arg exits 2", 2, r.rc);

  // a failed move must not touch state.yaml
  writeChange("usage", "2026-03-03-gamma");
  r = runArchive(repo, "no-such-topic");
  expectRC("usage: missing topic exits 2 (repo populated)", 2, r.rc);
  expectGrep("usage: failed move leaves state.yaml untouched", join(repo, ".skillgrid", "state.yaml"), "2026-03-03-gamma");
}

function runUsageNoArg(repo) {
  try {
    const out = execFileSync(NODE, [ARCHIVE], { encoding: "utf8", cwd: repo, stdio: ["ignore", "pipe", "pipe"] });
    return { rc: 0, out };
  } catch (e) {
    return { rc: e.status ?? 1, out: `${e.stdout || ""}${e.stderr || ""}` };
  }
}

// --- usage: archive target already exists ------------------------------------------

function caseExists() {
  if (FILTER && FILTER !== "exists") return;

  const repo = newRepo("exists");
  writeChange("exists", "2026-04-04-delta");
  mkdirSync(join(repo, ".skillgrid", "change-archive", "2026-04-04-delta"), { recursive: true });

  const r = runArchive(repo, "2026-04-04-delta");
  expectRC("exists: pre-occupied archive target exits 2", 2, r.rc);
  expectFile("exists: source untouched on refusal",
    join(repo, ".skillgrid", "change-specs", "2026-04-04-delta", "briefing.md"));
  expectGrep("exists: state.yaml untouched on refusal", join(repo, ".skillgrid", "state.yaml"), "2026-04-04-delta");
}

// --- move: non-git fallback (plain mv, still mechanical) -----------------------------

function caseNogit() {
  if (FILTER && FILTER !== "nogit") return;

  // a plain directory (no git) — the script must still move + readback,
  // using the rename fallback and reporting used_git_mv=false
  const dir = join(WORK, "nogit");
  rmSync(dir, { recursive: true, force: true });
  const topicDir = join(dir, ".skillgrid", "change-specs", "2026-05-05-eps");
  mkdirSync(topicDir, { recursive: true });
  writeFileSync(join(topicDir, "briefing.md"), "# B\n");

  const r = runArchive(dir, "2026-05-05-eps");
  expectRC("nogit: plain-dir archive exits 0", 0, r.rc);
  expectFile("nogit: archived file exists", join(dir, ".skillgrid", "change-archive", "2026-05-05-eps", "briefing.md"));
  expectGone("nogit: source dir is gone", join(dir, ".skillgrid", "change-specs", "2026-05-05-eps"));
  if (r.out.includes('"used_git_mv": false')) {
    PASS++; RESULTS.push("PASS  nogit: JSON reports the rename fallback");
  } else {
    FAIL++; RESULTS.push(`FAIL  nogit: JSON reports the rename fallback (got: ${r.out})`);
  }
}

// --- run all (respect filter) -------------------------------------------------------

caseMove();
caseContent();
caseUsage();
caseExists();
caseNogit();

// --- report ------------------------------------------------------------------------

console.log("");
for (const line of RESULTS) console.log(line);
console.log("");
console.log("========================================");
console.log(`Results: ${PASS} passed, ${FAIL} failed`);
rmSync(TMP_ROOT, { recursive: true, force: true });
process.exit(FAIL === 0 ? 0 : 1);
