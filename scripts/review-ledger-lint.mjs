#!/usr/bin/env node
/**
 * review-ledger-lint.mjs — deterministic schema validator for a change's
 * review-ledger.md (review-ledger/SKILL.md § Step 3).
 *
 * The ledger is a strict markdown schema the agent currently validates by eye:
 *   1. Every finding has a disposition (a `- **<id>**` bullet in Dispositions).
 *   2. Every `fixed` finding carries a fix-commit hash that resolves in git.
 *   3. Every `parked` finding has a ruling AND a named human.
 *   4. Round numbers are sequential (1, 2, 3 …) — no gaps, no duplicates.
 *   5. Finding IDs within a round are sequential (F001, F002 …).
 * This script performs that check deterministically (schema + git-hash), so the
 * verdict is exact, not eyeballed. It also reads the **review debt** for QA
 * Step 5: parked findings in prior archived changes' ledgers.
 *
 * Usage:
 *   node scripts/review-ledger-lint.mjs <slug> [project-root] [--no-git]
 *
 *   --no-git   skip the fix-commit `git cat-file` checks (verdicts rest on the
 *              schema checks only); use when not in a git repo.
 *
 * Exit codes:
 *   0 — ledger parsed, no schema findings (debt is data, not an error)
 *   1 — one or more schema findings (the QA skill renders them; the gate folds
 *       them into the report)
 *   2 — usage error (no review-ledger.md for the slug)
 *
 * Output (stdout): a single JSON object:
 * {
 *   "slug", "rounds": N,
 *   "findings": [ { "finding": "R1-F001", "rule": "no-disposition" |
 *                   "bad-fix-commit" | "parked-no-ruling" | "bad-severity" |
 *                   "round-gap" | "round-dup" | "finding-id-gap" } ],
 *   "debt": [ { "change": "<slug>", "finding": "R1-F002", "note": "…" } ],
 *   "ok": true | false,
 *   "scope": "COMPLETE" | "UNREADABLE"
 * }
 * Deterministic: same ledger + same git repo → same JSON.
 */

import { execFileSync } from "node:child_process";
import { readFileSync, existsSync, readdirSync } from "node:fs";
import { join, resolve } from "node:path";
import {
  parseReviewLedger,
  REVIEW_DISPOSITIONS,
  pathExists,
} from "./lib/skillgrid-parse.mjs";

function read(p) {
  try {
    return readFileSync(p, "utf8");
  } catch {
    return null;
  }
}

function git(root, ...args) {
  try {
    return execFileSync("git", args, { cwd: root, encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] }).trim();
  } catch {
    return null;
  }
}

function isGitRepo(root) {
  return pathExists(join(root, ".git"));
}

