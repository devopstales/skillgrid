import { ProjectConfig, TaskRepository } from "./ports";
import { Task, nextTaskId } from "../domain/task";
import { fetchIssues } from "../infrastructure/github";

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
      // sprint/rank/points/priority/parent/links/description.
      // Closed → first done status; open (incl. reopens) → first open
      // status, so a reopened issue leaves Done again.
      const closed = issue.state === "closed";
      const newStatus = closed
        ? (project.doneStatuses[0] ?? match.status)
        : (project.statuses[0] ?? match.status);
      // null assignee from GitHub clears a stale local assignee
      const newAssignee: string[] | null = issue.assignee ? [issue.assignee.login] : null;
      await repo.write(project, { ...match, title: issue.title, assignee: newAssignee, status: newStatus }, match.body ?? "");
    } else {
      const id = nextTaskId(existing, project.taskPrefix);
      const body = issue.body ? fence(issue.body) : "";
      const task: Task = {
        id,
        title: issue.title,
        status: issue.state === "closed" ? (project.doneStatuses[0] ?? project.statuses[0]) : project.statuses[0] ?? "To Do",
        assignee: issue.assignee ? [issue.assignee.login] : null,
        references: [sourceRef],
        body,
        path: "",
      };
      await repo.create(project, task, body);
      existing.push(task);
    }
  }
}

/** Wrap a pulled issue body in a code fence; escalate to 4 backticks when
 *  the body contains its own ``` fences so it renders intact. */
function fence(body: string): string {
  const f = body.includes("```") ? "````" : "```";
  return `${f}\n${body}\n${f}`;
}
