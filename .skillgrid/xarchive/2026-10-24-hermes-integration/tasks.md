# Tasks — Hermes Agent Integration

> **STATUS:** `sliced` (2026-10-25)

> Sliced from `.skillgrid/specs/2026-10-24-hermes-integration/blueprint.md`.
> Vertical tracer-bullet tickets, dependency-ordered, sized for one fresh agent context window.

## Epic Summary

Make Hermes Agent a first-class install target for skillgrid: a native MemoryProvider plugin (Python) gives Hermes auto-prefetch, sync-turn persistence, and first-class memory tools; an MCP server complements it with code intelligence and web search; the skill catalog is deployed; the memory protocol is injected; and Hermes Kanban serves as a second ticketing backend. All wired by `skillgrid install --agents hermes`.

## Delivery Strategy

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1500-2000 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (provider) → PR 2 (setup + MCP) → PR 3 (install + skills + Kanban) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | MemoryProvider plugin (Python) | PR 1 | Manual: copy to ~/.hermes/plugins/mnemonic/, hermes memory setup, round-trip | Real Hermes session | `plugins/hermes/` (3 files, delete to revert) |
| 2 | SetupHermes + MNEMONIC_MCP_EXCLUDE | PR 2 | `cd mnemonic && go test ./internal/setup/ -run TestSetupHermes` | Go test (no Hermes needed) | `mnemonic/internal/setup/hermes.go` + `hermes_test.go` (delete to revert) |
| 3 | Install flow + skills + Kanban ticketing | PR 3 | `cd skillgrid-cli && go test ./...` + manual: skillgrid install --dry-run --agents hermes | Go test + manual install | `skillgrid-cli/internal/install/hermes.go` + `install.go` edit + `ticketing/SKILL.md` edit |

> The four plain-text guard lines above are the **contract** — `skillgrid:subagent-execution` matches them literally. If risk is High, `Chained PRs recommended` MUST be `Yes` and every work unit MUST name a focused test, a runtime harness (or `N/A` + reason), and a rollback boundary.

## Tickets

### TICKET-01 — MemoryProvider plugin: provider.py + __init__.py + plugin.yaml

- **Tracker ID:** TASK-115.01
- **Scope:** Create the Python MemoryProvider plugin at `plugins/hermes/` — the full lifecycle class, the `register(ctx)` entry point, and the plugin manifest. Verify the mnemonic CLI flags the provider needs (add `--json` if missing).
- **Acceptance:** `plugins/hermes/provider.py` defines `MnemonicMemoryProvider(MemoryProvider)` with all 12 methods (is_available, initialize, system_prompt_block, prefetch, queue_prefetch, sync_turn, get_tool_schemas, handle_tool_call, on_session_end, on_pre_compress, on_memory_write, shutdown). `__init__.py` defines `register(ctx)`. `plugin.yaml` has `name: mnemonic`. The provider's `is_available()` returns True when `mnemonic` is on PATH. `get_tool_schemas()` returns 5 valid OpenAI schemas. `handle_tool_call("mem_save", {...})` calls `mnemonic mem save` and returns JSON.
- **SATISFIES:** happy path — MemoryProvider is discovered and activates; happy path — MemoryProvider exposes memory tool schemas
- **Files:** `plugins/hermes/__init__.py` (new), `plugins/hermes/provider.py` (new), `plugins/hermes/plugin.yaml` (new), `mnemonic/cmd/mnemonic/mem.go` (edit: add `--json` flags if missing)
- **Size:** ~400 (S)
- **Blocks:** TICKET-02
- **Blocked by:** none
- **Reversibility:** reversible
- **Fails-when:** Python import error when loading `plugins/hermes/`; `mnemonic mem save --json` exits non-zero

### TICKET-02 — SetupHermes: Go installer for Hermes

