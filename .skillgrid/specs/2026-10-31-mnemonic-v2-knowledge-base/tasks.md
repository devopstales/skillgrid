# Knowledge-Base Change — Tasks

> Sliced from `blueprint.md` (per `skillgrid:slicing`). Vertical tracer-bullet tickets, wave order.

## Epic: Knowledge-Base Engine (`2026-10-31-mnemonic-v2-knowledge-base`)

### Wave 1 (engine core)

- [ ] TICKET-01 — `kb_bootstrap` vault scaffold (idempotent, no-KB detection) — Task 1.
- [ ] TICKET-02 — `kb_index` / `kb_reindex` derived FTS rebuild over the vault (incremental + full, idempotent) — Task 2.

### Wave 2 (retrieval + drift)

- [ ] TICKET-03 — `kb_query` wiki-first cited retrieval + memory fallback + `degraded: stale_index`.
- [ ] TICKET-04 — `kb_validate` pin-check (all_of / line+version / symbol anchors) + claims.
- [ ] TICKET-05 — `kb_lint` deterministic health + `--fix` safe subset.

### Wave 3 (state + skill + eval)

- [ ] TICKET-06 — `kb_status` first verb + `kb_run_state` / `kb_page_baselines` / `kb_events`.
- [ ] TICKET-07 — `kb_cite_source` external citation + re-fetch section granularity.
- [ ] TICKET-08 — `kb_eval` retrieval regression net (fixture corpus, recall@K, meanRankDist, topScoreBelowOne; threshold = observed baseline).
- [ ] TICKET-09 — `kb` management skill (SKILL.md / CONVENTIONS.md / PAGES.md / CLAIMS.md / MERMAID.md) + `009-skill.md` amendments.

## Exit criteria

- All Must-Haves (truths / artifacts / key links) green; QA gate PASS; two-axis review clean; ship.
