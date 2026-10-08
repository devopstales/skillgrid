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
//
// `node tool-call-capture.js register` reads the same stdin shape and
// registers the harness session (POST /sessions?project=…) instead of a call;
// `usage` reports the session's token totals (POST /sessions/{id}/usage);
// `policy` is the pre-tool check (POST /policy/evaluate) and is the one mode
// that may block: Cursor gets {"permission":"deny"}, OpenCode/Kilo exit 2.

const BASE = (process.env.SKILLGRID_MNEMONIC_HTTP_URL || "http://127.0.0.1:7438").replace(/\/+$/, "")

function checkpointBase() {
  return (process.env.SKILLGRID_CHECKPOINT_URL || BASE).replace(/\/+$/, "")
}

function policyTimeoutMs() {
  return Number(process.env.SKILLGRID_POLICY_TIMEOUT_MS) || 1500
}
const AGENT = process.env.SKILLGRID_AGENT || "opencode"
const MAX_PREVIEW = 500
const CTX_BYPASS = process.env.SKILLGRID_CTX_BYPASS === "1"
const CTX_THRESHOLD = 4096

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
    case "multiedit":
    case "strreplace":
    case "patch":
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
  let s = str
  let prev
  do {
    prev = s
    s = s.replace(/<private>[\s\S]*?(<\/private>|$)/gi, "")
  } while (s !== prev)
  return s.replace(/[ \t]{2,}/g, " ").trim()
}

function truncate(str, max) {
  if (!str) return ""
  return str.length > max ? str.slice(0, max) + "\n[truncated]" : str
}

async function readPayload() {
  const chunks = []
  if (!process.stdin.isTTY) {
    for await (const chunk of process.stdin) chunks.push(chunk)
  }
  try {
    return JSON.parse(Buffer.concat(chunks).toString() || "{}")
  } catch {
    return {}
  }
}

function sessionIdOf(payload) {
  return (
    payload.session_id ||
    payload.conversation_id ||
    process.env.OPENCODE_SESSION_ID ||
    process.env.SKILLGRID_CURSOR_SESSION_ID ||
    ""
  )
}

function directoryOf(payload) {
  return (
    payload.cwd ||
    (Array.isArray(payload.workspace_roots) && payload.workspace_roots[0]) ||
    process.env.OPENCODE_PROJECT_DIR ||
    process.cwd()
  )
}

async function post(url, body) {
  try {
    await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      signal: AbortSignal.timeout(3000),
    })
  } catch {}
}

// register: create (or tag) the harness session row so the Sessions view
// shows it with its agent before the first tool call lands.
async function register() {
  const payload = await readPayload()
  const sessionId = sessionIdOf(payload)
  if (!sessionId) return
  const directory = directoryOf(payload)
  await post(`${BASE}/sessions?directory=${encodeURIComponent(directory)}`, {
    id: sessionId,
    agent: AGENT,
    directory,
    title: AGENT,
  })
}

// openCodeStorageDirs lists where OpenCode-family harnesses keep per-message
// JSON (storage/message/<session>/*.json). SKILLGRID_USAGE_STORAGE overrides.
function openCodeStorageDirs() {
  const path = require("path")
  const os = require("os")
  if (process.env.SKILLGRID_USAGE_STORAGE) return [process.env.SKILLGRID_USAGE_STORAGE]
  const data = process.env.XDG_DATA_HOME || path.join(os.homedir(), ".local", "share")
  const names = AGENT === "kilo" ? ["kilo", "kilocode", "opencode"] : ["opencode"]
  return names.map((n) => path.join(data, n, "storage"))
}

