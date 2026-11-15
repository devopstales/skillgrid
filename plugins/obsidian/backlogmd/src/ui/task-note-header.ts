import { App, MarkdownView, Notice, TFile, setIcon } from "obsidian";
import { PluginContext } from "../compose";
import { ProjectConfig } from "../application/ports";
import { TaskModal } from "./task-modal";

const MARKER = "backlogmd-edit-task";

export function projectForPath(projects: ProjectConfig[], path: string): ProjectConfig | null {
  for (const project of projects) {
    if (path.startsWith(`${project.vaultBacklogDir}/tasks/`)) return project;
  }
  return null;
}

/** Open the edit modal for the task whose file is `file`, resolving the owning
 *  project and the task's current state from the repo. Resolves true when a
 *  task was found and the modal opened; false otherwise. Shared by the note
 *  header button and the "Edit task" command (H2). */
export async function openTaskModalForFile(app: App, ctx: PluginContext, file: TFile): Promise<boolean> {
  const project = projectForPath(ctx.projects, file.path);
  if (!project) return false;
  const all = await ctx.repo.list(project);
  // Task lookup is by file path, not by id parsed from the file name —
  // frontmatter ids differ from file names in case.
  const task = all.find((x) => x.path === file.path);
  if (!task) return false;
  new TaskModal(app, ctx, project, task, all).open();
  return true;
}

/** Keeps an "Edit task" header button on every markdown view showing a task
 *  note (opens the edit modal for it) and removes it from all others. Call on
 *  file-open and layout-ready. */
export function syncTaskHeaderButtons(app: App, ctx: PluginContext): void {
  for (const leaf of app.workspace.getLeavesOfType("markdown")) {
    const view = leaf.view as MarkdownView;
    // headerEl is a MarkdownView runtime property missing from the obsidian type defs
    const headerHost = view as MarkdownView & { headerEl?: HTMLElement };
    const headerEl = headerHost.headerEl;
    if (!headerEl) continue;
    headerEl.querySelectorAll(`.${MARKER}`).forEach((n) => n.remove());

    const file = view.file;
    if (!(file instanceof TFile)) continue;
    if (!projectForPath(ctx.projects, file.path)) continue;

    const btn = headerEl.createEl("div", { cls: `clickable-icon ${MARKER}`, title: "Edit task" });
    setIcon(btn, "pencil");
    btn.addEventListener("click", () => {
      void openTaskModalForFile(app, ctx, file).then((opened) => {
        if (!opened) new Notice("No task in active note");
      });
    });
  }
}
