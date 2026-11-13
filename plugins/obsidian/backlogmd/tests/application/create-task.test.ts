import { describe, it, expect } from "vitest";
import { nextTaskId, createTask } from "../../src/application/create-task";
import { TaskRepository, ProjectConfig } from "../../src/application/ports";
import { Task } from "../../src/domain/task";

const t = (id: string, ordinal?: number): Task => ({ id, title: id, status: "To Do", ordinal, path: "" });

class MemoryRepo implements TaskRepository {
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
    expect(w.body).toBe("nb");
    expect(ret).toBe("tasks/OB-004.md"); // create() return passes through
  });

  it("starts at ordinal 1000 and prefix-001 when no tasks exist", async () => {
    const repo = new MemoryRepo();
    await createTask(repo, project, { summary: "First" }, []);
    const w = repo.written[0];
    expect(w.task.id).toBe("OB-001");
    expect(w.task.ordinal).toBe(1000);
    expect(w.body).toBe(""); // no body given -> empty string
  });
});
