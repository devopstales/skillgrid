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

export const SkillgridCheckpoint = async ({ client, directory }) => {
  /** Skip one idle after prompting so the harness does not re-claim in a loop. */
  const skipClaimAfterPrompt = new Set()

  return {
    event: async ({ event }) => {
      try {
        if (event?.type !== "session.idle") return
        const sessionID =
          event?.properties?.sessionID ?? event?.properties?.sessionId ?? event?.properties?.session_id
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
  }
}
