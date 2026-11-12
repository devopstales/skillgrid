import { ItemView, WorkspaceLeaf } from "obsidian";
import { PluginContext } from "../compose";

export const BACKLOG_VIEW_TYPE = "backlogmd-backlog";

export class BacklogView extends ItemView {
  private ctx: PluginContext;
  private filter = "";

  constructor(leaf: WorkspaceLeaf, ctx: PluginContext) {
    super(leaf);
    this.ctx = ctx;
  }
  getViewType(): string { return BACKLOG_VIEW_TYPE; }
  getDisplayText(): string { return "Backlog List"; }

  async onOpen(): Promise<void> { await this.render(); }

  async render(): Promise<void> {
    const container = this.containerEl.children[1] as HTMLElement;
    container.empty();
    const project = this.ctx.projects[0];
    if (!project) { container.createEl("div", { text: "No project", cls: "backlogmd-empty" }); return; }
    const input = container.createEl("input", { type: "text", placeholder: "Filter..." });
    input.value = this.filter;
    input.addEventListener("input", () => { this.filter = input.value; this.renderList(container, project); });
    const list = container.createDiv();
    await this.renderList(list, project);
  }

  private async renderList(container: HTMLElement, project: any): Promise<void> {
    // remove previous list children except input
    const lists = container.querySelectorAll(".backlogmd-list");
    lists.forEach((n) => n.remove());
    const ul = container.createDiv({ cls: "backlogmd-list" });
    const tasks = await this.ctx.repo.list(project);
    const q = this.filter.toLowerCase();
    for (const t of tasks) {
      if (q && !t.title.toLowerCase().includes(q) && t.type !== q) continue;
      const row = ul.createDiv({ cls: "backlogmd-card" });
      row.createEl("span", { text: `${t.id} ${t.title}` });
      if (t.type) row.createEl("small", { text: ` [${t.type}]` });
    }
  }
}