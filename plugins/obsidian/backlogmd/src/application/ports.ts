import type { Milestone } from "../domain/milestone";

export interface ProjectConfig {
  name: string;
  rootPath: string;
  backlogDirectory: string;
  /** Vault-relative path of the backlog directory — the location the
   *  repository reads/writes task files. Equals rootPath/backlogDirectory for
   *  in-vault projects; for skillgrid-style layouts (project root above the
   *  vault root) it is the in-vault tail of the repo-relative backlogDirectory. */
  vaultBacklogDir: string;
  statuses: string[];
  types: string[];
  priorities: string[];
  doneStatuses: string[];
  taskPrefix: string;
}

export interface FileSystemPort {
  exists(path: string): Promise<boolean>;
  readText(path: string): Promise<string>;
  listDirs(path: string): Promise<string[]>;
  listFiles(path: string): Promise<string[]>;
  writeText(path: string, content: string): Promise<void>;
  /** Absolute path of the vault root (""-relative path space maps onto it). */
  getVaultRootAbs(): Promise<string>;
  /** Nearest ancestor of the vault root (including the vault root itself, up
   *  to maxDepth levels) that contains `backlog.config.yml`, as an absolute
   *  path; null when none exists. Skillgrid layout: the repo root sits above
   *  the projectwiki vault. */
  findAncestorProjectRoot(maxDepth?: number): Promise<string | null>;
  /** Read a file by absolute path (outside the vault path space). */
  readTextAbs(absPath: string): Promise<string>;
}

export interface MilestoneRepository {
  list(project: ProjectConfig): Promise<Milestone[]>;
  write(project: ProjectConfig, m: Milestone, body: string): Promise<void>;
  create(project: ProjectConfig, m: Milestone, body: string): Promise<string>;
}

export interface VaultPort {
  getVaultRoot(): string;
}

export interface TaskRepository {
  list(project: ProjectConfig): Promise<import("../domain/task").Task[]>;
  get(project: ProjectConfig, id: string): Promise<import("../domain/task").Task | null>;
  write(project: ProjectConfig, task: import("../domain/task").Task, body: string): Promise<void>;
  create(project: ProjectConfig, task: import("../domain/task").Task, body: string): Promise<string>;
}