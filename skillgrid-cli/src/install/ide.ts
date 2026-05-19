import { copyFileSync, existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { homedir } from "node:os";
import { spawnSync } from "node:child_process";
import stripJsonComments from "strip-json-comments";
import {
  emitMcpForAntigravity,
  emitMcpForCursor,
  emitMcpForVscode,
  emitOpencodeStyleMcpObject,
  type McpServersShape,
} from "./mcp.js";
import { logInfo, logSuccess, logWarn } from "./log.js";
import { commandOnPath } from "./exec.js";
import {
  contextModeInMerged,
  ensureContextModePlugin,
  omitContextModeFromMcpRecord,
} from "./context-mode.js";

function ensureDir(dir: string, dryRun: boolean) {
  if (dryRun) {
    console.log(`[DRY-RUN] Would create directory: ${dir}`);
    return;
  }
  mkdirSync(dir, { recursive: true });
}

export function setupCursor(project: string, merged: McpServersShape | null, mergeMcp: boolean, dryRun: boolean) {
  const toolDir = join(project, ".cursor");
  const mcpFile = join(toolDir, "mcp.json");
  logInfo("Setting up Cursor configuration...");
  ensureDir(toolDir, dryRun);
  if (!mergeMcp || !merged) {
    logInfo("Skipping Cursor mcp.json (--no-mcp or no merged data)");
    logSuccess("Cursor setup complete");
    return;
  }
  const out = emitMcpForCursor(merged);
  const body = `${JSON.stringify(out, null, 2)}\n`;
  if (dryRun) {
    console.log(`[DRY-RUN] Would write ${mcpFile}`);
  } else {
    writeFileSync(mcpFile, body, "utf8");
    const count = Object.keys(out.mcpServers ?? {}).length;
    logSuccess(`Generated: ${mcpFile} (${count} server(s))`);
  }
  logSuccess("Cursor setup complete");
}

export function setupCopilot(project: string, merged: McpServersShape | null, mergeMcp: boolean, dryRun: boolean) {
  const toolDir = join(project, ".vscode");
  const mcpFile = join(toolDir, "mcp.json");
  logInfo("Setting up Copilot configuration...");
  ensureDir(toolDir, dryRun);
  if (!mergeMcp || !merged) {
    logInfo("Skipping Copilot mcp.json (--no-mcp or no merged data)");
    logSuccess("Copilot setup complete");
    return;
  }
  const out = emitMcpForVscode(merged);
  const body = `${JSON.stringify(out, null, 2)}\n`;
  if (dryRun) {
    console.log(`[DRY-RUN] Would write ${mcpFile}`);
  } else {
    writeFileSync(mcpFile, body, "utf8");
    const count = Object.keys(out.servers ?? {}).length;
    logSuccess(`Generated: ${mcpFile} (${count} server(s))`);
  }
  logSuccess("Copilot setup complete");
}

export function setupKilo(project: string, merged: McpServersShape | null, mergeMcp: boolean, dryRun: boolean) {
  const toolDir = join(project, ".kilo");
  let kiloCfg = join(toolDir, "kilo.jsonc");
  if (!existsSync(kiloCfg) && existsSync(join(toolDir, "kilo.json"))) {
    kiloCfg = join(toolDir, "kilo.json");
  } else if (!existsSync(kiloCfg)) {
    kiloCfg = join(toolDir, "kilo.jsonc");
  }

  logInfo("Setting up Kilocode configuration...");
  ensureDir(toolDir, dryRun);

  if (!dryRun && !commandOnPath("kilo")) {
    logInfo("Installing KiloCode CLI (@kilocode/cli)...");
    if (commandOnPath("npm")) {
      const r = spawnSync("npm", ["install", "-g", "@kilocode/cli"], { stdio: "inherit" });
      if (r.status === 0) logSuccess("KiloCode CLI installed successfully");
      else logWarn("KiloCode CLI installation failed — you may need: npm install -g @kilocode/cli");
    } else logWarn("npm not found — skipping KiloCode CLI installation");
  } else if (dryRun && !commandOnPath("kilo")) {
    console.log("[DRY-RUN] Would npm install -g @kilocode/cli (if kilo missing)");
  } else if (!dryRun) {
    logInfo("KiloCode CLI already installed");
  }

  if (!mergeMcp || !merged) {
    logInfo("Skipping Kilo MCP (--no-mcp or no merged data)");
    logSuccess("Kilocode setup complete");
    return;
  }

  let mcpObj = emitOpencodeStyleMcpObject(merged) as Record<string, unknown>;
  if (contextModeInMerged(merged, mergeMcp)) {
    mcpObj = omitContextModeFromMcpRecord(mcpObj);
  }
  if (dryRun) {
    console.log(`[DRY-RUN] Would merge MCP into ${kiloCfg}`);
    logSuccess("Kilocode setup complete");
    return;
  }

  let doc: Record<string, unknown>;
  if (existsSync(kiloCfg)) {
    const raw = readFileSync(kiloCfg, "utf8");
    try {
      doc = JSON.parse(stripJsonComments(raw)) as Record<string, unknown>;
    } catch {
      logWarn(`Could not parse ${kiloCfg} — replacing with MCP-only config`);
      doc = {};
    }
  } else {
    doc = {};
  }
  doc.mcp = mcpObj;
  if (contextModeInMerged(merged, mergeMcp)) {
    ensureContextModePlugin(doc);
  }
  writeFileSync(kiloCfg, `${JSON.stringify(doc, null, 2)}\n`, "utf8");
  const count = Object.keys(mcpObj).length;
  logSuccess(`Wrote Kilo MCP: ${kiloCfg} (${count} server(s); OpenCode-style local/remote)`);
  if (existsSync(join(toolDir, "mcp.json"))) {
    logInfo(`Note: legacy ${join(toolDir, "mcp.json")} exists — Kilo reads ${kiloCfg}; remove mcp.json if unused`);
  }
  logSuccess("Kilocode setup complete");
}

export function setupOpencode(hubRoot: string, project: string, merged: McpServersShape | null, mergeMcp: boolean, dryRun: boolean) {
  const toolDir = join(project, ".opencode");
  const mcpFile = join(toolDir, "opencode.json");
  const hubOpencode = join(hubRoot, ".configs", "opencode.json");

  ensureDir(toolDir, dryRun);

  if (existsSync(hubOpencode)) {
    if (dryRun) console.log(`[DRY-RUN] Would copy hub -> ${mcpFile}`);
    else {
      copyFileSync(hubOpencode, mcpFile);
      logSuccess(`Copied hub config -> ${mcpFile}`);
    }
  }

  if (!existsSync(mcpFile) && !dryRun) {
    logInfo(`Skipping opencode: ${mcpFile} missing (add .configs/opencode.json to hub or .opencode/)`);
    return;
  }
  if (dryRun && !existsSync(hubOpencode) && !existsSync(mcpFile)) {
    logInfo(`Skipping opencode: ${mcpFile} would be missing`);
    return;
  }

  if (mergeMcp && merged) {
    let opencodeMcp = emitOpencodeStyleMcpObject(merged) as Record<string, unknown>;
    if (contextModeInMerged(merged, mergeMcp)) {
      opencodeMcp = omitContextModeFromMcpRecord(opencodeMcp);
    }
    if (dryRun) {
      console.log(`[DRY-RUN] Would patch .mcp in ${mcpFile}`);
    } else {
      let doc: Record<string, unknown> | undefined;
      try {
        doc = JSON.parse(readFileSync(mcpFile, "utf8")) as Record<string, unknown>;
      } catch {
        logWarn(`Could not read ${mcpFile} — skipping MCP patch; still mirroring opencode.json to project root`);
      }
      if (doc) {
        doc.mcp = opencodeMcp;
        if (contextModeInMerged(merged, mergeMcp)) {
          ensureContextModePlugin(doc);
        }
        writeFileSync(mcpFile, `${JSON.stringify(doc, null, 2)}\n`, "utf8");
        const count = Object.keys(opencodeMcp).length;
        logSuccess(`Updated MCP config: ${mcpFile} (${count} server(s))`);
      }
    }
  } else {
    logInfo("Skipping opencode MCP (--no-mcp or no merged data)");
  }

  /* install.sh always mirrors .opencode/opencode.json → project root after MCP step (when file exists). */
  if (dryRun) {
    if (existsSync(mcpFile)) console.log(`[DRY-RUN] Would copy ${mcpFile} -> ${join(project, "opencode.json")}`);
    return;
  }
  if (!existsSync(mcpFile)) return;
  const rootOpencode = join(project, "opencode.json");
  copyFileSync(mcpFile, rootOpencode);
  logSuccess("Wrote project root opencode.json (same content as .opencode/opencode.json)");
}

export function setupAntigravity(_project: string, merged: McpServersShape | null, mergeMcp: boolean, dryRun: boolean) {
  const mcpDir = join(homedir(), ".gemini", "antigravity");
  const mcpFile = join(mcpDir, "mcp_config.json");

  logInfo("Setting up Google Antigravity configuration...");
  ensureDir(mcpDir, dryRun);

  if (!mergeMcp || !merged) {
    logInfo("Skipping Antigravity mcp_config.json (--no-mcp or no merged data)");
    logSuccess("Antigravity setup complete");
    return;
  }

  const out = emitMcpForAntigravity(merged);
  const body = `${JSON.stringify(out, null, 2)}\n`;
  if (dryRun) {
    console.log(`[DRY-RUN] Would write ${mcpFile}`);
  } else {
    writeFileSync(mcpFile, body, "utf8");
    const count = Object.keys(out.mcpServers ?? {}).length;
    logSuccess(`Generated: ${mcpFile} (${count} server(s); Antigravity transport shape)`);
  }
  logSuccess("Antigravity setup complete");
}

export function verifyEngramSetup(hubRoot: string, merged: McpServersShape | null, mergeMcp: boolean) {
  if (!mergeMcp) {
    logInfo("Engram MCP: skipped because --no-mcp was used");
    return;
  }
  const fragment = join(hubRoot, ".configs", "mcp", "command", "engram.json");
  if (!existsSync(fragment)) {
    logWarn("Engram MCP: missing fragment .configs/mcp/command/engram.json");
    return;
  }
  if (commandOnPath("engram")) {
    const which = spawnSync("command", ["-v", "engram"], { shell: true, encoding: "utf8" }).stdout?.trim();
    logSuccess(`Engram CLI available: ${which || "engram"}`);
  } else logWarn("Engram CLI not on PATH — install with: brew install gentleman-programming/tap/engram");
  if (!merged || Object.keys(merged.mcpServers ?? {}).length === 0) {
    logWarn("Engram MCP: no merged MCP config was produced");
    return;
  }
  const engramEntry = merged.mcpServers?.engram;
  if (engramEntry !== undefined && engramEntry !== null) {
    logSuccess("Engram MCP server included in merged config");
  } else {
    logWarn("Engram MCP server not included in merged config — select it during MCP setup or use all MCP servers");
  }
}
