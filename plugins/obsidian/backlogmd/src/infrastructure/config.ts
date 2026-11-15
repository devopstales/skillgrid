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

/** Max subfolder depth scanned below each scan root / manual root
 *  (depth 1 = direct subfolder). */
const MAX_SCAN_DEPTH = 4;

/** Load + parse a YAML config file; on parse failure log with the path and
 *  return null (the config is treated as absent / the project skipped). */
async function loadYaml<T>(fs: FileSystemPort, read: () => Promise<string>, file: string): Promise<T | null> {
  try {
    const text = await read();
    return (yaml.load(text) as T) ?? ({} as T);
  } catch (err) {
    console.error(`[backlogmd] skipping malformed config ${file}: ${err instanceof Error ? err.message : String(err)}`);
    return null;
  }
}

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
    const loaded = await loadYaml<BacklogAdvancedConfig>(fs, () => fs.readText(advPath), advPath);
    if (loaded === null) return null; // malformed root config — skip project
    adv = loaded;
    if (adv.backlog_directory) backlogDir = adv.backlog_directory;
  }

  const configPath = `${rootPath}/${backlogDir}/config.yml`;
  // also try config.yaml
  let legacy: RawConfig = {};
  const candidates = [configPath, `${rootPath}/${backlogDir}/config.yaml`];
  for (const c of candidates) {
    if (await fs.exists(c)) {
      const loaded = await loadYaml<RawConfig>(fs, () => fs.readText(c), c);
      if (loaded === null) return null; // malformed legacy config — skip project
      legacy = loaded;
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
 *  root config's backlog_directory must point at or into the vault, or the
 *  backlog files are invisible to Obsidian and the ancestor is rejected. */
export async function loadAncestorProjectConfig(
  fs: FileSystemPort,
  rootAbs: string,
  vaultRel: string,
): Promise<ProjectConfig | null> {
  const loaded = await loadYaml<BacklogAdvancedConfig>(fs, () => fs.readTextAbs(path.join(rootAbs, "backlog.config.yml")), path.join(rootAbs, "backlog.config.yml"));
  if (loaded === null) return null;
  const adv = loaded;
  const backlogDir = adv.backlog_directory ?? DEFAULT_BACKLOG_DIR;
  // The backlog files must resolve to a location inside the vault (equality
  // allowed — the backlog directory may be the vault root itself), or they
  // are invisible to Obsidian (vaultRel "" means project root == vault root).
  if (vaultRel && backlogDir !== vaultRel && !backlogDir.startsWith(`${vaultRel}/`)) return null;
  const inVault = vaultRel ? (backlogDir === vaultRel ? "" : backlogDir.slice(vaultRel.length + 1)) : backlogDir;
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

/** Project markers: an advanced root config, or a default-layout legacy
 *  config. Used to prefilter scanned directories. */
async function isProjectRoot(fs: FileSystemPort, root: string): Promise<boolean> {
  return (
    (await fs.exists(`${root}/backlog.config.yml`)) ||
    (await fs.exists(`${root}/.backlog/config.yml`)) ||
    (await fs.exists(`${root}/.backlog/config.yaml`))
  );
}

/** Depth-bounded DFS collecting `root` plus its subfolders (depth 1..maxDepth). */
async function collectRoots(fs: FileSystemPort, root: string, maxDepth: number, out: string[]): Promise<void> {
  out.push(root);
  if (maxDepth <= 0) return;
  let dirs: string[] = [];
  try {
    dirs = await fs.listDirs(root);
  } catch {
    return;
  }
  for (const d of dirs) {
    await collectRoots(fs, `${root}/${d}`, maxDepth - 1, out);
  }
}

export async function discoverProjects(fs: FileSystemPort, scanRoots: string[], manualRoots: string[]): Promise<ProjectConfig[]> {
  const results: ProjectConfig[] = [];
  const seen = new Set<string>();
  const tried = new Set<string>();

  const add = (cfg: ProjectConfig | null) => {
    if (cfg && !seen.has(`${cfg.rootPath}::${cfg.vaultBacklogDir}`)) {
      seen.add(`${cfg.rootPath}::${cfg.vaultBacklogDir}`);
      results.push(cfg);
    }
  };

  const tryRoot = async (root: string) => {
    const r = root.replace(/^\/+/, "");
    if (!r || tried.has(r)) return;
    tried.add(r);
    if (!(await isProjectRoot(fs, r))) return; // cheap marker prefilter
    add(await loadProjectConfig(fs, r));
  };

  // 1. Skillgrid layout: project root is an ancestor of the vault root
  try {
    const rootAbs = await fs.findAncestorProjectRoot();
    if (rootAbs) {
      const vaultAbs = await fs.getVaultRootAbs();
      const vaultRel = path.relative(rootAbs, vaultAbs).split(path.sep).join("/");
      add(await loadAncestorProjectConfig(fs, rootAbs, vaultRel));
    }
  } catch (err) {
    // adapter without abs-path support (mobile) — fall through to in-vault discovery
    console.error(`[backlogmd] ancestor project discovery skipped: ${err instanceof Error ? err.message : String(err)}`);
  }

  // 2. Manual roots: the root itself plus its subfolders (depth-bounded)
  for (const root of manualRoots) {
    const r = root.replace(/^\/+/, "");
    const roots: string[] = [];
    await collectRoots(fs, r, MAX_SCAN_DEPTH, roots);
    for (const cr of roots) await tryRoot(cr);
  }

  // 3. Scan roots: walk subfolders depth-bounded, marker-prefiltered
  for (const scanRoot of scanRoots) {
    const sub: string[] = [];
    await collectRoots(fs, scanRoot.replace(/^\/+/, ""), MAX_SCAN_DEPTH, sub);
    sub.shift(); // the scan root itself is covered by manual/ancestor paths
    for (const cr of sub) await tryRoot(cr);
  }

  return results;
}
