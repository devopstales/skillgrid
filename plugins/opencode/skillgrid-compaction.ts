// @ts-nocheck
import * as fs from "node:fs"
import * as path from "node:path"
import * as os from "node:os"

const BASE_URL =
  process.env.SKILLGRID_CHECKPOINT_URL ||
  process.env.SKILLGRID_MNEMONIC_HTTP_URL ||
  "http://127.0.0.1:7438"

const AGENT = process.env.SKILLGRID_AGENT || "opencode"

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

async function getJSON(pathname) {
  try {
    const res = await fetch(BASE_URL + pathname, {
      signal: AbortSignal.timeout(3000),
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return await res.json()
  } catch {
    return null
  }
}

function safeParse(raw) {
  if (typeof raw !== "string") return null
  try {
    return JSON.parse(raw)
  } catch {
    return null
  }
}

function distillSummary(rawText) {
  if (!rawText) return ""
  const lines = rawText
    .split("\n")
    .map((l) => l.replace(/^#+\s*/, "").trim())
    .filter(Boolean)
  const result = []
  let inSummary = false
  for (const line of lines) {
    if (/^##?\s+(Goal|Summary|Objective|What)/i.test(line)) {
      inSummary = true
      continue
    }
    if (/^##?\s+(Next|TODO|Relevant|Files|Notes|Discovery)/i.test(line)) {
      inSummary = false
      continue
    }
    if (inSummary && line) result.push(line)
  }
  if (result.length === 0 && lines.length > 0) {
    result.push(lines.slice(0, 10).join(" ").slice(0, 500))
  }
  return result.join("\n").slice(0, 4000)
}

function estimateTokens(text) {
  if (!text) return 0
  return Math.ceil(text.length / 4)
}

function primeText(
  summary,
  facts,
  observations,
  skills,
  maxTokens = 800
) {
  const sections = []
  if (summary) {
    sections.push("## Session Summary\n" + summary)
  }
  if (facts.length > 0) {
    sections.push("## Key Facts\n" + facts.map((f) => "- " + f.content).join("\n"))
  }
  if (observations.length > 0) {
    sections.push(
      "## Recent Observations\n" +
        observations.map((o) => `- [${o.type}] ${o.title}`).join("\n")
    )
  }
  if (skills.length > 0) {
    sections.push(
      "## Active Skills\n" + skills.map((s) => `- ${s.name}`).join("\n")
    )
  }
  let text = sections.join("\n\n")
  const maxChars = maxTokens * 4
  if (text.length > maxChars) text = text.slice(0, maxChars) + "\n…"
  return text
}

async function buildContext(sessionID) {
  const ctxData = await getJSON("/context?session_id=" + sessionID)
  if (!ctxData) return null

  const summary =
    ctxData.summary ||
    distillSummary(
      ctxData.summaryText ||
        (ctxData.lastSummary ? safeParse(ctxData.lastSummary)?.summary || "" : "")
    )

  const facts = (ctxData.facts || []).slice(0, 5)
  const observations = (ctxData.observations || []).slice(0, 3)
  const skills = (ctxData.skills || []).slice(0, 5)

  const text = primeText(summary, facts, observations, skills)
  if (!text) return null

  return {
    text,
    session_id: sessionID,
    tokens: estimateTokens(text),
  }
}

async function handleSessionCreated(sessionID) {
  const ctx = await buildContext(sessionID)
  if (ctx) {
    const ctxFile = path.join(
      os.tmpdir(),
      "skillgrid-context-" + sessionID + ".txt"
    )
    fs.writeFileSync(ctxFile, ctx.text)
  }
  await postJSON("/sessions/" + sessionID + "/start", { agent: AGENT })
}

async function handleSessionIdle(sessionID) {
  await postJSON("/sessions/" + sessionID + "/end", { agent: AGENT })
}

async function handleSessionCompacted(sessionID) {
  const ctx = await buildContext(sessionID)
  await postJSON("/memory/context/prime", {
    session_id: sessionID,
    context: ctx || {},
    agent: AGENT,
  })
}

export const SkillgridCompaction = async () => ({
  event: async ({ event }) => {
    const sessionID = event.properties?.sessionID
    if (!sessionID) return

    switch (event.type) {
      case "session.created":
        await handleSessionCreated(sessionID)
        break
      case "session.idle":
        await handleSessionIdle(sessionID)
        break
      case "session.compacted":
        await handleSessionCompacted(sessionID)
        break
      case "experimental.session.compacting": {
        const ctx = await buildContext(sessionID)
        if (ctx) {
          const ctxFile = path.join(
            os.tmpdir(),
            "skillgrid-precompact-" + sessionID + ".txt"
          )
          fs.writeFileSync(ctxFile, ctx.text)
        }
        break
      }
    }
  },
})
