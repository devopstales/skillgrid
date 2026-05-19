export type IdeId = "cursor" | "copilot" | "kilo" | "opencode" | "antigravity";

export type OptionalToolId =
  | "openspec"
  | "dmux"
  | "brave-search-cli"
  | "cocoindex-code"
  | "gitnexus"
  | "engram";

export interface InstallOptions {
  projectPath: string;
  hubRoot: string;
  selectedIdes: IdeId[];
  allIdes: boolean;
  selectedTools: OptionalToolId[];
  toolsInteractive: boolean;
  dryRun: boolean;
  uninstall: boolean;
  checkDeps: boolean;
  sanityCheck: boolean;
  nonInteractive: boolean;
  mergeMcp: boolean;
  mcpKeyFilter: string[] | null;
  /** When true (default), install sdd-gate pre-commit/pre-push hooks after .skillgrid sync. */
  installSddHooks: boolean;
}

export const INSTALL_VERSION = "1.0.0";
