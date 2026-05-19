# Web UI

The AISkillGrid web UI is a local dashboard for project workflow state. It makes agent work visible without requiring a hosted platform.

Start it from the project root:

```bash
node .skillgrid/scripts/skillgrid-ui.mjs
```

Then open the local address printed by the server. The common default is:

```text
127.0.0.1:8787
```

Alternative: `skillgrid serve` from a built `skillgrid-cli` binary (see hub README).

For **terminal** beads triage and graph-aware task ranking, use [beads_viewer](17-external-tools.md) (`bv`) — Skillgrid does not ship a built-in CLI TUI.

## Why The Web UI Exists

AI work can become invisible when it lives only in chat. The web UI gives users and stakeholders a place to see:

- What PRDs exist.
- Which phase a change is in.
- What is blocked.
- What subagents did.
- Which previews exist.
- Whether graph output is available.
- Whether Engram shared-memory export metadata exists.
- Whether the project skill registry exists.
- What event history has been recorded.

This turns AISkillGrid from a prompt library into an observable workflow.

## Main Views

```mermaid
flowchart TD
  WebUI[Local Web UI] --> Board[Board View]
  WebUI --> Agents[Agents View]
  WebUI --> Workflow[Workflow View]
  WebUI --> Subagents[Subagents View]
  WebUI --> Previews[Preview Links]
  WebUI --> Graph[GitNexus View]
  Board --> PRD[PRD Files]
  Agents --> OpenSessions[OpenSessions WS 7391]
  Workflow --> Events[Event Logs]
  Subagents --> Reports[Research And Reports]
  Graph --> GitNexus[GitNexus UI]
  WebUI --> Memory[Engram And Registry Status]
```

## Board View

The Board view shows PRDs in workflow columns.

It helps answer:

- What work exists?
- What status is each PRD in?
- Which items are blocked?
- Which items have previews?
- Which item should be opened in the Workflow view?

When the workflow status changes, the dashboard updates the PRD status field. That keeps the board tied to files instead of a hidden database.

The visible workflow phase order defaults to:

```text
brainstorm, design, plan, breakdown, apply, test, security, validate, finish
```

Projects can override the Workflow view order through `.skillgrid/config.json`:

```json
{
  "workflow": {
    "phaseOrder": ["plan", "breakdown", "apply", "validate", "finish"]
  }
}
```

Events remain the source of truth; the configured phase order only controls the dashboard lanes and empty-state ordering.

## Workflow View

The Workflow view focuses on one selected PRD or change.

It can show:

- Current phase.
- Current state.
- Next recommended action.
- HITL blockers.
- AFK-ready work.
- Event timeline.
- Artifacts such as PRD, handoff, previews, research, tests, and review reports.

This is the view to use when asking, “Can the agent keep going, or does a human need to decide something?”

## Agents View (OpenSessions)

The **Agents** tab shows live coding-agent status from [opensessions](17-external-tools.md) when the tmux sidebar server is running.

It helps answer:

- Which tmux sessions match this repository?
- What agent is running (idle, tool-running, done, error, …)?
- Is there progress metadata or an unseen session marker?
- What branch and ports are attached to each session?

The dashboard polls the OpenSessions WebSocket API on **`127.0.0.1:7391`** (override with `OPENSESSIONS_HOST` / `OPENSESSIONS_PORT`). When the server is offline, the view shows the install/start hint from the tools panel instead of session cards.

This complements file-based subagent reports: OpenSessions reflects **live terminal/agent runtime**, while `.skillgrid/tasks/research/` holds durable persona and research outputs.

## Subagents View

The Subagents view collects delegated work activity.

It is useful for:

- Seeing what reviewers, researchers, critics, auditors, and verifiers did.
- Finding their output files.
- Spotting blockers.
- Checking whether independent reports agree.

This makes multiagent work easier to trust because the activity is visible.

## Preview Links

When a workflow produces HTML previews, the dashboard can surface them from:

```text
.skillgrid/preview/
```

This is especially useful for UI design, prototypes, visual comparisons, or generated documentation pages.

## GitNexus View

When GitNexus is indexed or available, the dashboard exposes a GitNexus view with index status, setup commands, and the embedded GitNexus web UI. Starting the Skillgrid dashboard also starts the local GitNexus web runtime, so users do not need to run a separate `gitnexus serve` process first.

Typical local source:

```text
.gitnexus/
```

This gives users a quick way to jump from workflow state into codebase structure, impact analysis, and graph-aware exploration.

## Data Sources

The dashboard reads files that already belong to the Skillgrid workflow:

| Source | What It Powers |
|---|---|
| `.skillgrid/prd/` | Board cards and product intent |
| `.skillgrid/tasks/context_<change-id>.md` | Current state and next action |
| `.skillgrid/tasks/events/<change-id>.jsonl` | Timeline and subagent activity |
| `.skillgrid/tasks/research/<change-id>/` | Reports and research artifacts |
| `.skillgrid/preview/` | Preview links |
| `.gitnexus/` | GitNexus view and graph status |
| `.engram/manifest.json` | Engram export counts, when team memory sync is used |
| `.skillgrid/project/SKILL_REGISTRY.md` | Skill registry availability and skill count |
| OpenSessions WebSocket (`127.0.0.1:7391`) | Agents tab — live tmux session and agent status |

No separate database is required for the core local dashboard model.

The dashboard reads `.engram/manifest.json` directly when it exists. It does not call the Engram CLI or expose memory contents.

## Practical Advantage

The web UI gives AISkillGrid a visible operating surface. New users can understand what is happening without reading every artifact by hand. Leads and reviewers can inspect state without asking the agent to summarize itself.

That visibility is a major difference between a full workflow solution and a collection of isolated prompts.
