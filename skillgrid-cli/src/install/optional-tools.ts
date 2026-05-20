import { existsSync } from "node:fs";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import type { OptionalToolId } from "./types.js";
import { logInfo, logSuccess, logWarn } from "./log.js";
import { commandOnPath } from "./exec.js";
import { toolIsSelected } from "./optional-tools-helpers.js";
import { mcpTrivyIsSelected } from "./deps.js";
import { installContextModeCli } from "./context-mode.js";

function hubNpmBin(hubRoot: string, name: string): string {
  return join(hubRoot, "node_modules", ".bin", name);
}

function hubEnsureNpmInstall(hubRoot: string, dryRun: boolean): boolean {
  if (!existsSync(join(hubRoot, "package.json"))) return false;
  if (dryRun) {
    console.log(`[DRY-RUN] (cd "${hubRoot}" && npm ci)`);
    return false;
  }
  logInfo(`Hub Node dependencies (npm ci in ${hubRoot})...`);
  const r = spawnSync("npm", ["ci"], { cwd: hubRoot, stdio: "inherit" });
  if (r.status === 0) return true;
  const r2 = spawnSync("npm", ["install"], { cwd: hubRoot, stdio: "inherit" });
  if (r2.status !== 0) {
    logWarn("hub npm ci failed — check Node, network, and package-lock.json (see docs/01-installation.md)");
    return false;
  }
  return true;
}

function installOpenspecCli(hubRoot: string, dryRun: boolean): void {
  if (commandOnPath("openspec")) {
    logInfo("openspec CLI already on PATH");
    return;
  }
  if (dryRun) {
    console.log("[DRY-RUN] brew install openspec  ||  hub npm ci  ||  npm install -g @fission-ai/openspec@latest");
    return;
  }
  if (commandOnPath("brew")) {
    const r = spawnSync("brew", ["install", "openspec"], { stdio: "inherit" });
    if (r.status === 0) {
      logSuccess("openspec installed (Homebrew)");
      return;
    }
  }
  if (hubEnsureNpmInstall(hubRoot, false) && existsSync(hubNpmBin(hubRoot, "openspec"))) {
    logSuccess(`openspec available: cd "${hubRoot}" && npx openspec (or add node_modules/.bin to PATH)`);
    return;
  }
  if (commandOnPath("npm")) {
    logInfo("Installing openspec via npm global (@fission-ai/openspec)...");
    const r = spawnSync("npm", ["install", "-g", "@fission-ai/openspec@latest"], { stdio: "inherit" });
    if (r.status === 0) {
      logSuccess("openspec installed (npm -g)");
      return;
    }
  }
  logWarn("openspec: install manually — brew install openspec  OR  npm ci in hub  OR  npm install -g @fission-ai/openspec@latest");
}

function ensureUv(dryRun: boolean): boolean {
  if (commandOnPath("uv")) return true;
  if (dryRun) {
    console.log("[DRY-RUN] brew install uv  (or: curl -LsSf https://astral.sh/uv/install.sh | sh)");
    return false;
  }
  logInfo("Installing uv...");
  if (commandOnPath("brew")) {
    spawnSync("brew", ["install", "uv"], { stdio: "inherit" });
  } else if (commandOnPath("curl")) {
    const curl = spawnSync("curl", ["-LsSf", "https://astral.sh/uv/install.sh"], { encoding: "utf8" });
    if (curl.stdout) {
      spawnSync("sh", ["-c", curl.stdout], { stdio: "inherit" });
    }
  } else {
    logWarn("uv not found — install with: brew install uv (or https://docs.astral.sh/uv/)");
  }
  return commandOnPath("uv");
}

/** Mutates `selected` to always include gitnexus, engram, and context-mode (install.sh behavior). */
export function reconcileDefaultTools(selected: OptionalToolId[]): void {
  if (!toolIsSelected(selected, "gitnexus")) selected.push("gitnexus");
  if (!toolIsSelected(selected, "engram")) selected.push("engram");
  if (!toolIsSelected(selected, "context-mode")) selected.push("context-mode");
}

