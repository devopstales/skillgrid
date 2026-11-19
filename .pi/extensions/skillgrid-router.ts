// .pi/extensions/skillgrid-router.ts — pi project extension (TICKET-03).
//
// Deterministic router-bootstrap injection for the plain pi host
// (blueprint Task 3; Design contract: "Pointer bootstrap, not full-content
// injection"). Pointer bootstrap: the shared asset is appended as a small
// bootstrap section; the full router loads on demand via the skill system.
//
// Mechanism: `before_agent_start` → append the bootstrap section to the
// system prompt via `event.systemPromptOptions.appendSystemPrompt`.
//
// Guards (R3): main agent only (`ctx.agent.kind === "main"`); project only
// (a `.skillgrid/config.yaml` exists at or above cwd — bounded walk-up).
// Fail-open (R4): never throw into the host; unresolvable asset → static
// fallback; non-project cwd → no injection.

import fs from "node:fs";
import path from "node:path";
import { homedir } from "node:os";

const BOOTSTRAP_REL = path.join(".agents", "skills", "_shared", "agent-config", "router-bootstrap.md");
const BOOTSTRAP_MARKER = "Using skillgrid:using-skillgrid";
const STATIC_BOOTSTRAP =
  "skillgrid: router bootstrap fallback - first action: read `.agents/skills/using-skillgrid/SKILL.md`; announce `Using skillgrid:using-skillgrid to {purpose}`.";

type RouterBeforeAgentStartEvent = { systemPromptOptions?: { appendSystemPrompt?: string } };
type RouterBeforeAgentStartContext = { cwd?: string; agent?: { kind?: string } };

function findProjectRoot(cwd: string): string | null {
  let dir = path.resolve(cwd || process.cwd());
  for (;;) {
    if (fs.existsSync(path.join(dir, ".skillgrid", "config.yaml"))) return dir;
    const parent = path.dirname(dir);
    if (parent === dir) return null;
    dir = parent;
  }
}

function resolveBootstrap(projectRoot: string): string {
  const installed = path.join(process.env.HOME || homedir(), ".skillgrid", "router-bootstrap.md");
  if (fs.existsSync(installed)) return fs.readFileSync(installed, "utf8");
  const asset = path.join(projectRoot, BOOTSTRAP_REL);
  if (fs.existsSync(asset)) return fs.readFileSync(asset, "utf8");
  return STATIC_BOOTSTRAP;
}

export default function (pi: ExtensionAPI) {
  pi.on("before_agent_start", (event: RouterBeforeAgentStartEvent, ctx: RouterBeforeAgentStartContext) => {
    try {
      if (ctx?.agent?.kind !== "main") return;
      const projectRoot = findProjectRoot(ctx?.cwd || process.cwd());
      if (!projectRoot) return;
      const bootstrap = resolveBootstrap(projectRoot);
      const opts = event?.systemPromptOptions;
      if (opts) {
        const existing = typeof opts.appendSystemPrompt === "string" ? opts.appendSystemPrompt : "";
        if (!existing.includes(BOOTSTRAP_MARKER)) {
          opts.appendSystemPrompt = existing ? `${existing}\n\n${bootstrap}` : bootstrap;
        }
      }
    } catch {
      // fail-open (R4): never surface an error into the host session
    }
  });
}
