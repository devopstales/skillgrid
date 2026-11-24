#!/usr/bin/env node
// test-ship-drift-check.mjs — prove the structure drift check: clean diffs
// under the anticipated set exit 0; a file outside it exits 1 and is listed;
// uncommitted worktree changes are counted; default and custom excludes work;
// unknown base refs and usage errors exit 2 with git's stderr / usage text.
//
// Hermetic: builds throwaway git repos under mkdtemp; torn down on exit.
//
// Usage: node test-ship-drift-check.mjs
// Exit code: 0 if all pass, 1 if any fail.
import { execFileSync } from "node:child_process";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const script = join(here, "ship-drift-check.mjs");

const GITENV = {
  ...process.env,
  GIT_AUTHOR_NAME: "t", GIT_AUTHOR_EMAIL: "t@example.com",
  GIT_COMMITTER_NAME: "t", GIT_COMMITTER_EMAIL: "t@example.com",
  // hermetic: bypass the host's global hooksPath so fixture commits use no hooks
  GIT_CONFIG_COUNT: "1", GIT_CONFIG_KEY_0: "core.hooksPath", GIT_CONFIG_VALUE_0: "",
};

let pass = 0, fail = 0;
const work = mkdtempSync(join(tmpdir(), "skillgrid-ship-drift-"));

