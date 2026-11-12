import { describe, it, expect } from "vitest";
import { moveTask, reorderColumn } from "../../src/application/move-task";
import { TaskRepository, ProjectConfig } from "../../src/application/ports";
import { Task } from "../../src/domain/task";

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
    return task.path;
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
  taskPrefix: "task",
};

const t = (id: string, status: string, ordinal?: number): Task => ({
  id,
  title: id,
  status,
  ordinal,
  path: `tasks/${id}.md`,
  body: `body-${id}`,
});

describe("moveTask", () => {
  it("writes status, ordinal and updated_date, keeping body", async () => {
    const repo = new MemoryRepo();
    await moveTask(repo, project, t("TASK-001", "To Do", 1000), "In Progress", 2000);
    expect(repo.written).toHaveLength(1);
    const w = repo.written[0];
    expect(w.task.status).toBe("In Progress");
    expect(w.task.ordinal).toBe(2000);
    expect(w.task.updated_date).toBeTruthy();
    expect(w.body).toBe("body-TASK-001");
  });
});

describe("reorderColumn", () => {
  it("reassigns 1000-step ordinals in the given order, bodies intact", async () => {
    const repo = new MemoryRepo();
    const a = t("TASK-001", "To Do", 1000);
    const b = t("TASK-002", "To Do", 2000);
    const c = t("TASK-003", "To Do", 3000);
    await reorderColumn(repo, project, [c, a, b]);
    const byId = new Map(repo.written.map((w) => [w.task.id, w]));
    expect(byId.get("TASK-003")?.task.ordinal).toBe(1000);
    expect(byId.get("TASK-001")?.task.ordinal).toBe(2000);
    expect(byId.get("TASK-002")?.task.ordinal).toBe(3000);
    expect(byId.get("TASK-003")?.body).toBe("body-TASK-003");
  });

  it("writes nothing for an empty column", async () => {
    const repo = new MemoryRepo();
    await reorderColumn(repo, project, []);
    expect(repo.written).toHaveLength(0);
  });
});
