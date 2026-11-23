import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import {
  createCheckpoint,
  listCheckpoints,
  pruneOtherSessions,
  repoRoot,
  resetSessionState,
  restoreCheckpoint,
  seedDedup,
  type Checkpoint,
} from "./core/scratch-core.ts";
import { getCache, rebuildCache, restoreDialog, staleMarkerDialog, updateFooter } from "./ui.ts";
import { clearMarker, markWaiting } from "./markers.ts";

// ---------------------------------------------------------------------------
// Scratch extension (extension 1): turn-level scratch checkpoints.
// turn_end → capture (deduped by worktree tree SHA); session_start → cache
// rebuild + stale-marker report + markWaiting + background cross-session
// prune; `scratch-restore` tool + `/checkpoint` command → confirm dialog →
// restore; footer `◆ N checkpoints`; session_shutdown → clearMarker.
// Marker lifecycle (markers.ts is the sole marker writer; the wt extension
// owns agent_start/agent_end). Core (src/core/) stays Pi-free.
// ---------------------------------------------------------------------------

const SCRATCH_RESTORE_DESCRIPTION = [
  "Restore the scratch workspace to a previous checkpoint.",
  'Without "id": lists the current session\'s checkpoints (turn, created, label, id).',
  'With "id": confirms a dialog and restores the worktree (and git index) to that',
  "checkpoint's captured state. Takes an automatic before-restore checkpoint",
  "first. Restore is refused across branches; the refusal names both branches.",
].join("\n");

const LABEL_MAX = 60;

/** Strip whitespace/newlines from a raw prompt, cap at 60 chars (blueprint Task 5). */
function makeLabel(prompt: string): string {
  const t = prompt.trim().replace(/\s+/g, " ").slice(0, LABEL_MAX);
  return t || "turn";
}

function listText(cps: Checkpoint[]): string {
  if (cps.length === 0) return "No checkpoints for this session yet.";
  const lines = cps.map((cp) => `  [turn ${cp.turn}] ${cp.created}  ${cp.label}\n    id: ${cp.id}`);
  return `${cps.length} checkpoint(s), oldest first:\n${lines.join("\n")}\nRestore one: scratch-restore {id: "<id exactly as listed>"}`;
}

