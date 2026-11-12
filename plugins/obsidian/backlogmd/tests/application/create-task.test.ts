import { describe, it, expect } from "vitest";
import { nextTaskId } from "../../src/application/create-task";
import { Task } from "../../src/domain/task";

const t = (id: string): Task => ({ id, title: id, status: "To Do", path: "" });

describe("nextTaskId", () => {
  it("starts at 001 when no tasks exist", () => {
    expect(nextTaskId([], "task")).toBe("TASK-001");
  });

  it("takes max numeric suffix + 1 (not count — deletions leave gaps)", () => {
    expect(nextTaskId([t("TASK-001"), t("TASK-002"), t("TASK-007")], "task")).toBe("TASK-008");
  });

  it("zero-pads to three digits", () => {
    expect(nextTaskId([t("TASK-009")], "task")).toBe("TASK-010");
  });

  it("ignores IDs without a numeric suffix", () => {
    expect(nextTaskId([t("EPIC-ALPHA"), t("TASK-003")], "task")).toBe("TASK-004");
  });

  it("uppercases the prefix", () => {
    expect(nextTaskId([], "TASK")).toBe("TASK-001");
  });
});
