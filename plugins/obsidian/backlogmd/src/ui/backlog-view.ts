import { ItemView, TFile, WorkspaceLeaf } from "obsidian";
import { PluginContext } from "../compose";
import type { ProjectConfig } from "../application/ports";
import type { Task } from "../domain/task";
import { isBlocked } from "../domain/task";
import type { Milestone } from "../domain/milestone";
import { effectiveState, daysLeft } from "../domain/milestone";
import { startMilestone, completeMilestone, planTask } from "../application/milestone";
import { reorderColumn } from "../application/move-task";
import { renderProjectSwitcher } from "./project-switcher";
import { TaskModal } from "./task-modal";
import { MilestoneModal } from "./milestone-modal";

export const BACKLOG_VIEW_TYPE = "backlogmd-backlog";

interface Section {
  milestoneId: string | null; // null = unplanned backlog
  milestone?: Milestone;
  tasks: Task[];
}

const byOrdinal = (a: Task, b: Task): number =>
  (a.ordinal ?? Number.MAX_SAFE_INTEGER) - (b.ordinal ?? Number.MAX_SAFE_INTEGER) ||
  (a.created_date ?? "").localeCompare(b.created_date ?? "") ||
  a.id.localeCompare(b.id);

const msNumber = (id: string): number => parseInt(id.replace(/[^0-9]/g, ""), 10) || 0;

export class BacklogView extends ItemView {
  private ctx: PluginContext;
  projectIndex = 0;
  private filter = "";
  private sectionsEl: HTMLElement | null = null;
  private indicatorEl: HTMLElement | null = null;
  private renderSeq = 0;

  constructor(leaf: WorkspaceLeaf, ctx: PluginContext) {
    super(leaf);
    this.ctx = ctx;
  }

  getViewType(): string {
    return BACKLOG_VIEW_TYPE;
  }
  getDisplayText(): string {
    return "Backlog List";
  }

  async onOpen(): Promise<void> {
    await this.render();
  }

  async onClose(): Promise<void> {
    this.removeIndicator();
  }

  async render(): Promise<void> {
    const container = this.containerEl.children[1] as HTMLElement;
    container.empty();
    this.sectionsEl = null;

    const project = this.ctx.projects[this.projectIndex];
    if (!project) {
      container.createEl("div", { text: "No project", cls: "backlogmd-empty" });
      return;
    }

    const toolbar = container.createDiv({ cls: "backlogmd-toolbar" });
    renderProjectSwitcher(toolbar, this.ctx.projects, this.projectIndex, (i) => {
      this.projectIndex = i;
      void this.render();
    });
    const input = toolbar.createEl("input", { type: "text", cls: "backlogmd-filter", placeholder: "Filter by id, title, type, assignee" });
    input.value = this.filter;
    input.addEventListener("input", () => {
      this.filter = input.value;
      void this.renderSections(project);
    });
    const newTask = toolbar.createEl("button", { text: "+ New task", cls: "backlogmd-new-task" });
    newTask.addEventListener("click", () => this.openTaskModal(project, null));
    const newMilestone = toolbar.createEl("button", { text: "+ Milestone", cls: "backlogmd-new-task" });
    newMilestone.addEventListener("click", () => {
      void this.ctx.milestones.list(project).then((all) => {
        const modal = new MilestoneModal(this.app, this.ctx, project, all);
        const origClose = modal.onClose.bind(modal);
        modal.onClose = () => {
          origClose();
          void this.renderSections(project);
        };
        modal.open();
      });
    });

    this.sectionsEl = container.createDiv({ cls: "backlogmd-sections" });
    await this.renderSections(project);
  }

  private buildSections(project: ProjectConfig, tasks: Task[], milestones: Milestone[]): Section[] {
    const done = new Set(project.doneStatuses);
    const active = milestones.find((m) => m.state === "active");
    const sections: Section[] = [];
    if (active) {
      sections.push({ milestoneId: active.id, milestone: active, tasks: tasks.filter((t) => t.milestone === active.id).sort(byOrdinal) });
    }
    const planned = milestones
      .filter((m) => effectiveState(m, tasks, [...done]) !== "completed" && m.id !== active?.id)
      .sort((a, b) => msNumber(a.id) - msNumber(b.id) || a.id.localeCompare(b.id));
    for (const m of planned) {
      sections.push({ milestoneId: m.id, milestone: m, tasks: tasks.filter((t) => t.milestone === m.id).sort(byOrdinal) });
    }
    sections.push({
      milestoneId: null,
      tasks: tasks.filter((t) => !t.milestone).sort(byOrdinal),
    });
    return sections;
  }

