// @ts-nocheck
const BASE_URL =
  process.env.SKILLGRID_CHECKPOINT_URL ||
  process.env.SKILLGRID_MNEMONIC_HTTP_URL ||
  "http://127.0.0.1:7438"

const AGENT = process.env.SKILLGRID_AGENT || "opencode"

// ── Failure latch (after sdd-task-result-artifacts.ts) ──────────────────────
// Squad tasks move through a linear pipeline (spawn → pull → output → review
// → done). If a phase fails, no later phase may silently advance on top of the
// broken state. We latch the failing task and refuse further transitions for
// it, emitting a machine-readable handoff the agent surfaces instead of
// retrying.
//
// Caveat: this is an IN-MEMORY latch, scoped to the plugin's lifetime. The
// authoritative state machine lives in the Go server (no terminal
// failed/blocked status today), so a restarted agent loses the latch. The
// durable version belongs server-side (a task status that /output and /review
// refuse to advance from); this client guard is the minimal version that
// still stops the silent cascade within a session.
const SQUAD_FAILURE_PREFIX = "SKILLGRID_SQUAD_FAILURE "
const failedTasks = new Map() // task_id -> { code, summary, at }

function isLatched(taskId) {
  return Boolean(taskId && failedTasks.has(taskId))
}

function latch(taskId, code, summary) {
  failedTasks.set(taskId, {
    code,
    summary,
    at: new Date().toISOString(),
  })
}

function handoff(taskId, code, summary) {
  return (
    SQUAD_FAILURE_PREFIX +
    JSON.stringify({
      schemaName: "skillgrid.squad-task-failure/v1",
      status: "blocked",
      code,
      task_id: taskId,
      summary,
      guidance:
        "Do not retry or advance this task in the current session; inspect the " +
        "task state with squad_read_task and surface the failure to the user. " +
        "Start a new session to retry.",
    })
  )
}

function throwLatched(taskId, action) {
  const f = failedTasks.get(taskId)
  throw new Error(
    handoff(
      taskId,
      "squad_task_latched",
      `${action} was not dispatched: ${taskId} is latched (${f.code}) from an earlier failure.`
    )
  )
}

function assertUnlatched(taskId, action) {
  if (isLatched(taskId)) throwLatched(taskId, action)
}

// Classify a thrown squad error into a stable code for the latch + handoff.
function errorCode(err) {
  const msg = String((err && err.message) || "")
  if (/^HTTP 4\d\d/.test(msg)) return "squad_task_client_error"
  if (/^HTTP 5\d\d/.test(msg)) return "squad_task_server_error"
  return "squad_task_failed"
}

// Run a squad mutation; on failure latch the task (unless it is a benign
// "no task" pull) and re-throw the prefixed handoff so the agent sees a
// structured terminal signal, not a bare HTTP status.
async function runMutation(taskId, action, fn, opts = {}) {
  try {
    return await fn()
  } catch (err) {
    if (taskId && !opts.ignoreFailure) {
      latch(taskId, errorCode(err), String((err && err.message) || err))
      throw new Error(handoff(taskId, errorCode(err), String((err && err.message) || err)))
    }
    throw err
  }
}

async function api(method, path, body) {
  const res = await fetch(BASE_URL + path, {
    method,
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
    signal: AbortSignal.timeout(5000),
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(data.error || `HTTP ${res.status}`)
  }
  return data
}

const squadSpawnTask = {
  description:
    "Spawn a pending team task with a markdown brief. Returns the new task_id.",
  args: {
    title: tool.schema.string().describe("Short task title"),
    brief: tool.schema.string().describe("Markdown brief content"),
    team_id: tool
      .schema.string()
      .optional()
      .describe("Team id (defaults to default)"),
    priority: tool
      .schema.number()
      .optional()
      .describe("Priority; higher claims first (default 0)"),
    created_by: tool
      .schema.string()
      .optional()
      .describe("Optional creator id"),
  },
  async execute(args) {
    const data = await api("POST", "/teams/tasks", {
      ...args,
      created_by: args.created_by || AGENT,
    })
    return JSON.stringify(data)
  },
}

const squadPullNextTask = {
  description:
    "Claim the highest-priority pending team task for a member. Returns the task or a 'no task' message.",
  args: {
    member_id: tool.schema.string().describe("Team member / agent id"),
    team_id: tool
      .schema.string()
      .optional()
      .describe("Team id (defaults to default)"),
  },
  async execute(args) {
    const data = await api("POST", "/teams/tasks/pull", args)
    return JSON.stringify(data)
  },
}

const squadSubmitOutput = {
  description:
    "Write task output.md and advance status to review_spec. Returns the updated task. Latches the task on failure.",
  args: {
    task_id: tool.schema.string().describe("Task id"),
    output: tool.schema.string().describe("Markdown output content"),
    summary: tool
      .schema.string()
      .optional()
      .describe("Optional one-line summary"),
  },
  async execute(args) {
    assertUnlatched(args.task_id, "squad_submit_output")
    const data = await runMutation(
      args.task_id,
      "squad_submit_output",
      () => api("POST", `/teams/tasks/${args.task_id}/output`, args)
    )
    return JSON.stringify(data)
  },
}

const squadSubmitReview = {
  description:
    "Submit a peer review (spec_compliance or code_quality) with markdown comments. Latches the task on failure.",
  args: {
    task_id: tool.schema.string().describe("Task id"),
    reviewer_id: tool.schema.string().describe("Reviewer member id"),
    passed: tool.schema.boolean().describe("Whether the review passed"),
    comments: tool
      .schema.string()
      .optional()
      .describe("Markdown review comments"),
    review_type: tool
      .schema.string()
      .optional()
      .describe("spec_compliance (default) or code_quality"),
  },
  async execute(args) {
    assertUnlatched(args.task_id, "squad_submit_review")
    const data = await runMutation(
      args.task_id,
      "squad_submit_review",
      () =>
        api("POST", `/teams/tasks/${args.task_id}/reviews`, args)
    )
    return JSON.stringify(data)
  },
}

const squadReadTask = {
  description:
    "Read task metadata and brief from disk. Read-only: never latches the task.",
  args: {
    task_id: tool.schema.string().describe("Task id"),
  },
  async execute(args) {
    const data = await api("GET", `/teams/tasks/${args.task_id}`)
    return JSON.stringify(data)
  },
}

const squadMarkDone = {
  description:
    "Mark a team task as done. Latches the task on failure.",
  args: {
    task_id: tool.schema.string().describe("Task id"),
  },
  async execute(args) {
    assertUnlatched(args.task_id, "squad_mark_done")
    const data = await runMutation(
      args.task_id,
      "squad_mark_done",
      () => api("POST", `/teams/tasks/${args.task_id}/done`, { agent: AGENT })
    )
    return JSON.stringify(data)
  },
}

export const SkillgridSquad = async () => ({
  tool: {
    squad_spawn_task: tool(squadSpawnTask),
    squad_pull_next_task: tool(squadPullNextTask),
    squad_submit_output: tool(squadSubmitOutput),
    squad_submit_review: tool(squadSubmitReview),
    squad_read_task: tool(squadReadTask),
    squad_mark_done: tool(squadMarkDone),
  },
})
