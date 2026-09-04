# Tasks: 004-hermes-memory — Step 01-facts-schema

> Goal: Facts + FTS + forgetting_events + sqlite-vec embeddings.
> Depends on: none

## Review Workload Forecast (change-level, present in step 01's tasks.md; carried by reference in others)
| Field | Value |
|---|---|
| Estimated changed lines (this step) | ~350–450 |
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

- [ ] 01.1 Create `skillgrid-cli/internal/mnemonic/store/migrations/011_facts_skills.sql` — `facts`, `facts_fts`, `fact_embeddings`, `forgetting_events`, `skills`, `skills_fts`, `skill_embeddings`, `skill_usage`, plus trail columns for fact/skill ids (after **003** `010_*`).
- [ ] 01.2 Create `skillgrid-cli/internal/mnemonic/vec/vec.go` — `vec.Index` Upsert/SearchKNN **Interface**; sqlite-vec loader + fake adapter; fail closed when extension missing.
- [ ] 01.3 Modify `skillgrid-cli/internal/mnemonic/store/store.go` — hook vec extension on Open when available; leave **003** Pure Go `path_embeddings` path untouched.
- [ ] 01.4 Create `skillgrid-cli/internal/mnemonic/facts/facts.go` — Fact Memory **Module** implementing Add/Search/Forget/Decay over SQL+FTS+vec **Seam**.
- [ ] 01.5 Cover WHAT: store open creates **Fact Memory** tables without rewriting observations; **Tiered Storage** rows intact after migrate.
- [ ] 01.6 Cover WHAT edge: re-open idempotent; missing sqlite-vec fails closed on vec ops only (non-vec open/search still works).
