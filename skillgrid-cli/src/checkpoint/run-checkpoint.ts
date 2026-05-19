import { spawnSync } from "node:child_process";
import path from "node:path";
import { execSync } from "node:child_process";
import fs from "node:fs";

function findRepoRoot(start: string): string {
  try {
    return execSync("git rev-parse --show-toplevel", {
      cwd: start,
      encoding: "utf8"
    }).trim();
  } catch {
    throw new Error("Not inside a git repository. Run from your project root.");
  }
}

export function printCheckpointHelp(): void {
  console.log(`skillgrid checkpoint — record a Tier 1 operational checkpoint

Usage:
  skillgrid checkpoint --change <id> --name <label> [options]

Required:
  --change <id>       OpenSpec / Skillgrid change id
  --name <label>      Checkpoint name (e.g. before-apply, verify-pass)

Options:
  --trigger <id>      Trigger id (default: same as --name)
  --phase <phase>     SDD phase (apply, verify, archive, loop, handoff)
  --slice <text>      Active slice or task line
  --evidence <text>   One-line evidence summary
  --prd <path>        PRD path (auto-detected when omitted)
  --context <path>    Handoff path (default: .skillgrid/tasks/context_<change>.md)
  --dry-run           Print actions without writing

See: docs/18-checkpoints.md and .agents/skills/skillgrid-checkpoints/SKILL.md
`);
}

export function runCheckpointCommand(argv: string[]): number {
  if (argv.length === 0 || argv[0] === "-h" || argv[0] === "--help") {
    printCheckpointHelp();
    return 0;
  }

  let repoRoot: string;
  try {
    repoRoot = findRepoRoot(process.cwd());
  } catch (e) {
    console.error((e as Error).message);
    return 2;
  }

  const scriptPath = path.join(repoRoot, ".skillgrid", "scripts", "checkpoint-record.sh");
  if (!fs.existsSync(scriptPath)) {
    console.error(`Missing checkpoint script: ${scriptPath}`);
    console.error("Run skillgrid install in this repo or copy .skillgrid/scripts from the hub.");
    return 2;
  }

  const result = spawnSync("bash", [scriptPath, ...argv], {
    cwd: repoRoot,
    stdio: "inherit",
    env: process.env
  });

  if (result.error) {
    console.error(result.error.message);
    return 3;
  }

  return result.status ?? 1;
}
