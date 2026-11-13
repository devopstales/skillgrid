import { ProjectConfig, TaskRepository } from "./ports";
import { Task } from "../domain/task";
import { fetchIssues } from "../infrastructure/github";

/** Next free `<prefix>-NNN` id: highest existing numeric suffix (any case,
 * this or related ids) + 1, zero-padded to 3 digits. */
export function nextTaskId(existing: Task[], prefix: string): string {
  const re = new RegExp(`^${prefix.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}[-_ ]?(\\d+)$`, "i");
  let max = 0;
  for (const t of existing) {
    const m = t.id.match(re);
    if (m) max = Math.max(max, parseInt(m[1], 10));
  }
  return `${prefix}-${String(max + 1).padStart(3, "0")}`;
}

export async function pullIssues(
  repo: TaskRepository,
  project: ProjectConfig,
  token: string,
  owner: string,
  repository: string,
): Promise<void> {
  const issues = await fetchIssues(token, owner, repository);
  const existing = [...(await repo.list(project))]; // working copy: repos may alias

  for (const issue of issues) {
    const sourceRef = `github:${owner}/${repository}#${issue.number}`;
    const match = existing.find((t) => t.references?.includes(sourceRef));
    if (match) {
      // sync title, assignee, open/closed state only — never overwrite
      // sprint/rank/points/priority/parent/links/description
      const closed = issue.state === "closed";
      const newStatus = closed ? (project.doneStatuses[0] ?? match.status) : match.status;
      await repo.write(project, { ...match, title: issue.title, assignee: issue.assignee ? [issue.assignee] : match.assignee, status: newStatus }, match.body ?? "");
    } else {
      const id = nextTaskId(existing, project.taskPrefix);
      const task: Task = {
        id,
        title: issue.title,
        status: issue.state === "closed" ? (project.doneStatuses[0] ?? project.statuses[0]) : project.statuses[0] ?? "To Do",
        assignee: issue.assignee ? [issue.assignee] : undefined,
        references: [sourceRef],
        body: issue.body ? `\`\`\`\n${issue.body}\n\`\`\`` : "",
        path: "",
      };
      await repo.create(project, task, task.body ?? "");
      existing.push(task);
    }
  }
}
