import { App, Modal, Setting } from "obsidian";
import { PluginContext } from "../compose";
import { ProjectConfig } from "../application/ports";
import { Task } from "../domain/task";
import { createTask, CreateTaskInput } from "../application/create-task";
import { editTask } from "../application/edit-task";

export class TaskModal extends Modal {
  private ctx: PluginContext;
  private project: ProjectConfig;
  private task: Task | null;
  private summary = "";
  private type = "";
  private priority = "";
  private status = "";

  constructor(app: App, ctx: PluginContext, project: ProjectConfig, task: Task | null) {
    super(app);
    this.ctx = ctx;
    this.project = project;
    this.task = task;
    if (task) {
      this.summary = task.title;
      this.type = task.type ?? "";
      this.priority = task.priority ?? "";
      this.status = task.status;
    } else {
      this.status = project.statuses[0] ?? "To Do";
    }
  }

  onOpen(): void {
    const { contentEl } = this;
    contentEl.createEl("h2", { text: this.task ? "Edit task" : "Create task" });

    new Setting(contentEl).setName("Summary").addText((t) => {
      t.setValue(this.summary).onChange((v) => (this.summary = v));
    });

    new Setting(contentEl).setName("Type").addDropdown((d) => {
      d.addOption("", "");
      for (const ty of this.project.types) d.addOption(ty, ty);
      d.setValue(this.type).onChange((v) => (this.type = v));
    });

    new Setting(contentEl).setName("Priority").addDropdown((d) => {
      d.addOption("", "");
      for (const p of this.project.priorities) d.addOption(p, p);
      d.setValue(this.priority).onChange((v) => (this.priority = v));
    });

    new Setting(contentEl).setName("Status").addDropdown((d) => {
      for (const s of this.project.statuses) d.addOption(s, s);
      d.setValue(this.status).onChange((v) => (this.status = v));
    });

    new Setting(contentEl).addButton((b) => {
      b.setButtonText(this.task ? "Save" : "Create");
      b.setCta();
      b.onClick(async () => {
        if (!this.summary.trim()) return;
        if (this.task) {
          await editTask(this.ctx.repo, this.project, this.task, {
            title: this.summary,
            type: this.type || undefined,
            priority: this.priority || undefined,
            status: this.status,
          });
        } else {
          const input: CreateTaskInput = { summary: this.summary, type: this.type || undefined, priority: this.priority || undefined, status: this.status };
          const tasks = await this.ctx.repo.list(this.project);
          await createTask(this.ctx.repo, this.project, input, tasks.length);
        }
        this.close();
      });
    });
  }

  onClose(): void {
    this.contentEl.empty();
  }
}