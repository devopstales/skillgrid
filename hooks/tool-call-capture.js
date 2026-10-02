// tool-call-capture.js — fire-and-forget POST of tool-call summaries to the
// Mnemonic HTTP API. Shared by OpenCode/Kilo yaml-hooks and Cursor
// hooks/cursor-tool-capture.sh (SKILLGRID_AGENT=cursor).
//
// Stdin JSON (either shape):
//   OpenCode: { session_id, cwd, tool_name, tool_args, ... }
//   Cursor:   { conversation_id|session_id, cwd, tool_name, tool_input, tool_output, ... }
//
// Fail-open: never throws, always exits 0.
// Agent label: SKILLGRID_AGENT env (default "opencode").

const BASE = (process.env.SKILLGRID_MNEMONIC_HTTP_URL || "http://127.0.0.1:7438").replace(/\/+$/, "")
const AGENT = process.env.SKILLGRID_AGENT || "opencode"
const MAX_PREVIEW = 500

function contentHash(str) {
  let h = 0
  for (let i = 0; i < str.length; i++) {
    h = (h << 5) - h + str.charCodeAt(i)
    h |= 0
  }
  return "h" + (h >>> 0).toString(16).padStart(8, "0")
}

function mapToolType(tool) {
  switch (String(tool ?? "").toLowerCase()) {
    case "bash":
    case "shell":
      return "shell"
    case "read":
    case "glob":
    case "grep":
      return "file_read"
    case "write":
    case "edit":
    case "delete":
      return "file_write"
    default:
      return "other"
  }
}

function privateTools() {
  return new Set(
    String(process.env.SKILLGRID_MNEMONIC_PRIVATE_TOOLS ?? "")
      .split(",")
      .map((s) => s.trim().toLowerCase())
      .filter(Boolean),
  )
}

function sensitivePath(p) {
  return /(^|\/)\.(git|ssh|aws|config)(\/|$)/i.test(p)
}

function stripPrivateTags(str) {
  if (!str) return ""
  return str.replace(/<private>[\s\S]*?<\/private>/gi, "[REDACTED]").trim()
}

function truncate(str, max) {
  if (!str) return ""
  return str.length > max ? str.slice(0, max) + "…" : str
}

function resolveProject(directory) {
  try {
    const remote = require("child_process")
      .execSync(`git -C ${JSON.stringify(directory)} remote get-url origin`, {
        stdio: ["ignore", "pipe", "ignore"],
        timeout: 2000,
      })
      .toString()
      .trim()
    if (remote) {
      const name = remote
        .replace(/\.git$/, "")
        .split(/[/:]/)
        .pop()
      if (name) return name.toLowerCase().replace(/[-_]+/g, "-").replace(/^-|-$/g, "")
    }
  } catch {}
  const base = (directory || "").replace(/\/+$/, "").split("/").pop()
  return (base || "unknown").toLowerCase()
}

async function main() {
  const chunks = []
  for await (const chunk of process.stdin) chunks.push(chunk)
  let payload
  try {
    payload = JSON.parse(Buffer.concat(chunks).toString() || "{}")
  } catch {
    process.exit(0)
  }

  const sessionId =
    payload.session_id ||
    payload.conversation_id ||
    process.env.OPENCODE_SESSION_ID ||
    process.env.SKILLGRID_CURSOR_SESSION_ID ||
    ""
  const tool = String(payload.tool_name ?? "")
  const args = payload.tool_args ?? payload.tool_input ?? {}
  if (!sessionId || !tool) process.exit(0)

  const directory =
    payload.cwd ||
    (Array.isArray(payload.workspace_roots) && payload.workspace_roots[0]) ||
    process.cwd()
  const path = String(args.filePath ?? args.path ?? args.file ?? args.target ?? "")
  const command = String(args.command ?? "")
  const resultText = privateTools().has(tool.toLowerCase())
    ? ""
    : String(payload.tool_output ?? "").slice(0, MAX_PREVIEW)
  const content = stripPrivateTags([path, command, resultText].join("\n"))

  const body = {
    session_id: sessionId,
    agent: AGENT,
    type: mapToolType(tool),
    tool_name: tool,
    path: sensitivePath(path) ? "[REDACTED]" : path,
    command: sensitivePath(command) ? "[REDACTED]" : command,
    result_status: "success",
    content_hash: content === "" ? "" : contentHash(content),
    content_preview: truncate(content, MAX_PREVIEW),
  }

  try {
    await fetch(
      `${BASE}/sessions/${encodeURIComponent(sessionId)}/tool-calls?project=${encodeURIComponent(resolveProject(directory))}`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
        signal: AbortSignal.timeout(3000),
      },
    )
  } catch {}
  process.exit(0)
}

main().catch(() => process.exit(0))
