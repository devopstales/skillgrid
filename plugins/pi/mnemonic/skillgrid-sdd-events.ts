// @ts-nocheck
// SDD event capture for the Pi extension: work-unit commit events and the
// gate-stop state are POSTed to the session's tool-calls route so the Go
// server sees them on the session event stream (the pi counterpart of the
// opencode/kilo hooks/tool-call-capture.js + git-hook paths). Every path is
// fail-open — a dead server, a missing file, or a non-git cwd never blocks
// a tool result or shutdown.

import { spawnSync } from "node:child_process"
import { readFileSync } from "node:fs"
import { join } from "node:path"
import { AGENT, bestEffort, sessionIDOf } from "./api.js"

const COMMIT_RE = /(^|\s|&&|\|\||;)\s*git\s+.*\bcommit\b/
const CONTEXT_MARKER = "[skillgrid-context]"

function toolCommand(input) {
  const c = input?.command ?? input?.cmd
  return typeof c === "string" ? c : ""
}

function gitLogMessage(cwd) {
  const r = spawnSync("git", ["log", "-1", "--format=%B"], {
    cwd: cwd || process.cwd(),
    encoding: "utf8",
    timeout: 5000,
    stdio: ["ignore", "pipe", "pipe"],
  })
  if (r.status !== 0) return ""
  return r.stdout || ""
}

// skillgridContextBlock returns the [skillgrid-context] block of a commit
// message, or "" when absent.
function skillgridContextBlock(message) {
  const i = message.indexOf(CONTEXT_MARKER)
  if (i < 0) return ""
  return message.slice(i).trim()
}

function toolCallPath(sid, directory) {
  return `/sessions/${encodeURIComponent(sid)}/tool-calls?directory=${encodeURIComponent(directory)}`
}

function commitSummary(block) {
  const lines = {}
  for (const line of block.split("\n")) {
    for (const key of ["change:", "tier:", "remaining:"]) {
      if (line.trim().toLowerCase().startsWith(key)) {
        lines[key] = line.trim().slice(key.length).trim()
      }
    }
  }
  if (lines["change:"]) {
    return `Change: ${lines["change:"]}${lines["tier:"] ? " / Tier: " + lines["tier:"] : ""}${
      lines["remaining:"] ? " / Remaining: " + lines["remaining:"] : ""
    }`
  }
  return block
}

async function captureCommit(ctx, input) {
  const sid = sessionIDOf(ctx)
  if (!sid) return
  const cwd = ctx?.cwd || process.cwd()
  const block = skillgridContextBlock(gitLogMessage(cwd))
  if (!block) return
  await bestEffort(toolCallPath(sid, cwd), {
    session_id: sid,
    agent: AGENT,
    directory: cwd,
    type: "commit",
    tool_name: "git",
    path: "",
    result_status: "success",
    content_preview: commitSummary(block),
  })
}

async function captureGateStop(ctx) {
  const sid = sessionIDOf(ctx)
  if (!sid) return
  const cwd = ctx?.cwd || process.cwd()
  let content
  try {
    content = readFileSync(join(cwd, ".skillgrid", "sdd", "gate-stop-state.json"), "utf8")
  } catch {
    return
  }
  await bestEffort(toolCallPath(sid, cwd), {
    session_id: sid,
    agent: AGENT,
    directory: cwd,
    type: "gate_stop",
    tool_name: "gate-stop",
    path: "",
    result_status: "success",
    content: content,
  })
}

export function registerSddEventCapture(pi) {
  pi.on("tool_result", async (event, ctx) => {
    try {
      if (String(event?.toolName || "") !== "bash") return
      if (!COMMIT_RE.test(toolCommand(event?.input))) return
      await captureCommit(ctx, event?.input)
    } catch {
      // fail-open
    }
  })

  pi.on("session_shutdown", async (event, ctx) => {
    try {
      if (event?.reason !== "quit") return
      await captureGateStop(ctx)
    } catch {
      // fail-open
    }
  })
}
