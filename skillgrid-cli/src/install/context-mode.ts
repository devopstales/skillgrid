import { copyFileSync, existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import type { IdeId } from "./types.js";
import type { McpServersShape } from "./mcp.js";
import { logInfo, logSuccess, logWarn } from "./log.js";
import { commandOnPath } from "./exec.js";

const CONTEXT_MODE_KEY = "context-mode";

export function contextModeInMerged(merged: McpServersShape | null, mergeMcp: boolean): boolean {
  if (!mergeMcp || !merged?.mcpServers) return false;
  return Object.prototype.hasOwnProperty.call(merged.mcpServers, CONTEXT_MODE_KEY);
}

export function installContextModeCli(dryRun: boolean): void {
  if (commandOnPath("context-mode")) {
    logInfo("context-mode CLI already on PATH");
    return;
  }
  if (dryRun) {
    console.log("[DRY-RUN] npm install -g context-mode");
    return;
  }
  if (!commandOnPath("npm")) {
    logWarn("context-mode: npm not found — install Node.js, then run: npm install -g context-mode");
    return;
  }
  logInfo("Installing context-mode (npm install -g context-mode)...");
  const r = spawnSync("npm", ["install", "-g", "context-mode"], { stdio: "inherit" });
  if (r.status === 0) logSuccess("context-mode installed");
  else logWarn("context-mode: npm install -g failed — see https://github.com/mksglu/context-mode");
}

/** OpenCode/Kilo use the in-process plugin; duplicate MCP registration breaks ctx_* tools. */
export function omitContextModeFromMcpRecord<T extends Record<string, unknown>>(mcp: T): T {
  if (!Object.prototype.hasOwnProperty.call(mcp, CONTEXT_MODE_KEY)) return mcp;
  const next = { ...mcp };
  delete next[CONTEXT_MODE_KEY];
  return next;
}

export function ensureContextModePlugin(doc: Record<string, unknown>): void {
  const plugins = doc.plugin;
  if (Array.isArray(plugins)) {
    if (!plugins.includes(CONTEXT_MODE_KEY)) plugins.push(CONTEXT_MODE_KEY);
    return;
  }
  doc.plugin = [CONTEXT_MODE_KEY];
}

function hubContextModeDir(hubRoot: string, ...parts: string[]): string {
  return join(hubRoot, ".configs", "context-mode", ...parts);
}

function copyIfMissing(src: string, dest: string, dryRun: boolean, label: string): void {
  if (!existsSync(src)) {
    logWarn(`context-mode: missing hub template ${src}`);
    return;
  }
  if (existsSync(dest)) {
    logInfo(`context-mode: ${label} already exists — left unchanged (${dest})`);
    return;
  }
  if (dryRun) {
    console.log(`[DRY-RUN] Would copy ${label} -> ${dest}`);
    return;
  }
  mkdirSync(join(dest, ".."), { recursive: true });
  copyFileSync(src, dest);
  logSuccess(`context-mode: installed ${label}`);
}

export function setupContextModeAssets(
  hubRoot: string,
  project: string,
  selectedIdes: IdeId[],
  mergeMcp: boolean,
  merged: McpServersShape | null,
  dryRun: boolean,
): void {
  if (!contextModeInMerged(merged, mergeMcp)) {
    logInfo("context-mode assets: skipped (not in merged MCP selection)");
    return;
  }

  if (selectedIdes.includes("cursor")) {
    const rulesDir = join(project, ".cursor", "rules");
    copyIfMissing(
      hubContextModeDir(hubRoot, "cursor", "context-mode.mdc"),
      join(rulesDir, "context-mode.mdc"),
      dryRun,
      "Cursor routing rules",
    );
    copyIfMissing(
      hubContextModeDir(hubRoot, "cursor", "hooks.json"),
      join(project, ".cursor", "hooks.json"),
      dryRun,
      "Cursor hooks",
    );
  }

  if (selectedIdes.includes("copilot")) {
    const hooksDest = join(project, ".github", "hooks", "context-mode.json");
    const hooksSrc = hubContextModeDir(hubRoot, "copilot", "hooks.json");
    if (!existsSync(hooksSrc)) {
      logWarn(`context-mode: missing hub template ${hooksSrc}`);
    } else if (dryRun) {
      console.log(`[DRY-RUN] Would write Copilot hooks -> ${hooksDest}`);
    } else {
      mkdirSync(join(project, ".github", "hooks"), { recursive: true });
      copyFileSync(hooksSrc, hooksDest);
      logSuccess(`context-mode: installed Copilot hooks (${hooksDest})`);
    }
  }
}

export function verifyContextModeSetup(merged: McpServersShape | null, mergeMcp: boolean): void {
  if (!mergeMcp) {
    logInfo("context-mode: skipped because --no-mcp was used");
    return;
  }
  if (commandOnPath("context-mode")) {
    const which = spawnSync("command", ["-v", "context-mode"], { shell: true, encoding: "utf8" }).stdout?.trim();
    logSuccess(`context-mode CLI available: ${which || "context-mode"}`);
  } else {
    logWarn("context-mode CLI not on PATH — install with: npm install -g context-mode");
  }
  if (!contextModeInMerged(merged, mergeMcp)) {
    logWarn("context-mode MCP server not in merged config — select it during MCP setup or use all MCP servers");
    return;
  }
  logSuccess("context-mode MCP server included in merged config");
}

/** Patch opencode/kilo JSON on disk after MCP merge (plugin path, no duplicate MCP). */
export function patchJsonFileForContextModePlugin(filePath: string, dryRun: boolean): void {
  if (!existsSync(filePath)) return;
  let doc: Record<string, unknown>;
  try {
    doc = JSON.parse(readFileSync(filePath, "utf8")) as Record<string, unknown>;
  } catch {
    logWarn(`context-mode: could not parse ${filePath} for plugin merge`);
    return;
  }
  ensureContextModePlugin(doc);
  if (doc.mcp && typeof doc.mcp === "object" && !Array.isArray(doc.mcp)) {
    doc.mcp = omitContextModeFromMcpRecord(doc.mcp as Record<string, unknown>);
  }
  if (dryRun) {
    console.log(`[DRY-RUN] Would add context-mode plugin to ${filePath}`);
    return;
  }
  writeFileSync(filePath, `${JSON.stringify(doc, null, 2)}\n`, "utf8");
  logSuccess(`context-mode: OpenCode/Kilo plugin registered in ${filePath}`);
}
