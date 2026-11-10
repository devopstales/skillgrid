# ProjectWiki Change — Tasks

> Sliced from `blueprint.md` (per `skillgrid:slicing`). Vertical tracer-bullet tickets, wave order.

## Epic: ProjectWiki Engine (`2026-10-31-mnemonic-v2-knowledge-base`)

### Wave 1 (engine core)

- [ ] TICKET-01 — `projectwiki_bootstrap` vault scaffold (idempotent, no-ProjectWiki detection) — Task 1.
- [ ] TICKET-02 — `projectwiki_index` / `projectwiki_reindex` derived FTS rebuild over the vault (incremental + full, idempotent) — Task 2.

### Wave 2 (retrieval + drift)

- [ ] TICKET-03 — `projectwiki_query` wiki-first cited retrieval + memory fallback + `degraded: stale_index`.
- [ ] TICKET-04 — `projectwiki_validate` pin-check (all_of / line+version / symbol anchors) + claims.
- [ ] TICKET-05 — `projectwiki_lint` deterministic health + `--fix` safe subset.

### Wave 3 (state + skill + eval)

- [ ] TICKET-06 — `projectwiki_status` first verb + `projectwiki_run_state` / `projectwiki_page_baselines` / `projectwiki_events`.
- [ ] TICKET-07 — `projectwiki_cite_source` external citation + re-fetch section granularity.
- [ ] TICKET-08 — `projectwiki_eval` retrieval regression net (fixture corpus, recall@K, meanRankDist, topScoreBelowOne; threshold = observed baseline).
- [ ] TICKET-09 — `projectwiki` management skill (SKILL.md / CONVENTIONS.md / PAGES.md / CLAIMS.md / MERMAID.md) + `009-skill.md` amendments.

## Exit criteria

- All Must-Haves (truths / artifacts / key links) green; QA gate PASS; two-axis review clean; ship.

---

## Epic: 013 Wiki Autolearning (research results + user preferences into the vault)

> Sliced from `specs/013-wiki-autolearning.md` §9 (engine → scaffold → skills order). ADR-0060 ACCEPTED 2026-11-09 (explicit user OK) — LOCKED row flipped. The 011 tickets above landed as KB-T1..T6 (2026-11-05, state `P26-11-05-kb011-exec`). Delivered 2026-11-09 (state `P26-11-09-kb013-exec`). **Resolved (2026-11-09, user decision):** okf-lint rule 4 changed per OKF issue #176 (GoogleCloudPlatform/knowledge-catalog) — `[[wikilinks]]` are allowed as an equivalent in-bundle link syntax; unresolved wikilinks are advisory-only (OKF §5.3 broken-link tolerance), never drift. The lint's resolution set mirrors the engine (basename / slug / rel / title / aliases, case-insensitive) plus the documented `[[type/slug]]` form. **Engine gap fixed (2026-11-10):** `recomputeInDegree` (index_inc.go) also registers rel without `.md` — the documented `[[type/slug]]` form now resolves (title / aliases / slugOf(rel) / rel / rel-without-.md); pinned by `TestIndexInDegree` (type/slug link counted in in_degree).

- [x] T-013-1 — **Engine: four contextual page types** (spec 013 §2/§9.1). `PageTypeDirs` 6→10 (+ `sources` `concepts` `syntheses` `preferences`; taxonomy 8→12 with the opt-in 012 `skills`/`cases`); `pageDirs` index whitelist extension (KB-T16 umbrella — `collectPages` honors the 001 §7 scope for the four folders); `ValidatePagePath` accepts the four types; `projectwiki_bootstrap` scaffolds the four empty folders + `page_types` + the four `templates/` files + the `recall` block (001 §8 sentinel: `inline_types` incl. `preference`, `inline_max` 1600, `links_threshold` 50). Pinned tests: path validation, whitelist, bootstrap scaffold shape.
- [x] T-013-2 — **Scaffold/docs: this vault** (spec 013 §9.2 + 001 §8). `PROJECTWIKI_SCHEMA.md` sections for the four types + the contradiction golden rule (013 §6); the four type folders (`.gitkeep`) + `templates/` (`source.md` `concept.md` `synthesis.md` `preference.md` mirroring the engine templates); `config.json` `page_types` += four + `recall` block; same-edit `projectwiki index` + `validate`.
- [x] T-013-3 — **Skills: the autolearning write paths** (spec 013 §3/§4/§5/§9.3). `research/SKILL.md`: capture step — `sources/` pages + `draft` `syntheses/` hub at research time, same-edit index (WA-INV-5); `reflect/SKILL.md`: triage step — the 013 §5 learning→home table replacing the stale `reference/06-research-findings.md` lift, `draft → stable` finalization at ship; web-cache promotion candidates retargeted from "one concept file" to the `sources/`/`syntheses/`/`concepts/` tree; `projectwiki/SKILL.md`: the 12-type form (the 013 lines reverted at the mcp-split review re-apply now that 013 lands); `projectwiki-data-loading/SKILL.md` un-parked from the change folder back to `.agents/skills/`.
