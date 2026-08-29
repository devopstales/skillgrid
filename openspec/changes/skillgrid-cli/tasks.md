# Tasks: skillgrid-cli Documentation and MCP Install

## Epic 1: Documentation complete

- [x] 1-1 Document install command and pipeline
- [x] 1-2 Document sync-repo command
- [x] 1-3 Document agent selection (interactive + preset)
- [x] 1-4 Document tool installation and agents copy
- [x] 1-5 Document config system and flags
- [x] 1-6 Document Taskfile build tasks (build / all / test / install / version)

## Epic 2: MCP configuration in install pipeline

- [x] 2-1 Add MCP packages install step from `config.d/tools.yaml`
- [x] 2-2 Add MCP server configuration from `config.d/mcp.yaml`
- [x] 2-3 Add tests for MCP packages and server configuration
- [x] 2-4 Validate with `openspec validate skillgrid-cli --type change --strict`

## Epic 3: Archive

- [ ] 3-1 Archive via `openspec archive skillgrid-cli`
