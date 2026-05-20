import { promises as fs, realpathSync } from "node:fs";
import path from "node:path";

/** Expand leading `~` (Node `path.resolve` does not). */
export function expandUserPath(input: string): string {
  const trimmed = input.trim();
  if (trimmed === "~") {
    return process.env.HOME ?? process.env.USERPROFILE ?? trimmed;
  }
  if (trimmed.startsWith("~/")) {
    const home = process.env.HOME ?? process.env.USERPROFILE ?? "";
    return home ? path.join(home, trimmed.slice(2)) : trimmed;
  }
  return trimmed;
}

/** Canonical repo root for dashboard + opensessions matching. */
export function resolveRepoRoot(repoRoot: string): string {
  return path.resolve(expandUserPath(repoRoot));
}

function tryRealpath(filePath: string): string {
  try {
    return realpathSync.native(filePath);
  } catch {
    return path.resolve(filePath);
  }
}

/**
 * True when a tmux session cwd is the repo root or a subdirectory (symlink-safe).
 */
export function sessionDirMatchesRepo(sessionDir: string, repoRoot: string): boolean {
  if (!sessionDir?.trim()) return false;
  const resolvedSession = tryRealpath(path.resolve(sessionDir));
  const resolvedRepo = tryRealpath(resolveRepoRoot(repoRoot));
  if (resolvedSession === resolvedRepo) return true;
  const repoPrefix = resolvedRepo.endsWith(path.sep) ? resolvedRepo : `${resolvedRepo}${path.sep}`;
  return resolvedSession.startsWith(repoPrefix);
}

export async function pathExists(filePath: string): Promise<boolean> {
  try {
    await fs.access(filePath);
    return true;
  } catch {
    return false;
  }
}

export async function readTextIfExists(filePath: string): Promise<string | undefined> {
  try {
    return await fs.readFile(filePath, "utf8");
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "ENOENT") {
      return undefined;
    }
    throw error;
  }
}

export async function listFilesRecursive(root: string): Promise<string[]> {
  if (!(await pathExists(root))) {
    return [];
  }

  const entries = await fs.readdir(root, { withFileTypes: true });
  const files = await Promise.all(
    entries.map(async (entry) => {
      const entryPath = path.join(root, entry.name);
      if (entry.isDirectory()) {
        return listFilesRecursive(entryPath);
      }
      return [entryPath];
    })
  );

  return files.flat();
}

export async function listDirectories(root: string): Promise<string[]> {
  if (!(await pathExists(root))) {
    return [];
  }

  const entries = await fs.readdir(root, { withFileTypes: true });
  return entries.filter((entry) => entry.isDirectory()).map((entry) => path.join(root, entry.name));
}

export function toPosixRelative(repoRoot: string, filePath: string): string {
  return path.relative(repoRoot, filePath).split(path.sep).join("/");
}

export function slugify(value: string): string {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export function isInside(parent: string, child: string): boolean {
  const relative = path.relative(parent, child);
  return Boolean(relative) && !relative.startsWith("..") && !path.isAbsolute(relative);
}
