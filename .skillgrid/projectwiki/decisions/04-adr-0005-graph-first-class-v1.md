# Knowledge graph is first-class and queryable in v1

---
status: "accepted"
date: 2026-11-24
supersedes: none
---

## Context and Problem Statement

The Phase 3 pillar stores a knowledge graph of the vault. Two dimensions were on the table: the node model (lean 3 kinds vs rich 6 kinds) and whether the graph is exposed as a query target in v1 (traversal tools) or only via page-level tools (v2 surface).

## Considered Options

- Lean model (page/claim/source) + v2 surface only — the graph is an internal join structure
- Rich model (page, section, claim, source, term, ADR) + v2 surface only
- Rich model + graph traversal tools (neighborhood/impact/path) in v1

## Decision Outcome

Chosen option: "Rich model + graph traversal tools in v1", because the user picked both the rich node model and the 1:1+traversal surface: the graph is the pillar's reason to exist, and hiding it behind page-level tools would deliver the schema without the capability. The six node kinds: page, section, claim, source (external cite | memory `sync_id` | code anchor), term (glossary), ADR (with `supersedes` edges).

### Consequences

- Good, because per-section grounding, term networks, and ADR supersession trails are queryable from day one
- Good, because the edge tables are properly indexed — they are a query target, not just join fodder
- Bad, because the parse surface is larger (sections, term tables, ADR frontmatter) and v1 ships three new MCP tools beyond v2 parity
