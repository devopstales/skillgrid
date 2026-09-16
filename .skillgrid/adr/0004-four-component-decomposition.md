# PRD component map is a 4-way decomposition

---
status: "accepted"
supersedes: none
date: 2026-09-16
---

## Context and Problem Statement

The PRD's system view needs a component map. The code has ~25 `internal/`
packages, but those are implementation units, not product components. The map
had to be decided because it shapes the Technical Specifications section and how
future changes reference "the installer" vs "the engine" vs "the surface."

## Considered Options

- **4 components** — Installer, Mnemonic Engine (memory + code-intel + search), Distribution Surface (MCP stdio + HTTP/REST + embedded UI + CLI), Hub Content (skills + hooks + config.d, shipped as repo)
- **3 components** — Installer, Mnemonic Engine, Hub Content; UI folded into the engine since it's embedded
- **5 components** — split Code Intelligence from Memory as separate components

## Decision Outcome

Chosen option: "4 components," because the Distribution Surface is a distinct
concern (three transports, each with different auth, embedding, and lifecycle
implications: stdio is spawned per-agent, HTTP is long-running and binds
127.0.0.1, the CLI is both a user surface and the installer). Folding the UI
into the engine (3-way) hides that the embedded SPA + OpenAPI/Swagger are a
serving concern, not a data concern. Splitting memory from code-intel (5-way)
over-decomposes: they share the same per-project SQLite store and the same
project-resolution layer, and the user experience of "memory + code
intelligence" is one thing.

### Consequences

- Good, because each component has one clear purpose and a testable boundary:
  the Installer writes to `~/.skillgrid/` + agent configs, the Engine owns the
  SQLite store, the Surface owns the transports, the Hub Content is shipped not
  run.
- Good, because the PRD's trust-boundary section maps cleanly: the one real
  boundary (MCP-spawned process with project read access) lives between the
  Surface and the Engine.
- Bad, because "Distribution Surface" is a new term that the code does not use
  (the code has `internal/mnemonic/mcp`, `internal/mnemonic/http`, and the CLI
  command group as separate packages); a glossary entry is required to keep the
  term from drifting.
