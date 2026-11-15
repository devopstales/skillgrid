import { App, ButtonComponent, Modal, Notice, Setting } from "obsidian";
import { PluginContext } from "../compose";
import { ProjectConfig } from "../application/ports";
import { Task } from "../domain/task";
import { createTask, CreateTaskInput } from "../application/create-task";
import { editTask } from "../application/edit-task";
import { linkTasks } from "../application/link-tasks";

function listValue(values: string[] | null | undefined): string {
  return (values ?? []).join(", ");
}

export function parseList(value: string): string[] {
  return value.split(",").map((s) => s.trim()).filter(Boolean);
}

export interface DepDiff {
  /** Deps in `next` that are not in `current` (user order preserved). */
  added: string[];
  /** Deps in `current` that are not in `next` (current order preserved). */
  removed: string[];
}

/** Diff the modal's next dependency list against the freshly-read task's
 *  current dependencies. The task's own id is excluded (no self-links). Pure
 *  so the link wiring (H1) is unit-testable without the DOM. */
export function diffDependencies(current: string[] | undefined, next: string[], selfId: string): DepDiff {
  const cur = new Set(current ?? []);
  const nxt = new Set(next);
  const added: string[] = [];
  const removed: string[] = [];
  for (const d of next) if (!cur.has(d) && d !== selfId) added.push(d);
  for (const d of cur) if (!nxt.has(d) && d !== selfId) removed.push(d);
  return { added, removed };
}

