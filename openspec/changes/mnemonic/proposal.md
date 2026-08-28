## Why

Skillgrid today depends on two external indexers (Engram for memory, GitNexus/ccc for code search) with overlapping "find things in the repo" surface area. Operators must install, configure, and teach agents which tool for which question. External tools also violate skillgrid's single-binary reproducibility goal. MemIndex consolidates memory, code search, and web research caching into one local-first SQLite store inside `skillgrid-cli`.

## What Changes

- Add `internal/memindex/` subsystem to `skillgrid-cli` with service, store, memory, codeindex, webcache, search, http, and mcp packages
- Add `skillgrid mcp` command (MCP stdio server for agent tool calls)
- Add `skillgrid serve [port]` command (HTTP API for OpenCode/Kilo plugins)
- Add `skillgrid index [path]` command (incremental code indexing)
- Add `config.d/indexing.yaml` for index configuration
- Add `plugins/memindex/` with OpenCode plugin, Kilo plugin copy, Cursor rule, and shared memory protocol
- Add `skillgrid setup opencode|kilocode|cursor` commands
- Fork `engram-memory` skill → `memindex-memory`
- Add `web_cache_*` tools for caching remote MCP research results
- Inject Mnemonic Memory Protocol into AGENTS.md (matching engram protocol block pattern)
- Integrate mnemonic tools into skillgrid skills (brainstorming, openspec-apply-change, openspec-explore, project-context, spec-as-source)

## Capabilities

### New Capabilities

- `memory-layer`: Engram-style curated observations with FTS5 search (`mem_*` tools)
- `code-layer`: Incremental file/chunk indexing with FTS5 search (`code_*` tools)
- `web-cache-layer`: Local snapshots of research MCP results with TTL + dedup (`web_*` tools)
- `mcp-transport`: MCP stdio server (`skillgrid mcp`) for agent tool calls
- `http-api`: HTTP REST API (`skillgrid serve`) for OpenCode/Kilo plugin hooks
- `agent-plugins`: OpenCode/Kilo plugins for session lifecycle, Memory Protocol injection, compaction recovery
- `agents-rules`: Mnemonic protocol block injected into AGENTS.md (matching engram protocol pattern)
- `skill-integration`: Mnemonic tool references integrated into skillgrid skills (brainstorming, openspec-apply-change, etc.)
- `vscode-plugin`: VSCode extension for skillgrid integration (skills, MCP servers, commands)

### Modified Capabilities

- `engram-memory`: Fork to `memindex-memory` (same protocol; tool names unchanged; add `web_*` rules)
- `exa-search`: Add "cache results via `web_cache_save`" step after search
- `mcp-deepwiki`: Add cache-first lookup guidance

## Impact

- **Affected code**: New `internal/memindex/` packages, new `cmd/mcp.go`, `cmd/serve.go`, `cmd/index.go`, new `plugins/memindex/` directory
- **Affected systems**: SQLite store at `~/.skillgrid/memindex/`, HTTP server on `127.0.0.1:7438`
- **Dependencies**: `github.com/mark3labs/mcp-go` (MCP library), `modernc.org/sqlite` (SQLite driver)
- **Users**: All skillgrid users — replaces external Engram + GitNexus in default profile