// sumOpenCodeUsage totals every assistant message of the session. Re-reading
// the whole transcript on each idle makes the report a replaceable total.
function sumOpenCodeUsage(sessionId) {
  const fs = require("fs")
  const path = require("path")
  for (const root of openCodeStorageDirs()) {
    const dir = path.join(root, "message", sessionId)
    let files
    try {
      files = fs.readdirSync(dir).filter((f) => f.endsWith(".json"))
    } catch {
      continue
    }
    const u = { model: "", input: 0, output: 0, cache: 0, cost: 0, created: 0 }
    for (const f of files) {
      let m
      try {
        m = JSON.parse(fs.readFileSync(path.join(dir, f), "utf8"))
      } catch {
        continue
      }
      if (m.role !== "assistant" || !m.tokens) continue
      const t = m.tokens
      u.input += Number(t.input) || 0
      u.output += (Number(t.output) || 0) + (Number(t.reasoning) || 0)
      u.cache += (Number(t.cache?.read) || 0) + (Number(t.cache?.write) || 0)
      u.cost += Number(m.cost) || 0
      const created = Number(m.time?.created) || 0
      if (m.modelID && created >= u.created) {
        u.model = m.modelID
        u.created = created
      }
    }
    return u
  }
  return null
}

// usage: report token usage for the session. OpenCode and Kilo totals come
// from the harness's own message store; Cursor hooks carry the model only,
// so Cursor sessions get a model and no invented token count.
async function usage() {
  const payload = await readPayload()
  const sessionId = sessionIdOf(payload)
  if (!sessionId) return
  const directory = directoryOf(payload)
  let body
  if (AGENT === "cursor") {
    const model = payload.model_id || payload.model || ""
    if (!model) return
    body = { model, agent: AGENT, directory }
  } else {
    const u = sumOpenCodeUsage(sessionId)
    if (!u) return
    body = {
      model: u.model,
      input: u.input,
      output: u.output,
      cache: u.cache,
      total: true,
      agent: AGENT,
      directory,
      reported_cost: u.cost > 0 ? u.cost : undefined,
    }
  }
  await post(`${BASE}/sessions/${encodeURIComponent(sessionId)}/usage?directory=${encodeURIComponent(directory)}`, body)
}

// hasPolicyFile mirrors policy.Load's lookup (repo file walking up from dir,
// then ~/.skillgrid/policy.yaml) so calls skip the HTTP round-trip when no
// policy exists.
function hasPolicyFile(dir) {
  const fs = require("fs")
  const path = require("path")
  const os = require("os")
  let d = path.resolve(dir || ".")
  for (;;) {
    if (fs.existsSync(path.join(d, ".skillgrid", "policy.yaml"))) return true
    const parent = path.dirname(d)
    if (parent === d) break
    d = parent
  }
  return fs.existsSync(path.join(os.homedir(), ".skillgrid", "policy.yaml"))
}

// policyRequest maps a pre-tool payload to /policy/evaluate fields.
// Cursor: beforeShellExecution {command}, beforeReadFile {file_path},
// beforeMCPExecution {tool_name, tool_input}. OpenCode/Kilo: tool.before.*
// {tool_name, tool_args}.
function policyRequest(payload) {
  const event = String(payload.hook_event_name || "")
  if (event === "beforeShellExecution" || (!payload.tool_name && payload.command)) {
    return { action: "command_exec", tool: "Shell", command: String(payload.command || "") }
  }
  if (event === "beforeReadFile" || (!payload.tool_name && payload.file_path)) {
    return { action: "file_read", tool: "Read", path: String(payload.file_path || "") }
  }
  const tool = String(payload.tool_name || "")
  let args = payload.tool_args ?? payload.tool_input ?? {}
  if (typeof args === "string") {
    try {
      args = JSON.parse(args)
    } catch {
      args = {}
    }
  }
  const req = {
    tool,
    path: String(args.filePath ?? args.file_path ?? args.path ?? args.file ?? args.target ?? ""),
    command: String(args.command ?? ""),
  }
  if (event === "beforeMCPExecution") req.action = "tool_use"
  return req
}

