# ADR Index

Single source for the ADR in-force set. Each row points at the full record in `04-adr-NNNN-slug.md` (which carries the Context / Options / Decision / Consequences). Consumers read this index to compute what is **currently in force** without walking every ADR file.

**In force** = `status: accepted` AND no later ADR's `supersedes` names it. Superseded / deprecated ADRs stay in the table (frozen) but are marked out of force.

| # | Title | Status | Supersedes | Date | In force | Record |
|---|-------|--------|------------|------|----------|--------|
| 0001 | PRD scope is the whole Hub Product, not the binary alone | accepted | — | 2026-09-16 | yes | [04-adr-0001](04-adr-0001-prd-scope-whole-hub.md) |
| 0002 | PRD framing is engine-first, not installer-first | accepted | — | 2026-09-16 | yes | [04-adr-0002](04-adr-0002-prd-engine-first-framing.md) |
| 0003 | v1.0 line is "the engine is trustworthy"; UI is Phase 2 | accepted | — | 2026-09-16 | yes | [04-adr-0003](04-adr-0003-v1.0-line-engine-trustworthy.md) |
| 0004 | PRD component map is a 4-way decomposition | accepted | — | 2026-09-16 | yes | [04-adr-0004](04-adr-0004-four-component-decomposition.md) |
| 0005 | Trust boundary: MCP-spawned process with project read access | accepted | — | 2026-09-16 | yes | [04-adr-0005](04-adr-0005-trust-boundary-mcp-spawn-model.md) |
| 0006 | Vector search: in-memory brute-force cosine, no SQLite vector extension | accepted | — | 2026-09-16 | yes | [04-adr-0006](04-adr-0006-vector-search-in-memory-brute-force.md) |
| 0007 | SDD multi-method docs viewer with generic, existence-gated doc roots | accepted | — | 2026-09-17 | yes | [04-adr-0007](04-adr-0007-sdd-multi-method-docs-viewer.md) |
| 0008 | State drift guard: Node.js + `yaml` package, read-only verifier | accepted | — | 2026-09-24 | yes | [04-adr-0008](04-adr-0008-state-drift-guard-js-yaml.md) |
| 0009 | Vector search: in-SQL sqlite-vec latency corrected; viant deferred; revisit path confirmed | accepted | — | 2026-09-24 | yes | [04-adr-0009](04-adr-0009-vector-search-in-sql-latency-viant-deferred.md) |
| 0010 | Verification scope discriminator: a zero is never a bare zero (COMPLETE/TRUNCATED/UNSCOPED/UNREADABLE) | accepted | — | 2026-09-24 | yes | [04-adr-0010](04-adr-0010-verification-scope-discriminator.md) |

**Highest sequence in use:** 0010 (next ADR is `04-adr-0011-slug.md`).

## How to update

- New ADR: append a row, set the next sequence, link the record. Update the "Highest sequence in use" line.
- Supersession: the new ADR's row lists `Supersedes: 00NN`; flip the superseded row's **In force** to `no`. Never delete a row — the trail is the record.
