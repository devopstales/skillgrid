import { describe, it, expect } from "vitest";
import { editTask } from "../../src/application/edit-task";
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
  taskPrefix: "OB",
};

const t = (): Task => ({
  id: "OB-1",
  title: "Edit me",
  status: "To Do",
  path: "tasks/OB-1.md",
  body: "orig body",
});

describe("editTask", () => {
  it("applies the patch, stamps updated_date (YYYY-MM-DD), keeps stored body", async () => {
    const repo = new MemoryRepo();
    await editTask(repo, project, t(), { priority: "high" });
    expect(repo.written).toHaveLength(1);
    const w = repo.written[0];
    expect(w.task.priority).toBe("high");
    expect(w.task.id).toBe("OB-1");
    expect(w.task.title).toBe("Edit me");
    expect(w.task.updated_date).toBe(new Date().toISOString().slice(0, 10));
    expect(w.body).toBe("orig body");
  });

  it("explicit body argument overrides the stored body", async () => {
    const repo = new MemoryRepo();
    await editTask(repo, project, t(), { status: "In Progress" }, "new body");
    const w = repo.written[0];
    expect(w.task.status).toBe("In Progress");
    expect(w.body).toBe("new body");
  });
});
