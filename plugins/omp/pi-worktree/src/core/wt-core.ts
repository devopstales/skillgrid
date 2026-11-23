import { execFileSync, spawn } from "node:child_process";
import { mkdtemp, rm, writeFile, readFile, chmod, stat } from "node:fs/promises";
import { tmpdir } from "node:os";
import { isAbsolute, join, resolve as resolvePath } from "node:path";
import { AGENT_BRANCH_RE } from "./constants.ts";
import { git } from "./scratch-core.ts";

// ---------------------------------------------------------------------------
// Pure-parse invariants
// ---------------------------------------------------------------------------

/**
 * Merge invariance: `wt merge` always runs as `--no-squash --no-commit`
 * unless the caller explicitly requested squash (`--squash` is preserved).
 * The input array is never mutated.
 */
export function applyMergeDefaults(args: string[]): string[] {
  if (args[0] !== "merge") return args;
  const rest = args.slice(1);
  const out: string[] = ["merge"];
  const wantsSquash = rest.includes("--squash");
  if (!wantsSquash && !rest.includes("--no-squash")) out.push("--no-squash");
  if (!rest.includes("--no-commit")) out.push("--no-commit");
  out.push(...rest);
  return out;
}

/**
 * Allow-list invariance: worktree-context branches must match AGENT_BRANCH_RE;
 * the primary checkout is unconstrained.
 */
export function checkBranchAllowList(
  branch: string,
  isWorktree: boolean,
): { ok: true } | { ok: false; suggestion: string } {
  if (!isWorktree) return { ok: true };
  if (AGENT_BRANCH_RE.test(branch)) return { ok: true };
  return { ok: false, suggestion: `agent-${branch}` };
}

// ---------------------------------------------------------------------------
// In-process mutex (serializes every runDirectedWt call)
// ---------------------------------------------------------------------------

let _mutex: Promise<void> = Promise.resolve();
function withMutex<T>(fn: () => Promise<T>): Promise<T> {
  const prev = _mutex;
  let release!: () => void;
  _mutex = new Promise<void>((r) => { release = r; });
  return prev.then(fn).finally(() => release());
}

// ---------------------------------------------------------------------------
// Directive runner
// ---------------------------------------------------------------------------

export interface DirectedWtResult {
  stdout: string;
  stderr: string;
  code: number;
  /** Target path written by wt via WORKTRUNK_DIRECTIVE_CD_FILE, if any. */
  directive?: string;
}

export interface DirectedWtOptions {
  cwd?: string;
  signal?: AbortSignal;
  timeoutMs?: number;
  env?: NodeJS.ProcessEnv;
}

/**
 * Run `wt` with the directive-file protocol:
 * - a per-call temp dir + 0600 directive file, advertised via
 *   WORKTRUNK_DIRECTIVE_CD_FILE,
 * - PWD/WORKTRUNK_SHELL_CWD pinned to cwd,
 * - the temp dir is always removed (success or failure),
 * - all calls serialize on the module mutex.
 */
export function runDirectedWt(
  args: string[],
  opts: DirectedWtOptions = {},
): Promise<DirectedWtResult> {
  return withMutex(async () => {
    const cwd = opts.cwd ?? process.cwd();
    const tempDir = await mkdtemp(join(tmpdir(), "pi-wt-dir-"));
    const directiveFile = join(tempDir, "directive");
    await writeFile(directiveFile, "");
    await chmod(directiveFile, 0o600);

    const env: NodeJS.ProcessEnv = {
      ...process.env,
      ...(opts.env ?? {}),
      WORKTRUNK_DIRECTIVE_CD_FILE: directiveFile,
      WORKTRUNK_SHELL_CWD: cwd,
      PWD: cwd,
    };

    return await new Promise<DirectedWtResult>((resolveP) => {
      const child = spawn("wt", args, {
        cwd,
        env,
        stdio: ["pipe", "pipe", "pipe"],
        signal: opts.signal,
      });
      let stdout = "";
      let stderr = "";
      let settled = false;

      const finish = async (code: number) => {
        if (settled) return;
        settled = true;
        let directive: string | undefined;
        try {
          const content = (await readFile(directiveFile, "utf8")).trim();
          if (content) directive = content;
        } catch {
          // directive not written → absent
        }
        await rm(tempDir, { recursive: true, force: true });
        resolveP({ stdout, stderr, code, directive });
      };

      child.stdout?.on("data", (d: Buffer) => (stdout += d));
      child.stderr?.on("data", (d: Buffer) => (stderr += d));
      child.on("error", () => {
        void finish(-1);
      });
      child.on("close", (code) => {
        void finish(code ?? -1);
      });

      if (opts.timeoutMs !== undefined) {
        const t = setTimeout(() => child.kill("SIGTERM"), opts.timeoutMs);
        child.on("close", () => clearTimeout(t));
      }
    });
  });
}

