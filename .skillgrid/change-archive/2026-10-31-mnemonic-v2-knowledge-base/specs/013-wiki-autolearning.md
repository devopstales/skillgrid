# 013 — Wiki Autolearning (research results + user preferences into the vault)

**Status:** DRAFT · **Owner:** project-wiki change
**Depends on:** `001-structure` (taxonomy — amended 2026-11-08 for the four types in
this spec), `002-sources` (`external:` citation + `build_fingerprint`), `004-index`
(whitelist), `005-query` (stage-1 inline carve-out), `009-skill` (authoring discipline).
**Reference:** pi-llm-wiki (`sources/` / `concepts/` / `syntheses/` flat type folders;
golden rules 4–7), pi-hermes-memory (background learning at work-unit boundaries,
correction detection, failure memory, consolidation).
**Decision record:** ADR-0060 (PROPOSED — LOCKED row awaits explicit user OK).

---

## 1. Capability

Research results and user preferences flow into the ProjectWiki as **structured,
cross-linked OKF pages** — cut into multiple pages and linked like a real wiki (the
llm-wiki pattern), not dumped into a single findings file. Two write paths feed the
vault automatically (no user prompt required):

- **research-time capture** (the `research` skill) — at the moment the sources are
  fresh: `sources/` pages + a `draft` `syntheses/` hub.
- **reflect-time finalization** (the `reflect` skill, post-ship) — confirm citations,
  link to what the change shipped, `draft → stable`, the closing `projectwiki_generate`.

Memory stays the **fast tier** (`mem_save` in-flight, `mem_capture_passive`); the vault
is the **durable tier**. These two write paths are the **promotion gate** (memory →
vault) — the same role as llm-wiki's `wiki_retro`, plus typed structure and a human
gate on ADRs.

**What this spec is NOT:** a new retrieval engine (wikilinks, backlinks, FTS, stage-1
links-first recall all exist — 001 §6, 005), and not a change to the memory tier.

## 2. Research-result structure (flat, type-driven)

A research result is **never one page**. It is a tree of flat pages in their type
folders, all vault-root-qualified wikilinks:

```
syntheses/<result-slug>.md          THE result — question, method, TL;DR, findings,
                                    answer. The hub of the tree.
sources/<name>-<version>.md         one page per source — summary + the claims taken
                                    from it + the `external:` citation (version,
                                    source_url, source_commit, build_fingerprint).
concepts/<slug>.md                  one page per reusable finding — an idea / pattern
                                    / technique that outlives this result.
```

**Rules**

- **One file per thing** (001 §4): every part of the result is its own page; a result
  with three sources and two reusable findings is five pages.
- **Cross-reference everything** (001 §6): the hub links to every `sources/` page it
  drew on and every `concepts/` page it spawned (satisfies ≥2 outgoing). Each
  `sources/` page links back to the hub and to the concepts it informs. Each
  `concepts/` page links to the hub **and** to the project page(s) where it applies
  (`gotchas/`, `architecture/`, `decisions/`, component pages) — the link that makes
  an external finding actionable in this repo.
- **Cite, don't store** (KB-8): a `sources/` page stores a *synthesis* of the source;
  the raw content is never copied into the vault. The citation carries the
  `build_fingerprint` staleness token (001 §5.4).
- **Per-source staleness:** because the fingerprint lives on the individual `sources/`
  page, one upstream doc drifting flags exactly one page — never the whole result.
  The hub's `## Findings` stays true; the drifted source page is re-verified or
  deprecated on its own.
- **Naming:** `sources/<name>-<version>.md` uses the citation's `<name>` + `version`
  (e.g. `sources/context7-embed-rs-v0.4.md`); slug is kebab-case, stable across edits.

## 3. User preferences

- **Home:** `preferences/` pages (type `Preference`), one page per **preference area**
  (`preferences/commit-style.md`, `preferences/tooling.md`, …). A single
  `preferences/general.md` is the default first page; areas split out when an area
  grows past a handful of bullets.
- **Content:** each preference is a bullet **cited to the `memory:` observation**
  where the user stated it (INV-1 provenance — "the user said X in session S" is a
  checkable citation, not a vibe).
- **Links:** a preference page links to the convention page(s) it constrains
  (`invariants/`, `workflows/`, component pages) — so querying that domain surfaces
  the preference in backlinks, and the ≥2-outgoing rule holds.
- **Scope:** the vault holds the **project-scoped** view of preferences. Global user
  preferences (true in every project) stay in memory (global scope); the vault page
  cites the memory observation.
- **Ownership:** agent-generated + **user-owned** (001 §3 row) — the user may edit a
  preference page directly; the agent must not override a user edit without the user
  saying so.

## 4. Write paths (the autolearning loop)

| Trigger | Who | What lands |
|---|---|---|
| Research run (in-flight, during any change or standalone) | `research` skill | `sources/` pages + `syntheses/<slug>.md` with `status: draft`; same-edit `projectwiki_index` |
| Ship → reflect (terminal phase) | `reflect` skill | Triage step (§5): vault writes per the table; `draft → stable` on the result's hub (citations re-confirmed, links to shipped pages added); the closing `projectwiki_generate` |
| User correction (in-flight, any session) | the agent | `mem_save` (memory tier, always); **plus** a vault edit in the same turn when the correction is a code fact or a preference |
| Session close (non-SDD session) | the agent (the `mem_session_finalize` hook) | **Checkpoint:** if the session produced a durable code fact / preference not yet in the vault → one page edit + index, or it becomes a **named Next Step** in the session summary. A clean session produces nothing — do not force a write |

