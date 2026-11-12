import { Plugin } from "obsidian";
import { ObsidianFileSystem } from "./infrastructure/obsidian-fs";
import { discoverProjects } from "./infrastructure/config";
import { FileTaskRepository } from "./infrastructure/task-repository";

export interface PluginContext {
  fs: ObsidianFileSystem;
  repo: FileTaskRepository;
  projects: Awaited<ReturnType<typeof discoverProjects>>;
  start(): Promise<void>;
}

export function compose(plugin: Plugin): PluginContext {
  const fs = new ObsidianFileSystem(plugin.app);
  const repo = new FileTaskRepository(fs);
  const ctx: PluginContext = {
    fs,
    repo,
    projects: [],
    async start() {
      // discover projects from vault root folders (top-level)
      const root = "/";
      const roots: string[] = [];
      // Obsidian vault root children folders
      const vault = plugin.app.vault;
      const rootFolder = vault.getRoot() as unknown as { children: { path: string; name: string; children?: unknown }[] };
      for (const c of rootFolder.children) {
        if (c.children !== undefined) roots.push(c.path || c.name);
      }
      // manual roots from settings could be added later
      ctx.projects = await discoverProjects(fs, ["/"], roots);
    },
  };
  return ctx;
}