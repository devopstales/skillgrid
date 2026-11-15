import { ProjectConfig, TaskRepository } from "./ports";
import { Task } from "../domain/task";
import { today } from "../domain/date";

export async function moveTask(
  repo: TaskRepository,
  project: ProjectConfig,
  task: Task,
  newStatus: string,
  ordinal?: number,
): Promise<void> {
  const updated: Task = { ...task, status: newStatus, ordinal, updated_date: today() };
  await repo.write(project, updated, task.body ?? "");
}

/** Rewrite the column's tasks in the given order, reassigning 1000-step ordinals. */
export async function reorderColumn(
  repo: TaskRepository,
  project: ProjectConfig,
  ordered: Task[],
): Promise<void> {
  for (let i = 0; i < ordered.length; i++) {
    const updated: Task = { ...ordered[i], ordinal: i * 1000 + 1000 };
    await repo.write(project, updated, ordered[i].body ?? "");
  }
}
