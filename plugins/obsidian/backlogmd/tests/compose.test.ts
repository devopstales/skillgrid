import { describe, it, expect, vi } from "vitest";

vi.mock("obsidian", () => ({}));

import { coerceManualRoots } from "../src/compose";

describe("coerceManualRoots (M5)", () => {
  it("keeps string entries of an array, drops the rest", () => {
    expect(coerceManualRoots(["Extra", 42, null, "Other", undefined])).toEqual(["Extra", "Other"]);
    expect(coerceManualRoots([])).toEqual([]);
  });

  it("coerces a bare string to a single-element array", () => {
    expect(coerceManualRoots("Extra")).toEqual(["Extra"]);
    expect(coerceManualRoots("  Extra  ")).toEqual(["Extra"]);
    expect(coerceManualRoots("")).toEqual([]);
  });

  it("returns [] for anything else (number, null, undefined, object)", () => {
    expect(coerceManualRoots(42)).toEqual([]);
    expect(coerceManualRoots(null)).toEqual([]);
    expect(coerceManualRoots(undefined)).toEqual([]);
    expect(coerceManualRoots({ roots: ["Extra"] })).toEqual([]);
  });
});
