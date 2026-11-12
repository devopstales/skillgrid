import { ItemView, WorkspaceLeaf, TFile } from "obsidian";
import { PluginContext } from "../compose";
import { ProjectConfig, TaskRepository } from "../application/ports";
import { Task } from "../domain/task";
import { isBlocked } from "../domain/task";
import type { Milestone } from "../domain/milestone";
import { daysLeft } from "../domain/milestone";
import { moveTask, reorderColumn } from "../application/move-task";
import { renderProjectSwitcher } from "./project-switcher";
import { TaskModal } from "./task-modal";

export const KANBAN_VIEW_TYPE = "backlogmd-kanban";

const byOrdinal = (a: Task, b: Task): number =>
  (a.ordinal ?? Number.MAX_SAFE_INTEGER) - (b.ordinal ?? Number.MAX_SAFE_INTEGER) ||
  (a.created_date ?? "").localeCompare(b.created_date ?? "") ||
  a.id.localeCompare(b.id);

export class KanbanView extends ItemView {
  private ctx: PluginContext;
  projectIndex = 0;
  private filter = "";
  private boardEl: HTMLElement | null = null;
  private boardHeaderEl: HTMLElement | null = null;
  private indicatorEl: HTMLElement | null = null;
  private renderSeq = 0;

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

  async onClose(): Promise<void> {
    this.removeIndicator();
  }

  async render(): Promise<void> {
    try {
      await this.renderInner();
    } catch (e) {
      console.error("[backlogmd] kanban render failed:", (e as Error).stack ?? String(e));
      throw e;
    }
  }

  private renderBoardHeader(project: ProjectConfig, active: Milestone | null, boardTasks: Task[]): void {
    const header = this.boardHeaderEl;
    if (!header) return;
    header.empty();
    if (active) {
      header.createEl("span", { text: "active", cls: "backlogmd-chip backlogmd-chip-active" });
      header.createEl("span", { text: `${active.id} ${active.title}`, cls: "backlogmd-section-title" });
      const bits: string[] = [];
      if (active.end_date) {
        const dl = daysLeft(active);
        if (dl !== null) bits.push(dl >= 0 ? `ends ${active.end_date} · ${dl}d left` : `overdue by ${-dl}d`);
      }
      const done = boardTasks.filter((t) => project.doneStatuses.includes(t.status)).length;
      bits.push(`${done}/${boardTasks.length} done`);
      header.createEl("span", { text: bits.join(" · "), cls: "backlogmd-section-meta" });
    } else {
      header.createEl("span", { text: "All tasks", cls: "backlogmd-section-title" });
      header.createEl("span", { text: `${boardTasks.length} · no active milestone`, cls: "backlogmd-section-meta" });
    }
  }

  private async renderInner(): Promise<void> {
    const container = this.containerEl.children[1] as HTMLElement;
    container.empty();
    this.boardEl = null;

    const project = this.ctx.projects[this.projectIndex];
    if (!project) {
      container.createEl("div", { text: "No backlog project found.", cls: "backlogmd-empty" });
      return;
    }

    const toolbar = container.createDiv({ cls: "backlogmd-toolbar" });
    renderProjectSwitcher(toolbar, this.ctx.projects, this.projectIndex, (i) => {
      this.projectIndex = i;
      void this.render();
    });
    const filter = toolbar.createEl("input", { type: "text", cls: "backlogmd-filter", placeholder: "Filter by id, title, type, assignee, label" });
    filter.value = this.filter;
    filter.addEventListener("input", () => {
      this.filter = filter.value;
      void this.renderBoard(project);
    });
    const newBtn = toolbar.createEl("button", { text: "+ New task", cls: "backlogmd-new-task" });
    newBtn.addEventListener("click", () => this.openModal(project, null));

    this.boardHeaderEl = container.createDiv({ cls: "backlogmd-board-header" });
    this.boardEl = container.createDiv({ cls: "backlogmd-board" });
    await this.renderBoard(project);
  }

  private async renderBoard(project: ProjectConfig): Promise<void> {
    const seq = ++this.renderSeq;
    const board = this.boardEl;
    if (!board) return;
    board.empty();
    this.removeIndicator();

    const repo = this.ctx.repo;
    const tasks = await repo.list(project);
    const milestones = await this.ctx.milestones.list(project);
    if (seq !== this.renderSeq) return; // a newer render owns the DOM now
    const active = milestones.find((m) => m.state === "active") ?? null;
    const boardTasks = active ? tasks.filter((t) => t.milestone === active.id) : tasks;
    this.renderBoardHeader(project, active, boardTasks);
    const byId = new Map(tasks.map((t) => [t.id, t]));
    const q = this.filter.trim().toLowerCase();
    const matches = (t: Task): boolean =>
      !q ||
      t.id.toLowerCase().includes(q) ||
      t.title.toLowerCase().includes(q) ||
      (t.type ?? "").toLowerCase().includes(q) ||
      (t.assignee ?? []).join(" ").toLowerCase().includes(q) ||
      (t.labels ?? []).join(" ").toLowerCase().includes(q) ||
      (t.milestone ?? "").toLowerCase().includes(q);

    for (const status of project.statuses) {
      const allInCol = boardTasks.filter((t) => t.status === status).sort(byOrdinal);
      const visible = allInCol.filter(matches);

      const col = board.createDiv({ cls: "backlogmd-column" });
      const header = col.createDiv({ cls: "backlogmd-column-header" });
      header.createEl("h3", { text: status, cls: "backlogmd-column-title" });
      header.createEl("span", { text: String(visible.length), cls: "backlogmd-count" });

      const listEl = col.createDiv({ cls: "backlogmd-cards" });
      for (const task of visible) {
        listEl.appendChild(this.renderCard(project, task, byId));
      }

      listEl.addEventListener("dragover", (e) => {
        e.preventDefault();
        this.showIndicator(listEl, e.clientY, e.dataTransfer?.getData("text/plain") ?? null);
      });
      listEl.addEventListener("drop", async (e) => {
        e.preventDefault();
        this.removeIndicator();
        const id = e.dataTransfer?.getData("text/plain");
        if (!id) return;
        await this.handleDrop(repo, project, allInCol, status, id, e.clientY, listEl);
      });
    }
  }

