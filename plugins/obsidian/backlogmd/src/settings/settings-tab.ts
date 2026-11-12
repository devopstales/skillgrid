import { App, PluginSettingTab, Setting } from "obsidian";
import { PluginContext } from "../compose";

export interface BacklogMdSettings {
  manualRoots: string[];
  activeProject: string;
  githubTokenKey: string;
}

export const DEFAULT_SETTINGS: BacklogMdSettings = {
  manualRoots: [],
  activeProject: "",
  githubTokenKey: "backlogmd-github-token",
};

export class BacklogMdSettingTab extends PluginSettingTab {
  private ctx: PluginContext;
  private settings: BacklogMdSettings;

  constructor(app: App, plugin: any, ctx: PluginContext, settings: BacklogMdSettings) {
    super(app, plugin);
    this.ctx = ctx;
    this.settings = settings;
  }

  display(): void {
    const { containerEl } = this;
    containerEl.empty();

    new Setting(containerEl).setName("Manual roots").setDesc("Comma-separated vault folders").addText((t) => {
      t.setValue(this.settings.manualRoots.join(", ")).onChange((v) => {
        this.settings.manualRoots = v.split(",").map((s) => s.trim()).filter(Boolean);
      });
    });

    new Setting(containerEl).setName("GitHub token (secret storage key)").addText((t) => {
      t.setValue(this.settings.githubTokenKey).onChange((v) => {
        this.settings.githubTokenKey = v;
      });
    });
  }
}