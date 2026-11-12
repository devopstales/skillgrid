import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import { FileSystemPort, ProjectConfig } from "../../src/application/ports";
import { discoverProjects, walkUpForFile } from "../../src/infrastructure/config";
import { FileTaskRepository } from "../../src/infrastructure/task-repository";

const TASK_MD = (id: string, status: string) =>
  `---\nid: ${id}\ntitle: "Task ${id}"\nstatus: "${status}"\ntype: feature\npriority: medium\n---\nBody ${id}\n`;

const ROOT_CONFIG = (name: string, backlogDir: string) =>
  [
    `project_name: "${name}"`,
    `default_status: "To Do"`,
    `statuses: ["To Do", "In Progress", "Done"]`,
    `labels: []`,
    `types: ["feature", "bug", "enhancement", "refactor", "docs", "chore"]`,
    `priorities: ["high", "medium", "low"]`,
    `task_prefix: "task"`,
    `backlog_directory: "${backlogDir}"`,
  ].join("\n");

/** Vault-relative FS backed by a real temp directory (same mechanics as
 *  ObsidianFileSystem's abs-path methods: node fs from the vault root). */
class FakeFS implements FileSystemPort {
  constructor(private vaultAbs: string) {}

  private vaultRel(p: string): string {
    return path.join(this.vaultAbs, p.replace(/^\/+/, ""));
  }
  async exists(p: string) {
    return fs.existsSync(this.vaultRel(p));
  }
  async readText(p: string) {
    return fs.promises.readFile(this.vaultRel(p), "utf8");
  }
  async writeText(p: string, c: string) {
    await fs.promises.mkdir(path.dirname(this.vaultRel(p)), { recursive: true });
    await fs.promises.writeFile(this.vaultRel(p), c);
  }
  async listDirs(p: string) {
    const dir = this.vaultRel(p);
    if (!fs.existsSync(dir)) return [];
    return fs
      .readdirSync(dir, { withFileTypes: true })
      .filter((d) => d.isDirectory())
      .map((d) => d.name);
  }
  async listFiles(p: string) {
    const dir = this.vaultRel(p);
    if (!fs.existsSync(dir)) return [];
    return fs
      .readdirSync(dir, { withFileTypes: true })
      .filter((d) => d.isFile())
      .map((d) => d.name);
  }
  async getVaultRootAbs() {
    return this.vaultAbs;
  }
  async findAncestorProjectRoot(maxDepth = 5) {
    return walkUpForFile(this.vaultAbs, "backlog.config.yml", maxDepth, (x) => fs.existsSync(x));
  }
  async readTextAbs(a: string) {
    return fs.promises.readFile(a, "utf8");
  }
}

let tmp: string;

beforeEach(() => {
  tmp = fs.mkdtempSync(path.join(os.tmpdir(), "backlogmd-cfg-"));
});
afterEach(() => {
  fs.rmSync(tmp, { recursive: true, force: true });
});

function write(rel: string, content: string) {
  const p = path.join(tmp, rel);
  fs.mkdirSync(path.dirname(p), { recursive: true });
  fs.writeFileSync(p, content);
}

