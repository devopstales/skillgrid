import { existsSync, mkdirSync, copyFileSync, rmSync } from "node:fs";
import { resolve, join } from "node:path";
import { spawnSync } from "node:child_process";
import * as readline from "node:readline/promises";
import { stdin as input, stdout as output } from "node:process";
import type { IdeId, InstallOptions, OptionalToolId } from "./types.js";
import { INSTALL_VERSION } from "./types.js";
import { collectMcpMergePaths, filterMcpByKeys, mergeMcpJsonFiles, type McpServersShape } from "./mcp.js";
import { rsyncDelete, rsyncDir, rsyncPlain } from "./rsync.js";
import {
  setupAntigravity,
  setupCopilot,
  setupCursor,
  setupKilo,
  setupOpencode,
  verifyEngramSetup,
} from "./ide.js";
import { setupContextModeAssets, verifyContextModeSetup } from "./context-mode.js";
import { installAgentClis } from "./agent-tools.js";
import { ensureTrivyMcpPlugin, installOptionalToolClis } from "./optional-tools.js";
import { countMissingDeps, installDependencyPackages, showDependencies } from "./deps.js";
import { toolIsSelected } from "./optional-tools-helpers.js";
import { logInfo, logSuccess, logWarn } from "./log.js";
import {
  interactiveAgentsSelection,
  interactiveIdeSelection,
  interactiveMcpSelection,
  interactiveToolsSelection,
} from "./interactive.js";
import type { ParsedInstallArgv } from "./parse-install-argv.js";
import { ensureReleaseHubCache } from "./clone-release-hub.js";
import { installSddGateHooks, uninstallSddGateHooks } from "./sdd-gate-hooks.js";

function ideFolder(ide: IdeId): string {
  switch (ide) {
    case "cursor":
      return ".cursor";
    case "copilot":
      return ".vscode";
    case "kilo":
      return ".kilo";
    case "opencode":
      return ".opencode";
    case "antigravity":
      return ".agents";
    default:
      return "";
  }
}

function validateHub(hubRoot: string): void {
  const marker = join(hubRoot, "install.sh");
  if (!existsSync(marker)) {
    throw new Error(`Hub root does not look like AISkillgrid (missing install.sh): ${hubRoot}`);
  }
}

function computeMergedMcp(hubRoot: string, mergeMcp: boolean, filterKeys: string[] | null): McpServersShape | null {
  if (!mergeMcp) return null;
  const paths = collectMcpMergePaths(hubRoot);
  if (paths.length === 0) return null;
  let m = mergeMcpJsonFiles(paths);
  if (filterKeys?.length) m = filterMcpByKeys(m, filterKeys);
  return m;
}

async function maybePromptInstallDeps(
  selectedIdes: IdeId[],
  allIdes: boolean,
  selectedTools: OptionalToolId[],
  nonInteractive: boolean,
  dryRun: boolean,
  mergeMcp: boolean,
  mcpKeyFilter: string[] | null,
): Promise<void> {
  const { brew, pip, npm } = countMissingDeps(selectedIdes, allIdes, selectedTools, mergeMcp, mcpKeyFilter);
  const missing = brew.length + pip.length + npm.length;
  if (missing === 0) {
    console.log("All dependencies are installed!");
    console.log("");
    return;
  }
  console.log(`Found ${missing} missing dependencies.`);
  if (nonInteractive) {
    console.log("Continuing without installing dependencies (non-interactive mode)...");
    console.log("");
    return;
  }
  const rl = readline.createInterface({ input, output });
  try {
    const ans = (await rl.question("Would you like to install missing dependencies? [y/N] ")).trim().toLowerCase();
    console.log("");
    if (ans === "y" || ans === "yes") {
      installDependencyPackages(brew, pip, npm, dryRun);
      ensureTrivyMcpPlugin(mergeMcp, mcpKeyFilter, dryRun);
    } else {
      console.log("Continuing without installing dependencies...");
      console.log("");
    }
  } finally {
    rl.close();
  }
}

