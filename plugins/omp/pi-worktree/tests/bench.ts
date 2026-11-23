// Latency bench: createCheckpoint across repo sizes (20 / 500 / 5,000 / 50,000 files).
// Build a temp repo with N tracked files, time createCheckpoint 5x, print a
// {files, median_ms} table; exit 1 if the 5,000-file median > 150ms (MH-6).
//
// Run: bun run bench   (plain script — no bun:test import)

import { mkdtemp, rm, writeFile } from "node:fs/promises";
import { writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createCheckpoint, git, resetSessionState } from "../src/core/scratch-core.ts";

const SIZES = [20, 500, 5000, 50000];
const RUNS = 5;

async function buildRepo(files: number): Promise<string> {
  const repo = await mkdtemp(join(tmpdir(), "pi-wt-bench-"));
  await git(["init", "-b", "main"], { cwd: repo });
  await git(["config", "core.hooksPath", "/dev/null"], { cwd: repo });
  await git(["config", "user.email", "bench@t"], { cwd: repo });
  await git(["config", "user.name", "bench"], { cwd: repo });
  for (let i = 0; i < files; i += 1000) {
    const chunk = Array.from({ length: Math.min(1000, files - i) }, (_, j) => join(repo, `f${i + j}.txt`));
    for (const f of chunk) writeFileSync(f, "x");
  }
  // Batch adds in chunks of 500 to avoid per-file git overhead
  for (let i = 0; i < files; i += 500) {
    const batch = Array.from({ length: Math.min(500, files - i) }, (_, j) => `f${i + j}.txt`);
    await git(["add", ...batch], { cwd: repo });
  }
  await git(["commit", "-m", "chore: init"], { cwd: repo });
  return repo;
}

function median(nums: number[]): number {
  const s = [...nums].sort((a, b) => a - b);
  const m = Math.floor(s.length / 2);
  return s.length % 2 ? s[m] : (s[m - 1] + s[m]) / 2;
}

const results: { files: number; median_ms: number }[] = [];
for (const files of SIZES) {
  const repo = await buildRepo(files);
  resetSessionState();
  // Warm-up pass (page cache + git object-dir warmup) — uncounted.
  await writeFile(join(repo, "bench-warm.txt"), "warm");
  await createCheckpoint({ root: repo, sessionId: "bench-warm", turn: 0, label: "warm" });
  resetSessionState();
  const times: number[] = [];
  for (let i = 0; i < RUNS; i++) {
    // each run must change the tree so dedup does not skip
    await writeFile(join(repo, `bench-${i}.txt`), `run ${i}`);
    const t0 = performance.now();
    const r = await createCheckpoint({ root: repo, sessionId: "bench", turn: i + 1, label: `run ${i}` });
    times.push(performance.now() - t0);
    if (r.status !== "created") throw new Error(`bench capture not created: ${JSON.stringify(r)}`);
  }
  const med = median(times);
  results.push({ files, median_ms: Math.round(med * 10) / 10 });
  await rm(repo, { recursive: true, force: true });
}

console.log("files,median_ms");
for (const r of results) console.log(`${r.files},${r.median_ms}`);

const k5 = results.find((r) => r.files === 5000);
if (k5 && k5.median_ms > 150) {
  console.error(`5k-file median ${k5.median_ms}ms > 150ms — failing`);
  process.exit(1);
}
