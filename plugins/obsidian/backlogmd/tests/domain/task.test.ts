import { describe, it, expect } from "vitest";
import { isDone, isBlocked, Task } from "../../src/domain/task";

describe("task domain", () => {
  const done = ["Done"];

  it("isDone true for done status", () => {
    const t: Task = { id: "T1", title: "x", status: "Done", path: "p" };
    expect(isDone(t, done)).toBe(true);
  });

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