**Capture fresh, finalize late.** Research pages are written *at research time* because
fingerprints and source context are current; reflect finalizes at ship because it is
the only phase with the gates, the ship context, and the shipped code to link against.
A result that ships without reflect (blocked cycle) stays `draft` — visible,
searchable, and marked as un-finalized (001 §5.1: `deprecated` is unsearchable,
`draft` is not — deliberately).

**Memory → vault promotion is lossy by design:** the vault stores the *distilled,
cited, linked* knowledge; the memory store keeps the raw observations. `sources:`
(`memory:<sync_id>`) is the bridge — a vault page is re-groundable to its origin
observation.

## 5. Learning → home triage (reflect)

Reflect's sourced learnings (Decisions / Lessons / Patterns / Surprises) + preferences
+ research results land per this table (replaces the stale
`reference/06-research-findings.md` lift in `reflect/SKILL.md`):

| Learning | Vault home | Write mode |
|---|---|---|
| Locked decision (a choice that constrains future work) | `decisions/04-adr-NNNN-*.md` + `ASSUMPTIONS.md` § LOCKED | **propose + confirm** — a locked decision enters the vault only on explicit user yes (same discipline as web-cache promotion) |
| User preference | `preferences/<area>.md` | direct agent write (user-owned) |
| Research result | `syntheses/` hub + `sources/` + `concepts/` tree | direct agent write, cross-linked, fingerprints |
| Convention (must-hold, checkable) | `invariants/` (with `claims:`) | direct agent write |
| Gotcha / tool quirk / failure lesson | `gotchas/` | direct agent write |
| Code fact ("how it works" changed) | component page / `architecture/` | direct write + anchors; grounded re-pin |
| Domain term | `glossary/` | direct agent write |
| Process bug (governs *process*, not code) | the governing skill / `_shared` rule | **not a vault home** — follow-up ticket, as before |

A finding true only for this one change stays in `report.md` alone (the existing
outlives-this-change test).

## 6. Contradiction rule (golden rule 7, adopted from llm-wiki)

When a new finding contradicts an existing page:

1. The new page links to the old page **and** the old page links back.
2. The superseded page (or, where only one claim is wrong, the specific `claims:`
   entry's page) moves to `status: deprecated` (OKF lifecycle, 001 §5.1) — **not
   deleted** (the IRON RULE: never delete a decision/finding file or its row).
3. Both sides stay readable; the contradiction is documented, never silently
   overwritten. An unsearchable-but-linked deprecated page is the vault's equivalent
   of llm-wiki's "document both sides."

A `deprecated` page with zero incoming links from any stable page is an orphan
candidate under the existing 007 lint rule — the pair must therefore be linked at
write time, not later.

## 7. Query / recall

- **`recall.inline_types` gains `preference`** (001 §8): preference pages are short
  and immediately actionable — inlined at stage 1 like `skill`/`case` (llm-wiki's
  skill/case carve-out). `source` / `concept` / `synthesis` stay **links-only** at
  stage 1 (navigational; the hub is reached by query, the parts by link).
- Backlinks do the navigation: `projectwiki_read` on a `gotchas/` page shows the
  syntheses that cite it; on a `syntheses/` hub, the source pages and concepts.
- No new tools: this spec uses only the existing `projectwiki_*` surface.

## 8. Invariants

- **WA-INV-1 (pages are the result).** A research result exists in the vault only as
  committed, cross-linked pages. An un-promoted `web_cache` entry is a proposal, not
  knowledge; a `draft` hub is a result awaiting finalization, not a completed one.
- **WA-INV-2 (cited, not stored.)** `sources/` pages cite `external:` + version +
  `build_fingerprint`; raw source content is never copied into the vault (KB-8).
- **WA-INV-3 (per-source staleness.)** A fingerprint drift flags exactly one
  `sources/` page — never the whole result tree.
- **WA-INV-4 (contradictions linked, never silently overwritten.)** Per §6.
- **WA-INV-5 (same-edit index.)** Every write path lands its page(s) +
  `projectwiki_index` in the same edit (001 §3 guardrail); the closing
  `projectwiki_generate` runs once per write path, not per page.

## 9. Open items (slicing input)

1. **Engine:** `collectPages` whitelist extension for the four folders (under the
   KB-T16 umbrella — the shipped `collectPages` is a bare `WalkDir` that must honor
   the 001 §7 scope anyway); `projectwiki_bootstrap` scaffolds the four empty folders
   + `page_types` + the four templates.
2. **Scaffold/docs:** `PROJECTWIKI_SCHEMA.md` sections for the four types + the
   contradiction rule; `templates/` gains `source.md`, `concept.md`, `synthesis.md`,
   `preference.md`.
3. **Skills:** `research/SKILL.md` capture step (draft hub + source pages at research
   time); `reflect/SKILL.md` triage step (this spec §5, replacing the stale
   `reference/06-research-findings.md` lift); web-cache promotion candidates
   retargeted from "one concept file" to the `sources/`/`syntheses/`/`concepts/` tree.
4. **ADR-0060:** PROPOSED body committed; **LOCKED row in `ASSUMPTIONS.md` awaits
   explicit user OK** (001 § LOCKED rule: a row requires a user OK).
5. **Slicing:** tickets follow from §9.1–3 (engine → scaffold → skills order; the
   KB-T16 whitelist ticket subsumes item 1's engine half).