  private async renderSections(project: ProjectConfig): Promise<void> {
    const seq = ++this.renderSeq;
    const host = this.sectionsEl;
    if (!host) return;
    host.empty();
    this.removeIndicator();

    const tasks = await this.ctx.repo.list(project);
    const milestones = await this.ctx.milestones.list(project);
    if (seq !== this.renderSeq) return; // a newer render owns the DOM now
    const byId = new Map(tasks.map((t) => [t.id, t]));
    const q = this.filter.trim().toLowerCase();
    const matches = (t: Task): boolean =>
      !q ||
      t.id.toLowerCase().includes(q) ||
      t.title.toLowerCase().includes(q) ||
      (t.type ?? "").toLowerCase().includes(q) ||
      (t.assignee ?? []).join(" ").toLowerCase().includes(q) ||
      (t.labels ?? []).join(" ").toLowerCase().includes(q);

    for (const section of this.buildSections(project, tasks, milestones)) {
      const visible = section.tasks.filter(matches);
      const el = host.createDiv({ cls: "backlogmd-section" });

      const header = el.createDiv({ cls: "backlogmd-section-header" });
      if (section.milestone) {
        const state = section.milestone.state === "active" ? "active" : "planned";
        header.createEl("span", { text: state, cls: `backlogmd-chip ${state === "active" ? "backlogmd-chip-active" : ""}` });
        header.createEl("span", { text: section.milestone.id, cls: "backlogmd-section-id" });
        header.createEl("span", { text: section.milestone.title, cls: "backlogmd-section-title" });
        const metaBits: string[] = [];
        if (section.milestone.end_date) {
          const dl = daysLeft(section.milestone);
          metaBits.push(dl !== null ? `ends ${section.milestone.end_date} (${dl >= 0 ? `${dl}d left` : `${-dl}d over`})` : "");
        }
        const doneCount = section.tasks.filter((t) => project.doneStatuses.includes(t.status)).length;
        metaBits.push(`${doneCount}/${section.tasks.length} done`);
        header.createEl("span", { text: metaBits.filter(Boolean).join(" · "), cls: "backlogmd-section-meta" });
        if (section.milestone.goal) header.createEl("span", { text: section.milestone.goal, cls: "backlogmd-section-goal", title: "Goal" });
        const actions = header.createDiv({ cls: "backlogmd-section-actions" });
        if (section.milestone.state !== "active") {
          const start = actions.createEl("button", { text: "Start", cls: "backlogmd-section-btn" });
          start.addEventListener("click", async () => {
            await startMilestone(this.ctx.repo, this.ctx.milestones, project, section.milestone!);
            await this.renderSections(project);
          });
        }
        const complete = actions.createEl("button", { text: "Complete", cls: "backlogmd-section-btn" });
        complete.addEventListener("click", async () => {
          await completeMilestone(this.ctx.repo, this.ctx.milestones, project, section.milestone!, tasks);
          await this.renderSections(project);
        });
      } else {
        header.createEl("span", { text: "Backlog", cls: "backlogmd-section-title" });
        header.createEl("span", { text: `${section.tasks.length}`, cls: "backlogmd-section-meta" });
      }

      const rowsEl = el.createDiv({ cls: "backlogmd-rows" });
      for (const t of visible) {
        rowsEl.appendChild(this.renderRow(project, t, byId));
      }

      rowsEl.addEventListener("dragover", (e) => {
        e.preventDefault();
        this.showIndicator(rowsEl, e.clientY, e.dataTransfer?.getData("text/plain") ?? null);
      });
      rowsEl.addEventListener("drop", async (e) => {
        e.preventDefault();
        this.removeIndicator();
        const id = e.dataTransfer?.getData("text/plain");
        if (!id) return;
        await this.handleDrop(project, section, tasks, id, e.clientY, rowsEl);
      });
    }
  }

