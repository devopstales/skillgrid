import matter from "gray-matter";
import yaml from "js-yaml";
import { Task } from "../domain/task";

export interface ParsedTask {
  task: Task;
  body: string;
  raw: string;
}

/** The frontmatter block between the leading `---` lines, or null if the
 *  content has none. Used for deterministic pre-validation: gray-matter
 *  4.0.3's tokenizer carries global regex state, so the SECOND parse of a
 *  malformed frontmatter does not throw — it returns empty data and the
 *  file surfaces as a ghost task (and a later write would clobber it).
 *  js-yaml's load is stateless, so validating the block first makes a
 *  malformed file throw on every parse. */
function frontMatterBlock(content: string): string | null {
  let start = 0;
  if (content.startsWith("---\n")) start = 4;
  else if (content.startsWith("---\r\n")) start = 5;
  else return null;
  const m = content.slice(start).match(/\r?\n---(\r?\n|$)/);
  if (!m || m.index === undefined) return null;
  return content.slice(start, start + m.index);
}

export function parseTask(path: string, content: string): ParsedTask {
  const block = frontMatterBlock(content);
  if (block !== null) yaml.load(block); // throws on malformed YAML (stateless)
  const parsed = matter(content);
  const data = parsed.data as Record<string, unknown>;
  const task: Task = {
    id: String(data.id ?? ""),
    title: String(data.title ?? ""),
    status: String(data.status ?? ""),
    type: data.type as string | undefined,
    priority: data.priority as string | undefined,
    labels: data.labels as string[] | undefined,
    assignee: data.assignee as string[] | null | undefined,
    milestone: data.milestone as string | undefined,
    sprint: data.sprint as string | undefined,
    created_date: data.created_date as string | undefined,
    updated_date: data.updated_date as string | undefined,
    dependencies: data.dependencies as string[] | undefined,
    references: data.references as string[] | undefined,
    documentation: data.documentation as string[] | undefined,
    parent_task_id: data.parent_task_id as string | undefined,
    ordinal: typeof data.ordinal === "number" ? data.ordinal : undefined,
    raw: data,
    body: parsed.content,
    path,
  };
  return { task, body: parsed.content, raw: content };
}

const KNOWN_ORDER = [
  "id",
  "title",
  "status",
  "type",
  "priority",
  "labels",
  "assignee",
  "milestone",
  "sprint",
  "created_date",
  "updated_date",
  "dependencies",
  "references",
  "documentation",
  "parent_task_id",
  "ordinal",
];

/** Lossless round-trip serializer. The original frontmatter (task.raw,
 *  insertion order = file order) is the base: known fields update in place,
 *  fields patched to undefined are deleted, unknown fields (points, rank,
 *  custom…) survive at their original positions, new fields append.
 *  A task without raw (never parsed from a file) serializes known fields
 *  only, in canonical order. The body passes through byte-identical. */
export function serializeTask(task: Task, body: string): string {
  const data: Record<string, unknown> = task.raw ? { ...task.raw } : {};
  for (const k of KNOWN_ORDER) {
    const v = (task as unknown as Record<string, unknown>)[k];
    if (v === undefined) delete data[k];
    else data[k] = v;
  }
  return matter.stringify(body, data);
}
