---
type: TechnicalDecision
status: in-force
supersedes: none
date: 2026-11-02
---

# ADR-0043 — Code-index tree-sitter dependency (gotreesitter)


Change: `2026-10-31-mnemonic-v2-code-index` (spec zone: `adr.md` § ADR-0043)

## Context

The code-index's extraction ladder (TICKET-03) needs a parser for the 6 rich languages
(go/python/java/js/ts/tsx) + the map-tier languages (24 distinct maps + the `c`→`cpp`
alias — 25 data-table rows, the counting rule in spec `003-extraction`). v1 uses
`gotreesitter` (v0.55.1), the pure-Go binding (no CGo — the v1 VERIFIED cgo-free
stack). It is in `mnemonic_back/go.mod` but new to `mnemonic-v2/go.mod`.

## Decision

Add `github.com/odvcencio/gotreesitter v0.55.1` to `mnemonic-v2/go.mod`.

## Alternatives

- **`tree-sitter` (the C binding)** — rejected: adds CGo, breaks the cgo-free
  invariant (`skillgrid doctor` reports `cgo: free`).
- **Hand-rolled parsers** — rejected: the rich-6 + map-tier coverage is not
  hand-rollable; v1's ladder is the proven mechanism.

## Consequences

The rich-6 + map-tier ladder is liftable from v1 (`mnemonic_back/internal/extract/`).
A gotreesitter upgrade later is a re-test of the extraction ladder — the extractor
stamp (D-STAMP, content hash over the pinned input set) catches a behavior change and
forces a full re-walk. The dep lands at TICKET-03, gated by this ADR.
