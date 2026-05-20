import { hubRootFromCliModule } from "./install/hub-root.js";
import { parseInstallArgv } from "./install/parse-install-argv.js";
import { runInstallCli } from "./install/run-install.js";
import { printCheckpointHelp, runCheckpointCommand } from "./checkpoint/run-checkpoint.js";
import { printServeHelp, runServeCommand } from "./serve/run-serve.js";

function printTopHelp() {
  console.log(`skillgrid — Skillgrid hub CLI

Usage:
  skillgrid install [OPTIONS]   Install from shared hub cache (/tmp/…/skillgrid-aiskillgrid-release) → rsync
  skillgrid serve [OPTIONS]     Skillgrid Dashboard (--repo, --port, --open; see serve --help)
  skillgrid checkpoint [OPTS]   Record Tier 1 checkpoint (docs/14-checkpoints.md)
  skillgrid --help              Show this message
`);
}

async function main() {
  const argv = process.argv.slice(2);
  if (argv.length === 0 || argv[0] === "-h" || argv[0] === "--help") {
    printTopHelp();
    process.exit(0);
  }

  if (argv[0] === "install") {
    const hubRoot = hubRootFromCliModule(import.meta.url);
    let parsed;
    try {
      parsed = parseInstallArgv(argv.slice(1));
    } catch (e) {
      console.error((e as Error).message);
      process.exit(1);
    }
    const code = await runInstallCli(hubRoot, parsed);
    process.exit(code);
  }

  if (argv[0] === "checkpoint") {
    const rest = argv.slice(1);
    if (rest[0] === "-h" || rest[0] === "--help") {
      printCheckpointHelp();
      process.exit(0);
    }
    process.exit(runCheckpointCommand(rest));
  }

  if (argv[0] === "serve") {
    const rest = argv.slice(1);
    if (rest[0] === "-h" || rest[0] === "--help") {
      printServeHelp();
      process.exit(0);
    }
    try {
      await runServeCommand(import.meta.url, rest);
      /* Do not process.exit: the dashboard keeps the process alive until SIGINT/SIGTERM. */
    } catch (e) {
      console.error((e as Error).message);
      process.exit(1);
    }
    return;
  }

  console.error(`Unknown command: ${argv[0]}`);
  printTopHelp();
  process.exit(1);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
