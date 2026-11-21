#!/usr/bin/env node
// test-verdict-render.mjs — prove the four-state gate truth table: every
// matrix row, the floor (weakest dimension), the fail-closed scope rule,
// the WAIVED-is-a-human-decision rule, and the override semantics.
//
// Usage: node test-verdict-render.mjs
// Exit: 0 all pass, 1 any fail.

import { execFileSync } from "node:child_process";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const RENDER = join(HERE, "verdict-render.mjs");

let passed = 0;
let failed = 0;
const failures = [];

function check(name, cond, detail = "") {
  if (cond) passed++;
  else {
    failed++;
    failures.push(name);
    console.log(`  FAIL ${name}${detail ? ` — ${detail}` : ""}`);
  }
}

function render(input) {
  try {
    const out = execFileSync("node", [RENDER], { input: JSON.stringify(input), encoding: "utf8" });
    return { code: 0, json: JSON.parse(out), out };
  } catch (e) {
    return { code: e.status, json: null, out: (e.stdout || "") + (e.stderr || "") };
  }
}

const intentPass = { name: "intent", truths: "VERIFIED", scope: "COMPLETE" };
const intentUnverified = { name: "intent", truths: "UNVERIFIED", scope: "COMPLETE" };
const intentPartial = { name: "intent", truths: "PRESENT_BEHAVIOR_UNVERIFIED", scope: "COMPLETE" };
const functionalPass = { name: "functional", scenarios: "COVERED", tests: "PASS", scope: "COMPLETE" };
const functionalFail = { name: "functional", scenarios: "COVERED", tests: "FAIL", scope: "COMPLETE" };
const functionalUncovered = { name: "functional", scenarios: "UNCOVERED", tests: "PASS", scope: "COMPLETE" };
const cqPass = { name: "code-quality", gates: "PASS", duplication: "N/A" };
const cqDup = { name: "code-quality", gates: "PASS", duplication: "ABOVE" };
const secPass = { name: "security", critical: "NONE", warning: "NONE", failOn: "N/A" };
const secCritical = { name: "security", critical: "FOUND", warning: "NONE", failOn: "N/A" };
const secWarn = { name: "security", critical: "NONE", warning: "FOUND", failOn: "N/A" };
const reviewPass = { name: "review", signal: "REVIEW-PASS" };
const reviewOpen = { name: "review", signal: "BACK-TO-APPLY-OPEN" };

// --- 1. all green -> PASS -------------------------------------------------------
let r = render({ areas: [intentPass, functionalPass, cqPass, secPass, reviewPass] });
check("all-green: verdict PASS", r.code === 0 && r.json.verdict === "PASS", r.out);
check("all-green: no non-answers", r.json.non_answers.length === 0, r.out);

// --- 2. UNVERIFIED truth -> FAIL -------------------------------------------------
r = render({ areas: [intentUnverified, functionalPass, cqPass, secPass, reviewPass] });
check("unverified-truth: verdict FAIL", r.json.verdict === "FAIL", r.out);
check("unverified-truth: floor is intent", r.json.floor_area === "intent", r.out);

// --- 3. PRESENT_BEHAVIOR_UNVERIFIED -> CONCERNS, needs human ---------------------
r = render({ areas: [intentPartial, functionalPass, cqPass, secPass, reviewPass] });
check("partial-truth: verdict CONCERNS", r.json.verdict === "CONCERNS", r.out);
check("partial-truth: needs_human", r.json.needs_human === true, r.out);

// --- 4. failing test -> FAIL ------------------------------------------------------
r = render({ areas: [intentPass, functionalFail, cqPass, secPass, reviewPass] });
check("failing-test: verdict FAIL", r.json.verdict === "FAIL", r.out);

// --- 5. uncovered scenario -> FAIL -------------------------------------------------
r = render({ areas: [intentPass, functionalUncovered, cqPass, secPass, reviewPass] });
check("uncovered-scenario: verdict FAIL", r.json.verdict === "FAIL", r.out);

// --- 6. CRITICAL security -> FAIL ---------------------------------------------------
r = render({ areas: [intentPass, functionalPass, cqPass, secCritical, reviewPass] });
check("critical-security: verdict FAIL", r.json.verdict === "FAIL", r.out);

