# Skillgrid Editor Plugins — Design Spec

**Date:** 2026-08-27
**Status:** Draft
**Author:** Skillgrid Team

## Goal

Enable full Skillgrid integration (skills, MCP servers, slash commands, OpenSpec workflow enforcement) across three AI coding tools — **OpenCode**, **Kilo**, and **Cursor** — via native editor plugins. The `skillgrid-cli` is the primary installation method; plugins do everything they can natively within their editor.

## Architecture

### Design Principles

1. **Independent plugins** — each editor plugin is self-contained under `plugins/`. No shared runtime code. Shared conventions only (bootstrap content, skill layout, MCP merge rules).
2. **CLI-primary** — `skillgrid-cli` is the recommended way to install and manage plugins. Plugins can self-manage when the CLI isn't available.
3. **Full workflow enforcement** — plugins inject a bootstrap meta-skill at session start that enforces the Skillgrid workflow (1% rule, OpenSpec pipeline).
4. **Config-driven** — `config.d/` YAML remains the single source of truth for MCP servers and tool installs. Plugins read from it.

### Directory Structure

```
skillgrid/
├── .agents/skills/              # Source of truth for skills
├── plugins/                     # All editor plugins
│   ├── opencode/
│   │   ├── plugin.json          # Manifest
│   │   └── skillgrid.js         # Entry point (hooks)
│   ├── kilo/
│   │   ├── package.json         # npm package definition
│   │   ├── plugin.json          # Manifest
│   │   └── src/
│   │       └── index.ts         # Plugin entry (exports Plugin function)
│   └── cursor/
│       ├── plugin.json          # Manifest
│       ├── hooks/
│       │   └── hooks-cursor.json
│       └── bootstrap.md         # Injected on session start
├── config.d/                    # MCP/tool config (single source of truth)
├── skillgrid-cli/               # Go CLI
└── docs/specs/                  # Design specs
```

## Plugin Capabilities

All three plugins deliver the same capabilities, adapted to their platform:

| Capability | Mechanism |
|------------|-----------|
| Install skills | Register the plugin's skills directory in the editor's skill discovery path |
| Inject bootstrap | Inject meta-skill content at session start (method varies by platform) |
| Register MCP | Read `config.d/*.yaml`, merge into editor's native MCP config |
| Register slash commands | Install command files into the editor's command directory |
| Workflow enforcement | Bootstrap meta-skill enforces 1% rule + OpenSpec pipeline |

## Bootstrap Meta-Skill

The bootstrap is injected at session start on every platform. Content:

```
You are running with Skillgrid. Before any task:

1. SKILL CHECK (1% Rule): If there is even a 1% chance a skill applies,
   invoke it. Check available skills before responding.

2. WORKFLOW ENFORCEMENT:
   - New feature or design work → invoke brainstorming skill FIRST.
     No code before design approval.
   - Bug fix or unexpected behavior → invoke systematic-debugging skill.
   - Multi-step implementation → invoke writing-plans skill.
   - Read openspec/config.yaml for project-specific workflow rules.

3. OPENSPEC PIPELINE: Follow the project's declared workflow:
   proposal → specs → design → adr → tasks

4. PROJECT CONFIG: Read AGENTS.md and openspec/config.yaml at session
   start for project conventions and rules.
```

## Platform Integrations

### OpenCode Plugin

**Distribution**: Plugin specifier in `opencode.json`:
```json
{
  "plugin": ["skillgrid@git+https://github.com/ORG/skillgrid.git"]
}
```
Replace `ORG` with the GitHub organization or username that owns the skillgrid repo.

**Entry point**: `plugins/opencode/skillgrid.js` within the cloned repo. OpenCode's plugin system loads the JS module from this path after cloning the repo referenced in the specifier.

**Hooks used**:
- `config(config)` — adds the plugin's skills directory to `config.skills.paths`
- `experimental.chat.messages.transform` — prepends bootstrap to the first user message of each session

**Self-management**: Reads `config.d/*.yaml` from the repo, merges MCP servers into runtime config at startup.

**Implementation note**: Verify OpenCode's plugin discovery path. OpenCode may expect the plugin at `.opencode/plugins/<name>.js` by convention. If so, either follow that convention or document how to configure a custom entry point path in the specifier.

### Kilo Plugin

**Distribution**: npm package installed via `kilo plugin skillgrid-kilo-plugin`. The CLI runs this command.

**Package structure**: The `plugins/kilo/` directory IS the npm package. It contains its own `package.json` with the plugin entrypoint declared. Publishing to npm is done from this directory.

**Entry point**: `plugins/kilo/src/index.ts`

**Hooks used**:
- `config(config)` — inspect current config at startup
- `experimental.chat.system.transform` — append bootstrap to system prompt array
- `event({ event })` — listen for `session.created` to trigger setup
- `tool` (optional) — register a `skillgrid` custom tool for status/info

