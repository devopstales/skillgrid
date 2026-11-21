#!/usr/bin/env node
// area-scope.mjs — emit the area-selection table for a change: which of the
// five test areas to run, which are N/A, and why. Reads the change dir's
// artifacts to classify the change shape, then prints a table the controller
// reads in one call.
//
// Usage: node area-scope.mjs CHANGE_DIR [TARGET_URL]
//   CHANGE_DIR  the change dir under .skillgrid/change-specs/ (has briefing.md,
//               acceptance.feature, blueprint.md, tasks.md)
//   TARGET_URL  (optional) a reachable, authorized URL. The controller asks the
//               human for it in Step 0 before calling this; presence selects the
//               DAST security tools (nuclei/wapiti/akca) and the browser pass.
//               Omit it (or pass none) when the human says there is no target.
//
// Exit codes (agentic script contract, _shared/craft/deterministic-boundary.md):
//   0  selection made (table on stdout)
//   2  usage error / CHANGE_DIR missing
import { existsSync, readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

function usage(msg) {
  if (msg) console.error(msg);
  console.error("usage: node area-scope.mjs CHANGE_DIR [TARGET_URL]");
  process.exit(2);
}

const dir = process.argv[2];
const url = process.argv[3] || "";
if (!dir) usage("missing CHANGE_DIR");
if (!existsSync(dir)) usage(`no such change dir: ${dir}`);

const briefing = join(dir, "briefing.md");
const acceptance = join(dir, "acceptance.feature");
const tasks = join(dir, "tasks.md");

function text(p) {
  return existsSync(p) ? readFileSync(p, "utf8") : "";
}

// --- classify the change shape from the artifacts ---------------------------

const has_briefing = existsSync(briefing);
const has_scenarios = /scenario|feature|rule/i.test(text(acceptance));

// trust-boundary signal: any .md artifact names auth, input, a dependency, or config
let trust = false;
for (const f of readdirSync(dir)) {
  if (f.endsWith(".md") && /auth|login|session|token|permission|input|param|upload|webhook|deserial|ssrf|dependenc|package|require|import|dockerfile|terraform|k8s|iac|config|secret|credential/i.test(text(join(dir, f)))) {
    trust = true;
    break;
  }
}

// refactor-only signal: tasks.md names only move/rename/refactor/split
const tasksText = text(tasks);
const refactor =
  /rename|move|refactor|split|extract|reorganize/i.test(tasksText) &&
  !/feature|add |new |implement|support/i.test(tasksText);

// --- decide each area --------------------------------------------------------

// Intent: run when a requirement (briefing + scenarios) is in scope.
const intent = has_briefing || has_scenarios
  ? "RUN"
  : "N/A — no requirement (briefing/acceptance.feature absent)";

// Functional: always run the suite; browser pass only when a URL is reachable.
let functional = "RUN";
if (url) functional = `RUN (incl. browser pass — ${url})`;

// Code quality: always run at least the exact-clone pass.
const code_quality = "RUN (exact-clone pass minimum)";

// Security: trivy always (static, no target needed). DAST tools only with a URL.
let security = "RUN (trivy)";
if (trust) security = "RUN (trivy + owasp — trust boundary touched)";
if (url) security += "; DAST available (nuclei/wapiti/akca — pick per references/security.md selection table)";

// Review: the diff is the point of a refactor; for other shapes it runs after a
// PASS/CONCERNS intent+functional gate (and is skipped at T0 per rigor-tiers).
const review = refactor
  ? "RUN (the diff is the point)"
  : "RUN (after a PASS/CONCERNS intent+functional gate; skip at T0)";

// --- emit the table ----------------------------------------------------------

console.log(`# Area selection for: ${dir}`);
console.log(url ? `Target URL: ${url}` : "Target URL: (none — DAST tools not selectable)");
console.log("");
console.log("| Area | Decision | Why |");
console.log("|---|---|---|");
console.log(intent === "RUN"
  ? `| Intent | ${intent} | requirement in scope (briefing/scenarios present) |`
  : `| Intent | ${intent} | |`);
console.log(`| Functional | ${functional} | suite always; browser pass needs a reachable URL |`);
console.log(`| Code quality | ${code_quality} | duplication/complexity lens; minimum exact-clone pass |`);
console.log(`| Security | ${security} | trivy static; owasp on trust boundary; DAST needs a target |`);
console.log(`| Review | ${review} | standards/spec gate; runs after intent+functional pass |`);
console.log("");
console.log("Run order: intent -> functional -> code quality -> security -> review.");
console.log("An area not selected is recorded 'N/A — <reason>' in the report, never silently absent.");
