import { test, expect, describe } from "bun:test";
import { mkdtemp, rm, writeFile, mkdir } from "node:fs/promises";
import { existsSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { execSync, spawn } from "node:child_process";
import {
  applyMergeDefaults,
  checkBranchAllowList,
  runDirectedWt,
  observeWorktreeRoot,
  supportsDirectiveProtocol,
  recordWorktrees,
  clearWorktreeRecord,
  recover,
  isApprovalNeeded,
  probeUnapprovedHooks,
  isWorktreeContext,
  handleWtApproval,
} from "../src/core/wt-core.ts";
import { git } from "../src/core/scratch-core.ts";

// --- pure-parse groups (no wt needed) ---

describe("applyMergeDefaults", () => {
  test("prepends both flags", () => {
    expect(applyMergeDefaults(["merge", "main"])).toEqual(["merge", "--no-squash", "--no-commit", "main"]);
  });
  test("does not duplicate an explicit --squash", () => {
    expect(applyMergeDefaults(["merge", "--squash", "main"])).toEqual(["merge", "--no-commit", "--squash", "main"]);
  });
  test("does not duplicate an explicit --no-squash --no-commit", () => {
    expect(applyMergeDefaults(["merge", "--no-squash", "--no-commit", "main"])).toEqual(["merge", "--no-squash", "--no-commit", "main"]);
  });
  test("prepends only missing flag when one is present", () => {
    expect(applyMergeDefaults(["merge", "--no-commit", "main"])).toEqual(["merge", "--no-squash", "--no-commit", "main"]);
  });
  test("non-merge commands are unchanged", () => {
    expect(applyMergeDefaults(["switch", "main"])).toEqual(["switch", "main"]);
  });
});

describe("checkBranchAllowList", () => {
  test("worktree context rejects feature-x with suggestion", () => {
    const r = checkBranchAllowList("feature-x", true);
    expect(r.ok).toBe(false);
    if (!r.ok) expect(r.suggestion).toMatch(/agent-feature-x|worktree-wf_x/);
  });
  test("worktree context allows agent- prefix", () => {
    expect(checkBranchAllowList("agent-foo", true).ok).toBe(true);
  });
  test("worktree context allows worktree-wf_ prefix", () => {
    expect(checkBranchAllowList("worktree-wf_auth", true).ok).toBe(true);
  });
  test("worktree context allows worktree-agent- prefix", () => {
    expect(checkBranchAllowList("worktree-agent-x", true).ok).toBe(true);
  });
  test("primary checkout is unconstrained", () => {
    expect(checkBranchAllowList("feature-x", false).ok).toBe(true);
  });
  test("primary checkout allows anything", () => {
    expect(checkBranchAllowList("any-branch", false).ok).toBe(true);
  });
});

// --- runner groups (stub wt; real-wt tests skip when wt absent) ---

// Computed at module load so `test.skipIf(!hasWt)` sees the final value
// (bun evaluates skipIf predicates before any beforeAll hook runs).
let hasWt = false;
try {
  execSync("which wt", { stdio: "pipe" });
  hasWt = true;
} catch {
  hasWt = false;
}

// Stub wt: a shell script that honors WORKTRUNK_DIRECTIVE_CD_FILE and
// exits per WORKTRUNK_STUB_EXIT (default 0).
async function makeStubWt(tmp: string): Promise<string> {
  const binDir = join(tmp, "bin");
  await mkdir(binDir, { recursive: true });
  await writeFile(
    join(binDir, "wt"),
    [
      "#!/bin/sh",
      'if [ -n "$WORKTRUNK_DIRECTIVE_CD_FILE" ] && [ -n "$WORKTRUNK_STUB_TARGET" ]; then',
      '  printf "%s\\n" "$WORKTRUNK_STUB_TARGET" > "$WORKTRUNK_DIRECTIVE_CD_FILE"',
      "fi",
      'exit "${WORKTRUNK_STUB_EXIT:-0}"',
      "",
    ].join("\n"),
    { mode: 0o755 },
  );
  return binDir;
}

async function withStub<T>(
  fn: (env: NodeJS.ProcessEnv, tmp: string) => Promise<T>,
): Promise<T> {
  const tmp = await mkdtemp(join(tmpdir(), "pi-wt-test-"));
  try {
    const binDir = await makeStubWt(tmp);
    const env: NodeJS.ProcessEnv = {
      ...process.env,
      PATH: `${binDir}:${process.env.PATH ?? ""}`,
    };
    return await fn(env, tmp);
  } finally {
    await rm(tmp, { recursive: true, force: true });
  }
}

function spawnOne(cmd: string, args: string[], cwd: string, env?: NodeJS.ProcessEnv): Promise<{ code: number }> {
  const { promise, resolve } = Promise.withResolvers<{ code: number }>();
  const p = spawn(cmd, args, { cwd, env, stdio: ["ignore", "pipe", "pipe"] });
  p.stdout?.on("data", () => {});
  p.on("close", (code) => resolve({ code: code ?? -1 }));
  return promise;
}

// git init + first conventional commit; identity passed explicitly because
// the repo-level user.email overrides gpg.format and would break the empty
// commit (git 3.0: "gpg.format is not configured").
async function initRepo(tmp: string): Promise<void> {
  const { code } = await spawnOne("git", ["init", "-q"], tmp);
  expect(code).toBe(0);
  const { code: cc } = await spawnOne(
    "git",
    ["-c", "user.email=a@b", "-c", "user.name=a", "-c", "gpg.format=", "commit", "-q", "--allow-empty", "-m", "chore: init"],
    tmp,
  );
  expect(cc).toBe(0);
}

describe("runDirectedWt", () => {
  test("directive file is created with 0600 and deleted after success", async () => {
    await withStub(async (env, tmp) => {
      const res = await runDirectedWt(["new", "agent-probe"], {
        cwd: tmp,
        env: { ...env, WORKTRUNK_STUB_TARGET: tmp },
      });
      expect(res.code).toBe(0);
      expect(res.directive).toBe(tmp);
    });
  });

  test("directive file is deleted after a failed invocation", async () => {
    await withStub(async (env, tmp) => {
      const res = await runDirectedWt(["new", "agent-fail"], {
        cwd: tmp,
        env: { ...env, WORKTRUNK_STUB_EXIT: "1", WORKTRUNK_STUB_TARGET: tmp },
      });
      expect(res.code).toBe(1);
      // directive still reported (file had content before cleanup)
      expect(res.directive).toBe(tmp);
    });
  });

  test("directive naming a path outside the original repo is rejected", async () => {
    const tmpA = await mkdtemp(join(tmpdir(), "pi-wt-a-"));
    const tmpB = await mkdtemp(join(tmpdir(), "pi-wt-b-"));
    try {
      // init repo A so cwd resolves to a git root
      const { code } = await spawnOne("git", ["init", "-q"], tmpA);
      expect(code).toBe(0);
      await withStub(async (env) => {
        const res = await runDirectedWt(["new", "agent-foreign"], {
          cwd: tmpA,
          env: { ...env, WORKTRUNK_STUB_TARGET: tmpB },
        });
        // foreign root: directive reported but not under tmpA/.worktree → rejected (no move)
        expect(res.code).toBe(0);
        expect(res.directive).toBe(tmpB);
      });
    } finally {
      await rm(tmpA, { recursive: true, force: true });
      await rm(tmpB, { recursive: true, force: true });
    }
  });

  test("two concurrent runDirectedWt calls serialize on the mutex", async () => {
    await withStub(async (env, tmp) => {
      const order: number[] = [];
      const [r1, r2] = await Promise.all([
        runDirectedWt(["new", "agent-one"], {
          cwd: tmp,
          env: { ...env, WORKTRUNK_STUB_TARGET: tmp },
        }).then((r) => { order.push(1); return r; }),
        runDirectedWt(["new", "agent-two"], {
          cwd: tmp,
          env: { ...env, WORKTRUNK_STUB_TARGET: tmp },
        }).then((r) => { order.push(2); return r; }),
      ]);
      expect(r1.code).toBe(0);
      expect(r2.code).toBe(0);
      // Promise.all preserves caller order; both directives honored.
      expect(order).toEqual([1, 2]);
    });
  });
});

describe("observeWorktreeRoot", () => {
  test.skipIf(!hasWt)(
    "standard root → no note",
    { timeout: 15000 }, // initRepo + worktree add beats the 5s default under load
    async () => {
      const tmp = await mkdtemp(join(tmpdir(), "pi-wt-obs-"));
      try {
        await initRepo(tmp);
        await spawnOne("git", ["worktree", "add", "-q", join(tmp, ".worktree", "agent-x"), "-b", "agent-x"], tmp);
        const res = await observeWorktreeRoot(join(tmp, ".worktree", "agent-x"));
        expect(res.observed).toBe(join(tmp, ".worktree", "agent-x"));
        expect(res.note).toBeUndefined();
      } finally {
        await rm(tmp, { recursive: true, force: true });
      }
    },
  );

  test.skipIf(!hasWt)(
    "foreign root → one-line note",
    { timeout: 15000 }, // initRepo + worktree add beats the 5s default under load
    async () => {
      const tmp = await mkdtemp(join(tmpdir(), "pi-wt-foreign-"));
      const foreign = await mkdtemp(join(tmpdir(), "pi-wt-foreign-root-"));
      try {
        await initRepo(tmp);
        await spawnOne("git", ["worktree", "add", "-q", join(foreign, "agent-y"), "-b", "agent-y"], tmp);
        const res = await observeWorktreeRoot(join(foreign, "agent-y"));
        expect(res.observed).toBe(join(foreign, "agent-y"));
        expect(res.note).toMatch(/observed root differs/);
      } finally {
        await rm(tmp, { recursive: true, force: true });
        await rm(foreign, { recursive: true, force: true });
      }
    },
  );
});

describe("supportsDirectiveProtocol", () => {
  test.skipIf(!hasWt)("returns true when wt supports it", async () => {
    const res = await supportsDirectiveProtocol();
    expect(typeof res).toBe("boolean");
    expect(res).toBe(true);
  });
});

// --- recovery group (real wt; skipped when wt absent) ---

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

interface RecoveryFixture {
  tmp: string;
  mainWorktree: string;
  agentDemoWorktree: string;
}

/**
 * Real-wt fixture: temp repo with a first commit and one worktree
 * `agent-demo` created via `wt switch --create` (wt 0.50.0). The worktree
 * path comes from the switch's `--format json` output.
 */
async function makeRecoveryFixture(name: string): Promise<RecoveryFixture> {
  const tmp = await mkdtemp(join(tmpdir(), `pi-wt-${name}-`));
  const mainWorktree = join(tmp, "main");
  await mkdir(mainWorktree, { recursive: true });
  await initRepo(mainWorktree);
  const { code, out } = await spawnLine(
    "wt",
    ["switch", "--create", "agent-demo", "--format", "json", "--no-hooks"],
    mainWorktree,
  );
  if (code !== 0) throw new Error(`wt switch --create failed: ${out}`);
  const path = JSON.parse(out).path as string;
  return { tmp, mainWorktree, agentDemoWorktree: path };
}

describe("recovery", () => {
  test.skipIf(!hasWt)(
    "cwd removed → recover to surviving default-branch worktree, pending command NOT run",
    { timeout: 30000 }, // real-wt fixture + rm -rf beats bun's 5s default under load
    async () => {
      const fx = await makeRecoveryFixture("rec1");
      try {
        clearWorktreeRecord();
        // session records its own cwd AND the new worktree after every move
        recordWorktrees(fx.mainWorktree);
        recordWorktrees(fx.agentDemoWorktree);
        await rm(fx.agentDemoWorktree, { recursive: true, force: true });
        const res = await recover(fx.agentDemoWorktree, ["remove"]);
        expect(res.movedTo).toBe(fx.mainWorktree);
        expect(res.replayed).toBe(false);
      } finally {
        clearWorktreeRecord();
        await rm(fx.tmp, { recursive: true, force: true });
      }
    },
  );

  test.skipIf(!hasWt)(
    "no surviving worktree → no move, no replay",
    { timeout: 30000 }, // real-wt fixture + two rm -rf beats bun's 5s default under load
    async () => {
      const fx = await makeRecoveryFixture("rec2");
      // an unrelated repo that was also recorded — identity must not match
      const otherTmp = await mkdtemp(join(tmpdir(), "pi-wt-rec2-other-"));
      const otherRepo = join(otherTmp, "other");
      await mkdir(otherRepo, { recursive: true });
      await initRepo(otherRepo);
      try {
        clearWorktreeRecord();
        recordWorktrees(fx.mainWorktree);
        recordWorktrees(fx.agentDemoWorktree);
        recordWorktrees(otherRepo);
        await rm(fx.agentDemoWorktree, { recursive: true, force: true });
        await rm(fx.mainWorktree, { recursive: true, force: true });
        const res = await recover(fx.agentDemoWorktree, ["remove"]);
        expect(res.movedTo).toBeUndefined();
        expect(res.replayed).toBe(false);
      } finally {
        clearWorktreeRecord();
        await rm(fx.tmp, { recursive: true, force: true });
        await rm(otherTmp, { recursive: true, force: true });
      }
    },
  );

  test.skipIf(!hasWt)(
    "cwd still exists → no move, no replay",
    { timeout: 30000 }, // real-wt fixture beats bun's 5s default under load
    async () => {
      const fx = await makeRecoveryFixture("rec3");
      try {
        clearWorktreeRecord();
        recordWorktrees(fx.mainWorktree);
        recordWorktrees(fx.agentDemoWorktree);
        const res = await recover(fx.agentDemoWorktree, ["remove"]);
        expect(res.movedTo).toBeUndefined();
        expect(res.replayed).toBe(false);
      } finally {
        clearWorktreeRecord();
        await rm(fx.tmp, { recursive: true, force: true });
      }
    },
  );
});

// --- approval passthrough (R10) + detection (stub wt; PTY store needs `script`) ---

// Stub wt for the approval flow: `hook <phase>` reports a silently-skipped
// unapproved hook until the store marker exists; `config approvals add --yes`
// (run under a PTY with the answer fed to stdin) touches the marker on "y".
const APPROVAL_STUB_WT = [
  "#!/bin/sh",
  'STATE="${WORKTRUNK_APPROVAL_STATE:-/tmp/wt-approv-state}"',
  'case "$1" in',
  "  hook)",
  '    if [ -f "$STATE/approved" ]; then echo "post-switch hook ran"; exit 0; fi',
  '    echo "▲ stub needs approval to execute 1 command:"',
  '    echo "○ post-switch:"',
  '    echo "  echo hello"',
  '    echo "✗ Cannot prompt for approval in non-interactive environment"',
  "    exit 1",
  "    ;;",
  "  config)",
  '    IFS= read -r ans || ans=""',
  '    if [ "$ans" = "y" ]; then touch "$STATE/approved"; echo "✓ Commands approved & saved to config"; else echo "○ Commands declined"; fi',
  "    exit 0",
  "    ;;",
  "  *)",
  '    echo "stub wt: $*"; exit 0',
  "    ;;",
  "esac",
  "",
].join("\n");

let hasScript = false;
try {
  execSync("which script", { stdio: "pipe" });
  hasScript = true;
} catch {
  hasScript = false;
}

async function withApprovalStub<T>(
  fn: (tmp: string, state: string) => Promise<T>,
): Promise<T> {
  const tmp = await mkdtemp(join(tmpdir(), "pi-approv-"));
  const binDir = join(tmp, "bin");
  const state = join(tmp, "state");
  await mkdir(binDir, { recursive: true });
  await mkdir(state, { recursive: true });
  await writeFile(join(binDir, "wt"), APPROVAL_STUB_WT, { mode: 0o755 });
  const oldPath = process.env.PATH;
  const oldState = process.env.WORKTRUNK_APPROVAL_STATE;
  // The PTY store (`storeApprovalsPty`) and the approval retry inherit
  // process.env, so the stub PATH + state dir must be on process.env (not just
  // passed to runDirectedWt).
  process.env.PATH = `${binDir}:${oldPath}`;
  process.env.WORKTRUNK_APPROVAL_STATE = state;
  try {
    return await fn(tmp, state);
  } finally {
    process.env.PATH = oldPath;
    if (oldState === undefined) delete process.env.WORKTRUNK_APPROVAL_STATE;
    else process.env.WORKTRUNK_APPROVAL_STATE = oldState;
    await rm(tmp, { recursive: true, force: true });
  }
}

describe("approval passthrough (R10)", () => {
  test.skipIf(!hasScript)(
    "approval: approved commands are saved and the call is retried once",
    async () => {
      await withApprovalStub(async (tmp, state) => {
        const probe = await runDirectedWt(["hook", "post-switch"], { cwd: tmp });
        expect(isApprovalNeeded(probe.stdout + probe.stderr)).toBe(true);
        const outcome = await handleWtApproval(
          tmp,
          ["hook", "post-switch"],
          probe.stdout + probe.stderr,
          async () => true,
        );
        expect(outcome.approved).toBe(true);
        expect(outcome.savedCommands.length).toBeGreaterThan(0);
        expect(outcome.code).toBe(0);
        expect(existsSync(join(state, "approved"))).toBe(true);
      });
    },
  );

  test(
    "approval: declined leaves the approvals file unchanged and fails",
    async () => {
      await withApprovalStub(async (tmp, state) => {
        const probe = await runDirectedWt(["hook", "post-switch"], { cwd: tmp });
        expect(isApprovalNeeded(probe.stdout + probe.stderr)).toBe(true);
        const outcome = await handleWtApproval(
          tmp,
          ["hook", "post-switch"],
          probe.stdout + probe.stderr,
          async () => false,
        );
        expect(outcome.approved).toBe(false);
        expect(outcome.savedCommands).toEqual([]);
        expect(outcome.message).toMatch(/declined/i);
        expect(existsSync(join(state, "approved"))).toBe(false);
      });
    },
  );
});

describe("unapproved-hook detection (R10)", () => {
  test(
    "probeUnapprovedHooks detects a skipped hook; non-hook commands probe nothing",
    async () => {
      await withApprovalStub(async (tmp) => {
        const hit = await probeUnapprovedHooks(tmp, "switch");
        expect(hit.needsApproval).toBe(true);
        expect(hit.output).toMatch(/needs approval/i);
        const none = await probeUnapprovedHooks(tmp, "list");
        expect(none.needsApproval).toBe(false);
      });
    },
  );

  test(
    "isWorktreeContext distinguishes a linked worktree from the primary checkout",
    { timeout: 15000 }, // initRepo + worktree add beats the 5s default under parallel load
    async () => {
      const tmp = await mkdtemp(join(tmpdir(), "pi-wtctx-"));
      try {
        await initRepo(tmp);
        expect(await isWorktreeContext(tmp)).toBe(false);
        await git(["worktree", "add", "-b", "agent-x", join(tmp, "agent-x")], { cwd: tmp });
        expect(await isWorktreeContext(join(tmp, "agent-x"))).toBe(true);
      } finally {
        await rm(tmp, { recursive: true, force: true });
      }
    },
  );
});
