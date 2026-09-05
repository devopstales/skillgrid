# Interview: 003-mnemonic-self-evolving-context-database

> User-gate revise via `questioning` (2026-09-05). Spec already written; grill before apply.

## Facts gathered (agent)

- Repo SDD-initialized; change `State.phase: spec` with full `tasks.md` + `acceptance.feature`; 0/5 steps PASS.
- **002 archived** and already ships observation embeddings + blended FTS/vector search behind `MNEMONIC_EMBED` (`memory/search_embed.go`, `SetEmbedding`, RRF tests).
- **003 code is absent** (verified): no `010_*.sql`, no `tiered/`, no `embedder/`, no `semantic_search` / `load_full_details` / `mnemonic_commit`, no `migrate --tier` / `trail` CLI.
- Legacy plan path `docs/plan/03-mnemonic-self-evolving-context-database.md` is missing from tree; `change.md` remains source of truth.
- Open question still listed in `change.md`: local embedder library (onnxruntime-go vs hash-embedder stub) — deferred to step 03.
- Spec↔stack tension to grill: new `semantic_search` + `path_embeddings` vs existing `mem_search` + observation embeddings; "tier-eligible" content without 001 teams briefs; 5-step / ~1600–2200 LOC / high 400-line PR risk.

## Round 1 — asked 2026-09-05

Q1 search surface · Q2 tier eligibility · Q3 delivery slice

**Answers (user 2026-09-05):** `1 3 1`

## Decisions

| ID | Decision | Choice |
|----|----------|--------|
| D1 | Search surface | **Separate** `semantic_search` + `load_full_details` (as written) |
| D2 | Tier eligibility without 001 | **Seam ready**; ship writers via `migrate --tier` + `mnemonic_commit` only; live content-write producers deferred |
| D3 | Delivery slice | **Keep all 01–05** in this change |

## Round 2 — asked 2026-09-05

Q4 seam proof · Q5 embedder · Q6 PR shipping

**Answers (user 2026-09-05):** `2 2 1`

| ID | Decision | Choice |
|----|----------|--------|
| D4 | Seam proof without 001 | **Thin internal producer** — after `mnemonic_commit` writes L2, hook tier generation |
| D5 | Local embedder | **onnxruntime-go** (or similar) real local model behind `MNEMONIC_EMBED` — *reopened in Round 3 vs Pure Go constraint* |
| D6 | PR shipping | **One PR** for the whole change |

## Round 3 — asked 2026-09-05

Q7 Pure Go vs ONNX · Q8 commit→tier await · Q9 search corpus

**Answers (user 2026-09-05):** `1 1 3`

| ID | Decision | Choice |
|----|----------|--------|
| D5′ | Local embedder (revises D5) | **Pure Go** — drop onnxruntime; Pure Go local embedder and/or stub + optional remote HTTP |
| D7 | Commit → tier await | **Fire-and-forget** — commit succeeds when L2 (+ LTM row) durable; tiers async |
| D8 | Search corpus | **Both** `long_term_memories` and `tiered_contents`, with filter param; **default = LTM-only** |

## Shared understanding

**Confirmed** by user `yes` (2026-09-05). Proceed to revise `change.md` / `tasks.md` / `acceptance.feature`.
