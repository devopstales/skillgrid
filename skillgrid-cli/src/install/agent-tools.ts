import { spawnSync } from "node:child_process";
import type { AgentId } from "./types.js";
import { logInfo, logSuccess, logWarn } from "./log.js";
import { commandOnPath } from "./exec.js";
import { agentIsSelected } from "./agent-tools-helpers.js";

function npmGlobal(dryRun: boolean, pkg: string, label: string, bin: string): void {
  if (commandOnPath(bin)) {
    logInfo(`${label} already on PATH`);
    return;
  }
  if (dryRun) {
    console.log(`[DRY-RUN] npm install -g ${pkg}`);
    return;
  }
  if (!commandOnPath("npm")) {
    logWarn(`${label}: npm not found — install Node.js, then: npm install -g ${pkg}`);
    return;
  }
  logInfo(`Installing ${label} (npm install -g ${pkg})...`);
  const r = spawnSync("npm", ["install", "-g", pkg], { stdio: "inherit" });
  if (r.status === 0) logSuccess(`${label} installed (${bin})`);
  else logWarn(`${label}: npm install -g failed`);
}

/** Install CLIs for selected agents (install.sh install_agent_clis parity). */
export function installAgentClis(selected: AgentId[], dryRun: boolean): void {
  if (selected.length === 0) return;

  console.log("");
  console.log("Agent CLIs — installing selected coding-agent tools...");
  console.log("");

  if (agentIsSelected(selected, "claude-code")) {
    npmGlobal(dryRun, "@anthropic-ai/claude-code", "Claude Code", "claude");
  }

  if (agentIsSelected(selected, "opencode")) {
    if (commandOnPath("opencode")) {
      logInfo(`OpenCode already on PATH`);
    } else if (dryRun) {
      console.log("[DRY-RUN] curl -fsSL https://opencode.ai/install | bash");
      console.log("[DRY-RUN] fallback: brew install opencode");
    } else if (commandOnPath("curl")) {
      logInfo("Installing OpenCode (official install script)...");
      const r = spawnSync("sh", ["-c", "curl -fsSL https://opencode.ai/install | bash"], { stdio: "inherit" });
      if (r.status === 0) logSuccess("OpenCode installed (opencode)");
      else if (commandOnPath("brew")) {
        const b = spawnSync("brew", ["install", "opencode"], { stdio: "inherit" });
        if (b.status === 0) logSuccess("OpenCode installed via Homebrew (opencode)");
        else logWarn("OpenCode: install failed — try: curl -fsSL https://opencode.ai/install | bash");
      } else logWarn("OpenCode: install failed — try: curl -fsSL https://opencode.ai/install | bash");
    } else if (commandOnPath("brew")) {
      logInfo("Installing OpenCode (Homebrew)...");
      const b = spawnSync("brew", ["install", "opencode"], { stdio: "inherit" });
      if (b.status === 0) logSuccess("OpenCode installed (opencode)");
      else logWarn("OpenCode: brew install failed");
    } else logWarn("OpenCode: curl and brew not found — install manually from https://opencode.ai");
  }

  if (agentIsSelected(selected, "kilo")) {
    if (commandOnPath("kilo")) {
      logInfo("Kilocode already on PATH");
    } else if (dryRun) {
      console.log("[DRY-RUN] npm install -g @kilocode/cli");
      console.log("[DRY-RUN] fallback: brew install Kilo-Org/tap/kilo");
    } else if (commandOnPath("npm")) {
      logInfo("Installing Kilocode CLI (npm install -g @kilocode/cli)...");
      const r = spawnSync("npm", ["install", "-g", "@kilocode/cli"], { stdio: "inherit" });
      if (r.status === 0) logSuccess("Kilocode installed (kilo)");
      else if (commandOnPath("brew")) {
        const b = spawnSync("brew", ["install", "Kilo-Org/tap/kilo"], { stdio: "inherit" });
        if (b.status === 0) logSuccess("Kilocode installed via Homebrew (kilo)");
        else logWarn("Kilocode: npm install -g failed — try: npm install -g @kilocode/cli");
      } else logWarn("Kilocode: npm install -g failed — try: npm install -g @kilocode/cli");
    } else if (commandOnPath("brew")) {
      logInfo("Installing Kilocode (Homebrew)...");
      const b = spawnSync("brew", ["install", "Kilo-Org/tap/kilo"], { stdio: "inherit" });
      if (b.status === 0) logSuccess("Kilocode installed (kilo)");
      else logWarn("Kilocode: brew install failed");
    } else logWarn("Kilocode: npm and brew not found — install Node.js or Homebrew first");
  }

  if (agentIsSelected(selected, "codex")) {
    if (commandOnPath("codex")) {
      logInfo("Codex already on PATH");
    } else if (dryRun) {
      console.log("[DRY-RUN] npm install -g @openai/codex");
      console.log("[DRY-RUN] fallback: brew install --cask codex");
    } else if (commandOnPath("npm")) {
      logInfo("Installing Codex (npm install -g @openai/codex)...");
      const r = spawnSync("npm", ["install", "-g", "@openai/codex"], { stdio: "inherit" });
      if (r.status === 0) logSuccess("Codex installed (codex)");
      else if (commandOnPath("brew")) {
        const b = spawnSync("brew", ["install", "--cask", "codex"], { stdio: "inherit" });
        if (b.status === 0) logSuccess("Codex installed via Homebrew cask (codex)");
        else logWarn("Codex: npm install -g failed — see https://developers.openai.com/codex");
      } else logWarn("Codex: npm install -g failed — see https://developers.openai.com/codex");
    } else if (commandOnPath("brew")) {
      logInfo("Installing Codex (Homebrew cask)...");
      const b = spawnSync("brew", ["install", "--cask", "codex"], { stdio: "inherit" });
      if (b.status === 0) logSuccess("Codex installed (codex)");
      else logWarn("Codex: brew install --cask codex failed");
    } else logWarn("Codex: npm and brew not found — install Node.js or Homebrew first");
  }

  if (agentIsSelected(selected, "gemini")) {
    if (commandOnPath("gemini")) {
      logInfo("Gemini CLI already on PATH");
    } else if (dryRun) {
      console.log("[DRY-RUN] npm install -g @google/gemini-cli");
      console.log("[DRY-RUN] fallback: brew install gemini-cli");
    } else if (commandOnPath("npm")) {
      logInfo("Installing Gemini CLI (npm install -g @google/gemini-cli)...");
      const r = spawnSync("npm", ["install", "-g", "@google/gemini-cli"], { stdio: "inherit" });
      if (r.status === 0) logSuccess("Gemini CLI installed (gemini)");
      else if (commandOnPath("brew")) {
        const b = spawnSync("brew", ["install", "gemini-cli"], { stdio: "inherit" });
        if (b.status === 0) logSuccess("Gemini CLI installed via Homebrew (gemini)");
        else logWarn("Gemini CLI: npm install -g failed — see https://github.com/google-gemini/gemini-cli");
      } else logWarn("Gemini CLI: npm install -g failed — see https://github.com/google-gemini/gemini-cli");
    } else if (commandOnPath("brew")) {
      logInfo("Installing Gemini CLI (Homebrew)...");
      const b = spawnSync("brew", ["install", "gemini-cli"], { stdio: "inherit" });
      if (b.status === 0) logSuccess("Gemini CLI installed (gemini)");
      else logWarn("Gemini CLI: brew install failed");
    } else logWarn("Gemini CLI: npm and brew not found — install Node.js or Homebrew first");
  }

  if (agentIsSelected(selected, "pi")) {
    npmGlobal(dryRun, "@mariozechner/pi-coding-agent", "pi", "pi");
  }

  console.log("");
}
