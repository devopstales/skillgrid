# Mnemonic — Understanding Record

## VERIFIED

- Mnemonic is a strictly local-first, zero-CGo Go persistent memory engine for AI agents: SQLite-only (`~/.mnemonic/global.db` + `~/.mnemonic/<repo>.db`), pure-Go `modernc.org/sqlite` + `vec0`, ONNX CPU embeddings (384-dim `bge-small-en-v1.5`), CLI + MCP + daemon handler parity, `internal/platform` kernel with pluggable subsystems (Phase 1 spec, approved).
- Phase 2 adds the `code` subsystem in `<repo>.db`: gotreesitter extraction, confidence-tiered edges (`LSP_RESOLVED|EXTRACTED|SYNTHESIZED|INFERRED|AMBIGUOUS`), FTS5 + name-segment + vec0 hybrid RRF search, CSR PageRank `code_explore`, `observation_code_links` joining memory to code (Phase 2 spec, approved for implementation planning).
- skillgrid-v2's `projectwiki` family (Phase 3 lineage): `projectwiki_pages/claims/run_state/page_baselines/events` + FTS in the project DB; frontmatter pin anchors (`symbol`, `line+version`, `all_of`) validated against the code graph; `memory:<sync_id>` / `code:<path>#Lx-Ly` source links; deterministic generate with the Synthesizer seam wired nil (known gotcha: generate pollutes a curated vault); CLI+MCP parity; ADR-0068 accepted but not landed (extract/digest/review/curate verbs, `skills/` zone).

## INFERRED

- Phase 3 ports the projectwiki family onto the new `internal/platform` kernel as the `projectwiki` subsystem, with cross-domain links materialized as tables instead of frontmatter strings.

## LOCKED

ADR index — in-force decisions. One row per decision; the body lives in the linked file. Highest sequence in use: 0003.

| ADR | Decision | In force | Record |
| --- | --- | --- | --- |
| ADR-0001 | Phase 3 = projectwiki pillar; separate `projectwiki_*` table family, no unified cross-domain graph | yes | [decisions/04-adr-0001-projectwiki-pillar-separate-tables.md](decisions/04-adr-0001-projectwiki-pillar-separate-tables.md) |
| ADR-0002 | Vault is source of truth; DB is derived, always-rebuildable index | yes | [decisions/04-adr-0002-vault-is-source-of-truth.md](decisions/04-adr-0002-vault-is-source-of-truth.md) |
| ADR-0003 | v1 generation = index/serve/validate/cite + LLM Synthesizer ("1+3") | yes | [decisions/04-adr-0003-llm-synthesizer-in-v1.md](decisions/04-adr-0003-llm-synthesizer-in-v1.md) |

## Locked constraints (inherited from v2)

- No new dependencies without an ADR.
- Spec-zone commits before code-zone commits.
- Serial development — one change at a time.
