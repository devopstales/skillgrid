import * as readline from "node:readline/promises";
import { stdin as input, stdout as output } from "node:process";
import { collectMcpMergePaths, getAvailableMcpServerKeys, mergeMcpJsonFiles } from "./mcp.js";
import type { AgentId, IdeId, OptionalToolId } from "./types.js";
import { logInfo, logWarn } from "./log.js";

function isCi(): boolean {
  const c = process.env.CI?.toLowerCase();
  return c === "true" || c === "1" || c === "yes";
}

function interactiveEligible(nonInteractive: boolean, selectedIdes: IdeId[]): boolean {
  if (nonInteractive) return false;
  if (isCi()) return false;
  if (!input.isTTY || !output.isTTY) return false;
  return selectedIdes.length === 0;
}

function mcpInteractiveEligible(nonInteractive: boolean, mergeMcp: boolean): boolean {
  if (!mergeMcp) return false;
  if (nonInteractive) return false;
  if (isCi()) return false;
  if (!input.isTTY || !output.isTTY) return false;
  return true;
}

function toolsInteractiveEligible(nonInteractive: boolean, toolsInteractive: boolean): boolean {
  if (!toolsInteractive) return false;
  if (nonInteractive) return false;
  if (isCi()) return false;
  if (!input.isTTY || !output.isTTY) return false;
  return true;
}

function agentsInteractiveEligible(nonInteractive: boolean, agentsInteractive: boolean): boolean {
  if (!agentsInteractive) return false;
  if (nonInteractive) return false;
  if (isCi()) return false;
  if (!input.isTTY || !output.isTTY) return false;
  return true;
}

export async function interactiveIdeSelection(nonInteractive: boolean, current: IdeId[]): Promise<{ ides: IdeId[]; allIdes: boolean }> {
  if (!interactiveEligible(nonInteractive, current) || current.length > 0) {
    return { ides: current, allIdes: false };
  }

  const rl = readline.createInterface({ input, output });
  try {
    console.log("");
    console.log("\u001b[0;36mIDE integration\u001b[0m — symlink the hub into which tools?");
    console.log("  1) Cursor (.cursor/)");
    console.log("  2) Copilot (.vscode/)");
    console.log("  3) Kilocode (.kilo/)");
    console.log("  4) OpenCode (.opencode/)");
    console.log("  5) Antigravity (.agents/)");
    console.log("");
    console.log("  a — all five   |   e.g. 1,3,5 — only those numbers");
    console.log("");

    while (true) {
      const choice = (await rl.question("IDE choice [a]: ")).trim();
      if (!choice) {
        logInfo("IDE: all five (default)");
        return { ides: ["cursor", "copilot", "kilo", "opencode", "antigravity"], allIdes: true };
      }
      const lower = choice.toLowerCase();
      if (lower === "a" || lower === "all") {
        logInfo("IDE: all five");
        return { ides: ["cursor", "copilot", "kilo", "opencode", "antigravity"], allIdes: true };
      }

      const ides: IdeId[] = [];
      const map: Record<string, IdeId> = {
        "1": "cursor",
        "2": "copilot",
        "3": "kilo",
        "4": "opencode",
        "5": "antigravity",
      };
      let bad = false;
      for (const tok of choice.split(",")) {
        const t = tok.trim();
        if (!t) continue;
        const id = map[t];
        if (!id) {
          logWarn(`invalid index: ${t} (use 1–5 or a)`);
          bad = true;
          break;
        }
        ides.push(id);
      }
      if (bad) continue;
      if (ides.length === 0) {
        logWarn("Pick at least one number (1–5) or a for all");
        continue;
      }
      logInfo(`IDE: selected ${ides.length} tool(s)`);
      return { ides, allIdes: false };
    }
  } finally {
    rl.close();
  }
}