// evaluatePolicy answers {effect, message, rule}. Fail-open (ADR-0021): no
// policy file, no server, a slow server, or any error means allow.
async function evaluatePolicy(payload) {
  const allow = { effect: "allow", message: "", rule: "" }
  const directory = directoryOf(payload)
  if (!hasPolicyFile(directory)) return allow
  const timeout = policyTimeoutMs()
  try {
    const res = await fetch(`${BASE}/policy/evaluate?directory=${encodeURIComponent(directory)}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ session_id: sessionIdOf(payload), agent: AGENT, directory, ...policyRequest(payload) }),
      signal: AbortSignal.timeout(timeout),
    })
    if (!res.ok) return allow
    const d = await res.json()
    if (!d || !["block", "warn", "guide", "allow"].includes(d.effect)) return allow
    return { effect: d.effect, message: String(d.message || ""), rule: String(d.rule || "") }
  } catch {
    return allow
  }
}

// policy: the pre-tool hook. Cursor gets {"permission": ...} on stdout;
// OpenCode/Kilo get exit 2 with the reason on stderr to block.
async function policyHook() {
  const payload = await readPayload()
  const d = await evaluatePolicy(payload)
  const note = d.message ? `[skillgrid policy${d.rule ? ": " + d.rule : ""}] ${d.message}` : ""
  if (AGENT === "cursor") {
    const out = { permission: d.effect === "block" ? "deny" : "allow" }
    if (note) {
      out.agent_message = note
      if (d.effect === "block") out.user_message = note
    }
    process.stdout.write(JSON.stringify(out) + "\n")
    process.exit(0)
  }
  if (d.effect === "block") {
    process.stderr.write(note + "\n")
    process.exit(2)
  }
  if (note) process.stderr.write(note + "\n")
  process.exit(0)
}

// checkpoint: Cursor stop hook claims a memory checkpoint; may emit followup_message.
async function checkpoint() {
  const empty = () => {
    process.stdout.write("{}\n")
    process.exit(0)
  }
  const payload = await readPayload()
  const loopCount = Number(payload.loop_count) || 0
  if (loopCount >= 2) return empty()
  const sessionId = sessionIdOf(payload)
  if (!sessionId) return empty()
  const directory = directoryOf(payload)
  const base = checkpointBase()
  try {
    const url = `${base}/sessions/${encodeURIComponent(sessionId)}/checkpoint/claim?directory=${encodeURIComponent(directory)}`
    const res = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ directory }),
      signal: AbortSignal.timeout(policyTimeoutMs()),
    })
    if (!res.ok) return empty()
    const d = await res.json()
    if (d && d.due === true && typeof d.prompt === "string" && d.prompt) {
      process.stdout.write(JSON.stringify({ followup_message: d.prompt }) + "\n")
      process.exit(0)
    }
  } catch {}
  return empty()
}

async function main() {
  if (process.argv[2] === "policy") {
    await policyHook()
    return
  }
  if (process.argv[2] === "checkpoint") {
    await checkpoint()
    return
  }
  if (process.argv[2] === "register") {
    await register()
    process.exit(0)
  }
  if (process.argv[2] === "usage") {
    await usage()
    process.exit(0)
  }
  const payload = await readPayload()

  const sessionId = sessionIdOf(payload)
  const tool = String(payload.tool_name ?? "")
  const args = payload.tool_args ?? payload.tool_input ?? {}
  if (!sessionId || !tool) process.exit(0)

  const directory = directoryOf(payload)
  const path = String(args.filePath ?? args.file_path ?? args.path ?? args.file ?? args.target ?? "")
  const command = String(args.command ?? "")
  const rawOut = privateTools().has(tool.toLowerCase()) ? "" : String(payload.tool_output ?? "")
  const resultText = truncate(rawOut, MAX_PREVIEW)
  const content = stripPrivateTags([path, command, resultText].join("\n"))

  const body = {
    session_id: sessionId,
    agent: AGENT,
    directory,
    type: mapToolType(tool),
    tool_name: tool,
    path: sensitivePath(path) ? "[REDACTED]" : stripPrivateTags(path),
    command: sensitivePath(command) ? "[REDACTED]" : command,
    result_status: "success",
    content_hash: content === "" ? "" : contentHash(content),
    content_preview: truncate(content, MAX_PREVIEW),
    content: (!CTX_BYPASS && rawOut.length > CTX_THRESHOLD) ? stripPrivateTags(rawOut) : "",
  }

  await post(`${BASE}/sessions/${encodeURIComponent(sessionId)}/tool-calls?directory=${encodeURIComponent(directory)}`, body)
  process.exit(0)
}

main().catch(() => {
  if (process.argv[2] === "policy" && AGENT === "cursor") process.stdout.write('{"permission":"allow"}\n')
  process.exit(0)
})
