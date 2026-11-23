# Single handler, two adapters (K6 parity seam)

---
status: accepted
date: 2026-11-23
supersedes: none
---

## Context and Problem Statement

[[entities/mnemonic]] has two front doors (CLI + MCP stdio). If each surface
carries its own argument mapping and result shaping, the two will drift:
different error handling, different truncation, different flags for the same
operation. The predecessor
([[sources/skillgrid-mnemonic-v2]]) and this repo's projectwiki engine both
solve it with one shared handler.

## Considered Options

- **Two independent surfaces** (MCP tools + CLI each call the store):
  fastest to start, drift guaranteed; parity is untested folklore.
- **MCP as the source of truth, CLI a thin client**: the CLI needs a server
  (or an in-process one) and a transport for a local tool.
- **One shared `Handler`; adapters map arguments 1:1 and render the same
  envelope**: logic lives once; parity is a byte-level test.

## Decision Outcome

Chosen option: **one shared `Handler`** — the `memory` domain exposes Handler
methods; the CLI flag-parser and the MCP dispatchers contain no domain logic;
both render the same JSON envelope (`{"ok","code","data","degraded",…}`) and
parity tests assert **byte-identical** payloads for matched inputs
(golden comparison, not field-by-field).

### Consequences

- Good: a semantic change lands once; both surfaces change together; the
  test suite proves it.
- Good: the envelope + stable error `code` strings + exit codes (0/1/2) are
  the machine-readable contract agents and scripts both rely on.
- Bad: the Handler interface is a design surface that must stay small and
  boring — adding a surface quirk to one adapter to "fix" a UX problem is
  the drift this ADR exists to prevent.
- Bad: parity tests must cover every Handler method (the test burden is the
  price).

## Not adopted

- **CLI-as-HTTP-client over the MCP server** — a process + protocol for a
  local binary; the in-process seam is strictly simpler.

## Related

- Supersedes: none.
- Depends on: [[concepts/single-handler-two-adapters]] (the full concept),
  [[sources/skillgrid-mnemonic-v2]] (predecessor implementation).
- Consumed by: [[architecture/mnemonic-memory-core]] (the layering),
  [[syntheses/mnemonic-core]], [[decisions/0010-multi-tool-binary]]
  (per-tool adapters over the same seam).
