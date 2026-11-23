import { checkWriteEdit } from "../src/guards.ts";

/**
 * omp hook factory: the skillgrid P0 guard trio on `tool_call` (pre-execution)
 * for the `write` and `edit` tools.
 *
 * Discovery: ambient project discovery loads `<cwd>/.omp/hooks/pre/*.ts`
 * (https://omp.sh/docs/hooks); this project wires it via the thin shim at
 * `.omp/hooks/pre/skillgrid-guards.ts`.
 *
 * Fail-open: the legacy runner fails closed when a tool_call handler throws,
 * so every guard error is caught here — a guard bug degrades to no-op, never
 * blocks the session.
 */

type Handler = (
  event: { toolName: string; input?: Record<string, unknown> },
  ctx: { cwd?: string },
) => unknown;

export default function skillgridGuards(
  pi: { on: (event: string, handler: Handler) => void },
): void {
  pi.on("tool_call", async (event, ctx) => {
    if (event.toolName !== "write" && event.toolName !== "edit") return;
    try {
      const cwd = typeof ctx?.cwd === "string" && ctx.cwd !== "" ? ctx.cwd : process.cwd();
      const d = await checkWriteEdit(cwd, event.toolName, (event.input ?? {}) as Record<string, unknown>);
      if (!d) return;
      if (d.block) return { block: true, reason: d.reason };
      if (d.context) return { additionalContext: d.context };
    } catch {
      /* fail-open */
    }
  });
}