function run(args, root) {
  try {
    const out = execFileSync("node", [script, ...args], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"], cwd: root });
    return { rc: 0, out };
  } catch (e) {
    return { rc: e.status ?? 1, out: `${e.stdout || ""}${e.stderr || ""}` };
  }
}

function check(label, haystack, needle) {
  if (haystack.includes(needle)) { console.log(`  ok: ${label}`); pass++; }
  else { console.log(`  FAIL: ${label} (missing: ${needle})`); fail++; }
}
function notcheck(label, haystack, needle) {
  if (haystack.includes(needle)) { console.log(`  FAIL: ${label} (unexpected: ${needle})`); fail++; }
  else { console.log(`  ok: ${label}`); pass++; }
}

function git(cwd, ...args) {
  return execFileSync("git", args, { cwd, encoding: "utf8", env: GITENV, stdio: ["ignore", "pipe", "pipe"] });
}

// repo with: commit1 = src/a.txt + package-lock.json (tag `base`),
// commit2 = docs/c.txt added; then the caller mutates the worktree
function makeRepo(name) {
  const repo = join(work, name);
  mkdirSync(join(repo, "src"), { recursive: true });
  writeFileSync(join(repo, "src", "a.txt"), "a\n");
  writeFileSync(join(repo, "package-lock.json"), "{}\n");
  git(repo, "init", "-q", "-b", "main");
  git(repo, "add", "-A");
  git(repo, "commit", "-q", "-m", "base");
  git(repo, "tag", "base");
  mkdirSync(join(repo, "docs"), { recursive: true });
  writeFileSync(join(repo, "docs", "c.txt"), "c\n");
  git(repo, "add", "-A");
  git(repo, "commit", "-q", "-m", "add docs");
  return repo;
}

try {
  // --- clean: every changed file under the anticipated dirs --------------------
  {
    const repo = makeRepo("clean");
    writeFileSync(join(repo, "src", "a.txt"), "a-modified\n"); // uncommitted
    const r = run(["check", "base", "--anticipated", "src", "docs"], repo);
    if (r.rc === 0) { console.log("ok: clean exit 0"); pass++; }
    else { console.log(`FAIL: clean (exit ${r.rc})\n${r.out}`); fail++; }
    check("clean: DRIFT", r.out, "DRIFT: clean");
    check("clean: SCOPE", r.out, "SCOPE: git diff base: 2 file(s) enumerated, 2 anticipated root(s), 5 excludes");
  }

  // --- drift: one file outside the anticipated set ----------------------------
  {
    const repo = makeRepo("drift");
    writeFileSync(join(repo, "src", "a.txt"), "a-modified\n"); // uncommitted
    const r = run(["check", "base", "--anticipated", "src"], repo);
    if (r.rc === 1) { console.log("ok: drift exit 1"); pass++; }
    else { console.log(`FAIL: drift (exit ${r.rc})\n${r.out}`); fail++; }
    check("drift: hit count", r.out, "DRIFT: 1 unexpected file(s)");
    check("drift: listed", r.out, "  - docs/c.txt");
    check("drift: SCOPE", r.out, "SCOPE: git diff base: 2 file(s) enumerated, 1 anticipated root(s), 5 excludes");
    notcheck("drift: src not listed", r.out, "  - src/a.txt");
  }

  // --- anticipated file path (not just dir) ------------------------------------
  {
    const repo = makeRepo("file-anticipated");
    const r = run(["check", "base", "--anticipated", "src/a.txt"], repo);
    if (r.rc === 1) { console.log("ok: file prefix anticipates only that file"); pass++; }
    else { console.log(`FAIL: file prefix (exit ${r.rc})\n${r.out}`); fail++; }
    check("file prefix: docs listed", r.out, "  - docs/c.txt");
    notcheck("file prefix: a.txt not listed", r.out, "  - src/a.txt");
    notcheck("file prefix: b.txt not listed", r.out, "  - src/b.txt");
  }

  // --- uncommitted change counted (no commit since base) ------------------------
  {
    const repo = makeRepo("uncommitted");
    // commit2 (docs/c.txt) is committed; the worktree change is NOT
    writeFileSync(join(repo, "src", "a.txt"), "uncommitted\n");
    const r = run(["check", "base", "--anticipated", "src", "docs"], repo);
    if (r.rc === 0) { console.log("ok: uncommitted change counted, clean"); pass++; }
    else { console.log(`FAIL: uncommitted (exit ${r.rc})\n${r.out}`); fail++; }
    // the worktree file must be part of the enumerated set: narrow the base
    // comparison by anticipating nothing — src/a.txt must show as unexpected
    const r2 = run(["check", "base", "--anticipated", "docs"], repo);
    if (r2.rc === 1 && r2.out.includes("  - src/a.txt")) { console.log("ok: uncommitted file appears in the diff"); pass++; }
    else { console.log(`FAIL: uncommitted appears (exit ${r2.rc})\n${r2.out}`); fail++; }
    check("uncommitted: SCOPE", r2.out, "2 file(s) enumerated");
  }

  // --- default excludes: package-lock.json change is not drift ------------------
  {
    const repo = makeRepo("default-exclude");
    writeFileSync(join(repo, "package-lock.json"), "{}\nmodified\n");
    const r = run(["check", "base", "--anticipated", "src", "docs"], repo);
    if (r.rc === 0) { console.log("ok: default excludes clean"); pass++; }
    else { console.log(`FAIL: default excludes (exit ${r.rc})\n${r.out}`); fail++; }
    notcheck("default excludes: lockfile not listed", r.out, "package-lock.json");
  }

  // --- custom --exclude glob -----------------------------------------------------
  {
    const repo = makeRepo("custom-exclude");
    mkdirSync(join(repo, "bench"), { recursive: true });
    writeFileSync(join(repo, "bench", "notes.tmp"), "bench\n");
    git(repo, "add", "-A");
    git(repo, "commit", "-q", "-m", "bench");
    writeFileSync(join(repo, "bench", "notes.tmp"), "bench-modified\n");
    const r = run(["check", "base", "--anticipated", "src", "--exclude", "*.tmp"], repo);
    if (r.rc === 1) { console.log("ok: custom exclude still drifts on docs"); pass++; }
    else { console.log(`FAIL: custom exclude (exit ${r.rc})\n${r.out}`); fail++; }
    notcheck("custom exclude: .tmp not listed", r.out, "  - bench/notes.tmp");
    check("custom exclude: docs listed", r.out, "  - docs/c.txt");
    check("custom exclude: SCOPE exclude count", r.out, "6 excludes");
  }

  // --- unknown base ref → exit 2 with git stderr ---------------------------------
  {
    const repo = makeRepo("bad-ref");
    const r = run(["check", "no-such-ref", "--anticipated", "src"], repo);
    if (r.rc === 2) { console.log("ok: unknown ref exit 2"); pass++; }
    else { console.log(`FAIL: unknown ref (exit ${r.rc})\n${r.out}`); fail++; }
    check("unknown ref: git stderr surfaced", r.out, "unknown revision");
  }

  // --- usage errors → exit 2 --------------------------------------------------------
  {
    const repo = makeRepo("usage");
    let r = run([], repo);
    if (r.rc === 2) { console.log("ok: no args exit 2"); pass++; }
    else { console.log(`FAIL: no args (exit ${r.rc})`); fail++; }
    check("no args: usage text", r.out, "usage:");

    r = run(["check", "--anticipated", "src"], repo);
    if (r.rc === 2) { console.log("ok: missing base-ref exit 2"); pass++; }
    else { console.log(`FAIL: missing base-ref (exit ${r.rc})`); fail++; }
    check("missing base-ref: message", r.out, "missing <base-ref>");

    r = run(["check", "base"], repo);
    if (r.rc === 2) { console.log("ok: missing --anticipated exit 2"); pass++; }
    else { console.log(`FAIL: missing --anticipated (exit ${r.rc})`); fail++; }
    check("missing --anticipated: message", r.out, "missing --anticipated");

    r = run(["check", "base", "--anticipated", "src", "--bogus", "x"], repo);
    if (r.rc === 2) { console.log("ok: unknown flag exit 2"); pass++; }
    else { console.log(`FAIL: unknown flag (exit ${r.rc})`); fail++; }
    check("unknown flag: message", r.out, "unknown flag: --bogus");

    r = run(["diff", "base", "--anticipated", "src"], repo);
    if (r.rc === 2) { console.log("ok: unknown subcommand exit 2"); pass++; }
    else { console.log(`FAIL: unknown subcommand (exit ${r.rc})`); fail++; }
  }

  // --- repeated --anticipated and space-separated values ---------------------------
  {
    const repo = makeRepo("multi-anticipated");
    writeFileSync(join(repo, "src", "a.txt"), "a-modified\n");
    const r = run(["check", "base", "--anticipated", "src", "docs"], repo);
    if (r.rc === 0) { console.log("ok: multiple anticipated roots clean"); pass++; }
    else { console.log(`FAIL: multiple roots (exit ${r.rc})\n${r.out}`); fail++; }
    check("multi: SCOPE root count", r.out, "2 anticipated root(s)");
  }
} finally {
  rmSync(work, { recursive: true, force: true });
}

console.log("");
console.log(`ship-drift-check: ${pass} passed, ${fail} failed`);
process.exit(fail === 0 ? 0 : 1);
