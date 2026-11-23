import { test, expect, beforeEach, afterEach } from "bun:test";
import { mkdtemp, rm, writeFile, mkdir, stat, readFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createCheckpoint, git, resetSessionState, restoreCheckpoint, pruneOtherSessions, prune, seedDedup, listCheckpoints } from "../src/core/scratch-core.ts";
import { REF_BASE } from "../src/core/constants.ts";

const pathExists = async (p: string) => {
  try { await stat(p); return true; } catch { return false; }
};

/** Seed N checkpoint refs for a session (fast path for prune tests). */
async function seedRefs(root: string, count: number): Promise<void> {
  const head = (await git(["rev-parse", "HEAD"], { cwd: root })).trim();
  const tree = (await git(["write-tree"], { cwd: root })).trim();
  const msgFile = join(root, ".seed-msg");
  const refId = (t: number) => `s1-t${t}-${Date.now() + t}`;
  for (let t = 1; t <= count; t++) {
    const created = new Date(Date.now() + t * 1000).toISOString(); // increasing → oldest-first
    await writeFile(msgFile, [
      `branch main`, `head ${head}`, `index-tree ${tree}`, `worktree-tree ${tree}`,
      `session s1`, `turn ${t}`, `label seed ${t}`, `created ${created}`,
      `preexisting-untracked []`, `largeFiles []`, `largeDirs []`,
    ].join("\n"));
    const sha = (await git(["commit-tree", tree, "-p", head, "-F", msgFile], { cwd: root })).trim();
    await git(["update-ref", `${REF_BASE}/${refId(t)}`, sha], { cwd: root });
  }
  await rm(msgFile);
}

let repo: string;
beforeEach(async () => {
  repo = await mkdtemp(join(tmpdir(), "pi-wt-"));
  await git(["init", "-b", "main"], { cwd: repo });
  await git(["config", "core.hooksPath", "/dev/null"], { cwd: repo });
  await git(["config", "user.email", "t@t"], { cwd: repo });
  await git(["config", "user.name", "t"], { cwd: repo });
  await writeFile(join(repo, "a.txt"), "one\n");
  await git(["add", "a.txt"], { cwd: repo });
  await git(["commit", "-m", "chore: init"], { cwd: repo });
});
afterEach(() => rm(repo, { recursive: true, force: true }));

test("mutating turn creates exactly one ref with full metadata", async () => {
  resetSessionState();
  await writeFile(join(repo, "b.txt"), "two\n");
  const r1 = await createCheckpoint({ root: repo, sessionId: "s1", turn: 1, label: "edit b" });
  expect(r1.status).toBe("created");
  const refs = (await git(["for-each-ref", "refs/skillgrid/"], { cwd: repo })).split("\n").filter(Boolean);
  expect(refs).toHaveLength(1);
  // ref-qualified with explicit cwd: the id is not an object, and git
  // resolves the ref against the repo of the child's cwd (the temp repo)
  const msg = await git(["log", "-1", "--format=%B", `${REF_BASE}/${r1.id}`], { cwd: repo });
  for (const field of ["branch main", "head ", "index-tree ", "worktree-tree ", "session s1", "turn 1", "label edit b", "preexisting-untracked ["]) {
    expect(msg).toContain(field);
  }
});

test("capture leaves the real index byte-identical", async () => {
  const before = await git(["write-tree"], { cwd: repo });
  await createCheckpoint({ root: repo, sessionId: "s1", turn: 1, label: "x" });
  const after = await git(["write-tree"], { cwd: repo });
  expect(before).toBe(after);
});

test("read-only turn is skipped (dedup on worktree tree SHA)", async () => {
  resetSessionState();
  const first = await createCheckpoint({ root: repo, sessionId: "s1", turn: 1, label: "a" });
  expect(first.status).toBe("created");
  const second = await createCheckpoint({ root: repo, sessionId: "s1", turn: 2, label: "b" });
  expect(second).toEqual({ status: "skipped", reason: "unchanged" });
});

test("resume re-seeds the dedup baseline and skips an unchanged first turn", async () => {
  resetSessionState();
  await writeFile(join(repo, "b.txt"), "two\n");
  const first = await createCheckpoint({ root: repo, sessionId: "s1", turn: 1, label: "first" });
  expect(first.status).toBe("created");
  const cps = await listCheckpoints(repo, "s1");
  // Simulate a session restart: in-memory state is cleared, then session_start
  // re-seeds the dedup baseline from the newest ref (R3-F007).
  resetSessionState();
  seedDedup("s1", cps[cps.length - 1].worktreeTreeSha);
  const resumed = await createCheckpoint({ root: repo, sessionId: "s1", turn: 2, label: "unchanged" });
  expect(resumed).toEqual({ status: "skipped", reason: "unchanged" });
  const refs = (await git(["for-each-ref", "refs/skillgrid/"], { cwd: repo })).split("\n").filter(Boolean);
  expect(refs).toHaveLength(1); // no duplicate ref for the unchanged tree
});

