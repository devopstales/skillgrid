// @ts-nocheck
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

// R3: an unreadable or non-regular-file config is "no project" (fail-open).
function isReadableProjectConfig(p: string): boolean {
  try {
    if (!fs.statSync(p).isFile()) return false;
    fs.accessSync(p, fs.constants.R_OK);
    return true;
  } catch {
    return false;
  }
}

function findProjectRoot(cwd: string): string | null {
  let dir = path.resolve(cwd || process.cwd());
  for (;;) {
    if (isReadableProjectConfig(path.join(dir, ".skillgrid", "config.yaml"))) return dir;
    const parent = path.dirname(dir);
    if (parent === dir) return null;
    dir = parent;
  }
}

function resolveBootstrap(projectRoot: string): string {
  // Installed copy wins, then the checkout asset; missing, unreadable, or
  // whitespace-only copies fall through (R4).
  for (const candidate of [
    path.join(process.env.HOME || homedir(), ".skillgrid", "router-bootstrap.md"),
    path.join(projectRoot, BOOTSTRAP_REL),
  ]) {
    try {
      const raw = fs.readFileSync(candidate, "utf8");
      if (raw.trim()) return raw;
    } catch {
      // missing/unreadable copy -> next source
    }
  }
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
