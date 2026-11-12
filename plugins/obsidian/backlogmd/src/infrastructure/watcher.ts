import { App, TFile } from "obsidian";

export class FileWatcher {
  private timer: ReturnType<typeof setTimeout> | null = null;
  private cb: () => void;

  constructor(app: App, cb: () => void) {
    this.cb = cb;
    app.vault.on("modify", (file) => {
      if (file instanceof TFile && file.extension === "md") this.schedule();
    });
    app.vault.on("create", (file) => {
      if (file instanceof TFile && file.extension === "md") this.schedule();
    });
    app.vault.on("delete", (file) => {
      if (file instanceof TFile && file.extension === "md") this.schedule();
    });
  }

  private schedule() {
    if (this.timer) clearTimeout(this.timer);
    this.timer = setTimeout(() => this.cb(), 300);
  }

  destroy() {
    if (this.timer) clearTimeout(this.timer);
  }
}