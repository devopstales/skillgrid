import { spawn } from "node:child_process";
import { git, repoRoot } from "./core/scratch-core.ts";

// ---------------------------------------------------------------------------
// Marker lifecycle — the SOLE writer of worktrunk marker state (Global
// Constraint, blueprint Task 8). Both extensions import this file; neither
// writes markers directly.
//
// Lifecycle (findings.md R1 marker table):
//   session_start    → markWaiting   (💬)
//   agent_start      → markWorking   (🤖)
//   agent_end        → markWaiting   (💬)
//   session_shutdown → clearMarker
//
// The marker targets the branch of the session's cwd (wt 0.50.0 has no
// --cwd flag for `config state marker`; it writes
// `worktrunk.state.<branch>.marker` via `--branch <branch>`). A stale marker
// (kill before session_shutdown) is reported and manually clearable with
// `wt config state marker clear`. No dual-plugin runtime detection
// (Goldfish Stage 5 R1-F003) — the README note covers the conflict.
// ---------------------------------------------------------------------------

/**
 * Run `wt` with an explicit argv — never through a shell.
 * Resolves with stdout on exit 0; rejects with an Error carrying stderr.
 */
function wt(args: string[], cwd: string): Promise<string> {
  const { promise, resolve, reject } = Promise.withResolvers<string>();
  const child = spawn("wt", args, { cwd, stdio: ["pipe", "pipe", "pipe"] });
  let stdout = "";
  let stderr = "";
  child.stdout.on("data", (d: Buffer) => (stdout += d));
  child.stderr.on("data", (d: Buffer) => (stderr += d));
  child.on("error", (err) => reject(err));
  child.on("close", (code) => {
    if (code === 0) resolve(stdout);
    else reject(new Error(stderr.trim() || `wt ${args.join(" ")} exited ${code}`));
  });
  return promise;
}

/** `wt config state marker set <emoji> --branch <branch>` in the session's repo root. */
async function setMarker(cwd: string, marker: string): Promise<void> {
  const branch = (await git(["rev-parse", "--abbrev-ref", "HEAD"], { cwd })).trim();
  await wt(["config", "state", "marker", "set", marker, "--branch", branch], cwd);
}

/** `wt config state marker clear --branch <branch>` in the session's repo root. */
async function clearMarkerFor(cwd: string): Promise<void> {
  const branch = (await git(["rev-parse", "--abbrev-ref", "HEAD"], { cwd })).trim();
  await wt(["config", "state", "marker", "clear", "--branch", branch], cwd);
}

/** 💬 — the session is idle, waiting for input (session_start, agent_end). */
export async function markWaiting(cwd: string): Promise<void> {
  await setMarker(cwd, "💬");
}

/** 🤖 — the agent is working (agent_start). */
export async function markWorking(cwd: string): Promise<void> {
  await setMarker(cwd, "🤖");
}

/** Remove the marker (session_shutdown). */
export async function clearMarker(cwd: string): Promise<void> {
  await clearMarkerFor(cwd);
}

/**
 * The current marker of the branch of `cwd` (read path for tests + the
 * stale report): `wt config state marker get --branch <branch> --format json`
 * → the marker string, or undefined when none is set.
 */
export async function markerStatus(cwd: string): Promise<string | undefined> {
  const branch = (await git(["rev-parse", "--abbrev-ref", "HEAD"], { cwd })).trim();
  const out = await wt(["config", "state", "marker", "get", "--branch", branch, "--format", "json"], cwd);
  const parsed = JSON.parse(out) as { marker?: string } | null;
  return parsed?.marker;
}

/**
 * Stale detection (blueprint Task 8 Step 3): a marker survives only when the
 * session was killed before session_shutdown, i.e. it was never cleared.
 * `marker` is the leftover value (💬/🤖); the manual-clear command is the
 * exact one from findings.md R1.
 */
export function staleMarkerReport(cwd: string, marker: string): string {
  return `stale marker in ${cwd} (${marker}) — run \`wt config state marker clear\` to remove`;
}
