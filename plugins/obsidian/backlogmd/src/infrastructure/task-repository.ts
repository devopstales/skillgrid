import { ProjectConfig, TaskRepository, FileSystemPort } from "../application/ports";
import { Task } from "../domain/task";
import { parseTask, serializeTask } from "./vault";

export class FileTaskRepository implements TaskRepository {
  /** Vault-relative paths of task files skipped in the most recent list()
   *  call (malformed YAML frontmatter or unreadable). Cleared at the start
   *  of every list() call. */
  lastErrors: string[] = [];

  constructor(private fs: FileSystemPort) {}

  private tasksDir(project: ProjectConfig): string {
    return `${project.vaultBacklogDir}/tasks`;
  }

  async list(project: ProjectConfig): Promise<Task[]> {
    const dir = this.tasksDir(project);
    this.lastErrors = [];
    let files: string[] = [];
    try {
      files = await this.fs.listFiles(dir);
    } catch {
      return [];
    }
    // Parallel reads; a parse failure in one file skips that file only and
    // records its path — it never blanks the whole board.
    const results = await Promise.all(
      files
        .filter((x) => x.endsWith(".md"))
        .map(async (f): Promise<Task | null> => {
          const filePath = `${dir}/${f}`;
          try {
            const content = await this.fs.readText(filePath);
            return parseTask(filePath, content).task;
          } catch (err) {
            this.lastErrors.push(filePath);
            console.error(`[backlogmd] skipping malformed task file ${filePath}: ${err instanceof Error ? err.message : String(err)}`);
            return null;
          }
        }),
    );
    return results.filter((t): t is Task => t !== null);
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
