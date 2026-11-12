import { ItemView, WorkspaceLeaf, TFile } from "obsidian";
import { PluginContext } from "../compose";
import { Task } from "../domain/task";
import { isBlocked } from "../domain/task";
import { moveTask } from "../application/move-task";

export const KANBAN_VIEW_TYPE = "backlogmd-kanban";

export class KanbanView extends ItemView {
  private ctx: PluginContext;
  private projectIndex = 0;

  constructor(leaf: WorkspaceLeaf, ctx: PluginContext) {
    super(leaf);
    this.ctx = ctx;
  }

  getViewType(): string {
    return KANBAN_VIEW_TYPE;
  }
  getDisplayText(): string {
    return "Backlog Kanban";
  }

  async onOpen(): Promise<void> {
    await this.render();
  }

  async render(): Promise<void> {
    const container = this.containerEl.children[1] as HTMLElement;
    container.empty();
    container.addClass("backlogmd-board");

    const project = this.ctx.projects[this.projectIndex];
    if (!project) {
      container.createEl("div", { text: "No backlog project found.", cls: "backlogmd-empty" });
      return;
    }

    const repo = this.ctx.repo;
    const tasks = await repo.list(project);
    const byId = new Map(tasks.map((t) => [t.id, t]));
    const doneStatuses = project.doneStatuses;

    for (const status of project.statuses) {
      const col = container.createDiv({ cls: "backlogmd-column" });
      col.createEl("h3", { text: status });
      const cards = tasks.filter((t) => t.status === status);
      for (const task of cards) {
        const card = col.createDiv({ cls: "backlogmd-card" });
        if (isBlocked(task, byId, doneStatuses)) card.addClass("blocked");
        card.createEl("div", { text: task.title });
        card.createEl("small", { text: task.id });
        if (isBlocked(task, byId, doneStatuses)) card.createEl("span", { text: " ⚠" });

        card.setAttribute("draggable", "true");
        card.addEventListener("dragstart", (e) => {
          e.dataTransfer?.setData("text/plain", task.id);
        });
        card.addEventListener("click", (e) => {
          if (e.metaKey || e.ctrlKey) {
            const file = this.app.vault.getAbstractFileByPath(task.path);
            if (file instanceof TFile) this.app.workspace.getLeaf(true).openFile(file);
          }
        });

        col.addEventListener("dragover", (e) => e.preventDefault());
        col.addEventListener("drop", async (e) => {
          e.preventDefault();
          const id = e.dataTransfer?.getData("text/plain");
          if (!id) return;
          const t = tasks.find((x) => x.id === id);
          if (!t || t.status === status) return;
          await moveTask(repo, project, t, status);
          await this.render();
        });
      }
    }
  }
}