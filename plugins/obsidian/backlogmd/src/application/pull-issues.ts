import { ProjectConfig, TaskRepository } from "./ports";
import { Task } from "../domain/task";
import { fetchIssues, GhIssue } from "../infrastructure/github";

export async function pullIssues(
  repo: TaskRepository,
  project: ProjectConfig,
  token: string,
  owner: string,
  repository: string,
): Promise<void> {
  const issues = await fetchIssues(token, owner, repository);
  const existing = await repo.list(project);
  const bySource = new Map(existing.filter((t) => t.references?.some((r) => r.includes("github:"))).map((t) => [t.id, t]));

  for (const issue of issues) {
    const sourceRef = `github:${owner}/${repository}#${issue.number}`;
    const match = existing.find((t) => t.references?.includes(sourceRef));
    if (match) {
      // sync title, assignee, open/closed state only
      const newStatus = issue.state === "closed" ? (project.doneStatuses[0] ?? match.status) : match.status;
      // don't overwrite sprint/rank/points/priority/parent/links/description
      await repo.write(project, { ...match, title: issue.title, assignee: issue.assignee ? [issue.assignee] : match.assignee, status: issue.state === "closed" ? (project.doneStatuses[0] ?? match.status) : match.status }, match.body ?? "");
    } else {
      const id = `${project.taskPrefix.toUpperCase()}-${existing.length + 1}`;
      const task: Task = {
        id,
        title: issue.title,
        status: issue.state === "closed" ? (project.doneStatuses[0] ?? project.statuses[0]) : (project.statuses[0] ?? "To Do"),
        assignee: issue.assignee ? [issue.assignee] : undefined,
        references: [sourceRef],
        body: issue.body ? `\`\`\`\n${issue.body}\n\`\`\`` : "",
        path: "",
      };
      await repo.create(project, task, task.body ?? "");
    }
  }
}