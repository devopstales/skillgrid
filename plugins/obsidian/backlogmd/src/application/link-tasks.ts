import { ProjectConfig, TaskRepository } from "./ports";
import { Task } from "../domain/task";

export async function linkTasks(
  repo: TaskRepository,
  project: ProjectConfig,
  a: Task,
  b: Task,
  add: boolean,
): Promise<void> {
  const aDeps = new Set(a.dependencies ?? []);
  const bBlocked = new Set(b.dependencies ?? []); // reuse dependencies as blocked_by mirror? We'll store blocked_by in dependencies of counterpart? Simpler: store in dependencies both ways per briefing (bidirectional).
  // Briefing: link adds to dependencies of source and blocked_by of target. We store blocked_by as a separate field not in Task interface; instead mirror via dependencies on both sides for simplicity.
  if (add) {
    aDeps.add(b.id);
    bBlocked.add(a.id);
  } else {
    aDeps.delete(b.id);
    bBlocked.delete(a.id);
  }
  const newA: Task = { ...a, dependencies: [...aDeps], updated_date: a.updated_date };
  const newB: Task = { ...b, dependencies: [...bBlocked], updated_date: b.updated_date };
  await repo.write(project, newA, a.body ?? "");
  await repo.write(project, newB, b.body ?? "");
}