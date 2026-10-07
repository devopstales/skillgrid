// @ts-nocheck
const BASE_URL =
  process.env.SKILLGRID_CHECKPOINT_URL ||
  process.env.SKILLGRID_MNEMONIC_HTTP_URL ||
  "http://127.0.0.1:7438"

const AGENT = process.env.SKILLGRID_AGENT || "opencode"

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
    "Write task output.md and advance status to review_spec. Returns the updated task.",
  args: {
    task_id: tool.schema.string().describe("Task id"),
    output: tool.schema.string().describe("Markdown output content"),
    summary: tool
      .schema.string()
      .optional()
      .describe("Optional one-line summary"),
  },
  async execute(args) {
    const data = await api(
      "POST",
      `/teams/tasks/${args.task_id}/output`,
      args
    )
    return JSON.stringify(data)
  },
}

const squadSubmitReview = {
  description:
    "Submit a peer review (spec_compliance or code_quality) with markdown comments.",
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
    const data = await api(
      "POST",
      `/teams/tasks/${args.task_id}/review`,
      args
    )
    return JSON.stringify(data)
  },
}

const squadReadTask = {
  description: "Read task metadata and brief from disk.",
  args: {
    task_id: tool.schema.string().describe("Task id"),
  },
  async execute(args) {
    const data = await api("GET", `/teams/tasks/${args.task_id}`)
    return JSON.stringify(data)
  },
}

const squadMarkDone = {
  description: "Mark a team task as done.",
  args: {
    task_id: tool.schema.string().describe("Task id"),
  },
  async execute(args) {
    const data = await api(
      "POST",
      `/teams/tasks/${args.task_id}/done`,
      { agent: AGENT }
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
