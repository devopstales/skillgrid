// @ts-nocheck
// .omp/extensions/skillgrid-router-omp.ts — omp project extension
// (2026-11-17-skillgrid-host-plugins, TICKET-07 fix).
//
// Full-content router injection for the omp host (blueprint: "omp keeps the
// shipped full-content hook" — the whole using-skillgrid SKILL.md, not the
// 15-line pointer bootstrap the other hosts get).
//
// Mechanism: `before_agent_start` → append the router as a system-prompt
// section (event.systemPrompt is the live array of prompt sections in
// omp 18.16; a pushed section reaches the model — verified by behavioral
// probe in TICKET-07). NOTE: `systemPromptOptions.appendSystemPrompt` is
// NOT honored by omp 18.16 (probe-verified), and the @hsingjui/pi-hooks
// plugin's SessionStart listener never fires on omp (npm plugins register
// after the session_start dispatch) — hence this native extension.
//
// Guards (R3): main agent only (ctx.agent.kind === "main"); project only
// (a `.skillgrid/config.yaml` exists at or above cwd — bounded walk-up).
// Fail-open (R4): any error skips injection; session start never breaks.

import fs from "node:fs";
import path from "node:path";
import { homedir } from "node:os";

const ROUTER_REL = path.join(".agents", "skills", "using-skillgrid", "SKILL.md");
const INTRO =
  "skillgrid loaded. Route through the using-skillgrid flow before ANY response (including clarifying questions).";
const STATIC_ROUTER =
  "skillgrid: using-skillgrid router not resolvable (installed copy + checkout both missing) - read `.agents/skills/using-skillgrid/SKILL.md` as the first action of any skillgrid task.";

function findProjectRoot(cwd: string): string | null {
  let dir = path.resolve(cwd || process.cwd());
  for (;;) {
    if (fs.existsSync(path.join(dir, ".skillgrid", "config.yaml"))) return dir;
    const parent = path.dirname(dir);
    if (parent === dir) return null;
    dir = parent;
  }
}

function resolveRouter(projectRoot: string): string {
  // Installed copy (installer-mirrored .agents) wins, then the checkout
  // asset, then the static fallback (fail-open, R4).
  const installed = path.join(process.env.HOME || homedir(), ".skillgrid", ROUTER_REL);
  try {
    return fs.readFileSync(installed, "utf8");
  } catch {
    // fall through
  }
  try {
    return fs.readFileSync(path.join(projectRoot, ROUTER_REL), "utf8");
  } catch {
    return STATIC_ROUTER;
  }
}

type RouterBeforeAgentStartEvent = { systemPrompt?: string[] };
type RouterBeforeAgentStartContext = { cwd?: string; agent?: { kind?: string } };

export default function (pi: ExtensionAPI) {
  pi.on("before_agent_start", (event: RouterBeforeAgentStartEvent, ctx: RouterBeforeAgentStartContext) => {
    try {
      // Main agent only (R3): subagents/child sessions carry depth > 0 or a
      // non-main agent kind and must not receive the router twice.
      if (!ctx || !ctx.agent || ctx.agent.kind !== "main") return;
      const root = findProjectRoot(ctx.cwd || process.cwd());
      if (!root) return;
      const sections = event.systemPrompt;
      if (!Array.isArray(sections)) return; // unexpected shape: fail open
      sections.push(`${INTRO}\n\n${resolveRouter(root)}`);
    } catch {
      // Fail-open: injection must never break session start (R4).
    }
  });
}
