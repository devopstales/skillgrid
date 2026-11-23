import { test, expect } from "bun:test";
import { checkBannerEdit, readBanner } from "../src/banner.ts";
import { bannerDoc } from "./fixtures.ts";

// --- readBanner -------------------------------------------------------------

test("parses banner with backticks and date", () => {
  expect(readBanner(bannerDoc("T", "draft"))).toBe("draft");
});

test("parses banner without backticks", () => {
  expect(readBanner("# T\n\n> **STATUS:** approved (2026-11-23)\n")).toBe("approved");
});

test("finds banner within first 10 lines, not buried in body", () => {
  const buried = `# T\n\nx\ny\nz\nw\nv\nu\nt\ns\nr\n> **STATUS:** \`draft\` (2026-11-23)\n`;
  expect(readBanner(buried)).toBeNull();
});

test("null when no banner", () => {
  expect(readBanner("# T\n\nbody\n")).toBeNull();
});

test("rejects uppercase and leading-digit values", () => {
  expect(readBanner("# T\n\n> **STATUS:** `Draft` (x)\n")).toBeNull();
  expect(readBanner("# T\n\n> **STATUS:** `1st` (x)\n")).toBeNull();
});

// --- ownership: fresh file --------------------------------------------------

test("fresh briefing.md with initial value draft passes", () => {
  const d = checkBannerEdit("briefing.md", null, bannerDoc("T", "draft"));
  expect(d.block).toBeUndefined();
});

test("fresh blueprint.md must start proposed, not draft", () => {
  const d = checkBannerEdit("blueprint.md", null, bannerDoc("T", "draft"));
  expect(d.block).toBe(true);
  expect(d.reason).toContain("proposed");
});

test("fresh tasks.md must start sliced", () => {
  expect(checkBannerEdit("tasks.md", null, bannerDoc("T", "draft")).block).toBe(true);
  expect(checkBannerEdit("tasks.md", null, bannerDoc("T", "sliced")).block).toBeUndefined();
});

test("fresh file without banner is blocked", () => {
  const d = checkBannerEdit("briefing.md", null, "# T\n\nbody\n");
  expect(d.block).toBe(true);
  expect(d.reason).toContain("STATUS banner");
});

test("agent cannot mint approved on a fresh file (user-only value)", () => {
  const d = checkBannerEdit("briefing.md", null, bannerDoc("T", "approved"));
  expect(d.block).toBe(true);
});

// --- ownership: edits -------------------------------------------------------

test("agent edit draft -> approved is blocked (user-only value)", () => {
  const d = checkBannerEdit("briefing.md", bannerDoc("T", "draft"), bannerDoc("T", "approved"));
  expect(d.block).toBe(true);
  expect(d.reason).toContain("user-only");
});

test("agent edit approved -> revised is allowed (the contract revert)", () => {
  const d = checkBannerEdit("briefing.md", bannerDoc("T", "approved"), bannerDoc("T", "revised"));
  expect(d.block).toBeUndefined();
});

test("agent edit of approved doc that keeps banner approved is blocked", () => {
  const before = bannerDoc("T", "approved") + "\nold line\n";
  const after = bannerDoc("T", "approved") + "\nnew line\n";
  const d = checkBannerEdit("briefing.md", before, after);
  expect(d.block).toBe(true);
  expect(d.reason).toContain("revised");
});

test("agent edit approved -> revised plus body change is allowed", () => {
  const before = bannerDoc("T", "approved") + "\nold line\n";
  const after = bannerDoc("T", "revised") + "\nnew line\n";
  expect(checkBannerEdit("briefing.md", before, after).block).toBeUndefined();
});

test("edit removing the banner is blocked", () => {
  const d = checkBannerEdit("briefing.md", bannerDoc("T", "draft"), "# T\n\nbody\n");
  expect(d.block).toBe(true);
});

test("edit to an invalid banner value is blocked", () => {
  const d = checkBannerEdit("tasks.md", bannerDoc("T", "sliced"), bannerDoc("T", "done"));
  expect(d.block).toBe(true);
  expect(d.reason).toContain("not a valid value");
});

test("no-op edit (identical content) passes even on approved docs", () => {
  const c = bannerDoc("T", "approved");
  expect(checkBannerEdit("briefing.md", c, c).block).toBeUndefined();
});

test("edits between agent-writable values pass (draft -> revised)", () => {
  const d = checkBannerEdit("briefing.md", bannerDoc("T", "draft"), bannerDoc("T", "revised"));
  expect(d.block).toBeUndefined();
});

test("non-banner files are not checked", () => {
  const d = checkBannerEdit("findings.md", null, "# findings\nno banner here\n");
  expect(d.block).toBeUndefined();
});

test("legacy file without banner: agent may add a valid non-approved banner", () => {
  const d = checkBannerEdit("blueprint.md", "# T\n\nbody\n", bannerDoc("T", "proposed"));
  expect(d.block).toBeUndefined();
});

test("legacy file without banner: agent may not add approved", () => {
  const d = checkBannerEdit("blueprint.md", "# T\n\nbody\n", bannerDoc("T", "approved"));
  expect(d.block).toBe(true);
});