**Plugin context** provides: `project`, `directory`, `worktree`, `$` (Bun shell), `client` (Kilo SDK), `serverUrl`.

**npm package**: Published as `skillgrid-kilo-plugin` from the `plugins/kilo/` directory. `package.json` declares the plugin entrypoint via the `kilo.plugin` field.

### Cursor Plugin

**Distribution**: Cursor marketplace or `/add-plugin skillgrid`.

**Entry point**: `plugins/cursor/plugin.json` manifest.

**Hooks used**:
- `hooks-cursor.json` — inject bootstrap on session start
- Skills directory declared in manifest

**Bootstrap injection**: Via hook on session start event.

## CLI Integration

The `skillgrid-cli` provides the primary installation UX:

```bash
# Install for a specific editor
skillgrid install --target=opencode
skillgrid install --target=kilo
skillgrid install --target=cursor

# Auto-detect and install for all detected editors
skillgrid install

# Sync skills + MCP config after install
skillgrid sync
```

**Install flow per target**:

| Target | CLI Action |
|--------|------------|
| `opencode` | Add plugin specifier to `opencode.json` |
| `kilo` | Run `kilo plugin skillgrid-kilo-plugin` (installs npm package) |
| `cursor` | Print instructions to run `/add-plugin skillgrid` in Cursor |

**Auto-detection**: CLI checks for `opencode`, `kilo`, `cursor` binaries in `$PATH`.

## MCP Server Registration

Plugins read from `config.d/*.yaml` (single source of truth) and merge into the editor's native MCP config:

| Platform | MCP config location | Merge behavior |
|----------|---------------------|----------------|
| OpenCode | `opencode.json` → `mcp` | Merge at startup; `skillgrid-` prefix avoids collisions |
| Kilo | `kilo.json` → `mcp` | Merge at startup; `skillgrid-` prefix avoids collisions |
| Cursor | Cursor settings → MCP | Merge at startup; `skillgrid-` prefix avoids collisions |

Existing entries are preserved. Skillgrid-managed entries are prefixed with `skillgrid-` for identification and clean removal.

## Slash Commands

Plugins install slash commands into the editor's command directory:

| Platform | Command directory | Command files |
|----------|-------------------|---------------|
| OpenCode | `.opencode/commands/` | `.md` files with frontmatter |
| Kilo | `.kilo/command/` | `.md` files with frontmatter |
| Cursor | Cursor commands | TBD — investigate Cursor command format during implementation |

Commands provided:
- `/skillgrid-status` — show plugin status (version, skills count, MCP servers)
- `/skillgrid-sync` — re-sync skills and MCP config from `config.d/`
- `/skillgrid-update` — update the plugin to latest version

## Workflow Diagram

```
User runs: skillgrid install --target=kilo
         │
         ▼
CLI detects kilo binary in PATH
         │
         ▼
CLI runs: kilo plugin skillgrid-kilo-plugin
         │
         ▼
Kilo installs npm package, writes to kilo.json
         │
         ▼
Next Kilo session starts
         │
         ▼
Plugin loads → config hook fires
         │
         ├── Registers skills directory
         ├── Merges MCP servers from config.d/
         └── Installs slash commands
         │
         ▼
Session starts → system.transform hook fires
         │
         ▼
Bootstrap meta-skill injected into system prompt
         │
         ▼
Agent now enforces Skillgrid workflow
```

## Testing Strategy

1. **Unit tests**: Each plugin's MCP merge logic, bootstrap generation, and config parsing tested in isolation.
2. **Integration tests**: Install plugin in a test project, start a session, verify bootstrap is injected and skills are discoverable.
3. **Acceptance test**: "Let's make a react todo list" — agent must auto-trigger brainstorming skill before writing code.

## Open Questions

1. **Kilo npm package name**: `skillgrid-kilo-plugin` vs `@skillgrid/kilo-plugin` — depends on npm org availability.
2. **Cursor marketplace publishing**: Requires Cursor developer account and review process.
3. **Plugin update mechanism**: How plugins check for and apply updates (poll repo? npm update? CLI-triggered?).
4. **Multi-editor projects**: If a project uses both Kilo and Cursor, do plugins coexist? (Yes — independent configs.)
5. **OpenCode discovery path**: Does OpenCode's plugin loader look for a fixed directory (e.g., `.opencode/plugins/`) or is the entry point configurable? Affects where the plugin file must live.
6. **Kilo `package.json` field**: Verify the exact field name Kilo uses to declare the plugin entrypoint (assumed `kilo.plugin` — confirm against `@kilocode/plugin` types).
7. **Cursor bootstrap injection**: Confirm the hook name/event in `hooks-cursor.json` that fires on session start for content injection.

## Out of Scope

- Plugin for Claude Code / Codex / Antigravity (future work)
- GUI for plugin management
- Plugin-to-plugin communication
- Centralized plugin registry or analytics
