import { ProjectConfig, TaskRepository } from "./ports";
import { Task } from "../domain/task";
import { today } from "../domain/date";

/** Add/remove a bidirectional dependency edge: `a` depends on `b` and `b`
 *  mirrors it. Self-links are a no-op. Both sides get updated_date stamped. */
export async function linkTasks(
  repo: TaskRepository,
  project: ProjectConfig,
  a: Task,
  b: Task,
  add: boolean,
): Promise<void> {
  if (a.id === b.id) return;
  const aDeps = new Set(a.dependencies ?? []);
  const bDeps = new Set(b.dependencies ?? []);
  if (add) {
    aDeps.add(b.id);
    bDeps.add(a.id);
  } else {
    aDeps.delete(b.id);
    bDeps.delete(a.id);
  }
  const now = today();
  await repo.write(project, { ...a, dependencies: [...aDeps], updated_date: now }, a.body ?? "");
  await repo.write(project, { ...b, dependencies: [...bDeps], updated_date: now }, b.body ?? "");
}
