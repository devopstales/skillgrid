import { App, Modal, Setting } from "obsidian";
import { PluginContext } from "../compose";
import { ProjectConfig } from "../application/ports";
import { Task } from "../domain/task";
import { createTask, CreateTaskInput } from "../application/create-task";
import { editTask } from "../application/edit-task";

function listValue(values: string[] | undefined): string {
  return (values ?? []).join(", ");
}

function parseList(value: string): string[] {
  return value.split(",").map((s) => s.trim()).filter(Boolean);
}

export class TaskModal extends Modal {
  private ctx: PluginContext;
  private project: ProjectConfig;
  private task: Task | null;
  private allTasks: Task[];
  private summary = "";
  private body = "";
  private type = "";
  private priority = "";
  private status = "";
  private assignee = "";
  private milestone = "";
  private dependencies = "";

  constructor(app: App, ctx: PluginContext, project: ProjectConfig, task: Task | null, allTasks: Task[], defaultMilestone?: string) {
    super(app);
    this.ctx = ctx;
    this.project = project;
    this.task = task;
    this.allTasks = allTasks;
    if (task) {
      this.summary = task.title;
      this.body = task.body ?? "";
      this.type = task.type ?? "";
      this.priority = task.priority ?? "";
      this.status = task.status;
      this.assignee = listValue(task.assignee);
      this.milestone = task.milestone ?? "";
      this.dependencies = listValue(task.dependencies);
    } else {
      this.status = project.statuses[0] ?? "To Do";
      this.milestone = defaultMilestone ?? "";
    }
  }

  private addDatalist(input: HTMLTextAreaElement, id: string, options: string[]): void {
    const list = this.contentEl.createEl("datalist");
    list.id = id;
    for (const o of options) list.createEl("option", { value: o });
    input.setAttribute("list", id);
  }

  onOpen(): void {
    const { contentEl } = this;
    contentEl.createEl("h2", { text: this.task ? `Edit ${this.task.id}` : "Create task" });

    new Setting(contentEl).setName("Summary").setDesc("Required").addText((t) => {
      t.setPlaceholder("Task title").setValue(this.summary).onChange((v) => (this.summary = v));
    });

    new Setting(contentEl).setName("Description").setDesc("Markdown, saved as the task body").addTextArea((t) => {
      t.setPlaceholder("What needs to happen?").setValue(this.body).onChange((v) => (this.body = v));
      t.inputEl.rows = 6;
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

    new Setting(contentEl).setName("Assignee").setDesc("Comma-separated").addText((t) => {
      t.setPlaceholder("@user, @user2").setValue(this.assignee).onChange((v) => (this.assignee = v));
    });

    new Setting(contentEl).setName("Milestone").addText((t) => {
      t.setPlaceholder("m-0").setValue(this.milestone).onChange((v) => (this.milestone = v));
    });

    new Setting(contentEl).setName("Dependencies").setDesc("Comma-separated task IDs that must be done first — unfinished ones flag this task as blocked").addTextArea((t) => {
      t.setPlaceholder("TASK-001, TASK-002").setValue(this.dependencies).onChange((v) => (this.dependencies = v));
      t.inputEl.rows = 2;
      this.addDatalist(t.inputEl, "backlogmd-task-ids", this.allTasks.map((x) => x.id));
    });

    new Setting(contentEl).addButton((b) => {
      b.setButtonText(this.task ? "Save" : "Create");
      b.setCta();
      b.onClick(async () => {
        if (!this.summary.trim()) return;
        const patch = {
          title: this.summary.trim(),
          type: this.type || undefined,
          priority: this.priority || undefined,
          status: this.status,
          assignee: parseList(this.assignee),
          milestone: this.milestone.trim() || undefined,
          dependencies: parseList(this.dependencies),
        };
        if (this.task) {
          await editTask(this.ctx.repo, this.project, this.task, patch, this.body);
        } else {
          const input: CreateTaskInput = {
            summary: this.summary.trim(),
            body: this.body,
            type: this.type || undefined,
            priority: this.priority || undefined,
            status: this.status,
            assignee: parseList(this.assignee),
            milestone: this.milestone.trim() || undefined,
            dependencies: parseList(this.dependencies),
          };
          await createTask(this.ctx.repo, this.project, input, this.allTasks);
        }
        this.close();
      });
    });
  }

  onClose(): void {
    this.contentEl.empty();
  }
}
