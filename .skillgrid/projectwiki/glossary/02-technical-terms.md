# Technical glossary

Architecture, platform, protocol, and design-pattern terms for the mnemonic project. One row per term.

| Term | Definition | Use When | Avoid |
| --- | --- | --- | --- |
| derived index | The DB copy of the vault: body mirror + hashes, FTS, graph tables, vectors. Always rebuildable from the vault, never hand-edited. | When discussing what lives in `<repo>.db` vs the vault. | "cache" (implies discardable), "mirror" (implies 1:1 only) |
| separate-table family | The `projectwiki_*` tables in `<repo>.db` alongside `observations` and `code_*` with no unified node space; cross-domain references via `sync_id` and code anchors. | When scoping schema changes or cross-domain joins. | "unified graph", "shared node space" |
| Synthesizer | The optional LLM seam (interface) that writes/updates page bodies from resolved facts; nil → deterministic cited-entry floor. | When discussing `projectwiki generate` behavior and degradation. | "LLM writer", "generator" (generator = the whole pipeline) |
| pin-check | Validation that resolves page frontmatter anchors (`symbol`, `line+version`, `all_of`) against the live Phase 2 code graph and marks pages `fresh|stale|unresolved` (fail-closed). | When discussing page staleness and `projectwiki validate`. | "verification"; "staleness check" (staleness is the state, pin-check is the act) |
