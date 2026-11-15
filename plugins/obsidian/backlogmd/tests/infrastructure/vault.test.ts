import { describe, it, expect } from "vitest";
import { parseTask, serializeTask } from "../../src/infrastructure/vault";
import { Task } from "../../src/domain/task";

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

// Non-canonical field order (title before id) plus unknown fields the
// CLI/agents manage outside the plugin's known set (points, rank, custom).
// (Custom value is plain-YAML-safe so the round trip stays byte-identical:
// safeDump would quote a YAML 1.1 boolean-looking string like "yes".)
const RICH = `---
title: Fix login redirect
id: TASK-12
status: In Progress
points: 3
rank: 12
parent_task_id: TASK-11
custom_flag: enabled
dependencies: []
---
Body rich
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

  it("round-trips a rich file byte-identically: unknown fields kept, original key order, body untouched", () => {
    const p = parseTask("tasks/task-012 - fix-login-redirect.md", RICH);
    expect(p.task.raw).toBeDefined();
    const out = serializeTask(p.task, p.body);
    expect(out).toBe(RICH);
  });

  it("keeps unknown fields at their original positions and updates known fields in place", () => {
    const p = parseTask("x.md", RICH);
    const out = serializeTask({ ...p.task, status: "Done" }, p.body);
    const lines = out.split("\n");
    // original order preserved: title before id, status in its original slot
    expect(lines[1]).toBe("title: Fix login redirect");
    expect(lines[2]).toBe("id: TASK-12");
    expect(lines[3]).toBe("status: Done"); // updated in place, not moved
    expect(out).toContain("points: 3");
    expect(out).toContain("rank: 12");
    expect(out).toContain("custom_flag: enabled");
    expect(out).toContain("parent_task_id: TASK-11");
  });

  it("patching a field to undefined deletes the key from the output", () => {
    const p = parseTask("x.md", RICH);
    const out = serializeTask({ ...p.task, milestone: undefined }, p.body);
    expect(out).not.toContain("milestone");
    // a field present in the original file is removed when patched undefined
    const p2 = parseTask("x.md", `---\nid: TASK-1\ntitle: "x"\nstatus: "To Do"\nmilestone: m-1\npoints: 5\n---\nB\n`);
    const out2 = serializeTask({ ...p2.task, milestone: undefined }, p2.body);
    expect(out2).not.toContain("milestone");
    expect(out2).toContain("points: 5");
  });

  it("a task without raw serializes known fields only, in canonical order", () => {
    const t: Task = {
      id: "TASK-9",
      title: "bare",
      status: "To Do",
      priority: "high",
      path: "",
    };
    const out = serializeTask(t, "body");
    expect(out).toBe(`---\nid: TASK-9\ntitle: bare\nstatus: To Do\npriority: high\n---\nbody\n`);
  });

  it("parse preserves ordinal and serialize round-trips it", () => {
    const withOrdinal = `---
id: TASK-1
title: "Hello"
status: "To Do"
ordinal: 1000
---
Body
`;
    const p = parseTask("x.md", withOrdinal);
    expect(p.task.ordinal).toBe(1000);
    const out = serializeTask(p.task, p.body);
    expect(parseTask("x.md", out).task.ordinal).toBe(1000);
    expect(out).toContain("ordinal: 1000");
  });

  it("parse leaves ordinal undefined when absent", () => {
    const p = parseTask("x.md", SAMPLE);
    expect(p.task.ordinal).toBeUndefined();
  });

  it("parse throws on malformed YAML frontmatter (error path, not silent corruption)", () => {
    const malformed = `---
id: OB-1
status: [unclosed
---
Body
`;
    expect(() => parseTask("tasks/OB-1.md", malformed)).toThrow();
  });
});
