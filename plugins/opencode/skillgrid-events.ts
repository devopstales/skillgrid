// @ts-nocheck
// Dual-compatible: the v1 API uses the named `SkillgridEvents` export below; the
// v2 API (opencode >= 2.0) loads `default` and validates { id, effect|setup }.
// `define` is a no-op identity helper, so the default export is schema-valid
// without pulling the effect runtime into module eval.
import { define } from "@opencode-ai/plugin/v2/promise"
import * as crypto from "node:crypto"

const BASE_URL =
  process.env.SKILLGRID_CHECKPOINT_URL ||
  process.env.SKILLGRID_MNEMONIC_HTTP_URL ||
  "http://127.0.0.1:7438"

const AGENT = process.env.SKILLGRID_AGENT || "opencode"

const MAX_PREVIEW = 500

async function postJSON(pathname, body) {
  try {
    const res = await fetch(BASE_URL + pathname, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      signal: AbortSignal.timeout(5000),
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return await res.json()
  } catch {
    return null
  }
}

function stripPrivateTags(text) {
  if (!text) return text
  return text.replace(/<private>[\s\S]*?<\/private>/g, "[private]")
}

function contentHash(text) {
  if (!text) return null
  return crypto.createHash("sha256").update(text).digest("hex").slice(0, 16)
}

function mapToolType(tool) {
  const t = (tool || "").toLowerCase()
  if (t === "read" || t === "cat" || t.includes("read")) return "file_read"
  if (t === "write" || t === "create" || t.includes("write")) return "file_write"
  if (t === "edit" || t === "patch" || t.includes("edit")) return "file_write"
  if (t === "bash" || t === "shell" || t === "exec" || t.includes("bash")) return "command_exec"
  return "tool_use"
}

function sensitivePath(p) {
  if (!p) return false
  return /(\.env|secret|credential|token|password|private|\.pem|\.key|id_rsa)/i.test(p)
}

function extractFilePath(args) {
  if (!args) return null
  return args.file_path || args.path || args.filePath || args.file || null
}

async function handleToolAfter(input, output) {
  const sessionID = input.sessionID
  const tool = input.tool
  const args = output.args || input.args || {}

  // Fail-open: capture is observe-only, a dead server or a timeout must never
  // block the agent on a tool call.
  const action = mapToolType(tool)
  const filePath = extractFilePath(args)
  const command =
    action === "command_exec" ? args.command || args.cmd || null : null

  let resultPreview = null
  const result = output.result ?? output.output ?? null
  if (typeof result === "string") {
    resultPreview = stripPrivateTags(result.slice(0, MAX_PREVIEW))
  }

  let usage = null
  if (output.usage) {
    usage = {
      input_tokens: output.usage.input_tokens || output.usage.inputTokens || 0,
      output_tokens:
        output.usage.output_tokens || output.usage.outputTokens || 0,
      cache_read_tokens:
        output.usage.cache_read_tokens || output.usage.cacheReadTokens || 0,
      cache_write_tokens:
        output.usage.cache_write_tokens || output.usage.cacheWriteTokens || 0,
    }
  }

  await postJSON("/memory/session-events", {
    session_id: sessionID,
    agent: AGENT,
    action: action,
    tool: tool || null,
    file: filePath,
    command: command,
    result_preview: resultPreview,
    result_hash: contentHash(typeof result === "string" ? result : null),
    usage: usage,
    sensitive: sensitivePath(filePath) || (command && /(\.env|secret|token|password)/i.test(command)),
    source: "skillgrid-events",
  })
}

// Policy gate: a *block* decision is authoritative and MUST surface as a
// thrown error; a *server failure* (network, timeout, non-2xx) is a policy
// failure, not a block, and fails open. A prior bug wrapped the whole gate in
// a try/catch keyed on the message text, which could swallow a block whose
// reason did not match. Keep the two paths distinct.
async function handleToolBefore(input, output) {
  const sessionID = input.sessionID
  const tool = input.tool
  const args = output.args || input.args || {}

  if (tool === "edit" || tool === "write" || tool === "create" || tool === "patch") {
    const filePath = extractFilePath(args)
    if (filePath) {
      let res
      try {
        // ADR-0021: the policy gate is POST /policy/evaluate and answers
        // {effect, message, rule}. effect "block" is authoritative.
        res = await fetch(BASE_URL + "/policy/evaluate", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            session_id: sessionID,
            agent: AGENT,
            action: "file_write",
            tool: tool,
            path: filePath,
          }),
          signal: AbortSignal.timeout(3000),
        })
      } catch {
        return // fail-open: server unreachable, no policy decision
      }
      if (res.ok) {
        const data = await res.json().catch(() => null)
        if (data && data.effect === "block") {
          throw new Error(
            "Write blocked by skillgrid policy: " +
              (data.message || data.rule || "no reason given")
          )
        }
      }
      // non-ok without a parseable body -> fail-open
    }
  }
}