async function maybePromptInstallDepsForDepsFlag(
  selectedIdes: IdeId[],
  allIdes: boolean,
  selectedTools: OptionalToolId[],
  dryRun: boolean,
  mergeMcp: boolean,
  mcpKeyFilter: string[] | null,
): Promise<void> {
  const { brew, pip, npm } = countMissingDeps(selectedIdes, allIdes, selectedTools, mergeMcp, mcpKeyFilter);
  const missing = brew.length + pip.length + npm.length;
  if (missing === 0) return;
  console.log(`Found ${missing} missing dependencies.`);
  const rl = readline.createInterface({ input, output });
  try {
    const ans = (await rl.question("Would you like to install missing dependencies? [y/N] ")).trim().toLowerCase();
    console.log("");
    if (ans === "y" || ans === "yes") {
      installDependencyPackages(brew, pip, npm, dryRun);
      ensureTrivyMcpPlugin(mergeMcp, mcpKeyFilter, dryRun);
    }
  } finally {
    rl.close();
  }
}

function uninstall(opts: InstallOptions): void {
  const { projectPath, selectedIdes, dryRun, selectedTools } = opts;
  console.log(`Uninstalling AI config folders from: ${projectPath}`);
  console.log("");

  for (const ide of selectedIdes) {
    const folder = ideFolder(ide);
    const target = join(projectPath, folder);
    if (existsSync(target)) {
      if (dryRun) console.log(`[DRY-RUN] Would remove: ${folder}`);
      else {
        console.log(`Removing: ${folder}`);
        rmSync(target, { recursive: true, force: true });
      }
    } else {
      console.log(`Skipping: ${folder} (not found)`);
    }
  }

  if (selectedIdes.includes("opencode")) {
    const rootOc = join(projectPath, "opencode.json");
    if (existsSync(rootOc)) {
      if (dryRun) console.log("[DRY-RUN] Would remove: opencode.json (project root)");
      else {
        console.log("Removing: opencode.json (project root)");
        rmSync(rootOc, { force: true });
      }
    }
  }

  if (toolIsSelected(selectedTools, "openspec")) {
    const target = join(projectPath, "openspec");
    if (existsSync(target)) {
      if (dryRun) console.log("[DRY-RUN] Would remove: openspec");
      else {
        console.log("Removing: openspec");
        rmSync(target, { recursive: true, force: true });
      }
    }
  }

  if (toolIsSelected(selectedTools, "gitnexus")) {
    const target = join(projectPath, ".gitnexus");
    if (existsSync(target)) {
      if (dryRun) console.log("[DRY-RUN] Would remove: .gitnexus");
      else {
        console.log("Removing: .gitnexus");
        rmSync(target, { recursive: true, force: true });
      }
    }
  }

  if (selectedIdes.includes("copilot")) {
    for (const extra of [".github", ".copilot"]) {
      const target = join(projectPath, extra);
      if (existsSync(target)) {
        if (dryRun) console.log(`[DRY-RUN] Would remove: ${extra}`);
        else {
          console.log(`Removing: ${extra}`);
          rmSync(target, { recursive: true, force: true });
        }
      }
    }
  }

  console.log("");
  console.log("Done!");
}

