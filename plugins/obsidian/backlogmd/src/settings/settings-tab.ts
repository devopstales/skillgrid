import { App, Plugin, PluginSettingTab, Setting } from "obsidian";

export interface BacklogMdSettings {
  manualRoots: string[];
  githubTokenKey: string;
}

export const DEFAULT_SETTINGS: BacklogMdSettings = {
  manualRoots: [],
  githubTokenKey: "backlogmd-github-token",
};

export class BacklogMdSettingTab extends PluginSettingTab {
  private plugin: Plugin;
  private settings: BacklogMdSettings;

  constructor(app: App, plugin: Plugin, settings: BacklogMdSettings) {
    super(app, plugin);
    this.plugin = plugin;
    this.settings = settings;
  }

  display(): void {
    const { containerEl } = this;
    containerEl.empty();

    const save = () => void this.plugin.saveData(this.settings);

    new Setting(containerEl).setName("Manual roots").setDesc("Comma-separated vault folders").addText((t) => {
      t.setValue(this.settings.manualRoots.join(", ")).onChange((v) => {
        this.settings.manualRoots = v.split(",").map((s) => s.trim()).filter(Boolean);
        save();
      });
    });

    new Setting(containerEl).setName("GitHub token (secret storage key)").setDesc("Obsidian secret storage key the pull dialog reads the token from").addText((t) => {
      t.setValue(this.settings.githubTokenKey).onChange((v) => {
        this.settings.githubTokenKey = v;
        save();
      });
    });
  }
}