- **Tracker ID:** TASK-115.02
- **Scope:** Implement `SetupHermes()` in `mnemonic/internal/setup/hermes.go` — copies the plugin, writes MCP config to `~/.hermes/config.yaml` (YAML), sets `MNEMONIC_MCP_EXCLUDE=mem`, writes SOUL.md marker block, creates Kanban board, copies skills, updates ticketing config. Add the `"hermes"` case to `RunSetup`. Add `MNEMONIC_MCP_EXCLUDE` handling to `mnemonic mcp`.
- **Acceptance:** `SetupHermes()` copies `plugins/hermes/` → `~/.hermes/plugins/mnemonic/`. Writes `mcp_servers` block to `config.yaml` with all servers from `config.d/mcp.yaml`. The mnemonic entry has `env: {MNEMONIC_MCP_EXCLUDE: "mem"}`. SOUL.md gets a marker-delimited protocol block (idempotent — replaced on re-run). Kanban board created (non-fatal if exists). Skills copied. `.skillgrid/config.yaml` updated with `type: hermes-kanban`. `RunSetup("hermes", ...)` dispatches to `SetupHermes()`. `MNEMONIC_MCP_EXCLUDE=mem` suppresses `mem_*` tools from `mnemonic mcp`. Dry-run writes nothing.
- **SATISFIES:** happy path — MCP server is registered in Hermes config; happy path — MCP server passes the Hermes test; edge — all MCP servers from config.d/mcp.yaml are registered; failure — malformed config.yaml triggers backup and error; happy path — protocol fallback in SOUL.md; failure — SOUL.md block is replaced on re-install
- **Files:** `mnemonic/internal/setup/hermes.go` (new), `mnemonic/internal/setup/hermes_test.go` (new), `mnemonic/internal/setup/setup.go` (edit), `mnemonic/cmd/mnemonic/mcp.go` (edit)
- **Size:** ~600 (M)
- **Blocks:** TICKET-03
- **Blocked by:** TICKET-01
- **Fails-when:** `cd mnemonic && go test ./internal/setup/ -run TestSetupHermes` fails; `go build ./...` fails

### TICKET-03 — Install flow: hermes in agent selection + prerequisite check

- **Tracker ID:** TASK-115.03
- **Scope:** Add `hermes` to the skillgrid installer's agent selection, prerequisite check (hermes binary on PATH, non-fatal), and install flow. Wire `setup.RunSetup("hermes", ...)` into the install sequence.
- **Acceptance:** `skillgrid install --agents hermes` selects hermes, checks `hermes` on PATH (prints `uv tool install hermes-agent` if missing, continues), calls `SetupHermes`. `skillgrid install --dry-run --agents hermes` prints planned changes without writing. `skillgrid install --agents opencode` still works (no regression).
- **SATISFIES:** happy path — full install configures all layers; edge — dry-run prints planned changes without writing; failure — install continues when hermes binary is missing; happy path — existing agents still install correctly
- **Files:** `skillgrid-cli/internal/install/install.go` (edit), `skillgrid-cli/internal/install/agents.go` (edit)
- **Size:** ~200 (S)
- **Blocks:** TICKET-04
- **Blocked by:** TICKET-02
- **Fails-when:** `cd skillgrid-cli && go test ./...` fails

### TICKET-04 — Kanban ticketing backend in the ticketing skill

- **Tracker ID:** TASK-115.04
- **Scope:** Add the `hermes-kanban` backend to the `ticketing` skill. Read `ticketing.type` from `.skillgrid/config.yaml`; when `hermes-kanban`, publish tickets via `hermes kanban create` with `--assignee`, `parents`, `--board`, `--skill`. Map skillgrid ticket fields to Kanban task fields. Update the Backlog.md entry with the Kanban task ID (mirror).
- **Acceptance:** When `ticketing.type: hermes-kanban`, the ticketing skill calls `hermes kanban create <title> --assignee <profile> --board <slug>`. Tickets with dependencies get `parents`. Coding tickets use `worktree` workspace; research tickets use `scratch`. The local Backlog.md entry is updated with the Kanban task ID. When `ticketing.type: backlogmd`, the skill writes to `.backlog/` (unchanged behavior).
- **SATISFIES:** happy path — ticket is published to Hermes Kanban; happy path — ticket with dependencies creates a Kanban link; edge — Kanban board is created if it does not exist; failure — Kanban board creation failure is non-fatal; edge — existing Backlog.md ticketing still works
- **Files:** `.agents/skills/planning/ticketing/SKILL.md` (edit), `.skillgrid/config.yaml` (edit: add `hermes-kanban` schema)
- **Size:** ~150 (S)
- **Blocks:** none
- **Blocked by:** TICKET-03
- **Fails-when:** `hermes kanban list` does not show the published ticket; Backlog.md ticketing path is broken