// ---------------------------------------------------------------------------
// Directive protocol capability probe
// ---------------------------------------------------------------------------

/**
 * True when the `wt` on PATH supports the WORKTRUNK_DIRECTIVE_CD_FILE
 * protocol. False when wt is missing or never writes the directive file.
 */
export async function supportsDirectiveProtocol(): Promise<boolean> {
  // 1) wt must exist and report a version
  const version = await new Promise<boolean>((resolveP) => {
    const p = spawn("wt", ["--version"], { stdio: ["ignore", "pipe", "pipe"] });
    p.on("error", () => resolveP(false));
    p.on("close", (code) => resolveP(code === 0));
  });
  if (!version) return false;

  // 2) run `wt switch --create` in a temp git repo with the directive env set
  const tempRepo = await mkdtemp(join(tmpdir(), "pi-wt-probe-"));
  const tempDir = await mkdtemp(join(tmpdir(), "pi-wt-dir-"));
  const directiveFile = join(tempDir, "directive");
  await writeFile(directiveFile, "");
  await chmod(directiveFile, 0o600);
  try {
    const init = await new Promise<boolean>((resolveP) => {
      const p = spawn("git", ["init", "-q"], { cwd: tempRepo, stdio: ["ignore", "pipe", "pipe"] });
      p.on("error", () => resolveP(false));
      p.on("close", (code) => resolveP(code === 0));
    });
    if (!init) return false;

    // wt's conventional-commit hook requires a first commit before switch
    const commit = await new Promise<boolean>((resolveP) => {
      const p = spawn("git", ["commit", "-q", "--allow-empty", "-m", "chore: init"], {
        cwd: tempRepo,
        stdio: ["ignore", "pipe", "pipe"],
      });
      p.on("error", () => resolveP(false));
      p.on("close", (code) => resolveP(code === 0));
    });
    if (!commit) return false;

    const ran = await new Promise<boolean>((resolveP) => {
      const p = spawn("wt", ["switch", "--create", "agent-wtprobe", "-y"], {
        cwd: tempRepo,
        stdio: ["ignore", "pipe", "pipe"],
        env: { ...process.env, WORKTRUNK_DIRECTIVE_CD_FILE: directiveFile },
      });
      p.on("error", () => resolveP(false));
      p.on("close", (code) => resolveP(code === 0));
    });
    if (!ran) return false;

    const content = (await readFile(directiveFile, "utf8")).trim();
    return content.length > 0;
  } catch {
    return false;
  } finally {
    await rm(tempRepo, { recursive: true, force: true });
    await rm(tempDir, { recursive: true, force: true });
  }
}

// ---------------------------------------------------------------------------
// Root observation
// ---------------------------------------------------------------------------

export interface ObservedRoot {
  observed: string;
  note?: string;
}

/**
 * Observe which worktree the given cwd belongs to.
 * Standard root = `<projectroot>/.worktree/`; a foreign root yields a
 * one-line note instead of failing.
 *
 * Note: `rev-parse --show-toplevel` inside a linked worktree returns the
 * worktree's own root, not the main checkout — so the porcelain list is
 * run from `cwd` itself and the project root is taken from the first
 * block (git always lists the main checkout first).
 */
