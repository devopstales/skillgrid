import { promises as fs } from "node:fs";
import { basename, isAbsolute, relative, resolve } from "node:path";
import { checkBannerEdit } from "./banner.ts";
import { checkArtifactOrder } from "./artifact-order.ts";
import { checkFileSet } from "./fileset.ts";
import { resolveChange, readFileCached } from "./resolve-change.ts";
import type { Decision, ToolInput } from "./types.ts";

/**
 * Combined guard entrypoint: one call per write/edit, runs the P0 guard trio
 * (spec-zone file set, STATUS banner ownership, strict artifact order) and
 * merges the decisions.
 *
 * Merge rule: the first `block` wins (reason is the error text); non-blocking
 * `context` notes are joined. Any internal error → undefined (fail-open: a
 * guard bug must not brick the session; the hook wrapper is fail-closed on
 * thrown handlers, so everything is caught here).
 */

async function pathExists(path: string): Promise<boolean> {
  try {
    await fs.access(path);
    return true;
  } catch {
    return false;
  }
}

/**
 * Simulate the post-edit content for the banner check. Returns null when the
 * result cannot be determined (missing input, old_string not found — the tool
 * itself will then fail or apply fuzzy matching the guard cannot see).
 */
function simulatedContent(
  toolName: string,
  input: ToolInput,
  oldContent: string | null,
): string | null {
  if (toolName === "write") {
    return typeof input.content === "string" ? input.content : null;
  }
  const oldStr = typeof input.old_string === "string" ? input.old_string : null;
  const newStr = typeof input.new_string === "string" ? input.new_string : null;
  if (oldContent === null || oldStr === null || oldStr === "" || newStr === null) {
    return null;
  }
  if (!oldContent.includes(oldStr)) return null;
  return input.replace_all === true
    ? oldContent.split(oldStr).join(newStr)
    : oldContent.replace(oldStr, newStr);
}

function targetPath(cwd: string, input: ToolInput): string | null {
  const raw =
    typeof input.path === "string" && input.path !== ""
      ? input.path
      : Array.isArray(input.paths) && typeof input.paths[0] === "string"
        ? input.paths[0]
        : null;
  if (raw === null) return null;
  return isAbsolute(raw) ? resolve(raw) : resolve(cwd, raw);
}

/**
 * Check a pending write/edit.
 *
 * @param cwd       the session's working directory (hook ctx.cwd)
 * @param toolName  "write" | "edit" (anything else → undefined)
 * @param input     the tool's normalized input
 */
export async function checkWriteEdit(
  cwd: string,
  toolName: string,
  input: ToolInput,
): Promise<Decision | undefined> {
  if (toolName !== "write" && toolName !== "edit") return undefined;

  const target = targetPath(cwd, input);
  if (target === null) return undefined;

  const ref = await resolveChange(cwd);
  if (ref === null) return undefined; // not a skillgrid project

  let blockReason: string | undefined;
  const contexts: string[] = [];

  const decide = (d: { block?: boolean; reason?: string; context?: string } | undefined) => {
    if (!d) return;
    if (d.block && blockReason === undefined) blockReason = d.reason;
    if (d.context && blockReason === undefined) contexts.push(d.context);
  };

  // 1. Spec-zone file set (new files only block; stray existing files warn).
  const rel = relative(ref.specsRoot, target);
  if (rel !== "" && !rel.startsWith("..") && !isAbsolute(rel)) {
    const isNewFile = !(await pathExists(target));
    decide(checkFileSet(rel, isNewFile));
  }

  // 2. STATUS banner ownership (the four banner files).
  const oldContent = await readFileCached(target);
  const newContent = simulatedContent(toolName, input, oldContent);
  if (newContent !== null) {
    decide(checkBannerEdit(basename(target), oldContent, newContent));
  }

  // 3. Strict artifact order (predecessor chain + execution gate).
  const entry = ref.slug ? (ref.entries.find((e) => e.slug === ref.slug) ?? null) : null;
  decide(
    await checkArtifactOrder(target, {
      root: ref.root,
      specsRoot: ref.specsRoot,
      scratchDir: ref.scratchDir,
      changeDir: ref.changeDir,
      slug: ref.slug,
      entry,
    }),
  );

  if (blockReason !== undefined) return { block: true, reason: blockReason };
  if (contexts.length > 0) return { context: contexts.join("\n") };
  return undefined;
}
