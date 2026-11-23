import { test, expect } from "bun:test";
import { mkdir, mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { checkWriteEdit } from "../src/guards.ts";
import skillgridGuards from "../hooks/skillgrid-guards.ts";
import { SLUG, bannerDoc, makeProject, type Fixture } from "./fixtures.ts";

const W = (fx: Fixture, rel: string, content: string) =>
  checkWriteEdit(fx.root, "write", { path: join(fx.root, rel), content });

test("golden path: the chain passes in order and blocks out of order", async () => {
  const fx = await makeProject();
  const spec = (name: string) => join(fx.change, name);
  try {
    // Agent writes land on disk after the guard passes (the tool executes).
    const w = async (name: string, content: string) => {
      const d = await W(fx, `.skillgrid/change-specs/${fx.slug}/${name}`, content);
      if (d?.block !== true) await writeFile(spec(name), content);
      return d;
    };

    // 1. briefing (chain head, initial banner) — passes
    expect(await w("briefing.md", bannerDoc("T", "draft"))).toBeUndefined();
    // 2. invented name — fileset blocks
    let d = await W(fx, `.skillgrid/change-specs/${fx.slug}/plan.md`, "# plan\n");
    expect(d?.block).toBe(true);
    expect(d?.reason).toContain("canonical");
    // 3. blueprint before briefing approval — order blocks
    d = await W(fx, `.skillgrid/change-specs/${fx.slug}/blueprint.md`, bannerDoc("T", "proposed"));
    expect(d?.block).toBe(true);
    expect(d?.reason).toContain("approved");
    // user flips the briefing (hand-edit = direct file write, not via the guard)
    await writeFile(spec("briefing.md"), bannerDoc("T", "approved"));
    // 4. blueprint now passes (with its initial banner)
    expect(await w("blueprint.md", bannerDoc("T", "proposed"))).toBeUndefined();
    // 5. acceptance before blueprint approval — order blocks
    d = await W(fx, `.skillgrid/change-specs/${fx.slug}/acceptance.feature`, bannerDoc("T", "draft"));
    expect(d?.block).toBe(true);
    expect(d?.reason).toContain("blueprint");
    // user flips the blueprint
    await writeFile(spec("blueprint.md"), bannerDoc("T", "approved"));
    // 6. agent tries to mint approved itself — banner guard blocks
    d = await W(fx, `.skillgrid/change-specs/${fx.slug}/acceptance.feature`, bannerDoc("T", "approved"));
    expect(d?.block).toBe(true);
    expect(d?.reason).toContain("user-only");
    // 7. acceptance with its initial banner passes
    expect(await w("acceptance.feature", bannerDoc("T", "draft"))).toBeUndefined();
    // 8. tasks before acceptance approval — order blocks
    d = await W(fx, `.skillgrid/change-specs/${fx.slug}/tasks.md`, bannerDoc("T", "sliced"));
    expect(d?.block).toBe(true);
    expect(d?.reason).toContain("acceptance.feature");
    // user flips acceptance; agent writes tasks
    await writeFile(spec("acceptance.feature"), bannerDoc("T", "approved"));
    expect(await w("tasks.md", bannerDoc("T", "sliced"))).toBeUndefined();
    // 9. code write with a cleared gate passes (fixture state has gate "")
    expect(await W(fx, "src/index.ts", "export {}\n")).toBeUndefined();
  } finally {
    await fx.cleanup();
  }
});

test("edit simulation: agent flips its own draft banner to approved → blocked", async () => {
  const fx = await makeProject();
  try {
    const p = join(fx.change, "briefing.md");
    await writeFile(p, bannerDoc("T", "draft"));
    const d = await checkWriteEdit(fx.root, "edit", {
      path: p,
      old_string: "> **STATUS:** `draft` (2026-11-23)",
      new_string: "> **STATUS:** `approved` (2026-11-23)",
    });
    expect(d?.block).toBe(true);
    expect(d?.reason).toContain("user-only");
  } finally {
    await fx.cleanup();
  }
});

test("edit simulation: replace_all and missing old_string behave", async () => {
  const fx = await makeProject();
  try {
    const p = join(fx.change, "briefing.md");
    await writeFile(p, bannerDoc("T", "draft") + "tail\n");
    // missing old_string → guard cannot simulate → fail-open (no decision)
    expect(
      await checkWriteEdit(fx.root, "edit", { path: p, old_string: "nope", new_string: "x" }),
    ).toBeUndefined();
    // banner-line edit via replace_all to a valid agent value passes
    const d = await checkWriteEdit(fx.root, "edit", {
      path: p,
      old_string: "> **STATUS:** `draft` (2026-11-23)",
      new_string: "> **STATUS:** `revised` (2026-11-23)",
      replace_all: true,
    });
    expect(d?.block).toBeUndefined();
  } finally {
    await fx.cleanup();
  }
});

test("fail-open: no project, garbage state, unknown tool, missing path", async () => {
  const tmp = await mkdtemp(join(tmpdir(), "sg-noproj-"));
  expect(await checkWriteEdit(tmp, "write", { path: join(tmp, "a.ts"), content: "x" })).toBeUndefined();
  expect(await checkWriteEdit(tmp, "bash", { path: join(tmp, "a.ts") })).toBeUndefined();
  expect(await checkWriteEdit(tmp, "write", { content: "x" })).toBeUndefined();

  const fx = await makeProject({ state: "not: [yaml" });
  try {
    // ambiguous (two dirs) or single-dir fallback both must not throw
    await mkdir(join(fx.specs, "2026-11-22-other"), { recursive: true });
    await writeFile(join(fx.specs, "2026-11-22-other/briefing.md"), "# x\n");
    expect(
      await checkWriteEdit(fx.root, "write", {
        path: join(fx.specs, SLUG, "findings.md"),
        content: "# f\n",
      }),
    ).toBeUndefined(); // findings.md is canonical; ambiguity fails open to no decision
  } finally {
    await fx.cleanup();
  }
});

test("fail-open: writing to a directory target does not throw", async () => {
  const fx = await makeProject();
  try {
    // directory target: not a banner file, parts.length === 1 in the spec zone
    // → every guard passes; the entrypoint must resolve, not throw
    const d = await checkWriteEdit(fx.root, "write", {
      path: fx.change,
      content: "x",
    });
    expect(d).toBeUndefined();
  } finally {
    await fx.cleanup();
  }
});

test("hook adapter: registers tool_call, blocks bad writes, ignores other tools", async () => {
  const fx = await makeProject();
  try {
    const handlers: Record<string, (e: unknown, c: unknown) => unknown> = {};
    skillgridGuards({ on: (ev, h) => (handlers[ev] = h as never) });
    expect(typeof handlers["tool_call"]).toBe("function");

    // non-covered tool → no decision
    const r1 = await handlers["tool_call"]!(
      { toolName: "bash", input: { command: "ls" } },
      { cwd: fx.root },
    );
    expect(r1).toBeUndefined();

    // covered tool, canonical new file → no decision (briefing is chain head)
    const r2 = await handlers["tool_call"]!(
      {
        toolName: "write",
        input: { path: join(fx.change, "briefing.md"), content: bannerDoc("T", "draft") },
      },
      { cwd: fx.root },
    );
    expect(r2).toBeUndefined();

    // covered tool, invented name → block with reason
    const r3 = await handlers["tool_call"]!(
      { toolName: "write", input: { path: join(fx.change, "plan.md"), content: "# p\n" } },
      { cwd: fx.root },
    );
    expect(r3).toEqual({ block: true, reason: expect.stringContaining("canonical") });

    // handler never throws, even with a garbage event
    expect(await handlers["tool_call"]!({}, { cwd: fx.root })).toBeUndefined();
  } finally {
    await fx.cleanup();
  }
});

test("context notes: editing a pre-existing stray file passes with a warning", async () => {
  const fx = await makeProject();
  try {
    const p = join(fx.change, "notes.md");
    await writeFile(p, "# stray\nline\n");
    const d = await checkWriteEdit(fx.root, "edit", {
      path: p,
      old_string: "line",
      new_string: "line2",
    });
    expect(d?.block).toBeUndefined();
    expect(d?.context).toContain("not a canonical");
  } finally {
    await fx.cleanup();
  }
});
