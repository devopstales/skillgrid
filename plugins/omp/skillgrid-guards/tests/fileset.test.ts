import { test, expect } from "bun:test";
import { checkFileSet } from "../src/fileset.ts";

const S = "2026-11-23-topic";

// --- canonical names ---------------------------------------------------------

test("every canonical file is allowed", () => {
  for (const f of [
    "briefing.md",
    "blueprint.md",
    "acceptance.feature",
    "tasks.md",
    "adr.md",
    "findings.md",
    "ledger.md",
    "report.md",
    "review.md",
  ]) {
    expect(checkFileSet(`${S}/${f}`, true)).toEqual({});
  }
});

test("council notes are allowed beside artifacts", () => {
  expect(checkFileSet(`${S}/blueprint.council.md`, true)).toEqual({});
  expect(checkFileSet(`${S}/tasks.council.md`, true)).toEqual({});
});

test("canonical subdirectories are allowed at any depth", () => {
  expect(checkFileSet(`${S}/shared/context.md`, true)).toEqual({});
  expect(checkFileSet(`${S}/modules/auth/session.ts`, true)).toEqual({});
  expect(checkFileSet(`${S}/sketches/a.png`, true)).toEqual({});
  expect(checkFileSet(`${S}/evaluations/primary.md`, true)).toEqual({});
});

// --- violations ---------------------------------------------------------------

test("invented file name is blocked on create", () => {
  const d = checkFileSet(`${S}/plan.md`, true);
  expect(d.block).toBe(true);
  expect(d.reason).toContain("plan.md");
  expect(d.reason).toContain("load-bearing");
});

test("edit of a pre-existing stray file warns instead of blocking", () => {
  const d = checkFileSet(`${S}/notes.md`, false);
  expect(d.block).toBeUndefined();
  expect(d.context).toContain("not a canonical");
});

test("file directly under the spec zone is blocked on create", () => {
  const d = checkFileSet("briefing.md", true);
  expect(d.block).toBe(true);
});

test("non-change-dir slug under the spec zone is blocked on create", () => {
  const d = checkFileSet("shared-stuff/context.md", true);
  expect(d.block).toBe(true);
});

// --- change-archive immutability ---------------------------------------------

test("new file in change-archive is blocked", () => {
  const d = checkFileSet(`change-archive/${S}/notes.md`, true);
  expect(d.block).toBe(true);
  expect(d.reason).toContain("immutable");
});

test("report.md in change-archive is allowed (reflect appends in place)", () => {
  expect(checkFileSet(`change-archive/${S}/report.md`, true)).toEqual({});
});

test("edit of archived briefing is not blocked (it pre-exists the archive move)", () => {
  expect(checkFileSet(`change-archive/${S}/briefing.md`, false)).toEqual({});
});
