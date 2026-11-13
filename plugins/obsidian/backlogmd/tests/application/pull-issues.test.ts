import { describe, it, expect, vi, afterEach } from "vitest";
import { pullIssues, nextTaskId } from "../../src/application/pull-issues";
import type { ProjectConfig, TaskRepository } from "../../src/application/ports";
import type { Task } from "../../src/domain/task";
import type { GhIssue } from "../../src/infrastructure/github";

const project: ProjectConfig = {
  name: "test",
  rootPath: "/",
  backlogDirectory: "backlog",
  vaultBacklogDir: "backlog",
  statuses: ["To Do", "In Progress", "Done"],
  types: [],
  priorities: [],
  doneStatuses: ["Done"],
  taskPrefix: "task",
};

function task(id: string, extra: Partial<Task> = {}): Task {
  return { id, title: id, status: "To Do", path: `backlog/tasks/${id}.md`, ...extra };
}

class MemoryTasks implements TaskRepository {
  tasks: Task[] = [];
  async list() {
    return this.tasks;
  }
  async get(_p: ProjectConfig, id: string) {
    return this.tasks.find((t) => t.id === id) ?? null;
  }
  async write(_p: ProjectConfig, t: Task, _body: string) {
    const i = this.tasks.findIndex((x) => x.id === t.id);
    if (i === -1) this.tasks.push(t);
    else this.tasks[i] = t;
  }
  async create(_p: ProjectConfig, t: Task, _body: string) {
    this.tasks.push(t);
    return t.path;
  }
}

function mockFetch(issues: (GhIssue & { pull_request?: unknown })[]) {
  vi.stubGlobal(
    "fetch",
    async () => ({ ok: true, status: 200, json: async () => issues }),
  );
}

afterEach(() => vi.unstubAllGlobals());

describe("nextTaskId", () => {
  it("starts at prefix-001 for an empty tracker", () => {
    expect(nextTaskId([], "task")).toBe("task-001");
  });

  it("is the highest numeric suffix + 1, zero-padded", () => {
    expect(nextTaskId([task("task-103")], "task")).toBe("task-104");
    expect(nextTaskId([task("task-103"), task("task-105")], "task")).toBe("task-106");
    expect(nextTaskId([task("task-9")], "task")).toBe("task-010");
  });

  it("is case-insensitive and ignores subtask or unrelated ids", () => {
    expect(nextTaskId([task("TASK-105")], "task")).toBe("task-106");
    expect(nextTaskId([task("task-116.16"), task("FEATURE-7")], "task")).toBe("task-001");
  });
});

describe("pullIssues", () => {
  it("first pull creates tasks with source ref and code-blocked body", async () => {
    mockFetch([
      { number: 1, title: "Open issue", state: "open", body: "the body" },
      { number: 2, title: "Closed issue", state: "closed" },
      { number: 3, title: "A pull request", state: "open", pull_request: {} },
    ]);
    const repo = new MemoryTasks();
    await pullIssues(repo, project, "tok", "owner", "repo");

    expect(repo.tasks.map((t) => t.id)).toEqual(["task-001", "task-002"]);
    expect(repo.tasks[0].references).toEqual(["github:owner/repo#1"]);
    expect(repo.tasks[0].body).toBe("```\nthe body\n```");
    expect(repo.tasks[0].status).toBe("To Do");
    expect(repo.tasks[1].status).toBe("Done");
  });

  it("second pull syncs title/assignee/state only, keeps local planning and body", async () => {
    mockFetch([{ number: 1, title: "v1", state: "open", assignee: "alice", body: "b1" }]);
    const repo = new MemoryTasks();
    await pullIssues(repo, project, "tok", "owner", "repo");
    // local planning happens after the first pull
    repo.tasks[0] = { ...repo.tasks[0], milestone: "m-0", priority: "high" };

    mockFetch([{ number: 1, title: "v2", state: "closed", assignee: "bob" }]);
    await pullIssues(repo, project, "tok", "owner", "repo");

    expect(repo.tasks).toHaveLength(1);
    const t = repo.tasks[0];
    expect(t.title).toBe("v2");
    expect(t.assignee).toEqual(["bob"]);
    expect(t.status).toBe("Done");
    expect(t.milestone).toBe("m-0"); // local planning untouched
    expect(t.priority).toBe("high");
    expect(t.body).toBe("```\nb1\n```"); // description never overwritten
  });

  it("assigns collision-free ids when several new issues arrive at once", async () => {
    mockFetch([
      { number: 1, title: "a", state: "open" },
      { number: 2, title: "b", state: "open" },
      { number: 3, title: "c", state: "open" },
    ]);
    const repo = new MemoryTasks();
    await pullIssues(repo, project, "tok", "owner", "repo");
    expect(repo.tasks.map((t) => t.id)).toEqual(["task-001", "task-002", "task-003"]);
  });
});
