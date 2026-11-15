import { Plugin } from "obsidian";
import { ObsidianFileSystem } from "./infrastructure/obsidian-fs";
import { discoverProjects } from "./infrastructure/config";
import { FileTaskRepository } from "./infrastructure/task-repository";
import { FileMilestoneRepository } from "./infrastructure/milestone-repository";
import type { MilestoneRepository, ProjectConfig } from "./application/ports";

/** Coerce hand-edited / stale `manualRoots` from data.json: array → its
 *  string entries, string → single-element array, anything else → []. */
export function coerceManualRoots(value: unknown): string[] {
  if (Array.isArray(value)) return value.filter((v): v is string => typeof v === "string");
  if (typeof value === "string") {
    const trimmed = value.trim();
    return trimmed ? [trimmed] : [];
  }
  return [];
}

export interface PluginContext {
  fs: ObsidianFileSystem;
  repo: FileTaskRepository;
  milestones: MilestoneRepository;
  projects: ProjectConfig[];
  start(manualRoots: string[] | string | null | undefined): Promise<void>;
}

export function compose(plugin: Plugin): PluginContext {
  const fs = new ObsidianFileSystem(plugin.app);
  const repo = new FileTaskRepository(fs);
  const milestones = new FileMilestoneRepository(fs);
  const ctx: PluginContext = {
    fs,
    repo,
    milestones,
    projects: [],
    async start(manualRoots: string[] | string | null | undefined) {
      // discover projects from vault root folders (top-level)
      const roots: string[] = [];
      // Obsidian vault root children folders
      const vault = plugin.app.vault;
      const rootFolder = vault.getRoot() as unknown as { children: { path: string; name: string; children?: unknown }[] };
      for (const c of rootFolder.children) {
        if (c.children !== undefined) roots.push(c.path || c.name);
      }
      for (const r of coerceManualRoots(manualRoots)) {
        if (!roots.includes(r)) roots.push(r);
      }
      ctx.projects = await discoverProjects(fs, ["/"], roots);
    },
  };
  return ctx;
}
