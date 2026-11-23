import { Text } from "@earendil-works/pi-tui";
import type { ExtensionContext } from "@earendil-works/pi-coding-agent";
import { git, listCheckpoints, repoRoot, type Checkpoint } from "./core/scratch-core.ts";
import { clearMarker, markerStatus, staleMarkerReport } from "./markers.ts";

// ---------------------------------------------------------------------------
// Shared UI state for the scratch extension (extension 1): the checkpoint
// cache, the footer, and the restore dialog. Task 8 (wt markers) extends
// this file's dialog half; keep restoreDialog's "files-only" | "cancel"
// return contract.
//
// Pinned shapes (blueprint Task 5, Stage 9 F1):
//   updateFooter(ctx, created: boolean): void — renders `◆ N checkpoints`
//     (singular `1 checkpoint`) for the current session; `created` triggers
//     a brief "✓ checkpoint" flash
//   rebuildCache(ctx): Promise<void> — walks refs/skillgrid via
//     listCheckpoints, populates the in-memory Map<sessionId, Checkpoint[]>
//   restoreDialog(ctx, cp: Checkpoint): Promise<"files-only" | "cancel"> —
//     one-line diff preview + ["files-only", "cancel"] select
// ---------------------------------------------------------------------------

/** sessionId → checkpoints, oldest first. */
const cache = new Map<string, Checkpoint[]>();

const FOOTER_STATUS_KEY = "scratch";
const FLASH_MS = 5000;
const DIFF_PREVIEW_PATHS = 5;

let flashUntil = 0;

/** The current session's cached checkpoints (oldest first); [] when uncached. */
export function getCache(sessionId: string): Checkpoint[] {
  return cache.get(sessionId) ?? [];
}

/** Footer line: `◆ N checkpoints` (singular at 1), with a fresh-capture flash. */
function footerLine(sessionId: string): string {
  const n = (cache.get(sessionId) ?? []).length;
  const label = `◆ ${n} checkpoint${n === 1 ? "" : "s"}`;
  return Date.now() < flashUntil ? `✓ checkpoint — ${label}` : label;
}

/** Footer text component; registered once with the TUI, refreshed in place. */
const footerText = new Text();

export function updateFooter(ctx: ExtensionContext, created: boolean): void {
  const sessionId = ctx.sessionManager.getSessionId();
  if (created) {
    flashUntil = Date.now() + FLASH_MS;
    setTimeout(() => footerText.invalidate(), FLASH_MS);
  }
  footerText.setText(footerLine(sessionId));
  // Status slot: visible in all dialog-capable modes.
  ctx.ui.setStatus(FOOTER_STATUS_KEY, footerLine(sessionId));
  if (ctx.mode === "tui") {
    ctx.ui.setFooter(() => footerText);
  }
}

/**
 * Rebuild the cache for the current session from `refs/skillgrid/`
 * (pinned shape, Stage 9 F1). A resumed session's dedup baseline is seeded
 * implicitly: core's capture compares the new worktree tree SHA against
 * lastTree, so an unchanged first turn returns `skipped`, not a duplicate.
 */
export async function rebuildCache(ctx: ExtensionContext): Promise<void> {
  const sessionId = ctx.sessionManager.getSessionId();
  try {
    cache.set(sessionId, await listCheckpoints(await repoRoot(ctx.cwd), sessionId));
  } catch {
    cache.set(sessionId, []);
  }
}

/** One-line diff preview: file count + first 5 changed paths vs current HEAD. */
async function diffPreview(root: string, cp: Checkpoint): Promise<string> {
  const out = await git(["diff", "--name-only", cp.worktreeTreeSha, "HEAD"], { cwd: root }).catch(() => "");
  const paths = out.split("\n").map((l) => l.trim()).filter(Boolean);
  if (paths.length === 0) return `restores the tree at ${cp.id}`;
  const shown = paths.slice(0, DIFF_PREVIEW_PATHS).join(", ");
  const more = paths.length > DIFF_PREVIEW_PATHS ? `, … +${paths.length - DIFF_PREVIEW_PATHS} more` : "";
  return `${paths.length} file(s) vs current: ${shown}${more}`;
}

/**
 * Restore confirmation (pinned shape, Stage 9 F1): one-line diff preview in
 * the dialog, then ["files-only", "cancel"]. Print/json modes have no
 * dialog: a headless restore with an explicit id proceeds ("files-only").
 */
export async function restoreDialog(ctx: ExtensionContext, cp: Checkpoint): Promise<"files-only" | "cancel"> {
  if (ctx.mode !== "tui" && ctx.mode !== "rpc") return "files-only";
  const root = await repoRoot(ctx.cwd).catch(() => ctx.cwd);
  const preview = await diffPreview(root, cp);
  const answer = await ctx.ui.select(
    `Restore checkpoint ${cp.id}?\nTurn ${cp.turn} · ${cp.created} · branch ${cp.branch}\n${cp.label}\n${preview}`,
    ["files-only", "cancel"],
  );
  return answer === "files-only" ? "files-only" : "cancel";
}

// ---------------------------------------------------------------------------
// Marker dialog half (blueprint Task 8): a marker left over from a killed
// session (kill before session_shutdown → clearMarker never ran) is reported
// on session_start; the user can clear it manually or keep it.
// ---------------------------------------------------------------------------

/**
 * Report a stale marker, offering a manual clear (exact report text from
 * findings.md R1). Non-dialog modes just notify. Resolves true when the
 * marker was cleared.
 */
export async function staleMarkerDialog(ctx: ExtensionContext): Promise<boolean> {
  const marker = await markerStatus(await repoRoot(ctx.cwd).catch(() => ctx.cwd)).catch(() => undefined);
  if (marker === undefined) return false;
  const report = staleMarkerReport(ctx.cwd, marker);
  if (ctx.mode !== "tui" && ctx.mode !== "rpc") {
    ctx.ui.notify(report, "warning");
    return false;
  }
  const answer = await ctx.ui.select(report + "\nClear it?", ["clear", "keep"]);
  if (answer !== "clear") return false;
  await clearMarker(ctx.cwd);
  return true;
}
