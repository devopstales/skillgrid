#!/usr/bin/env node
// test-area-scope.mjs — prove the area-selection table: each change shape
// classifies to the right areas, and usage errors exit 2.
//
// Hermetic: builds throwaway change dirs under mkdtemp; torn down on exit.
//
// Usage: node test-area-scope.mjs
// Exit code: 0 if all pass, 1 if any fail.
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const scope = join(here, "area-scope.mjs");

let pass = 0, fail = 0;
const work = mkdtempSync(join(tmpdir(), "skillgrid-area-scope-"));

function run(args) {
  try {
    const out = execFileSync("node", [scope, ...args], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
    return { rc: 0, out };
  } catch (e) {
    return { rc: e.status ?? 1, out: `${e.stdout || ""}${e.stderr || ""}` };
  }
}

function check(label, haystack, needle) {
  if (haystack.includes(needle)) { console.log(`  ok: ${label}`); pass++; }
  else { console.log(`  FAIL: ${label} (missing: ${needle})`); fail++; }
}
function notcheck(label, haystack, needle) {
  if (haystack.includes(needle)) { console.log(`  FAIL: ${label} (unexpected: ${needle})`); fail++; }
  else { console.log(`  ok: ${label}`); pass++; }
}

try {
  // --- usage error: missing change dir ----------------------------------------
  let r = run([join(work, "nope")]);
  if (r.rc === 2) { console.log("ok: usage error exit 2"); pass++; }
  else { console.log(`FAIL: usage error (exit ${r.rc})`); fail++; }
  check("usage error message", r.out, "no such change dir");

  // --- behavior change, no URL -------------------------------------------------
  const c = join(work, "change-auth");
  mkdirSync(c, { recursive: true });
  writeFileSync(join(c, "briefing.md"), "Goal: add OAuth login\nTruth: user can view own profile\n");
  writeFileSync(join(c, "acceptance.feature"), "### Scenario: successful login\n");
  writeFileSync(join(c, "blueprint.md"), "Tier: T2\nAdd /login route, session middleware, token handling.\n");
  writeFileSync(join(c, "tasks.md"), "Add login endpoint\n");
  r = run([c]);
  check("behavior: intent RUN", r.out, "| Intent | RUN |");
  check("behavior: functional RUN", r.out, "| Functional | RUN |");
  check("behavior: code quality RUN", r.out, "| Code quality | RUN");
  check("behavior: security trivy+owasp", r.out, "trivy + owasp");
  check("behavior: review RUN", r.out, "| Review | RUN");
  notcheck("behavior: no DAST without URL", r.out, "DAST available");
  notcheck("behavior: no browser pass", r.out, "browser pass — ");

  // --- behavior change, WITH URL ----------------------------------------------
  r = run([c, "http://localhost:3000"]);
  check("url: browser pass selected", r.out, "browser pass — http://localhost:3000");
  check("url: DAST available", r.out, "DAST available");

  // --- refactor-only, no requirement -------------------------------------------
  const rf = join(work, "change-refactor");
  mkdirSync(rf, { recursive: true });
  writeFileSync(join(rf, "tasks.md"), "Rename a to b\nMove helpers into shared module\n");
  r = run([rf]);
  check("refactor: intent N/A", r.out, "| Intent | N/A");
  check("refactor: functional still RUN", r.out, "| Functional | RUN |");
  check("refactor: security trivy only", r.out, "| Security | RUN (trivy) |");
  check("refactor: review RUN (diff is the point)", r.out, "the diff is the point");
} finally {
  rmSync(work, { recursive: true, force: true });
}

console.log("");
console.log(`area-scope: ${pass} passed, ${fail} failed`);
process.exit(fail === 0 ? 0 : 1);
