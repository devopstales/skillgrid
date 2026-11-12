import matter from "gray-matter";
import { Task } from "../domain/task";

export interface ParsedTask {
  task: Task;
  body: string;
  raw: string;
}

export function parseTask(path: string, content: string): ParsedTask {
  const parsed = matter(content);
  const data = parsed.data as Record<string, unknown>;
  const task: Task = {
    id: String(data.id ?? ""),
    title: String(data.title ?? ""),
    status: String(data.status ?? ""),
    type: data.type as string | undefined,
    priority: data.priority as string | undefined,
    labels: data.labels as string[] | undefined,
    assignee: data.assignee as string[] | undefined,
    milestone: data.milestone as string | undefined,
    sprint: data.sprint as string | undefined,
    created_date: data.created_date as string | undefined,
    updated_date: data.updated_date as string | undefined,
    dependencies: data.dependencies as string[] | undefined,
    references: data.references as string[] | undefined,
    documentation: data.documentation as string[] | undefined,
    parent_task_id: data.parent_task_id as string | undefined,
    body: parsed.content,
    path,
  };
  return { task, body: parsed.content, raw: content };
}

export function serializeTask(task: Task, body: string): string {
  const data: Record<string, unknown> = {};
  // preserve known fields in a stable order
  const order = ["id", "title", "status", "type", "priority", "labels", "assignee", "milestone", "sprint", "created_date", "updated_date", "dependencies", "references", "documentation", "parent_task_id"];
  for (const k of order) {
    const v = (task as unknown as Record<string, unknown>)[k];
    if (v !== undefined) data[k] = v;
  }
  return matter.stringify(body, data);
}

export function updateFrontmatter(content: string, patch: Record<string, unknown>): string {
  const parsed = matter(content);
  const data = { ...parsed.data, ...patch };
  return matter.stringify(parsed.content, data);
}