export async function observeWorktreeRoot(cwd: string): Promise<ObservedRoot> {
  const cwdAbs = join(cwd);
  const stdout = await git(["worktree", "list", "--porcelain"], { cwd: cwdAbs });

  const blocks: string[][] = [];
  let current: string[] = [];
  for (const line of stdout.split("\n")) {
    if (line.startsWith("worktree ")) {
      if (current.length) blocks.push(current);
      current = [line];
    } else if (line && current.length) {
      current.push(line);
    }
  }
  if (current.length) blocks.push(current);

  // git lists the main checkout first — that is the project root
  const root = blocks[0]?.[0].slice("worktree ".length) ?? cwdAbs;

  let observed: string | undefined;
  for (const block of blocks) {
    const path = block[0].slice("worktree ".length);
    if (cwdAbs === path) {
      observed = path;
      break;
    }
  }
  if (observed === undefined) {
    for (const block of blocks) {
      const path = block[0].slice("worktree ".length);
      if (cwdAbs.startsWith(path + "/")) {
        observed = path;
        break;
      }
    }
  }
  if (observed === undefined) {
    observed = cwdAbs;
  }

  const standardRoot = join(root, ".worktree");
  if (observed === standardRoot || observed.startsWith(standardRoot + "/")) {
    return { observed };
  }
  return {
    observed,
    note: `observed root differs: worktree landed at ${observed} (standard root is ${standardRoot})`,
  };
}

// ---------------------------------------------------------------------------
// Directive validation (R1 safety)
// ---------------------------------------------------------------------------

export type DirectiveValidation = { ok: true } | { ok: false; rejected: string };

/**
 * A requested directory must belong to the original repository — it must
 * resolve to a git toplevel whose common git dir is the same repository as
 * `cwd`'s (main checkout or any of its linked worktrees). A non-git
 * directory (e.g. `<repo>/.worktree/` before its worktree is created) is
 * inside the repo and accepted. Rejected directives never trigger a
 * fallback move; the caller reports `rejected` as-is.
 *
 * Synchronous by contract (Task 7's directive-follow uses it on the move
 * path); the git lookups are execFileSync (no shell).
 */
export function validateDirective(cwd: string, directive: string): DirectiveValidation {
  if (!directive) return { ok: false, rejected: directive };
  const exec = (dir: string, args: string[]): string => {
    try {
      return execFileSync("git", ["-C", dir, ...args], { encoding: "utf8" }).trim();
    } catch {
      return "";
    }
  };
  const cwdRoot = exec(cwd, ["rev-parse", "--show-toplevel"]);
  if (!cwdRoot) return { ok: false, rejected: directive };
  const dirRoot = exec(directive, ["rev-parse", "--show-toplevel"]);
  // directive is not inside a git toplevel → it must be a subdirectory of
  // cwd's repo to be acceptable.
  if (!dirRoot) {
    const resolved = isAbsolute(directive) ? directive : resolvePath(cwd, directive);
    const prefix = cwdRoot.endsWith("/") ? cwdRoot : cwdRoot + "/";
    return resolved === cwdRoot || resolved.startsWith(prefix)
      ? { ok: true }
      : { ok: false, rejected: directive };
  }
  const commonOf = (dir: string): string => {
    const root = exec(dir, ["rev-parse", "--show-toplevel"]);
    if (!root) return "";
    const raw = exec(root, ["rev-parse", "--git-common-dir"]);
    return isAbsolute(raw) ? raw : resolvePath(root, raw);
  };
  const cwdCommon = commonOf(cwd);
  const dirCommon = commonOf(directive);
  if (!cwdCommon || !dirCommon) return { ok: false, rejected: directive };
  if (resolvePath(cwdCommon) !== resolvePath(dirCommon)) return { ok: false, rejected: directive };
  return { ok: true };
}

// ---------------------------------------------------------------------------
// Recovery (R1): recorded worktrees + no-replay destination selection
// ---------------------------------------------------------------------------

/**
 * Recorded worktrees are the recovery eligibility set (R1 safety): recovery
 * only ever moves to a worktree the session previously occupied. Session-
 * scoped lifetime: the extension records at `session_start` and on every
 * move, and calls `clearWorktreeRecord()` on `session_shutdown`.
 *
 * In-memory only; the directive-file mutex does not guard this map.
 */
const worktrees = new Map<string, { gitCommonDir: string; recordedAt: number }>();

const gitSync = (dir: string, args: string[]): string => {
  try {
    return execFileSync("git", ["-C", dir, ...args], {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "ignore"],
    }).trim();
  } catch {
    return "";
  }
};

