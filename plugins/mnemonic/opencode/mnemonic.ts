/**
 * Skillgrid Mnemonic — OpenCode / Kilo plugin adapter
 *
 * Thin HTTP layer for session lifecycle, protocol injection, compaction recovery,
 * and code-index nudges. Agent tool calls use MCP (`skillgrid mcp` in opencode.json).
 *
 * Flow:
 *   OpenCode hooks → this plugin → HTTP → skillgrid serve → SQLite
 */

import type { Plugin } from "@opencode-ai/plugin"
import {
  ensureServer,
  getBaseUrl,
  mnemonicFetch,
} from "../shared/http-client.ts"

const PROTOCOL_REL = "plugins/mnemonic/shared/memory-protocol.md"

const COMPACTION_REMINDER =
  `CRITICAL INSTRUCTION FOR COMPACTED SUMMARY:\n` +
  `The agent has access to Skillgrid Mnemonic persistent memory via MCP tools.\n` +
  `You MUST include the following instruction at the TOP of the compacted summary:\n\n` +
  `"FIRST ACTION REQUIRED: Call mem_session_summary with the content of this compacted summary. ` +
  `This preserves what was accomplished before compaction. Do this BEFORE any other work."\n\n` +
  `This is NOT optional. Without this, everything done before compaction is lost from memory.`

// ─── Helpers ────────────────────────────────────────────────────────────────

function stripPrivateTags(str: string): string {
  if (!str) return ""
  return str.replace(/<private>[\s\S]*?<\/private>/gi, "[REDACTED]").trim()
}

function protocolPaths(): string[] {
  const home = process.env.HOME ?? ""
  return [
    `${home}/.skillgrid/repos/skillgrid/${PROTOCOL_REL}`,
    `${home}/.skillgrid/repos/aiskillgrid/${PROTOCOL_REL}`,
  ]
}

async function loadMemoryProtocol(): Promise<string> {
  for (const path of protocolPaths()) {
    try {
      const file = Bun.file(path)
      if (await file.exists()) {
        const text = await file.text()
        if (text.trim()) return text.trim()
      }
    } catch {
      /* try next path */
    }
  }
  return ""
}

function appendToSystem(system: string[], text: string): void {
  if (!text) return
  if (system.length > 0) {
    system[system.length - 1] += "\n\n" + text
  } else {
    system.push(text)
  }
}

function formatContextSessions(
  sessions: Array<{ summary?: string; started_at?: string; id?: string }>,
): string {
  const blocks = sessions
    .filter((s) => s.summary?.trim())
    .map((s) => {
      const when = s.started_at ? `[${s.started_at}] ` : ""
      return `${when}${s.summary!.trim()}`
    })

  if (blocks.length === 0) return ""
  return (
    "## Previous session context (from Mnemonic)\n\n" +
    blocks.join("\n\n---\n\n")
  )
}

function codeStatusNudge(status: {
  stale?: boolean
  file_count?: number
  chunk_count?: number
  last_indexed?: string
}): string {
  if (!status.stale) return ""

  const indexed = status.last_indexed ? ` Last indexed: ${status.last_indexed}.` : ""
  const counts =
    status.file_count != null && status.chunk_count != null
      ? ` Index has ${status.file_count} files / ${status.chunk_count} chunks.`
      : ""

  return (
    "## Code index stale\n\n" +
    `Mnemonic code search may be outdated.${counts}${indexed} ` +
    "Run `skillgrid index` in the project root before relying on `code_search`, " +
    "or call `code_status` via MCP to confirm freshness."
  )
}

// ─── Plugin export ───────────────────────────────────────────────────────────

export const Mnemonic: Plugin = async (ctx) => {
  let sessionId = ""
  let projectId = ""
  let memoryProtocol = ""
  let codeNudge = ""

  await ensureServer(getBaseUrl())
  memoryProtocol = await loadMemoryProtocol()

  const session = await mnemonicFetch<{ session_id?: string; project?: string }>(
    "/sessions",
    {
      method: "POST",
      body: { directory: ctx.directory },
    },
  )
  if (session?.session_id) sessionId = session.session_id
  if (session?.project) projectId = session.project

  if (projectId) {
    const status = await mnemonicFetch<{
      stale?: boolean
      file_count?: number
      chunk_count?: number
      last_indexed?: string
    }>(`/code/status?project=${encodeURIComponent(projectId)}`)
    if (status) codeNudge = codeStatusNudge(status)
  }

  return {
    event: async ({ event }) => {
      if (event.type !== "session.created") return

      const info = (event.properties as { info?: { id?: string; parentID?: string } })?.info
      if (info?.parentID) return

      if (!sessionId) {
        const created = await mnemonicFetch<{ session_id?: string; project?: string }>(
          "/sessions",
          {
            method: "POST",
            body: { directory: ctx.directory },
          },
        )
        if (created?.session_id) sessionId = created.session_id
        if (created?.project) projectId = created.project
      }

      if (projectId && !codeNudge) {
        const status = await mnemonicFetch<{
          stale?: boolean
          file_count?: number
          chunk_count?: number
          last_indexed?: string
        }>(`/code/status?project=${encodeURIComponent(projectId)}`)
        if (status) codeNudge = codeStatusNudge(status)
      }
    },

    "experimental.chat.system.transform": async (_input, output) => {
      appendToSystem(output.system, memoryProtocol)
      appendToSystem(output.system, codeNudge)
    },

    "experimental.session.compacting": async (_input, output) => {
      if (projectId) {
        const data = await mnemonicFetch<{
          sessions?: Array<{ summary?: string; started_at?: string; id?: string }>
        }>(
          `/context?project=${encodeURIComponent(projectId)}&limit=5`,
          { timeoutMs: 2000 },
        )
        const contextText = formatContextSessions(data?.sessions ?? [])
        if (contextText) {
          output.context.push(stripPrivateTags(contextText))
        }
      }

      output.context.push(COMPACTION_REMINDER)
    },
  }
}