export async function runInstallCli(localHubFallback: string, parsed: ParsedInstallArgv): Promise<number> {
  if (parsed.help) {
    printInstallHelp();
    return 0;
  }
  if (parsed.version) {
    console.log(`skillgrid install (native) version ${INSTALL_VERSION}`);
    return 0;
  }

  let hubRoot: string;
  if (!parsed.uninstall) {
    hubRoot = ensureReleaseHubCache().hubRoot;
    validateHub(hubRoot);
  } else {
    hubRoot = localHubFallback;
  }

  if (parsed.sanityCheck) {
    const { runSanityCheck } = await import("./sanity.js");
    return runSanityCheck(hubRoot);
  }

  let selectedIdes = [...parsed.selectedIdes];
  let allIdes = parsed.allIdes;
  const idePick = await interactiveIdeSelection(parsed.nonInteractive, selectedIdes);
  selectedIdes = idePick.ides;
  allIdes = idePick.allIdes || allIdes;

  if (selectedIdes.length === 0) {
    selectedIdes = ["cursor", "copilot", "kilo", "opencode", "antigravity"];
    allIdes = true;
  }

  let mergeMcp = parsed.mergeMcp;
  let mcpFilter = parsed.mcpKeyFilter;
  const mcpPick = await interactiveMcpSelection(hubRoot, parsed.nonInteractive, mergeMcp, parsed.allMcp);
  mergeMcp = mcpPick.mergeMcp;
  mcpFilter = mcpPick.filterKeys ?? mcpFilter;

  let selectedTools = [...parsed.selectedTools];
  selectedTools = [...selectedTools, ...(await interactiveToolsSelection(parsed.nonInteractive, parsed.toolsInteractive))];

  const selectedAgents = await interactiveAgentsSelection(parsed.nonInteractive, parsed.agentsInteractive);

  if (parsed.checkDeps) {
    showDependencies(selectedIdes, allIdes, selectedTools, mergeMcp, mcpFilter);
    await maybePromptInstallDepsForDepsFlag(selectedIdes, allIdes, selectedTools, parsed.dryRun, mergeMcp, mcpFilter);
    if (!parsed.projectPath) {
      return 0;
    }
  }

  if (!parsed.projectPath) {
    console.error("Error: Project path is required.");
    console.error("");
    printInstallHelp();
    return 1;
  }

  const projectPath = resolve(parsed.projectPath);
  if (!existsSync(projectPath)) {
    console.error(`Error: Directory '${parsed.projectPath}' does not exist`);
    return 1;
  }

  const opts: InstallOptions = {
    projectPath,
    hubRoot,
    selectedIdes,
    allIdes,
    selectedTools,
    toolsInteractive: parsed.toolsInteractive,
    selectedAgents,
    agentsInteractive: parsed.agentsInteractive,
    dryRun: parsed.dryRun,
    uninstall: parsed.uninstall,
    checkDeps: parsed.checkDeps,
    sanityCheck: parsed.sanityCheck,
    nonInteractive: parsed.nonInteractive,
    mergeMcp,
    mcpKeyFilter: mcpFilter,
    installSddHooks: parsed.installSddHooks,
  };

  if (opts.uninstall) {
    if (opts.installSddHooks) {
      uninstallSddGateHooks(opts.projectPath, opts.dryRun);
    }
    uninstall(opts);
    return 0;
  }

  if (!parsed.checkDeps) {
    console.log("Checking dependencies...");
    console.log("");
    await maybePromptInstallDeps(selectedIdes, allIdes, selectedTools, parsed.nonInteractive, parsed.dryRun, mergeMcp, mcpFilter);
  }

  installOptionalToolClis(hubRoot, [...opts.selectedTools], opts.dryRun);
  installAgentClis([...opts.selectedAgents], opts.dryRun);
  ensureTrivyMcpPlugin(mergeMcp, mcpFilter, opts.dryRun);

  if (opts.dryRun) {
    console.log("=== DRY RUN MODE ===");
    console.log("");
  }

  console.log(`Installing AI config folders to: ${opts.projectPath}`);
  console.log("");

  for (const ide of opts.selectedIdes) {
    const folder = ideFolder(ide);
    const src = join(hubRoot, folder);
    const dst = join(opts.projectPath, folder);
    if (!existsSync(src)) {
      console.log(`Skipping: ${folder} (not found in source)`);
      continue;
    }
    if (existsSync(dst)) {
      console.log(opts.dryRun ? `[DRY-RUN] Would update: ${folder}` : `Updating: ${folder}`);
    } else {
      console.log(opts.dryRun ? `[DRY-RUN] Would create: ${folder}` : `Creating: ${folder}`);
    }
    rsyncDir(src, dst, opts.dryRun);
  }

  const hubAgents = join(hubRoot, ".configs", "AGENTS.md");
  if (existsSync(hubAgents) && opts.selectedIdes.length > 0) {
    console.log("");
    if (opts.dryRun) {
      console.log(`[DRY-RUN] Would copy .configs/AGENTS.md -> ${join(opts.projectPath, "AGENTS.md")}`);
      for (const ide of opts.selectedIdes) {
        console.log(`[DRY-RUN] Would copy .configs/AGENTS.md -> ${join(opts.projectPath, ideFolder(ide), "AGENTS.md")}`);
      }
    } else {
      console.log("Copying AGENTS.md from hub (.configs/AGENTS.md)...");
      copyFileSync(hubAgents, join(opts.projectPath, "AGENTS.md"));
      logSuccess("Wrote AGENTS.md (project root)");
      for (const ide of opts.selectedIdes) {
        const dstDir = join(opts.projectPath, ideFolder(ide));
        mkdirSync(dstDir, { recursive: true });
        copyFileSync(hubAgents, join(dstDir, "AGENTS.md"));
        logSuccess(`Wrote ${ideFolder(ide)}/AGENTS.md`);
      }
    }
  } else if (!existsSync(hubAgents) && opts.selectedIdes.length > 0) {
    logInfo(".configs/AGENTS.md not found in hub — skipping AGENTS.md copy");
  }

  const hubSkillgrid = join(hubRoot, ".skillgrid");
  if (existsSync(hubSkillgrid)) {
    console.log("");
    const dstSkillgrid = join(opts.projectPath, ".skillgrid");
    if (opts.dryRun) {
      console.log(existsSync(dstSkillgrid) ? "[DRY-RUN] Would update: .skillgrid" : "[DRY-RUN] Would create: .skillgrid");
    } else {
      console.log(existsSync(dstSkillgrid) ? "Updating: .skillgrid" : "Creating: .skillgrid");
      rsyncPlain(hubSkillgrid, dstSkillgrid, opts.dryRun);
      if (!opts.dryRun) logSuccess("Synced .skillgrid/ (hub templates → project)");
    }
  } else {
    logInfo(".skillgrid/ not found in hub — skipping");
  }

  const hubAgentsRoot = join(hubRoot, ".agents");
  if (existsSync(hubAgentsRoot)) {
    console.log("");
    const dstAgents = join(opts.projectPath, ".agents");
    if (opts.dryRun) {
      console.log(existsSync(dstAgents) ? "[DRY-RUN] Would update: .agents (rules, workflows, skills)" : "[DRY-RUN] Would create: .agents (rules, workflows, skills)");
    } else {
      console.log(existsSync(dstAgents) ? "Updating: .agents (rules, workflows, skills)" : "Creating: .agents (rules, workflows, skills)");
      mkdirSync(dstAgents, { recursive: true });
      rsyncDelete(hubAgentsRoot, dstAgents, opts.dryRun);
      if (!opts.dryRun) logSuccess("Synced .agents/ (hub → project root)");
    }
  } else {
    logInfo(".agents/ not found in hub — skipping root .agents sync");
  }

  const hubAgentSingular = join(hubRoot, ".agent");
  if (opts.selectedIdes.includes("antigravity")) {
    if (existsSync(hubAgentSingular)) {
      console.log("");
      const dstAgent = join(opts.projectPath, ".agent");
      if (opts.dryRun) {
        console.log(
          existsSync(dstAgent)
            ? "[DRY-RUN] Would update: .agent (Antigravity / Gemini-style hub mirror)"
            : "[DRY-RUN] Would create: .agent (Antigravity / Gemini-style hub mirror)",
        );
      } else {
        console.log(
          existsSync(dstAgent)
            ? "Updating: .agent (Antigravity / Gemini-style hub mirror)"
            : "Creating: .agent (Antigravity / Gemini-style hub mirror)",
        );
        mkdirSync(dstAgent, { recursive: true });
        rsyncDelete(hubAgentSingular, dstAgent, opts.dryRun);
        if (!opts.dryRun) logSuccess("Synced .agent/ (hub → project root)");
      }
    } else {
      logInfo(".agent/ not found in hub — skipping (Antigravity selected but no hub .agent/)");
    }
  } else if (existsSync(hubAgentSingular)) {
    logInfo(".agent/: skipped — hub has .agent/ but Antigravity is not selected (-a or -A); Gemini CLI TBD");
  }

  console.log("");
  console.log("Syncing skills configurations...");
  const skillsSrc = join(hubRoot, ".agents", "skills");
  if (existsSync(skillsSrc)) {
    for (const ide of opts.selectedIdes) {
      if (ide === "antigravity") {
        const target = join(opts.projectPath, ".agents", "skills");
        if (opts.dryRun) console.log("[DRY-RUN] Would sync skills to: .agents/skills");
        else {
          mkdirSync(target, { recursive: true });
          console.log("Syncing skills to: .agents/skills");
          rsyncDelete(skillsSrc, target, opts.dryRun);
        }
      } else if (ide === "cursor") {
        const target = join(opts.projectPath, ".cursor", ".agents", "skills");
        if (opts.dryRun) console.log("[DRY-RUN] Would sync skills to: .cursor/.agents/skills");
        else {
          mkdirSync(target, { recursive: true });
          console.log("Syncing skills to: .cursor/.agents/skills");
          rsyncDelete(skillsSrc, target, opts.dryRun);
        }
      } else if (ide === "kilo") {
        const target = join(opts.projectPath, ".kilo", "skills");
        if (opts.dryRun) console.log("[DRY-RUN] Would sync skills to: .kilo/skills");
        else {
          mkdirSync(target, { recursive: true });
          console.log("Syncing skills to: .kilo/skills");
          rsyncDelete(skillsSrc, target, opts.dryRun);
        }
      } else if (ide === "opencode") {
        const target = join(opts.projectPath, ".opencode", "skills");
        if (opts.dryRun) console.log("[DRY-RUN] Would sync skills to: .opencode/skills");
        else {
          mkdirSync(target, { recursive: true });
          console.log("Syncing skills to: .opencode/skills");
          rsyncDelete(skillsSrc, target, opts.dryRun);
        }
      }
    }
  } else {
    logInfo(`Skills source not found: ${skillsSrc} — skipping skills sync`);
  }

  if (opts.installSddHooks) {
    console.log("");
    installSddGateHooks(opts.projectPath, opts.dryRun);
  }

  if (opts.selectedIdes.includes("copilot")) {
    for (const extra of [".github", ".copilot"]) {
      const src = join(hubRoot, extra);
      const dst = join(opts.projectPath, extra);
      if (!existsSync(src)) {
        console.log(`Skipping: ${extra} (not found in source)`);
        continue;
      }
      if (existsSync(dst)) {
        console.log(opts.dryRun ? `[DRY-RUN] Would update: ${extra}` : `Updating: ${extra}`);
      } else {
        console.log(opts.dryRun ? `[DRY-RUN] Would create: ${extra}` : `Creating: ${extra}`);
      }
      rsyncDelete(src, dst, opts.dryRun);
    }
  }

  console.log("");
  console.log("Merging MCP configurations...");
  const merged = computeMergedMcp(hubRoot, opts.mergeMcp, opts.mcpKeyFilter);
  verifyEngramSetup(hubRoot, merged, opts.mergeMcp);
  verifyContextModeSetup(merged, opts.mergeMcp);

  console.log("");
  console.log("Setting up IDE configurations...");
  for (const ide of opts.selectedIdes) {
    if (opts.dryRun) {
      console.log(`[DRY-RUN] Would setup: ${ide}`);
      continue;
    }
    switch (ide) {
      case "cursor":
        setupCursor(opts.projectPath, merged, opts.mergeMcp, false);
        break;
      case "copilot":
        setupCopilot(opts.projectPath, merged, opts.mergeMcp, false);
        break;
      case "kilo":
        setupKilo(opts.projectPath, merged, opts.mergeMcp, false);
        break;
      case "opencode":
        setupOpencode(hubRoot, opts.projectPath, merged, opts.mergeMcp, false);
        break;
      case "antigravity":
        setupAntigravity(opts.projectPath, merged, opts.mergeMcp, false);
        break;
      default:
        break;
    }
  }

  console.log("");
  logInfo("context-mode integration...");
  setupContextModeAssets(hubRoot, opts.projectPath, opts.selectedIdes, opts.mergeMcp, merged, opts.dryRun);

  if (toolIsSelected(opts.selectedTools, "openspec") && existsSync(join(hubRoot, "openspec"))) {
    const target = join(opts.projectPath, "openspec");
    if (existsSync(target)) {
      console.log(opts.dryRun ? "[DRY-RUN] Would update: openspec" : "Updating: openspec");
    } else {
      console.log(opts.dryRun ? "[DRY-RUN] Would create: openspec" : "Creating: openspec");
    }
    rsyncDir(join(hubRoot, "openspec"), target, opts.dryRun);
  } else if (toolIsSelected(opts.selectedTools, "openspec")) {
    logInfo("openspec: hub has no openspec/ — skip copy");
  }

  if (toolIsSelected(opts.selectedTools, "openspec")) {
    console.log("");
    if (opts.dryRun) {
      console.log(`[DRY-RUN] Would run: openspec init in ${join(opts.projectPath, "openspec")}`);
    } else if (existsSync(join(opts.projectPath, "openspec"))) {
      console.log("Running openspec init...");
      spawnSync("openspec", ["init"], { cwd: join(opts.projectPath, "openspec"), stdio: "inherit", shell: process.platform === "win32" });
    }
  }

  console.log("");
  if (opts.dryRun) {
    console.log("=== DRY RUN COMPLETE ===");
    console.log("(No changes were made)");
  } else {
    console.log(`Done! IDE config folders have been copied to ${opts.projectPath}`);
  }

  return 0;
}

