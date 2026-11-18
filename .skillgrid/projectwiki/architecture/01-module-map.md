---
type: Architecture
title: "Module Map"
status: stable
sources:
  - "code:mnemonic/internal/transport/handler/handler_projectwiki.go"
---

# Module Map

The runtime is two Go modules in one workspace: the engine (`mnemonic/`) and the
installer (`skillgrid-cli/`). Everything else (skills, hooks, plugins) is Hub
Content shipped by the installer, not runtime code.

## Engine (`mnemonic/`)

- `cmd/mnemonic/` — hand-rolled dispatch (family-first: `memory` / `code` /
  `projectwiki`), MCP stdio entry points.
- `internal/core/store/` — SQLite (WAL) + embedded migrations + FTS5 + vec0.
- `internal/memory/` — observations, dedup, consolidation, importance.
- `internal/projectwiki/` — the wiki engine: bootstrap/index/reindex/query/
  validate/lint/status/cite-source/eval.
- `internal/codeindex/` — incremental indexer, graph, search, vectorstore, watch.
- `internal/intelligence/` — LLM/embedder seams with fail-open floors.
- `internal/transport/handler/` — the K6 seam: one handler set over MCP + CLI.
- `internal/transport/mcp/` — three single-purpose stdio servers.

## Installer (`skillgrid-cli/`)

- `cmd/skillgrid/` — installer shell.
- `internal/setup/`, `internal/ui/`, `internal/harnessconfig/` — staging + config merge.

## Hub Content (repo, shipped)

- `.agents/skills/` — the skill grid (see [[workflows/development-workflow]]).
- `plugins/`, `hooks/`, `git-hooks/`, `config.d/`, `rules/`.

## Why this matters

A change that adds a subsystem usually touches one engine package plus the handler
seam; the installer and Hub Content are separate worlds. See
[[decisions/04-adr-0004-four-component-decomposition]] for the 4-way decomposition
and [[decisions/04-adr-0033-three-module-workspace-ctx]] for the workspace split.
Locked build constraints live in [[invariants/05-locked-constraints]].

## Operational notes

- Stale installed binary masks fixes: [[gotchas/stale-installed-mnemonic-binary]].
- `projectwiki generate` runs degraded: [[gotchas/projectwiki-generate-has-no-synthesizer]].
- Fail-open floors permeate the seams: [[concepts/fail-open-floors]].
