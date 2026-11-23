---
type: Entity
title: "mnemonic (local-first memory binary)"
---

# mnemonic (local-first memory binary)

A new local-first persistent memory for AI agents — one Go 1.26 binary named
`mnemonic`, one SQLite file per project, two front doors: **CLI** and **MCP
stdio** over one shared handler. Local-only, no daemon, no cloud, no external
vector database.

- **Home:** `skillgrid-skills/mnemonic/` in **this repo** (programmer decision
  2026-11-23 — not a new sibling repo, not the old `../skillgrid/mnemonic`
  package evolved in place).
- **Change spec:** `.skillgrid/change-specs/2026-11-23-mnemonic-core/`
  (`brainstorm.md` pre-decision record, `briefing.md` the clean plan, status
  draft).
- **Predecessors studied:** [[sources/skillgrid-mnemonic-v2]] (data model +
  seams), [[sources/engram]] (product shape).
- **v1 scope (locked):** memory core only — sessions, observations, FTS5,
  in-file vectors, hybrid recall, trails, ops. Deferred (named): `code` tool,
  `web` tool, `serve`, projectwiki, `team_*`, LLM curator, old-store importer,
  cloud, TUI.
- **CLI shape (locked):** multi-tool binary — `mnemonic <tool> <verb>`; v1
  tool is `memory` (`mnemonic memory save|search|get|context|inject|…`);
  `mnemonic memory mcp` runs the stdio MCP server (`mnemonic-memory`, 12
  tools); top-level `mnemonic init` / `mnemonic doctor` are cross-tool.
- **Stack (locked):** `modernc.org/sqlite` (pure Go) + FTS5 + in-file vectors
  (BLOB + vec0 + brute-force floor); default embedder = local ONNX CPU model
  (C/CGo-free target via `unum-cloud/onnx`, spike-gated), API embedder
  opt-in (any OpenAI-compatible server, e.g. local Ollama), hash-64 floor.
- **Success criteria:** 9 numbered criteria in the briefing (persistence,
  semantic recall, degradation, upsert, K6 parity, single file, speed, ops,
  offline default).
- **Design hub:** [[syntheses/mnemonic-core]] · architecture (draft):
  [[architecture/mnemonic-memory-core]]
