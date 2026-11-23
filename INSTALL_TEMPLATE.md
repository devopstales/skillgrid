- copy the .agents/ folder to your repo root
- copy the content of git/hooks/ to the repo's .git/hooks/ folder
- copy the content of omp/ to the repo's .omp/ folder

```
.agents/
  agents/        # shared role + persona briefs (canonical source)
  skills/
.git/
  hooks
.omp/
  agents/        # omp project agent definitions (one per persona → .agents/agents/)
  hooks/
```

The `omp/agents/` files are omp project agent definitions (per
`omp.sh/docs/subagent-authoring`): YAML frontmatter (`name` matching the
filename, a selection-hint `description`, a minimal `tools` grant, and a
`model` role alias or `prewalk` where the role has a stable routing) plus a
body that points the subagent at the shared brief under `.agents/agents/`
(one file per persona). Edit the shared brief, not the reference — the
reference only carries the capability boundary, the routing, and the "read
first, or tell the parent" fallback.

Model routing (role aliases; the concrete model is set per project in
`/model` → Roles, not in these files):

| Agent | Model | Why |
|---|---|---|
| `scout` | `@smol` | fast, read-only, JSON-out — matches omp's bundled scout route |
| `adversarial-author` | `@smol` | cheaper tier for the draft; the reviewer is the stronger tier (cross-model round) |
| `adversarial-reviewer`, `task-reviewer`, `security-sentinel`, `decomposer` | `@slow` | thorough review / architecture work |
| `implementer`, `senior-dev`, `senior-qa`, `re-reviewer`, `wiki-keeper`, `red-team-skeptic`, `project-manager-backlog` | inherited | model is chosen per dispatch by `skillgrid:execute`'s Model Selection — an omitted `model` here does not pin it |

The `prewalk: true` on the implementer roles hands planning to the resolved
model before a cheaper model implements — opt per project in `/agents` if
the cost is not wanted. `scout` intentionally shadows omp's bundled `scout`
agent (project definitions win over bundled ones) — skillgrid's scout has a
different return contract (zero-inference JSON), so the bundled behavior is
replaced on purpose.
