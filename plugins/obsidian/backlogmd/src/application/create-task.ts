import { ProjectConfig, TaskRepository } from "./ports";
import { Task, nextTaskId } from "../domain/task";
import { today } from "../domain/date";

export interface CreateTaskInput {
  summary: string;
  body?: string;
  type?: string;
  priority?: string;
  status?: string;
  assignee?: string[];
  milestone?: string;
  dependencies?: string[];
  /** Provided value wins; defaults to [] (the ticketing gate requires the
   *  field to be present). */
  references?: string[];
}

const DOD_SECTION = `## Definition of Done

<!-- DOD:BEGIN -->
- [ ] Tests pass
- [ ] Lint and formatting pass
- [ ] Edge cases covered
- [ ] No new warnings introduced
- [ ] Spec/docs updated if behavior changes
<!-- DOD:END -->`;

const PLAN_SECTION = `## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
- [ ] Research / confirm deps
- [ ] Implement
- [ ] Verify with tests; mark AC/DoD
<!-- SECTION:PLAN:END -->`;

/** Append the ticketing-gate body scaffolding (DoD + plan with unchecked
 *  boxes) when the body lacks it; a body that already carries the sections
 *  (heading or marker) is left untouched. */
function scaffoldBody(body: string): string {
  const hasDod = body.includes("## Definition of Done") || body.includes("DOD:BEGIN");
  const hasPlan = body.includes("## Implementation Plan") || body.includes("SECTION:PLAN:BEGIN");
  const sections = [hasDod ? null : DOD_SECTION, hasPlan ? null : PLAN_SECTION].filter((s): s is string => s !== null);
  if (sections.length === 0) return body;
  const base = body.replace(/\s+$/, "");
  return base ? `${base}\n\n${sections.join("\n\n")}\n` : `${sections.join("\n\n")}\n`;
}

export async function createTask(
  repo: TaskRepository,
  project: ProjectConfig,
  input: CreateTaskInput,
  existing: Task[],
): Promise<string> {
  const id = nextTaskId(existing, project.taskPrefix);
  const ordinal = existing.length ? Math.max(...existing.map((t) => t.ordinal ?? 0)) + 1000 : 1000;
  const task: Task = {
    id,
    title: input.summary,
    status: input.status ?? project.statuses[0] ?? "To Do",
    type: input.type || undefined,
    priority: input.priority || undefined,
    assignee: input.assignee?.length ? input.assignee : undefined,
    milestone: input.milestone || undefined,
    dependencies: input.dependencies?.length ? input.dependencies : undefined,
    // ticketing gate: references must be present on a new ticket
    references: input.references?.length ? input.references : [],
    created_date: today(),
    ordinal,
    path: "",
  };
  return repo.create(project, task, scaffoldBody(input.body ?? ""));
}
