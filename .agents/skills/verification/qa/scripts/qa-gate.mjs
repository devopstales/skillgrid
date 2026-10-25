#!/usr/bin/env node
/**
 * qa-gate.mjs — deterministic gate-runner for the QA skill.
 *
 * Runs the three drift/budget sub-scripts, captures their stdout + exit
 * codes + SCOPE lines, and prints a structured JSON summary. The QA skill
 * reads this output and combines it with the audit findings (Steps 3-9)
 * to render the four-state verdict. The gate script owns the "run the
 * deterministic checks" half; the skill owns the "render the verdict" half.
 *
 * Usage:
 *   node .agents/skills/verification/qa/scripts/qa-gate.mjs [project-root]
 *
 * The project root defaults to the current working directory.
 *
 * Exit codes:
 *   0 — all sub-checks clean (no drift, budget OK)
 *   1 — one or more sub-checks detected an issue (drift or overage)
 *   2 — usage/parse error (a sub-script failed to run)
 *
 * Output (stdout): a single JSON object:
 * {
 *   "verdict": "CLEAN" | "ISSUES" | "ERROR",
 *   "checks": {
 *     "state_drift":   { "exit": 0, "scope": "COMPLETE", "lines": [...] },
 *     "ship_drift":    { "exit": 0, "scope": "COMPLETE", "lines": [...] },
 *     "size_budget":   { "exit": 0, "lines": [...] }
 *   },
 *   "scopes": {
 *     "state_drift": "COMPLETE",
 *     "ship_drift": "COMPLETE",
 *     "composite": "COMPLETE"
 *   }
 * }
 *
 * The `scopes.composite` field is the worst scope across all checks that
 * emit a SCOPE line (per verification-scope.md: worst-scope-wins). The QA
 * skill's fail-closed rule (Step 10, rule 7) reads this field: a composite
 * scope that is not "COMPLETE" routes the gate away from PASS.
 */

import { execFileSync } from "node:child_process";
import { readFileSync, readdirSync, existsSync } from "node:fs";
import { join, resolve } from "node:path";

const SCOPE_RANK = { COMPLETE: 0, TRUNCATED: 1, UNSCOPED: 2, UNREADABLE: 3 };

function worstScope(a, b) {
  if (!a) return b || "COMPLETE";
  if (!b) return a;
  return SCOPE_RANK[b] > SCOPE_RANK[a] ? b : a;
}

function runScript(root, scriptPath, args) {
  const fullArgs = [...args, root];
  let stdout = "";
  let exit = 0;
  try {
    stdout = execFileSync("node", [scriptPath, ...fullArgs], {
      cwd: root,
      encoding: "utf8",
      stdio: ["pipe", "pipe", "pipe"],
    }).trim();
  } catch (e) {
    stdout = ((e.stdout || "") + "\n" + (e.stderr || "")).trim();
    exit = e.status ?? 2;
  }
  const lines = stdout ? stdout.split("\n") : [];
  const scopeMatch = stdout.match(/SCOPE:\s*(\w+)/);
  return {
    exit,
    scope: scopeMatch ? scopeMatch[1] : null,
    lines,
  };
}

// Find the active change's slug (newest dir under .skillgrid/specs/).
function findActiveSlug(root) {
  const specsDir = join(root, ".skillgrid", "specs");
  if (!existsSync(specsDir)) return null;
  const dirs = readdirSync(specsDir, { withFileTypes: true })
    .filter((e) => e.isDirectory())
    .map((e) => e.name)
    .sort();
  return dirs.length ? dirs[dirs.length - 1] : null;
}

// Find the active change's tasks.md (newest dir under .skillgrid/specs/).
function findTasksPath(root) {
  const slug = findActiveSlug(root);
  if (!slug) return null;
  const p = join(root, ".skillgrid", "specs", slug, "tasks.md");
  return existsSync(p) ? p : null;
}

