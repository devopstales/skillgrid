import { test, expect } from "bun:test";
import { mkdir, writeFile } from "node:fs/promises";
import {
  checkArtifactOrder,
  parseWaiver,
  type OrderContext,
} from "../src/artifact-order.ts";
import { clearFileCache } from "../src/resolve-change.ts";
import {
  SLUG,
  bannerDoc,
  makeProject,
  waiverBlock,
  type Fixture,
} from "./fixtures.ts";

async function ocFor(fx: Fixture): Promise<OrderContext> {
  return {
    root: fx.root,
    specsRoot: fx.specs,
    scratchDir: `${fx.root}/.skillgrid/work`,
    changeDir: fx.change,
    slug: fx.slug,
    entry: { slug: fx.slug, phase: "execution", status: "in_progress", gate: "" },
  };
}

const at = (fx: Fixture, name: string) => `${fx.change}/${name}`;

test("briefing.md is the chain head — always writable", async () => {
  const fx = await makeProject();
  try {
    expect(await checkArtifactOrder(at(fx, "briefing.md"), await ocFor(fx))).toEqual({});
  } finally {
    await fx.cleanup();
  }
});

test("blueprint.md blocked until briefing approved", async () => {
  const fx = await makeProject();
  try {
    await writeFile(at(fx, "briefing.md"), bannerDoc("T", "draft"));
    const d = await checkArtifactOrder(at(fx, "blueprint.md"), await ocFor(fx));
    expect(d?.block).toBe(true);
    expect(d?.reason).toContain("approved");

    await writeFile(at(fx, "briefing.md"), bannerDoc("T", "approved"));
    clearFileCache();
    expect(await checkArtifactOrder(at(fx, "blueprint.md"), await ocFor(fx))).toEqual({});
  } finally {
    await fx.cleanup();
  }
});

test("acceptance.feature blocked until blueprint approved; assumed/accepted count", async () => {
  const fx = await makeProject();
  try {
    await writeFile(at(fx, "briefing.md"), bannerDoc("T", "approved"));
    const d = await checkArtifactOrder(at(fx, "acceptance.feature"), await ocFor(fx));
    expect(d?.block).toBe(true);

    await writeFile(at(fx, "blueprint.md"), bannerDoc("T", "proposed"));
    clearFileCache();
    expect((await checkArtifactOrder(at(fx, "acceptance.feature"), await ocFor(fx)))?.block).toBe(true);

    for (const v of ["approved", "assumed", "accepted"]) {
      await writeFile(at(fx, "blueprint.md"), bannerDoc("T", v));
      clearFileCache();
      expect(
        (await checkArtifactOrder(at(fx, "acceptance.feature"), await ocFor(fx)))?.block,
      ).toBeUndefined();
    }
  } finally {
    await fx.cleanup();
  }
});

test("tasks.md blocked until acceptance approved", async () => {
  const fx = await makeProject();
  try {
    await writeFile(at(fx, "briefing.md"), bannerDoc("T", "approved"));
    await writeFile(at(fx, "blueprint.md"), bannerDoc("T", "approved"));
    expect((await checkArtifactOrder(at(fx, "tasks.md"), await ocFor(fx)))?.block).toBe(true);

    await writeFile(at(fx, "acceptance.feature"), bannerDoc("T", "approved"));
    clearFileCache();
    expect(await checkArtifactOrder(at(fx, "tasks.md"), await ocFor(fx))).toEqual({});
  } finally {
    await fx.cleanup();
  }
});

test("report.md blocked until an execution signal exists", async () => {
  const fx = await makeProject();
  try {
    await writeFile(at(fx, "tasks.md"), bannerDoc("T", "sliced") + "- [ ] T1 do thing\n");
    expect((await checkArtifactOrder(at(fx, "report.md"), await ocFor(fx)))?.block).toBe(true);

    await writeFile(at(fx, "tasks.md"), bannerDoc("T", "complete") + "- [x] T1 do thing\n");
    clearFileCache();
    expect(await checkArtifactOrder(at(fx, "report.md"), await ocFor(fx))).toEqual({});

    // the scratch ledger alone is also an execution signal
    await writeFile(at(fx, "tasks.md"), bannerDoc("T", "sliced") + "- [ ] T2 next\n");
    await mkdir(`${fx.root}/.skillgrid/work/${fx.slug}`, { recursive: true });
    await writeFile(`${fx.root}/.skillgrid/work/${fx.slug}/progress.md`, "ledger\n");
    clearFileCache();
    expect(await checkArtifactOrder(at(fx, "report.md"), await ocFor(fx))).toEqual({});
  } finally {
    await fx.cleanup();
  }
});

