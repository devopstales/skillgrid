---
type: Concept
title: "Single handler, two adapters (K6 seam)"
status: stable
sources:
  - ".skillgrid/change-specs/2026-11-23-mnemonic-core/briefing.md R8 (2026-11-23)"
  - "local: ../skillgrid/mnemonic/cmd/mnemonic/main.go (predecessor implementation)"
---

# Single handler, two adapters (K6 seam)

CLI and MCP are **two adapters over one shared `Handler`** — neither adapter
may contain domain logic. The CLI flag-parser and the MCP tool-dispatcher map
arguments 1:1 to Handler methods and render the same JSON envelope.

## Why it matters

- **Byte-identical parity** is the testable contract: for identical inputs,
  `mnemonic memory search …` and the MCP `mem_search` tool must produce
  byte-identical envelopes (golden comparison, not field-by-field). Parity
  tests are mandatory for every Handler method.
- **One behavior, two surfaces** — a bug fix or a semantic change lands once,
  in the Handler; both surfaces change together. The adapters can never
  silently diverge.
- **Envelopes are the contract:** `{"ok":…,"code":…,"data":…,"degraded":…}`
  with stable error `code` strings; exit codes 0 ok · 1 handler error ·
  2 usage (usage errors go to stderr with no envelope).

## Precedent

Inherited from [[sources/skillgrid-mnemonic-v2]] (the "K6 byte-identical
seam" — its CLI and MCP stdio server are two adapters over one Handler) and
from the projectwiki engine in this repo (`projectwiki_*` MCP tools over one
ProjectWikiHandler). The same seam will serve the future `code`/`web` tools
of [[entities/mnemonic]].

## Related

- Supersedes: none.
- Consumed by: [[syntheses/mnemonic-core]], [[architecture/mnemonic-memory-core]].
- Depends on: [[decisions/0007-single-handler-two-adapters]] (the locked
  decision), [[concepts/degradation-floor]] (envelope `degraded` flag).
