/** Minimal runtime stub of the "obsidian" API for vitest (the real package
 *  is types-only for the bundle; esbuild marks it external). Extend only
 *  when a test actually exercises a member at runtime. */

export class Plugin {
  app: unknown;
  private events: unknown[] = [];
  constructor(app?: unknown) {
    this.app = app;
  }
  registerEvent(_ref: unknown): void {
    this.events.push(_ref);
  }
  registerInterval(_id: unknown): void {
    this.events.push(_id);
  }
}

export class TFile {
  path = "";
  name = "";
  basename = "";
  extension = "";
  constructor(path?: string) {
    this.path = path ?? "";
    this.name = this.path.split("/").pop() ?? "";
    const dot = this.name.lastIndexOf(".");
    this.basename = dot === -1 ? this.name : this.name.slice(0, dot);
    this.extension = dot === -1 ? "" : this.name.slice(dot + 1);
  }
}

export class Notice {
  static instances: Notice[] = [];
  message: string;
  timeout: number;
  constructor(message: string, timeout = 5000) {
    this.message = message;
    this.timeout = timeout;
    Notice.instances.push(this);
  }
}

export class Modal {
  app: unknown;
  contentEl: HTMLElement;
  titleEl: HTMLElement;
  constructor(app?: unknown) {
    this.app = app;
    this.contentEl = globalThis.document
      ? globalThis.document.createElement("div")
      : ({} as HTMLElement);
    this.titleEl = this.contentEl;
  }
  open(): void {}
  close(): void {}
}

export class Setting {
  settingEl: unknown;
  constructor(containerEl?: unknown) {
    this.settingEl = containerEl;
  }
  setName(_n: string) {
    return this;
  }
  setDesc(_d: string) {
    return this;
  }
  addButton(_cb: (b: ButtonLike) => void) {
    return this;
  }
  addText(_cb: (t: ButtonLike) => void) {
    return this;
  }
  addTextArea(_cb: (t: ButtonLike) => void) {
    return this;
  }
  addDropdown(_cb: (d: ButtonLike) => void) {
    return this;
  }
}

export interface ButtonLike {
  setButtonText(t: string): ButtonLike;
  setCta(): ButtonLike;
  setDisabled(d: boolean): ButtonLike;
  setPlaceholder(p: string): ButtonLike;
  setValue(v: string): ButtonLike;
  onClick(cb: () => void): ButtonLike;
  onChange(cb: () => void): ButtonLike;
}
