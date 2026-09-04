# Tasks: 003-mnemonic-self-evolving-context-database — Step 01-schema-extensions

> Goal: Additive SQL for tier paths, long-term memories, trails, embeddings.
> Depends on: none

## Review Workload Forecast (change-level)
| Field | Value |
|---|---|
| Estimated changed lines (this step) | ~180 |
| Estimated changed lines (change) | ~1600–2200 |
| 400-line budget risk (change) | High |
| Chained PRs recommended (change) | Yes |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

## Execution

- [ ] 01.1 Create `skillgrid-cli/internal/mnemonic/store/migrations/010_tiered_context.sql` with `tiered_contents`, `long_term_memories`, `retrieval_trails`, `path_embeddings` (leave `009_*` for 001).
- [ ] 01.2 Add Tiered Storage / Semantic Search / Retrieval Trail terms to `docs/skillgrid/agents/glossary/technical.md`.
- [ ] 01.3 Add Long-term Memory to `docs/skillgrid/agents/glossary/business.md`.
- [ ] 01.4 Cover WHAT: store open creates tier/memory/trail/embedding tables without rewriting existing rows; observations/FTS/code index intact after migrate.
- [ ] 01.5 Cover WHAT edge: schema 008 → 010 once; re-open idempotent.
