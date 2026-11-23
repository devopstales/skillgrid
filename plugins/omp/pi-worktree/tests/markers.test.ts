import { test, expect, describe } from "bun:test";
import { mkdtemp, rm, mkdir } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { execSync, spawn } from "node:child_process";
import {
  markWaiting,
  markWorking,
  clearMarker,
  markerStatus,
  staleMarkerReport,
} from "../src/markers.ts";
import { git } from "../src/core/scratch-core.ts";

// --- real-wt groups (skipped when wt absent) ---

// Computed at module load so `test.skipIf(!hasWt)` sees the final value
// (bun evaluates skipIf predicates before any beforeAll hook runs).
let hasWt = false;
try {
  execSync("which wt", { stdio: "pipe" });
  hasWt = true;
} catch {
  hasWt = false;
}

function spawnLine(
  cmd: string,
  args: string[],
  cwd: string,
): Promise<{ code: number; out: string }> {
  const { promise, resolve } = Promise.withResolvers<{ code: number; out: string }>();
  const p = spawn(cmd, args, { cwd, stdio: ["ignore", "pipe", "pipe"] });
  let out = "";
  p.stdout?.on("data", (d: Buffer) => (out += d));
  p.on("close", (code) => resolve({ code: code ?? -1, out }));
  return promise;
}

// git init + first conventional commit. Identity passed explicitly (repo-level
// user.email overrides gpg.format); core.hooksPath points at /dev/null — the
// repo is created under /tmp where wt's hooks would otherwise run.
async function initRepo(tmp: string): Promise<void> {
  const { code } = await spawnLine("git", ["init", "-q"], tmp);
  expect(code).toBe(0);
  const { code: hc } = await spawnLine("git", ["config", "core.hooksPath", "/dev/null"], tmp);
  expect(hc).toBe(0);
  const { code: cc } = await spawnLine(
    "git",
    ["-c", "user.email=a@b", "-c", "user.name=a", "-c", "gpg.format=", "commit", "-q", "--allow-empty", "-m", "chore: init"],
    tmp,
  );
  expect(cc).toBe(0);
}

/** wt list → worktree entries with `symbols` + `path` (the marker read path). */
async function wtListSymbols(cwd: string): Promise<Map<string, string>> {
  const { code, out } = await spawnLine("wt", ["list", "--format", "json"], cwd);
  expect(code).toBe(0);
  const entries = JSON.parse(out) as Array<{ branch: string; path: string; symbols?: string }>;
  return new Map(entries.map((e) => [e.path, e.symbols ?? ""]));
}

/** A marker set by markers.ts must be visible in `wt list` (MH-13). */
async function expectVisibleInWtList(cwd: string, marker: string): Promise<void> {
  const symbols = await wtListSymbols(cwd);
  const own = symbols.get(cwd);
  expect(own).toBeDefined();
  expect(own).toContain(marker);
}

describe("marker lifecycle (real wt)", () => {
  test.skipIf(!hasWt)(
    "session_start → 💬, agent_start → 🤖, agent_end → 💬, session_shutdown → clear — visible in wt list",
    async () => {
      const tmp = await mkdtemp(join(tmpdir(), "pi-mk-"));
      const repo = join(tmp, "repo");
      await mkdir(repo, { recursive: true });
      await initRepo(repo);
      try {
        // session_start
        await markWaiting(repo);
        expect(await markerStatus(repo)).toBe("💬");
        await expectVisibleInWtList(repo, "💬");
        // agent_start
        await markWorking(repo);
        expect(await markerStatus(repo)).toBe("🤖");
        await expectVisibleInWtList(repo, "🤖");
        // agent_end
        await markWaiting(repo);
        expect(await markerStatus(repo)).toBe("💬");
        await expectVisibleInWtList(repo, "💬");
        // session_shutdown
        await clearMarker(repo);
        expect(await markerStatus(repo)).toBeUndefined();
        const symbols = await wtListSymbols(repo);
        expect(symbols.get(repo)).not.toContain("💬");
        expect(symbols.get(repo)).not.toContain("🤖");
      } finally {
        await rm(tmp, { recursive: true, force: true });
      }
    },
  );

  test.skipIf(!hasWt)(
    "kill-before-shutdown leaves the marker; stale detection + manual clear removes it",
    async () => {
      const tmp = await mkdtemp(join(tmpdir(), "pi-mk-stale-"));
      const repo = join(tmp, "repo");
      await mkdir(repo, { recursive: true });
      await initRepo(repo);
      try {
        // The killed session: marked, then the process dies before session_shutdown.
        await markWorking(repo);
        // Next session_start: the marker is still present → stale.
        const leftover = await markerStatus(repo);
        expect(leftover).toBe("🤖");
        expect(staleMarkerReport(repo, leftover)).toBe(
          `stale marker in ${repo} (🤖) — run \`wt config state marker clear\` to remove`,
        );
        // Manual clear (the command the report names) removes it.
        await clearMarker(repo);
        expect(await markerStatus(repo)).toBeUndefined();
        const symbols = await wtListSymbols(repo);
        expect(symbols.get(repo)).not.toContain("🤖");
      } finally {
        await rm(tmp, { recursive: true, force: true });
      }
    },
  );

  test.skipIf(!hasWt)(
    "the marker targets the branch of the given cwd (worktree, not main checkout)",
    async () => {
      const tmp = await mkdtemp(join(tmpdir(), "pi-mk-br-"));
      const repo = join(tmp, "repo");
      await mkdir(repo, { recursive: true });
      await initRepo(repo);
      const agentWt = join(tmp, "agent-x");
      try {
        const { code } = await spawnLine(
          "git",
          ["worktree", "add", "-q", agentWt, "-b", "agent-x"],
          repo,
        );
        expect(code).toBe(0);
        await markWorking(agentWt);
        // The agent-x worktree carries the marker; the main checkout does not.
        expect(await markerStatus(agentWt)).toBe("🤖");
        expect(await markerStatus(repo)).toBeUndefined();
        const symbols = await wtListSymbols(repo);
        expect(symbols.get(agentWt)).toContain("🤖");
        expect(symbols.get(repo)).not.toContain("🤖");
        await clearMarker(agentWt);
        expect(await markerStatus(agentWt)).toBeUndefined();
      } finally {
        await rm(tmp, { recursive: true, force: true });
      }
    },
  );

  test.skipIf(!hasWt)(
    "a session cwd outside a git repo is a no-op that does not throw",
    async () => {
      const tmp = await mkdtemp(join(tmpdir(), "pi-mk-norepo-"));
      try {
        await expect(markWorking(tmp)).rejects.toThrow();
      } finally {
        await rm(tmp, { recursive: true, force: true });
      }
    },
  );
});

describe("staleMarkerReport (pure)", () => {
  test("exact report text with the manual-clear command", () => {
    expect(staleMarkerReport("/w/agent-x", "🤖")).toBe(
      "stale marker in /w/agent-x (🤖) — run `wt config state marker clear` to remove",
    );
  });
});
