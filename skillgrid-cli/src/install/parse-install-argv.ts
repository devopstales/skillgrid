import type { IdeId, OptionalToolId } from "./types.js";

export interface ParsedInstallArgv {
  projectPath: string | null;
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
  /** Same as install.sh -AA / --all-mcp: skip MCP prompt, merge every hub server (still respects later --no-mcp). */
  allMcp: boolean;
  mcpKeyFilter: string[] | null;
  installSddHooks: boolean;
  help: boolean;
  version: boolean;
}

export function parseInstallArgv(argv: string[]): ParsedInstallArgv {
  const out: ParsedInstallArgv = {
    projectPath: null,
    selectedIdes: [],
    allIdes: false,
    selectedTools: [],
    toolsInteractive: false,
    dryRun: false,
    uninstall: false,
    checkDeps: false,
    sanityCheck: false,
    nonInteractive: false,
    mergeMcp: true,
    allMcp: false,
    mcpKeyFilter: null,
    installSddHooks: true,
    help: false,
    version: false,
  };

  let i = 0;
  while (i < argv.length) {
    const a = argv[i];
    if (a === "-p" || a === "--path") {
      out.projectPath = argv[i + 1] ?? null;
      i += 2;
      continue;
    }
    if (a === "-c" || a === "--cursor") {
      out.selectedIdes.push("cursor");
      i += 1;
      continue;
    }
    if (a === "-C" || a === "--copilot") {
      out.selectedIdes.push("copilot");
      i += 1;
      continue;
    }
    if (a === "-k" || a === "--kilo") {
      out.selectedIdes.push("kilo");
      i += 1;
      continue;
    }
    if (a === "-o" || a === "--opencode") {
      out.selectedIdes.push("opencode");
      i += 1;
      continue;
    }
    if (a === "-a" || a === "--antigravity") {
      out.selectedIdes.push("antigravity");
      i += 1;
      continue;
    }
    if (a === "-AA" || a === "--all-mcp") {
      out.allMcp = true;
      out.mergeMcp = true;
      out.mcpKeyFilter = null;
      i += 1;
      continue;
    }
    if (a === "-A" || a === "--all" || a === "--all-ides") {
      out.selectedIdes = ["cursor", "copilot", "kilo", "opencode", "antigravity"];
      out.allIdes = true;
      i += 1;
      continue;
    }
    if (a === "-t" || a === "--tools") {
      out.toolsInteractive = true;
      i += 1;
      continue;
    }
    if (a === "-d" || a === "--deps") {
      out.checkDeps = true;
      i += 1;
      continue;
    }
    if (a === "--sanity-check") {
      out.sanityCheck = true;
      out.nonInteractive = true;
      i += 1;
      continue;
    }
    if (a === "-y" || a === "--yes") {
      out.nonInteractive = true;
      i += 1;
      continue;
    }
    if (a === "--no-mcp") {
      out.mergeMcp = false;
      i += 1;
      continue;
    }
    if (a === "--no-sdd-hooks") {
      out.installSddHooks = false;
      i += 1;
      continue;
    }
    if (a === "-n" || a === "--dry-run") {
      out.dryRun = true;
      i += 1;
      continue;
    }
    if (a === "-u" || a === "--uninstall") {
      out.uninstall = true;
      i += 1;
      continue;
    }
    if (a === "-v" || a === "--version") {
      out.version = true;
      i += 1;
      continue;
    }
    if (a === "-h" || a === "--help") {
      out.help = true;
      i += 1;
      continue;
    }
    throw new Error(`Unknown option: ${a}`);
  }

  return out;
}
