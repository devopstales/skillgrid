import { spawn } from "node:child_process";
import { existsSync, realpathSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { resolveRepoRoot } from "../dashboard/server/fs-utils.js";
import { startDashboardServer } from "../dashboard/server/server.js";

const DASHBOARD_CLIENT_INDEX = ["dist", "dashboard", "client", "index.html"] as const;

/**
 * Bun `--compile` bundles use a virtual `import.meta.url` (`/$bunfs/...`), so the package root
 * cannot be derived from `fileURLToPath`. Prefer a real on-disk layout: `dist/dashboard/client`
 * next to `dist/serve` (dev) or next to `bin/` (compiled binary).
 */
function resolveSkillgridCliPackageRoot(importMetaUrl: string): string {
  const fromMeta = path.resolve(path.dirname(fileURLToPath(importMetaUrl)), "..");
  if (existsSync(path.join(fromMeta, ...DASHBOARD_CLIENT_INDEX))) {
    return fromMeta;
  }

  try {
    const execDir = path.dirname(realpathSync(process.execPath));
    const fromExec = path.resolve(execDir, "..");
    if (existsSync(path.join(fromExec, ...DASHBOARD_CLIENT_INDEX))) {
      return fromExec;
    }
  } catch {
    // ignore
  }

  const argv1 = process.argv[1];
  if (argv1 && !argv1.startsWith("-")) {
    try {
      const fromArgv = path.resolve(path.dirname(realpathSync(argv1)), "..");
      if (existsSync(path.join(fromArgv, ...DASHBOARD_CLIENT_INDEX))) {
        return fromArgv;
      }
    } catch {
      // ignore
    }
  }

  return fromMeta;
}

function envWithLegacy(currentName: string, legacyName: string): string | undefined {
  const current = process.env[currentName];
  if (current != null && current !== "") return current;
  const legacy = process.env[legacyName];
  if (legacy != null && legacy !== "") return legacy;
  return undefined;
}

type ServeOptions = {
  repoRoot: string;
  host: string;
  port: number;
  open: boolean;
  dev: boolean;
  help: boolean;
};

function parseServeOptions(argv: string[]): ServeOptions {
  let repoRoot = process.cwd();
  let host = envWithLegacy("SKILLGRID_UI_HOST", "PRD_KANBAN_HOST") || "127.0.0.1";
  const rawPortEnv = envWithLegacy("SKILLGRID_UI_PORT", "PRD_KANBAN_PORT");
  let port =
    rawPortEnv != null && rawPortEnv !== "" && Number.isFinite(Number(rawPortEnv))
      ? Number(rawPortEnv)
      : 0;
  let open = true;
  let dev = false;
  let help = false;

  for (let i = 0; i < argv.length; i += 1) {
    const a = argv[i];
    const n = argv[i + 1];
    if (a === "--help" || a === "-h") help = true;
    else if (a === "--no-open" || a === "--open=false") open = false;
    else if (a === "--open") open = true;
    else if (a === "--dev") dev = true;
    else if (a === "--repo" && n) {
      repoRoot = resolveRepoRoot(n);
      i += 1;
    } else if (a === "--prd-dir" || a === "--prd_dir") {
      if (n) i += 1;
      throw new Error(
        "The --prd-dir option is not supported. Pass the repository root with --repo <path> (PRDs are read from <repo>/.skillgrid/prd)."
      );
    } else if (a === "--host" && n) {
      host = n;
      i += 1;
    } else if (a === "--port" && n) {
      port = Number.parseInt(n, 10);
      i += 1;
    }
  }

  return { repoRoot, host, port, open, dev, help };
}

function openBrowser(url: string): void {
  const command =
    process.platform === "darwin" ? "open" : process.platform === "win32" ? "cmd" : "xdg-open";
  const args = process.platform === "win32" ? ["/c", "start", "", url] : [url];
  const child = spawn(command, args, {
    detached: true,
    stdio: "ignore"
  });
  child.unref();
}

export function printServeHelp(): void {
  console.log(`Skillgrid Dashboard

Usage:
  skillgrid serve [options]

Options:
  --repo <path>       Repository to inspect. Defaults to the current directory.
  --host <host>       Host to bind. Defaults to 127.0.0.1.
  --port <port>       Port to bind. Defaults to an available port unless SKILLGRID_UI_PORT is set.
  --open              Open the dashboard in the browser. Default.
  --no-open           Do not open the browser.
  --dev               Serve the React UI with Vite middleware (requires devDependencies).
  --help, -h          Show this help.

Environment (optional):
  SKILLGRID_UI_PORT, SKILLGRID_UI_HOST
  PRD_KANBAN_PORT, PRD_KANBAN_HOST (deprecated aliases for port/host)
  SKILLGRID_TRUECOURSE_PORT, SKILLGRID_TRUECOURSE_HOST (TrueCourse API; default 127.0.0.1:3001)

Bundled tools (build with npm run build in skillgrid-cli):
  /gitnexus/    GitNexus web UI
  /truecourse/  TrueCourse dashboard (start API: truecourse dashboard)
`);
}

export async function runServeCommand(importMetaUrl: string, argv: string[]): Promise<void> {
  const opts = parseServeOptions(argv);
  if (opts.help) {
    printServeHelp();
    return;
  }

  if (!Number.isFinite(opts.port) || opts.port < 0 || opts.port > 65535) {
    throw new Error("Invalid --port");
  }

  const prdDir = path.join(opts.repoRoot, ".skillgrid", "prd");
  if (!existsSync(prdDir)) {
    console.error(`PRD directory not found: ${prdDir}`);
    console.error("Use --repo <path> pointing at the repository root (expects <repo>/.skillgrid/prd).");
    process.exit(1);
  }

  const packageRoot = resolveSkillgridCliPackageRoot(importMetaUrl);
  const clientRoot = path.join(packageRoot, "dist", "dashboard", "client");
  const gitnexusClientRoot = path.join(packageRoot, "dist", "dashboard", "gitnexus");
  const truecourseClientRoot = path.join(packageRoot, "dist", "dashboard", "truecourse");
  const dashboardSrcRoot = opts.dev ? path.join(packageRoot, "src", "dashboard") : undefined;
  const truecoursePort = process.env.TRUECOURSE_PORT ?? process.env.SKILLGRID_TRUECOURSE_PORT ?? "3001";
  const truecourseHost = process.env.TRUECOURSE_HOST ?? process.env.SKILLGRID_TRUECOURSE_HOST ?? "127.0.0.1";
  const truecourseApiUrl = `http://${truecourseHost}:${truecoursePort}`;

  const server = await startDashboardServer({
    repoRoot: opts.repoRoot,
    host: opts.host,
    port: opts.port,
    dev: opts.dev,
    clientRoot,
    dashboardSrcRoot,
    gitnexusClientRoot,
    truecourseClientRoot,
    truecourseApiUrl
  });

  console.log(`Skillgrid Dashboard`);
  console.log(`Repo: ${opts.repoRoot}`);
  console.log(`URL:  ${server.url}`);
  console.log(`TrueCourse UI: ${server.url}/truecourse/ (requires TrueCourse API at ${truecourseApiUrl})`);

  if (opts.open) openBrowser(server.url);

  const shutdown = async () => {
    await server.close();
    process.exit(0);
  };
  process.once("SIGINT", shutdown);
  process.once("SIGTERM", shutdown);
}
