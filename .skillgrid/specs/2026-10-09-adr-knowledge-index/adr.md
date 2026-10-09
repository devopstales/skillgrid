# ADR manifest — 2026-10-09-adr-knowledge-index

No new ADR is introduced by this change. It is an additive feature (config include + read-only
search tool) with no architectural decision, new dependency, or data-model change.

Decisions it relies on (all in-force):

- **ADR-0019** — decisions are files (`04-adr-NNNN-slug.md`); `ASSUMPTIONS.md` holds the path.
  Source: `.skillgrid/artifacts/04-adr-0019-decisions-are-files.md`.
- **ADR-0012** — SQLite as the memory / second-brain store. The `adr_search` tool reads the
  existing `chunks`/`files`/`chunks_fts` tables in the project store.
  Source: `.skillgrid/artifacts/04-adr-0012-sqlite-as-second-brain.md`.

Open Question #5 in `ASSUMPTIONS.md` ("Ingesting local `.skillgrid/` files into that store. Needs
a fresh decision/ADR before any re-attempt.") is **partially answered** by this change: Part 1
ingests `.skillgrid/**/*.md` into the *code* index (a targeted `DocInclude` carve-out), not into
the observation store. If the broader "source knowledge into the observation store" direction is
re-attempted, it still needs a fresh ADR.
