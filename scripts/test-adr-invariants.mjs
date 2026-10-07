#!/usr/bin/env node
/**
 * test-adr-invariants.mjs — fixture-driven tests for check-adr-invariants.mjs.
 *
 * Usage: node scripts/test-adr-invariants.mjs
 * Exit: 0 all pass, 1 any fail.
 */

import { spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, writeFileSync, rmSync, copyFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const CHECK = join(import.meta.dirname, "check-adr-invariants.mjs");

let passed = 0;
let failed = 0;
const failures = [];

function assert(name, cond, detail = "") {
  if (cond) {
    passed++;
    console.log(`  ok   ${name}`);
  } else {
    failed++;
    failures.push(name);
    console.log(`  FAIL ${name}${detail ? ` — ${detail}` : ""}`);
  }
}

const ADR_BODY = (n) => [
  `# ADR ${n} test decision`,
  "",
  "---",
  'status: "accepted"',
  "supersedes: none",
  "date: 2026-10-07",
  "---",
  "",
  "## Context",
  "",
  "A context paragraph.",
  "",
  "## Decision",
  "",
  "Pick option A.",
  "",
  "## Consequences",
  "",
  "- It works.",
].join("\n");

const MANIFEST = `# ADR Review Manifest

- **Change:** 2026-10-07-test
- **Status:** completed
- **Review date:** 2026-10-07

## In-Force ADRs Reviewed

- None in force.

## New Durable ADRs Created

- None.

## Supersessions

- None.
`;

function makeProject({ artifacts = {}, assumptions = "", specs = {} }) {
  const root = mkdtempSync(join(tmpdir(), "sgadr-"));
  mkdirSync(join(root, ".skillgrid", "artifacts"), { recursive: true });
  for (const [f, content] of Object.entries(artifacts)) {
    writeFileSync(join(root, ".skillgrid", "artifacts", f), content);
  }
  writeFileSync(join(root, ".skillgrid", "ASSUMPTIONS.md"), assumptions);
  for (const [dir, content] of Object.entries(specs)) {
    mkdirSync(join(root, ".skillgrid", "specs", dir), { recursive: true });
    writeFileSync(join(root, ".skillgrid", "specs", dir, "adr.md"), content);
  }
  return { root, cleanup: () => rmSync(root, { recursive: true, force: true }) };
}

const ROW = (n, record) => `| ${n} | test title | accepted | — | 2026-10-07 | yes | \`${record}\` |`;
const ASSUMPTIONS_HEADER = "# ASSUMPTIONS\n\n### ADR index\n\n| # | Description | In force | File |\n|---|-------------|----------|------|\n";

function run() {
  console.log("check-adr-invariants.mjs tests\n");

  // The check resolves ROOT from its own file location, so each test copies the
  // script into a temp repo layout and runs it there.
  function withTempRepo(setup) {
    const root = mkdtempSync(join(tmpdir(), "sgadr-repo-"));
    mkdirSync(join(root, "scripts"), { recursive: true });
    mkdirSync(join(root, ".skillgrid", "artifacts"), { recursive: true });
    mkdirSync(join(root, ".skillgrid", "specs"), { recursive: true });
    setup(root);
    copyFileSync(CHECK, join(root, "scripts", "check-adr-invariants.mjs"));
    const r = spawnSync("node", [join(root, "scripts", "check-adr-invariants.mjs")], {
      cwd: root,
      encoding: "utf8",
    });
    const out = (r.stdout ?? "") + (r.stderr ?? "");
    rmSync(root, { recursive: true, force: true });
    return { code: r.status ?? -1, out };
  }

  // 2. Clean: file + row + manifest → exit 0.
  {
    const r = withTempRepo((root) => {
      writeFileSync(join(root, ".skillgrid", "artifacts", "04-adr-0001-test-decision.md"), ADR_BODY(1));
      writeFileSync(join(root, ".skillgrid", "ASSUMPTIONS.md"),
        ASSUMPTIONS_HEADER + ROW("0001", ".skillgrid/artifacts/04-adr-0001-test-decision.md") + "\n");
      mkdirSync(join(root, ".skillgrid", "specs", "2026-10-07-test"), { recursive: true });
      writeFileSync(join(root, ".skillgrid", "specs", "2026-10-07-test", "adr.md"), MANIFEST);
    }, "clean");
    assert("clean: exit 0", r.code === 0, r.out);
    assert("clean: PASS message", r.out.includes("ADR invariants: PASS"), r.out);
  }

  // 3. Orphan file (no table row) → exit 1.
  {
    const r = withTempRepo((root) => {
      writeFileSync(join(root, ".skillgrid", "artifacts", "04-adr-0001-test-decision.md"), ADR_BODY(1));
      writeFileSync(join(root, ".skillgrid", "ASSUMPTIONS.md"), ASSUMPTIONS_HEADER + "\n");
    }, "orphan");
    assert("orphan: exit 1", r.code === 1, r.out);
    assert("orphan: names the file", r.out.includes("04-adr-0001-test-decision.md") && r.out.includes("orphan"), r.out);
  }

  // 4. Dangling row (file missing) → exit 1.
  {
    const r = withTempRepo((root) => {
      writeFileSync(join(root, ".skillgrid", "ASSUMPTIONS.md"),
        ASSUMPTIONS_HEADER + ROW("0001", ".skillgrid/artifacts/04-adr-0001-test-decision.md") + "\n");
    }, "dangling");
    assert("dangling: exit 1", r.code === 1, r.out);
    assert("dangling: names the row", r.out.includes("missing file"), r.out);
  }

  // 5. Duplicate number → exit 1.
  {
    const r = withTempRepo((root) => {
      writeFileSync(join(root, ".skillgrid", "artifacts", "04-adr-0001-test-decision.md"), ADR_BODY(1));
      writeFileSync(join(root, ".skillgrid", "artifacts", "04-adr-0001-other-decision.md"), ADR_BODY(1));
      writeFileSync(join(root, ".skillgrid", "ASSUMPTIONS.md"),
        ASSUMPTIONS_HEADER +
        ROW("0001", ".skillgrid/artifacts/04-adr-0001-test-decision.md") + "\n" +
        ROW("0001", ".skillgrid/artifacts/04-adr-0001-other-decision.md") + "\n");
    }, "dup");
    assert("dup: exit 1", r.code === 1, r.out);
    assert("dup: names the number", r.out.includes("duplicated number 0001"), r.out);
  }

  // 6. Missing frontmatter → exit 1.
  {
    const noFm = "# ADR\n\n## Context\n\n## Decision\n\n## Consequences\n";
    const r = withTempRepo((root) => {
      writeFileSync(join(root, ".skillgrid", "artifacts", "04-adr-0001-test-decision.md"), noFm);
      writeFileSync(join(root, ".skillgrid", "ASSUMPTIONS.md"),
        ASSUMPTIONS_HEADER + ROW("0001", ".skillgrid/artifacts/04-adr-0001-test-decision.md") + "\n");
    }, "frontmatter");
    assert("frontmatter: exit 1", r.code === 1, r.out);
    assert("frontmatter: names the field", r.out.includes("missing frontmatter field 'status'"), r.out);
  }

  // 7. Spec adr.md with a decision body (the drift we're guarding) → exit 1.
  {
    const r = withTempRepo((root) => {
      mkdirSync(join(root, ".skillgrid", "specs", "2026-10-05-drift"), { recursive: true });
      writeFileSync(join(root, ".skillgrid", "specs", "2026-10-05-drift", "adr.md"), ADR_BODY(1));
      writeFileSync(join(root, ".skillgrid", "ASSUMPTIONS.md"), ASSUMPTIONS_HEADER + "\n");
    }, "body-drift");
    assert("body-drift: exit 1", r.code === 1, r.out);
    assert("body-drift: names the spec", r.out.includes("specs/2026-10-05-drift/adr.md"), r.out);
    assert("body-drift: tells the fix", r.out.includes("move the body to .skillgrid/artifacts"), r.out);
  }

  // 8. Manifest with a dangling pointer → exit 1.
  {
    const manifest = MANIFEST.replace(
      "## New Durable ADRs Created\n\n- None.",
      "## New Durable ADRs Created\n\n- `.skillgrid/artifacts/04-adr-0042-gone.md` — gone."
    );
    const r = withTempRepo((root) => {
      mkdirSync(join(root, ".skillgrid", "specs", "2026-10-07-test"), { recursive: true });
      writeFileSync(join(root, ".skillgrid", "specs", "2026-10-07-test", "adr.md"), manifest);
      writeFileSync(join(root, ".skillgrid", "ASSUMPTIONS.md"), ASSUMPTIONS_HEADER + "\n");
    }, "dangling-pointer");
    assert("dangling-pointer: exit 1", r.code === 1, r.out);
    assert("dangling-pointer: names the pointer", r.out.includes("04-adr-0042-gone.md"), r.out);
  }

  console.log(`\n${passed} passed, ${failed} failed`);
  process.exit(failed === 0 ? 0 : 1);
}

run();
