#!/usr/bin/env bun
/**
 * Self-validation backstop (skillgrid SKILL.md § Verification): the new
 * checkers run against THIS repo's live spec zone — negative half first (a
 * deliberately broken copy is caught), positive half after (clean writes pass).
 *
 * Usage: bun scripts/backstop.mjs [project-root]
 * Exit 0 = both halves proven; 1 = a half failed; 2 = no active change found.
 */
import { checkWriteEdit } from "../src/guards.ts";

const root = process.argv[2] ?? process.cwd();

let failures = 0;
const check = (label, actual, want) => {
  const ok =
    want === "block" ? actual?.block === true : want === "pass" ? actual === undefined : actual?.reason !== undefined;
  console.log(`${ok ? "PASS" : "FAIL"}  ${label}`);
  if (!ok) {
    failures++;
    console.log(`      got: ${JSON.stringify(actual)}`);
  }
};

// --- negative half: deliberately broken copies --------------------------------
{
  const d = await checkWriteEdit(root, "write", {
    path: `${root}/.skillgrid/change-specs/2026-11-23-backstop-probe/plan.md`,
    content: "# plan (invented name)\n",
  });
  check("negative: invented name plan.md in a change dir is blocked", d, "block");

  const d2 = await checkWriteEdit(root, "write", {
    path: `${root}/.skillgrid/change-specs/2026-11-23-backstop-probe/briefing.md`,
    content: "# B\n\n> **STATUS:** `approved` (2026-11-23)\n",
  });
  check("negative: agent minting approved on a fresh briefing is blocked", d2, "block");

  const d3 = await checkWriteEdit(root, "write", {
    path: `${root}/.skillgrid/change-specs/2026-11-23-backstop-probe/tasks.md`,
    content: "# T\n\n> **STATUS:** `sliced` (2026-11-23)\n",
  });
  check("negative: tasks.md with no acceptance.feature is blocked", d3, "block");
}

// --- positive half: canonical writes pass --------------------------------------
{
  const d = await checkWriteEdit(root, "write", {
    path: `${root}/.skillgrid/change-specs/2026-11-23-backstop-probe/adr.md`,
    content: "# ADR manifest\n",
  });
  check("positive: adr.md (no predecessor) passes", d, "pass");

  const d2 = await checkWriteEdit(root, "write", {
    path: `${root}/.skillgrid/change-specs/2026-11-23-backstop-probe/findings.md`,
    content: "# findings\n## Research: x\n",
  });
  check("positive: findings.md (no predecessor) passes", d2, "pass");

  const d3 = await checkWriteEdit(root, "write", {
    path: `${root}/.skillgrid/change-specs/2026-11-23-backstop-probe/blueprint.council.md`,
    content: "# council notes\n",
  });
  check("positive: council notes beside artifacts pass", d3, "pass");
}

console.log(failures === 0 ? "backstop: all halves proven" : `backstop: ${failures} failure(s)`);
process.exit(failures === 0 ? 0 : 1);
