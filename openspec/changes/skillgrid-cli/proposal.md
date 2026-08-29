## Why

The `skillgrid-cli` is the primary installation method for Skillgrid, but its behavior, flags, and install pipeline are only documented in source code. A formal OpenSpec document ensures operators and contributors have a single source of truth for what the CLI does, its flags, its install sequence, and its configuration.

## What Changes

- Document the existing `skillgrid-cli` behavior in OpenSpec format
- Describe all commands (`install`, `sync-repo`), flags, and their semantics
- Document the install pipeline steps and their ordering
- Document agent selection, tool installation, and config system
- Add MCP server configuration as part of the normal install pipeline
- Implement MCP configuration during install for selected agents

## Capabilities

This change documents the existing CLI and implements MCP configuration during install. The capabilities below name the spec files describing each documented area, so the proposal and specs stay in 1:1 agreement:
`install-command`, `sync-repo`, `agent-selection`, `tool-install`, `config-system`,
`build`, `mcp-install`, `plugin-install`.

### New Capabilities

- `mcp-install`: MCP server configuration is performed automatically for selected agents during the normal install pipeline.

### Modified Capabilities

- `install-command`: The install pipeline now includes MCP configuration for selected agents.

## Impact

- **Affected docs**: `openspec/changes/skillgrid-cli/` is the canonical CLI reference
- **Affected code**: `skillgrid-cli/internal/install/install.go` — MCP setup added to install pipeline
- **Users**: Operators and contributors gain a formal reference for CLI behavior; MCP is now configured automatically during install