// --- 7. WARNING security -> CONCERNS -------------------------------------------------
r = render({ areas: [intentPass, functionalPass, cqPass, secWarn, reviewPass] });
check("warning-security: verdict CONCERNS", r.json.verdict === "CONCERNS", r.out);

// --- 8. duplication above -> FAIL -----------------------------------------------------
r = render({ areas: [intentPass, functionalPass, cqDup, secPass, reviewPass] });
check("dup-above: verdict FAIL", r.json.verdict === "FAIL", r.out);

// --- 9. review open -> FAIL; accepted back-to-apply -> PASS -----------------------------
r = render({ areas: [intentPass, functionalPass, cqPass, secPass, reviewOpen] });
check("review-open: verdict FAIL", r.json.verdict === "FAIL", r.out);
const reviewAccepted = { name: "review", signal: "BACK-TO-APPLY-ACCEPTED" };
r = render({ areas: [intentPass, functionalPass, cqPass, secPass, reviewAccepted] });
check("review-accepted: verdict PASS", r.json.verdict === "PASS", r.out);

// --- 10. the floor: one FAIL area sinks an otherwise-green run ---------------------------
r = render({ areas: [intentPass, functionalPass, cqPass, secCritical, reviewPass] });
check("floor: green run + one FAIL area = FAIL", r.json.verdict === "FAIL", r.out);
check("floor: floor area named", r.json.floor_area === "security", r.out);

// --- 11. fail-closed scope: TRUNCATED caps at CONCERNS, never a silent PASS ----------------
r = render({ areas: [{ ...intentPass, scope: "TRUNCATED" }, functionalPass, cqPass, secPass, reviewPass] });
check("scope-truncated: verdict CONCERNS", r.json.verdict === "CONCERNS", r.out);
check("scope-truncated: recorded as non-answer", r.json.non_answers.some((n) => n.includes("TRUNCATED")), r.out);

// --- 12. P0 untriangulated caps at CONCERNS -----------------------------------------------
r = render({ areas: [intentPass, functionalPass, cqPass, secPass, reviewPass], p0_untriangulated: true });
check("p0-untri: verdict CONCERNS", r.json.verdict === "CONCERNS", r.out);

// --- 13. stale verification caps at CONCERNS -----------------------------------------------
r = render({ areas: [intentPass, functionalPass, cqPass, secPass, reviewPass], stale_verification: true });
check("stale: verdict CONCERNS", r.json.verdict === "CONCERNS", r.out);

// --- 14. WAIVED without a criterion is not a waiver ------------------------------------------
r = render({ areas: [intentPass, functionalPass, cqPass, secPass, reviewPass], waived: true });
check("waived-no-criterion: stays PASS with a note", r.json.verdict === "PASS" && r.json.notes.length === 1, r.out);

// --- 15. WAIVED with a criterion -> WAIVED ----------------------------------------------------
r = render({ areas: [intentPass, functionalPass, cqPass, secPass, reviewPass], waived: true, waiver_criterion: "functional browser pass" });
check("waived-with-criterion: verdict WAIVED", r.json.verdict === "WAIVED", r.out);

// --- 16. human override accepts CONCERNS, never FAIL --------------------------------------------
r = render({ areas: [intentPartial, functionalPass, cqPass, secPass, reviewPass], human_override: "accept-concerns" });
check("override-concerns: verdict PASS", r.json.verdict === "PASS", r.out);
r = render({ areas: [intentUnverified, functionalPass, cqPass, secPass, reviewPass], human_override: "accept-concerns" });
check("override-fail: stays FAIL with a note", r.json.verdict === "FAIL" && r.json.notes.some((n) => n.includes("does not clear a FAIL")), r.out);

// --- 17. unknown area is a non-answer, not a pass -----------------------------------------------
r = render({ areas: [intentPass, functionalPass, cqPass, secPass, reviewPass, { name: "mystery", whatever: "x" }] });
check("unknown-area: verdict FAIL (unreadable value is not a pass)", r.json.verdict === "FAIL", r.out);

// --- 18. usage: no areas -> exit 2 ------------------------------------------------------------------
r = render({});
check("usage: missing areas exits 2", r.code === 2, r.out);

console.log(`\nverdict-render tests: ${passed} passed, ${failed} failed`);
if (failed > 0) {
  console.error(failures.join("\n"));
  process.exit(1);
}
