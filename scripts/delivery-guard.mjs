#!/usr/bin/env node
/**
 * delivery-guard.mjs — single parser for the four delivery-guard lines in
 * tasks.md (subagent-execution "Delivery Guard").
 *
 * Three skills — subagent-execution, ship, and qa-gate — each regex the same
 * four plain-text lines independently. A reworded line silently breaks the
 * guard (the execution skill even says "the executor matches them literally. Do
 * not reword or drop any."). This script is the single source of truth: it
 * reads the four lines once and emits a structured decision the skills reuse.
 *
 * The four contract lines (plain-text, `Key: value`):
 *   Decision needed before apply: <Yes|No> [— <reason>]
 *   Chained PRs recommended: <Yes|No>
 *   Chain strategy: <ref|name>
 *   400-line budget risk: <Low|Medium|High>
 *
 * Usage:
 *   node scripts/delivery-guard.mjs <slug> [project-root]
 *
 * Exit codes:
 *   0 — guard parsed (blocked or not; the decision is data the skill renders)
 *   2 — usage error (no tasks.md for the slug)
 *
 * Output (stdout): a single JSON object:
 * {
 *   "slug",
 *   "decisionNeeded": true | false | null,   // null = line missing
 *   "decisionReason": "<reason>" | null,
 *   "chainedPRs": true | false | null,
 *   "chainStrategy": "<ref>" | null,
 *   "budgetRisk": "Low" | "Medium" | "High" | null,
 *   "consistency": "ok" | "inconsistent" | "unknown",
 *   "blocked": "<reason>" | null,           // non-null = the guard blocks apply
 *   "baseBranch": "<ref>" | null,           // the Chain strategy value (ship reuses)
 *   "scope": "COMPLETE" | "UNSCOPED"
 * }
 *
 * `blocked` is non-null when `decisionNeeded` is true (a human must decide
 * before apply). `consistency` is "inconsistent" when budgetRisk is High but
 * chainedPRs is not Yes (the contract: High risk ⇒ chained PRs recommended).
 * Deterministic: same tasks.md → same JSON.
 */

import { readFileSync, existsSync } from "node:fs";
import { join, resolve } from "node:path";

function read(p) {
  try {
    return readFileSync(p, "utf8");
  } catch {
    return null;
  }
}

// Match a `Key: value` line. The value runs to end-of-line. The `—` (em dash)
// separates a reason after a Yes/No. Keys are matched by prefix so a reworded
// *suffix* still resolves; the contract is the key itself.
function guardLine(text, key) {
  // Anchor at line start; allow leading whitespace; the key is literal.
  const re = new RegExp(`^[ \\t]*${key}\\s*:\\s*(.*)$`, "im");
  const m = text.match(re);
  return m ? m[1].trim() : null;
}

function parseYesNo(value) {
  if (value === null) return null;
  if (/^yes\b/i.test(value)) return true;
  if (/^no\b/i.test(value)) return false;
  return null;
}

function parseRisk(value) {
  if (value === null) return null;
  const v = value.trim().toLowerCase();
  if (v.startsWith("low")) return "Low";
  if (v.startsWith("medium")) return "Medium";
  if (v.startsWith("high")) return "High";
  return null;
}

function main() {
  const args = process.argv.slice(2);
  const positional = args.filter((a) => !a.startsWith("--"));
  const slug = positional[0];
  const rootArg = positional.length > 1 ? positional[positional.length - 1] : ".";
  if (!slug) {
    console.error("Usage: delivery-guard.mjs <slug> [project-root]");
    process.exit(2);
  }
  const root = resolve(rootArg);
  const tasksPath = join(root, ".skillgrid", "specs", slug, "tasks.md");
  if (!existsSync(tasksPath)) {
    console.error(`ERROR: no tasks.md for '${slug}'`);
    process.exit(2);
  }
  const text = read(tasksPath);

  const decisionRaw = guardLine(text, "Decision needed before apply");
  const chainedRaw = guardLine(text, "Chained PRs recommended");
  const chainStrategyRaw = guardLine(text, "Chain strategy");
  const riskRaw = guardLine(text, "400-line budget risk");

  const decisionNeeded = parseYesNo(decisionRaw);
  const decisionReason =
    decisionNeeded === true && decisionRaw
      ? decisionRaw.split(/[—–-]\s*/)[1]?.trim() || null
      : null;
  const chainedPRs = parseYesNo(chainedRaw);
  const chainStrategy =
    chainStrategyRaw && chainStrategyRaw !== "—" ? chainStrategyRaw : null;
  const budgetRisk = parseRisk(riskRaw);

  // Consistency: the contract says High risk ⇒ Chained PRs recommended Yes.
  let consistency = "unknown";
  if (budgetRisk !== null && chainedPRs !== null) {
    consistency = budgetRisk === "High" && chainedPRs === false ? "inconsistent" : "ok";
  }

  // blocked: a human must decide before apply when decisionNeeded is Yes.
  let blocked = null;
  if (decisionNeeded === true) {
    blocked = decisionReason
      ? `decision needed: ${decisionReason}`
      : "decision needed before apply (no reason given)";
  }

  const present = [decisionNeeded, chainedPRs, chainStrategy, budgetRisk];
  const scope = present.every((v) => v !== null) ? "COMPLETE" : "UNSCOPED";

  const result = {
    slug,
    decisionNeeded,
    decisionReason,
    chainedPRs,
    chainStrategy,
    budgetRisk,
    consistency,
    blocked,
    baseBranch: chainStrategy,
    scope,
  };
  console.log(JSON.stringify(result, null, 2));
  process.exit(0);
}

main();
