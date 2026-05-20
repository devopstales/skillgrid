export type IdeId = "cursor" | "copilot" | "kilo" | "opencode" | "antigravity";

export type OptionalToolId =
  | "openspec"
  | "dmux"
  | "brave-search-cli"
  | "cocoindex-code"
  | "truecourse"
  | "gitnexus"
  | "engram"
  | "context-mode";

/** Coding-agent CLIs selectable via -g / --agents (install.sh parity). */
export type AgentId = "claude-code" | "opencode" | "kilo" | "codex" | "gemini" | "pi";

export interface InstallOptions {
  projectPath: string;
  hubRoot: string;
  selectedIdes: IdeId[];
  allIdes: boolean;
  selectedTools: OptionalToolId[];
  toolsInteractive: boolean;
  selectedAgents: AgentId[];
  agentsInteractive: boolean;
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
