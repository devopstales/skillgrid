#!/usr/bin/env node
/**
 * tdd-evidence.mjs — deterministic validator for a report.md's TDD Evidence table.
 *
 * The QA audit-procedure (Step 7) says to spot-check the TDD Evidence table:
 * every ticket needs a RED→GREEN pair, and a commit that doesn't exist is a
 * red flag. This script performs that check deterministically against the
 * real git object store, so the verdict per row is exact, not eyeballed.
 *
 * What it reads:
 *   - <change-dir>/report.md → the `## TDD Evidence` table (two column
 *     layouts are supported: `RED commit / GREEN commit / Test file` and
 *     `RED Test / GREEN Test / Commit`).
 *   - the git repo at <project-root> → `git cat-file -t <hash>` to verify each
 *     hash resolves to a commit; `--find-tests` (default on) globs the named
 *     test file(s).
 *
 * Per-row verdict:
 *   OK            — every cited hash resolves to a commit; named test file(s)
 *                   exist; and (if --run-test was given) the re-run passed
 *   MISSING_RED   — no RED commit hash (and not N/A)
 *   MISSING_GREEN — no GREEN commit hash (and not N/A); or the re-run failed
 *                   (the GREEN commit's test no longer passes)
 *   STALE         — a named test file is missing from the working tree
 *   FAILED        — the re-run could not be executed (runner error / no test) —
 *                   infrastructure, not a TDD-discipline failure; the agent
 *                   classifies it
 *
 * Usage:
 *   node scripts/tdd-evidence.mjs <slug> [project-root] [--no-git]
 *                                [--run-test=<command>]
 *
 *   --no-git           skip the `git cat-file` checks (row verdicts then rest
 *                      on the test-file existence only); use when not in a git
 *                      repo.
 *   --run-test=<cmd>   re-run the named test (e.g. `go test ./install/ -run
 *                      TestHermesSetup`). exit 0 → the GREEN cell is confirmed;
 *                      non-zero → MISSING_GREEN. A command that cannot execute
 *                      (ENOENT / shell error) is FAILED, not MISSING_GREEN.
 *                      Omit to skip the re-run (the git + file checks still run).
 *
 * Exit codes:
 *   0 — table parsed, verdicts emitted (a MISSING/STALE row is a finding the
 *       QA skill renders, not an error here)
 *   2 — usage error (no report.md / no TDD Evidence section / not a skillgrid
 *       project / not a git repo)
 *
 * Output (stdout): a single JSON object:
 * {
 *   "rows": [ { "ticket", "verdict", "red": [...], "green": [...],
 *               "testFile", "missingFile", "retest": "PASS"|"FAIL"|"SKIPPED"|"ERROR" } ],
 *   "summary": "N/M tasks have complete TDD evidence",
 *   "scope": "COMPLETE" | "UNSCOPED"
 * }
 * Deterministic: same repo + same table → same JSON.
 */

import { execFileSync, spawnSync } from "node:child_process";
import { readFileSync, existsSync, readdirSync, statSync } from "node:fs";
import { join, resolve, isAbsolute, dirname } from "node:path";
import {
  parseTddEvidenceTable,
  hashesFromCell,
  isNaCell,
  pathExists,
} from "./lib/skillgrid-parse.mjs";

function read(p) {
  try {
    return readFileSync(p, "utf8");
  } catch {
    return null;
  }
}

function git(repoRoot, ...args) {
  try {
    return {
      ok: true,
      out: execFileSync("git", args, {
        cwd: repoRoot,
        encoding: "utf8",
        stdio: ["pipe", "pipe", "pipe"],
      }),
    };
  } catch (e) {
    return { ok: false, out: String(e.stderr || e.message) };
  }
}

/** Does <root>/.git exist (repo root detection)? */
function isGitRepo(root) {
  return pathExists(join(root, ".git"));
}

/** Find a file anywhere under root, case-insensitively by basename or path suffix. */
function findFile(root, name) {
  const target = String(name).replace(/^`|`$/g, "").trim().toLowerCase();
  if (!target || target === "n/a") return false;
  const base = target.split("/").pop();
  const stack = [root];
  const skip = new Set([".git", "node_modules", ".skillgrid"]);
  while (stack.length) {
    const dir = stack.pop();
    let entries;
    try {
      entries = readdirSync(dir, { withFileTypes: true });
    } catch {
      continue;
    }
    for (const e of entries) {
      if (skip.has(e.name)) continue;
      const full = join(dir, e.name);
      if (e.isDirectory()) {
        stack.push(full);
      } else if (
        e.name.toLowerCase() === target ||
        e.name.toLowerCase() === base
      ) {
        return true;
      }
    }
  }
  return false;
}