test("unborn HEAD captures with zero-SHA head", async () => {
  resetSessionState();
  const bare = await mkdtemp(join(tmpdir(), "pi-wt-bare-"));
  await git(["init", "-b", "main"], { cwd: bare });
  await git(["config", "core.hooksPath", "/dev/null"], { cwd: bare });
  await writeFile(join(bare, "x.txt"), "x");
  const r = await createCheckpoint({ root: bare, sessionId: "s1", turn: 1, label: "unborn" });
  expect(r.status).toBe("created");
  expect(await git(["log", "-1", "--format=%B", `${REF_BASE}/${r.id}`], { cwd: bare })).toContain("head 0000000000000000000000000000000000000000");
  await rm(bare, { recursive: true, force: true });
});

test("an ignored-but-present file is absent from the captured tree", async () => {
  resetSessionState();
  await writeFile(join(repo, ".gitignore"), "node_modules/\n");
  await mkdir(join(repo, "node_modules"), { recursive: true });
  await writeFile(join(repo, "node_modules/x.js"), "x");
  const r = await createCheckpoint({ root: repo, sessionId: "s1", turn: 1, label: "ig" });
  expect(r.status).toBe("created");
  const tree = await git(["ls-tree", "-r", "--name-only", r.worktreeTreeSha], { cwd: repo });
  expect(tree).not.toContain("node_modules");
});

test("restore on matching branch: deleted tracked file returns, post-checkpoint untracked removed, pre-existing survives", async () => {
  resetSessionState(); // module-level lastTree persists across tests — clear it
  await writeFile(join(repo, "pre.txt"), "pre-existing\n");           // untracked AT capture
  const r = await createCheckpoint({ root: repo, sessionId: "s1", turn: 1, label: "base" });
  expect(r.status).toBe("created");
  await writeFile(join(repo, "a.txt"), "changed\n");
  await git(["add", "a.txt"], { cwd: repo });
  await git(["commit", "-m", "chore: change a"], { cwd: repo });
  await writeFile(join(repo, "after.txt"), "born after\n");           // untracked AFTER capture
  const res = await restoreCheckpoint(repo, r.id);
  expect(res.status).toBe("restored");
  expect(await readFile(join(repo, "a.txt"), "utf8")).toBe("one\n");
  expect(await pathExists(join(repo, "after.txt"))).toBe(false);
  expect(await pathExists(join(repo, "pre.txt"))).toBe(true);
});

test("restore takes a before-restore checkpoint first", async () => {
  resetSessionState();
  const base = await createCheckpoint({ root: repo, sessionId: "s1", turn: 1, label: "base" });
  await writeFile(join(repo, "a.txt"), "dirty\n");
  await restoreCheckpoint(repo, base.id);
  const refs = (await git(["for-each-ref", "refs/skillgrid/"], { cwd: repo })).split("\n").filter(Boolean);
  expect(refs).toHaveLength(2); // base + before-restore
});

test("refused: branch mismatch, named", async () => {
  resetSessionState();
  const base = (await createCheckpoint({ root: repo, sessionId: "s1", turn: 1, label: "b" })).id as string;
  await git(["checkout", "-b", "agent-other"], { cwd: repo });
  const res = await restoreCheckpoint(repo, base);
  expect(res.status).toBe("refused");
  expect(res.reason).toBe("branch");
  expect(res.detail).toContain("main");
  expect(res.detail).toContain("agent-other");
});

test("refused: no such checkpoint", async () => {
  const res = await restoreCheckpoint(repo, "s1-t9-nope");
  expect(res).toMatchObject({ status: "refused", reason: "missing" });
});

test("detached HEAD: capture ok, restore refused with DETACHED named", async () => {
  resetSessionState();
  await git(["checkout", "--detach"], { cwd: repo });
  const r = await createCheckpoint({ root: repo, sessionId: "s1", turn: 1, label: "d" });
  expect(r.status).toBe("created");
  const res = await restoreCheckpoint(repo, r.id);
  expect(res).toMatchObject({ status: "refused", reason: "branch" });
  expect(res.detail).toContain("DETACHED");
});

test("exactly 50 captures: no prune, all 50 kept", async () => {
  // Drive prune directly with seeded refs (50 sequential createCheckpoint calls
  // exceed the 5s test timeout on this machine).
  resetSessionState();
  await seedRefs(repo, 50);
  const n = await prune(repo, "s1");
  expect(n).toBe(0);
  const refs = (await git(["for-each-ref", "refs/skillgrid/"], { cwd: repo })).split("\n").filter(Boolean);
  expect(refs).toHaveLength(50);
});