export default function (pi: ExtensionAPI) {
  let lastLabel = "";
  let turnIndex = 0; // per session; reset at session_start

  pi.on("before_agent_start", (e) => {
    if (e.messageType === "user") lastLabel = makeLabel(e.prompt);
  });

  pi.on("turn_end", async (_e, ctx) => {
    turnIndex += 1;
    try {
      const r = await createCheckpoint({
        root: await repoRoot(ctx.cwd),
        sessionId: ctx.sessionManager.getSessionId(),
        turn: turnIndex,
        label: lastLabel,
      });
      updateFooter(ctx, r.status === "created");
    } catch {
      // Capture failure must not break the turn; the footer keeps its state.
    }
  });

  pi.on("session_start", async (_e, ctx) => {
    resetSessionState();
    turnIndex = 0;
    lastLabel = "";
    await rebuildCache(ctx);
    // Seed the dedup baseline from the session's newest checkpoint (R3-F007):
    // a resumed session must not re-capture an unchanged tree on its first turn.
    const sid = ctx.sessionManager.getSessionId();
    const newest = await listCheckpoints(await repoRoot(ctx.cwd), sid).catch(() => []);
    if (newest.length) seedDedup(sid, newest[newest.length - 1].worktreeTreeSha);
    // A killed session leaves 💬/🤖 behind (kill before session_shutdown);
    // report it, offering a manual clear, then mark the fresh session 💬.
    void staleMarkerDialog(ctx).catch(() => false);
    void markWaiting(ctx.cwd).catch(() => {});
    // Background, liveness-aware: stale other-session refs (>1h) pruned off-turn.
    void (async () => {
      try {
        await pruneOtherSessions(await repoRoot(ctx.cwd), ctx.sessionManager.getSessionId());
      } catch {
        // prune is best-effort at startup
      }
    })();
  });

  pi.on("session_shutdown", (_e, ctx) => {
    resetSessionState();
    turnIndex = 0;
    lastLabel = "";
    // Clear the marker; if the session was killed (no shutdown) this is the
    // leftover the next session_start reports as stale.
    void clearMarker(ctx.cwd).catch(() => {});
  });

  /** Resolve `target` to a checkpoint of the current session (cache first, refs second). */
  async function findCheckpoint(ctx: ExtensionContext, target: string): Promise<Checkpoint | null> {
    const sessionId = ctx.sessionManager.getSessionId();
    const want = target.startsWith("skillgrid/") ? target.slice("skillgrid/".length) : target;
    const cached = getCache(sessionId).find((c) => c.id === want);
    if (cached) return cached;
    const all = await listCheckpoints(await repoRoot(ctx.cwd), sessionId).catch(() => []);
    return all.find((c) => c.id === want) ?? null;
  }

  /** Confirm dialog → restore; returns the notify/tool text + kind. */
  async function restore(ctx: ExtensionContext, cp: Checkpoint): Promise<{ text: string; kind: "restored" | "refused" | "cancelled" }> {
    const choice = await restoreDialog(ctx, cp);
    if (choice !== "files-only") return { text: `Restore of ${cp.id} cancelled.`, kind: "cancelled" };
    const result = await restoreCheckpoint(await repoRoot(ctx.cwd), cp.id);
    if (result.status === "refused") return { text: `Restore refused: ${result.detail}`, kind: "refused" };
    await rebuildCache(ctx);
    updateFooter(ctx, result.beforeRestore === "created");
    const before =
      result.beforeRestore === "created"
        ? "A before-restore checkpoint was taken first."
        : result.beforeRestore === "skipped"
          ? "The current tree already equals the last capture, so no new before-restore ref was created."
          : "The before-restore checkpoint could not be taken; the restore proceeded without a safety-net ref.";
    return { text: `Restored to ${cp.id} (turn ${cp.turn}, ${cp.label}). ${before}`, kind: "restored" };
  }

  // --- scratch-restore tool ---------------------------------------------------

  pi.registerTool({
    name: "scratch-restore",
    label: "Scratch restore",
    description: SCRATCH_RESTORE_DESCRIPTION,
    promptSnippet: "Restore the scratch workspace to a previous checkpoint (no id → list)",
    promptGuidelines: [
      "Use scratch-restore to undo worktree changes back to a captured checkpoint.",
      "Restore is refused across branches; report the refusal detail to the user.",
    ],
    parameters: Type.Object({
      id: Type.Optional(Type.String({ description: "Checkpoint id to restore (exactly as listed). Omit to list checkpoints." })),
    }, { additionalProperties: false }),
    async execute(_id, p, _s, _u, ctx) {
      const sessionId = ctx.sessionManager.getSessionId();
      if (!p.id) {
        return { content: [{ type: "text" as const, text: listText(getCache(sessionId)) }], details: { listed: true } };
      }
      const cp = await findCheckpoint(ctx, p.id);
      if (!cp) {
        return {
          content: [{ type: "text" as const, text: `No checkpoint ${p.id} for this session.` }],
          details: { restored: false, refused: "missing", detail: `no checkpoint ${p.id}` },
        };
      }
      const outcome = await restore(ctx, cp);
      return { content: [{ type: "text" as const, text: outcome.text }], details: { restored: outcome.kind === "restored", id: cp.id } };
    },
  });

  // --- /checkpoint command ----------------------------------------------------

  pi.registerCommand("checkpoint", {
    description: "List (or restore) the current session's scratch checkpoints",
    handler: async (args, ctx) => {
      const tokens = args.trim().split(/\s+/).filter(Boolean);
      const sessionId = ctx.sessionManager.getSessionId();
      if (tokens.length === 0) {
        ctx.ui.notify(listText(getCache(sessionId)), "info");
        return;
      }
      const cp = await findCheckpoint(ctx, tokens[0]);
      if (!cp) {
        ctx.ui.notify(`No checkpoint ${tokens[0]} for this session.`, "error");
        return;
      }
      const outcome = await restore(ctx, cp);
      ctx.ui.notify(outcome.text, outcome.kind === "refused" ? "error" : "info");
    },
  });
}