function printInstallHelp() {
  console.log(`Usage:
  skillgrid install [OPTIONS]

Hub source (install / sanity-check):
  Shared shallow clone at /tmp/skillgrid-aiskillgrid-release (Unix/WSL) or %TEMP%/skillgrid-aiskillgrid-release (native Windows): first run clones devopstales/aiskillgrid release/2; later runs git fetch + reset. Rsync copies from that cache (requires git + network). Prefer WSL on Windows. Uninstall (-u) uses the local CLI hub path only and does not touch the cache.

Options:
  -p, --path <dir>      Install to a specific project path
  -c, --cursor          Setup configuration for Cursor
  -C, --copilot         Setup configuration for Copilot
  -k, --kilo            Setup configuration for Kilocode
  -o, --opencode        Setup configuration for opencode
  -a, --antigravity     Setup configuration for Google Antigravity
  -A, --all, --all-ides Setup for all supported IDEs (default if none selected)
  -AA, --all-mcp        Merge every hub MCP server (skip MCP prompt; respects later --no-mcp)
  -t, --tools           Interactive optional tools (openspec, dmux, brave-search-cli, cocoindex-code, truecourse); gitnexus, engram, context-mode always
  -g, --agents          Interactive agent CLIs (Claude Code, OpenCode, kilo, Codex, Gemini CLI, pi)
  -d, --deps            Check and install dependencies before install
  --sanity-check        Verify hub dependencies and expected files (read-only)
  -y, --yes             Non-interactive mode (skip prompts)
  --no-mcp              Skip MCP server configuration
  -n, --dry-run         Show what would be installed without making changes
  --no-sdd-hooks        Skip SDD gate pre-commit/pre-push git hooks
  -u, --uninstall       Remove managed IDE dirs from target
  -v, --version         Print version
  -h, --help            Show help`);
}
