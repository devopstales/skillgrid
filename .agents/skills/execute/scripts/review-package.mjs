#!/usr/bin/env node
// review-package.mjs — generate a review package: commit list, stat summary,
// and the net diff with extended context, written to a file the reviewer
// reads in one call. Using the recorded per-task BASE (not HEAD~1) keeps
// multi-commit tasks intact.
//
// Usage: node review-package.mjs PLAN_FILE BASE HEAD [OUTFILE]
// Default OUTFILE: <repo-root>/.skillgrid/meta/.sdd/<plan-basename>/review-<base7>..<head7>.diff
// (named per range, so a re-review after fixes gets a distinct fresh file).
//
// Exit codes: 0 = package written (path on stdout), 2 = usage / no such plan /
// bad BASE or HEAD.
import { existsSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { execFileSync } from "node:child_process";

function usage(msg) {
  if (msg) console.error(msg);
  console.error("usage: node review-package.mjs PLAN_FILE BASE HEAD [OUTFILE]");
  process.exit(2);
}

function git(cwd, args) {
  return execFileSync("git", args, { encoding: "utf8", cwd, stdio: ["ignore", "pipe", "pipe"] });
}
function gitOk(cwd, args) {
  try {
    execFileSync("git", args, { encoding: "utf8", cwd, stdio: "ignore" });
    return true;
  } catch {
    return false;
  }
}

const plan = process.argv[2];
const base = process.argv[3];
const head = process.argv[4];
const out = process.argv[5];
if (!plan || !base || !head) usage("missing PLAN_FILE, BASE, and/or HEAD");
if (!existsSync(plan)) usage(`no such plan file: ${plan}`);

let cwd;
try {
  cwd = execFileSync("git", ["rev-parse", "--show-toplevel"], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] }).trimEnd();
} catch {
  usage("not inside a git working tree (git rev-parse --show-toplevel failed)");
}
if (!gitOk(cwd, ["rev-parse", "--verify", "--quiet", base])) usage(`bad BASE: ${base}`);
if (!gitOk(cwd, ["rev-parse", "--verify", "--quiet", head])) usage(`bad HEAD: ${head}`);

let file = out;
if (!file) {
  const here = dirname(fileURLToPath(import.meta.url));
  let ws;
  try {
    ws = execFileSync("node", [join(here, "sdd-workspace.mjs"), plan], { encoding: "utf8" }).trimEnd();
  } catch (e) {
    usage(`workspace resolution failed: ${e.message.split("\n")[0]}`);
  }
  const shortBase = git(cwd, ["rev-parse", "--short", base]).trimEnd();
  const shortHead = git(cwd, ["rev-parse", "--short", head]).trimEnd();
  file = join(ws, `review-${shortBase}..${shortHead}.diff`);
}

const commits = git(cwd, ["log", "--oneline", `${base}..${head}`]).trimEnd();
const stat = git(cwd, ["diff", "--stat", `${base}..${head}`]).trimEnd();
const diff = git(cwd, ["diff", "-U10", `${base}..${head}`]);

const body = `# Review package: ${base}..${head}\n\n## Commits\n${commits}\n\n## Files changed\n${stat}\n\n## Diff\n${diff}`;
writeFileSync(file, body);
const commitCount = git(cwd, ["rev-list", "--count", `${base}..${head}`]).trimEnd();
console.log(`wrote ${file}: ${commitCount} commit(s), ${Buffer.byteLength(body)} bytes`);