### TICKET-05 — Protocol + Cron + end-to-end verification

- **Tracker ID:** TASK-115.05
- **Scope:** Verify the protocol is injected via `system_prompt_block()`. Add cron job templates (printed as next-steps after install). Run the full end-to-end verification: memory round-trip, MCP test, skills discovery, Kanban round-trip, no-regression check.
- **Acceptance:** The system prompt contains the Mnemonic Memory Protocol section. Cron templates are printed after install. Memory round-trip works (`hermes -z "Remember X"` → `hermes -z "What is X?"`). `hermes mcp test skillgrid-mnemonic` exits 0. Skills are discoverable. Kanban ticket round-trip works. `cd mnemonic && go test ./...` and `cd skillgrid-cli && go test ./...` pass.
- **SATISFIES:** happy path — protocol is injected via system_prompt_block; happy path — save and recall a fact; edge — prefetch returns empty on the first turn; failure — sync_turn does not block the agent on mnemonic error; failure — hermes install does not affect existing agent configs
- **Files:** `plugins/hermes/provider.py` (edit: verify system_prompt_block), `mnemonic/internal/setup/hermes.go` (edit: cron templates in install output)
- **Size:** ~100 (S)
- **Blocks:** none
- **Blocked by:** TICKET-04
- **Fails-when:** Memory round-trip fails; `hermes mcp test` fails; `go test ./...` fails

## Dependency Graph

```mermaid
graph LR
    T01[TICKET-01: Provider plugin] --> T02[TICKET-02: SetupHermes]
    T02 --> T03[TICKET-03: Install flow]
    T03 --> T04[TICKET-04: Kanban ticketing]
    T04 --> T05[TICKET-05: Protocol + E2E]
```

## Execution Order

- **Wave 1:** TICKET-01 (provider plugin — the primary integration, de-risks the Python↔Go boundary)
- **Wave 2:** TICKET-02 (SetupHermes — wires the provider into Hermes, adds MCP exclude)
- **Wave 3:** TICKET-03 (install flow — hermes in the agent selection)
- **Wave 4:** TICKET-04 (Kanban ticketing — second backend in the ticketing skill)
- **Wave 5:** TICKET-05 (protocol verification + cron + end-to-end)

> **Acceptance-first (BDD is always on):** a ticket's failing acceptance
> scenario (its `SATISFIES` scenario) is written and confirmed RED *before*
> the implementation that makes it green. Order RED-test / scenario tickets
> ahead of their implementation tickets in the dependency graph.

## Slicing Notes

- **Tracer thread:** TICKET-01 is the thin end-to-end path (Python provider → mnemonic CLI → SQLite). It de-risks the Python↔Go subprocess boundary before the Go installer thickens.
- **Serial, not parallel:** The tickets are strictly serial because each builds on the previous (provider → setup → install → ticketing → verification). No independent tickets within a wave.
- **Kanban is last:** The Kanban ticketing backend (TICKET-04) depends on the install flow (TICKET-03) because the board is created during install. It's a skill edit (markdown), not Go code — the smallest ticket.
- **No migration:** This integration adds new files (plugins/hermes/, hermes.go) and edits existing files (setup.go, install.go, mcp.go, ticketing/SKILL.md, config.yaml). No schema migrations.
- **Python in a Go repo:** The provider is Python (Hermes runtime requirement). It lives at `plugins/hermes/` alongside the existing `plugins/opencode/` (TypeScript) and `plugins/kilo/` (TypeScript) plugins. The Go installer copies it; the Go code never imports it.
