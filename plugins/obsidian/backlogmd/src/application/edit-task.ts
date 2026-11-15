import { ProjectConfig, TaskRepository } from "./ports";
import { Task } from "../domain/task";
import { today } from "../domain/date";

export async function editTask(
  repo: TaskRepository,
  project: ProjectConfig,
  task: Task,
  patch: Partial<Task>,
  body?: string,
): Promise<void> {
  const updated: Task = { ...task, ...patch, updated_date: today() };
  await repo.write(project, updated, body ?? task.body ?? "");
}