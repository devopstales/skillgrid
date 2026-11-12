import { App, TFile, TFolder } from "obsidian";
import { FileSystemPort } from "../application/ports";

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
}