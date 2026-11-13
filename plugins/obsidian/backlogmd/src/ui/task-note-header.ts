import { App, MarkdownView, TFile, setIcon } from "obsidian";
import { PluginContext } from "../compose";
import { ProjectConfig } from "../application/ports";
import { TaskModal } from "./task-modal";

const MARKER = "backlogmd-edit-task";

function projectForPath(projects: ProjectConfig[], path: string): ProjectConfig | null {
  for (const project of projects) {
    if (path.startsWith(`${project.vaultBacklogDir}/tasks/`)) return project;
  }
  return null;
}

/** Keeps an "Edit task" header button on every markdown view showing a task
 *  note (opens the edit modal for it) and removes it from all others. Call on
 *  file-open and layout-ready. Task lookup is by file path, not by id parsed
 *  from the file name — frontmatter ids differ from file names in case. */
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
    const project = projectForPath(ctx.projects, file.path);
    if (!project) continue;

    const btn = headerEl.createEl("div", { cls: `clickable-icon ${MARKER}`, title: "Edit task" });
    setIcon(btn, "pencil");
    btn.addEventListener("click", () => {
      void ctx.repo.list(project).then((all) => {
        const t = all.find((x) => x.path === file.path);
        if (t) new TaskModal(app, ctx, project, t, all).open();
      });
    });
  }
}
