#!/usr/bin/env node
// check-adr-invariants.mjs — machine-checkable invariants for the ADR system.
//
// "A rule asks; a hook guarantees." This is the guarantee for the ADR home
// rule: decision bodies live in .skillgrid/artifacts/04-adr-NNNN-slug.md,
// .skillgrid/ASSUMPTIONS.md holds the path (one in-force row per file), and a
// spec's adr.md is a per-change manifest of POINTERS — never a copy of a body.
//
// Violations (each a FAIL):
//   1. An artifacts ADR file has no row in the in-force table (orphan file).
//   2. An in-force table row points at a missing artifacts file.
//   3. A duplicated in-force number (two files, one number).
//   4. An artifacts ADR file lacks the status/supersedes/date frontmatter.
//   5. A spec's adr.md is not a manifest — it carries a decision body
//      (## Context / ## Decision / ## Consequences headings without the
//      "ADR Review Manifest" header). This is the drift the 2026-10-05
//      scan-findings spec introduced before the convention hardened.
//   6. A manifest "New Durable ADRs Created" pointer that does not exist.
//
// Usage:
//   node scripts/check-adr-invariants.mjs          # check everything
//   node scripts/check-adr-invariants.mjs --json   # machine-readable status
//
// Exit code: 0 if every invariant holds, 1 if any violation.
//
// # FUTURE CLI ---------------------------------------------------------------
// This script is the reference implementation. When skillgrid gains a CLI, this
// becomes a verb:
//
//   skillgrid check adr-invariants
//
// -----------------------------------------------------------------------------

import { readFileSync, existsSync, readdirSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const ARTIFACTS = path.join(ROOT, ".skillgrid", "artifacts");
const SPECS = path.join(ROOT, ".skillgrid", "specs");
const ASSUMPTIONS = path.join(ROOT, ".skillgrid", "ASSUMPTIONS.md");
const jsonFlag = process.argv.includes("--json");

const ADR_FILE = /^04-adr-(\d{4})-([a-z0-9]+(?:-[a-z0-9]+)*)\.md$/;
const FRONTMATTER_FIELDS = ["status", "supersedes", "date"];
// Section headings that mark a decision body (vs a manifest of pointers).
const BODY_HEADINGS = ["## Context", "## Decision", "## Consequences"];
const MANIFEST_HEADER = "ADR Review Manifest";

let FAIL = 0;
const LINES = [];

function adrFiles() {
  if (!existsSync(ARTIFACTS)) return [];
  return readdirSync(ARTIFACTS)
    .filter((f) => ADR_FILE.test(f))
    .sort();
}

function parseInForceTable() {
  if (!existsSync(ASSUMPTIONS)) return [];
  const content = readFileSync(ASSUMPTIONS, "utf8");
  const lines = [];
  for (const raw of content.split("\n")) {
    // Table row: | 0001 | Title | accepted | — | 2026-09-16 | yes | `.skillgrid/artifacts/...` |
    const m = raw.match(/^\|\s*(\d{4})\s*\|.*?\|\s*`([^`]+)`\s*\|\s*$/);
    if (m) lines.push({ num: m[1], record: m[2] });
  }
  return lines;
}

function main() {
  const files = adrFiles();
  const table = parseInForceTable();

  // Invariants 1 + 2 + 3: file <-> table bijection, no duplicate numbers.
  const fileNums = new Map();
  for (const f of files) {
    const m = ADR_FILE.exec(f);
    fileNums.set(m[1], (fileNums.get(m[1]) || []).concat(f));
  }
  for (const [num, names] of fileNums) {
    if (names.length > 1) {
      LINES.push(`DRIFT  duplicated number ${num}: ${names.join(", ")}`);
      FAIL = 1;
    }
  }
  const tableRecords = new Set(table.map((r) => r.record));
  for (const f of files) {
    const record = `.skillgrid/artifacts/${f}`;
    if (!tableRecords.has(record)) {
      LINES.push(`DRIFT  ${f}: no in-force table row (orphan ADR file)`);
      FAIL = 1;
    }
  }
  for (const r of table) {
    const p = path.join(ROOT, r.record);
    if (!existsSync(p)) {
      LINES.push(`DRIFT  in-force row ${r.num}: missing file ${r.record}`);
      FAIL = 1;
    }
  }

  // Invariant 4: every ADR file carries the frontmatter fields.
  for (const f of files) {
    const content = readFileSync(path.join(ARTIFACTS, f), "utf8");
    for (const field of FRONTMATTER_FIELDS) {
      if (!new RegExp(`^${field}:`, "m").test(content)) {
        LINES.push(`DRIFT  ${f}: missing frontmatter field '${field}'`);
        FAIL = 1;
      }
    }
  }

  // Invariant 5 + 6: spec adr.md files are manifests with valid pointers.
  if (existsSync(SPECS)) {
    for (const dir of readdirSync(SPECS, { withFileTypes: true }).filter((e) => e.isDirectory())) {
      const manifestPath = path.join(SPECS, dir.name, "adr.md");
      if (!existsSync(manifestPath)) continue;
      const content = readFileSync(manifestPath, "utf8");
      const isManifest = content.includes(MANIFEST_HEADER);
      const hasBody = BODY_HEADINGS.some((h) => new RegExp(`^${h.replace("##", "\\#\\#")}\\b`, "m").test(content));
      if (!isManifest && hasBody) {
        LINES.push(`DRIFT  specs/${dir.name}/adr.md: decision body instead of manifest (move the body to .skillgrid/artifacts/04-adr-NNNN-slug.md)`);
        FAIL = 1;
      }
      // Validate "New Durable ADRs Created" pointers exist.
      const section = content.split("## New Durable ADRs Created");
      if (section.length > 1) {
        const body = section[1].split("\n## ")[0];
        for (const m of body.matchAll(/`(\.skillgrid\/artifacts\/04-adr-[a-z0-9.-]+\.md)`/g)) {
          if (!existsSync(path.join(ROOT, m[1]))) {
            LINES.push(`DRIFT  specs/${dir.name}/adr.md: pointer to missing file ${m[1]}`);
            FAIL = 1;
          }
        }
      }
    }
  }

  if (FAIL === 0) {
    LINES.push(`OK     ${files.length} ADR files, ${table.length} in-force rows, all invariants hold`);
  }

  if (jsonFlag) {
    for (const line of LINES) console.log(JSON.stringify({ status: line }));
  } else {
    for (const line of LINES) console.log(line);
    console.log("\n========================================");
    if (FAIL === 0) {
      console.log("ADR invariants: PASS");
    } else {
      console.log("ADR invariants: FAIL — a file drifted from the ADR home rule.");
      console.log("Decision bodies belong in .skillgrid/artifacts/04-adr-NNNN-slug.md;");
      console.log("ASSUMPTIONS.md holds the path; a spec adr.md is a manifest of pointers only.");
    }
  }
  process.exit(FAIL === 0 ? 0 : 1);
}

main();