export async function interactiveMcpSelection(
  hubRoot: string,
  nonInteractive: boolean,
  mergeMcp: boolean,
  allMcp: boolean,
): Promise<{ mergeMcp: boolean; filterKeys: string[] | null }> {
  if (allMcp) {
    logInfo("MCP: all servers (--all-mcp)");
    return { mergeMcp: true, filterKeys: null };
  }
  if (!mcpInteractiveEligible(nonInteractive, mergeMcp)) {
    return { mergeMcp, filterKeys: null };
  }

  const paths = collectMcpMergePaths(hubRoot);
  if (paths.length === 0) {
    logInfo("MCP: no .configs/mcp/ directory found — skipping MCP selection");
    return { mergeMcp, filterKeys: null };
  }

  const merged = mergeMcpJsonFiles(paths);
  const servers = getAvailableMcpServerKeys(merged);
  if (servers.length === 0) {
    logInfo("MCP: no servers found — skipping");
    return { mergeMcp, filterKeys: null };
  }

  const rl = readline.createInterface({ input, output });
  try {
    console.log("");
    console.log("\u001b[0;36mMCP Servers\u001b[0m — which servers to enable?");
    console.log("  a — all servers   |   n — skip MCP   |   e.g. 1,3,5 — subset");
    console.log("");
    let i = 1;
    const indexToName: string[] = [];
    for (const s of servers) {
      console.log(`  ${i}) ${s}`);
      indexToName.push(s);
      i += 1;
    }
    console.log("");

    while (true) {
      const choice = (await rl.question("MCP choice [a]: ")).trim();
      if (!choice) {
        logInfo("MCP: all servers (default)");
        return { mergeMcp: true, filterKeys: null };
      }
      const lower = choice.toLowerCase();
      if (lower === "a" || lower === "all") {
        logInfo("MCP: all servers");
        return { mergeMcp: true, filterKeys: null };
      }
      if (lower === "n" || lower === "no" || lower === "skip") {
        logInfo("MCP: skipped");
        return { mergeMcp: false, filterKeys: null };
      }

      const names: string[] = [];
      let bad = false;
      for (const tok of choice.split(",")) {
        const t = tok.trim();
        if (!t) continue;
        const idx = Number(t);
        if (!Number.isInteger(idx) || idx < 1 || idx > indexToName.length) {
          logWarn(`invalid index: ${t} (use 1–${indexToName.length}, a, or n)`);
          bad = true;
          break;
        }
        names.push(indexToName[idx - 1]!);
      }
      if (bad) continue;
      if (names.length === 0) {
        logWarn("Pick at least one number, a for all, or n to skip");
        continue;
      }
      logInfo(`MCP: selected ${names.length} server(s)`);
      return { mergeMcp: true, filterKeys: names };
    }
  } finally {
    rl.close();
  }
}

