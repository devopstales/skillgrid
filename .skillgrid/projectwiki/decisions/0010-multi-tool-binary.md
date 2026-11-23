# Multi-tool binary: `mnemonic <tool> <verb>`, per-tool MCP servers

---
status: accepted
date: 2026-11-23
supersedes: none
---

## Context and Problem Statement

The programmer interjected (2026-11-23): "mnemonic will contain multiple
tools in the future. so put cli under `mnemonic memory` — so
`mnemonic memory mcp`." The binary is a **home for multiple memory tools**
(`memory` now; `code`, `web`, … later). The CLI hierarchy and the MCP
server naming have to reflect that from day one, not be refactored after
the second tool ships.

## Considered Options

- **Flat CLI** (`mnemonic save`, `mnemonic mcp`): clean while one tool
  exists; every future tool is a rename or a namespace collision.
- **`mnemonic <tool> <verb>`**: one verb per tool under the tool's
  namespace; top-level `mnemonic` holds only cross-tool commands (`init`,
  `doctor`); the MCP server is per-tool (`mnemonic memory mcp` → server
  `mnemonic-memory`).
- **Separate binary per tool** (`mnemonic-memory`, `mnemonic-code`): max
  isolation, but the shared store/config/data-dir story gets duplicated per
  binary and the "one local brain" story fragments.

## Decision Outcome

Chosen option: **`mnemonic <tool> <verb>`** — v1: `mnemonic memory
save|search|get|context|inject|stats|delete|pin`, `mnemonic memory session
start|end|summary`, `mnemonic memory mcp [--debug]` (stdio server
`mnemonic-memory`), `mnemonic memory export`; top-level `mnemonic init` and
`mnemonic doctor` are cross-tool (data dir + health of every registered
tool). Each future tool registers its own commands and its own MCP server;
a unified `mnemonic mcp` across tools is an open question, not a v1
promise.

### Consequences

- Good: the second tool (`code`) is additive — a new package + namespace,
  no renames, no CLI migration, no MCP server rename.
- Good: agents wire exactly the tool they need (`mnemonic-memory` only),
  not a superset server.
- Good: cross-tool concerns (`init`, `doctor`, data dir, config) live at
  the top level where they belong.
- Bad: one verb is two levels deep (`mnemonic memory save` vs
  `mnemonic save`) — a permanent small cost, accepted.
- Bad: `doctor` must discover registered tools (a small registry) instead
  of hardcoding checks.

## Not adopted

- **Flat CLI** — works for one tool, taxes every future tool.
- **Per-tool binaries** — fragments the single-brain story the whole design
  rests on ([[decisions/0003-per-project-sqlite-file]]).

## Related

- Supersedes: none.
- Depends on: [[decisions/0007-single-handler-two-adapters]] (each
  tool's adapters share its Handler), [[preferences/build-preferences]]
  (the multi-tool preference this records).
- Consumed by: [[entities/mnemonic]] (the CLI shape line),
  [[syntheses/mnemonic-core]], [[decisions/0008-memory-core-only-v1]]
  (where deferred tools land).
