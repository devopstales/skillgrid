import { App, Modal, Notice, Setting } from "obsidian";
import { PluginContext } from "../compose";
import type { ProjectConfig } from "../application/ports";
import type { Milestone } from "../domain/milestone";
import { nextMilestoneId } from "../application/milestone";

export class MilestoneModal extends Modal {
  private title = "";
  private goal = "";
  private endDate = "";

  constructor(
    app: App,
    private ctx: PluginContext,
    private project: ProjectConfig,
  ) {
    super(app);
  }

  onOpen(): void {
    const { contentEl } = this;
    contentEl.createEl("h2", { text: "Create milestone (sprint)" });

    new Setting(contentEl).setName("Title").setDesc("Required").addText((t) => {
      t.setPlaceholder("e.g. 2026-11-20-auth-hardening").setValue(this.title).onChange((v) => (this.title = v));
    });
    new Setting(contentEl).setName("Goal").setDesc("What this sprint is for").addTextArea((t) => {
      t.setPlaceholder("One sentence").setValue(this.goal).onChange((v) => (this.goal = v));
      t.inputEl.rows = 2;
    });
    new Setting(contentEl).setName("End date").setDesc("yyyy-mm-dd, optional").addText((t) => {
      t.setPlaceholder("2026-11-20").setValue(this.endDate).onChange((v) => (this.endDate = v));
    });

    new Setting(contentEl).addButton((b) => {
      b.setButtonText("Create");
      b.setCta();
      b.onClick(async () => {
        if (!this.title.trim()) return;
        // H5: re-list immediately before minting the id so it comes from fresh
        // state (a stale snapshot mints duplicate ids).
        const id = nextMilestoneId(await this.ctx.milestones.list(this.project));
        const m: Milestone = {
          id,
          title: this.title.trim(),
          goal: this.goal.trim() || undefined,
          end_date: this.endDate.trim() || undefined,
          state: "planned",
          path: "",
        };
        const body = `\n## Description\n\nMilestone: ${m.title}\n${this.goal.trim() ? `\nGoal: ${this.goal.trim()}\n` : ""}`;
        try {
          await this.ctx.milestones.create(this.project, m, body);
          new Notice(`Created milestone ${id}`);
          this.close();
        } catch (e) {
          new Notice(`Create failed: ${e instanceof Error ? e.message : String(e)}`);
        }
      });
    });
  }

  onClose(): void {
    this.contentEl.empty();
  }
}