export const SkillgridEvents = async () => ({
  "tool.execute.before": async (input, output) => {
    try {
      await handleToolBefore(input, output)
    } catch (err) {
      // Re-throw only the authoritative policy block; any other error is a
      // gate failure and fails open.
      if (err && err.message && err.message.includes("blocked by skillgrid policy")) {
        throw err
      }
    }
  },
  "tool.execute.after": async (input, output) => {
    try {
      await handleToolAfter(input, output)
    } catch {
      // fail-open: never block the agent on capture failure
    }
  },
})

// v2 entry point: { id, setup }. The v1 `tool.execute.before` (policy gate,
// block-by-throw) has NO v2 equivalent — there is no mutating before-hook — so
// it stays v1-only and the gate is best-effort in v2. `tool.execute.after`
// (capture) IS reproducible via the SSE `session.next.tool.*` events, so setup
// subscribes and posts each tool call/result to /memory/session-events, mirroring
// handleToolAfter. usage tokens are not present on these events (v2 gap) and are
// omitted; result preview is derived from the success/failed content.
export default define({
  id: "skillgrid-events",
  setup: async (_ctx) => {
    const ctrl = new AbortController()
    // callID -> { tool, args, file } captured from .called, joined on .success/.failed
    const inflight = new Map()
    try {
      const { v2SSE } = await import("./skillgrid-v2sse")
      for await (const event of v2SSE(ctrl.signal)) {
        const props = event?.properties || {}
        const sessionID = props.sessionID
        if (!sessionID) continue
        const tool = props.tool || null
        const callID = props.callID
        if (!callID) continue

        if (event.type === "session.next.tool.called") {
          inflight.set(callID, { tool, args: props.input || {} })
        } else if (event.type === "session.next.tool.success" || event.type === "session.next.tool.failed") {
          const meta = inflight.get(callID)
          inflight.delete(callID)
          const args = meta?.args || {}
          const resultText =
            event.type === "session.next.tool.failed"
              ? String(props.error?.message || props.error || "")
              : firstText(props.content)
          await postJSON("/memory/session-events", {
            session_id: sessionID,
            agent: AGENT,
            action: mapToolType(meta?.tool || tool),
            tool: meta?.tool || tool,
            file: extractFilePath(args),
            command:
              mapToolType(meta?.tool || tool) === "command_exec"
                ? args.command || args.cmd || null
                : null,
            result_preview: resultText ? stripPrivateTags(resultText.slice(0, MAX_PREVIEW)) : null,
            result_hash: resultText ? contentHash(resultText) : null,
            sensitive:
              sensitivePath(extractFilePath(args)) ||
              (args.command && /(\.env|secret|token|password)/i.test(args.command)),
            source: "skillgrid-events",
          })
        }
      }
    } catch {
      // fail-open: capture is observe-only; a broken SSE stream must never break
      // plugin setup or the agent
    }
  },
})

// firstText pulls the first text block out of a v2 LlmToolContent[] result.
function firstText(content) {
  if (!Array.isArray(content)) return null
  for (const c of content) {
    if (c && typeof c.text === "string" && c.text) return c.text
    if (c && typeof c === "string") return c
  }
  return null
}