const AUTOSAVE_MS = 1200;

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
  private saveBtn: ButtonComponent | null = null;
  private statusEl: HTMLElement | null = null;
  private autosaveTimer: ReturnType<typeof setTimeout> | undefined;

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
      t.setPlaceholder("Task title").setValue(this.summary).onChange((v) => {
        this.summary = v;
        this.saveBtn?.setDisabled(!v.trim()); // H7: keep the button in step with the summary
        this.scheduleAutosave();
      });
    });

    new Setting(contentEl).setName("Description").setDesc("Markdown, saved as the task body").addTextArea((t) => {
      t.setPlaceholder("What needs to happen?").setValue(this.body).onChange((v) => {
        this.body = v;
        this.scheduleAutosave();
      });
      t.inputEl.rows = 6;
    });

    new Setting(contentEl).setName("Type").addDropdown((d) => {
      d.addOption("", "");
      for (const ty of this.project.types) d.addOption(ty, ty);
      d.setValue(this.type).onChange((v) => {
        this.type = v;
        this.scheduleAutosave();
      });
    });

    new Setting(contentEl).setName("Priority").addDropdown((d) => {
      d.addOption("", "");
      for (const p of this.project.priorities) d.addOption(p, p);
      d.setValue(this.priority).onChange((v) => {
        this.priority = v;
        this.scheduleAutosave();
      });
    });

    new Setting(contentEl).setName("Status").addDropdown((d) => {
      for (const s of this.project.statuses) d.addOption(s, s);
      d.setValue(this.status).onChange((v) => {
        this.status = v;
        this.scheduleAutosave();
      });
    });

    new Setting(contentEl).setName("Assignee").setDesc("Comma-separated").addText((t) => {
      t.setPlaceholder("@user, @user2").setValue(this.assignee).onChange((v) => {
        this.assignee = v;
        this.scheduleAutosave();
      });
    });

    new Setting(contentEl).setName("Milestone").addText((t) => {
      t.setPlaceholder("m-0").setValue(this.milestone).onChange((v) => {
        this.milestone = v;
        this.scheduleAutosave();
      });
    });

    new Setting(contentEl).setName("Dependencies").setDesc("Comma-separated task IDs that must be done first — unfinished ones flag this task as blocked").addTextArea((t) => {
      t.setPlaceholder("TASK-001, TASK-002").setValue(this.dependencies).onChange((v) => {
        this.dependencies = v;
        this.scheduleAutosave();
      });
      t.inputEl.rows = 2;
      this.addDatalist(t.inputEl, "backlogmd-task-ids", this.allTasks.map((x) => x.id));
    });

    new Setting(contentEl).addButton((b) => {
      this.saveBtn = b;
      b.setButtonText(this.task ? "Save" : "Create");
      b.setCta();
      b.setDisabled(!this.summary.trim()); // H7/G6: never a silent no-op
      b.onClick(() => {
        void this.saveAndClose();
      });
    });

    this.statusEl = contentEl.createDiv({ cls: "backlogmd-task-status" });
    if (this.task) this.statusEl.setText("Edits autosave");
  }

  /** A field changed: in edit mode, schedule a debounced autosave (H3).
   *  Create mode stays explicit — no autosave. */
  private scheduleAutosave(): void {
    if (!this.task) return;
    clearTimeout(this.autosaveTimer);
    this.autosaveTimer = setTimeout(() => {
      this.autosaveTimer = undefined;
      void this.autosave();
    }, AUTOSAVE_MS);
  }

  private cancelAutosave(): void {
    if (this.autosaveTimer) {
      clearTimeout(this.autosaveTimer);
      this.autosaveTimer = undefined;
    }
  }

  private setStatus(text: string): void {
    if (this.statusEl) this.statusEl.setText(text);
  }

  /** Explicit Save/Create button: flush any pending autosave, save, close on
   *  success. On failure: Notice + status, stay open (H7). */
  private async saveAndClose(): Promise<void> {
    this.cancelAutosave();
    if (!this.summary.trim()) return;
    this.setStatus("Saving…");
    try {
      if (this.task) await this.saveEdit();
      else await this.saveCreate();
      this.close();
    } catch (e) {
      new Notice(`Save failed: ${e instanceof Error ? e.message : String(e)}`);
      this.setStatus("Save failed");
    }
  }

  /** Debounced autosave (edit mode only): same save path as the button but
   *  without closing, and it updates the inline status indicator (H3). */
  private async autosave(): Promise<void> {
    if (!this.task) return;
    if (!this.summary.trim()) return;
    this.setStatus("Saving…");
    try {
      await this.saveEdit();
      this.setStatus(`Saved ${new Date().toTimeString().slice(0, 5)}`);
    } catch (e) {
      new Notice(`Autosave failed: ${e instanceof Error ? e.message : String(e)}`);
      this.setStatus("Save failed");
    }
  }

  /** Edit-mode save (C3/H1): re-read the task at save time (never the open-time
   *  snapshot), wire dependency changes both directions, then write the fresh
   *  task with the field patch as the single source of truth for its own fields. */
  private async saveEdit(): Promise<void> {
    const repo = this.ctx.repo;
    const fresh = await repo.get(this.project, this.task!.id);
    if (!fresh) throw new Error("Task no longer exists");

    const all = await repo.list(this.project);
    const byId = new Map(all.map((t) => [t.id, t]));
    const diff = diffDependencies(fresh.dependencies, parseList(this.dependencies), fresh.id);
    // linkTasks writes both sides; run it before editTask so the task's own
    // final write (full new dep list) is the last word on its dependencies.
    for (const id of diff.added) {
      const target = byId.get(id);
      if (target) await linkTasks(repo, this.project, fresh, target, true);
    }
    for (const id of diff.removed) {
      const target = byId.get(id);
      if (target) await linkTasks(repo, this.project, fresh, target, false);
    }
    await editTask(repo, this.project, fresh, this.buildPatch(), this.body);
  }

  /** Create-mode save (C3): re-list at save time so the id is minted from
   *  fresh state and the collision check sees current tasks. */
  private async saveCreate(): Promise<void> {
    const all = await this.ctx.repo.list(this.project);
    await createTask(this.ctx.repo, this.project, this.buildCreateInput(), all);
  }

  private buildPatch(): Partial<Task> {
    return {
      title: this.summary.trim(),
      type: this.type || undefined,
      priority: this.priority || undefined,
      status: this.status,
      assignee: parseList(this.assignee),
      milestone: this.milestone.trim() || undefined,
      dependencies: parseList(this.dependencies),
    };
  }

  private buildCreateInput(): CreateTaskInput {
    return {
      summary: this.summary.trim(),
      body: this.body,
      type: this.type || undefined,
      priority: this.priority || undefined,
      status: this.status,
      assignee: parseList(this.assignee),
      milestone: this.milestone.trim() || undefined,
      dependencies: parseList(this.dependencies),
    };
  }

  onClose(): void {
    this.cancelAutosave();
    this.statusEl = null;
    this.contentEl.empty();
  }
}
