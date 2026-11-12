import { App, DataAdapter, TFile, TFolder } from "obsidian";
import { FileSystemPort } from "../application/ports";
import { walkUpForFile } from "./config";
import fs from "fs";

interface BasePathAdapter extends DataAdapter {
  getBasePath(): string;
}

export class ObsidianFileSystem implements FileSystemPort {
  constructor(private app: App) {}

  private normalize(p: string): string {
    return p.replace(/^\/+/, "");
  }

  async exists(path: string): Promise<boolean> {
    return this.app.vault.adapter.exists(this.normalize(path));
  }

  async readText(path: string): Promise<string> {
    return this.app.vault.adapter.read(this.normalize(path));
  }

  async writeText(path: string, content: string): Promise<void> {
    const p = this.normalize(path);
    if (await this.app.vault.adapter.exists(p)) {
      await this.app.vault.adapter.write(p, content);
    } else {
      await this.app.vault.adapter.write(p, content);
    }
  }

  async listDirs(path: string): Promise<string[]> {
    const p = this.normalize(path);
    const folder = this.app.vault.getAbstractFileByPath(p);
    if (folder instanceof TFolder) {
      return folder.children.filter((c) => c instanceof TFolder).map((c) => c.name);
    }
    return [];
  }

  async listFiles(path: string): Promise<string[]> {
    const p = this.normalize(path);
    const folder = this.app.vault.getAbstractFileByPath(p);
    if (folder instanceof TFolder) {
      return folder.children.filter((c) => c instanceof TFile).map((c) => c.name);
    }
    return [];
  }

  getVaultRoot(): string {
    return "";
  }

  // Desktop only: FileSystemAdapter exposes getBasePath; the mobile adapter
  // does not (methods that need it throw/return null, discovery falls back
  // to in-vault roots).
  private basePath(): string | null {
    const a = this.app.vault.adapter as BasePathAdapter;
    return typeof a.getBasePath === "function" ? a.getBasePath() : null;
  }

  async getVaultRootAbs(): Promise<string> {
    const b = this.basePath();
    if (b === null) throw new Error("adapter has no getBasePath");
    return b;
  }

  async findAncestorProjectRoot(maxDepth = 5): Promise<string | null> {
    const b = this.basePath();
    if (b === null) return null;
    return walkUpForFile(b, "backlog.config.yml", maxDepth, (p) => fs.existsSync(p));
  }

  async readTextAbs(absPath: string): Promise<string> {
    return fs.promises.readFile(absPath, "utf8");
  }
}