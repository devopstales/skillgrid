#!/usr/bin/env node
// test-sdd-workspace.mjs — prove the per-plan SDD workspace scripts resolve
// isolated directories, keep git status clean, and fail loudly on bad input.
//
// Each case builds a throwaway git repo under a mktemp dir, drops in plan
// fixtures, runs sdd-workspace / task-brief / review-package, and asserts
// exit codes and file placement. No fixtures leak into the real repo; the
// temp root is torn down on exit.
//
// Usage:
//   node .agents/skills/execute/scripts/test-sdd-workspace.mjs          # run everything
//   node .agents/skills/execute/scripts/test-sdd-workspace.mjs workspace # run only cases tagged workspace
//
// Exit code: 0 if all pass, 1 if any fail.
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const SCRIPTS_DIR = dirname(fileURLToPath(import.meta.url));
const NODE = process.execPath;
const WORKSPACE = join(SCRIPTS_DIR, "sdd-workspace.mjs");
const BRIEF = join(SCRIPTS_DIR, "task-brief.mjs");
const REVIEW = join(SCRIPTS_DIR, "review-package.mjs");

const FILTER = process.argv[2] || "";

let PASS = 0, FAIL = 0;
const RESULTS = [];

const TMP_ROOT = mkdtempSync(join(tmpdir(), "skillgrid-sdd-workspace-"));
const WORK = join(TMP_ROOT, "work");
// Hermetic git fixtures: the ambient global core.hooksPath (skillgrid guards
// refuse commits on protected branches) must not leak into throwaway repos.
const NO_HOOKS = join(TMP_ROOT, "no-hooks");
mkdirSync(NO_HOOKS, { recursive: true });

function sh(cmd, args, cwd) {
  try {
    return execFileSync(cmd, args, { encoding: "utf8", cwd, stdio: ["ignore", "pipe", "pipe"] });
  } catch (e) {
    return { __error: true, status: e.status ?? 1, out: `${e.stdout || ""}${e.stderr || ""}` };
  }
}

function git(cwd, args) {
  return sh("git", args, cwd);
}

// --- helpers -----------------------------------------------------------------

function newRepo(name) {
  const path = join(WORK, name);
  rmSync(path, { recursive: true, force: true });
  mkdirSync(path, { recursive: true });
  git(path, ["init", "-q", "-b", "sdd-test", "."]);
  git(path, ["config", "user.email", "sdd-test@skillgrid.local"]);
  git(path, ["config", "user.name", "SDD Test"]);
  git(path, ["config", "commit.gpgsign", "false"]);
  git(path, ["config", "core.hooksPath", NO_HOOKS]);
  return path;
}

// write_plan <repo> <relpath> — writes a two-task plan fixture. Task 1's
// section embeds a fenced code block containing a fake "## Task 2" heading
// (a naive heading scan would misread it; the fence-aware extractor must not).
function writePlan(repo, rel) {
  const p = join(WORK, repo, rel);
  mkdirSync(dirname(p), { recursive: true });
  writeFileSync(p, `# Plan: demo change

## Task 1 — first thing

Do the first thing.

\`\`\`
## Task 2 — fake heading inside a fence, not a real task
\`\`\`

## Task 2 — second thing

Do the second thing.
`);
}

function expectRC(name, want, got) {
  if (got === want) { PASS++; RESULTS.push(`PASS  ${name}`); }
  else { FAIL++; RESULTS.push(`FAIL  ${name} (want rc=${want}, got rc=${got})`); }
}
function expectFile(name, path) {
  if (existsSync(path)) { PASS++; RESULTS.push(`PASS  ${name}`); }
  else { FAIL++; RESULTS.push(`FAIL  ${name} (missing file: ${path})`); }
}

// run a .mjs script capturing rc + combined stdout (the scripts print their
// result lines on stdout by design)
function runScript(script, args, cwd) {
  const r = sh(NODE, [script, ...args], cwd);
  return r && r.__error ? { rc: r.status, out: r.out } : { rc: 0, out: r };
}

// --- workspace: distinct dirs per plan ---------------------------------------