export function installOptionalToolClis(hubRoot: string, selected: OptionalToolId[], dryRun: boolean): void {
  reconcileDefaultTools(selected);

  console.log("");
  console.log("Optional tools — installing CLIs (includes gitnexus, engram, and context-mode for bundled MCP)...");
  console.log("");

  if (toolIsSelected(selected, "gitnexus")) {
    if (commandOnPath("gitnexus")) logInfo("gitnexus CLI already present");
    else if (dryRun) console.log("[DRY-RUN] npm install -g gitnexus@1.3.11");
    else if (commandOnPath("npm")) {
      logInfo("Installing GitNexus (npm install -g gitnexus@1.3.11)...");
      const r = spawnSync("npm", ["install", "-g", "gitnexus@1.3.11"], { stdio: "inherit" });
      if (r.status === 0) logSuccess("gitnexus installed");
      else logWarn("gitnexus: npm install -g failed");
    } else logWarn("gitnexus: npm not found — install Node.js, then run: npm install -g gitnexus@1.3.11");
  }

  if (toolIsSelected(selected, "cocoindex-code")) {
    ensureUv(dryRun) || true;
    if (dryRun) console.log("[DRY-RUN] uv tool install --upgrade 'cocoindex-code[full]'");
    else if (commandOnPath("uv")) {
      logInfo("Installing/upgrading cocoindex-code[full] (ccc) via uv tool install...");
      const r = spawnSync("uv", ["tool", "install", "--upgrade", "cocoindex-code[full]"], { stdio: "inherit" });
      if (r.status === 0) logSuccess("cocoindex-code (ccc) installed or upgraded");
      else logWarn("cocoindex-code: uv tool install --upgrade failed");
    } else logWarn("cocoindex-code: uv missing — run: uv tool install --upgrade 'cocoindex-code[full]'");
  }

  if (toolIsSelected(selected, "openspec")) installOpenspecCli(hubRoot, dryRun);

  if (toolIsSelected(selected, "dmux")) {
    if (commandOnPath("dmux")) logInfo("dmux CLI already on PATH");
    else if (dryRun) console.log("[DRY-RUN] hub npm ci  ||  npm install -g dmux");
    else if (hubEnsureNpmInstall(hubRoot, false) && existsSync(hubNpmBin(hubRoot, "dmux"))) {
      logSuccess(`dmux available: cd "${hubRoot}" && npx dmux (or add node_modules/.bin to PATH)`);
    } else if (commandOnPath("npm")) {
      logInfo("Installing dmux (npm -g fallback)...");
      const r = spawnSync("npm", ["install", "-g", "dmux"], { stdio: "inherit" });
      if (r.status === 0) logSuccess("dmux installed (npm -g)");
      else logWarn("dmux: npm install -g failed");
    } else logWarn("dmux: npm not found — install Node.js");
  }

  if (toolIsSelected(selected, "context-mode")) {
    installContextModeCli(dryRun);
  }

  if (toolIsSelected(selected, "engram")) {
    if (commandOnPath("engram")) logInfo(`engram CLI already installed`);
    else if (dryRun) {
      console.log("[DRY-RUN] brew install gentleman-programming/tap/engram");
      console.log("[DRY-RUN] verify MCP fragment: .configs/mcp/command/engram.json");
    } else if (commandOnPath("brew")) {
      logInfo("Installing engram (Homebrew)...");
      const r = spawnSync("brew", ["install", "gentleman-programming/tap/engram"], { stdio: "inherit" });
      if (r.status === 0) logSuccess("engram installed");
      else logWarn("engram: brew install failed — install manually with: brew install gentleman-programming/tap/engram");
    } else logWarn("engram: Homebrew not found — run: brew install gentleman-programming/tap/engram");
  }

  if (toolIsSelected(selected, "brave-search-cli")) {
    if (commandOnPath("bx")) logInfo("brave-search-cli (bx) already on PATH");
    else if (dryRun) {
      console.log("[DRY-RUN] curl -fsSL https://raw.githubusercontent.com/brave/brave-search-cli/main/scripts/install.sh | sh");
    } else if (commandOnPath("curl")) {
      logInfo("Installing brave-search-cli (official install.sh → bx)...");
      const r = spawnSync("sh", ["-c", "curl -fsSL https://raw.githubusercontent.com/brave/brave-search-cli/main/scripts/install.sh | sh"], {
        stdio: "inherit",
      });
      if (r.status === 0) logSuccess("brave-search-cli installed (bx)");
      else logWarn("brave-search-cli: install script failed");
    } else logWarn("brave-search-cli: curl not found — install curl or run the install command manually");
  }

  if (toolIsSelected(selected, "truecourse")) {
    if (commandOnPath("truecourse")) logInfo("truecourse CLI already on PATH");
    else if (dryRun) console.log("[DRY-RUN] npm install -g truecourse");
    else if (commandOnPath("npm")) {
      logInfo("Installing TrueCourse (npm install -g truecourse)...");
      const r = spawnSync("npm", ["install", "-g", "truecourse"], { stdio: "inherit" });
      if (r.status === 0) logSuccess("truecourse installed");
      else logWarn("truecourse: npm install -g failed — see https://github.com/truecourse-ai/truecourse");
    } else logWarn("truecourse: npm not found — install Node.js, then run: npm install -g truecourse");
  }

  console.log("");
}

/** When hub MCP includes `trivy`, ensure the Trivy MCP plugin (install.sh ensure_trivy_mcp_plugin). */
export function ensureTrivyMcpPlugin(mergeMcp: boolean, mcpKeyFilter: string[] | null, dryRun: boolean): void {
  if (!mcpTrivyIsSelected(mergeMcp, mcpKeyFilter)) return;
  if (!commandOnPath("trivy")) return;
  if (dryRun) {
    console.log("[DRY-RUN] trivy plugin install mcp");
    console.log("");
    return;
  }
  console.log("Installing trivy MCP plugin...");
  spawnSync("trivy", ["plugin", "install", "mcp"], { stdio: "inherit" });
  console.log("");
}

/** For sanity: dmux in hub node_modules counts as present. */
export function dmuxAvailable(hubRoot: string): boolean {
  return commandOnPath("dmux") || existsSync(hubNpmBin(hubRoot, "dmux"));
}

/** gitnexus or npx (install.sh sanity). */
export function gitnexusOrNpx(): boolean {
  return commandOnPath("gitnexus") || commandOnPath("npx");
}