test("51st capture prunes the oldest, keeping 50", async () => {
  resetSessionState();
  await seedRefs(repo, 51);
  const n = await prune(repo, "s1");
  expect(n).toBe(1);
  const refs = (await git(["for-each-ref", "refs/skillgrid/"], { cwd: repo })).split("\n").filter(Boolean);
  expect(refs).toHaveLength(50);
});

test("pruneOtherSessions removes other sessions' refs, keeps the current one", async () => {
  resetSessionState();
  const old = await createCheckpoint({ root: repo, sessionId: "old", turn: 1, label: "o" });
  await writeFile(join(repo, "n.txt"), "n");
  await createCheckpoint({ root: repo, sessionId: "cur", turn: 1, label: "c" });
  // Backdate "old"'s `created` to 2h ago so it crosses the liveness threshold
  // (the injectable `now` would also work, but backdating makes the ref itself
  // deterministically old — Stage 9 M-13)
  expect(old.status).toBe("created");
  const oldSha = (await git(["rev-parse", `${REF_BASE}/${old.id}`], { cwd: repo })).trim();
  const oldCreated = new Date(Date.now() - 2 * 3600 * 1000).toISOString();
  const oldMsg = await git(["log", "-1", "--format=%B", oldSha], { cwd: repo });
  const msgFile = join(repo, ".meta-backdate");
  await writeFile(msgFile, oldMsg.replace(/^created .*/m, `created ${oldCreated}`));
  const tree = (await git(["rev-parse", `${oldSha}^{tree}`], { cwd: repo })).trim();
  const parent = (await git(["rev-parse", `${oldSha}^`], { cwd: repo })).trim();
  const reborn = (await git(["commit-tree", tree, "-p", parent, "-F", msgFile], {
    cwd: repo,
    env: { ...process.env, GIT_AUTHOR_DATE: oldCreated, GIT_COMMITTER_DATE: oldCreated },
  })).trim();
  await git(["update-ref", `${REF_BASE}/${old.id}`, reborn], { cwd: repo });
  await rm(msgFile);
  const n = await pruneOtherSessions(repo, "cur");
  expect(n).toBe(1);
  const refs = (await git(["for-each-ref", "refs/skillgrid/"], { cwd: repo })).split("\n").filter(Boolean);
  expect(refs).toHaveLength(1);
  expect(refs[0]).toContain("cur");
});

test("pruneOtherSessions keeps a sibling session's fresh ref (liveness threshold)", async () => {
  resetSessionState();
  // fan-out scenario: session "sib" is concurrently running in a sibling worktree —
  // its ref is FRESH (created < 1h ago); session "dead" is long gone (> 1h old).
  // Both captures happen now, so backdate "dead"'s `created` metadata to 2h ago.
  await createCheckpoint({ root: repo, sessionId: "sib", turn: 1, label: "s" });
  await writeFile(join(repo, "d.txt"), "d");
  const dead = await createCheckpoint({ root: repo, sessionId: "dead", turn: 1, label: "d" });
  expect(dead.status).toBe("created");
  const deadSha = (await git(["rev-parse", `${REF_BASE}/${dead.id}`], { cwd: repo })).trim();
  const oldCreated = new Date(Date.now() - 2 * 3600 * 1000).toISOString();
  const msg = await git(["log", "-1", "--format=%B", deadSha], { cwd: repo });
  const backdated = msg.replace(/^created .*/m, `created ${oldCreated}`);
  const msgFile = join(repo, ".meta-backdate");
  await writeFile(msgFile, backdated);
  const parent = (await git(["rev-parse", `${deadSha}^`], { cwd: repo })).trim();
  const tree = (await git(["rev-parse", `${deadSha}^{tree}`], { cwd: repo })).trim();
  const reborn = (await git(["commit-tree", tree, "-p", parent, "-F", msgFile], {
    cwd: repo,
    env: { ...process.env, GIT_AUTHOR_DATE: oldCreated, GIT_COMMITTER_DATE: oldCreated },
  })).trim();
  await git(["update-ref", `${REF_BASE}/${dead.id}`, reborn], { cwd: repo });
  await rm(msgFile);
  // "dead" is now 2h old, "sib" is fresh — only "dead" crosses the threshold
  const n = await pruneOtherSessions(repo, "cur");
  expect(n).toBe(1); // only "dead" pruned
  const refs = (await git(["for-each-ref", "refs/skillgrid/"], { cwd: repo })).split("\n").filter(Boolean);
  expect(refs.some(r => r.includes("sib"))).toBe(true);
  expect(refs.some(r => r.includes("dead"))).toBe(false);
});
