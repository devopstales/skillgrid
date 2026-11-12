import { ProjectConfig, FileSystemPort } from "../application/ports";
import yaml from "js-yaml";

interface RawConfig {
  project_name?: string;
  statuses?: string[];
  types?: string[];
  priorities?: string[];
  done_statuses?: string[];
  task_prefix?: string;
  backlog_directory?: string;
}

interface BacklogAdvancedConfig {
  project_name?: string;
  backlog_directory?: string;
  default_status?: string;
  statuses?: string[];
  labels?: string[];
  task_prefix?: string;
}

const DEFAULT_BACKLOG_DIR = ".backlog";

export async function loadProjectConfig(fs: FileSystemPort, rootPath: string): Promise<ProjectConfig | null> {
  // Advanced layout: root-level backlog.config.yml
  const advPath = `${rootPath}/backlog.config.yml`;
  let backlogDir = DEFAULT_BACKLOG_DIR;
  let adv: BacklogAdvancedConfig = {};
  if (await fs.exists(advPath)) {
    const text = await fs.readText(advPath);
    adv = (yaml.load(text) as BacklogAdvancedConfig) ?? {};
    if (adv.backlog_directory) backlogDir = adv.backlog_directory;
  }

  const configPath = `${rootPath}/${backlogDir}/config.yml`;
  // also try config.yaml
  let legacy: RawConfig = {};
  const candidates = [configPath, `${rootPath}/${backlogDir}/config.yaml`];
  for (const c of candidates) {
    if (await fs.exists(c)) {
      const text = await fs.readText(c);
      legacy = (yaml.load(text) as RawConfig) ?? {};
      break;
    }
  }

  // If neither advanced nor legacy config exists, not a project
  const hasAny = (await fs.exists(advPath)) || (await fs.exists(`${rootPath}/${backlogDir}/config.yml`)) || (await fs.exists(`${rootPath}/${backlogDir}/config.yaml`));
  if (!hasAny) return null;

  const statuses = legacy.statuses ?? (adv.statuses as string[] | undefined) ?? ["To Do", "In Progress", "Done"];
  const doneStatuses = legacy.done_statuses ?? ["Done"];
  return {
    name: adv.project_name ?? legacy.project_name ?? rootPath.split("/").pop() ?? "project",
    rootPath,
    backlogDirectory: backlogDir,
    statuses,
    types: legacy.types ?? [],
    priorities: legacy.priorities ?? [],
    doneStatuses,
    taskPrefix: legacy.task_prefix ?? adv.task_prefix ?? "task",
  };
}

export async function discoverProjects(fs: FileSystemPort, scanRoots: string[], manualRoots: string[]): Promise<ProjectConfig[]> {
  const results: ProjectConfig[] = [];
  const seen = new Set<string>();

  const tryRoot = async (root: string) => {
    const cfg = await loadProjectConfig(fs, root);
    if (cfg && !seen.has(cfg.rootPath)) {
      seen.add(cfg.rootPath);
      results.push(cfg);
    }
  };

  for (const root of manualRoots) await tryRoot(root);

  for (const scanRoot of scanRoots) {
    let dirs: string[] = [];
    try {
      dirs = await fs.listDirs(scanRoot);
    } catch {
      continue;
    }
    for (const d of dirs) {
      await tryRoot(`${scanRoot}/${d}`);
    }
  }

  return results;
}