function caseWorkspace() {
  if (FILTER && FILTER !== "workspace") return;

  const ws = newRepo("ws");
  writePlan("ws", ".skillgrid/change-specs/2026-09-21-alpha/plan-a.md");
  writePlan("ws", ".skillgrid/change-specs/2026-09-21-beta/plan-b.md");
  git(ws, ["add", "-A"]);
  git(ws, ["commit", "-q", "-m", "chore: fixture plans"]);

  let r = runScript(WORKSPACE, [".skillgrid/change-specs/2026-09-21-alpha/plan-a.md"], ws);
  expectRC("workspace: plan A resolves (rc 0)", 0, r.rc);
  const da = r.out.trim();
  r = runScript(WORKSPACE, [".skillgrid/change-specs/2026-09-21-beta/plan-b.md"], ws);
  expectRC("workspace: plan B resolves (rc 0)", 0, r.rc);
  const db = r.out.trim();

  if (da && db && da !== db) { PASS++; RESULTS.push("PASS  workspace: distinct plans resolve distinct dirs"); }
  else { FAIL++; RESULTS.push(`FAIL  workspace: distinct plans resolve distinct dirs (A=${da} B=${db})`); }

  // resolve via git so symlinked TMPDIRs (macOS /var -> /private/var) compare equal
  const top = sh("git", ["rev-parse", "--show-toplevel"], ws).trimEnd();
  const want = join(top, ".skillgrid", "meta", ".sdd", "2026-09-21-alpha");
  if (da === want) { PASS++; RESULTS.push("PASS  workspace: dir is <root>/.skillgrid/meta/.sdd/<spec-folder>"); }
  else { FAIL++; RESULTS.push(`FAIL  workspace: dir is <root>/.skillgrid/meta/.sdd/<spec-folder> (got ${da})`); }

  if (existsSync(da) && existsSync(db)) { PASS++; RESULTS.push("PASS  workspace: both dirs created"); }
  else { FAIL++; RESULTS.push("FAIL  workspace: both dirs created"); }

  // parent .gitignore keeps git status clean
  const gi = readFileSync(join(ws, ".skillgrid", "meta", ".sdd", ".gitignore"), "utf8");
  if (gi.trim() === "*") { PASS++; RESULTS.push("PASS  workspace: parent .gitignore contains '*'"); }
  else { FAIL++; RESULTS.push(`FAIL  workspace: parent .gitignore contains '*' (got '${gi}')`); }
  writeFileSync(join(da, "progress.md"), "# SDD ledger — plan: plan-a\n");
  writeFileSync(join(da, "task-1-brief.md"), "x\n");
  const porcelain = sh("git", ["status", "--porcelain"], ws);
  if (porcelain.trim() === "") { PASS++; RESULTS.push("PASS  workspace: git status clean with artifacts present"); }
  else { FAIL++; RESULTS.push(`FAIL  workspace: git status clean with artifacts present (got: ${porcelain})`); }
}

// --- workspace: missing plan errors ------------------------------------------

function caseMissing() {
  if (FILTER && FILTER !== "missing") return;

  const miss = newRepo("miss");
  writeFileSync(join(miss, "a.txt"), "a\n");
  git(miss, ["add", "-A"]);
  git(miss, ["commit", "-q", "-m", "chore: init"]);

  let r = runScript(WORKSPACE, ["no-such-plan.md"], miss);
  expectRC("workspace: missing plan file exits 2", 2, r.rc);
  r = runScript(WORKSPACE, [], miss);
  expectRC("workspace: missing arg exits 2", 2, r.rc);
  r = runScript(BRIEF, ["no-such-plan.md", "1"], miss);
  expectRC("task-brief: missing plan file exits 2", 2, r.rc);
  r = runScript(REVIEW, ["no-such-plan.md", "HEAD", "HEAD"], miss);
  expectRC("review-package: missing plan file exits 2", 2, r.rc);

  // no directory may be created on the failure path
  if (!existsSync(join(miss, ".skillgrid", "meta", ".sdd"))) { PASS++; RESULTS.push("PASS  workspace: no dir created for missing plan"); }
  else { FAIL++; RESULTS.push("FAIL  workspace: no dir created for missing plan"); }
}

// --- task-brief: extraction into the plan's own dir ---------------------------