const gitCommonDirOf = (dir: string): string => {
  const root = gitSync(dir, ["rev-parse", "--show-toplevel"]);
  if (!root) return "";
  const raw = gitSync(root, ["rev-parse", "--git-common-dir"]);
  return raw ? (isAbsolute(raw) ? raw : resolvePath(root, raw)) : "";
};

const isExistingGitDir = (dir: string): boolean => {
  if (!gitSync(dir, ["rev-parse", "--git-dir"])) return false;
  return gitCommonDirOf(dir) !== "";
};

/**
 * Record the repository identity for `cwd` so `recover()` may move there
 * later. Idempotent — re-recording refreshes `recordedAt`.
 */
export function recordWorktrees(cwd: string): void {
  const root = gitSync(cwd, ["rev-parse", "--show-toplevel"]);
  if (!root) return;
  const gitCommonDir = gitSync(root, ["rev-parse", "--git-common-dir"]);
  if (!gitCommonDir) return;
  worktrees.set(root, {
    gitCommonDir: isAbsolute(gitCommonDir) ? gitCommonDir : resolvePath(root, gitCommonDir),
    recordedAt: Date.now(),
  });
}

/** Drop every recorded worktree (the extension calls this on session_shutdown). */
export function clearWorktreeRecord(): void {
  worktrees.clear();
}

export interface RecoverResult {
  /** A surviving worktree of the same repository, or undefined. */
  movedTo?: string;
  /** Always false — no-replay: a pending command requested after the
   * directory died is never run. */
  replayed: boolean;
}

/**
 * The repo's default branch: `origin/HEAD`'s branch when set, falling back
 * to `main` then `master`.
 */
function defaultBranchOf(path: string): string {
  const symbolic = gitSync(path, ["symbolic-ref", "--short", "refs/remotes/origin/HEAD"]);
  if (symbolic.startsWith("refs/remotes/origin/")) {
    const name = symbolic.slice("refs/remotes/origin/".length);
    if (name) return name;
  }
  for (const candidate of ["main", "master"]) {
    if (gitSync(path, ["rev-parse", "--verify", "--quiet", `refs/heads/${candidate}`])) {
      return candidate;
    }
  }
  return "main";
}

/**
 * Recovery (R1): when the session's cwd is gone, move it to a surviving
 * worktree of the SAME repository that was previously recorded. The
 * survivor is the worktree whose branch is the repository's default
 * (`git symbolic-ref --short refs/remotes/origin/HEAD`, fallback `main`
 * then `master`); when several same-repo worktrees exist and none is on
 * the default branch, the most recently recorded one is used.
 *
 * `pendingArgs` is intentionally never run (no-replay). `cwd` must have
 * already been recorded (session_start / every move); when its directory
 * still exists, `recordWorktrees(cwd)` is called defensively first so the
 * identity is available for the comparison.
 *
 * Creating the linked "Session recovered" session is the extension's job —
 * this function only returns `movedTo`.
 */
export async function recover(
  cwd: string,
  pendingArgs: string[] | null,
): Promise<RecoverResult> {
  void pendingArgs; // no-replay: requested after the directory died, never run

  let cwdAlive = true;
  try {
    await stat(cwd);
  } catch {
    cwdAlive = false;
  }
  // Identity of the original cwd: the recorded entry for it when available
  // (recorded at session_start / every move), else derived from the path
  // before it died.
  let cwdIdentity = worktrees.get(cwd)?.gitCommonDir ?? "";
  if (!cwdIdentity) {
    for (const [path, info] of worktrees) {
      if (cwd.startsWith(path + "/")) {
        cwdIdentity = info.gitCommonDir;
        break;
      }
    }
  }
  if (!cwdIdentity) cwdIdentity = gitCommonDirOf(cwd);

  if (cwdAlive) {
    // The directory is still usable — nothing to recover. Defensive
    // re-record (sync) so the identity stays available for a later
    // recover once the directory dies.
    recordWorktrees(cwd);
    return { movedTo: undefined, replayed: false };
  }

  const candidates = [...worktrees.entries()].filter(([, info]) => {
    if (!cwdIdentity) return true;
    return info.gitCommonDir === cwdIdentity;
  });
  if (candidates.length === 0) return { movedTo: undefined, replayed: false };

  const surviving = (await Promise.all(
    candidates.map(async ([path]) => {
      let alive = false;
      try {
        await stat(path);
        alive = isExistingGitDir(path);
      } catch {
        alive = false;
      }
      return alive ? path : undefined;
    }),
  )).filter((p): p is string => p !== undefined);
  if (surviving.length === 0) return { movedTo: undefined, replayed: false };

  let target: string | undefined;
  // Resolve the default branch once, against a surviving checkout.
  const defaultBranch = defaultBranchOf(surviving[0]);
  for (const path of surviving) {
    const branch = gitSync(path, ["rev-parse", "--abbrev-ref", "HEAD"]);
    if (branch && branch === defaultBranch) {
      target = path;
      break;
    }
  }
  if (target === undefined) {
    // none on the default branch — most recently recorded survivor
    target = surviving
      .sort((a, b) => (worktrees.get(b)?.recordedAt ?? 0) - (worktrees.get(a)?.recordedAt ?? 0))[0];
  }
  return { movedTo: target, replayed: false };
}

