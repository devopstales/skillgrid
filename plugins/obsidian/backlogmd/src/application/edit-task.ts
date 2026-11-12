import { ProjectConfig, TaskRepository } from "./ports";
import { Task } from "../domain/task";

export async function editTask(
  repo: TaskRepository,
  project: ProjectConfig,
  task: Task,
  patch: Partial<Task>,
  body?: string,
): Promise<void> {
  const updated: Task = { ...task, ...patch, updated_date: new Date().toISOString().slice(0, 10) };
  await repo.write(project, updated, body ?? task.body ?? "");
}