export async function interactiveToolsSelection(nonInteractive: boolean, toolsInteractive: boolean): Promise<OptionalToolId[]> {
  if (!toolsInteractiveEligible(nonInteractive, toolsInteractive)) {
    if (toolsInteractive && (nonInteractive || isCi())) {
      logInfo("Optional tools: skipping -t prompt (--yes or CI); gitnexus, engram, and context-mode CLIs still run with the rest of install");
    }
    return [];
  }

  const rl = readline.createInterface({ input, output });
  try {
    console.log("");
    console.log("\u001b[0;36mOptional tools\u001b[0m — CLIs via npm, uv, hub npm ci, brew, or Brave install (see docs/01-installation.md)");
    console.log("  1) openspec — OpenSpec (brew, hub npx, or npm -g)");
    console.log("  2) dmux — tmux pane manager (hub npx, or npm -g fallback)");
    console.log("  3) brave-search-cli — Brave Search CLI, bx (curl | sh from brave/brave-search-cli)");
    console.log("  4) cocoindex-code — CocoIndex Code, ccc (uv tool install --upgrade 'cocoindex-code[full]')");
    console.log("  5) truecourse — architecture analysis CLI (npm install -g truecourse)");
    console.log("");
    console.log("  (gitnexus, engram, and context-mode are installed automatically for hub MCP — not listed here.)");
    console.log("");
    console.log("  a — all five   |   n — none   |   e.g. 1,2 — pick by number");
    console.log("");

    while (true) {
      const choice = (await rl.question("Tool choice [n]: ")).trim();
      if (!choice) {
        logInfo("Optional tools: none (default)");
        return [];
      }
      const lower = choice.toLowerCase();
      if (lower === "a" || lower === "all") {
        logInfo("Optional tools: openspec, dmux, brave-search-cli, cocoindex-code, truecourse (gitnexus, engram, context-mode always)");
        return ["openspec", "dmux", "brave-search-cli", "cocoindex-code", "truecourse"];
      }
      if (lower === "n" || lower === "no" || lower === "none" || lower === "skip") {
        logInfo("Optional tools: none");
        return [];
      }

      const tools: OptionalToolId[] = [];
      const map: Record<string, OptionalToolId> = {
        "1": "openspec",
        "2": "dmux",
        "3": "brave-search-cli",
        "4": "cocoindex-code",
        "5": "truecourse",
      };
      let bad = false;
      for (const tok of choice.split(",")) {
        const t = tok.trim();
        if (!t) continue;
        const id = map[t];
        if (!id) {
          logWarn(`invalid index: ${t} (use 1–5, a, or n)`);
          bad = true;
          break;
        }
        tools.push(id);
      }
      if (bad) continue;
      if (tools.length === 0) {
        logWarn("Pick at least one number (1–5), a for all, or n for none");
        continue;
      }
      logInfo(`Optional tools: selected ${tools.length} tool(s)`);
      return tools;
    }
  } finally {
    rl.close();
  }
}

export async function interactiveAgentsSelection(nonInteractive: boolean, agentsInteractive: boolean): Promise<AgentId[]> {
  if (!agentsInteractiveEligible(nonInteractive, agentsInteractive)) {
    if (agentsInteractive && (nonInteractive || isCi())) {
      logInfo("Agent CLIs: skipping -g prompt (--yes or CI)");
    }
    return [];
  }

  const rl = readline.createInterface({ input, output });
  try {
    console.log("");
    console.log("\u001b[0;36mAgent CLIs\u001b[0m — install coding-agent command-line tools?");
    console.log("  1) Claude Code (claude)");
    console.log("  2) OpenCode (opencode)");
    console.log("  3) Kilocode (kilo)");
    console.log("  4) Codex (codex)");
    console.log("  5) Gemini CLI (gemini)");
    console.log("  6) pi (@mariozechner/pi-coding-agent)");
    console.log("");
    console.log("  a — all six   |   n — none   |   e.g. 1,3,5 — pick by number");
    console.log("");

    while (true) {
      const choice = (await rl.question("Agent choice [n]: ")).trim();
      if (!choice) {
        logInfo("Agent CLIs: none (default)");
        return [];
      }
      const lower = choice.toLowerCase();
      if (lower === "a" || lower === "all") {
        logInfo("Agent CLIs: all six selected");
        return ["claude-code", "opencode", "kilo", "codex", "gemini", "pi"];
      }
      if (lower === "n" || lower === "no" || lower === "none" || lower === "skip") {
        logInfo("Agent CLIs: none");
        return [];
      }

      const agents: AgentId[] = [];
      const map: Record<string, AgentId> = {
        "1": "claude-code",
        "2": "opencode",
        "3": "kilo",
        "4": "codex",
        "5": "gemini",
        "6": "pi",
      };
      let bad = false;
      for (const tok of choice.split(",")) {
        const t = tok.trim();
        if (!t) continue;
        const id = map[t];
        if (!id) {
          logWarn(`invalid index: ${t} (use 1–6, a, or n)`);
          bad = true;
          break;
        }
        agents.push(id);
      }
      if (bad) continue;
      if (agents.length === 0) {
        logWarn("Pick at least one number (1–6), a for all, or n for none");
        continue;
      }
      logInfo(`Agent CLIs: selected ${agents.length} agent(s)`);
      return agents;
    }
  } finally {
    rl.close();
  }
}
