#!/usr/bin/env node
// ship-gates.mjs — the pre-mutation gate reader (ship SKILL.md, "Gates (all
// must pass before ANY mutation)"). One call returns the machine verdict of
// both gates so "all must pass" is a JSON field, not a re-reading of two
// markdown files.
//
// Reads (read-only, pre-move — the files are still in change-specs/):
//   report.md  → ## Gate Decision verdict (PASS / CONCERNS / FAIL / WAIVED)
//                + unresolved CRITICAL scan
//   review.md  → ## Verdict (met / met-with-fixes / not met) + Grade C axes
//
// Usage: node ship-gates.mjs <change-dir>
//
// Contract (per _shared/craft/deterministic-boundary.md "agentic script
// contract"): JSON on stdout, diagnostics on stderr, exit 0 = every gate
// allows ship, exit 1 = a gate blocks or warns (findings say which),
// exit 2 = usage / files missing or unparseable.
//
// The script only READS. The human-override decisions (CONCERNS cleared,
// review "ship it anyway") are the agent's call — the JSON names what the
// human must decide; it never decides for them.

import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

function usage(msg) {
  if (msg) console.error(msg);
  console.error("usage: node ship-gates.mjs <change-dir>");
  process.exit(2);
}

const changeDir = process.argv[2];
if (!changeDir) usage("missing <change-dir>");

const reportPath = join(changeDir, "report.md");
const reviewPath = join(changeDir, "review.md");
if (!existsSync(reportPath)) usage(`no report.md in ${changeDir} (the QA half must exist before ship)`);

// --- extract the report.md verdict ---------------------------------------------

function extractVerdict(text) {
  const lines = text.split("\n");
  let inGate = false;
  for (const line of lines) {
    if (/^##\s+Gate Decision\s*$/.test(line.trim())) {
      inGate = true;
      continue;
    }
    if (inGate) {
      if (/^##\s+/.test(line)) break; // next H2 ends the section
      const m = line.match(/\*\*Verdict:\*\*\s*(.+)$/);
      if (m) {
        const rest = m[1].trim();
        // the placeholder ("PASS / CONCERNS / FAIL / WAIVED") and PENDING are
        // not a rendered verdict — read them as null, never as their first word
        if (/^PASS\s*\/\s*CONCERNS/i.test(rest) || rest.toUpperCase() === "PENDING") return null;
        const v = rest.split(/\s*\//)[0].trim().toUpperCase();
        if (["PASS", "CONCERNS", "FAIL", "WAIVED"].includes(v)) return v;
        return null;
      }
    }
  }
  return null;
}

function extractUnresolvedCriticals(text) {
  const out = [];
  const lines = text.split("\n");
  let inGate = false;
  for (const raw of lines) {
    const line = raw.trim();
    if (/^##\s+Gate Decision\s*$/.test(line)) inGate = true;
    if (inGate && /^##\s+/.test(line) && !/Gate Decision/.test(line)) inGate = false;
    if (!inGate) continue;
    const m = line.match(/^[-*]\s*\[([ xX])\]\s*(.*CRITICAL.*)$/i) || line.match(/^[-*]\s*\[([ xX])\]\s*(.*\bCRITICAL\b.*)$/i);
    if (m) {
      const done = m[1].toLowerCase() === "x";
      const text_ = m[2].replace(/\[([ xX])\]\s*/, "").trim();
      if (!done) out.push(text_);
    }
  }
  return out;
}

// --- extract the review.md verdict + Grade C axes --------------------------------

function extractReviewVerdict(text) {
  const m = text.match(/\*\*Verdict:\*\*\s*(met-with-fixes|met|not met|MET-WITH-FIXES|MET|NOT MET)/i);
  if (!m) return null;
  const v = m[1].toLowerCase();
  return v === "met-with-fixes" ? "met-with-fixes" : v === "met" ? "met" : "not met";
}

function extractGradeCAxes(text) {
  const out = [];
  for (const line of text.split("\n")) {
    const m = line.match(/\|\s*([^|]+?)\s*\|\s*(?:Grade\s+)?C\s*\|/i);
    if (m) {
      const axis = m[1].trim();
      // skip the header row and the separator row
      if (/^[-|:\s]+$/.test(axis) || /^axis$/i.test(axis)) continue;
      out.push(axis);
    }
  }
  return out;
}

// --- evaluate ------------------------------------------------------------------

const report = readFileSync(reportPath, "utf8");
const qaVerdict = extractVerdict(report);
const criticals = extractUnresolvedCriticals(report);

let qa = null;
let qaFindings = [];
if (qaVerdict === null) {
  qa = "unverifiable";
  qaFindings.push("report.md has no completed ## Gate Decision verdict line (QA gate not rendered)");
} else {
  qa = qaVerdict;
  if (qa === "FAIL") qaFindings.push("QA verdict is FAIL");
  if (criticals.length) qaFindings.push(`${criticals.length} unresolved CRITICAL finding(s): ${criticals.join("; ")}`);
  if (qa === "CONCERNS" && criticals.length === 0 && qaVerdict !== "FAIL") {
    // CONCERNS is advisory — surfaced, cleared by a human override
    qaFindings.push("QA verdict is CONCERNS (advisory — human override required to proceed)");
  }
}

const qaBlocks = qa === "FAIL" || (qa === "unverifiable") || criticals.length > 0;
const qaNeedsOverride = qa === "CONCERNS";

let review = "absent";
let reviewFindings = [];
if (existsSync(reviewPath)) {
  const reviewText = readFileSync(reviewPath, "utf8");
  const rv = extractReviewVerdict(reviewText);
  if (rv === null) {
    review = "unverifiable";
    reviewFindings.push("review.md has no completed ## Verdict line");
  } else {
    review = rv;
    if (rv === "not met") reviewFindings.push("review verdict is not met (advisory — human owns the decision)");
    const axes = extractGradeCAxes(reviewText);
    if (axes.length) reviewFindings.push(`Grade C axis(es): ${axes.join(", ")} (advisory — human owns the decision)`);
  }
} else {
  reviewFindings.push("review.md absent (the review gate has not run — advisory, does not block on its own)");
}
const reviewNeedsOverride = review === "not met" || review === "unverifiable" || review === "absent";

const blocked = qaBlocks;

const result = {
  qa: qa,
  qa_blocks: qaBlocks,
  qa_needs_override: qaNeedsOverride,
  review: review,
  review_needs_override: reviewNeedsOverride,
  blocked: blocked,
  block_reason: blocked
    ? (qa === "unverifiable" ? "qa-unverifiable" : criticals.length ? "qa-critical-unresolved" : "qa-failed")
    : null,
  findings: [...qaFindings, ...reviewFindings],
};

console.log(JSON.stringify(result, null, 2));
process.exit(blocked ? 1 : 0);
