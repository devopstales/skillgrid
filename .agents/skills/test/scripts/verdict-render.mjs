#!/usr/bin/env node
// verdict-render.mjs — the four-state gate (test SKILL.md, Step 3 "Render the
// Verdict") as one call. The verdict matrix (PASS / CONCERNS / FAIL / WAIVED
// criteria + "the floor is the weakest dimension") is a pure function of the
// area signals; this script IS that function, so the agent stops re-implementing
// the truth table from prose each run.
//
// Input: one JSON object on stdin (or a file as the first argument) describing
// the run:
//   {
//     "areas": [
//       // one entry per area that RAN. Each signal is a string state;
//       // "N/A" means the area (or the signal's gate) is not applicable.
//       { "name": "intent",
//         "truths": "VERIFIED" | "PRESENT_BEHAVIOR_UNVERIFIED" | "UNVERIFIED",
//         "scope": "COMPLETE" | "TRUNCATED" | "UNSCOPED" | "UNREADABLE" },
//       { "name": "functional",
//         "scenarios": "COVERED" | "UNCOVERED",
//         "tests": "PASS" | "FAIL" | "N/A",
//         "scope": "COMPLETE" | ... },
//       { "name": "code-quality",
//         "gates": "PASS" | "N/A",            // coverage/mutation/lint/typecheck
//         "duplication": "PASS" | "ABOVE" | "N/A" },
//       { "name": "security",
//         "critical": "NONE" | "FOUND",
//         "warning": "NONE" | "FOUND",
//         "failOn": "BELOW" | "AT_ABOVE" | "N/A" },
//       { "name": "review",
//         "signal": "REVIEW-PASS" | "BACK-TO-APPLY-ACCEPTED" | "BACK-TO-APPLY-OPEN" | "N/A" }
//     ],
//     "p0_untriangulated": false,             // a P0 finding without triangulation
//     "coverage_dip": false,                  // changed-file coverage dip
//     "stale_verification": false,            // a STALE verification signal
//     "waived": false,                        // the human waived a specific criterion
//     "waiver_criterion": null,               // which criterion (required with waived)
//     "human_override": null                  // "accept-concerns" | "ship-it-anyway" | null
//   }
//
// Output: JSON on stdout —
//   { verdict, floor_area, floor_reasons, non_answers, needs_human, notes }
//
// Exit codes: 0 = rendered (any verdict, incl. FAIL — the render itself is
// not a failure), 2 = usage / unparseable input.
//
// The floor: the verdict can never be higher than the verdict the weakest
// area earns. Area value: FAIL < CONCERNS < PASS. Any non-answer area (scope
// not COMPLETE, UNVERIFIED truth, uncovered scenario, open review) caps the
// floor at CONCERNS. The human override accepts CONCERNS; it never accepts
// FAIL. WAIVED is a human decision, never a machine one.

import { readFileSync } from "node:fs";

function usage(msg) {
  if (msg) console.error(msg);
  console.error("usage: node verdict-render.mjs [input.json]   (or pipe JSON on stdin)");
  process.exit(2);
}

let raw;
if (process.argv[2]) {
  try {
    raw = readFileSync(process.argv[2], "utf8");
  } catch (e) {
    usage(`cannot read ${process.argv[2]}: ${e.message}`);
  }
} else {
  raw = readFileSync(0, "utf8");
}

let input;
try {
  input = JSON.parse(raw);
} catch (e) {
  usage(`input is not valid JSON: ${e.message}`);
}
if (!input || !Array.isArray(input.areas)) usage('input must be an object with an "areas" array');

const NOTES = [];
const nonAnswers = [];

// --- area value: FAIL < CONCERNS < PASS -----------------------------------------

const FAIL = 0, CONCERNS = 1, PASS = 2;