  private renderCard(project: ProjectConfig, task: Task, byId: Map<string, Task>): HTMLElement {
    // window.createEl (global), not document.createDiv — Node.createEl
    // appends to its receiver, and document cannot take element children.
    const card = createEl("div", { cls: "backlogmd-card" });
    card.dataset.id = task.id;
    const blocked = isBlocked(task, byId, project.doneStatuses);
    if (blocked) card.addClass("blocked");

    const top = card.createDiv({ cls: "backlogmd-card-top" });
    top.createEl("span", { text: task.id, cls: "backlogmd-card-id" });
    if (task.type) top.createEl("span", { text: task.type, cls: "backlogmd-badge" });
    if (task.priority) top.createEl("span", { text: task.priority, cls: `backlogmd-badge backlogmd-p-${task.priority}` });
    if (blocked) top.createEl("span", { text: "⛔", cls: "backlogmd-blocked", title: "Blocked by unfinished dependencies" });

    card.createEl("div", { text: task.title, cls: "backlogmd-card-title" });

    const meta = card.createDiv({ cls: "backlogmd-card-meta" });
    if (task.milestone) meta.createEl("span", { text: task.milestone, cls: "backlogmd-milestone" });
    if (task.assignee?.length) meta.createEl("span", { text: task.assignee.join(", "), cls: "backlogmd-assignee" });
    if (task.labels?.length) meta.createEl("span", { text: task.labels.map((l) => `#${l}`).join(" "), cls: "backlogmd-labels" });
    if (task.dependencies?.length) meta.createEl("span", { text: task.dependencies.join(", "), cls: "backlogmd-deps" });

    card.setAttribute("draggable", "true");
    card.addEventListener("dragstart", (e) => {
      e.dataTransfer?.setData("text/plain", task.id);
      card.addClass("dragging");
    });
    card.addEventListener("dragend", () => card.removeClass("dragging"));
    card.addEventListener("click", (e) => {
      if (e.metaKey || e.ctrlKey) {
        const file = this.app.vault.getAbstractFileByPath(task.path);
        if (file instanceof TFile) this.app.workspace.getLeaf(true).openFile(file);
      } else {
        this.openModal(project, task);
      }
    });
    return card;
  }

  private openModal(project: ProjectConfig, task: Task | null): void {
    void Promise.all([this.ctx.repo.list(project), this.ctx.milestones.list(project)]).then(([all, milestones]) => {
      const active = milestones.find((m) => m.state === "active");
      const modal = new TaskModal(this.app, this.ctx, project, task, all, active?.id);
      const origClose = modal.onClose.bind(modal);
      modal.onClose = () => {
        origClose();
        void this.renderBoard(project);
      };
      modal.open();
    });
  }

  /** Index where a card dropped at clientY would land, excluding the dragged card. */
  private dropIndex(listEl: HTMLElement, clientY: number, draggedId: string | null): number {
    const cards = Array.from(listEl.querySelectorAll<HTMLElement>(".backlogmd-card")).filter((c) => c.dataset.id !== draggedId);
    for (let i = 0; i < cards.length; i++) {
      const r = cards[i].getBoundingClientRect();
      if (clientY < r.top + r.height / 2) return i;
    }
    return cards.length;
  }

  private showIndicator(listEl: HTMLElement, clientY: number, draggedId: string | null): void {
    this.removeIndicator();
    const index = this.dropIndex(listEl, clientY, draggedId);
    const line = createEl("div", { cls: "backlogmd-drop-line" });
    this.indicatorEl = line;
    const cards = Array.from(listEl.querySelectorAll<HTMLElement>(".backlogmd-card")).filter((c) => c.dataset.id !== draggedId);
    if (index >= cards.length) listEl.appendChild(line);
    else listEl.insertBefore(line, cards[index]);
  }

  private removeIndicator(): void {
    this.indicatorEl?.remove();
    this.indicatorEl = null;
  }

  private async handleDrop(
    repo: TaskRepository,
    project: ProjectConfig,
    allInCol: Task[],
    status: string,
    id: string,
    clientY: number,
    listEl: HTMLElement,
  ): Promise<void> {
    const dragged =
      allInCol.find((t) => t.id === id) ?? (await repo.list(project)).find((t) => t.id === id);
    if (!dragged) return;

    if (dragged.status === status) {
      // Reorder within the column. With a filter active the visible list is a
      // subset, so reorder only when the full column is visible.
      if (this.filter.trim()) return;
      const index = this.dropIndex(listEl, clientY, id);
      const from = allInCol.findIndex((t) => t.id === id);
      if (from === -1) return;
      const ordered = [...allInCol];
      ordered.splice(from, 1);
      ordered.splice(index, 0, dragged);
      if (ordered.every((t, i) => t === allInCol[i])) return; // no-op drop
      await reorderColumn(repo, project, ordered);
    } else {
      const ordinal = allInCol.length ? Math.max(...allInCol.map((t) => t.ordinal ?? 0)) + 1000 : 1000;
      await moveTask(repo, project, dragged, status, ordinal);
    }
    await this.renderBoard(project);
  }
}
