import { describe, it, expect } from "vitest";
import { parseTask, serializeTask, updateFrontmatter } from "../../src/infrastructure/vault";

const SAMPLE = `---
id: TASK-1
title: "Hello"
status: "To Do"
priority: medium
dependencies: []
---
Body line 1
Body line 2
`;

describe("vault adapter", () => {
  it("parse reads task", () => {
    const p = parseTask("x.md", SAMPLE);
    expect(p.task.id).toBe("TASK-1");
    expect(p.task.status).toBe("To Do");
  });

  it("round trip body stable", () => {
    const p = parseTask("x.md", SAMPLE);
    const out = serializeTask(p.task, p.body);
    // body present
    expect(out).toContain("Body line 1");
  });

  it("updateFrontmatter changes only patched field", () => {
    const out = updateFrontmatter(SAMPLE, { status: "In Progress" });
    const p2 = parseTask("x.md", out);
    expect(p2.task.status).toBe("In Progress");
    expect(p2.body.trim()).toBe("Body line 1\nBody line 2");
  });
});