import { describe, it, expect } from "vitest";
import { linkTasks } from "../../src/application/link-tasks";
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

const t = (id: string, deps?: string[]): Task => ({
  id,
  title: id,
  status: "To Do",
  path: `tasks/${id}.md`,
  body: `body-${id}`,
  ...(deps ? { dependencies: deps } : {}),
});

describe("linkTasks", () => {
  it("add: both sides reference each other (bidirectional), bodies kept", async () => {
    const repo = new MemoryRepo();
    await linkTasks(repo, project, t("OB-1"), t("OB-7"), true);
    expect(repo.written).toHaveLength(2);
    const byId = new Map(repo.written.map((w) => [w.task.id, w]));
    expect(byId.get("OB-1")?.task.dependencies).toEqual(["OB-7"]);
    expect(byId.get("OB-7")?.task.dependencies).toEqual(["OB-1"]);
    expect(byId.get("OB-1")?.body).toBe("body-OB-1");
    expect(byId.get("OB-7")?.body).toBe("body-OB-7");
  });

  it("add: existing dependencies are preserved, not replaced", async () => {
    const repo = new MemoryRepo();
    await linkTasks(repo, project, t("OB-1", ["OB-2"]), t("OB-7", ["OB-3"]), true);
    const byId = new Map(repo.written.map((w) => [w.task.id, w]));
    expect(byId.get("OB-1")?.task.dependencies).toEqual(["OB-2", "OB-7"]);
    expect(byId.get("OB-7")?.task.dependencies).toEqual(["OB-3", "OB-1"]);
  });

  it("remove: neither task references the other", async () => {
    const repo = new MemoryRepo();
    await linkTasks(repo, project, t("OB-1", ["OB-7"]), t("OB-7", ["OB-1"]), false);
    const byId = new Map(repo.written.map((w) => [w.task.id, w]));
    expect(byId.get("OB-1")?.task.dependencies).toEqual([]);
    expect(byId.get("OB-7")?.task.dependencies).toEqual([]);
  });
});
