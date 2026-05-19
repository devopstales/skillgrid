# External Helper Tools

This page documents the three external tools that Skillgrid integrates with: `opensessions`, `beads_viewer` (bv), and `ralph-tui`. Each section covers purpose, installation, configuration, and integration patterns.

---

## 1. opensessions

**Repo:** [Ataraxy-Labs/opensessions](https://github.com/Ataraxy-Labs/opensessions)  
**License:** MIT  
**Stack:** TypeScript / Bun, OpenTUI sidebar

### Purpose

`opensessions` is a `tmux` sidebar for coding agents. It provides per-thread session state, live markers (done / error / interrupted), agent-aware detection, and a programmatic HTTP API for scripts to push status and logs.

### Prerequisites

- `tmux`
- `bun >= 1.0.0`
- TPM (optional, for plugin management)

### Installation

#### Via TPM (recommended)

Add to `~/.tmux.conf`:

```tmux
set -g @plugin 'Ataraxy-Labs/opensessions'
```

Then reload and install:

```bash
tmux source-file ~/.tmux.conf
~/.tmux/plugins/tpm/bin/install_plugins
```

Or run the one-liner installer:

```bash
grep -q "Ataraxy-Labs/opensessions" ~/.tmux.conf 2>/dev/null || printf '\nset -g @plugin '\''Ataraxy-Labs/opensessions'\''\n' >> ~/.tmux.conf && tmux source-file ~/.tmux.conf && ~/.tmux/plugins/tpm/bin/install_plugins
```

#### Local clone

```bash
git clone https://github.com/Ataraxy-Labs/opensessions.git
cd opensessions
bun install
```

Add to `~/.tmux.conf`:

```tmux
source-file /absolute/path/to/opensessions/opensessions.tmux
```

### Keybindings

| Shortcut | Action |
|----------|--------|
| `prefix o -> s` | Reveal and focus the sidebar |
| `prefix o -> t` | Toggle the sidebar |
| `prefix o -> e` | Even-horizontal layout (spread non-sidebar panes) |
| `prefix o -> 1` through `prefix o -> 9` | Quick session switch |
| `j` / `k` | Navigate session list |
| `Enter` | Switch session |
| `n` / `c` | Create new session |
| `t` | Theme picker |

### Configuration

Override defaults in `~/.tmux.conf`:

```tmux
set -g @opensessions-width "30"
```

### Agent Detection

`opensessions` watches transcript files from supported agents:

| Agent | Watched Path |
|-------|-------------|
| Amp | `~/.local/share/amp/threads/*.json` |
| Claude Code | `~/.claude/projects/` (JSONL) |
| Codex | `~/.codex/sessions/` or `$CODEX_HOME/sessions/` |
| OpenCode | SQLite database in `~/.local/share/opencode/opencode.db` |

### Programmatic API

The local server runs on `127.0.0.1:7391`. All endpoints accept `POST` with `Content-Type: application/json`.

#### Set status

```bash
curl -sS -X POST http://127.0.0.1:7391/set-status \
  -H 'content-type: application/json' \
  -d '{"session":"my-session","text":"Building","tone":"info"}'
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `session` | `string` | yes | Mux session name |
| `text` | `string \| null` | yes | Status text (or `null` to clear) |
| `tone` | `string` | no | `neutral`, `info`, `success`, `warn`, `error` |

#### Set progress

```bash
curl -sS -X POST http://127.0.0.1:7391/set-progress \
  -H 'content-type: application/json' \
  -d '{"session":"my-session","current":3,"total":10,"label":"files"}'
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `session` | `string` | yes | Session name |
| `current` | `number` | no | Current step |
| `total` | `number` | no | Total steps |
| `percent` | `number` | no | Progress as 0.0–1.0 |
| `label` | `string` | no | Short label shown with the number |
| `clear` | `boolean` | no | Set to `true` to clear |

#### Push a log entry

```bash
curl -sS -X POST http://127.0.0.1:7391/log \
  -H 'content-type: application/json' \
  -d '{"session":"my-session","message":"Build started","source":"ci","tone":"info"}'
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `session` | `string` | yes | Session name |
| `message` | `string` | yes | Log message (max 500 chars) |
| `tone` | `string` | no | Tone level |
| `source` | `string` | no | Source label (e.g. `ci`, `build`) |

#### Other endpoints

| Endpoint | Purpose |
|----------|---------|
| `POST /clear-log` | Clear all log entries for a session |
| `POST /notify` | Send a notification (same fields as `/log`) |

### Update

```bash
~/.tmux/plugins/tpm/bin/update_plugins opensessions
```

### Uninstall

```bash
sh ~/.tmux/plugins/opensessions/integrations/tmux-plugin/scripts/uninstall.sh
# Then remove the set -g @plugin line from ~/.tmux.conf
tmux source-file ~/.tmux.conf
```

### Local Development

```bash
git clone https://github.com/Ataraxy-Labs/opensessions.git
cd opensessions
bun install
bun test
bun run start:tui
```

### Integration with Skillgrid

- Use the HTTP API from build/CI scripts to push status pills and log entries during SDD phases.
- Session context (branch, working directory, thread names, detected localhost ports) updates automatically via agent watchers.
- Per-thread unseen markers help agents track which SDD steps have been communicated to humans.

---

## 2. beads_viewer (bv)

**Repo:** [Dicklesworthstone/beads_viewer](https://github.com/Dicklesworthstone/beads_viewer)  
**License:** MIT + OpenAI/Anthropic Rider  
**Stack:** Go, Bubbletea TUI  
**Version:** v0.16.1 (as of 2026-05-14)

### Purpose

`bv` is a graph-aware TUI for the Beads issue tracker. It treats a project as a directed graph and computes 9 graph-theoretic metrics (PageRank, betweenness, HITS, critical path, etc.) to produce deterministic, dependency-aware task recommendations for AI agents.

### Installation

#### Homebrew (macOS/Linux)

```bash
brew install dicklesworthstone/tap/bv
```

#### Scoop (Windows)

```powershell
scoop bucket add dicklesworthstone https://github.com/Dicklesworthstone/scoop-bucket
scoop install dicklesworthstone/bv
```

#### Install script (Linux/macOS)

```bash
curl -fsSL "https://raw.githubusercontent.com/Dicklesworthstone/beads_viewer/main/install.sh" | bash
```

#### Direct download

Download from [the latest release page](https://github.com/Dicklesworthstone/beads_viewer/releases/latest).

### Prerequisites for JSONL Source

`bv` reads from `.beads/beads.jsonl`. Generate it from:

| Tool | Command |
|------|---------|
| `br` (Beads-Rust) | Writes `.beads/beads.jsonl` by default |
| `bd` (Beads-Go) | `bd export --no-memories -o .beads/beads.jsonl` |

### Modes

TUI modes (single `key` trigger):

| Key | Mode | Description |
|-----|------|-------------|
| (default) | List view | Split-view dashboard with fast list + rich details |
| `b` | Kanban board | Columnar view (Open, In Progress, Blocked, Closed) |
| `g` | Graph view | Visual dependency DAG with ASCII renderer |
| `i` | Insights | PageRank, betweenness, cycles, critical path dashboard |
| `h` | History | Timeline of commits mapped to bead modifications |
| `t` | Time-travel | Compare against any git revision |
| `T` | HEAD~5 | Quick comparison against HEAD~5 |

TUI navigation:

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate list |
| `/` | Fuzzy search across ID, title, content, labels |
| `o` | Filter to Open issues |
| `c` | Filter to Closed issues |
| `r` | Filter to Ready (unblocked) issues |
| `E` | Export all issues to timestamped Markdown with Mermaid diagrams |
| `C` | Copy selected issue as formatted Markdown |
| `O` | Open `.beads/beads.jsonl` in GUI editor |
| `t` | Time-travel to any git revision |

### Robot Mode (AI integration)

**Always use `--robot-*` flags. Never run bare `bv` in an agent context.**

```bash
# Triage: complete overview with recommendations
bv --robot-triage

# Minimal: top pick + claim command
bv --robot-next

# Graph analysis
bv --robot-insights

# Graph export
bv --robot-graph
bv --robot-graph --graph-format=dot
bv --robot-graph --graph-format=mermaid
bv --robot-graph --graph-root=bv-123 --graph-depth=3

# Planning
bv --robot-plan
bv --robot-priority

# Priority
bv --robot-label-health
bv --robot-label-flow
bv --robot-label-attention

# History
bv --robot-history
bv --robot-diff --diff-since HEAD~10

# Other
bv --robot-burndown <sprint>
bv --robot-forecast <id>
bv --robot-alerts
bv --robot-suggest
```

Token-optimized output (TOON):

```bash
bv --robot-triage --format toon
# or set environment variable
export BV_OUTPUT_FORMAT=toon
```

Scope filtering:

```bash
bv --robot-triage --label backend
bv --robot-triage --label backend,auth
bv --robot-plan --recipe actionable
bv --robot-triage --recipe high-impact
bv --robot-triage --robot-triage-by-track
bv --robot-triage --robot-triage-by-label
bv --robot-insights --as-of HEAD~30
```

#### Agent Auto-Integration

`bv` can inject its instructions into agent files:

```bash
bv --agents-check    # Check if blurb is present
bv --agents-add      # Add blurb (creates file if needed)
bv --agents-remove   # Remove blurb
bv --agents-update   # Update blurb to latest version
bv --agents-dry-run  # Preview changes
```

Supported files (checked in order): `AGENTS.md`, `CLAUDE.md`, `agents.md`, `claude.md`.

#### Key Graph Metrics (9 total)

| Metric | Measures | Pragmatic meaning |
|--------|----------|-------------------|
| PageRank | Recursive dependency importance | Foundational blockers |
| Betweenness | Shortest-path traffic | Bottlenecks and bridges |
| HITS (Hub) | Dependency aggregators | Epics |
| HITS (Authority) | Prerequisite providers | Utilities |
| Critical Path | Longest dependency chain | Keystones (zero slack) |
| Eigenvector | Influence via neighbors | Strategic dependencies |
| Degree | Direct connection count | Immediate blockers/blockeds |
| Density | Edge-to-node ratio | Project coupling health |
| Cycle Detection | Circular dependencies | Structural errors |

#### Output Schema

Robot commands output JSON to stdout. Key fields:

```json
{
  "data_hash": "abc123",
  "status": "computed|approx|timeout|skipped",
  "as_of": "2026-05-14T...",
  "bottlenecks": [{ "id": "CORE-123", "value": 0.45 }],
  "keystones": [{ "id": "API-001", "value": 12.0 }],
  "hubs": [{ "id": "EPIC-100", "value": 0.67 }],
  "cycles": [["TASK-A", "TASK-B", "TASK-A"]],
  "clusterDensity": 0.045,
  "stats": { "pageRank": {}, "betweenness": {}, ... }
}
```

Always check `status` field: `computed` = reliable, `approx` = large-graph heuristic, `skipped` = unavailable.

### Interactive Graph Export

Generate self-contained HTML visualization:

```bash
bv --export-graph graph.html --graph-title "Sprint Review"
```

Outputs a 400KB-1MB HTML file with force-directed graph, node hover details, PageRank sizing, and full search/filter/keyboard navigation. No server required.

### Integration with Skillgrid

- Use `bv --robot-triage` or `bv --robot-next` in SDD planning phases to get deterministic task priority recommendations based on graph structure.
- Use `bv --agents-add` to auto-inject robot-mode instructions into your project's `AGENTS.md`.
- Feed `bv` triage output into SDD task breakdown to align Beads issues with implementation priority.

---

## 3. ralph-tui

**Site:** [ralph-tui.com](https://ralph-tui.com)  
**GitHub:** [subsy/ralph-tui](https://github.com/subsy/ralph-tui)  
**Version:** v0.11.0  
**License:** MIT  
**Stack:** Bun runtime, OpenTUI

> **Skillgrid’s built-in loop:** For OpenSpec/SDD projects, use native [`/sdd-loop`](17-sdd-ralph-loop.md) and `.skillgrid/scripts/sdd-ralph-loop.sh` — not ralph-tui. Ralph TUI is optional when you want beads or `prd.json` trackers.

### Purpose

Ralph TUI is an AI agent loop orchestrator that manages autonomous coding agents through intelligent task routing. It picks the next task, builds a prompt, executes the agent, detects completion, and marks tasks complete -- in a continuous loop.

### Prerequisites

- `bun >= 1.0.0` (required runtime -- not optional)
- At least one AI coding agent: Claude Code, OpenCode, or Factory Droid

### Installation

```bash
bun install -g ralph-tui
ralph --version
```

### Project Setup

```bash
cd your-project
ralph-tui setup
```

The interactive wizard:

1. Detects installed agents
2. Creates `.ralph-tui/config.toml`
3. Installs bundled skills for PRD creation and task conversion
4. Detects existing trackers (Beads, JSON)

### Quick Start

```bash
# Create a PRD interactively
ralph-tui create-prd --chat

# Run autonomous execution
ralph-tui run --prd ./prd.json
```

### Agent Plugins

| Agent | Installation | Subagent Tracing |
|-------|-------------|------------------|
| Claude Code | `npm i -g @anthropic-ai/claude-code` | Yes (Task tool) |
| OpenCode | `curl -fsSL https://opencode.ai/install | bash` | Yes (Task tool) |
| Factory Droid | [Factory Droid CLI](https://docs.factory.ai/reference/cli-reference) | No |
| Codex | Via Claude Code agent plugin | (via Codex agent) |
| Cursor | Via Claude Code agent plugin | (via Cursor agent) |
| Gemini | Via Claude Code agent plugin | (via Gemini agent) |
| GitHub Copilot | Via Claude Code agent plugin | (via Copilot agent) |

### Tracker Plugins

| Tracker | External CLI | Dependencies | Graph Analysis | Git Sync | Epic Hierarchy |
|---------|-------------|-------------|---------------|----------|---------------|
| JSON | None | None | No | Manual | No |
| Beads | `bd` | Yes | No | Yes (`bd sync`) | Yes |
| Beads-BV | `bd` | Yes | Yes (bv) | Yes | Yes |
| Beads-Rust | `br` | Yes | No | Yes | Yes |
| Beads-Rust-BV | `br` + `bv` | Yes | Yes (bv) | Yes | Yes |

### Tracker Configuration

#### JSON tracker (simplest)

No configuration needed. Create `prd.json`:

```json
{
  "title": "My Feature",
  "tasks": [
    { "id": "1", "title": "Design", "status": "pending" },
    { "id": "2", "title": "Implement", "status": "pending", "depends-on": ["1"] }
  ]
}
```

Run: `ralph-tui run --prd ./prd.json`

#### Beads tracker

```bash
# Initialize
bd init

# Create epic and tasks
bd create --title "Auth Feature" --type epic
bd create --title "Login form" --type task --parent <epic-id>
bd dep add <task-b> <task-a>

# Run
ralph-tui run --tracker beads --epic <epic-id>
```

Config in `.ralph-tui/config.toml`:

```toml
tracker = "beads"

[trackerOptions]
beadsDir = ".beads"
# labels = "frontend,backend"
```

#### Beads-BV tracker (graph-aware)

```bash
# Install bv for smart task selection
cargo install bv
# or: curl -fsSL https://get.beads.sh/bv | sh
```

```bash
ralph-tui run --tracker beads-bv --epic <epic-id>
```

Config:

```toml
[tracker]
plugin = "beads-bv"

[tracker.options]
epicId = "auth-1"
labels = "frontend,backend"
```

#### Beads-Rust-BV tracker (high-performance)

```bash
# Install br + bv
cargo install beads-rust bv
br init

# Run with graph-aware task selection
ralph-tui run --tracker beads-rust-bv --epic <epic-id>
```

Config:

```toml
[tracker]
plugin = "beads-rust-bv"

[tracker.options]
epicId = "auth-1"
labels = ["auth", "backend"]
workingDir = "./workspace"
beadsDir = ".beads"
```

Falls back to `br ready` if `bv` is unavailable.

#### Beads-Rust tracker (no graph)

Same as Beads-Rust-BV without `bv`. Config:

```toml
[tracker]
plugin = "beads-rust"
```

### Agent Configuration

```toml
# .ralph-tui/config.toml
agent = "claude"
tracker = "json"

[agentOptions]
model = "sonnet"
```

Multiple agent configs:

```toml
[[agents]]
name = "claude-fast"
plugin = "claude"
[agents.options]
model = "haiku"

[[agents]]
name = "claude-smart"
plugin = "claude"
default = true
[agents.options]
model = "opust"
```

Override from CLI: `ralph-tui run --agent opencode --tracker beads-bv --epic my-epic`

### Skills Management

Skills are auto-installed during `ralph-tui setup`. Manual management:

```bash
ralph-tui skills list
ralph-tui skills install              # to global directory
ralph-tui skills install --local      # to project directory
ralph-tui skills install --agent claude
ralph-tui skills install --force      # overwrite
```

Skills locations:

| Scope | Claude Code | OpenCode | Factory Droid |
|-------|------------|----------|---------------|
| Global | `~/.claude/skills/` | `~/.config/opencode/skills/` | `~/.factory/skills/` |
| Local | `.claude/skills/` | `.opencode/skills/` | `.factory/skills/` |

### CLI Commands

| Command | Description |
|---------|-------------|
| `ralph-tui run` | Start autonomous execution loop |
| `ralph-tui resume` | Resume a previous session |
| `ralph-tui status` | Show current execution status |
| `ralph-tui logs` | View execution logs |
| `ralph-tui setup` | Run setup wizard |
| `ralph-tui create-prd` | Create PRD interactively |
| `ralph-tui convert` | Convert PRD to tracker format |
| `ralph-tui remote` | Remote session management |
| `ralph-tui doctor` | Diagnose installation issues |
| `ralph-tui info` | Show system info |

### Parallel Execution

Ralph supports multi-epic parallel runs:

```bash
ralph-tui run --parallel --epic frontend-epic --epic backend-epic
```

Each epic runs as a separate worktree. Tasks with cross-epic dependencies (where both tasks are in the selected set) are honored.

### TUI Controls

Keys while Ralph is running:

#### Execution

| Key | Action |
|-----|--------|
| `s` | Start |
| `p` | Pause/Resume |
| `+` / `=` | Add 10 iterations |
| `-` / `_` | Remove 10 iterations |
| `q` | Quit |
| `?` | Help |

#### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Navigate down |
| `k` / `↑` | Navigate up |
| `Tab` | Switch panel focus |
| `Enter` | Drill in |
| `Esc` | Go back |

#### View controls

| Key | Action |
|-----|--------|
| `o` | Cycle right panel views |
| `O` | Jump to prompt preview |
| `d` | Toggle progress dashboard |
| `g` | Cycle epic/scope |
| `G` | Return to all epics |
| `a` | Agent/model picker |
| `h` | Toggle show/hide closed tasks |
| `r` | Refresh task list |
| `T` | Toggle subagent tree |
| `t` | Cycle subagent detail level |

### Integration with Skillgrid

- Use Ralph TUI to execute SDD implementation tasks autonomously with any Beads tracker.
- `Beads-BV` or `Beads-Rust-BV` trackers align best with Skillgrid's Beads issue tracking -- graph-aware task ordering picks the most impactful unblocked task first.
- Skills installed by `ralph-tui setup` include `/ralph-tui-prd` and `/ralph-tui-create-beads` for PRD creation and Convergence.
- Use Claude Code as the agent for best subagent tracing integration within SDD phases.

---

## Tool Comparison Matrix

| Need | Best tool |
|------|-----------|
| Session state in tmux while coding | `opensessions` |
| Browse and manage Beads issues with graph analysis | `bv` |
| Autonomous agent task execution | `ralph-tui` |
| CI/CD status in tmux sidebar | `opensessions` HTTP API |
| AI-ready task triage | `bv --robot-triage` |
| Multi-epic parallel execution | `ralph-tui --parallel` |
| Full workflow | `bv` (plan) + `ralph-tui` (execute) + `opensessions` (observe) |
