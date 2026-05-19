import { existsSync } from "node:fs";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import { CORE_DEPENDENCIES, IDE_DEPENDENCIES } from "./deps.js";
import { commandOnPath, shCheck } from "./exec.js";
import { dmuxAvailable, gitnexusOrNpx } from "./optional-tools.js";
import { logError, logSuccess } from "./log.js";

let failures = 0;

function sanityOk(label: string) {
  console.log(`  ✓ ${label}`);
}

function sanityFail(label: string) {
  console.log(`  ✗ ${label}`);
  failures += 1;
}

function sanityCheckCommand(label: string, cmd: string, hint: string) {
  if (shCheck(cmd)) sanityOk(label);
  else sanityFail(`${label} — ${hint}`);
}

function sanityCheckFile(label: string, filePath: string) {
  if (existsSync(filePath)) sanityOk(label);
  else sanityFail(`${label} — missing ${filePath}`);
}

export function runSanityCheck(hubRoot: string): number {
  failures = 0;
  console.log("=== install.sh Sanity Check (skillgrid install --sanity-check) ===");
  console.log("");

  console.log("Core commands:");
  for (const dep of CORE_DEPENDENCIES) {
    if (!dep.checkCmd) continue;
    let hint = "";
    if (dep.brew) hint = `install with: brew install ${dep.brew}`;
    else if (dep.npm) hint = `install with: npm install -g ${dep.npm}`;
    else if (dep.pip) hint = `install with: pip3 install ${dep.pip}`;
    sanityCheckCommand(dep.name, dep.checkCmd, hint);
  }

  console.log("");
  console.log("IDE and security CLIs:");
  for (const dep of IDE_DEPENDENCIES) {
    if (!dep.checkCmd) continue;
    let hint = "";
    if (dep.brew) hint = `install with: brew install ${dep.brew}`;
    else if (dep.npm) hint = `install with: npm install -g ${dep.npm}`;
    else if (dep.pip) hint = `install with: pip3 install ${dep.pip}`;
    sanityCheckCommand(dep.name, dep.checkCmd, hint);
  }

  console.log("");
  console.log("Optional Skillgrid tools:");
  sanityCheckCommand("uv", "command -v uv", "install with: brew install uv");
  if (gitnexusOrNpx()) sanityOk("gitnexus");
  else sanityFail("gitnexus — install with: npm install -g gitnexus@1.3.11");
  if (commandOnPath("ccc")) sanityOk("cocoindex-code (ccc)");
  else sanityFail("cocoindex-code (ccc) — install with: uv tool install --upgrade 'cocoindex-code[full]'");

  if (dmuxAvailable(hubRoot)) sanityOk("dmux");
  else sanityFail("dmux — run npm ci or install dmux");

  if (commandOnPath("engram")) sanityOk("engram");
  else sanityFail("engram — install with: brew install gentleman-programming/tap/engram");

  if (commandOnPath("context-mode")) sanityOk("context-mode");
  else sanityFail("context-mode — install with: npm install -g context-mode");

  if (commandOnPath("bx")) sanityOk("brave-search-cli (bx)");
  else sanityFail("brave-search-cli (bx) — install from brave-search-cli");

  console.log("");
  console.log("Hub files:");
  sanityCheckFile("AGENTS.md template", join(hubRoot, ".configs", "AGENTS.md"));
  sanityCheckFile("MCP config fragments", join(hubRoot, ".configs", "mcp"));
  sanityCheckFile("Engram MCP fragment", join(hubRoot, ".configs", "mcp", "command", "engram.json"));
  sanityCheckFile("context-mode MCP fragment", join(hubRoot, ".configs", "mcp", "context-mode.json"));
  sanityCheckFile("context-mode Cursor hooks template", join(hubRoot, ".configs", "context-mode", "cursor", "hooks.json"));
  sanityCheckFile("Skill catalog", join(hubRoot, ".agents", "skills"));
  sanityCheckFile("Skillgrid CLI package", join(hubRoot, "skillgrid-cli", "package.json"));
  sanityCheckFile("Preview script", join(hubRoot, ".skillgrid", "scripts", "preview.sh"));
  sanityCheckFile("Checkpoint script", join(hubRoot, ".skillgrid", "scripts", "checkpoint-record.sh"));
  sanityCheckFile("SDD gate script", join(hubRoot, ".skillgrid", "scripts", "sdd-gate.sh"));
  sanityCheckFile("IDE sync script", join(hubRoot, "scripts", "sync-ide-assets.sh"));
  sanityCheckFile("Node package manifest", join(hubRoot, "package.json"));

  console.log("");
  console.log("Hub script checks:");
  const checkpoint = join(hubRoot, ".skillgrid", "scripts", "checkpoint-record.sh");
  const r1 = spawnSync("bash", ["-n", checkpoint], { stdio: "ignore" });
  if (r1.status === 0) sanityOk("checkpoint-record.sh syntax");
  else sanityFail("checkpoint-record.sh syntax — check bash script");

  const sync = join(hubRoot, "scripts", "sync-ide-assets.sh");
  const r2 = spawnSync("bash", ["-n", sync], { stdio: "ignore" });
  if (r2.status === 0) sanityOk("sync-ide-assets.sh syntax");
  else sanityFail("sync-ide-assets.sh syntax — check sync script syntax");

  console.log("");
  if (failures === 0) {
    logSuccess("Sanity check passed");
    return 0;
  }
  logError(`Sanity check found ${failures} issue(s)`);
  return 1;
}
