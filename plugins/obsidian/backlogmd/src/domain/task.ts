export interface Task {
  id: string;
  title: string;
  status: string;
  type?: string;
  priority?: string;
  labels?: string[];
  assignee?: string[] | null;
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
  /** Original full frontmatter data (insertion order = file order), set by
   *  parseTask. Never read by the UI; used only by serializeTask for a
   *  lossless round-trip (unknown fields keep their position, known fields
   *  update in place, fields patched to undefined are deleted). */
  raw?: Record<string, unknown>;
  body?: string;
  path: string;
}

/** Next task id: max numeric suffix of existing ids whose prefix matches
 *  (case-insensitive, `^<prefix>[-_ ]?(\d+)$`) + 1, uppercased prefix,
 *  zero-padded to 3 digits — the Backlog.md CLI's `TASK-NNN` convention. */
export function nextTaskId(existing: Task[], prefix: string): string {
  const re = new RegExp(`^${prefix.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}[-_ ]?(\\d+)$`, "i");
  let max = 0;
  for (const t of existing) {
    const m = t.id.match(re);
    if (m) max = Math.max(max, parseInt(m[1], 10));
  }
  return `${prefix.toUpperCase()}-${String(max + 1).padStart(3, "0")}`;
}

export function isBlocked(task: Task, tasksById: Map<string, Task>, doneStatuses: string[]): boolean {
  const deps = task.dependencies ?? [];
  return deps.some((dep) => {
    const d = tasksById.get(dep);
    if (!d) return false;
    return !doneStatuses.includes(d.status);
  });
}
