export interface ProjectConfig {
  name: string;
  rootPath: string;
  backlogDirectory: string;
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