  private renderRow(project: ProjectConfig, task: Task, byId: Map<string, Task>): HTMLElement {
    const row = createEl("div", { cls: "backlogmd-row" });
    row.dataset.id = task.id;
    const blocked = isBlocked(task, byId, project.doneStatuses);
    if (blocked) row.addClass("blocked");

    row.createEl("span", { text: task.id, cls: "backlogmd-card-id" });
    if (task.type) row.createEl("span", { text: task.type, cls: "backlogmd-badge" });
    row.createEl("span", { text: task.title, cls: "backlogmd-row-title" });
    const right = row.createDiv({ cls: "backlogmd-row-right" });
    if (task.priority) right.createEl("span", { text: task.priority, cls: `backlogmd-badge backlogmd-p-${task.priority}` });
    if (task.assignee?.length) right.createEl("span", { text: task.assignee.join(", "), cls: "backlogmd-assignee" });
    if (blocked) right.createEl("span", { text: "⛔", cls: "backlogmd-blocked", title: "Blocked by unfinished dependencies" });
    right.createEl("span", { text: task.status, cls: "backlogmd-row-status" });

    row.setAttribute("draggable", "true");
    row.addEventListener("dragstart", (e) => {
      e.dataTransfer?.setData("text/plain", task.id);
      row.addClass("dragging");
    });
    row.addEventListener("dragend", () => row.removeClass("dragging"));
    row.addEventListener("click", (e) => {
      if (e.metaKey || e.ctrlKey) {
        const file = this.app.vault.getAbstractFileByPath(task.path);
        if (file instanceof TFile) this.app.workspace.getLeaf(true).openFile(file);
      } else {
        this.openTaskModal(project, task);
      }
    });
    return row;
  }

  private dropIndex(rowsEl: HTMLElement, clientY: number, draggedId: string | null): number {
    const rows = Array.from(rowsEl.querySelectorAll<HTMLElement>(".backlogmd-row")).filter((r) => r.dataset.id !== draggedId);
    for (let i = 0; i < rows.length; i++) {
      const r = rows[i].getBoundingClientRect();
      if (clientY < r.top + r.height / 2) return i;
    }
    return rows.length;
  }

  private showIndicator(rowsEl: HTMLElement, clientY: number, draggedId: string | null): void {
    this.removeIndicator();
    const index = this.dropIndex(rowsEl, clientY, draggedId);
    const line = createEl("div", { cls: "backlogmd-drop-line" });
    this.indicatorEl = line;
    const rows = Array.from(rowsEl.querySelectorAll<HTMLElement>(".backlogmd-row")).filter((r) => r.dataset.id !== draggedId);
    if (index >= rows.length) rowsEl.appendChild(line);
    else if (rows[index]) rowsEl.insertBefore(line, rows[index]);
  }

  private removeIndicator(): void {
    this.indicatorEl?.remove();
    this.indicatorEl = null;
  }

  private async handleDrop(project: ProjectConfig, section: Section, allTasks: Task[], id: string, clientY: number, rowsEl: HTMLElement): Promise<void> {
    const dragged = allTasks.find((t) => t.id === id);
    if (!dragged) return;

    if ((dragged.milestone ?? null) === section.milestoneId) {
      // Re-rank within the section (only when the full section is visible).
      if (this.filter.trim()) return;
      const index = this.dropIndex(rowsEl, clientY, id);
      const from = section.tasks.findIndex((t) => t.id === id);
      if (from === -1) return;
      const ordered = [...section.tasks];
      ordered.splice(from, 1);
      ordered.splice(index, 0, dragged);
      if (ordered.every((t, i) => t === section.tasks[i])) return;
      await reorderColumn(this.ctx.repo, project, ordered);
    } else {
      await planTask(this.ctx.repo, project, dragged, section.milestoneId, allTasks);
    }
    await this.renderSections(project);
  }

  private openTaskModal(project: ProjectConfig, task: Task | null): void {
    void Promise.all([this.ctx.repo.list(project), this.ctx.milestones.list(project)]).then(([all, milestones]) => {
      const active = milestones.find((m) => m.state === "active");
      const modal = new TaskModal(this.app, this.ctx, project, task, all, active?.id);
      const origClose = modal.onClose.bind(modal);
      modal.onClose = () => {
        origClose();
        void this.renderSections(project);
      };
      modal.open();
    });
  }
}
