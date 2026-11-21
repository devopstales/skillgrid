#!/usr/bin/env node
// sdd-workspace.mjs — resolve and ensure the working-tree directory SDD uses
// for one plan's short-lived artifacts: task briefs, implementer reports,
// review packages, and the progress ledger. Print the plan directory's
// absolute path (one line on stdout — this script's caller is a shell
// assignment, so stdout stays plain; everything else goes to stderr).
//
// One directory per plan (.skillgrid/meta/.sdd/<spec-change-folder>/) so a follow-up
// plan in the same working tree can never read or overwrite another plan's
// artifacts. A stale ledger misread as current progress makes controllers
// skip whole task sequences — plan-scoping removes that failure structurally.
// The slug is the spec change folder name (parent dir of the plan file), not
// the plan file basename, so every change gets a unique workspace even when
// the plan file is conventionally named blueprint.md.
//
// The workspace lives in the working tree (not under .git/) because Claude Code
// treats .git/ as a protected path and denies agent writes there — which blocks
// an implementer subagent from writing its report file. A self-ignoring
// .gitignore at .skillgrid/meta/.sdd/ keeps every plan's workspace out of
// `git status` and out of accidental commits without modifying any tracked file.
//
// Single source of truth for the workspace location, so task-brief and
// review-package cannot drift to different directories.
//
// Ledger convention: <workspace>/progress.md line 1 MUST be
// `# SDD ledger — plan: <plan-file-path>`. The mismatch guard (a ledger
// naming a different plan belongs to another plan — start at the first task
// and leave the foreign ledger untouched) is skill text, not script logic;
// these scripts just guarantee one directory per plan so the guard holds.
//
// Usage: node sdd-workspace.mjs PLAN_FILE
// Exit codes: 0 = resolved (path on stdout), 2 = usage / no such plan.
import { existsSync, mkdirSync, writeFileSync } from "node:fs";
import { dirname, basename, join } from "node:path";
import { execFileSync } from "node:child_process";

function usage(msg) {
  if (msg) console.error(msg);
  console.error("usage: node sdd-workspace.mjs PLAN_FILE");
  process.exit(2);
}

const plan = process.argv[2];
if (!plan) usage("missing PLAN_FILE");
if (!existsSync(plan)) usage(`no such plan file: ${plan}`);

// Prefer the spec change folder name (parent dir of the plan file) so each
// change gets a unique workspace slug. Fall back to the plan file basename
// when the parent is the specs root or empty (e.g. a plan dropped loose in
// the specs dir). This keeps one-dir-per-change even when every plan is
// named blueprint.md.
let slug = basename(dirname(plan));
if (!slug || slug === "." || slug === "specs") {
  slug = basename(plan).replace(/\.md$/, "");
}
if (!slug || slug === "." || slug === "..") usage(`cannot derive a workspace name from: ${plan}`);

let root;
try {
  root = execFileSync("git", ["rev-parse", "--show-toplevel"], { encoding: "utf8" }).trimEnd();
} catch {
  usage("not inside a git working tree (git rev-parse --show-toplevel failed)");
}
const base = join(root, ".skillgrid", "meta", ".sdd");
const dir = join(base, slug);
mkdirSync(dir, { recursive: true });
writeFileSync(join(base, ".gitignore"), "*\n");
console.log(dir);
