import { describe, it, expect } from "vitest";
import { parseMilestone, serializeMilestone, effectiveState, daysLeft, Milestone } from "../../src/domain/milestone";
import { nextMilestoneId } from "../../src/application/milestone";
import { Task } from "../../src/domain/task";

const MINIMAL = `---
id: m-0
title: "KubeDash 5.0 base structure"
---

## Description

Milestone: KubeDash 5.0 base structure
`;

const t = (id: string, milestone: string | undefined, status = "To Do"): Task => ({
  id,
  title: id,
  status,
  milestone,
  path: `tasks/${id}.md`,
});

describe("milestone domain", () => {
  it("parses the CLI's minimal format", () => {
    const p = parseMilestone("milestones/m-0 - x.md", MINIMAL);
    expect(p.milestone.id).toBe("m-0");
    expect(p.milestone.title).toBe("KubeDash 5.0 base structure");
    expect(p.milestone.state).toBeUndefined();
    expect(p.body).toContain("## Description");
  });

  it("round-trips sprint fields", () => {
    const m: Milestone = {
      id: "m-1",
      title: "x",
      state: "active",
      start_date: "2026-11-10",
      end_date: "2026-11-21",
      goal: "ship auth",
      completed_tasks: ["TASK-001"],
      carried_over_tasks: ["TASK-002"],
      path: "p",
    };
    const out = serializeMilestone(m, "body");
    const p = parseMilestone("p", out);
    expect(p.milestone.state).toBe("active");
    expect(p.milestone.end_date).toBe("2026-11-21");
    expect(p.milestone.goal).toBe("ship auth");
    expect(p.milestone.completed_tasks).toEqual(["TASK-001"]);
    expect(p.milestone.carried_over_tasks).toEqual(["TASK-002"]);
    expect(p.body.trim()).toBe("body");
  });

  it("effectiveState: explicit wins, otherwise derive from tasks", () => {
    const done = ["Done"];
    const m: Milestone = { id: "m-1", title: "x", path: "p" };
    expect(effectiveState(m, [t("TASK-001", "m-1", "Done")], done)).toBe("completed");
    expect(effectiveState(m, [t("TASK-001", "m-1", "Done"), t("TASK-002", "m-1", "To Do")], done)).toBe("planned");
    expect(effectiveState(m, [t("TASK-001", "m-2", "Done")], done)).toBe("planned");
    const explicit: Milestone = { ...m, state: "active" };
    expect(effectiveState(explicit, [t("TASK-001", "m-1", "Done")], done)).toBe("active");
  });

  it("daysLeft counts whole days", () => {
    const inThree: Milestone = { id: "m", title: "x", end_date: new Date(Date.now() + 3 * 86400000).toISOString().slice(0, 10), path: "" };
    expect(daysLeft(inThree)).toBe(3);
    const none: Milestone = { id: "m", title: "x", path: "" };
    expect(daysLeft(none)).toBeNull();
    const past: Milestone = { id: "m", title: "x", end_date: new Date(Date.now() - 2 * 86400000).toISOString().slice(0, 10), path: "" };
    expect(daysLeft(past)).toBe(-2);
  });

  it("nextMilestoneId takes max numeric suffix + 1", () => {
    const ms = (id: string): Milestone => ({ id, title: id, path: "" });
    expect(nextMilestoneId([ms("m-0"), ms("m-4"), ms("m-6")])).toBe("m-7");
    expect(nextMilestoneId([])).toBe("m-1");
  });
});