function areaValue(name, a) {
  const reasons = [];
  let v = PASS;

  const cap = (val, why) => {
    if (val < v) {
      v = val;
      if (why) reasons.push(why);
    }
  };

  if (name === "intent") {
    if (a.truths === "UNVERIFIED") cap(FAIL, "a truth is UNVERIFIED");
    if (a.truths === "PRESENT_BEHAVIOR_UNVERIFIED") cap(CONCERNS, "a truth is PRESENT_BEHAVIOR_UNVERIFIED");
  } else if (name === "functional") {
    if (a.scenarios === "UNCOVERED") cap(FAIL, "a scenario has no covering test");
    if (a.tests === "FAIL") cap(FAIL, "a test ran and failed");
  } else if (name === "code-quality") {
    if (a.gates === "FAIL") cap(FAIL, "a code-quality gate is below threshold");
    if (a.duplication === "ABOVE") cap(FAIL, "duplication % above fail_on");
  } else if (name === "security") {
    if (a.critical === "FOUND") cap(FAIL, "a CRITICAL security finding");
    if (a.warning === "FOUND") cap(CONCERNS, "a WARNING security finding");
    if (a.failOn === "AT_ABOVE") cap(FAIL, "Trivy finding at or above fail_on");
  } else if (name === "review") {
    if (a.signal === "BACK-TO-APPLY-OPEN") cap(FAIL, "an unresolved high/medium review entry (BACK-TO-APPLY open)");
  } else {
    cap(FAIL, `unknown area "${name}" — a value the gate cannot read is a non-answer, not a pass`);
  }

  // scope: fail-closed. Only COMPLETE satisfies the gate; any other scope is
  // a non-answer that caps at CONCERNS (a named re-run), never a silent PASS.
  if (a.scope && a.scope !== "COMPLETE") {
    cap(CONCERNS, `verification scope ${a.scope} (needs a re-run with full scope)`);
  }

  return { v, reasons };
}

// --- render -------------------------------------------------------------------

let floor = PASS;
let floorArea = null;
const floorReasons = [];

for (const a of input.areas) {
  const { v, reasons } = areaValue(a.name, a);
  if (v < floor) {
    floor = v;
    floorArea = a.name;
    floorReasons.length = 0;
    floorReasons.push(...reasons);
  } else if (v === floor && reasons.length) {
    floorReasons.push(...reasons);
  }
  if (a.scope && a.scope !== "COMPLETE") nonAnswers.push(`${a.name}: scope ${a.scope}`);
}

// whole-run non-answers (apply the floor across the run, not per area)
if (input.p0_untriangulated) {
  if (floor > CONCERNS) { floor = CONCERNS; floorReasons.push("a P0 finding without triangulation"); }
  nonAnswers.push("p0: untriangulated");
}
if (input.coverage_dip) {
  if (floor > CONCERNS) { floor = CONCERNS; floorReasons.push("a changed-file coverage dip"); }
  nonAnswers.push("coverage dip");
}
if (input.stale_verification) {
  if (floor > CONCERNS) { floor = CONCERNS; floorReasons.push("a STALE verification"); }
  nonAnswers.push("stale verification");
}

let verdict = floor === FAIL ? "FAIL" : floor === CONCERNS ? "CONCERNS" : "PASS";

// WAIVED: a human decision, never a machine one. The waiver must name the
// criterion it waives — an unnamed waiver is a no-op.
let needsHuman = false;
if (input.waived) {
  if (!input.waiver_criterion) {
    NOTES.push("waived=true without waiver_criterion — the waiver is not recorded; treating the run as unwaived");
  } else {
    verdict = "WAIVED";
    NOTES.push(`waiver recorded for: ${input.waiver_criterion}`);
  }
}

// human override: accepts CONCERNS (and a recorded review "ship it anyway");
// never accepts a FAIL floor — a FAIL is re-run, not waved.
if (verdict === "FAIL") {
  if (input.human_override) {
    NOTES.push(`human_override "${input.human_override}" does not clear a FAIL floor — re-run with passing evidence`);
  }
} else if (verdict === "CONCERNS") {
  if (input.human_override) {
    verdict = "PASS";
    NOTES.push(`human override "${input.human_override}" accepts the CONCERNS floor (recorded in the report)`);
  } else {
    needsHuman = true;
  }
}

const result = {
  verdict,
  floor_area: floorArea,
  floor_reasons: floorReasons,
  non_answers: nonAnswers,
  needs_human: needsHuman,
  notes: NOTES,
};

console.log(JSON.stringify(result, null, 2));
process.exit(0);
