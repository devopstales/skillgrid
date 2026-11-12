import { App, Modal, Notice, Setting } from "obsidian";
import { PluginContext } from "../compose";
import { ProjectConfig } from "../application/ports";
import { pullIssues } from "../application/pull-issues";

export class PullIssuesModal extends Modal {
  private owner = "";
  private repository = "";
  private token = "";

  constructor(
    app: App,
    private ctx: PluginContext,
    private project: ProjectConfig,
    private tokenKey: string,
  ) {
    super(app);
  }

  async onOpen(): Promise<void> {
    const { contentEl } = this;
    contentEl.createEl("h2", { text: "Pull issues from GitHub" });
    contentEl.createEl("p", {
      text: "Imports repository issues into the project's tasks (one-way). Later pulls keep title, assignee and open/closed state in step; local planning fields are never overwritten.",
      cls: "backlogmd-modal-desc",
    });

    const hasStored = (await this.app.secretStorage.getSecret(this.tokenKey)) !== null;

    new Setting(contentEl).setName("Owner").addText((t) => t.setPlaceholder("owner").setValue(this.owner).onChange((v) => (this.owner = v)));
    new Setting(contentEl).setName("Repository").addText((t) => t.setPlaceholder("repo").setValue(this.repository).onChange((v) => (this.repository = v)));
    new Setting(contentEl).setName("Token").setDesc("Optional for public repos; stored in Obsidian secret storage").addText((t) => {
      t.setPlaceholder(hasStored ? "stored — leave blank to keep" : "ghp_…");
      t.inputEl.type = "password";
      t.onChange((v) => (this.token = v));
    });

    new Setting(contentEl).addButton((b) => {
      b.setButtonText("Pull");
      b.setCta();
      b.onClick(async () => {
        const owner = this.owner.trim();
        const repository = this.repository.trim();
        if (!owner || !repository) return;
        const stored = (await this.app.secretStorage.getSecret(this.tokenKey)) ?? "";
        const token = this.token.trim() || stored;
        if (this.token.trim()) await this.app.secretStorage.setSecret(this.tokenKey, this.token.trim());
        try {
          await pullIssues(this.ctx.repo, this.project, token, owner, repository);
          new Notice(`Pulled issues from ${owner}/${repository}`);
          this.close();
        } catch (e) {
          new Notice(`Pull failed: ${e instanceof Error ? e.message : String(e)}`);
        }
      });
    });
  }

  onClose(): void {
    this.contentEl.empty();
  }
}
