// @ts-nocheck
// Mnemonic event capture for the Pi extension (ADR-0043): registers the host
// session on session_start, captures user prompts on before_agent_start
// (POST /prompts), and ends the session on quit. Every path is fail-open —
// a dead server never blocks a prompt or shutdown.

import { AGENT, bestEffort, redactPrivateTags, sessionIDOf } from "./api.js"

const MIN_PROMPT_LENGTH = 11 // matches the Go server's minPromptLength
const MAX_PROMPT_LENGTH = 2000 // matches the Go server's MaxPromptLength

export async function capturePrompt(ctx, prompt) {
  const sid = sessionIDOf(ctx)
  if (!sid) return
  const text = redactPrivateTags(String(prompt || ""))
  if (text.trim().length < MIN_PROMPT_LENGTH) return // short prompts are not useful
  await bestEffort("/prompts", {
    session_id: sid,
    content: text.slice(0, MAX_PROMPT_LENGTH),
  })
}

export function registerSessionHandlers(pi) {
  // Session start: register the host session id under the current project.
  pi.on("session_start", async (_event, ctx) => {
    const sid = sessionIDOf(ctx)
    if (!sid) return
    await bestEffort("/sessions", {
      id: sid,
      agent: AGENT,
      directory: ctx?.cwd || process.cwd(),
    })
  })

  // Prompt capture: after Pi's input expansion, once per user run.
  pi.on("before_agent_start", async (event, ctx) => {
    try {
      if (event?.messageType !== "user") return
      await capturePrompt(ctx, event?.prompt)
    } catch {
      // fail-open
    }
  })

  // Shutdown: end the session (quit only — not reload/new/resume/fork,
  // which replace the session without ending it).
  pi.on("session_shutdown", async (event, ctx) => {
    if (event?.reason !== "quit") return
    const sid = sessionIDOf(ctx)
    if (!sid) return
    await bestEffort(`/sessions/${encodeURIComponent(sid)}/end`, { agent: AGENT })
  })
}
