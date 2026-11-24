# Phase 3 is a projectwiki pillar with separate tables

---
status: "accepted"
date: 2026-11-24
supersedes: none
---

## Context and Problem Statement

Mnemonic Phase 3 adds the knowledge graph for `.skillgrid/projectwiki`: the engine indexes the project's wiki vault and serves it via CLI + MCP. Three shapes were on the table: a wiki pillar with its own table family, an in-DB knowledge graph with markdown as export, and a single unified node space across memory observations + code symbols + wiki pages.

## Considered Options

- Wiki pillar: `projectwiki_*` table family in `<repo>.db`, cross-domain references to Phase 1 memory (`sync_id`) and Phase 2 code (pin anchors) — separate tables, no shared node space
- In-DB knowledge graph: DB is source of truth, markdown is an export
- Unified cross-domain graph: observations, code symbols, and wiki pages as nodes of one graph

## Decision Outcome

Chosen option: "Wiki pillar with separate tables", because it keeps the skillgrid vault contract and v2 lineage intact, follows the Phase 1/2 `platform/registry.Subsystem` pattern (subsystem = "projectwiki", `<repo>.db` only), and cross-domain joins remain zero-overhead SQL via `sync_id` / code-symbol references. A unified node space would force unifying three ID schemes (obs, code symbol, page) before any value is delivered.

Naming (user-confirmed): CLI `mnemonic projectwiki <sub>`, MCP server `mnemonic-projectwiki`, table prefix `projectwiki_*` — v2-compatible.

### Consequences

- Good, because v2 schema lineage, the skillgrid ecosystem vocabulary, and the agent-owns-vault convention carry over unchanged
- Good, because each family's schema (memory / code / wiki) evolves independently
- Bad, because cross-domain traversal (observation → page → code symbol) is a join path, not a single graph hop; a "whole knowledge graph" view is a projection, not the store
