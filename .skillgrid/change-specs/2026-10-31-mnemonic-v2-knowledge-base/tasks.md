# ProjectWiki Change — Tasks

> Sliced from `blueprint.md` (per `skillgrid:slicing`). Vertical tracer-bullet tickets, wave order.

## Epic: ProjectWiki Engine (`2026-10-31-mnemonic-v2-knowledge-base`)

### Wave 1 (engine core)

- [x] TICKET-01 — `projectwiki_bootstrap` vault scaffold (idempotent, no-ProjectWiki detection) — Task 1. *(Landed as KB-T1..T5, 2026-11-05 — state `P26-11-05-kb011-exec`.)*
- [x] TICKET-02 — `projectwiki_index` / `projectwiki_reindex` derived FTS rebuild over the vault (incremental + full, idempotent) — Task 2. *(KB-T1..T5; Reindex derived-state parity fixed at QA, 2026-11-11.)*

### Wave 2 (retrieval + drift)

- [x] TICKET-03 — `projectwiki_query` wiki-first cited retrieval + memory fallback + `degraded: stale_index`. *(KB-T5 query; thin/fallback/degraded engine wiring landed at QA, 2026-11-11 — W003.)*
- [x] TICKET-04 — `projectwiki_validate` pin-check (all_of / line+version / symbol anchors) + claims. *(KB-T4/T5.)*
- [x] TICKET-05 — `projectwiki_lint` deterministic health + `--fix` safe subset. *(KB-T5; full spec 007 §2 check set landed at QA, 2026-11-11 — W006.)*

### Wave 3 (state + skill + eval)

- [x] TICKET-06 — `projectwiki_status` first verb + `projectwiki_run_state` / `projectwiki_page_baselines` / `projectwiki_events`. *(KB-T5; T7 by-type/stale/pending/orphan breakdown landed at QA, 2026-11-11 — W006.)*
- [x] TICKET-07 — `projectwiki_cite_source` external citation + re-fetch section granularity. *(KB-T7, handler parity.)*
- [x] TICKET-08 — `projectwiki_eval` retrieval regression net (fixture corpus, recall@K, meanRankDist, topScoreBelowOne; threshold = observed baseline). *(KB-T5, `TestEvalRecallAtOne`.)*
- [x] TICKET-09 — `projectwiki` management skill (SKILL.md / CONVENTIONS.md / PAGES.md / CLAIMS.md / MERMAID.md) + `009-skill.md` amendments. *(`.agents/skills/projectwiki/`, 5 files.)*

## Traceability (SATISFIES)

### TICKET-01
**SATISFIES:** bootstrap-scaffolds-contextual-types; assess-query-thin-on-empty-index

### TICKET-02
**SATISFIES:** index-incremental-only-changed; reindex-rebuilds-projectwiki-pages; index-soft-deletes-removed-pages; index-in-degree

### TICKET-03
**SATISFIES:** query-returns-ranked-projectwiki-pages; query-no-match-returns-empty; assess-query-degraded-when-index-lags-vault

### TICKET-04
**SATISFIES:** validate-fresh-page-passes; validate-stale-claim-fails; validate-unresolved-pin-fails

### TICKET-05
**SATISFIES:** lint-structural-checks; lint-weak-pin; lint-missing-title-and-fix

### TICKET-06
**SATISFIES:** status-reports-indexed-pages; status-breakdown

### TICKET-07
**SATISFIES:** cite-source-returns-content

### TICKET-08
**SATISFIES:** eval-recall-at-one

### T-013-1
**SATISFIES:** validate-page-path-contextual-types; collect-pages-whitelist-scope

### T-013-2
**SATISFIES:** bootstrap-scaffolds-contextual-types

## Exit criteria

- All Must-Haves (truths / artifacts / key links) green; QA gate PASS; two-axis review clean; ship.

---

## Epic: 013 Wiki Autolearning (research results + user preferences into the vault)

> Sliced from `specs/013-wiki-autolearning.md` §9 (engine → scaffold → skills order). ADR-0060 ACCEPTED 2026-11-09 (explicit user OK) — LOCKED row flipped. The 011 tickets above landed as KB-T1..T6 (2026-11-05, state `P26-11-05-kb011-exec`). Delivered 2026-11-09 (state `P26-11-09-kb013-exec`). **Resolved (2026-11-09, user decision):** okf-lint rule 4 changed per OKF issue #176 (GoogleCloudPlatform/knowledge-catalog) — `[[wikilinks]]` are allowed as an equivalent in-bundle link syntax; unresolved wikilinks are advisory-only (OKF §5.3 broken-link tolerance), never drift. The lint's resolution set mirrors the engine (basename / slug / rel / title / aliases, case-insensitive) plus the documented `[[type/slug]]` form. **Engine gap fixed (2026-11-10):** `recomputeInDegree` (index_inc.go) also registers rel without `.md` — the documented `[[type/slug]]` form now resolves (title / aliases / slugOf(rel) / rel / rel-without-.md); pinned by `TestIndexInDegree` (type/slug link counted in in_degree).

- [x] T-013-1 — **Engine: four contextual page types** (spec 013 §2/§9.1). `PageTypeDirs` 6→10 (+ `sources` `concepts` `syntheses` `preferences`; taxonomy 8→12 with the opt-in 012 `skills`/`cases`); `pageDirs` index whitelist extension (KB-T16 umbrella — `collectPages` honors the 001 §7 scope for the four folders); `ValidatePagePath` accepts the four types; `projectwiki_bootstrap` scaffolds the four empty folders + `page_types` + the four `templates/` files + the `recall` block (001 §8 sentinel: `inline_types` incl. `preference`, `inline_max` 1600, `links_threshold` 50). Pinned tests: path validation, whitelist, bootstrap scaffold shape.
- [x] T-013-2 — **Scaffold/docs: this vault** (spec 013 §9.2 + 001 §8). `PROJECTWIKI_SCHEMA.md` sections for the four types + the contradiction golden rule (013 §6); the four type folders (`.gitkeep`) + `templates/` (`source.md` `concept.md` `synthesis.md` `preference.md` mirroring the engine templates); `config.json` `page_types` += four + `recall` block; same-edit `projectwiki index` + `validate`.
- [x] T-013-3 — **Skills: the autolearning write paths** (spec 013 §3/§4/§5/§9.3). `research/SKILL.md`: capture step — `sources/` pages + `draft` `syntheses/` hub at research time, same-edit index (WA-INV-5); `reflect/SKILL.md`: triage step — the 013 §5 learning→home table replacing the stale `reference/06-research-findings.md` lift, `draft → stable` finalization at ship; web-cache promotion candidates retargeted from "one concept file" to the `sources/`/`syntheses/`/`concepts/` tree; `projectwiki/SKILL.md`: the 12-type form (the 013 lines reverted at the mcp-split review re-apply now that 013 lands); `projectwiki-data-loading/SKILL.md` un-parked from the change folder back to `.agents/skills/`.
