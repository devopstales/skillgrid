// @ts-nocheck
// Mnemonic compaction recovery for the Pi extension (ADR-0043): refreshes the
// session's title/summary + newest observations before Pi summarizes, so the
// compaction prompt sees durable context. Fail-open: a dead server never
// blocks the compaction itself.

import { api, sessionIDOf } from "./api.js"

export async function buildCompactionContext(ctx) {
  const sid = sessionIDOf(ctx)
  if (!sid) return ""
  try {
    const res = await api("GET", `/context/compaction?session_id=${encodeURIComponent(sid)}`, undefined, 3000)
    const ctxOut = res?.context || res || {}
    const parts = []
    if (ctxOut.title) parts.push("## Session\n" + ctxOut.title)
    if (ctxOut.summary) parts.push(ctxOut.summary)
    const observations = Array.isArray(ctxOut.observations) ? ctxOut.observations.slice(0, 5) : []
    if (observations.length > 0) {
      parts.push(
        "## Recent Observations\n" +
          observations
            .map((o) => `- [${o.type || "?"}] ${o.title || `#${o.id}`}`)
            .join("\n"),
      )
    }
    return parts.join("\n\n").slice(0, 4000)
  } catch {
    return "" // fail-open: compaction proceeds without the block
  }
}
