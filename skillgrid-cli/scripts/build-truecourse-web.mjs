#!/usr/bin/env node
/**
 * Vendors a production build of TrueCourse dashboard client from
 * https://github.com/truecourse-ai/truecourse into dist/dashboard/truecourse
 * so Skillgrid can serve it at /truecourse/ (API: TrueCourse server on :3001).
 *
 * Env:
 *   SKIP_TRUECOURSE_WEB=1  — skip (no network / no pnpm).
 *   TRUECOURSE_REF=main      — git ref (shallow clone).
 *   TRUECOURSE_FORCE_INSTALL=1 — always run pnpm install.
 */
import { spawnSync } from "node:child_process";
import { cpSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = path.join(path.dirname(fileURLToPath(import.meta.url)), "..");
const CLONE_DIR = path.join(ROOT, "tmp", "truecourse-src");
const CLIENT_DIR = path.join(CLONE_DIR, "apps", "dashboard", "client");
const VITE_CONFIG = path.join(CLIENT_DIR, "vite.config.ts");
const APP_TSX = path.join(CLIENT_DIR, "src", "App.tsx");
const OUT_DIR = path.join(ROOT, "dist", "dashboard", "truecourse");
const REF = process.env.TRUECOURSE_REF || "main";

function run(cmd, args, opts) {
  const r = spawnSync(cmd, args, { stdio: "inherit", shell: false, ...opts });
  if (r.status !== 0) process.exit(r.status ?? 1);
}

function commandOk(cmd, args) {
  const r = spawnSync(cmd, args, { stdio: "ignore", shell: false });
  return r.status === 0;
}

/** @type {readonly [string, string[]]} */
let pnpmCmd = ["pnpm", []];

function resolvePnpm() {
  if (commandOk("pnpm", ["--version"])) {
    pnpmCmd = ["pnpm", []];
    return;
  }
  if (commandOk("npx", ["pnpm", "--version"])) {
    pnpmCmd = ["npx", ["pnpm"]];
    return;
  }
  console.error(
    "build-truecourse-web: pnpm is required (install pnpm or use npx). Set SKIP_TRUECOURSE_WEB=1 to skip."
  );
  process.exit(1);
}

function runPnpm(args, opts) {
  const [cmd, prefix] = pnpmCmd;
  run(cmd, [...prefix, ...args], opts);
}

function patchViteBase() {
  let s = readFileSync(VITE_CONFIG, "utf8");
  if (s.includes("base: '/truecourse/'") || s.includes('base: "/truecourse/"')) return;
  const needle = "export default defineConfig({";
  if (!s.includes(needle)) {
    console.error("build-truecourse-web: unexpected vite.config.ts; cannot inject base.");
    process.exit(1);
  }
  s = s.replace(needle, `${needle}\n  base: '/truecourse/',`);
  writeFileSync(VITE_CONFIG, s);
}

function patchRouterBasename() {
  let s = readFileSync(APP_TSX, "utf8");
  if (s.includes('basename="/truecourse"') || s.includes("basename='/truecourse'")) return;
  if (!s.includes("<BrowserRouter>")) {
    console.error("build-truecourse-web: unexpected App.tsx; cannot inject BrowserRouter basename.");
    process.exit(1);
  }
  s = s.replace("<BrowserRouter>", '<BrowserRouter basename="/truecourse">');
  writeFileSync(APP_TSX, s);
}

function main() {
  if (process.env.SKIP_TRUECOURSE_WEB === "1") {
    console.warn("build-truecourse-web: SKIP_TRUECOURSE_WEB=1 — skipping TrueCourse web bundle.");
    return;
  }

  const major = Number.parseInt(process.versions.node.split(".")[0] ?? "0", 10);
  if (major < 20) {
    console.error(
      "build-truecourse-web: TrueCourse requires Node >= 20. Upgrade Node or set SKIP_TRUECOURSE_WEB=1."
    );
    process.exit(1);
  }

  resolvePnpm();

  mkdirSync(path.dirname(CLONE_DIR), { recursive: true });

  if (!existsSync(path.join(CLONE_DIR, ".git"))) {
    rmSync(CLONE_DIR, { recursive: true, force: true });
    run("git", ["clone", "--depth", "1", "--branch", REF, "https://github.com/truecourse-ai/truecourse.git", CLONE_DIR], {
      cwd: ROOT
    });
  } else {
    console.log(
      "build-truecourse-web: reusing tmp/truecourse-src — delete skillgrid-cli/tmp/truecourse-src to pull a fresh upstream checkout."
    );
  }

  const nodeModules = path.join(CLONE_DIR, "node_modules");
  if (!existsSync(nodeModules) || process.env.TRUECOURSE_FORCE_INSTALL === "1") {
    runPnpm(["install", "--frozen-lockfile"], { cwd: CLONE_DIR, env: { ...process.env } });
  }

  runPnpm(["--filter", "@truecourse/shared", "build"], { cwd: CLONE_DIR, env: { ...process.env } });

  patchViteBase();
  patchRouterBasename();

  runPnpm(["--filter", "@truecourse/dashboard-client", "build"], { cwd: CLONE_DIR, env: { ...process.env } });

  const built = path.join(CLIENT_DIR, "dist");
  if (!existsSync(path.join(built, "index.html"))) {
    console.error("build-truecourse-web: dashboard client build produced no dist/index.html");
    process.exit(1);
  }

  mkdirSync(path.dirname(OUT_DIR), { recursive: true });
  rmSync(OUT_DIR, { recursive: true, force: true });
  cpSync(built, OUT_DIR, { recursive: true });
  console.log(`build-truecourse-web: copied TrueCourse dashboard to ${OUT_DIR}`);
}

main();
