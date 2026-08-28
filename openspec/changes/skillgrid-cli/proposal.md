## Why

The `skillgrid-cli` is the primary installation method for Skillgrid, but its behavior, flags, and install pipeline are only documented in source code. A formal OpenSpec document ensures operators and contributors have a single source of truth for what the CLI does, its flags, its install sequence, and its configuration.

## What Changes

- Document the existing `skillgrid-cli` behavior in OpenSpec format
- Describe all commands (`install`, `sync-repo`), flags, and their semantics
- Document the install pipeline steps and their ordering
- Document agent selection, tool installation, and config system
- No code changes — this is a documentation-only change

## Capabilities

This change documents the existing CLI. The capabilities below name the spec files
describing each documented area, so the proposal and specs stay in 1:1 agreement:
`install-command`, `sync-repo`, `agent-selection`, `tool-install`, `config-system`,
`build`, `mcp-install`, `plugin-install`.

### New Capabilities

None — this change documents existing behavior, it introduces no new capability.

### Modified Capabilities

None — documentation only; no spec-level behaviour changes.

## Impact

- **Affected docs**: `openspec/changes/skillgrid-cli/` is the canonical CLI reference
- **Affected code**: None (documents the implemented `skillgrid-cli/` in this repo)
- **Users**: Operators and contributors gain a formal reference for CLI behavior
