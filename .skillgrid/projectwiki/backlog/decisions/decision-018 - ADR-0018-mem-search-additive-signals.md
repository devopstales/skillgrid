---
id: decision-018
title: "ADR-0018 mem_search returns additive per-signal scores; RRF runs only on the owner-scoped path"
date: "2026-10-02"
status: "accepted"
---

Source: `.skillgrid/artifacts/04-adr-0018-mem-search-additive-signals.md`

# mem_search returns additive per-signal scores; RRF runs only on the owner-scoped path

---
status: "accepted"
supersedes: none
date: 2026-10-02
---

## Context and Problem Statement

MCP `mem_search` calls `SearchOwnerScoped`, which is BM25 plus optional importance re-rank. `BlendedSearch` already fuses FTS and vector ranks with `ReciprocalRankFusion` (k=60), but it calls `SearchWithScope` and `SearchByVector`, neither of which applies `visibilityFilter`. Wiring that function into the MCP handler would let a reader see another owner's private observations. The comparison briefing also asks each hit to explain itself (`signals` + `matched_via`) and to rank frequently accessed observations higher. ADR-0017 is the D3 graph decision, so this record is 0018.

## Considered Options

- Call `BlendedSearch` from `handleMemSearch` and attach scores in the handler
- Add `SearchOwnerScopedBlend` that runs both legs under `visibilityFilter`, fuses with the existing RRF (k=60), applies reinforcement decay, and returns a `SearchHit` the DTO copies additively
- Leave `mem_search` BM25-only and put signals only on `mem_ask`

## Decision Outcome

Chosen option: "`SearchOwnerScopedBlend` under the existing visibility filter", because the private-observation boundary is already enforced on `SearchOwnerScoped` and must not be bypassed to gain a vector leg. `signals` and `matched_via` are additive JSON fields. With no embedder (`MNEMONIC_EMBED` unset) the vector leg is skipped, `matched_via` is `keyword`, and equal decay factors keep BM25 order. Reinforcement decay is Option A from the briefing: query-time, config `mnemonic.decay` default enabled, half-life 30 days, immunity at importance ≥ 4 or retrieval_usage ≥ 3. `edge_factor` is the constant 1 because the observation graph is out of scope. Days-since-access reads `LastSeenAt` only; an empty timestamp is 0 days so existing fixtures that never stamp `last_seen_at` keep their order.

### Consequences

- Good, because hybrid recall and a per-signal explanation land without opening private rows, and the no-embedder path stays a BM25 floor
- Bad, because result order changes when decay factors differ or an embedder is active; consumers that reject unknown JSON fields must be updated