function caseBrief() {
  if (FILTER && FILTER !== "brief") return;

  const brief = newRepo("brief");
  writePlan("brief", ".skillgrid/change-specs/2026-09-21-alpha/plan-a.md");
  writePlan("brief", ".skillgrid/change-specs/2026-09-21-beta/plan-b.md");
  git(brief, ["add", "-A"]);
  git(brief, ["commit", "-q", "-m", "chore: fixture plans"]);

  let r = runScript(BRIEF, [".skillgrid/change-specs/2026-09-21-alpha/plan-a.md", "1"], brief);
  expectRC("task-brief: existing task exits 0", 0, r.rc);
  expectFile("task-brief: default outfile lands in plan A's dir",
    join(brief, ".skillgrid", "meta", ".sdd", "2026-09-21-alpha", "task-1-brief.md"));

  const content = readFileSync(join(brief, ".skillgrid", "meta", ".sdd", "2026-09-21-alpha", "task-1-brief.md"), "utf8");
  if (content.includes("Do the first thing") && content.includes("fake heading inside a fence") && !content.includes("Do the second thing")) {
    PASS++; RESULTS.push("PASS  task-brief: extracts only Task 1, fence-aware");
  } else {
    FAIL++; RESULTS.push("FAIL  task-brief: extracts only Task 1, fence-aware");
  }

  // plan B's brief must not leak into plan A's dir
  r = runScript(BRIEF, [".skillgrid/change-specs/2026-09-21-beta/plan-b.md", "2"], brief);
  expectRC("task-brief: plan B task exits 0", 0, r.rc);
  if (existsSync(join(brief, ".skillgrid", "meta", ".sdd", "2026-09-21-beta", "task-2-brief.md"))
    && !existsSync(join(brief, ".skillgrid", "meta", ".sdd", "2026-09-21-alpha", "task-2-brief.md"))) {
    PASS++; RESULTS.push("PASS  task-brief: plan B brief stays in plan B dir");
  } else {
    FAIL++; RESULTS.push("FAIL  task-brief: plan B brief stays in plan B dir");
  }

  r = runScript(BRIEF, [".skillgrid/change-specs/2026-09-21-alpha/plan-a.md", "99"], brief);
  expectRC("task-brief: missing task exits 3", 3, r.rc);
}

// --- review-package: diff bundle into the plan's own dir ----------------------

function caseReview() {
  if (FILTER && FILTER !== "review") return;

  const review = newRepo("review");
  writePlan("review", ".skillgrid/change-specs/2026-09-21-alpha/plan-a.md");
  git(review, ["add", "-A"]);
  git(review, ["commit", "-q", "-m", "chore: fixture plan"]);
  const base = sh("git", ["rev-parse", "HEAD"], review).trimEnd();
  writeFileSync(join(review, "feat.txt"), "v2\n");
  git(review, ["add", "-A"]);
  git(review, ["commit", "-q", "-m", "feat(x): add feat"]);
  const head = sh("git", ["rev-parse", "HEAD"], review).trimEnd();

  let r = runScript(REVIEW, [".skillgrid/change-specs/2026-09-21-alpha/plan-a.md", base, head], review);
  expectRC("review-package: valid range exits 0", 0, r.rc);

  const shortBase = sh("git", ["rev-parse", "--short", base], review).trimEnd();
  const shortHead = sh("git", ["rev-parse", "--short", head], review).trimEnd();
  const want = join(review, ".skillgrid", "meta", ".sdd", "2026-09-21-alpha", `review-${shortBase}..${shortHead}.diff`);
  expectFile("review-package: default outfile lands in plan's dir", want);

  const body = existsSync(want) ? readFileSync(want, "utf8") : "";
  if (body.includes("## Commits") && body.includes("## Files changed") && body.includes("## Diff") && body.includes("feat.txt")) {
    PASS++; RESULTS.push("PASS  review-package: bundle holds commits, stat, and diff");
  } else {
    FAIL++; RESULTS.push("FAIL  review-package: bundle holds commits, stat, and diff");
  }

  r = runScript(REVIEW, [".skillgrid/change-specs/2026-09-21-alpha/plan-a.md", "deadbeef", head], review);
  expectRC("review-package: bad BASE exits 2", 2, r.rc);
  r = runScript(REVIEW, [".skillgrid/change-specs/2026-09-21-alpha/plan-a.md", base, "deadbeef"], review);
  expectRC("review-package: bad HEAD exits 2", 2, r.rc);
}

// --- ledger convention documented ---------------------------------------------

function caseLedger() {
  if (FILTER && FILTER !== "ledger") return;

  const header = readFileSync(WORKSPACE, "utf8");
  if (header.includes("# SDD ledger — plan: <plan-file-path>")) {
    PASS++; RESULTS.push("PASS  ledger: convention documented in sdd-workspace header");
  } else {
    FAIL++; RESULTS.push("FAIL  ledger: convention documented in sdd-workspace header");
  }
}

// --- run all (respect filter) ------------------------------------------------

caseWorkspace();
caseMissing();
caseBrief();
caseReview();
caseLedger();

// --- report ------------------------------------------------------------------

console.log("");
for (const line of RESULTS) console.log(line);
console.log("");
console.log("========================================");
console.log(`Results: ${PASS} passed, ${FAIL} failed`);
rmSync(TMP_ROOT, { recursive: true, force: true });
process.exit(FAIL === 0 ? 0 : 1);
