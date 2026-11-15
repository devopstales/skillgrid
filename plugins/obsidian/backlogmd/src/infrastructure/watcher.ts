import { App, EventRef, TAbstractFile, TFile } from "obsidian";

/** Vault events that can change the backlog. `rename` is included because
 *  the backlog CLI renames task files when a title changes — without it the
 *  board keeps stale titles until some other file changes (H6). */
const EVENTS: Array<"modify" | "create" | "delete" | "rename"> = ["modify", "create", "delete", "rename"];

/** Re-renders the board when a file the project actually reads changes.
 *  Handlers are registered through `register` (the plugin's registerEvent)
 *  so they are released on unload — no stale listeners across reloads (H6).
 *  Triggers are scoped to the projects' tasks/ and milestones/ directories
 *  so unrelated vault notes no longer cause re-renders (H6). */
export class FileWatcher {
  private timer: ReturnType<typeof setTimeout> | undefined;

  constructor(
    app: App,
    register: (event: EventRef) => void,
    /** Live supplier of the vault-relative backlog directories (projects can
     *  change after a settings edit — don't snapshot them). */
    projectDirs: () => string[],
    cb: () => void,
  ) {
    for (const name of EVENTS) {
      // The vault emitter is overload-typed per event name; a union `name`
      // matches no single overload, so widen the call.
      register(
        (app.vault.on as (n: string, cb: (file: TAbstractFile) => void) => EventRef)(name, (file) => {
          if (!(file instanceof TFile) || file.extension !== "md") return;
          if (!this.underProject(file.path, projectDirs())) return;
          this.schedule(cb);
        }),
      );
    }
  }

  private underProject(path: string, dirs: string[]): boolean {
    for (const d of dirs) {
      const base = d === "" ? "" : `${d}/`;
      if (path.startsWith(`${base}tasks/`) || path.startsWith(`${base}milestones/`)) return true;
    }
    return false;
  }

  private schedule(cb: () => void) {
    clearTimeout(this.timer);
    this.timer = setTimeout(() => cb(), 300);
  }

  destroy() {
    clearTimeout(this.timer);
  }
}
