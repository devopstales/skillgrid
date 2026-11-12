export interface Task {
  id: string;
  title: string;
  status: string;
  type?: string;
  priority?: string;
  labels?: string[];
  assignee?: string[];
  milestone?: string;
  sprint?: string;
  created_date?: string;
  updated_date?: string;
  dependencies?: string[];
  references?: string[];
  documentation?: string[];
  parent_task_id?: string;
  /** Position within a column (Backlog.md `ordinal`, 1000-step convention). */
  ordinal?: number;
  body?: string;
  path: string;
}

export function isDone(task: Task, doneStatuses: string[]): boolean {
  return doneStatuses.includes(task.status);
}

export function isBlocked(task: Task, tasksById: Map<string, Task>, doneStatuses: string[]): boolean {
  const deps = task.dependencies ?? [];
  return deps.some((dep) => {
    const d = tasksById.get(dep);
    if (!d) return false;
    return !doneStatuses.includes(d.status);
  });
}