describe("project discovery", () => {
  it("discovers a skillgrid vault whose project root sits above the vault", async () => {
    // <repo>/backlog.config.yml
    // <repo>/.skillgrid/projectwiki/            <- vault
    // <repo>/.skillgrid/projectwiki/backlog/tasks/...
    const repo = tmp;
    const vault = path.join(repo, ".skillgrid", "projectwiki");
    write("backlog.config.yml", ROOT_CONFIG("KubeDash", ".skillgrid/projectwiki/backlog"));
    write(".skillgrid/projectwiki/config.json", "{}");
    write(".skillgrid/projectwiki/backlog/tasks/task-001 - Scaffold.md", TASK_MD("TASK-001", "To Do"));
    write(".skillgrid/projectwiki/backlog/tasks/task-002 - Wire-up.md", TASK_MD("TASK-002", "In Progress"));

    const fsPort = new FakeFS(vault);
    const projects = await discoverProjects(fsPort, [""], []);

    expect(projects).toHaveLength(1);
    const p = projects[0];
    expect(p.name).toBe("KubeDash");
    expect(p.statuses).toEqual(["To Do", "In Progress", "Done"]);
    expect(p.types).toEqual(["feature", "bug", "enhancement", "refactor", "docs", "chore"]);
    expect(p.vaultBacklogDir).toBe("backlog");
    expect(p.backlogDirectory).toBe(".skillgrid/projectwiki/backlog");

    const repo2 = new FileTaskRepository(fsPort);
    const tasks = await repo2.list(p);
    expect(tasks.map((t) => t.id).sort()).toEqual(["TASK-001", "TASK-002"]);
  });

  it("discovers a child folder with a legacy .backlog/config.yml from a scan", async () => {
    const vault = tmp;
    write("Demo/.backlog/config.yml", "project_name: Demo\nstatuses: [\"Open\", \"Closed\"]\n");
    write("Demo/.backlog/tasks/task-001 - a.md", TASK_MD("TASK-001", "Open"));

    const fsPort = new FakeFS(vault);
    const projects = await discoverProjects(fsPort, [""], []);

    expect(projects).toHaveLength(1);
    expect(projects[0].name).toBe("Demo");
    expect(projects[0].rootPath).toBe("Demo");
    expect(projects[0].vaultBacklogDir).toBe("Demo/.backlog");
    expect(projects[0].statuses).toEqual(["Open", "Closed"]);
  });

  it("honors backlog_directory when the project root is the vault root", async () => {
    const vault = tmp;
    write("backlog.config.yml", ROOT_CONFIG("Custom", "custom-backlog"));
    write("custom-backlog/tasks/task-001 - a.md", TASK_MD("TASK-001", "To Do"));

    const fsPort = new FakeFS(vault);
    const projects = await discoverProjects(fsPort, [""], []);

    expect(projects).toHaveLength(1);
    expect(projects[0].name).toBe("Custom");
    expect(projects[0].vaultBacklogDir).toBe("custom-backlog");
    const tasks = await new FileTaskRepository(fsPort).list(projects[0]);
    expect(tasks).toHaveLength(1);
  });

  it("merges manual roots with the scan and dedupes", async () => {
    const vault = tmp;
    write("Demo/.backlog/config.yml", "project_name: Demo\nstatuses: [\"Open\"]\n");
    write("Demo/.backlog/tasks/task-001 - a.md", TASK_MD("TASK-001", "Open"));

    const fsPort = new FakeFS(vault);
    const projects = await discoverProjects(fsPort, [""], ["Demo"]);

    expect(projects).toHaveLength(1);
    expect(projects[0].name).toBe("Demo");
  });

  it("rejects an ancestor whose backlog directory is not inside the vault", async () => {
    const repo = tmp;
    const vault = path.join(repo, ".skillgrid", "projectwiki");
    write("backlog.config.yml", ROOT_CONFIG("Outside", ".backlog")); // .backlog stays at repo level
    write(".backlog/tasks/task-001 - a.md", TASK_MD("TASK-001", "To Do"));

    const fsPort = new FakeFS(vault);
    const projects = await discoverProjects(fsPort, [""], []);

    expect(projects).toHaveLength(0);
  });

  it("finds no project in a vault without any backlog config", async () => {
    const vault = tmp;
    write("notes/random.md", "# hi");

    const fsPort = new FakeFS(vault);
    const projects = await discoverProjects(fsPort, [""], []);

    expect(projects).toHaveLength(0);
  });

  it("walkUpForFile stops at maxDepth", () => {
    const deep = path.join(tmp, "a", "b", "c");
    fs.mkdirSync(deep, { recursive: true });
    const found = walkUpForFile(deep, "marker.txt", 1, (p) => fs.existsSync(p));
    expect(found).toBeNull();
    fs.writeFileSync(path.join(tmp, "marker.txt"), "x");
    expect(walkUpForFile(deep, "marker.txt", 5, (p) => fs.existsSync(p))).toBe(tmp);
  });
});
