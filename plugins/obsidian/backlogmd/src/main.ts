import { Plugin } from "obsidian";
import { compose, PluginContext } from "./compose";
import { KanbanView, KANBAN_VIEW_TYPE } from "./ui/kanban-view";
import { BacklogView, BACKLOG_VIEW_TYPE } from "./ui/backlog-view";
import { MilestoneView, MILESTONE_VIEW_TYPE } from "./ui/milestone-view";
import { TaskModal } from "./ui/task-modal";
import { BacklogMdSettingTab, DEFAULT_SETTINGS } from "./settings/settings-tab";
import { FileWatcher } from "./infrastructure/watcher";

export default class BacklogMdPlugin extends Plugin {
  ctx!: PluginContext;
  settings = DEFAULT_SETTINGS;

  async onload() {
    this.ctx = compose(this);
    await this.ctx.start();

    this.registerView(KANBAN_VIEW_TYPE, (leaf) => new KanbanView(leaf, this.ctx));
    this.registerView(BACKLOG_VIEW_TYPE, (leaf) => new BacklogView(leaf, this.ctx));
    this.registerView(MILESTONE_VIEW_TYPE, (leaf) => new MilestoneView(leaf, this.ctx));

    this.addCommand({ id: "open-kanban", name: "Open sprint board", callback: () => this.activateView(KANBAN_VIEW_TYPE) });
    this.addCommand({ id: "open-backlog", name: "Open backlog", callback: () => this.activateView(BACKLOG_VIEW_TYPE) });
    this.addCommand({ id: "open-milestones", name: "Open milestones", callback: () => this.activateView(MILESTONE_VIEW_TYPE) });
    this.addCommand({ id: "create-task", name: "Create task", callback: () => {
      const project = this.ctx.projects[0];
      if (project) new TaskModal(this.app, this.ctx, project, null).open();
    } });

    this.addSettingTab(new BacklogMdSettingTab(this.app, this, this.ctx, this.settings));

    new FileWatcher(this.app, () => {
      this.app.workspace.getLeavesOfType(KANBAN_VIEW_TYPE).forEach((l) => (l.view as KanbanView).render());
      this.app.workspace.getLeavesOfType(BACKLOG_VIEW_TYPE).forEach((l) => (l.view as BacklogView).render());
    });
  }

  async activateView(type: string) {
    const leaves = this.app.workspace.getLeavesOfType(type);
    if (leaves.length > 0) {
      this.app.workspace.revealLeaf(leaves[0]);
      return;
    }
    const leaf = this.app.workspace.getRightLeaf(false);
    if (leaf) {
      await leaf.setViewState({ type, active: true });
      this.app.workspace.revealLeaf(leaf);
    }
  }

  onunload() {}
}