import { ProjectConfig, FileSystemPort } from "../application/ports";
import yaml from "js-yaml";
import path from "path";

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
  types?: string[];
  priorities?: string[];
  done_statuses?: string[];
  task_prefix?: string;
}

const DEFAULT_BACKLOG_DIR = ".backlog";
const DEFAULT_STATUSES = ["To Do", "In Progress", "Done"];
const DEFAULT_DONE_STATUSES = ["Done", "done"];

/** Walk up from startDir (inclusive) at most maxDepth levels, returning the
 *  first directory containing fileName; null when none does. */
export function walkUpForFile(
  startDir: string,
  fileName: string,
  maxDepth: number,
  exists: (p: string) => boolean,
): string | null {
  let dir = startDir;
  for (let i = 0; i <= maxDepth; i++) {
    if (exists(path.join(dir, fileName))) return dir;
    const parent = path.dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  return null;
}

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

  const statuses = legacy.statuses ?? adv.statuses ?? DEFAULT_STATUSES;
  const doneStatuses = legacy.done_statuses ?? adv.done_statuses ?? DEFAULT_DONE_STATUSES;
  const base = rootPath ? `${rootPath}/${backlogDir}` : backlogDir;
  return {
    name: adv.project_name ?? legacy.project_name ?? rootPath.split("/").pop() ?? "project",
    rootPath,
    backlogDirectory: backlogDir,
    vaultBacklogDir: base,
    statuses,
    types: legacy.types ?? adv.types ?? [],
    priorities: legacy.priorities ?? adv.priorities ?? [],
    doneStatuses,
    taskPrefix: legacy.task_prefix ?? adv.task_prefix ?? "task",
  };
}

/** Skillgrid layout: the project root (holding backlog.config.yml) sits ABOVE
 *  the vault root (the projectwiki vault is a nested folder of the repo). The
 *  root config's backlog_directory must point back into the vault, or the
 *  backlog files are invisible to Obsidian and the ancestor is rejected. */
export async function loadAncestorProjectConfig(
  fs: FileSystemPort,
  rootAbs: string,
  vaultRel: string,
): Promise<ProjectConfig | null> {
  const adv: BacklogAdvancedConfig =
    (yaml.load(await fs.readTextAbs(path.join(rootAbs, "backlog.config.yml"))) as BacklogAdvancedConfig) ?? {};
  const backlogDir = adv.backlog_directory ?? DEFAULT_BACKLOG_DIR;
  // The backlog files must resolve to a location inside the vault, or they are
  // invisible to Obsidian (vaultRel "" means project root == vault root).
  if (vaultRel && !backlogDir.startsWith(`${vaultRel}/`)) return null;
  const inVault = vaultRel ? backlogDir.slice(vaultRel.length + 1) : backlogDir;
  if (!(await fs.exists(inVault))) return null;
  return {
    name: adv.project_name ?? (path.basename(rootAbs) || "project"),
    rootPath: "",
    backlogDirectory: backlogDir,
    vaultBacklogDir: inVault,
    statuses: adv.statuses ?? DEFAULT_STATUSES,
    types: adv.types ?? [],
    priorities: adv.priorities ?? [],
    doneStatuses: adv.done_statuses ?? DEFAULT_DONE_STATUSES,
    taskPrefix: adv.task_prefix ?? "task",
  };
}

export async function discoverProjects(fs: FileSystemPort, scanRoots: string[], manualRoots: string[]): Promise<ProjectConfig[]> {
  const results: ProjectConfig[] = [];
  const seen = new Set<string>();

  const add = (cfg: ProjectConfig | null) => {
    if (cfg && !seen.has(`${cfg.rootPath}::${cfg.vaultBacklogDir}`)) {
      seen.add(`${cfg.rootPath}::${cfg.vaultBacklogDir}`);
      results.push(cfg);
    }
  };

  const tryRoot = async (root: string) => {
    add(await loadProjectConfig(fs, root));
  };

  // 1. Skillgrid layout: project root is an ancestor of the vault root
  try {
    const rootAbs = await fs.findAncestorProjectRoot();
    if (rootAbs) {
      const vaultAbs = await fs.getVaultRootAbs();
      const vaultRel = path.relative(rootAbs, vaultAbs).split(path.sep).join("/");
      add(await loadAncestorProjectConfig(fs, rootAbs, vaultRel));
    }
  } catch {
    // adapter without abs-path support (mobile) — fall through to in-vault discovery
  }

  // 2. Manual roots
  for (const root of manualRoots) await tryRoot(root.replace(/^\/+/, ""));

  // 3. Scan roots: try each top-level subfolder as a project root
  for (const scanRoot of scanRoots) {
    let dirs: string[] = [];
    try {
      dirs = await fs.listDirs(scanRoot);
    } catch {
      continue;
    }
    for (const d of dirs) {
      await tryRoot(`${scanRoot}/${d}`.replace(/^\/+/, ""));
    }
  }

  return results;
}
