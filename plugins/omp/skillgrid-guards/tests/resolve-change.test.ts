import { test, expect } from "bun:test";
import { mkdir, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { resolveChange, findProjectRoot } from "../src/resolve-change.ts";
import { SLUG, makeProject } from "./fixtures.ts";

test("no .skillgrid above cwd → null (guards off)", async () => {
  const fx = await makeProject({ changeDir: false });
  try {
    expect(await findProjectRoot("/nonexistent/deep/path")).toBeNull();
  } finally {
    await fx.cleanup();
  }
});

test("single in-progress change resolves; archive excluded", async () => {
  const fx = await makeProject();
  try {
    const ref = await resolveChange(fx.root);
    expect(ref?.slug).toBe(SLUG);
    expect(ref?.changeDir).toBe(fx.change);
    expect(ref?.specsRoot).toBe(fx.specs);
    expect(ref?.scratchDir).toBe(join(fx.root, ".skillgrid", "work"));
  } finally {
    await fx.cleanup();
  }
});

test("archived-only project has no active change", async () => {
  const fx = await makeProject({
    state: `changes:\n  ${SLUG}:\n    phase: "archived"\n    status: complete\n    gate: ""\n`,
    changeDir: false,
  });
  try {
    const ref = await resolveChange(fx.root);
    expect(ref?.slug).toBeNull();
    expect(ref?.ambiguous).toEqual([]);
  } finally {
    await fx.cleanup();
  }
});

test("multiple active changes → ambiguous, slug null (fail-open)", async () => {
  const fx = await makeProject({
    state: `changes:\n  a-1:\n    phase: "execution"\n    status: in_progress\n    gate: ""\n  b-2:\n    phase: "qa"\n    status: in_progress\n    gate: ""\n`,
    changeDir: false,
  });
  try {
    const ref = await resolveChange(fx.root);
    expect(ref?.slug).toBeNull();
    expect(ref?.ambiguous.sort()).toEqual(["a-1", "b-2"]);
  } finally {
    await fx.cleanup();
  }
});

test("config overrides specs_root and scratch_dir", async () => {
  const fx = await makeProject({
    config: `conventions:\n  specs_root: .spec\n  scratch_dir: .scratch\n`,
    changeDir: false,
  });
  try {
    const ref = await resolveChange(fx.root);
    expect(ref?.specsRoot).toBe(join(fx.root, ".spec"));
    expect(ref?.scratchDir).toBe(join(fx.root, ".scratch"));
  } finally {
    await fx.cleanup();
  }
});

test("unparseable state.yaml falls back to a single spec-zone dir", async () => {
  const fx = await makeProject({ state: "not: [valid yaml" });
  try {
    const ref = await resolveChange(fx.root);
    expect(ref?.slug).toBe(SLUG);
  } finally {
    await fx.cleanup();
  }
});

test("unparseable state.yaml with two dirs → ambiguous", async () => {
  const fx = await makeProject({ state: "not: [valid yaml" });
  try {
    await mkdir(join(fx.specs, "2026-11-22-other"), { recursive: true });
    await writeFile(join(fx.specs, "2026-11-22-other/briefing.md"), "# x\n");
    const ref = await resolveChange(fx.root);
    expect(ref?.slug).toBeNull();
    expect(ref?.ambiguous).toHaveLength(2);
  } finally {
    await fx.cleanup();
  }
});

test("garbage config.yaml → defaults, no throw", async () => {
  const fx = await makeProject({ config: "conventions: [unclosed" });
  try {
    const ref = await resolveChange(fx.root);
    expect(ref?.specsRoot).toBe(fx.specs);
    expect(ref?.slug).toBe(SLUG);
  } finally {
    await fx.cleanup();
  }
});

test("resolves from a nested worktree cwd (climbs to root)", async () => {
  const fx = await makeProject();
  try {
    const nested = join(fx.root, ".worktree", "wt-x", "src");
    await mkdir(nested, { recursive: true });
    await writeFile(join(nested, "a.ts"), "export {}\n");
    const ref = await resolveChange(nested);
    expect(ref?.root).toBe(fx.root);
  } finally {
    await fx.cleanup();
  }
});
