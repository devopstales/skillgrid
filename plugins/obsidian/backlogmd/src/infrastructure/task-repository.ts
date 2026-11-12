import { ProjectConfig, TaskRepository, FileSystemPort } from "../application/ports";
import { Task } from "../domain/task";
import { parseTask, serializeTask } from "./vault";
import path from "path";

export class FileTaskRepository implements TaskRepository {
  constructor(private fs: FileSystemPort) {}

  private tasksDir(project: ProjectConfig): string {
    return `${project.vaultBacklogDir}/tasks`;
  }

  async list(project: ProjectConfig): Promise<Task[]> {
    const dir = this.tasksDir(project);
    let files: string[] = [];
    try {
      files = await this.fs.listFiles(dir);
    } catch {
      return [];
    }
    const tasks: Task[] = [];
    for (const f of files.filter((x) => x.endsWith(".md"))) {
      const content = await this.fs.readText(`${dir}/${f}`);
      tasks.push(parseTask(`${dir}/${f}`, content).task);
    }
    return tasks;
  }

  async get(project: ProjectConfig, id: string): Promise<Task | null> {
    const tasks = await this.list(project);
    return tasks.find((t) => t.id === id) ?? null;
  }

  async write(project: ProjectConfig, task: Task, body: string): Promise<void> {
    const dir = this.tasksDir(project);
    const filePath = task.path || `${dir}/${task.id}.md`;
    const content = serializeTask({ ...task, path: filePath }, body);
    await this.fs.writeText(filePath, content);
  }

  async create(project: ProjectConfig, task: Task, body: string): Promise<string> {
    const dir = this.tasksDir(project);
    // Match the backlog CLI convention: `<prefix>-<padded n> - <title-slug>.md`
    const num = task.id.replace(/[^0-9]/g, "");
    const slug = task.title
      .trim()
      .replace(/[^\p{L}\p{N}.-]+/gu, "-")
      .replace(/-{2,}/g, "-")
      .replace(/^-+|-+$/g, "")
      .slice(0, 80);
    const base = `${project.taskPrefix}-${num.padStart(3, "0")}`;
    const filePath = slug ? `${dir}/${base} - ${slug}.md` : `${dir}/${base}.md`;
    const t = { ...task, path: filePath };
    const content = serializeTask(t, body);
    await this.fs.writeText(filePath, content);
    return filePath;
  }
}