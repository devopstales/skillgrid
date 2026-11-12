import { ProjectConfig, TaskRepository } from "./ports";
import { Task } from "../domain/task";

export async function moveTask(
  repo: TaskRepository,
  project: ProjectConfig,
  task: Task,
  newStatus: string,
): Promise<void> {
  const updated: Task = { ...task, status: newStatus, updated_date: new Date().toISOString().slice(0, 10) };
  await repo.write(project, updated, task.body ?? "");
}