import { ItemView, WorkspaceLeaf } from "obsidian";
import { PluginContext } from "../compose";

export const MILESTONE_VIEW_TYPE = "backlogmd-milestones";

export class MilestoneView extends ItemView {
  private ctx: PluginContext;
  constructor(leaf: WorkspaceLeaf, ctx: PluginContext) { super(leaf); this.ctx = ctx; }
  getViewType(): string { return MILESTONE_VIEW_TYPE; }
  getDisplayText(): string { return "Milestones"; }
  async onOpen(): Promise<void> {
    const container = this.containerEl.children[1] as HTMLElement;
    container.empty();
    const project = this.ctx.projects[0];
    if (!project) { container.createEl("div", { text: "No project", cls: "backlogmd-empty" }); return; }
    const tasks = await this.ctx.repo.list(project);
    const ms = new Set(tasks.map((t) => t.milestone).filter(Boolean));
    container.createEl("h3", { text: "Milestones" });
    for (const m of ms) {
      const btn = container.createEl("button", { text: String(m) });
      btn.addEventListener("click", async () => {
        container.querySelectorAll(".ms-tasks").forEach((n) => n.remove());
        const div = container.createDiv({ cls: "ms-tasks" });
        for (const t of tasks.filter((x) => x.milestone === m)) {
          div.createEl("div", { text: `${t.id} ${t.title}` });
        }
      });
    }
  }
}