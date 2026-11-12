import { ProjectConfig, TaskRepository } from "./ports";
import { Task } from "../domain/task";

export interface CreateTaskInput {
  summary: string;
  body?: string;
  type?: string;
  priority?: string;
  status?: string;
  assignee?: string[];
  milestone?: string;
  dependencies?: string[];
}

/** Next zero-padded ID: max numeric suffix of existing tasks + 1
 *  (count-based IDs collide after deletions). */
export function nextTaskId(existing: Task[], prefix: string): string {
  let max = 0;
  for (const t of existing) {
    const m = t.id.match(/(\d+)\s*$/);
    if (m) max = Math.max(max, parseInt(m[1], 10));
  }
  return `${prefix.toUpperCase()}-${String(max + 1).padStart(3, "0")}`;
}

export async function createTask(
  repo: TaskRepository,
  project: ProjectConfig,
  input: CreateTaskInput,
  existing: Task[],
): Promise<string> {
  const id = nextTaskId(existing, project.taskPrefix);
  const ordinal = existing.length ? Math.max(...existing.map((t) => t.ordinal ?? 0)) + 1000 : 1000;
  const task: Task = {
    id,
    title: input.summary,
    status: input.status ?? project.statuses[0] ?? "To Do",
    type: input.type || undefined,
    priority: input.priority || undefined,
    assignee: input.assignee?.length ? input.assignee : undefined,
    milestone: input.milestone || undefined,
    dependencies: input.dependencies?.length ? input.dependencies : undefined,
    ordinal,
    path: "",
  };
  return repo.create(project, task, input.body ?? "");
}
