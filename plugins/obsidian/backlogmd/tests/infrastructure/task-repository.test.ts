import { describe, it, expect, vi, afterEach } from "vitest";
import { FileTaskRepository } from "../../src/infrastructure/task-repository";
import { FileSystemPort, ProjectConfig } from "../../src/application/ports";

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

const MD = (id: string, status: string) => `---\nid: ${id}\ntitle: "Task ${id}"\nstatus: "${status}"\n---\nBody ${id}\n`;
const BROKEN = `---\nid: TASK-002\nstatus: [unclosed\n---\nBody\n`;

/** In-memory vault: files keyed by vault-relative path. */
class InMemoryFS implements FileSystemPort {
  files = new Map<string, string>();
  maxConcurrentReads = 0;
  private inFlight = 0;

  add(dir: string, name: string, content: string) {
    this.files.set(`${dir}/${name}`, content);
  }

  async exists(p: string) {
    return this.files.has(p);
  }
  async readText(p: string) {
    this.inFlight++;
    this.maxConcurrentReads = Math.max(this.maxConcurrentReads, this.inFlight);
    await Promise.resolve(); // yield so parallel reads interleave (no wall clock)
    this.inFlight--;
    const content = this.files.get(p);
    if (content === undefined) throw new Error(`ENOENT: ${p}`);
    return content;
  }
  async writeText(p: string, content: string) {
    this.files.set(p, content);
  }
  async listDirs(p: string) {
    const prefix = p.replace(/^\/+/, "") + "/";
    const dirs = new Set<string>();
    for (const key of this.files.keys()) {
      if (key.startsWith(prefix)) dirs.add(key.slice(prefix.length).split("/")[0]);
    }
    return [...dirs];
  }
  async listFiles(p: string) {
    const prefix = p.replace(/^\/+/, "") + "/";
    return [...this.files.keys()]
      .filter((key) => key.startsWith(prefix) && !key.slice(prefix.length).includes("/"))
      .map((key) => key.slice(prefix.length));
  }
  async getVaultRootAbs() {
    return "";
  }
  async findAncestorProjectRoot() {
    return null;
  }
  async readTextAbs() {
    throw new Error("not supported");
  }
}

afterEach(() => vi.unstubAllGlobals());

describe("FileTaskRepository.list", () => {
  it("returns all parseable tasks", async () => {
    const fs = new InMemoryFS();
    fs.add(".backlog/tasks", "task-001 - a.md", MD("TASK-001", "To Do"));
    fs.add(".backlog/tasks", "task-002 - b.md", MD("TASK-002", "In Progress"));
    const repo = new FileTaskRepository(fs);
    const tasks = await repo.list(project);
    expect(tasks.map((t) => t.id).sort()).toEqual(["TASK-001", "TASK-002"]);
    expect(repo.lastErrors).toEqual([]);
  });

  it("isolates a malformed file: others still listed, bad path recorded in lastErrors", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const fs = new InMemoryFS();
    fs.add(".backlog/tasks", "task-001 - a.md", MD("TASK-001", "To Do"));
    fs.add(".backlog/tasks", "task-002 - broken.md", BROKEN);
    fs.add(".backlog/tasks", "task-003 - c.md", MD("TASK-003", "Done"));
    const repo = new FileTaskRepository(fs);
    const tasks = await repo.list(project);
    expect(tasks.map((t) => t.id).sort()).toEqual(["TASK-001", "TASK-003"]);
    expect(repo.lastErrors).toEqual([".backlog/tasks/task-002 - broken.md"]);
    expect(errorSpy).toHaveBeenCalled();
  });

  it("a second list() call starts clean (lastErrors not cumulative)", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    const fs = new InMemoryFS();
    fs.add(".backlog/tasks", "task-001 - a.md", MD("TASK-001", "To Do"));
    fs.add(".backlog/tasks", "task-002 - broken.md", BROKEN);
    const repo = new FileTaskRepository(fs);
    await repo.list(project);
    expect(repo.lastErrors).toEqual([".backlog/tasks/task-002 - broken.md"]);
    const again = await repo.list(project);
    expect(again).toHaveLength(1);
    // still exactly the one bad path — not doubled by the previous call
    expect(repo.lastErrors).toEqual([".backlog/tasks/task-002 - broken.md"]);
    fs.add(".backlog/tasks", "task-002 - broken.md", MD("TASK-002", "To Do")); // file fixed
    expect(await repo.list(project)).toHaveLength(2);
    expect(repo.lastErrors).toEqual([]);
  });

  it("reads task files in parallel (not serially)", async () => {
    const fs = new InMemoryFS();
    fs.add(".backlog/tasks", "task-001 - a.md", MD("TASK-001", "To Do"));
    fs.add(".backlog/tasks", "task-002 - b.md", MD("TASK-002", "To Do"));
    fs.add(".backlog/tasks", "task-003 - c.md", MD("TASK-003", "To Do"));
    const repo = new FileTaskRepository(fs);
    const tasks = await repo.list(project);
    expect(tasks).toHaveLength(3);
    expect(fs.maxConcurrentReads).toBeGreaterThanOrEqual(2);
  });

  it("returns [] when the tasks dir is missing, without throwing", async () => {
    const repo = new FileTaskRepository(new InMemoryFS());
    expect(await repo.list(project)).toEqual([]);
    expect(repo.lastErrors).toEqual([]);
  });
});

describe("FileTaskRepository raw round-trip via write", () => {
  it("write preserves unknown frontmatter fields and order", async () => {
    const fs = new InMemoryFS();
    const original = `---
title: Keep order
id: TASK-7
status: To Do
rank: 42
---
Body
`;
    fs.add(".backlog/tasks", "task-007 - keep-order.md", original);
    const repo = new FileTaskRepository(fs);
    const tasks = await repo.list(project);
    expect(tasks).toHaveLength(1);
    await repo.write(project, { ...tasks[0], status: "In Progress" }, tasks[0].body ?? "");
    expect(fs.files.get(".backlog/tasks/task-007 - keep-order.md")).toBe(`---
title: Keep order
id: TASK-7
status: In Progress
rank: 42
---
Body
`);
  });
});
