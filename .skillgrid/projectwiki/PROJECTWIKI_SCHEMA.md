# Knowledge Schema

Operating manual for the project wiki vault at this root.

The vault is a flat, type-driven research + preference structure (ADR-0060): six core type folders (`decisions/` `invariants/` `workflows/` `architecture/` `glossary/` `gotchas/`) plus the four contextual folders below. `index/` and `templates/` are scaffolding — out of scope for indexing. One file per thing; a page is reached by its `[[type/slug]]` wikilink (backlinks computed on the fly; every page needs ≥2 outgoing links).

## `sources/` — SourceSummary pages

One page per external source (library, paper, article, CLI). A **summary, never a copy** of the raw content (WA-INV-2). Frontmatter `sources:` carries the `external:<name> <version>` citation with a `build_fingerprint` staleness token — fingerprint drift flags exactly this page alone (WA-INV-3). Written by the `research` skill at research time (capture-fresh).

## `concepts/` — Concept pages

Reusable ideas / patterns / techniques that outlive the research result they came from. Each links back to the `syntheses/` result it was cut from and to at least one repo page (`gotchas/`, `architecture/`, `decisions/`) that makes the external finding actionable here.

## `syntheses/` — Synthesis pages (result hubs)

The hub of a research result (WA-INV-1): question, method, findings, answer. Cross-linked to every `sources/` page it drew on and every `concept` it spawned (≥2 outgoing, enforced). `status: draft` while the research is open; finalized to `stable` by `reflect` at ship.

## `preferences/` — Preference pages

Project-scoped user preferences. Each bullet cites the observation where the user stated it (`memory:<sync_id>` — INV-1 provenance); the global view stays in memory. Stage-1 injection inlines these pages (they are short and immediately actionable); the other contextual types stay links-only.

## Contradictions

A new page never silently overwrites an existing one: it **links** to the conflicting page and marks it `status: deprecated`, with the deprecation reason on the deprecated page. Contradiction pairs are discoverable in both directions — the deprecated page links to its replacement too. `deprecated` pages stay out of the index's ranking but remain readable.
