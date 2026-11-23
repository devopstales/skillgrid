#!/usr/bin/env node
// test-validate-envelope.mjs — prove the subset validator and the in-band
// rule. Negative halves first (a checker that never caught a broken
// envelope is not wired): each broken copy below is a real drift the
// validator must reject, with the right rule named. Then every fixture
// validates, and the positive in-band cases pass.
//
// Usage: node test-validate-envelope.mjs
// Exit: 0 all pass, 1 any fail.

import { execFileSync } from "node:child_process";
import { dirname, join } from "node:path";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const VALIDATE = join(HERE, "validate-envelope.mjs");
const SCHEMAS = join(HERE, "..", "schemas", "pipeline");

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

function validate(schemaName, envelope) {
  try {
    const out = execFileSync("node", [VALIDATE, join(SCHEMAS, schemaName), "-"], {
      input: JSON.stringify(envelope),
      encoding: "utf8",
    });
    return { code: 0, json: JSON.parse(out), out };
  } catch (e) {
    const stdout = (e.stdout || "").trim();
    let json = null;
    try { json = JSON.parse(stdout); } catch { /* usage error: no JSON */ }
    return { code: e.status, json, out: stdout + (e.stderr || "") };
  }
}

function finding(r, rule) {
  return r.json && Array.isArray(r.json.findings)
    ? r.json.findings.some((f) => f.rule === rule)
    : false;
}

const fixture = (name) => JSON.parse(readFileSync(join(SCHEMAS, "fixtures", name), "utf8"));

// --- 1. negatives: gate-result ----------------------------------------------------

const base = fixture("gate-result-allow.json");
let r;

r = validate("gate-result.schema.json", { ...base, next: undefined });
check("gr-missing-next: invalid", r.code === 1 && r.json && r.json.valid === false, r.out);
check("gr-missing-next: rule required", finding(r, "required"), r.out);

r = validate("gate-result.schema.json", { ...base, surprise: true });
check("gr-unknown-field: invalid", r.code === 1, r.out);
check("gr-unknown-field: rule additionalProperties", finding(r, "additionalProperties"), r.out);

r = validate("gate-result.schema.json", { ...base, schema: "skillgrid.gate-result/v2" });
check("gr-wrong-schema-const: invalid", r.code === 1, r.out);
check("gr-wrong-schema-const: rule const", finding(r, "const"), r.out);

r = validate("gate-result.schema.json", { ...base, allowed: false });
check("gr-allow-but-not-allowed: invalid", r.code === 1, r.out);
check("gr-allow-but-not-allowed: $.allowed const", finding(r, "const"), r.out);

const blocked = fixture("gate-result-blocked.json");
r = validate("gate-result.schema.json", { ...blocked, allowed: true });
check("gr-blocked-but-allowed: invalid", r.code === 1, r.out);
check("gr-blocked-but-allowed: $.allowed const", finding(r, "const"), r.out);

r = validate("gate-result.schema.json", { ...blocked, next: { reason_code: "qa-failed", step: "fix area <floor_area> and re-run" } });
check("gr-template-step: invalid", r.code === 1, r.out);
check("gr-template-step: rule in-band", finding(r, "in-band"), r.out);

r = validate("gate-result.schema.json", { ...blocked, next: { reason_code: "qa-failed", step: "the human decides" } });
check("gr-abstract-step: invalid", r.code === 1, r.out);
check("gr-abstract-step: rule in-band", finding(r, "in-band"), r.out);

r = validate("gate-result.schema.json", { ...base, notes: undefined });
check("gr-qa-verdict-missing-notes: invalid", r.code === 1, r.out);
check("gr-qa-verdict-missing-notes: rule required", finding(r, "required"), r.out);

r = validate("gate-result.schema.json", { ...base, subject: { feature: base.subject["acceptance.feature"] } });
check("gr-bad-subject-key: invalid", r.code === 1, r.out);
check("gr-bad-subject-key: rule propertyNames", finding(r, "propertyNames"), r.out);

