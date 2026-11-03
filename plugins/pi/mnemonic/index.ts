// @ts-nocheck
/**
 * skillgrid-mnemonic — Pi extension adapter (ADR-0043).
 *
 * Connects Pi (skill-pi) session events to the Mnemonic HTTP server.
 * Persistence remains owned by the Go binary (`mnemonic serve`). Mirrors the
 * opencode/kilo native plugin architecture (ADR-0032) for the Pi harness:
 *
 *   - registers the native mem_* / fact_* / code_* tools (typebox schemas)
 *   - captures user prompts on before_agent_start (POST /prompts)
 *   - injects the Mnemonic Memory Protocol into the system prompt
 *   - builds the compaction context block on session_before_compact
 *   - ends the session on session_shutdown (quit)
 *
 * Folder format (like plugins/opencode/):
 *   index.ts               — entrypoint: tool registration + event wiring
 *   api.js                 — shared HTTP / redaction / TypeBox helpers
 *   mnemonic-memory.ts     — mem_* tools
 *   mnemonic-facts.ts      — fact_* tools
 *   mnemonic-codeindex.ts  — code_* tools
 *   ctx-sandbox.ts         — ctx_* tools + tool_result overflow receipts (ADR-0044)
 *   skillgrid-events.ts    — session + prompt-capture handlers
 *   skillgrid-sdd-events.ts— SDD commit + gate-stop capture
 *   skillgrid-compaction.ts— compaction context recovery
 *
 * Environment:
 *   SKILLGRID_MNEMONIC_HTTP_URL  base URL (default http://127.0.0.1:7438)
 *   SKILLGRID_HTTP_TOKEN         bearer token for write routes (optional)
 *   SKILLGRID_AGENT              agent label (default "pi")
 *
 * Every capture path is fail-open: a dead server or timeout never blocks the
 * agent on a prompt, tool call, or shutdown.
 */

import {
  memSave,
  memSearch,
  memGetObservation,
  memUpdate,
  memSessionSummary,
  memContext,
  memCurrentProject,
  memDoctor,
  memCapturePassive,
} from "./mnemonic-memory.ts"
import { factAdd, factSearch, factForget } from "./mnemonic-facts.ts"
import { codeStatus, codeIndex, codeSearch, codeRead } from "./mnemonic-codeindex.ts"
import { registerSessionHandlers } from "./skillgrid-events.ts"
import { registerSddEventCapture } from "./skillgrid-sdd-events.ts"
import { buildCompactionContext } from "./skillgrid-compaction.ts"
import {
  ctxSearch,
  ctxGet,
  ctxStats,
  ctxPurge,
  registerToolResultOverflow,
} from "./ctx-sandbox.ts"

const MEMORY_PROTOCOL_SECTION = "## Mnemonic Persistent Memory — Protocol"

const MEMORY_PROTOCOL_BODY =
  "You have Mnemonic persistent memory (mem_* tools). Follow the recovery ladder: " +
  "mem_context first for recent summaries, mem_search for topic recall, mem_get_observation " +
  "for full content before relying on a hit. Save significant decisions, bug fixes, discoveries, " +
  "and preferences with mem_save (structured What/Why/Where/Learned). Reuse topic_key via " +
  "mem_update for evolving topics — do not duplicate. Before explicitly closing a session, call " +
  "mem_session_summary with the six sections (Goal, Instructions, Discoveries, Accomplished, " +
  "Next Steps, Relevant Files). Memory failures never block the user-facing answer."

export default function skillgridMnemonic(pi) {
  // Tools
  pi.registerTool(memSave)
  pi.registerTool(memSearch)
  pi.registerTool(memGetObservation)
  pi.registerTool(memUpdate)
  pi.registerTool(memSessionSummary)
  pi.registerTool(memContext)
  pi.registerTool(memCurrentProject)
  pi.registerTool(memDoctor)
  pi.registerTool(memCapturePassive)
  pi.registerTool(factAdd)
  pi.registerTool(factSearch)
  pi.registerTool(factForget)
  pi.registerTool(codeStatus)
  pi.registerTool(codeIndex)
  pi.registerTool(codeSearch)
  pi.registerTool(codeRead)
  pi.registerTool(ctxSearch)
  pi.registerTool(ctxGet)
  pi.registerTool(ctxStats)
  pi.registerTool(ctxPurge)

  // Session start / prompt capture / shutdown (fail-open, in skillgrid-events.js)
  registerSessionHandlers(pi)

  // SDD capture: work-unit commit events (tool_result on git commit) and the
  // gate-stop state at quit (fail-open, in skillgrid-sdd-events.js)
  registerSddEventCapture(pi)

  // ADR-0044: oversized tool results become server-side receipts with the
  // retrieval path (pi-context pattern). Fail-open, in ctx-sandbox.js.
  registerToolResultOverflow(pi)

  // Inject the Memory Protocol once per extension instance.
  pi.on("before_agent_start", (event) => {
    try {
      const opts = event?.systemPromptOptions
      if (
        opts &&
        typeof opts.appendSystemPrompt === "string" &&
        !opts.appendSystemPrompt.includes(MEMORY_PROTOCOL_SECTION)
      ) {
        const section = `${MEMORY_PROTOCOL_SECTION}\n\n${MEMORY_PROTOCOL_BODY}`
        opts.appendSystemPrompt = opts.appendSystemPrompt
          ? `${opts.appendSystemPrompt}\n\n${section}`
          : section
      }
    } catch {
      // fail-open
    }
  })

  // Compaction: refresh the session context block before the summary is
  // written (fail-open, in skillgrid-compaction.js).
  pi.on("session_before_compact", async (_event, ctx) => {
    await buildCompactionContext(ctx)
  })
}
