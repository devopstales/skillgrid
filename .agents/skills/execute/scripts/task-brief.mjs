#!/usr/bin/env node
// task-brief.mjs — extract one task's full text from an implementation plan
// into a file the implementer reads in one call, so the task text never has
// to be pasted through the controller's context.
//
// Usage: node task-brief.mjs PLAN_FILE TASK_NUMBER [OUTFILE]
// Default OUTFILE: <repo-root>/.skillgrid/meta/.sdd/<plan-basename>/task-<N>-brief.md
// (per plan and per worktree; concurrent runs of the SAME plan in the same
// working tree share it).
//
// Exit codes: 0 = brief written (path on stdout), 2 = usage / no such plan,
// 3 = no task heading matched (the plan has no Task N / TICKET-N outside
// fenced code blocks).
import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { execFileSync } from "node:child_process";

function usage(msg) {
  if (msg) console.error(msg);
  console.error("usage: node task-brief.mjs PLAN_FILE TASK_NUMBER [OUTFILE]");
  process.exit(2);
}

const plan = process.argv[2];
const n = process.argv[3];
const out = process.argv[4];
if (!plan || !n) usage("missing PLAN_FILE and/or TASK_NUMBER");
if (!existsSync(plan)) usage(`no such plan file: ${plan}`);
if (!/^\d+$/.test(n)) usage(`TASK_NUMBER must be an integer, got: ${n}`);

let file = out;
if (!file) {
  const here = dirname(fileURLToPath(import.meta.url));
  let dir;
  try {
    dir = execFileSync("node", [join(here, "sdd-workspace.mjs"), plan], { encoding: "utf8" }).trimEnd();
  } catch (e) {
    usage(`workspace resolution failed: ${e.message.split("\n")[0]}`);
  }
  file = join(dir, `task-${n}-brief.md`);
}

// Fence-aware: headings inside ``` blocks are not tasks. The original awk
// toggled on every ``` line; the same rule here.
const lines = readFileSync(plan, "utf8").split("\n");
const heading = new RegExp(`^#+[ \\t]+(Task[ \\t]+${n}|TICKET-${n})([^0-9]|$)`);
const anyTask = /^#+[ \t]+(Task[ \t]+\d+|TICKET-\d+)([^0-9]|$)/;
const collected = [];
let inFence = false;
let inTask = false;
for (const line of lines) {
  if (/^```/.test(line)) inFence = !inFence;
  if (!inFence && anyTask.test(line)) inTask = heading.test(line);
  if (inTask) collected.push(line);
}
if (collected.length === 0) {
  console.error(`task ${n} not found in ${plan} (no heading matching 'Task ${n}' or 'TICKET-${n}')`);
  process.exit(3);
}

writeFileSync(file, collected.join("\n") + "\n");
console.log(`wrote ${file}: ${collected.length} lines`);
