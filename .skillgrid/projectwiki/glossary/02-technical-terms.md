# Technical glossary

Architecture, platform, protocol, and design-pattern terms for the mnemonic project. One row per term.

| Term | Definition | Use When | Avoid |
| --- | --- | --- | --- |
| derived index | The DB copy of the vault: body mirror + hashes, FTS, graph tables, vectors. Always rebuildable from the vault, never hand-edited. | When discussing what lives in `<repo>.db` vs the vault. | "cache" (implies discardable), "mirror" (implies 1:1 only) |
| separate-table family | The `projectwiki_*` tables in `<repo>.db` alongside `observations` and `code_*` with no unified node space; cross-domain references via `sync_id` and code anchors. | When scoping schema changes or cross-domain joins. | "unified graph", "shared node space" |
| Synthesizer | The optional LLM seam (interface) that writes/updates page bodies from resolved facts; nil → deterministic cited-entry floor. | When discussing `projectwiki generate` behavior and degradation. | "LLM writer", "generator" (generator = the whole pipeline) |
| pin-check | Validation that resolves page frontmatter anchors (`symbol`, `line+version`, `all_of`) against the live Phase 2 code graph and marks pages `fresh|stale|unresolved` (fail-closed). | When discussing page staleness and `projectwiki validate`. | "verification"; "staleness check" (staleness is the state, pin-check is the act) |
| graph node | One of the six kinds materialized in the `projectwiki_*` tables: page, section, claim, source, term, ADR (ADR-0005). | When discussing the knowledge graph's node model or edge tables. | "entity", "record" |
| promote floor | The deterministic generate path: qualifying observations (importance ≥ 0.5 + type map) become cited-entry page drafts; the only generate path in v1 (ADR-0004). | When discussing `projectwiki generate` behavior. | "LLM generate", "auto-write" |
| page protection | The rule that generate only writes pages flagged `origin=auto/draft`; curated pages are never touched — the fix for v2's generate-pollutes-curation gotcha. | When discussing generate safety on curated vaults. | "read-only mode" |
| grounded type | A page type that pins code via frontmatter anchors (decision, invariant, workflow, invariantcontract, businessworkflow); pin-check is fail-closed for these. | When deciding which pages participate in validation. | "pinned page" |