// ---------------------------------------------------------------------------
// Worktree context (R8 allow-list invariance)
// ---------------------------------------------------------------------------

/**
 * True when `cwd` is a linked worktree (not the primary checkout). The
 * primary checkout's `--git-dir` equals its `--git-common-dir` (both `.git`);
 * a linked worktree's git-dir lives under the common dir's `worktrees/`
 * subdir, so the two differ.
 */
export async function isWorktreeContext(cwd: string): Promise<boolean> {
  const gitDir = await git(["rev-parse", "--git-dir"], { cwd }).catch(() => "");
  const commonDir = await git(["rev-parse", "--git-common-dir"], { cwd }).catch(() => "");
  return gitDir !== "" && gitDir !== commonDir;
}

// ---------------------------------------------------------------------------
// Approval passthrough (R10): detection, PTY store, and the retry flow.
// wt is spawned piped (non-interactive): an unapproved hook is silently
// skipped by the command (exit 0, no marker), and `wt config approvals add
// --yes` cannot confirm its "Allow and remember? [y/N]" prompt without a TTY.
// So detection probes the hook phase, and the store runs under a PTY with the
// answer fed to stdin. `onAsk` is injected (the extension wires ctx.ui.confirm).
// ---------------------------------------------------------------------------

/** wt flags the output of an unapproved project command with this marker. */
const APPROVAL_MARKER = "[approval]";

/** wt prints the unapproved commands after the marker or in a needs-approval block. */
export function approvalCommands(output: string): string[] {
  const lines = output.split("\n");
  const idx = lines.findIndex((l) => l.includes(APPROVAL_MARKER));
  if (idx >= 0) {
    const remainder = lines[idx].slice(lines[idx].indexOf(APPROVAL_MARKER) + APPROVAL_MARKER.length);
    if (remainder.trim()) return [remainder.trim()];
  }
  const commands: string[] = [];
  let inBlock = false;
  for (const line of lines) {
    if (/needs approval to execute/i.test(line)) {
      inBlock = true;
      continue;
    }
    if (!inBlock) continue;
    const m = line.match(/^\s*○\s*(\S+?):\s*(.*)$/);
    if (m) {
      const rest = m[2].trim();
      if (rest) commands.push(`${m[1]}: ${rest}`);
      continue;
    }
    if (/^\s{2,}\S/.test(line)) {
      const cmd = line.trim();
      if (cmd) commands.push(cmd);
    }
  }
  return commands;
}

export function isApprovalNeeded(output: string): boolean {
  return output.includes(APPROVAL_MARKER) || /needs approval|cannot prompt for approval/i.test(output);
}

/** wt commands that run project hooks, and the phases they trigger (R10 detection). */
const HOOK_PHASES: Record<string, string[]> = {
  switch: ["pre-switch", "post-switch"],
  merge: ["pre-merge", "post-merge"],
  start: ["pre-start", "post-start"],
  remove: ["pre-remove", "post-remove"],
  commit: ["pre-commit", "post-commit"],
};

