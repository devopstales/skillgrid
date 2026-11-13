import { ItemView, WorkspaceLeaf } from "obsidian";
import { PluginContext } from "../compose";
import { ProjectConfig } from "../application/ports";
import { Milestone, effectiveState, daysLeft } from "../domain/milestone";
import { renderProjectSwitcher } from "./project-switcher";

export const MILESTONE_VIEW_TYPE = "backlogmd-milestones";

const byMilestoneId = (a: Milestone, b: Milestone): number =>
  a.id.localeCompare(b.id, undefined, { numeric: true });

/** Milestone list loaded from backlog/milestones/: state chip, end date,
 *  done x/y; clicking a milestone filters the task list below it. */
export class MilestoneView extends ItemView {
  private ctx: PluginContext;
  projectIndex = 0;
  private listEl: HTMLElement | null = null;
  private renderSeq = 0;

  constructor(leaf: WorkspaceLeaf, ctx: PluginContext) {
    super(leaf);
    this.ctx = ctx;
  }

  getViewType(): string {
    return MILESTONE_VIEW_TYPE;
  }
  getDisplayText(): string {
    return "Milestones";
  }

  async onOpen(): Promise<void> {
    await this.render();
  }

  async render(): Promise<void> {
    try {
      await this.renderInner();
    } catch (e) {
      console.error("[backlogmd] milestones render failed:", (e as Error).stack ?? String(e));
      throw e;
    }
  }

  private async renderInner(): Promise<void> {
    const container = this.containerEl.children[1] as HTMLElement;
    container.empty();
    this.listEl = null;

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

    this.listEl = container.createDiv({ cls: "backlogmd-milestone-list" });
    await this.renderList(project);
  }

  private async renderList(project: ProjectConfig): Promise<void> {
    const host = this.listEl;
    if (!host) return;
    const seq = ++this.renderSeq;
    host.empty();

    const [tasks, milestones] = await Promise.all([this.ctx.repo.list(project), this.ctx.milestones.list(project)]);
    if (seq !== this.renderSeq) return; // a newer render owns the DOM now

    if (milestones.length === 0) {
      host.createEl("div", { text: "No milestones in backlog/milestones/.", cls: "backlogmd-empty" });
      return;
    }

    for (const m of [...milestones].sort(byMilestoneId)) {
      const msTasks = tasks.filter((t) => t.milestone === m.id);
      const state = effectiveState(m, msTasks, project.doneStatuses);
      const done = msTasks.filter((t) => project.doneStatuses.includes(t.status)).length;

      const row = host.createDiv({ cls: "backlogmd-milestone-row" });
      const header = row.createDiv({ cls: "backlogmd-milestone-header" });
      header.createEl("span", { text: state, cls: `backlogmd-chip${state === "active" ? " backlogmd-chip-active" : ""}` });
      header.createEl("span", { text: m.id, cls: "backlogmd-section-id" });
      header.createEl("span", { text: m.title, cls: "backlogmd-section-title" });
      const bits: string[] = [];
      if (m.end_date) {
        const dl = daysLeft(m);
        bits.push(dl !== null ? (dl >= 0 ? `ends ${m.end_date} · ${dl}d left` : `overdue by ${-dl}d`) : `ends ${m.end_date}`);
      }
      bits.push(`${done}/${msTasks.length} done`);
      header.createEl("span", { text: bits.join(" · "), cls: "backlogmd-section-meta" });

      const details = row.createDiv({ cls: "backlogmd-milestone-tasks" });
      details.style.display = "none";
      for (const t of msTasks) {
        const line = details.createDiv({ cls: "backlogmd-milestone-task" });
        line.createEl("span", { text: t.id, cls: "backlogmd-card-id" });
        line.createEl("span", { text: t.status, cls: "backlogmd-row-status" });
        line.createEl("span", { text: t.title, cls: "backlogmd-row-title" });
      }

      header.addEventListener("click", () => {
        details.style.display = details.style.display === "none" ? "" : "none";
      });
    }
  }
}