// Run a repo-level check script (traceability / tdd-evidence) for the active
// change. These emit JSON (not the drift-guard `SCOPE:` line protocol), so the
// gate parses the JSON and folds it into a check entry. A run that errors
// (no acceptance.feature / no TDD table — exit 2) is a finding (exit 1), not
// an ERROR, so the gate stays informative rather than failing closed.
function runRepoCheck(root, scriptName, args) {
  const scriptPath = join(root, "scripts", scriptName);
  if (!existsSync(scriptPath)) {
    return { exit: 2, lines: ["script not found"], summary: null };
  }
  try {
    const stdout = execFileSync("node", [scriptPath, ...args, root], {
      cwd: root,
      encoding: "utf8",
      stdio: ["pipe", "pipe", "pipe"],
    });
    let parsed = null;
    try { parsed = JSON.parse(stdout); } catch {}
    return {
      exit: 0,
      lines: parsed ? [parsed.summary] : stdout.trim().split("\n"),
      summary: parsed?.summary ?? null,
      detail: parsed ?? null,
    };
  } catch (e) {
    const msg = ((e.stdout || "") + "\n" + (e.stderr || "")).trim();
    return { exit: 1, lines: msg ? msg.split("\n") : ["no output"], summary: null, detail: null };
  }
}

// Read the base ref from tasks.md → "Chain strategy: X" line.
function readBaseRef(tasksPath) {
  try {
    const content = readFileSync(tasksPath, "utf8");
    const match = content.match(/Chain strategy:\s*(\S+)/);
    return match ? match[1] : null;
  } catch {
    return null;
  }
}