/** The hook phases a `wt <command>` invocation triggers (empty if none). */
export function commandHookPhases(command: string): string[] {
  return HOOK_PHASES[command] ?? [];
}

/**
 * Detect a hook that the command silently skipped: a hook-triggering command
 * whose phase has an unapproved hook exits 0 with no marker, so probe each
 * triggered phase (`wt hook <phase>`, piped — it prints "needs approval to
 * execute N command" and exits 1 without running when unapproved).
 */
export async function probeUnapprovedHooks(
  cwd: string,
  command: string,
): Promise<{ needsApproval: boolean; output: string }> {
  const phases = commandHookPhases(command);
  if (phases.length === 0) return { needsApproval: false, output: "" };
  const parts: string[] = [];
  for (const phase of phases) {
    const res = await runDirectedWt(["hook", phase], { cwd });
    if (isApprovalNeeded(res.stdout + res.stderr)) parts.push(res.stdout + res.stderr);
  }
  return { needsApproval: parts.length > 0, output: parts.join("\n") };
}

/**
 * Store project approvals non-interactively. Runs `wt config approvals add
 * --yes` under a PTY (`script`) with the answer fed to stdin — wt's
 * "Allow and remember? [y/N]" prompt cannot be confirmed from a pipe.
 * Returns whether wt reports the commands were saved.
 */
export function storeApprovalsPty(
  cwd: string,
  yes: boolean,
): Promise<{ ok: boolean; saved: boolean; message: string }> {
  return new Promise((resolveP) => {
    let child;
    try {
      child = spawn("script", ["-qec", "wt config approvals add --yes", "/dev/null"], {
        cwd,
        stdio: ["pipe", "pipe", "pipe"],
      });
    } catch {
      resolveP({ ok: false, saved: false, message: "could not spawn a PTY for the approval store" });
      return;
    }
    let out = "";
    child.stdout?.on("data", (d: Buffer) => (out += d));
    child.stderr?.on("data", (d: Buffer) => (out += d));
    child.on("error", () => resolveP({ ok: false, saved: false, message: "script (PTY) not available" }));
    child.on("close", (code) => {
      resolveP({
        ok: code === 0,
        saved: /approved & saved|saved to config/i.test(out),
        message: out.trim(),
      });
    });
    child.stdin?.write(yes ? "y\n" : "n\n");
  });
}

export interface ApprovalOutcome {
  approved: boolean;
  /** The commands saved to the approvals file (empty when not approved). */
  savedCommands: string[];
  code?: number;
  message: string;
}

/**
 * The approval flow: parse the pending commands, ask the user (via `onAsk`),
 * and on approval store them (PTY) and retry the command once. On decline the
 * approvals file is untouched. Testable without Pi — `onAsk` is injected.
 */
export async function handleWtApproval(
  cwd: string,
  args: string[],
  output: string,
  onAsk: () => Promise<boolean>,
): Promise<ApprovalOutcome> {
  const commands = approvalCommands(output);
  if (commands.length === 0) {
    return {
      approved: false,
      savedCommands: [],
      message:
        "Could not determine which project commands need approval. " +
        "Approve in a terminal with `wt config approvals add`, then retry.",
    };
  }
  const listing = commands.map((c) => `  ${c}`).join("\n");
  const confirmed = await onAsk();
  if (!confirmed) {
    return {
      approved: false,
      savedCommands: [],
      message: `Approval declined; Worktrunk did not run these project commands:\n${listing}`,
    };
  }
  const stored = await storeApprovalsPty(cwd, true);
  if (!stored.saved) {
    return {
      approved: false,
      savedCommands: [],
      message:
        `Could not store approvals${stored.message ? `: ${stored.message}` : ""}. ` +
        "Approve in a terminal with `wt config approvals add`, then retry.",
    };
  }
  const retry = await runDirectedWt(args, { cwd });
  if (isApprovalNeeded(retry.stdout + retry.stderr)) {
    return {
      approved: false,
      savedCommands: [],
      code: retry.code,
      message: `Command still needs approval after retry. Pending:\n${listing}`,
    };
  }
  return {
    approved: true,
    savedCommands: commands,
    code: retry.code,
    message: `Approved project commands:\n${listing}\n\n${retry.stdout}`,
  };
}
