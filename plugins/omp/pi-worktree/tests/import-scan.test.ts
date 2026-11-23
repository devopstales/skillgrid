import { test, expect } from "bun:test";
import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

const CORE = join(import.meta.dir, "..", "src", "core");

test("no core file imports a Pi package", () => {
  for (const file of readdirSync(CORE)) {
    if (!file.endsWith(".ts")) continue;
    const src = readFileSync(join(CORE, file), "utf8");
    expect(src).not.toMatch(/from\s+["']@earendil-works\//);
  }
});

test("constants match the contract values", async () => {
  const c = await import("../src/core/constants.ts");
  expect(c.REF_BASE).toBe("refs/skillgrid");
  expect(c.AGENT_BRANCH_RE.source).toBe("^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$");
  expect([...c.IGNORED_DIRS]).toEqual([
    "node_modules", ".venv", "venv", "env", ".env", "dist", "build",
    ".pytest_cache", ".mypy_cache", ".cache", ".tox", "__pycache__", ".worktree",
  ]);
  expect(c.MAX_UNTRACKED_FILE_SIZE).toBe(10 * 1024 * 1024);
  expect(c.MAX_UNTRACKED_DIR_FILES).toBe(200);
  expect(c.MAX_PER_SESSION).toBe(50);
  expect(c.isSafeRefId("abc-123_X")).toBe(true);
  expect(c.isSafeRefId("../evil")).toBe(false);
});
