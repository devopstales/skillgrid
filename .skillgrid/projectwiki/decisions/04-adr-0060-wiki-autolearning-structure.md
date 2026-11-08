---
type: TechnicalDecision
status: proposed
supersedes: none
date: 2026-11-08
---

# ADR-0060 — Wiki autolearning: flat type-driven research + preference structure

Change: 2026-10-31-mnemonic-v2-knowledge-base (spec-zone amendment; `specs/013-wiki-autolearning.md` + the 001-structure amendment of 2026-11-08)

## Context

- Research done in a change promotes at most **one "concept file" per confirmed web-cache candidate** (reflect step 3.7), and reflect's durable-findings lift points at `.skillgrid/projectwiki/reference/06-research-findings.md` — a path that **no longer exists** (the 2026-11-06 vault reorg removed the `reference/` folder; the only remaining reference to it is the stale line in `reflect/SKILL.md`). Unconfirmed research results never reach the vault at all.
- User preferences (how the user wants work done) live only in the memory tier; the vault has no home for them, so a preference stated in one session is not surfaced when a later session queries the domain it constrains.
- pi-llm-wiki solves the same problem with a proven shape: research is cut into `sources/` (one summary page per source) + `concepts/` (reusable ideas/patterns) + `syntheses/` (cross-cutting answers), all **flat type folders**, one file per thing, cross-referenced (≥2 links per page), cited, with **contradictions flagged** (both sides documented).
- pi-hermes-memory's autolearning loop (learning at work-unit boundaries, correction detection, failure memory, consolidation) has no vault-side equivalent here: the memory tier is promoted to the vault only by explicit generate runs. The shipped change — the richest sourced-learning source in the system — never flows back into the vault.
- The wiki mechanics already exist in the engine (001 §6): `[[type/slug]]` wikilinks, on-the-fly backlinks, `in_degree` ranking, the ≥2-outgoing-links golden rule, orphan lint, `sources:` citations with `external:` + `build_fingerprint` staleness tokens, and the OKF `draft|stable|deprecated` lifecycle. What is missing is the **page types** and the **write paths** that feed them.

## Decision

- **Research results and user preferences enter the vault as flat, type-driven OKF pages.** Four new contextual (non-grounded) page types: `SourceSummary` (`sources/`), `Concept` (`concepts/`), `Synthesis` (`syntheses/`), `Preference` (`preferences/`) — the page taxonomy grows 8 → 12 (001 §4/§5.1, amended).
- **A research result is a tree, never a file:** a `syntheses/` hub (question, method, findings, answer) + one `sources/` page per source (summary + `external:` citation with `build_fingerprint`) + one `concepts/` page per reusable finding. Per-source staleness: a fingerprint drift flags exactly one source page, not the whole result (WA-INV-3).
- **User preferences are `preferences/` pages**, one per preference area, each preference cited to the `memory:` observation where the user stated it; the vault holds the project-scoped view, memory the global. The vault row is user-owned (001 §3).
- **Write paths (autolearning loop):** the `research` skill captures source pages + a `draft` synthesis **at research time** (fingerprints current); `reflect` finalizes **at ship** (citations confirmed, links to what shipped added, `draft → stable`, closing `projectwiki_generate`); non-SDD sessions carry a **session-close checkpoint** (a durable fact not yet in the vault → one page edit or a named Next Step); an in-flight user correction → `mem_save` always, plus a vault edit in the same turn when it is a code fact or preference.
- **Contradiction rule (adopted as a golden rule, 001 §6):** conflicting pages link to each other; the superseded page/claim goes `status: deprecated`; never silently overwritten (WA-INV-4).
- **Confirmation:** a new ADR from reflect is **propose + confirm** (a locked decision enters the vault only on explicit user yes); every other vault home is a **direct agent write** under the 001 §3 ownership row.

## Alternatives considered

- **Nested `research/<result-slug>/` topic trees** — rejected: breaks the folder-qualified wikilink form (`[[type/slug]]` is a single-segment type folder, 001 §6) and the one-folder-per-type mapping; and "a page is not a directory of sub-pages" (001 §4).
- **The status-quo single `reference/06-research-findings.md` sink** — rejected: a retrieval black hole (one file, no per-type query, no per-source staleness) — and the path was already stale post-reorg.
- **A root `PREFERENCES.md`** — rejected: vault-root project docs are **out of index scope** (001 §7); unindexed preferences cannot be recalled, which defeats the point.

## Consequences

- The sentinel `page_types`, `recall.inline_types` (+`preference`), `PROJECTWIKI_SCHEMA.md`, `templates/`, and the `collectPages` whitelist all gain the four folders (013 §9); `projectwiki_bootstrap` scaffolds the four empty folders.
- `reflect` gains vault-write duty (the triage step replaces the stale lift) and the `research` skill gains a capture step — skill-zone edits, after this spec lands.
- Stage-1 recall inlines `preference` pages (short, immediately actionable — the llm-wiki skill/case carve-out); sources/concepts/syntheses stay links-only.
- Memory remains the fast tier; this ADR defines the **memory → vault promotion gate**, not a memory-tier change.
- **Status: PROPOSED.** The LOCKED index row in `ASSUMPTIONS.md` is added on explicit user OK (the row requires a user OK; the file may precede it). Sequence 0060 is the next free number per the index note.
