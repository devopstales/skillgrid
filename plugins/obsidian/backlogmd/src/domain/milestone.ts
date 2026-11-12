import matter from "gray-matter";
import { Task } from "./task";

/** Backlog.md milestone file — in skillgrid the milestone plays boardmd's
 *  sprint role. `state`, dates, goal and the completion record are optional
 *  extensions the plugin writes; the CLI derives completion from task states
 *  and ignores the extra fields. */
export interface Milestone {
  id: string;
  title: string;
  state?: "planned" | "active" | "completed";
  start_date?: string;
  end_date?: string;
  goal?: string;
  completed_tasks?: string[];
  carried_over_tasks?: string[];
  body?: string;
  path: string;
}

export interface ParsedMilestone {
  milestone: Milestone;
  body: string;
}

export function parseMilestone(path: string, content: string): ParsedMilestone {
  const parsed = matter(content);
  const data = parsed.data as Record<string, unknown>;
  const milestone: Milestone = {
    id: String(data.id ?? ""),
    title: String(data.title ?? ""),
    state: data.state as Milestone["state"] | undefined,
    start_date: data.start_date as string | undefined,
    end_date: data.end_date as string | undefined,
    goal: data.goal as string | undefined,
    completed_tasks: data.completed_tasks as string[] | undefined,
    carried_over_tasks: data.carried_over_tasks as string[] | undefined,
    body: parsed.content,
    path,
  };
  return { milestone, body: parsed.content };
}

export function serializeMilestone(m: Milestone, body: string): string {
  const data: Record<string, unknown> = {};
  const order = ["id", "title", "state", "start_date", "end_date", "goal", "completed_tasks", "carried_over_tasks"];
  for (const k of order) {
    const v = (m as unknown as Record<string, unknown>)[k];
    if (v !== undefined) data[k] = v;
  }
  return matter.stringify(body, data);
}

/** Explicit state wins; otherwise derive the CLI's way: a milestone with
 *  tasks, all of them done, is completed. */
export function effectiveState(m: Milestone, tasks: Task[], doneStatuses: string[]): "planned" | "active" | "completed" {
  if (m.state) return m.state;
  const msTasks = tasks.filter((t) => t.milestone === m.id);
  if (msTasks.length > 0 && msTasks.every((t) => doneStatuses.includes(t.status))) return "completed";
  return "planned";
}

/** Whole calendar days until end_date (date-only math, no time-of-day bias);
 *  null when not set or unparseable, negative when past. */
export function daysLeft(m: Milestone): number | null {
  if (!m.end_date) return null;
  const [y, mo, d] = m.end_date.split("-").map(Number);
  if (!y || !mo || !d) return null;
  const end = Date.UTC(y, mo - 1, d);
  const now = new Date();
  const today = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate());
  return Math.round((end - today) / 86400000);
}

export function today(): string {
  return new Date().toISOString().slice(0, 10);
}