test("review.md blocked until report.md exists", async () => {
  const fx = await makeProject();
  try {
    expect((await checkArtifactOrder(at(fx, "review.md"), await ocFor(fx)))?.block).toBe(true);
    await writeFile(at(fx, "report.md"), "report\n");
    clearFileCache();
    expect(await checkArtifactOrder(at(fx, "review.md"), await ocFor(fx))).toEqual({});
  } finally {
    await fx.cleanup();
  }
});

test("code write blocked while user-gate-slice is pending", async () => {
  const fx = await makeProject();
  try {
    await writeFile(at(fx, "tasks.md"), bannerDoc("T", "sliced"));
    const oc = await ocFor(fx);
    oc.entry = { slug: fx.slug, phase: "execution", status: "in_progress", gate: "user-gate-slice" };
    const d = await checkArtifactOrder(`${fx.root}/src/index.ts`, oc);
    expect(d?.block).toBe(true);
    expect(d?.reason).toContain("user-gate-slice");

    oc.entry.gate = "";
    expect(await checkArtifactOrder(`${fx.root}/src/index.ts`, oc)).toEqual({});
  } finally {
    await fx.cleanup();
  }
});

test("code write at a post-slicing phase with no tasks.md is blocked", async () => {
  const fx = await makeProject({ changeDir: false });
  try {
    const oc = await ocFor(fx);
    oc.changeDir = null;
    oc.changeDir = `${fx.specs}/${fx.slug}`; // dir may not exist; tasks.md absent
    const d = await checkArtifactOrder(`${fx.root}/src/index.ts`, oc);
    expect(d?.block).toBe(true);
    expect(d?.reason).toContain("tasks.md");
  } finally {
    await fx.cleanup();
  }
});

test("planning-phase change is not attributed code writes (fail-open)", async () => {
  const fx = await makeProject({
    state: `changes:\n  ${SLUG}:\n    phase: "blueprint"\n    status: in_progress\n    gate: ""\n`,
    changeDir: false,
  });
  try {
    const oc = await ocFor(fx);
    oc.entry = { slug: fx.slug, phase: "blueprint", status: "in_progress", gate: "" };
    const d = await checkArtifactOrder(`${fx.root}/src/index.ts`, oc);
    expect(d?.block).toBeUndefined();
  } finally {
    await fx.cleanup();
  }
});

test("no active change (bounded work) leaves code writes unblocked", async () => {
  const fx = await makeProject({
    state: `changes:\n  ${SLUG}:\n    phase: "archived"\n    status: complete\n    gate: ""\n`,
    changeDir: false,
  });
  try {
    const oc = await ocFor(fx);
    oc.slug = null;
    oc.changeDir = null;
    oc.entry = null;
    const d = await checkArtifactOrder(`${fx.root}/src/index.ts`, oc);
    expect(d?.block).toBeUndefined(); // no active change → not attributed, not blocked
  } finally {
    await fx.cleanup();
  }
});

// --- waivers -------------------------------------------------------------------

test("trivial waiver lets tasks.md skip the acceptance predecessor", async () => {
  const fx = await makeProject();
  try {
    const briefing = bannerDoc("T", "draft") + waiverBlock("trivial", "brainstorming, blueprint, acceptance.feature");
    await writeFile(at(fx, "briefing.md"), briefing);
    clearFileCache();
    expect(await checkArtifactOrder(at(fx, "tasks.md"), await ocFor(fx))).toEqual({});
  } finally {
    await fx.cleanup();
  }
});

test("small waiver skips blueprint but not acceptance.feature", async () => {
  const fx = await makeProject();
  try {
    const briefing = bannerDoc("T", "draft") + waiverBlock("small", "brainstorming, blueprint");
    await writeFile(at(fx, "briefing.md"), briefing);
    clearFileCache();
    expect(
      (await checkArtifactOrder(at(fx, "acceptance.feature"), await ocFor(fx)))?.block,
    ).toBeUndefined();
    expect((await checkArtifactOrder(at(fx, "tasks.md"), await ocFor(fx)))?.block).toBe(true);
  } finally {
    await fx.cleanup();
  }
});

test("parseWaiver: valid block parses; incomplete blocks are null", () => {
  const good = waiverBlock("small", "brainstorming, blueprint");
  const w = parseWaiver(good);
  expect(w).not.toBeNull();
  expect(w?.cls).toBe("small");
  expect(w?.skips).toEqual(["brainstorming", "blueprint"]);
  expect(w?.approvedBy).toBe("programmer");

  expect(parseWaiver(null)).toBeNull();
  expect(parseWaiver("no waiver here")).toBeNull();
  expect(parseWaiver("## Fast-Track Waiver\n- Class: small\n")).toBeNull(); // missing fields
  expect(
    parseWaiver("## Fast-Track Waiver\n- Class: medium\n- Skips: x\n- Approved by: y\n"),
  ).toBeNull(); // invalid class
});
