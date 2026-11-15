import type { ProjectConfig, TaskRepository, MilestoneRepository } from "./ports";
import type { Milestone } from "../domain/milestone";
import { today } from "../domain/date";
import type { Task } from "../domain/task";

/** Next milestone id: max numeric suffix of existing + 1 (`m-7`). */
export function nextMilestoneId(existing: Milestone[]): string {
  let max = 0;
  for (const m of existing) {
    const n = parseInt(m.id.replace(/[^0-9]/g, ""), 10);
    if (!Number.isNaN(n)) max = Math.max(max, n);
  }
  return `m-${max + 1}`;
}

function wrap(ms: string, err: unknown): Error {
  return new Error(
    `backlogmd: writing milestone ${ms} failed — on-disk state unchanged: ${err instanceof Error ? err.message : String(err)}`,
    { cause: err },
  );
}

/** Mark a milestone active. Any other explicitly active milestone is
 *  completed first (with its carry-over), keeping a single active sprint.
 *  The milestone file write is last; if any write throws, the error is
 *  rethrown descriptively and the milestone is not marked started. */
export async function startMilestone(
  tasks: TaskRepository,
  milestones: MilestoneRepository,
  project: ProjectConfig,
  ms: Milestone,
): Promise<void> {
  const all = await milestones.list(project);
  const taskList = await tasks.list(project);
  for (const other of all) {
    if (other.id !== ms.id && other.state === "active") {
      await completeMilestone(tasks, milestones, project, other, taskList);
    }
  }
  const updated: Milestone = { ...ms, state: "active", start_date: ms.start_date ?? today() };
  try {
    await milestones.write(project, updated, ms.body ?? "");
  } catch (err) {
    throw wrap(`start(${ms.id})`, err);
  }
}

/** Mark a milestone completed: record which tasks finished, carry the
 *  unfinished ones back to the unplanned backlog (their `milestone` is
 *  cleared), so the CLI's derive-from-tasks view stays consistent. Carry-over
 *  task writes go first; the milestone file write is last, and any failing
 *  write rethrows a descriptive error (state not marked complete). */
export async function completeMilestone(
  tasks: TaskRepository,
  milestones: MilestoneRepository,
  project: ProjectConfig,
  ms: Milestone,
  taskList?: Task[],
): Promise<void> {
  const all = taskList ?? (await tasks.list(project));
  const done = new Set(project.doneStatuses);
  const msTasks = all.filter((t) => t.milestone === ms.id);
  const completedTasks = msTasks.filter((t) => done.has(t.status)).map((t) => t.id);
  const carriedOver = msTasks.filter((t) => !done.has(t.status));
  const updated: Milestone = {
    ...ms,
    state: "completed",
    end_date: ms.end_date ?? today(),
    completed_tasks: completedTasks,
    carried_over_tasks: carriedOver.map((t) => t.id),
  };
  try {
    for (const t of carriedOver) {
      const tUpdated: Task = { ...t, milestone: undefined, updated_date: today() };
      await tasks.write(project, tUpdated, t.body ?? "");
    }
    await milestones.write(project, updated, ms.body ?? "");
  } catch (err) {
    throw wrap(`complete(${ms.id})`, err);
  }
}

/** Plan a task into a milestone (or clear its plan with null). Planning
 *  appends the task to the end of the target section's ordinal range. */
export async function planTask(
  tasks: TaskRepository,
  project: ProjectConfig,
  task: Task,
  milestoneId: string | null,
  taskList?: Task[],
): Promise<void> {
  let ordinal = task.ordinal;
  if (milestoneId) {
    const all = taskList ?? (await tasks.list(project));
    const peers = all.filter((t) => t.milestone === milestoneId);
    ordinal = (peers.length ? Math.max(...peers.map((p) => p.ordinal ?? 0)) : 0) + 1000;
  }
  const updated: Task = { ...task, milestone: milestoneId ?? undefined, ordinal, updated_date: today() };
  await tasks.write(project, updated, task.body ?? "");
}
