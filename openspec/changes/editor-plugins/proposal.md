## Why

Skillgrid provides a configuration hub for AI-assisted development — reusable skills, MCP server definitions, and an install script. But users work across multiple editors (OpenCode, Kilo, Cursor) and currently have no way to integrate Skillgrid's full workflow into their editor of choice. Without editor plugins, users must manually copy skills, merge MCP configs, and remember to follow the OpenSpec workflow — defeating the purpose of a configuration hub.

Editor plugins bring Skillgrid's full integration (skills, MCP servers, slash commands, workflow enforcement) directly into the editor via native plugin mechanisms. The CLI remains the primary installation method; plugins do everything they can natively within their editor.

## What Changes

- Add `plugins/opencode/` — OpenCode plugin (JS) that registers skills, injects bootstrap, merges MCP config
- Add `plugins/kilo/` — Kilo plugin (TypeScript/npm) that registers skills, injects bootstrap, merges MCP config
- Add `plugins/cursor/` — Cursor plugin (manifest + hooks) that registers skills, injects bootstrap, merges MCP config
- Add `skillgrid install --target=<editor>` command to the CLI for plugin installation
- Add `skillgrid sync` command to push latest skills/MCP config after install
- Define bootstrap meta-skill content that enforces the Skillgrid workflow (1% rule + OpenSpec pipeline)
- Each plugin reads `config.d/` as the single source of truth for MCP servers

## Capabilities

### New Capabilities

- `opencode-plugin`: OpenCode editor integration — skill registration, bootstrap injection, MCP merge, slash commands via native JS plugin hooks
- `kilo-plugin`: Kilo editor integration — skill registration, bootstrap injection, MCP merge, slash commands via native TypeScript plugin hooks
- `cursor-plugin`: Cursor editor integration — skill registration, bootstrap injection, MCP merge, slash commands via native plugin manifest + hooks

### Modified Capabilities

None — no existing spec-level behavior changes. This is a new subsystem.

## Impact

- **Affected code**: New `plugins/` directory at repo root; `skillgrid-cli` gains `install` and `sync` subcommands
- **Affected systems**: OpenCode plugin registry (git-based), Kilo npm registry (npm package), Cursor marketplace (future)
- **Dependencies**: Each plugin depends on its editor's native plugin API (`@kilocode/plugin` for Kilo, OpenCode plugin SDK for OpenCode, Cursor plugin format for Cursor)
- **Users**: Skillgrid users who work in OpenCode, Kilo, or Cursor gain one-command setup for full editor integration
