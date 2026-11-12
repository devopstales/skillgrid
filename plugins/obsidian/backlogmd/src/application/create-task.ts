import { ProjectConfig, TaskRepository } from "./ports";
import { Task } from "../domain/task";

export interface CreateTaskInput {
  summary: string;
  type?: string;
  priority?: string;
  status?: string;
}

export async function createTask(
  repo: TaskRepository,
  project: ProjectConfig,
  input: CreateTaskInput,
  existingCount: number,
): Promise<string> {
  const id = `${project.taskPrefix.toUpperCase()}-${existingCount + 1}`;
  const task: Task = {
    id,
    title: input.summary,
    status: input.status ?? project.statuses[0] ?? "To Do",
    type: input.type,
    priority: input.priority,
    path: "",
  };
  return repo.create(project, task, "");
}