r = validate("gate-result.schema.json", { ...base, subject: { "acceptance.feature": "abc123" } });
check("gr-bad-subject-digest: invalid", r.code === 1, r.out);
check("gr-bad-subject-digest: rule pattern", finding(r, "pattern"), r.out);

// --- 2. negatives: gate-consent ----------------------------------------------------

const consent = fixture("gate-consent-user-gate-slice.json");
r = validate("gate-consent.schema.json", { ...consent, choices: [...consent.choices, consent.choices[0]] });
check("gc-three-choices: invalid", r.code === 1, r.out);
check("gc-three-choices: rule maxItems", finding(r, "maxItems"), r.out);

r = validate("gate-consent.schema.json", { ...consent, off_path: undefined });
check("gc-missing-off-path: invalid", r.code === 1, r.out);
check("gc-missing-off-path: rule required", finding(r, "required"), r.out);

r = validate("gate-consent.schema.json", { ...consent, choices: [consent.choices[0], { ...consent.choices[1], answer: "granted" }] });
check("gc-second-choice-not-declined: invalid", r.code === 1, r.out);
check("gc-second-choice-not-declined: rule const", finding(r, "const"), r.out);

r = validate("gate-consent.schema.json", { ...consent, choices: [{ ...consent.choices[0], next_step: "ok" }] });
check("gc-abstract-next-step: invalid", r.code === 1, r.out);
check("gc-abstract-next-step: rule in-band", finding(r, "in-band"), r.out);

// --- 3. negatives: phase-status ----------------------------------------------------

const pending = fixture("phase-status-gate-pending.json");
const stopped = {
  ...pending,
  next_transition: { kind: "stop", reason_code: "blocked" },
};
r = validate("phase-status.schema.json", stopped);
check("ps-stop-no-continuation: invalid", r.code === 1, r.out);
check("ps-stop-no-continuation: rule required", finding(r, "required"), r.out);

r = validate("phase-status.schema.json", { ...stopped, next_transition: { ...stopped.next_transition, continuation: { step: "the flow is over" } } });
check("ps-stop-abstract-step: invalid", r.code === 1, r.out);
check("ps-stop-abstract-step: rule in-band", finding(r, "in-band"), r.out);

const executing = fixture("phase-status-execute.json");
r = validate("phase-status.schema.json", { ...executing, next_transition: { kind: "execute", reason_code: "slice-approved" } });
check("ps-execute-no-execute: invalid", r.code === 1, r.out);
check("ps-execute-no-execute: rule required", finding(r, "required"), r.out);

// --- 4. positives --------------------------------------------------------------------

const FAMILY = {
  "gate-result-allow.json": "gate-result",
  "gate-result-blocked.json": "gate-result",
  "gate-result-scope-changed.json": "gate-result",
  "gate-consent-user-gate-slice.json": "gate-consent",
  "phase-status-execute.json": "phase-status",
  "phase-status-gate-pending.json": "phase-status",
};
for (const [name, family] of Object.entries(FAMILY)) {
  r = validate(`${family}.schema.json`, fixture(name));
  check(`fixture ${name}: valid`, r.code === 0 && r.json && r.json.valid === true, r.out);
}

r = validate("phase-status.schema.json", {
  ...stopped,
  next_transition: {
    kind: "stop",
    reason_code: "blocked",
    continuation: { step: "re-run node .agents/skills/_shared/scripts/validate-envelope.mjs after fixing report.md" },
  },
});
check("ps-stop-runnable-step: valid", r.code === 0 && r.json && r.json.valid === true, r.out);

// --- 5. usage -------------------------------------------------------------------------

r = { code: 2 };
try {
  execFileSync("node", [VALIDATE], { encoding: "utf8", stdio: "pipe" });
} catch (e) {
  r = { code: e.status, out: (e.stderr || "").trim() };
}
check("usage: no args exits 2", r.code === 2, r.out);

console.log(`\nvalidate-envelope tests: ${passed} passed, ${failed} failed`);
if (failed > 0) {
  console.log(`failing: ${failures.join("; ")}`);
  process.exit(1);
}
