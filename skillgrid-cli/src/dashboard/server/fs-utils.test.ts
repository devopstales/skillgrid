import os from "node:os";
import path from "node:path";
import { describe, expect, it } from "vitest";
import { expandUserPath, resolveRepoRoot, sessionDirMatchesRepo } from "./fs-utils.js";

describe("expandUserPath", () => {
  it("expands ~/ to home directory", () => {
    const home = process.env.HOME ?? process.env.USERPROFILE;
    if (!home) return;
    expect(expandUserPath("~/projects/foo")).toBe(path.join(home, "projects/foo"));
  });
});

describe("sessionDirMatchesRepo", () => {
  it("matches exact repo root", () => {
    const repo = path.join(os.tmpdir(), "repo-a");
    expect(sessionDirMatchesRepo(repo, repo)).toBe(true);
  });

  it("matches subdirectory of repo", () => {
    const repo = path.join(os.tmpdir(), "repo-b");
    const sub = path.join(repo, "src", "lib");
    expect(sessionDirMatchesRepo(sub, repo)).toBe(true);
  });

  it("does not match sibling directory", () => {
    const repo = path.join(os.tmpdir(), "repo-c");
    const sibling = path.join(os.tmpdir(), "repo-c-other");
    expect(sessionDirMatchesRepo(sibling, repo)).toBe(false);
  });
});

describe("resolveRepoRoot", () => {
  it("resolves relative paths against cwd", () => {
    expect(resolveRepoRoot(".")).toBe(path.resolve("."));
  });
});
