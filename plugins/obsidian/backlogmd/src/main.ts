import { Plugin } from "obsidian";
import { compose, PluginContext } from "./compose";
import { KanbanView, KANBAN_VIEW_TYPE } from "./ui/kanban-view";
import { BacklogView, BACKLOG_VIEW_TYPE } from "./ui/backlog-view";
import { MilestoneView, MILESTONE_VIEW_TYPE } from "./ui/milestone-view";
import { TaskModal } from "./ui/task-modal";
import { PullIssuesModal } from "./ui/pull-issues-modal";
import { MilestoneModal } from "./ui/milestone-modal";
import { BacklogMdSettingTab, DEFAULT_SETTINGS } from "./settings/settings-tab";
import { FileWatcher } from "./infrastructure/watcher";

export default class BacklogMdPlugin extends Plugin {
  ctx!: PluginContext;
  settings = DEFAULT_SETTINGS;

  async onload() {
    this.ctx = compose(this);
    await this.ctx.start();
    console.log(
      "[backlogmd] discovered projects:",
      this.ctx.projects.map((p) => `${p.name} (${p.vaultBacklogDir})`).join(", ") || "(none)",
    );

    this.registerView(KANBAN_VIEW_TYPE, (leaf) => new KanbanView(leaf, this.ctx));
    this.registerView(BACKLOG_VIEW_TYPE, (leaf) => new BacklogView(leaf, this.ctx));
    this.registerView(MILESTONE_VIEW_TYPE, (leaf) => new MilestoneView(leaf, this.ctx));

    this.addCommand({ id: "open-kanban", name: "Open sprint board", callback: () => this.activateView(KANBAN_VIEW_TYPE) });
    this.addCommand({ id: "open-backlog", name: "Open backlog", callback: () => this.activateView(BACKLOG_VIEW_TYPE) });
    this.addCommand({ id: "open-milestones", name: "Open milestones", callback: () => this.activateView(MILESTONE_VIEW_TYPE) });
    this.addCommand({ id: "create-task", name: "Create task", callback: () => {
      const project = this.projectFromViews();
      if (project) void this.ctx.repo.list(project).then((all) => new TaskModal(this.app, this.ctx, project, null, all).open());
    } });
    this.addCommand({ id: "pull-issues", name: "Pull issues from GitHub repository", callback: () => {
      const project = this.projectFromViews();
      if (project) new PullIssuesModal(this.app, this.ctx, project, this.settings.githubTokenKey).open();
    } });
    this.addCommand({ id: "create-milestone", name: "Create milestone (sprint)", callback: () => {
      const project = this.projectFromViews();
      if (project) void this.ctx.milestones.list(project).then((all) => new MilestoneModal(this.app, this.ctx, project, all).open());
    } });

    this.addSettingTab(new BacklogMdSettingTab(this.app, this, this.ctx, this.settings));

    new FileWatcher(this.app, () => {
      const reRender = (type: string) => {
        // A leaf can be mid-restore at this point (view not swapped yet) —
        // only re-render views that are actually one of ours and ready.
        for (const l of this.app.workspace.getLeavesOfType(type)) {
          const v = l.view as { render?: () => Promise<void> } | null | undefined;
          if (v && typeof v.render === "function") void v.render();
        }
      };
      reRender(KANBAN_VIEW_TYPE);
      reRender(BACKLOG_VIEW_TYPE);
      reRender(MILESTONE_VIEW_TYPE);
    });
  }

  // Follow the project selected in an open board/backlog view, else the first project.
  private projectFromViews() {
    const k = this.app.workspace.getLeavesOfType(KANBAN_VIEW_TYPE)[0]?.view as unknown as KanbanView | undefined;
    const b = this.app.workspace.getLeavesOfType(BACKLOG_VIEW_TYPE)[0]?.view as unknown as BacklogView | undefined;
    const view = k ?? b;
    const idx = view ? view.projectIndex : 0;
    return this.ctx.projects[idx] ?? this.ctx.projects[0] ?? null;
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