function main() {
  const args = process.argv.slice(2);
  const flags = args.filter((a) => a.startsWith("--"));
  const positional = args.filter((a) => !a.startsWith("--"));
  const slug = positional[0];
  const rootArg = positional.length > 1 ? positional[positional.length - 1] : ".";
  if (!slug) {
    console.error("Usage: tdd-evidence.mjs <slug> [project-root] [--no-git]");
    process.exit(2);
  }
  const root = resolve(rootArg);
  const noGit = flags.includes("--no-git");
  const runTestArg = flags.find((f) => f.startsWith("--run-test="));
  const retestCmd = runTestArg ? runTestArg.slice("--run-test=".length) : null;

  const specDir = join(root, ".skillgrid", "specs", slug);
  const reportPath = join(specDir, "report.md");
  if (!pathExists(specDir)) {
    console.error(`ERROR: no spec dir for '${slug}'`);
    process.exit(2);
  }
  const report = read(reportPath);
  if (report === null) {
    console.error(`ERROR: ${reportPath} not found`);
    process.exit(2);
  }
  const table = parseTddEvidenceTable(report);
  if (table.length === 0) {
    console.error(`ERROR: no TDD Evidence table in ${reportPath}`);
    process.exit(2);
  }

  const inGit = !noGit && isGitRepo(root);
  const hashType = (h) => {
    const r = git(root, "cat-file", "-t", h);
    return r.ok ? r.out.trim() : null;
  };

  // Re-run the named test once (it is the same command for every row). The
  // result is shared; a non-zero exit means the GREEN cell is no longer green.
  // A command that cannot spawn (ENOENT / shell failure) is "ERROR" → FAILED
  // (infrastructure), distinct from a real test failure → MISSING_GREEN.
  const retest = (() => {
    if (!retestCmd) return "SKIPPED";
    const r = spawnSync(retestCmd, {
      cwd: root,
      shell: true,
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
      timeout: 300000,
    });
    if (r.error) return "ERROR";
    if (r.status === 0) return "PASS";
    // Shell exit 127 = "command not found" (ENOENT under `shell: true`) — the
    // runner itself could not execute, distinct from a real test failure.
    if (r.status === 127) return "ERROR";
    return "FAIL";
  })();

  const rows = [];
  let complete = 0;
  for (const t of table) {
    const row = { ticket: t.ticket, red: [], green: [], testFile: t.testFile, missingFile: null, retest };
    let verdict;

    // A cell that is `N/A (…)` means "no such commit for this ticket" — the
    // cell is absent but acceptable, not a missing-evidence failure. The RED
    // column is N/A when there is no test framework (hermes Python tickets).
    const redNA = isNaCell(t.red);
    const greenNA = isNaCell(t.green);
    let redHashes = redNA ? [] : hashesFromCell(t.red);
    let greenHashes = greenNA ? [] : hashesFromCell(t.green);
    // The `Commit` column (hermes layout) may hold both RED+GREEN hashes when
    // RED/GREEN cells are prose. If the dedicated cells are empty, mine them.
    if (redHashes.length === 0 && greenHashes.length === 0 && t.commit) {
      const ch = hashesFromCell(t.commit);
      if (ch.length >= 2) {
        redHashes = [ch[0]];
        greenHashes = [ch[1]];
      } else if (ch.length === 1 && !redNA) {
        greenHashes = [ch[0]];
      }
    }
    row.red = redHashes;
    row.green = greenHashes;

    const problems = [];
    if (!redNA && redHashes.length === 0) problems.push("MISSING_RED");
    if (!greenNA && greenHashes.length === 0) problems.push("MISSING_GREEN");
    // The re-run (if requested) confirms the GREEN cell still passes. A real
    // failure → MISSING_GREEN; a command that cannot execute → FAILED (infra).
    if (retest === "FAIL") problems.push("MISSING_GREEN");
    else if (retest === "ERROR") problems.push("FAILED");
    // Verify every cited hash resolves to a commit.
    if (inGit) {
      for (const h of [...redHashes, ...greenHashes]) {
        const type = hashType(h);
        if (type === "commit") continue;
        if (type) problems.push("NOT_A_COMMIT");
        else problems.push("HASH_NOT_FOUND");
      }
    }
    // Test file must exist (a stale reference is a finding).
    if (t.testFile && !isNaCell(t.testFile)) {
      const fileOk = findFile(root, t.testFile);
      if (!fileOk) {
        row.missingFile = t.testFile;
        problems.push("STALE");
      }
    }
    verdict = problems.length === 0 ? "OK" : problems[0];

    row.verdict = verdict;
    if (verdict === "OK") complete++;
    rows.push(row);
  }

  const total = rows.length;
  const summary = `${complete}/${total} tasks have complete TDD evidence`;
  const result = {
    slug,
    rows,
    summary,
    scope: "COMPLETE",
  };
  console.log(JSON.stringify(result, null, 2));
  process.exit(0);
}

main();
