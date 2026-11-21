#!/usr/bin/env node
// ship-archive.mjs — the mechanical archive move (ship Step 7) as one call.
//
// Moves .skillgrid/change-specs/<topic>/ to .skillgrid/change-archive/<topic>/
// with a pre-move snapshot and a mandatory `diff -r` readback, then closes out
// state.yaml (remove changes.<topic>, increment completed_changes) and
// releases the session lock. Content never passes through the model's
// Read/Write path — this script is the move.
//
// Usage:
//   node ship-archive.mjs <topic> [project-root]
//   <topic>         the change folder name (YYYY-MM-DD-<topic>)
//   [project-root]  defaults to cwd
//
// Contract (per _shared/craft/deterministic-boundary.md "agentic script
// contract"): JSON on stdout, diagnostics on stderr, exit 0 = clean move,
// exit 1 = move failed (readback non-empty, source left, or state error),
// exit 2 = usage / preconditions not met. Never prompts.

import { existsSync, readFileSync, writeFileSync, mkdtempSync, rmSync, mkdirSync, cpSync, renameSync } from "node:fs";
import { join, resolve } from "node:path";
import { execFileSync } from "node:child_process";

function usage(msg) {
  if (msg) console.error(msg);
  console.error("usage: node ship-archive.mjs <topic> [project-root]");
  process.exit(2);
}

const topic = process.argv[2];
const root = resolve(process.argv[3] || ".");
if (!topic) usage("missing <topic>");
if (!existsSync(join(root, ".skillgrid", "change-specs", topic)))
  usage(`no change-specs folder: ${join(root, ".skillgrid", "change-specs", topic)}`);

const sourceDir = join(root, ".skillgrid", "change-specs", topic);
const archiveDir = join(root, ".skillgrid", "change-archive", topic);
if (existsSync(archiveDir))
  usage(`archive target already exists: ${archiveDir} — the archive is an audit trail, never overwritten`);

function sh(cmd, args, opts = {}) {
  return execFileSync(cmd, args, { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"], ...opts }).trimEnd();
}

const result = { topic, moved: false, readback: "", source_gone: false, state: null, lock: null, commits: [] };

// --- 1. pre-move snapshot ------------------------------------------------------

const snapshotRoot = mkdtempSync(join(root, ".skillgrid", "ship.move.XXXXXX"));
try {
  cpSync(sourceDir, join(snapshotRoot, "source"), { recursive: true });

  // --- 2. mechanical move: git mv, fall back to plain mv -----------------------
  mkdirSync(join(root, ".skillgrid", "change-archive"), { recursive: true });
  let usedGit = false;
  try {
    sh("git", ["mv", sourceDir, archiveDir], { cwd: root });
    usedGit = true;
  } catch {
    // not a git repo or untracked folder — plain move (still mechanical, never Read/Write)
    renameSync(sourceDir, archiveDir);
  }
  result.used_git_mv = usedGit;

  // --- 3. source must be gone before comparing ---------------------------------
  result.source_gone = !existsSync(sourceDir);
  if (!result.source_gone) {
    result.readback = "archive move left the source directory in place";
    throw new Error(result.readback);
  }

  // --- 4. MANDATORY readback: only an empty diff passes -------------------------
  const diff = sh("diff", ["-r", join(snapshotRoot, "source"), archiveDir]);
  result.readback = diff; // "" = empty = pass
  if (diff !== "") throw new Error("diff -r readback non-empty");
  result.moved = true;
} finally {
  rmSync(snapshotRoot, { recursive: true, force: true });
}

// --- 5. close out state.yaml (same edit as the move) ----------------------------

const statePath = join(root, ".skillgrid", "state.yaml");
if (existsSync(statePath)) {
  try {
    const lines = readFileSync(statePath, "utf8").split("\n");
    const out = [];
    let inChanges = false;
    let removed = false;
    for (const line of lines) {
      if (line.startsWith("changes:")) inChanges = true;
      if (!removed && inChanges && line.trim() === `${topic}:` && line.startsWith("  ")) {
        removed = true; // drop the topic entry and its indented children
        continue;
      }
      if (removed && line.trim() !== "" && (line.startsWith("    ") || line.startsWith("\t\t"))) continue;
      // reset on the next top-level key — but not on `changes:` itself, which
      // is top-level AND the marker (the marker line sets the flag, it never
      // clears it)
      if (inChanges && !line.startsWith("changes:") && line !== "" && !line.startsWith(" ") && !line.startsWith("\t")) inChanges = false;
      out.push(line);
    }
    let text = out.join("\n");
    const inc = text.replace(/(completed_changes:\s*)(\d+)/, (_, pre, n) => `${pre}${parseInt(n, 10) + 1}`);
    if (inc !== text) {
      text = inc;
      result.state = { removed, completed_changes: incremented(inc) };
    } else {
      result.state = { removed, completed_changes: "no counter found — left unchanged" };
    }
    writeFileSync(statePath, text);
  } catch (e) {
    // state close-out failure does not un-move the archive, but it is a finding
    result.state = { error: e.message };
    throw e;
  }
} else {
  result.state = "state.yaml absent — skipped";
}

// --- 6. release the session lock (always exit 0 — no-op if absent) ---------------

const lockScript = findLockScript(root);
if (lockScript) {
  try {
    const out = execFileSync("node", [lockScript, "release", topic], { encoding: "utf8", cwd: root, stdio: ["ignore", "pipe", "pipe"] }).trimEnd();
    result.lock = out || "released";
  } catch {
    result.lock = "release failed (non-fatal)";
  }
} else {
  result.lock = "state-lock.mjs not found — skipped";
}

console.log(JSON.stringify(result, null, 2));
process.exit(0);

// --- helpers ---------------------------------------------------------------------

function incremented(text) {
  const m = text.match(/completed_changes:\s*(\d+)/);
  return m ? parseInt(m[1], 10) : null;
}

// locate the consumer-side state-lock script (sdd-structure.md § Session lock)
function findLockScript(root) {
  const candidates = [
    join(root, "scripts", "state-lock.mjs"),
    join(root, ".agents", "skills", "qa", "scripts", "state-lock.mjs"),
  ];
  return candidates.find((p) => existsSync(p)) || null;
}
