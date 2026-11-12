import { describe, it, expect } from "vitest";
import { startMilestone, completeMilestone, planTask } from "../../src/application/milestone";
import type { ProjectConfig, TaskRepository, MilestoneRepository } from "../../src/application/ports";
import type { Task } from "../../src/domain/task";
import type { Milestone } from "../../src/domain/milestone";

class MemoryTasks implements TaskRepository {
  tasks: Task[] = [];
  async list() {
    return this.tasks;
  }
  async get(_p: ProjectConfig, id: string) {
    return this.tasks.find((t) => t.id === id) ?? null;
  }
  async write(_p: ProjectConfig, task: Task, _body: string) {
    const i = this.tasks.findIndex((t) => t.id === task.id);
    if (i === -1) this.tasks.push(task);
    else this.tasks[i] = task;
  }
  async create(_p: ProjectConfig, task: Task, _body: string) {
    this.tasks.push(task);
    return task.path;
  }
}

class MemoryMilestones implements MilestoneRepository {
  items: Milestone[] = [];
  async list() {
    return this.items;
  }
  async write(_p: ProjectConfig, m: Milestone, _body: string) {
    const i = this.items.findIndex((x) => x.id === m.id);
    if (i === -1) this.items.push(m);
    else this.items[i] = m;
  }
  async create(_p: ProjectConfig, m: Milestone, _body: string) {
    this.items.push(m);
    return m.path;
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

const task = (id: string, milestone?: string, status = "To Do", ordinal = 1000): Task => ({
  id,
  title: id,
  status,
  milestone,
  ordinal,
  path: `tasks/${id}.md`,
  body: `b-${id}`,
});

const ms = (id: string, state?: Milestone["state"]): Milestone => ({ id, title: id, state, path: `milestones/${id}.md`, body: `mb-${id}` });

describe("startMilestone", () => {
  it("sets active + start_date, auto-completing a previously active milestone", async () => {
    const tasks = new MemoryTasks();
    const milestones = new MemoryMilestones();
    milestones.items = [ms("m-0", "active"), ms("m-1", "planned")];
    tasks.tasks = [task("TASK-001", "m-0", "To Do"), task("TASK-002", "m-0", "Done")];

    await startMilestone(tasks, milestones, project, ms("m-1"));

    const m1 = milestones.items.find((x) => x.id === "m-1")!;
    expect(m1.state).toBe("active");
    expect(m1.start_date).toBeTruthy();
    const m0 = milestones.items.find((x) => x.id === "m-0")!;
    expect(m0.state).toBe("completed");
    expect(m0.completed_tasks).toEqual(["TASK-002"]);
    expect(m0.carried_over_tasks).toEqual(["TASK-001"]);
    expect(tasks.tasks.find((t) => t.id === "TASK-001")!.milestone).toBeUndefined();
    expect(tasks.tasks.find((t) => t.id === "TASK-002")!.milestone).toBe("m-0");
  });
});

describe("completeMilestone", () => {
  it("records finished tasks and carries unfinished ones back to the backlog", async () => {
    const tasks = new MemoryTasks();
    const milestones = new MemoryMilestones();
    milestones.items = [ms("m-0", "active")];
    tasks.tasks = [
      task("TASK-001", "m-0", "Done"),
      task("TASK-002", "m-0", "In Progress"),
      task("TASK-003", undefined, "To Do"),
    ];

    await completeMilestone(tasks, milestones, project, ms("m-0"), tasks.tasks);

    const m0 = milestones.items.find((x) => x.id === "m-0")!;
    expect(m0.state).toBe("completed");
    expect(m0.completed_tasks).toEqual(["TASK-001"]);
    expect(m0.carried_over_tasks).toEqual(["TASK-002"]);
    expect(m0.end_date).toBeTruthy();
    expect(tasks.tasks.find((t) => t.id === "TASK-002")!.milestone).toBeUndefined();
    expect(tasks.tasks.find((t) => t.id === "TASK-001")!.milestone).toBe("m-0");
    expect(tasks.tasks.find((t) => t.id === "TASK-003")!.milestone).toBeUndefined();
  });
});

describe("planTask", () => {
  it("plans into a milestone and appends to its ordinal range", async () => {
    const tasks = new MemoryTasks();
    tasks.tasks = [task("TASK-001", "m-0", "To Do", 1000), task("TASK-002", "m-0", "To Do", 2000), task("TASK-003", undefined, "To Do")];

    await planTask(tasks, project, task("TASK-003", undefined, "To Do"), "m-0", tasks.tasks);

    const t3 = tasks.tasks.find((t) => t.id === "TASK-003")!;
    expect(t3.milestone).toBe("m-0");
    expect(t3.ordinal).toBe(3000);
  });

  it("unplans (null) and clears the milestone", async () => {
    const tasks = new MemoryTasks();
    tasks.tasks = [task("TASK-001", "m-0", "To Do")];

    await planTask(tasks, project, task("TASK-001", "m-0", "To Do"), null, tasks.tasks);

    expect(tasks.tasks.find((t) => t.id === "TASK-001")!.milestone).toBeUndefined();
  });
});
