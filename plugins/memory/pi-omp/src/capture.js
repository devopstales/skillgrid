// capture.js — per-prompt capture through the `save-prompt` verb
// (ADR-0067). Pure logic: spawn is injected (production spawn = index.ts,
// 5 s hard timeout, resolves only on exit 0). Fail-open: a dead or slow
// verb never blocks the host prompt; drops are counted, not thrown.
import { mkdir, readFile, writeFile } from "node:fs/promises"
import { homedir } from "node:os"
import { dirname, join } from "node:path"

const DROP_FILE = ".capture-drops.json"

// The extension-side data dir is the Go default (~/.skillgrid/mnemonic);
// a config-file data_dir override moves the store, the scratch follows
// the default (documented edge, Global Constraints).
export const defaultDataDir = () => join(homedir(), ".skillgrid", "mnemonic")

async function bumpDrops(dataDir, project) {
  const file = join(dataDir, project, DROP_FILE)
  let n = 1
  try { n = (JSON.parse(await readFile(file, "utf8")).n ?? 0) + 1 } catch { /* first drop */ }
  await mkdir(dirname(file), { recursive: true })
  await writeFile(file, JSON.stringify({ n, lastAt: new Date().toISOString() }))
}

export function createCapture({ bin, session, spawn, dataDir = defaultDataDir, projectOf }) {
  return {
    async capture(cwd, prompt) {
      const sid = session.sessionId()
      const text = String(prompt ?? "").trim()
      if (!sid || !text) return
      try {
        await spawn([bin, "memory", "save-prompt", "--id", sid], text, cwd)
      } catch {
        try { await bumpDrops(dataDir(), projectOf(cwd)) } catch { /* scratch is best-effort */ }
      }
    },
  }
}