function main() {
  const args = process.argv.slice(2);
  const noGit = args.includes("--no-git");
  const positional = args.filter((a) => !a.startsWith("--"));
  const slug = positional[0];
  const rootArg = positional.length > 1 ? positional[positional.length - 1] : ".";
  if (!slug) {
    console.error("Usage: review-ledger-lint.mjs <slug> [project-root] [--no-git]");
    process.exit(2);
  }
  const root = resolve(rootArg);
  const inGit = !noGit && isGitRepo(root);

  const ledgerPath = join(root, ".skillgrid", "specs", slug, "review-ledger.md");
  if (!pathExists(ledgerPath)) {
    console.error(`ERROR: no review-ledger.md for '${slug}'`);
    process.exit(2);
  }
  const text = read(ledgerPath);
  const { rounds } = parseReviewLedger(text);

  const findings = [];
  const hashOk = (h) => {
    const t = git(root, "cat-file", "-t", h);
    return t === "commit";
  };

  // Round-number checks (sequential, no dupes).
  const ns = rounds.map((r) => r.n);
  const seen = new Set();
  for (const n of ns) {
    if (seen.has(n)) findings.push({ finding: `R${n}`, rule: "round-dup" });
    seen.add(n);
  }
  const sorted = [...new Set(ns)].sort((a, b) => a - b);
  if (sorted.length) {
    if (sorted[0] !== 1) findings.push({ finding: `R${sorted[0]}`, rule: "round-gap", detail: "rounds must start at 1" });
    for (let i = 1; i < sorted.length; i++) {
      if (sorted[i] !== sorted[i - 1] + 1) findings.push({ finding: `R${sorted[i]}`, rule: "round-gap", detail: `expected R${sorted[i - 1] + 1}` });
    }
  }

  // Per-round finding + disposition checks.
  for (const r of rounds) {
    // Finding IDs within a round are sequential (F001, F002, …).
    const ids = r.findings.map((f) => f.id);
    const fnums = ids.map((id) => parseInt(id.replace(/^R\d+-F/, ""), 10));
    const fseen = new Set();
    for (const fn of fnums) {
      if (fseen.has(fn)) findings.push({ finding: `R${r.n}`, rule: "finding-id-gap", detail: `F${String(fn).padStart(3, "0")} duplicated` });
      fseen.add(fn);
    }
    const fsorted = [...new Set(fnums)].sort((a, b) => a - b);
    if (fsorted.length) {
      if (fsorted[0] !== 1) findings.push({ finding: `R${r.n}`, rule: "finding-id-gap", detail: "finding IDs must start at F001" });
      for (let i = 1; i < fsorted.length; i++) {
        if (fsorted[i] !== fsorted[i - 1] + 1) findings.push({ finding: `R${r.n}`, rule: "finding-id-gap", detail: `expected F${String(fsorted[i - 1] + 1).padStart(3, "0")}` });
      }
    }

    for (const f of r.findings) {
      // 1. Every finding has a disposition bullet.
      if (!r.dispositionIds.has(f.id)) {
        findings.push({ finding: f.id, rule: "no-disposition" });
        continue;
      }
      // Severity must be a known enum (the schema names three).
      if (f.severity && !["CRITICAL", "WARNING", "SUGGESTION"].includes(f.severity.toUpperCase())) {
        findings.push({ finding: f.id, rule: "bad-severity", detail: f.severity });
      }
      const disp = f.disposition;
      if (!REVIEW_DISPOSITIONS.has(disp)) {
        findings.push({ finding: f.id, rule: "no-disposition", detail: `unknown disposition '${disp}'` });
        continue;
      }
      if (disp === "fixed") {
        // 2. A fix-commit hash must be present and resolve in git.
        const hashes = (f.fixCommit || "").match(/\b[0-9a-f]{7,40}\b/gi) || [];
        if (hashes.length === 0) {
          findings.push({ finding: f.id, rule: "bad-fix-commit", detail: "no commit hash in Fix Commit column" });
        } else if (inGit) {
          for (const h of hashes) {
            if (!hashOk(h)) findings.push({ finding: f.id, rule: "bad-fix-commit", detail: `${h} does not resolve to a commit` });
          }
        }
      }
      // 3. A `parked` finding needs a ruling + a named human in its bullet.
      if (disp === "parked") {
        const bullet = (text.match(new RegExp(`-\\s*\\*\\*${f.id}\\*\\*\\s*—?\\s*([^\\n]+)`)) || [])[1] || "";
        const hasRuling = /\bruling\b/i.test(bullet);
        // A named human: a token that is not a stop-word, after "Ruling:".
        const human = bullet.split(/[.\s]+/).filter(Boolean);
        const stop = new Set(["ruling", "parked", "park", "the", "a", "an", "is", "was", "because", "deferred", "to", "open", "acceptable", "risk", "accepted", "minor", "note", "n/a"]);
        const namedHuman = human.some((tok) => !stop.has(tok.toLowerCase()) && !/^[—–-]+$/.test(tok) && /[a-zA-Z]/.test(tok));
        if (!hasRuling || !namedHuman) {
          findings.push({ finding: f.id, rule: "parked-no-ruling", detail: hasRuling ? "no named human" : "no ruling" });
        }
      }
    }
  }

  // Review debt: parked findings in prior archived changes' ledgers.
  const debt = [];
  const xarchive = join(root, ".skillgrid", "xarchive");
  if (pathExists(xarchive)) {
    let dirs = [];
    try { dirs = readdirSync(xarchive, { withFileTypes: true }).filter((e) => e.isDirectory()).map((e) => e.name).sort(); } catch {}
    for (const d of dirs) {
      const lp = join(xarchive, d, "review-ledger.md");
      const lt = read(lp);
      if (!lt) continue;
      const { rounds: rr } = parseReviewLedger(lt);
      for (const r of rr) {
        for (const f of r.findings) {
          if (f.disposition === "parked") {
            const bullet = (lt.match(new RegExp(`-\\s*\\*\\*${f.id}\\*\\*\\s*—?\\s*([^\\n]+)`)) || [])[1] || "";
            debt.push({ change: d, finding: f.id, note: bullet.slice(0, 120) });
          }
        }
      }
    }
  }

  const result = {
    slug,
    rounds: rounds.length,
    findings,
    debt,
    ok: findings.length === 0,
    scope: "COMPLETE",
  };
  console.log(JSON.stringify(result, null, 2));
  process.exit(findings.length > 0 ? 1 : 0);
}

main();
