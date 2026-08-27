## Context

Skillgrid is a configuration hub for AI-assisted development. It provides:
- `.agents/skills/` — 18+ reusable skills (adapted from Superpowers/BMAD)
- `config.d/` YAML — single source of truth for MCP servers and tool installs
- `skillgrid-cli` (Go) — installs skills, merges MCP configs, manages tool installs
- OpenSpec workflow (`proposal → specs → design → adr → tasks`)

Users work across three editors: **OpenCode**, **Kilo**, and **Cursor**. Each has a native plugin system but different integration mechanisms:
- OpenCode: JS plugin with `config` + `experimental.chat.messages.transform` hooks
- Kilo: TypeScript plugin via npm package, hooks for `config`, `event`, `experimental.chat.system.transform`
- Cursor: Plugin manifest (`plugin.json`) + `hooks-cursor.json` lifecycle hooks

The reference model is [obra/superpowers](https://github.com/obra/superpowers) — a single repo with platform-specific integration directories sharing a common `skills/` directory.

## Goals / Non-Goals

**Goals:**
- Full Skillgrid integration in each editor: skills, MCP servers, slash commands, workflow enforcement
- Plugins live in-repo under `plugins/` (independent, no shared runtime code)
- CLI is the primary installation method (`skillgrid install --target=<editor>`)
- Plugins self-manage when CLI isn't available (read `config.d/`, register skills, merge MCP)
- Bootstrap meta-skill injected at session start enforces 1% rule + OpenSpec pipeline

**Non-Goals:**
- Plugins for Claude Code / Codex / Antigravity (future work)
- GUI for plugin management
- Plugin-to-plugin communication
- Centralized plugin registry or analytics
- Replacing the CLI — plugins complement it

## Decisions

### 1. Plugin Architecture: Independent vs Shared Core

**Decision:** Independent plugins under `plugins/`. No shared runtime code. Shared conventions only (bootstrap content, skill layout, MCP merge rules).

**Alternatives considered:**
- Shared core (`plugins/core/` with common logic, thin platform adapters) — rejected because the platform-specific surface is small and coupling risks breaking all platforms on a core bug
- Separate repos per plugin — rejected because it fragments the source of truth and complicates keeping skills in sync

**Rationale:** Each plugin does the same thing (install skills, register MCP, inject bootstrap) but through different editor APIs. Independent plugins match the Superpowers model and allow per-platform optimization without cross-contamination.

### 2. Bootstrap Injection Mechanism

**Decision:** Each platform uses its native session-start hook to inject a bootstrap meta-skill.

| Platform | Hook | Injection point |
|----------|------|-----------------|
| OpenCode | `experimental.chat.messages.transform` | First user message |
| Kilo | `experimental.chat.system.transform` | System prompt array |
| Cursor | `hooks-cursor.json` session-start | Agent context |

**Alternatives considered:**
- Relying on `AGENTS.md` alone — rejected because it doesn't enforce the 1% rule or make skills discoverable via the editor's skill tool
- Reading `SKILL.md` files at session start — rejected because it burns tokens and doesn't scale

**Rationale:** Native hooks are the Superpowers-proven pattern. They inject exactly once per session and work with the editor's skill discovery mechanism.

### 3. CLI as Primary Installer

**Decision:** `skillgrid install --target=<editor>` detects the editor and runs platform-specific setup. Plugins can self-manage afterward.

**Alternatives considered:**
- CLI-only (no native plugins) — rejected because it can't inject bootstrap at session start or register skills in the editor's discovery path
- Plugins-only (no CLI role) — rejected because auto-detection and setup is complex; CLI provides a reliable entry point

**Rationale:** CLI handles detection and initial config; plugins handle runtime integration. This matches the user's requirement that "CLI should be usable as the main method" and "plugins do everything they can."

### 4. Kilo Plugin Distribution

**Decision:** Published as npm package (`skillgrid-kilo-plugin`). Installed via `kilo plugin skillgrid-kilo-plugin`.

**Alternatives considered:**
- `file://` path to repo — rejected because it requires the repo to be cloned locally and doesn't support updates
- Bundled in CLI binary — rejected because it couples release cycles

**Rationale:** npm is Kilo's native distribution mechanism. The `plugins/kilo/` directory IS the npm package.

## Risks / Trade-offs

- **Platform API changes** -> Mitigation: Independent plugins limit blast radius; only the affected platform breaks. Pin plugin API versions where possible.
- **npm package name collision** -> Mitigation: Check availability before publishing; fallback to `@skillgrid/kilo-plugin` scope.
- **Cursor marketplace rejection** -> Mitigation: Cursor plugin works without marketplace (manual install); marketplace is discoverability, not functionality.
- **Bootstrap token overhead** -> Mitigation: Bootstrap is ~50 tokens, injected once per session. Cache where the platform allows.
- **Multi-editor conflicts** -> Mitigation: Each editor has independent config. Plugins use `skillgrid-` prefix on MCP entries to avoid collisions.

## Migration Plan

1. Create `plugins/` directory structure with all three plugins
2. Implement OpenCode plugin first (simplest — single JS file, git-based distribution)
3. Implement Kilo plugin second (npm package, requires publishing)
4. Implement Cursor plugin third (marketplace publishing is longest pole)
5. Extend `skillgrid-cli` with `install` and `sync` subcommands
6. Test each plugin end-to-end: install → session start → bootstrap injection → skill discovery → MCP availability
7. Archive the change via `openspec archive editor-plugins`

No rollback needed — this is additive. Removing a plugin is uninstalling it from the editor's config.

## Open Questions

1. **OpenCode discovery path**: Does OpenCode's plugin loader look for a fixed directory (e.g., `.opencode/plugins/`) or is the entry point configurable? Affects where the plugin file must live.
2. **Kilo `package.json` field**: Verify the exact field name Kilo uses to declare the plugin entrypoint (assumed `kilo.plugin` — confirm against `@kilocode/plugin` types).
3. **Cursor bootstrap injection**: Confirm the hook name/event in `hooks-cursor.json` that fires on session start for content injection.
4. **Cursor command format**: TBD — investigate Cursor command format during implementation.
5. **Plugin update mechanism**: How plugins check for and apply updates (poll repo? npm update? CLI-triggered?).
