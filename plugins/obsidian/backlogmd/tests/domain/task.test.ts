import { describe, it, expect } from "vitest";
import { isBlocked, nextTaskId, Task } from "../../src/domain/task";

const t = (id: string): Task => ({ id, title: id, status: "To Do", path: "p" });

describe("task domain", () => {
  const done = ["Done"];

  it("isBlocked true when dependency not done", () => {
    const dep: Task = { id: "D", title: "d", status: "To Do", path: "p" };
    const t: Task = { id: "T1", title: "x", status: "To Do", dependencies: ["D"], path: "p" };
    const m = new Map([["D", dep]]);
    expect(isBlocked(t, m, done)).toBe(true);
  });

  it("isBlocked false when dependency done", () => {
    const dep: Task = { id: "D", title: "d", status: "Done", path: "p" };
    const t: Task = { id: "T1", title: "x", status: "To Do", dependencies: ["D"], path: "p" };
    const m = new Map([["D", dep]]);
    expect(isBlocked(t, m, done)).toBe(false);
  });
});

describe("nextTaskId (shared domain function)", () => {
  it("starts at 001 when no matching tasks exist", () => {
    expect(nextTaskId([], "task")).toBe("TASK-001");
  });

  it("takes max numeric suffix + 1 (not count — deletions leave gaps)", () => {
    expect(nextTaskId([t("TASK-001"), t("TASK-002"), t("TASK-007")], "task")).toBe("TASK-008");
  });

  it("zero-pads to three digits", () => {
    expect(nextTaskId([t("TASK-009")], "task")).toBe("TASK-010");
  });

  it("is prefix-anchored: unrelated prefixes never bump the counter", () => {
    expect(nextTaskId([t("EPIC-ALPHA"), t("TASK-003")], "task")).toBe("TASK-004");
    expect(nextTaskId([t("FEATURE-7"), t("BUG-12")], "task")).toBe("TASK-001");
    expect(nextTaskId([t("OBX9"), t("TASK9")], "OB")).toBe("OB-001"); // no dash/underscore/space separator
  });

  it("matches prefix case-insensitively and uppercases the prefix", () => {
    expect(nextTaskId([t("task-105")], "TASK")).toBe("TASK-106");
    expect(nextTaskId([t("TASK-105")], "task")).toBe("TASK-106");
    expect(nextTaskId([], "TASK")).toBe("TASK-001");
  });

  it("ignores subtask-style ids (trailing .N)", () => {
    expect(nextTaskId([t("task-116.16")], "task")).toBe("TASK-001");
  });
});
