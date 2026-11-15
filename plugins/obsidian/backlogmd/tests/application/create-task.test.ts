import { describe, it, expect } from "vitest";
import { createTask } from "../../src/application/create-task";
import { nextTaskId } from "../../src/domain/task";
import { TaskRepository, ProjectConfig } from "../../src/application/ports";
import { Task } from "../../src/domain/task";

const t = (id: string, ordinal?: number): Task => ({ id, title: id, status: "To Do", ordinal, path: "" });

class MemoryRepo implements TaskRepository {
  lastErrors: string[] = [];
  written: { task: Task; body: string }[] = [];
  async list() {
    return [];
  }
  async get() {
    return null;
  }
  async write(_project: ProjectConfig, task: Task, body: string) {
    this.written.push({ task, body });
  }
  async create(_project: ProjectConfig, task: Task, body: string) {
    this.written.push({ task, body });
    return `tasks/${task.id}.md`;
  }
}

const project: ProjectConfig = {
  name: "p",
  rootPath: "",
  backlogDirectory: ".backlog",
  vaultBacklogDir: ".backlog",
  statuses: ["To Do", "In Progress", "Done"],
  types: [],
  priorities: [],
  doneStatuses: ["Done"],
  taskPrefix: "OB",
};

describe("createTask", () => {
  it("writes a new task with the next prefixed id, default status, and body", async () => {
    const repo = new MemoryRepo();
    const existing = [t("OB-001", 1000), t("OB-003", 3000)];
    const ret = await createTask(repo, project, { summary: "New work", body: "nb" }, existing);
    expect(repo.written).toHaveLength(1);
    const w = repo.written[0];
    expect(w.task.id).toBe("OB-004"); // max numeric suffix + 1, gap-aware
    expect(w.task.title).toBe("New work");
    expect(w.task.status).toBe("To Do"); // project.statuses[0]
    expect(w.task.ordinal).toBe(4000); // max ordinal + 1000
    expect(w.task.type).toBeUndefined();
    expect(w.task.priority).toBeUndefined();
    expect(w.task.assignee).toBeUndefined();
    expect(w.body).toContain("nb");
    expect(ret).toBe("tasks/OB-004.md"); // create() return passes through
  });

  it("starts at ordinal 1000 and prefix-001 when no tasks exist", async () => {
    const repo = new MemoryRepo();
    await createTask(repo, project, { summary: "First" }, []);
    const w = repo.written[0];
    expect(w.task.id).toBe("OB-001");
    expect(w.task.ordinal).toBe(1000);
    expect(w.body).not.toBe(""); // ticket-gate scaffolding appended even for an empty body
  });

  it("stamps created_date (YYYY-MM-DD) on the new task", async () => {
    const repo = new MemoryRepo();
    await createTask(repo, project, { summary: "Dated" }, []);
    expect(repo.written[0].task.created_date).toBe(new Date().toISOString().slice(0, 10));
  });

  it("uses the shared prefix-anchored nextTaskId (FEATURE-7 does not bump the OB counter)", async () => {
    const repo = new MemoryRepo();
    const existing = [t("OB-002", 2000), t("FEATURE-7", 1000)];
    await createTask(repo, project, { summary: "X" }, existing);
    expect(repo.written[0].task.id).toBe("OB-003"); // only OB-prefixed ids count
  });
});

describe("ticket gate (C5)", () => {
  it("created task carries references (default []) and DoD + Plan body scaffolding with unchecked boxes", async () => {
    const repo = new MemoryRepo();
    await createTask(repo, project, { summary: "Gated", body: "my description" }, []);
    const w = repo.written[0];
    expect(w.task.references).toEqual([]);
    expect(w.body).toContain("## Definition of Done");
    expect(w.body).toContain("## Implementation Plan");
    expect(w.body).toMatch(/- \[ \]/); // unchecked boxes
    expect(w.body).toContain("<!-- DOD:BEGIN -->");
    expect(w.body).toContain("<!-- SECTION:PLAN:BEGIN -->");
  });

  it("does not clobber a provided references value", async () => {
    const repo = new MemoryRepo();
    await createTask(repo, project, { summary: "Refs", references: [".skillgrid/change-specs/x/briefing.md"] }, []);
    expect(repo.written[0].task.references).toEqual([".skillgrid/change-specs/x/briefing.md"]);
  });

  it("a body already containing the sections is not doubled", async () => {
    const repo = new MemoryRepo();
    const body = `my description

## Definition of Done

<!-- DOD:BEGIN -->
- [ ] Custom DoD
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
- [ ] Custom step
<!-- SECTION:PLAN:END -->`;
    await createTask(repo, project, { summary: "NoDouble", body }, []);
    const out = repo.written[0].body;
    expect(out.split("## Definition of Done")).toHaveLength(2); // exactly one heading
    expect(out.split("## Implementation Plan")).toHaveLength(2);
    expect(out).toContain("- [ ] Custom DoD");
    expect(out).not.toContain("Tests pass"); // default scaffolding not appended
  });
});
