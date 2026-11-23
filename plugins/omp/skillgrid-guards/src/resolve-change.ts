import { promises as fs } from "node:fs";
import { isAbsolute, join, resolve } from "node:path";
import { parse } from "yaml";

/**
 * Active-change resolution: project root, spec zone, and the change the
 * guards should attribute writes to. The `.skillgrid/` folder is the project
 * marker; `state.yaml` v2 (`changes:` map) names the active change(s).
 *
 * Fail-open by construction: any missing/unparseable input narrows the result
 * (null / ambiguous), it never throws.
 */

export interface ChangeEntry {
  slug: string;
  phase: string;
  status: string;
  gate: string;
}

export interface ChangeRef {
  /** Project root containing `.skillgrid/` */
  root: string;
  /** Resolved spec zone (absolute) */
  specsRoot: string;
  /** Resolved scratch dir (absolute) — execute's ledger zone */
  scratchDir: string;
  /** Acted-on change slug, or null when none active / ambiguous */
  slug: string | null;
  /** Acted-on change dir (absolute), or null */
  changeDir: string | null;
  /** Every `changes:` map entry, including archived */
  entries: ChangeEntry[];
  /** Slugs considered active but not resolvable to a single change */
  ambiguous: string[];
}

const ARCHIVED_PHASES: Record<string, true> = { complete: true, archived: true };

/**
 * mtime-keyed content cache. The hook fires on every write/edit; without this
 * each call re-reads state.yaml + config.yaml + banners. Cache entries are
 * validated by mtime, so an edit to a file invalidates its own entry on the
 * next call. (Tests call clearFileCache between fixtures.)
 */
const fileCache = new Map<string, { mtimeMs: number; size: number; data: string }>();

export async function readFileCached(path: string): Promise<string | null> {
  try {
    const st = await fs.stat(path);
    const hit = fileCache.get(path);
    // mtime + size: on fast tmpfs two writes can share an mtime tick — the
    // size check keeps same-tick rewrites from being served stale.
    if (hit && hit.mtimeMs === st.mtimeMs && hit.size === st.size) return hit.data;
    const data = await fs.readFile(path, "utf8");
    fileCache.set(path, { mtimeMs: st.mtimeMs, size: st.size, data });
    return data;
  } catch {
    return null;
  }
}

export function clearFileCache(): void {
  fileCache.clear();
}

/** Walk up from startDir (at most 10 levels) looking for a `.skillgrid/` dir. */
export async function findProjectRoot(startDir: string): Promise<string | null> {
  let dir = resolve(startDir);
  for (let i = 0; i < 10; i++) {
    try {
      const st = await fs.stat(join(dir, ".skillgrid"));
      if (st.isDirectory()) return dir;
    } catch {
      /* keep climbing */
    }
    const parent = join(dir, "..");
    if (parent === dir) break;
    dir = parent;
  }
  return null;
}

function toEntries(state: Record<string, unknown> | null): ChangeEntry[] {
  if (!state) return [];
  const changes = state.changes;
  if (!changes || typeof changes !== "object") return [];
  return Object.entries(changes as Record<string, unknown>).map(([slug, raw]) => {
    const e = (raw ?? {}) as Record<string, unknown>;
    return {
      slug,
      phase: typeof e.phase === "string" ? e.phase : "",
      status: typeof e.status === "string" ? e.status : "in_progress",
      gate: typeof e.gate === "string" ? e.gate : "",
    };
  });
}

/**
 * Resolve the project + acted-on change for a write/edit happening at `cwd`.
 * Returns null when the cwd is not inside a skillgrid project (guards off).
 *
 * Acted-on change: exactly one non-archived `changes:` entry. Zero entries with
 * a missing/unparseable state.yaml falls back to a single spec-zone dir.
 * Multiple active changes → ambiguous → slug null (guards fail open and the
 * caller may surface a note).
 */
export async function resolveChange(cwd: string): Promise<ChangeRef | null> {
  const root = await findProjectRoot(cwd);
  if (!root) return null;

  const configRaw = await readFileCached(join(root, ".skillgrid", "config.yaml"));
  let specsRoot = join(root, ".skillgrid", "change-specs");
  let scratchDir = join(root, ".skillgrid", "work");
  if (configRaw) {
    try {
      const config = parse(configRaw) as Record<string, unknown> | null;
      const conv = config?.conventions as Record<string, unknown> | undefined;
      if (conv) {
        if (typeof conv.specs_root === "string" && conv.specs_root) {
          specsRoot = isAbsolute(conv.specs_root) ? conv.specs_root : join(root, conv.specs_root);
        }
        if (typeof conv.scratch_dir === "string" && conv.scratch_dir) {
          scratchDir = isAbsolute(conv.scratch_dir) ? conv.scratch_dir : join(root, conv.scratch_dir);
        }
      }
    } catch {
      /* malformed config: defaults */
    }
  }

  const stateRaw = await readFileCached(join(root, ".skillgrid", "state.yaml"));
  let state: Record<string, unknown> | null = null;
  if (stateRaw) {
    try {
      const parsed = parse(stateRaw);
      if (parsed && typeof parsed === "object") state = parsed as Record<string, unknown>;
    } catch {
      state = null;
    }
  }

  const entries = toEntries(state);
  const active = entries.filter(
    (e) => !ARCHIVED_PHASES[e.phase] && e.status !== "complete",
  );

  let slug: string | null = null;
  let ambiguous: string[] = [];
  if (active.length === 1) {
    slug = active[0].slug;
  } else if (active.length > 1) {
    ambiguous = active.map((e) => e.slug);
  } else if (state === null) {
    // No parseable state.yaml: fall back to the spec-zone directory listing.
    try {
      const dirs = (await fs.readdir(specsRoot, { withFileTypes: true }))
        .filter((d) => d.isDirectory())
        .map((d) => d.name)
        .sort();
      if (dirs.length === 1) slug = dirs[0];
      else if (dirs.length > 1) ambiguous = dirs;
    } catch {
      /* spec zone absent */
    }
  }

  const changeDir = slug ? join(specsRoot, slug) : null;
  return { root, specsRoot, scratchDir, slug, changeDir, entries, ambiguous };
}
