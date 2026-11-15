import { describe, it, expect } from "vitest";
import { startMilestone, completeMilestone, planTask } from "../../src/application/milestone";
import type { ProjectConfig, TaskRepository, MilestoneRepository } from "../../src/application/ports";
import type { Task } from "../../src/domain/task";
import type { Milestone } from "../../src/domain/milestone";

class MemoryTasks implements TaskRepository {
  lastErrors: string[] = [];
  tasks: Task[] = [];
  /** When set, write() throws for tasks matching the predicate. */
  failWrite?: (t: Task) => boolean;
  writeCalls: string[] = [];
  async list() {
    return this.tasks;
  }
  async get(_p: ProjectConfig, id: string) {
    return this.tasks.find((t) => t.id === id) ?? null;
  }
  async write(_p: ProjectConfig, task: Task, _body: string) {
    this.writeCalls.push(task.id);
    if (this.failWrite?.(task)) throw new Error("disk on fire");
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
  failWrite = false;
  writeCalls: string[] = [];
  async list() {
    return this.items;
  }
  async write(_p: ProjectConfig, m: Milestone, _body: string) {
    this.writeCalls.push(m.id);
    if (this.failWrite) throw new Error("disk on fire");
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

  it("rethrows a descriptive error when the milestone write fails; state not marked active (H11)", async () => {
    const tasks = new MemoryTasks();
    const milestones = new MemoryMilestones();
    milestones.failWrite = true;
    milestones.items = [ms("m-1", "planned")];

    await expect(startMilestone(tasks, milestones, project, ms("m-1"))).rejects.toThrow(/m-1/);
    expect(milestones.items.find((x) => x.id === "m-1")?.state).toBe("planned"); // unchanged
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

  it("rethrows a descriptive error when a carry-over task write fails; milestone file NOT rewritten (H11)", async () => {
    const tasks = new MemoryTasks();
    const milestones = new MemoryMilestones();
    milestones.items = [ms("m-0", "active")];
    tasks.tasks = [
      task("TASK-001", "m-0", "Done"),
      task("TASK-002", "m-0", "In Progress"), // this write fails
    ];
    tasks.failWrite = (t) => t.id === "TASK-002";

    await expect(completeMilestone(tasks, milestones, project, ms("m-0"), tasks.tasks)).rejects.toThrow(/m-0/);
    // milestone file write is last and never reached
    expect(milestones.writeCalls).toEqual([]);
    // on-disk state not marked completed
    expect(milestones.items.find((x) => x.id === "m-0")!.state).toBe("active");
  });

  it("writes carry-over tasks first, milestone file last (H11 ordering)", async () => {
    const tasks = new MemoryTasks();
    const milestones = new MemoryMilestones();
    milestones.items = [ms("m-0", "active")];
    tasks.tasks = [task("TASK-001", "m-0", "Done"), task("TASK-002", "m-0", "In Progress")];

    await completeMilestone(tasks, milestones, project, ms("m-0"), tasks.tasks);

    expect(tasks.writeCalls).toEqual(["TASK-002"]); // task write first
    expect(milestones.writeCalls).toEqual(["m-0"]); // milestone write last
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
