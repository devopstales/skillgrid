# Plugins

AISkillGrid ships **plugins** as per-harness adapters: one shared behavior, one small adapter per agent host. Plugins install **globally** (machine-wide, under each host's config directory) with `skillgrid install`, so every project on the machine gets the behavior without per-project wiring. Project state stays per-repo in `.skillgrid/`; a plugin is only the bridge between a host session and that state.

## Quick path

| Plugin | Hosts | What it does | Status |
|--------|-------|--------------|--------|
| [Skillgrid router](#skillgrid-router) | omp, pi, opencode, Kilo, Cursor | auto-loads the `using-skillgrid` router at session start | proposed (change `2026-11-17-skillgrid-host-plugins`) |
| [Mnemonic memory](#mnemonic-memory) | omp, pi, opencode, Kilo, Cursor | `mem_*` / `code_*` / `kb_*` tools, prompt and session capture, compaction context | shipped |
| [TUI logo](#tui-logo) | opencode, Kilo | tokyonight theme and Skillgrid logo in the TUI | shipped |
| [Obsidian backlogmd](#obsidian-backlogmd) | Obsidian | Backlog.md kanban as an Obsidian vault view | shipped |

## Distribution

`skillgrid install --agents omp,skill-pi,opencode,kilo,cursor` clones the hub into the staging area (`~/.skillgrid/repos/skillgrid`), stages hooks and plugins under `~/.skillgrid/` (`hooks/`, `plugins/`), and wires each selected host's config directory. `--lsp` adds LSP servers, `--dry-run` previews the writes, `--yes` skips prompts. `skillgrid sync-repo` refreshes the staged clone from the current branch; hosts pick up plugin changes at their next session start or reload.

Where the installer writes, per host:

| Host | Config root | What lands there |
|------|-------------|------------------|
| omp | `~/.omp/agent/` | `extensions/` (TS), `AGENTS.md`, `mcp.json` (`mnemonic mcp`); plugin bundles under `~/.omp/plugins/` |
| pi (skill-pi) | `~/.pi/agent/` + `~/.skill-pi/agent/` | `extensions/` (TS), skills, `AGENTS.md` |
| opencode | `~/.config/opencode/` | `opencode.jsonc`, `tui.json`, `tui-plugins/` (logo bundle), native TS plugins, `AGENTS.md` |
| Kilo | `~/.config/kilo/` | `kilo.jsonc`, `tui.json`, `plugin/` (TS), `AGENTS.md` |
| Cursor | `~/.cursor/` | `mcp.json`, `agents/`; the Cursor plugin itself installs from the repo (`.cursor-plugin/`) via marketplace or add-from-folder |

How each host loads and updates plugins:

| Host | Load paths | Update semantics |
|------|------------|------------------|
| omp | `~/.omp/agent/extensions/*.ts`, plugin bundles (`package.json` `omp.extensions`) | extension changes need a full restart, not `/reload-plugins` |
| pi | `~/.pi/agent/extensions/*.ts`, project `.pi/extensions/` | `/reload` hot-reloads; project extensions need project trust |
| opencode | `~/.config/opencode/plugins/`, project `.opencode/plugins/` | watched dirs hot-reload; V1 plugin impls do not run in V2 |
| Kilo | `~/.config/kilo/plugin/` (auto-registered), project `.kilo/plugin/` | loaded at CLI/extension start; `KILO_PURE=1` skips external plugins |
| Cursor | `~/.cursor/plugins/local/<name>/` or a marketplace repo | reinstall or window reload |

## Skillgrid router

The router plugin makes each host **deterministically load the `using-skillgrid` skill at session start**, the way [obra/superpowers](https://github.com/obra/superpowers) auto-loads `using-superpowers`. It matters because the router is `disable-model-invocation: true`: every host hides it from the automatic skill list, so it only runs if the agent notices the `AGENTS.md` entry line. The plugin replaces that soft trigger with a hard injection.

Mechanism: at session start, if the project has a `.skillgrid/config.yaml`, the adapter injects a compact bootstrap (~15 lines) read live from `~/.skillgrid/router-bootstrap.md`. The bootstrap says one thing: **first action, read `.agents/skills/using-skillgrid/SKILL.md`** (or the host's skill-load command for `using-skillgrid`), then announce `Using [skill] to [purpose]`. The full 14KB router body stays on disk; each host's native skill system already auto-discovers `.agents/skills/` and loads it on demand. Injections are main-agent only (subagents skip, mirroring the skill's `<SUBAGENT-STOP>`), fail-open, and carry a static fallback string if the asset file is missing.

Per-host mechanism (native strongest for each):

| Host | Mechanism |
|------|-----------|
| Cursor | plugin `sessionStart` hook (`hooks/cursor-session-start.sh`) emits the bootstrap as `additional_context`, before the Mnemonic-server block so it injects with the server down |
| Kilo | auto-registered TS plugin in `~/.config/kilo/plugin/`: `config` hook adds the skills dir, `messages.transform` injects the bootstrap into the first user message (mirrors the proven superpowers Kilo adapter) |
| opencode | V2 `Plugin.define` plugin: `session.created` event → durable `instructions.entry` for the session; V1 named export for opencode 1.x |
| pi | extension `session_start` arms the bootstrap; the existing system-prompt injection seam (shared with the Mnemonic memory protocol) appends it |
| omp | extension under `~/.omp/agent/extensions/` overriding the system prompt at `before_agent_start`; the `AGENTS.md` entry line remains the no-code fallback |

Status: blueprint proposed at the `blueprint-validation` gate (change `2026-11-17-skillgrid-host-plugins`, spec zone under `.skillgrid/change-specs/`). Research evidence for every host mechanism: `/tmp/skillgrid-plugin-research/`.

## Mnemonic memory

Registers the native `mem_*` / `code_*` / `kb_*` tools and captures session activity for the Mnemonic store, fail-open throughout (a dead server never blocks a prompt).

- **opencode / Kilo**: five native TS plugins in `plugins/opencode/` (dual V1+V2 export, since V1 impls do not run in V2), installed to `~/.config/opencode/plugins/`; Kilo reuses the same adapters through the `kilo.jsonc` → `opencode.jsonc` bridges.
- **pi**: the extension in `plugins/pi/mnemonic/` (entry `index.ts`) registers the tools, captures prompts on `before_agent_start`, injects the memory protocol into the system prompt, builds the compaction context on `session_before_compact`, and ends the session on `session_shutdown`.
- **Cursor**: `plugins/cursor/mcp.json` exposes the Mnemonic MCP server (`skillgrid mcp`) through the Cursor plugin; `hooks/hooks-cursor.json` wires the sessionStart/sessionEnd/stop/tool-capture shell hooks.
- **omp**: the MCP server is registered in `~/.omp/agent/mcp.json` (`mnemonic mcp`), which mounts the tools as `xd://` devices.

See [Memory and indexing](05-memory-and-indexing.md) for the memory model and the HTTP server the adapters talk to.

## TUI logo

Sets the tokyonight theme and drops the Skillgrid logo into the opencode and Kilo TUIs. OpenCode never transpiles raw TSX, so the opencode payload is a pre-built ESM bundle (`plugins/opencode/skillgrid-logo.js`) referenced through a bridge-dir package; Kilo takes the `.tsx` directly into `tui-plugins/`. Configured through each host's `tui.json` during install.

## Obsidian backlogmd

A vault plugin under `plugins/obsidian/backlogmd/` that renders the Backlog.md kanban (tasks, milestones, sprints) as an Obsidian view. It is the only plugin that targets a non-agent host; it installs into the vault and is versioned by `versions.json`. See [Ticketing](07-ticketing.md) for the Backlog.md side.

## Adding a plugin or a host

A plugin is three things: the shared behavior (skills, hooks, or Go-side services), one adapter directory under `plugins/<host>/`, and the installer wiring in `skillgrid-cli/internal/install/` (the agent key in `--agents`, the staging copy, the host config edit, with `--dry-run`/`--verbose` parity). Adapters follow the house rules: fail-open, no new Go dependencies without an ADR, and a test that asserts the dry-run writes the expected files. The per-host event/injection APIs change often; the research notes in the change spec's `findings` are the current reference, not this page.
