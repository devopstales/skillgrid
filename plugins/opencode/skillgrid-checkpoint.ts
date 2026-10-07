// @ts-nocheck
// ponytail: Plugin/client types are inlined — no @opencode-ai/plugin import in repo.

function checkpointBase() {
  return (
    process.env.SKILLGRID_CHECKPOINT_URL ||
    process.env.SKILLGRID_MNEMONIC_HTTP_URL ||
    "http://127.0.0.1:7438"
  ).replace(/\/+$/, "")
}

function claimTimeoutMs() {
  return Number(process.env.SKILLGRID_POLICY_TIMEOUT_MS) || 1500
}

function sessionIDOf(event) {
  const p = event?.properties ?? {}
  return p.sessionID ?? p.sessionId ?? p.session_id ?? ""
}

function compactContextBlock(cc) {
  const lines = []
  if (cc?.title) lines.push(`## Session: ${cc.title}`)
  if (cc?.summary) lines.push(cc.summary)
  if (cc?.observations?.length) {
    lines.push("## Recent observations")
    for (const o of cc.observations) lines.push(`- ${o}`)
  }
  if (lines.length === 0) return ""
  return lines.join("\n")
}

export const SkillgridCheckpoint = async ({ client, directory }) => {
  /** Skip one idle after prompting so the harness does not re-claim in a loop. */
  const skipClaimAfterPrompt = new Set()

  return {
    event: async ({ event }) => {
      try {
        if (event?.type !== "session.idle") return
        const sessionID = sessionIDOf(event)
        if (!sessionID) return

        if (skipClaimAfterPrompt.has(sessionID)) {
          skipClaimAfterPrompt.delete(sessionID)
          return
        }

        const dir = directory || process.cwd?.() || "."
        const base = checkpointBase()
        const url = `${base}/sessions/${encodeURIComponent(sessionID)}/checkpoint/claim?directory=${encodeURIComponent(dir)}`
        const res = await fetch(url, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ directory: dir }),
          signal: AbortSignal.timeout(claimTimeoutMs()),
        })
        if (!res.ok) return
        const body = await res.json()
        if (body?.due !== true || typeof body.prompt !== "string" || !body.prompt) return

        skipClaimAfterPrompt.add(sessionID)
        await client.session.prompt({
          path: { id: sessionID },
          body: { parts: [{ type: "text", text: body.prompt }] },
        })
      } catch {
        // fail-open: never block the harness on checkpoint errors
      }
    },

    "experimental.session.compacting": async (input, output) => {
      try {
        // ponytail: input.sessionID not guaranteed by the docs; fall back to
        // input.id when the harness names the session that way.
        const sessionID = input?.sessionID ?? input?.id ?? ""
        if (!sessionID) return
        const dir = directory || process.cwd?.() || "."
        const base = checkpointBase()
        const url = `${base}/context/compaction?session_id=${encodeURIComponent(sessionID)}&directory=${encodeURIComponent(dir)}`
        const res = await fetch(url, {
          signal: AbortSignal.timeout(claimTimeoutMs()),
        })
        if (!res.ok) return
        const body = await res.json()
        const block = compactContextBlock(body?.context)
        if (!block) return
        if (Array.isArray(output?.context)) output.context.push(block)
      } catch {
        // fail-open: never block the harness on compaction-context errors
      }
    },
  }
}
