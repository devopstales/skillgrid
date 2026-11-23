import { spawn } from "node:child_process";
import { mkdtemp, rm, rm as fsRm } from "node:fs/promises";
import { statSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import {
  REF_BASE,
  IGNORED_DIRS,
  MAX_UNTRACKED_FILE_SIZE,
  MAX_UNTRACKED_DIR_FILES,
  MAX_PER_SESSION,
  isSafeRefId,
} from "./constants.ts";

/** 1h — a live session's restart gap never crosses this (Goldfish Stage 7 R3-F003). */
export const PRUNE_MIN_AGE_MS = 3600 * 1000;

export interface Checkpoint {
  id: string;
  sessionId: string;
  branch: string;
  headSha: string;
  indexTreeSha: string;
  worktreeTreeSha: string;
  turn: number;
  label: string;
  created: string;
  preexistingUntracked: string[];
  skippedLargeFiles: string[];
  skippedLargeDirs: string[];
}

export type CheckpointResult =
  | { status: "created"; id: string; worktreeTreeSha: string }
  | { status: "skipped"; reason: "unchanged" };

const ZEROS = "0".repeat(40);

/** sessionId → last captured worktree tree SHA (dedup). */
const lastTree = new Map<string, string>();

export function resetSessionState(): void {
  lastTree.clear();
}

/** Seed the dedup baseline for a session on resume (blueprint R3-F007). */
export function seedDedup(sessionId: string, worktreeTreeSha: string): void {
  lastTree.set(sessionId, worktreeTreeSha);
}

/**
 * Run git with an explicit argv — never through a shell.
 * Resolves with stdout on exit 0; rejects with an Error carrying stderr.
 */
export function git(args: string[], opts?: { cwd?: string; env?: NodeJS.ProcessEnv }): Promise<string> {
  const { promise, resolve, reject } = Promise.withResolvers<string>();
  const child = spawn("git", args, {
    cwd: opts?.cwd,
    env: opts?.env ?? process.env,
    stdio: ["pipe", "pipe", "pipe"],
  });
  let stdout = "";
  let stderr = "";
  child.stdout.on("data", (d: Buffer) => {
    stdout += d.toString();
  });
  child.stderr.on("data", (d: Buffer) => {
    stderr += d.toString();
  });
  child.on("error", (err) => reject(err));
  child.on("close", (code) => {
    if (code === 0) resolve(stdout);
    else reject(new Error(stderr.trim() || `git ${args.join(" ")} exited ${code}`));
  });
  return promise;
}

/** Absolute toplevel of the repo containing cwd. */
export function repoRoot(cwd: string): Promise<string> {
  return git(["-C", cwd, "rev-parse", "--show-toplevel"]).then((s) => s.trim());
}

/**
 * Decode one porcelain-v1 path field. We pass `core.quotePath=false`, so
 * non-ASCII bytes arrive raw (no C-quoting octal escapes that break JSON);
 * a path with special ASCII (newline/tab/quote) is still C-quoted and is
 * JSON-compatible. Rename/copy lines (`XY old -> new`) keep the target.
 */
function decodePath(xy: string, rest: string): string {
  if ((xy.startsWith("R") || xy.startsWith("C")) && rest.includes(" -> ")) {
    return rest.slice(rest.lastIndexOf(" -> ") + 4);
  }
  if (!rest.startsWith('"')) return rest;
  try {
    return JSON.parse(rest);
  } catch {
    // Malformed C-quote — fall back to the raw field minus the quotes.
    return rest.length >= 2 ? rest.slice(1, -1) : rest;
  }
}

/** Parse a metadata array field; malformed/absent → [] (non-throwing). */
function parseArr(v: string | undefined): string[] {
  if (!v) return [];
  try {
    const r: unknown = JSON.parse(v);
    return Array.isArray(r) ? (r as string[]) : [];
  } catch {
    return [];
  }
}

interface ParseResult {
  paths: string[];
  preexistingUntracked: string[];
  skippedLargeFiles: string[];
  skippedLargeDirs: string[];
}

/**
 * Parse `git status --porcelain=v1 -uall` output and apply the capture filters:
 * IGNORED_DIRS, file-size cap, per-directory untracked-file cap.
 */
async function filterCandidates(statusOut: string, root: string): Promise<ParseResult> {
  const lines = statusOut.split("\n").filter(Boolean);
  const preexistingUntracked: string[] = [];
  const skippedLargeFiles: string[] = [];
  const skippedLargeDirs = new Set<string>();
  const perDirCount = new Map<string, number>();

  // Which directory (relative) each untracked file would land in.
  const countsFor = (p: string) => {
    const d = dirname(p) === "." ? "" : dirname(p);
    perDirCount.set(d, (perDirCount.get(d) ?? 0) + 1);
    return d;
  };

  // First pass: counts + ignored dirs.
  for (const line of lines) {
    const xy = line.slice(0, 2);
    const rest = line.slice(3);
    if (!rest) continue;
    const p = decodePath(xy, rest);
    if (xy === "??") {
      // IGNORED_DIRS filter
      const comps = p.split("/");
      for (const c of comps) {
        if (IGNORED_DIRS.has(c)) {
          skippedLargeDirs.add(c);
          break;
        }
      }
      // Per-dir counts
      countsFor(p);
    }
  }

  const overflowingDirs = new Set<string>();
  for (const [d, n] of perDirCount) {
    if (n > MAX_UNTRACKED_DIR_FILES) overflowingDirs.add(d);
  }

  const paths: string[] = [];
  for (const line of lines) {
    const xy = line.slice(0, 2);
    const rest = line.slice(3);
    if (!rest) continue;
    const p = decodePath(xy, rest);

    if (xy === "??") {
      preexistingUntracked.push(p);
    }

    // Ignored-dir filter
    const comps = p.split("/");
    if (comps.some((c) => IGNORED_DIRS.has(c))) continue;

    // Overflowing-dir filter
    const d = dirname(p) === "." ? "" : dirname(p);
    if (overflowingDirs.has(d)) {
      skippedLargeDirs.add(d || ".");
      continue;
    }

    paths.push(p);
  }

  // Size-check untracked candidates in a single sync pass (per-file stat is
  // the dominant cost at 5k+ files; sync statSync avoids the per-file
  // promise microtask overhead of the async API — capture is not called on
  // a latency-critical hot path, one ~100ms block is the budget).
  const untracked = new Set(preexistingUntracked);
  const sized: string[] = [];
  for (const p of paths) {
    if (!untracked.has(p)) {
      sized.push(p);
      continue;
    }
    try {
      if (statSync(join(root, p)).size > MAX_UNTRACKED_FILE_SIZE) {
        skippedLargeFiles.push(p);
        continue;
      }
    } catch {
      // Vanished between listing and stat — skip.
      continue;
    }
    sized.push(p);
  }
  return {
    paths: sized,
    preexistingUntracked,
    skippedLargeFiles,
    skippedLargeDirs: [...skippedLargeDirs],
  };
}

export async function createCheckpoint(o: {
  root: string;
  sessionId: string;
  turn: number;
  label: string;
}): Promise<CheckpointResult> {
  const root = o.root;

  // 1. HEAD SHA (unborn HEAD → zeros)
  let headSha = ZEROS;
  try {
    headSha = (await git(["rev-parse", "HEAD"], { cwd: root })).trim();
  } catch {
    headSha = ZEROS;
  }

  // 2. Branch name. `symbolic-ref` resolves an unborn HEAD to the branch that
  //    will be created, so a pre-commit checkpoint records a restorable branch;
  //    a detached HEAD has no symbolic ref → "DETACHED" (never a safe target).
  let branch: string;
  try {
    branch = (await git(["symbolic-ref", "--short", "HEAD"], { cwd: root })).trim();
    if (!branch) branch = "DETACHED";
  } catch {
    branch = "DETACHED";
  }

  // 3. Real-index tree (read-only)
  const indexTreeSha = (await git(["write-tree"], { cwd: root })).trim();

  // 4. Temp index
  const tmpDir = await mkdtemp(join(tmpdir(), "pi-wt-idx-"));
  const tmpIndex = join(tmpDir, "index");
  const idxEnv: NodeJS.ProcessEnv = { ...process.env, GIT_INDEX_FILE: tmpIndex };
  try {
    // 5. Seed from HEAD
    if (headSha !== ZEROS) {
      await git(["read-tree", headSha], { cwd: root, env: idxEnv });
    }

    // 6. Candidates from the real index (quotePath=false: non-ASCII bytes stay
    //    raw so decodePath never meets git's C-quoting octal escapes).
    const statusOut = await git(
      ["-c", "core.quotePath=false", "status", "--porcelain=v1", "-uall"],
      { cwd: root },
    );
    const { paths, preexistingUntracked, skippedLargeFiles, skippedLargeDirs } =
      await filterCandidates(statusOut, root);

    // 8. Add in batches of 500 (args, no shell — 100/batch was spawn-bound at 5k files)
    for (let i = 0; i < paths.length; i += 500) {
      const batch = paths.slice(i, i + 500);
      await git(["add", "--all", "--", ...batch], { cwd: root, env: idxEnv });
    }

    // 9. Captured tree
    const worktreeTreeSha = (await git(["write-tree"], { cwd: root, env: idxEnv })).trim();

    // 10. Dedup
    if (lastTree.get(o.sessionId) === worktreeTreeSha) {
      return { status: "skipped", reason: "unchanged" };
    }

    // 11. Ref id
    const id = `${o.sessionId}-t${o.turn}-${Date.now()}`;
    if (!isSafeRefId(id)) throw new Error(`unsafe ref id: ${id}`);

    // 12. Metadata message
    const created = new Date().toISOString();
    const metadata = [
      `branch ${branch}`,
      `head ${headSha}`,
      `index-tree ${indexTreeSha}`,
      `worktree-tree ${worktreeTreeSha}`,
      `session ${o.sessionId}`,
      `turn ${o.turn}`,
      `label ${o.label}`,
      `created ${created}`,
      `preexisting-untracked ${JSON.stringify(preexistingUntracked)}`,
      `largeFiles ${JSON.stringify(skippedLargeFiles)}`,
      `largeDirs ${JSON.stringify(skippedLargeDirs)}`,
    ].join("\n");

    // 13. Commit the tree
    const commitSha = (await git(
      ["commit-tree", worktreeTreeSha, ...(headSha !== ZEROS ? ["-p", headSha] : []), "-m", metadata],
      { cwd: root, env: idxEnv },
    )).trim();

    // 14. Point the ref (real index env)
    await git(["update-ref", `${REF_BASE}/${id}`, commitSha], { cwd: root });

    // 15. Record for dedup
    lastTree.set(o.sessionId, worktreeTreeSha);

    // 16. Lifecycle: prune this session's oldest excess (51st capture → 50 refs).
    await prune(root, o.sessionId);

    return { status: "created", id, worktreeTreeSha };
  } finally {
    await rm(tmpDir, { recursive: true, force: true });
  }
}

/** Read a checkpoint's metadata block from its ref; null if absent/unsafe. */
export async function loadCheckpoint(root: string, id: string): Promise<Checkpoint | null> {
  if (!isSafeRefId(id)) return null;
  const meta = await git(["log", "-1", "--format=%B", `${REF_BASE}/${id}`], { cwd: root }).catch(() => null);
  if (meta == null) return null;
  const fields = new Map<string, string>();
  for (const line of meta.split("\n")) {
    const sp = line.indexOf(" ");
    if (sp > 0) fields.set(line.slice(0, sp), line.slice(sp + 1));
  }
  if (!fields.has("head") || !fields.has("worktree-tree")) return null;
  return {
    id,
    sessionId: fields.get("session") ?? "",
    branch: fields.get("branch") ?? "DETACHED",
    headSha: fields.get("head")!,
    indexTreeSha: fields.get("index-tree") ?? ZEROS,
    worktreeTreeSha: fields.get("worktree-tree")!,
    turn: Number(fields.get("turn") ?? "0"),
    label: fields.get("label") ?? "",
    created: fields.get("created") ?? new Date(0).toISOString(),
    preexistingUntracked: parseArr(fields.get("preexisting-untracked")),
    skippedLargeFiles: parseArr(fields.get("largeFiles")),
    skippedLargeDirs: parseArr(fields.get("largeDirs")),
  };
}

/** All checkpoints, optionally filtered by session, oldest first. */
export async function listCheckpoints(root: string, sessionId?: string): Promise<Checkpoint[]> {
  const out = await git(["for-each-ref", "--format=%(refname)", REF_BASE + "/"], { cwd: root });
  const outList: Checkpoint[] = [];
  for (const line of out.split("\n")) {
    const ref = line.trim();
    if (!ref) continue;
    const id = ref.slice(REF_BASE.length + 1);
    const cp = await loadCheckpoint(root, id);
    if (cp && (!sessionId || cp.sessionId === sessionId)) outList.push(cp);
  }
  return outList.sort((a, b) => a.created.localeCompare(b.created));
}

/**
 * Remove worktree files that must not survive a restore:
 * 1. Files present in the snapshot tree but absent in the restored HEAD
 *    (`git diff --diff-filter=AD HEAD <snapshot>`) — the blueprint's safe-clean set.
 * 2. Untracked files that appeared AFTER the snapshot (not in either tree).
 * Never deletes pre-existing untracked, skipped-large files, or anything under
 * a skipped/ignored dir.
 */
export async function safeClean(root: string, cp: Checkpoint): Promise<void> {
  const skipDirs = new Set(cp.skippedLargeDirs);
  const inSkippedDir = (rel: string) => rel.split("/").some((c) => skipDirs.has(c));
  const deleteSet = new Set<string>();

  // 1. Snapshot vs restored HEAD: added in the snapshot, absent in HEAD.
  const out = await git(["diff", "--name-only", "--diff-filter=AD", "HEAD", cp.worktreeTreeSha], {
    cwd: root,
  });
  for (const line of out.split("\n")) {
    const rel = line.trim();
    if (!rel || inSkippedDir(rel)) continue;
    if (cp.preexistingUntracked.includes(rel)) continue;
    if (cp.skippedLargeFiles.includes(rel)) continue;
    deleteSet.add(rel);
  }

  // 2. Untracked-now but not in the snapshot and not pre-existing.
  const others = await git(["ls-files", "--others", "--exclude-standard"], { cwd: root });
  for (const line of others.split("\n")) {
    const rel = line.trim();
    if (!rel) continue;
    if (cp.preexistingUntracked.includes(rel)) continue;
    if (cp.skippedLargeFiles.includes(rel)) continue;
    if (inSkippedDir(rel)) continue;
    const inSnapshot = deleteSet.has(rel); // it was in the snapshot tree (set 1)
    if (inSnapshot) continue;
    deleteSet.add(rel);
  }

  for (const rel of deleteSet) await fsRm(join(root, rel), { force: true });
}

export type RestoreResult =
  | { status: "restored"; beforeRestore: "created" | "skipped" | "failed" }
  | { status: "refused"; reason: "branch" | "missing"; detail: string };

export async function restoreCheckpoint(root: string, id: string): Promise<RestoreResult> {
  const cp = await loadCheckpoint(root, id);
  if (!cp) return { status: "refused", reason: "missing", detail: `no checkpoint ${id}` };

  // Raw name on purpose: a detached capture records "DETACHED" in metadata,
  // which never equals raw "HEAD" — detachment is never a safe restore target.
  let curBranch: string;
  try {
    curBranch = (await git(["rev-parse", "--abbrev-ref", "HEAD"], { cwd: root })).trim();
  } catch {
    curBranch = "HEAD";
  }
  if (curBranch !== cp.branch)
    return {
      status: "refused",
      reason: "branch",
      detail: `checkpoint on '${cp.branch}', current '${curBranch}'`,
    };

  // Before-restore snapshot: the safety net. Report its outcome — "created"
  // (a new ref), "skipped" (tree already equals the last capture), "failed"
  // (capture errored; the restore proceeds but with no safety-net ref).
  let beforeRestore: "created" | "skipped" | "failed";
  try {
    const pre = await createCheckpoint({ root, sessionId: cp.sessionId, turn: cp.turn + 1, label: `before-restore ${id}` });
    beforeRestore = pre.status;
  } catch {
    beforeRestore = "failed";
  }

  if (cp.headSha !== ZEROS) await git(["reset", "--hard", cp.headSha], { cwd: root });
  await git(["read-tree", "--reset", "-u", cp.worktreeTreeSha], { cwd: root });
  if (cp.headSha !== ZEROS) await safeClean(root, cp);
  await git(["read-tree", "--reset", cp.indexTreeSha], { cwd: root }); // real index
  return { status: "restored", beforeRestore };
}

/** Delete this session's oldest excess refs over MAX_PER_SESSION; returns count deleted. */
export async function prune(root: string, sessionId: string): Promise<number> {
  const cps = (await listCheckpoints(root, sessionId)).sort((a, b) => a.created.localeCompare(b.created));
  const excess = cps.length - MAX_PER_SESSION;
  for (let i = 0; i < excess; i++) await git(["update-ref", "-d", `${REF_BASE}/${cps[i].id}`], { cwd: root });
  return Math.max(0, excess);
}

/**
 * Liveness-aware cross-session prune (Goldfish Stage 7 M-1): delete refs of
 * OTHER sessions only when their `created` is older than PRUNE_MIN_AGE_MS.
 * `now` is injectable so the liveness threshold is testable without sleeping.
 */
export async function pruneOtherSessions(root: string, current: string, now: Date = new Date()): Promise<number> {
  let n = 0;
  for (const cp of await listCheckpoints(root)) {
    if (cp.sessionId === current) continue;
    if (now.getTime() - new Date(cp.created).getTime() <= PRUNE_MIN_AGE_MS) continue;
    await git(["update-ref", "-d", `${REF_BASE}/${cp.id}`], { cwd: root });
    n++;
  }
  return n;
}