function main() {
  const root = resolve(process.argv[2] || ".");
  const scriptsDir = join(root, ".agents", "skills", "verification", "qa", "scripts");

  const result = {
    verdict: "CLEAN",
    checks: {},
    scopes: { composite: "COMPLETE" },
  };

  // --- state-drift-check ---
  const stateScript = join(scriptsDir, "state-drift-check.mjs");
  if (existsSync(stateScript)) {
    result.checks.state_drift = runScript(root, stateScript, []);
    if (result.checks.state_drift.scope) {
      result.scopes.state_drift = result.checks.state_drift.scope;
    }
  } else {
    result.checks.state_drift = { exit: 2, scope: null, lines: ["script not found"] };
  }

  // --- ship-drift-check ---
  // Requires a base ref from tasks.md → "Chain strategy:" line.
  // If not found, skip (the QA skill handles this in Step 9.6).
  const shipScript = join(scriptsDir, "ship-drift-check.mjs");
  const tasksPath = findTasksPath(root);
  const baseRef = tasksPath ? readBaseRef(tasksPath) : null;

  if (existsSync(shipScript) && baseRef) {
    result.checks.ship_drift = runScript(root, shipScript, ["check", baseRef]);
    if (result.checks.ship_drift.scope) {
      result.scopes.ship_drift = result.checks.ship_drift.scope;
    }
  } else {
    result.checks.ship_drift = {
      exit: 0,
      scope: null,
      lines: [baseRef ? "skipped: no --anticipated paths" : "skipped: no base ref in tasks.md"],
    };
  }

  // --- skill-size-budget ---
  const budgetScript = join(scriptsDir, "skill-size-budget.mjs");
  if (existsSync(budgetScript)) {
    result.checks.size_budget = runScript(root, budgetScript, ["check"]);
  } else {
    result.checks.size_budget = { exit: 2, scope: null, lines: ["script not found"] };
  }

  // --- adr-invariants ---
  const adrScript = join(root, "scripts", "check-adr-invariants.mjs");
  if (existsSync(adrScript)) {
    result.checks.adr_invariants = runScript(root, adrScript, []);
  } else {
    result.checks.adr_invariants = { exit: 2, scope: null, lines: ["script not found"] };
  }

  // --- okf-lint (knowledge bundle) ---
  const okfScript = join(scriptsDir, "okf-lint.mjs");
  if (existsSync(okfScript)) {
    const okf = runScript(root, okfScript, []);
    result.checks.okf_lint = okf;
    if (okf.scope) result.scopes.okf_lint = okf.scope;
  } else {
    result.checks.okf_lint = { exit: 2, scope: null, lines: ["script not found"] };
  }

  // --- status-banner (spec-zone banner contract) ---
  const bannerScript = join(root, "scripts", "status-banner.mjs");
  if (existsSync(bannerScript)) {
    result.checks.status_banner = runScript(root, bannerScript, ["check"]);
  } else {
    result.checks.status_banner = { exit: 2, scope: null, lines: ["script not found"] };
  }

  // --- traceability (scenario → covering-test matrix, QA Step 5) ---
  // The counts are exact by construction; the QA skill renders the matrix.
  {
    const slug = findActiveSlug(root);
    if (slug) {
      result.checks.traceability = runRepoCheck(root, "traceability.mjs", [slug]);
    } else {
      result.checks.traceability = { exit: 0, lines: ["skipped: no active change"], summary: null };
    }
  }

  // --- tdd-evidence (RED→GREEN commit + test-file validation, QA Step 6) ---
  {
    const slug = findActiveSlug(root);
    if (slug) {
      result.checks.tdd_evidence = runRepoCheck(root, "tdd-evidence.mjs", [slug]);
    } else {
      result.checks.tdd_evidence = { exit: 0, lines: ["skipped: no active change"], summary: null };
    }
  }

  // --- review-ledger-lint (ledger schema validation, review-ledger Step 3) ---
  {
    const slug = findActiveSlug(root);
    if (slug) {
      result.checks.review_ledger = runRepoCheck(root, "review-ledger-lint.mjs", [slug]);
    } else {
      result.checks.review_ledger = { exit: 0, lines: ["skipped: no active change"], summary: null };
    }
  }

  // --- windows (broken-windows ledger schema, qa auto-append) ---
  // windows.mjs prints JSON (not the drift-guard SCOPE line), so fold it the
  // same way as runRepoCheck.
  {
    const scriptPath = join(root, "scripts", "windows.mjs");
    if (existsSync(scriptPath)) {
      try {
        const stdout = execFileSync("node", [scriptPath, "check", root], {
          cwd: root, encoding: "utf8", stdio: ["pipe", "pipe", "pipe"],
        });
        let parsed = null;
        try { parsed = JSON.parse(stdout); } catch {}
        result.checks.windows = {
          exit: 0,
          lines: parsed ? [`${parsed.windows} windows, next ${parsed.nextWindow}`] : stdout.trim().split("\n"),
          summary: parsed ? `${parsed.windows} windows, ${parsed.findings.length} schema findings` : null,
          detail: parsed ?? null,
        };
        if (parsed && parsed.findings.length > 0) result.checks.windows.exit = 1;
      } catch (e) {
        const msg = ((e.stdout || "") + "\n" + (e.stderr || "")).trim();
        let parsed = null;
        try { parsed = JSON.parse(e.stdout || ""); } catch {}
        result.checks.windows = {
          exit: e.status ?? 1,
          lines: parsed ? [`${parsed.windows} windows, ${parsed.findings.length} schema findings`] : (msg ? msg.split("\n") : ["no output"]),
          summary: parsed ? `${parsed.windows} windows, ${parsed.findings.length} schema findings` : null,
          detail: parsed ?? null,
        };
      }
    } else {
      result.checks.windows = { exit: 2, lines: ["script not found"], summary: null };
    }
  }

  // --- delivery-guard (the four guard lines, subagent-execution contract) ---
  {
    const slug = findActiveSlug(root);
    if (slug) {
      result.checks.delivery_guard = runRepoCheck(root, "delivery-guard.mjs", [slug]);
    } else {
      result.checks.delivery_guard = { exit: 0, lines: ["skipped: no active change"], summary: null };
    }
  }

  // --- composite scope (worst-scope-wins) ---
  let composite = "COMPLETE";
  for (const key of ["state_drift", "ship_drift", "okf_lint"]) {
    const s = result.scopes[key];
    if (s) composite = worstScope(composite, s);
  }
  result.scopes.composite = composite;

  // --- verdict ---
  let hasIssue = false;
  let hasError = false;
  for (const [, check] of Object.entries(result.checks)) {
    if (check.exit === 1) hasIssue = true;
    if (check.exit === 2) hasError = true;
  }

  if (hasError) result.verdict = "ERROR";
  else if (hasIssue) result.verdict = "ISSUES";

  console.log(JSON.stringify(result, null, 2));
  process.exit(result.verdict === "ERROR" ? 2 : result.verdict === "ISSUES" ? 1 : 0);
}

main();
