---
name: projectwiki-data-loading
description: "Load data into the ProjectWiki vault: search/research results and reflect results become typed, cross-linked OKF pages (the llm-wiki ingest discipline on the projectwiki engine)."
license: MIT
metadata:
  author: devopstales
  version: "1.1"
  part-of: skillgrid
  based_on: hermes:llm-wiki 2.1.0 + hermes:obsidian 1.0.0 (Karpathy LLM Wiki pattern, adapted)
---

# ProjectWiki Data Loading

Load durable data into the ProjectWiki vault: **search results** (web research, docs,
papers) and **reflect results** (post-ship sourced learnings) become typed,
cross-linked OKF pages that compound. This is a copy of the llm-wiki
ingest/query/lint discipline, changed for the projectwiki engine: the vault is
`.skillgrid/projectwiki/` (not `~/wiki`), the index is derived (not hand-maintained),
and raw sources are cited, not stored.

Unlike RAG (which rediscovers knowledge from scratch per query), the vault compiles
knowledge once and keeps it current: cross-references are already there,
contradictions have already been flagged, and per-source staleness is already pinned.

**Division of labor:** the human curates sources and confirms locked decisions. The
agent summarizes, cross-references, files, and maintains consistency.

## When This Skill Activates

Use this skill when:

- Search/research results are in hand and worth keeping (web_cache entries, a research
  run's findings, docs read for a change)
- `reflect` closes a change and its sourced learnings must land in the vault
- A substantial query answer should be filed back (a comparison, deep dive, or
  synthesis that would be painful to re-derive)
- The vault's health needs checking after a load pass (lint / validate / status)

**When NOT to use:** in-session knowledge that belongs in `mem_save` (the fast tier —
bugfix stories, session summaries); SDD page authoring mid-pipeline (the
`skillgrid:projectwiki` skill owns engine operation); minting ADRs (the
`skillgrid:architectural-decision-records` skill owns numbering — this skill
*proposes* ADR candidates from reflect results, never mints).

## Vault Location

The vault lives at `.skillgrid/projectwiki/` in the repo root — the single sentinel
dir (sentinel: `config.json`). No env var, no `~/wiki`.

```bash
VAULT=".skillgrid/projectwiki"
# no sentinel? the project has no ProjectWiki yet:
test -f "$VAULT/config.json" || mnemonic projectwiki bootstrap
```

The vault is a directory of OKF markdown — open it in Obsidian (`.obsidian/` at the
vault root is the user's view state; the agent never touches it) or any editor.
Retrieval goes through the engine, not the filesystem (see `skillgrid:projectwiki`).

## Architecture: Three Tiers

```
.skillgrid/projectwiki/
├── PROJECTWIKI_SCHEMA.md        # operating manual (scaffold — user-owned, agent guardrail)
├── config.json                  # sentinel: page types, anchor engine, ignore rules
├── sources/                     # Tier 2: one summary page per external source (013)
├── concepts/                    # Tier 2: reusable ideas / patterns / techniques (013)
├── syntheses/                   # Tier 2: cross-cutting analyses / durable query answers (013)
├── preferences/                 # Tier 2: user preferences (013; reflect-written, user-owned)
├── decisions/ invariants/ workflows/ architecture/ gotchas/ glossary/
│                                # Tier 2: the six core type folders (reflect triage homes)
└── index/                       # Tier 3 (derived, engine-written): last-update.json only.
                                 #   No catalog.md / log.md / reverse-index.md / events.jsonl
                                 #   exists — never author them.
```

- **Tier 1 — External sources are cited, not stored (KB-8).** There is **no `raw/`
  folder**. A source is a `sources:` citation (`external:<name>` + `version` +
  `source_url` + `source_commit` + **`build_fingerprint`**) on the pages that use it,
  plus the `web_cache` entry when the fetch went through the mnemonic web cache.
  Re-fetch the section-granular body on demand; the fingerprint is the drift detector
  (where llm-wiki hashes stored `raw/` files, we hash the upstream).
- **Tier 2 — The pages.** Agent-owned OKF markdown in the type folders. One file per
  thing; `[[type/slug]]` wikilinks (folder-qualified, vault-root-relative); every page
  carries ≥2 outgoing links and a `sources:` list (INV-1).
- **Tier 3 — Derived state.** `index/` + the `projectwiki_*` DB tables: rebuilt by
  `projectwiki index` / `reindex` / `rebuild-meta`, never hand-edited (PW-INV-3). The
  llm-wiki world maintains `index.md` and `log.md` by hand; here there is **no file
  projection at all** — the index is the SQLite tables, `index/` carries only
  `last-update.json`. The equivalent of "update the index" is one command in the
  **same edit** as the page write: `projectwiki index`.

## Resuming the Vault (CRITICAL — every session, before any load)

① **Read `PROJECTWIKI_SCHEMA.md`** — the page contract and folder map.
② **Orient cheap:** `projectwiki status --json` — `indexed`, `by_type`, `orphans`,
   `stale`. There is no catalog or log file to read; this is the state view.
③ **`projectwiki query "<topic terms>"`** — stage-1 ranked links; find existing pages
   for anything you are about to create. This is the difference between a growing
   vault and a pile of duplicates.

Only after orientation, load or query. Skipping it causes duplicate `sources/` pages,
missed cross-references, and contradictions with existing claims.

## Core Operation A — Load Search Results

When search/research results land (web search, docs, papers, a research run):

① **Capture the citation metadata** for every source worth keeping: name, version,
   source_url, source_commit, build_fingerprint (web_cache entries carry id + URL +
   fetched_at — those are the citation seeds). Nothing is copied into the vault (KB-8).

② **Check what already exists** (orientation step ③) — one search pass for all
   sources at once, not N. An existing `sources/<name>-<version>.md` is updated, not
   duplicated.

③ **Write a `sources/` page per source** (type `SourceSummary`):
- summary of what the source says (your words, not copied text)
- the specific claims taken from it (each checkable)
- `sources:` = the `external:` citation with the fingerprint
- ≥2 wikilinks: the `syntheses/` hub it feeds + the `concepts/` / project pages it informs
- filename: `sources/<name>-<version>.md` (e.g. `sources/context7-embed-rs-v0.4.md`)

④ **Cut the reusable parts out:** each idea / pattern / technique that outlives the
   result → its own flat `concepts/` page (cited to the source page). A page is not a
   directory of sub-pages — every part of a result is its own page in its type folder.
   A finding about this repo's own structure is an `architecture/` page, anchored to
   the code — **never an appendix to `ARCHITECTURE.md`**, which is a vault-root doc and
   outside index scope (readable by path, invisible to `query`/`lint`).

⑤ **Write the `syntheses/` hub** (type `Synthesis`, `status: draft`): question,
   method, TL;DR, findings (each finding links to its source/concept page), answer.
   One hub per research question, however many sources fed it.

⑥ **Same-edit `projectwiki index`.** One batch pass for many results: read all →
   identify all → one existence check → write all → index once (the engine appends
   the activity events; there is no log to hand-rotate).

⑦ **Report what changed** — every page created/updated + the index state. A single
   result can touch 5-15 pages; that is the compounding effect — normal and desired.

**Page thresholds** (kept from the llm-wiki schema): create a page when a concept
appears in 2+ sources or is central to one; add to an existing page on a passing
mention; split a page past ~200 lines; **don't page passing mentions**. Weak or
single-source claims load as `status: draft` — promotion to `stable` needs
corroboration (reflect, a second source, or a user confirmation).

**Contradictions at load time:** when a result conflicts with an existing page — link
the two pages to each other, add the `> ⚠️ **Contradiction:**` block on both, and move
the superseded page/claim to `status: deprecated`. Never silently overwrite (013 §6).

## Core Operation B — Load Reflect Results

`reflect` (post-ship, terminal SDD phase) produces sourced Decisions / Lessons /
Patterns / Surprises, user preferences, and the change's web-cache candidates. The
gates (Ship Gate, QA Gate) are owned by reflect — this operation is the load path once
the gates pass. Triage every learning per the table; a finding true only for this one
change stays in `report.md`:

| Learning | Vault home | Write mode |
|---|---|---|
| Locked decision | `decisions/` ADR + `ASSUMPTIONS.md` § LOCKED | **propose + confirm** (never mint on your own) |
| User preference | `preferences/<area>.md` — each bullet cited to the `memory:` observation it was stated in | direct (user-owned pages) |
| Research result | `syntheses/` + `sources/` + `concepts/` (Operation A) | direct, cross-linked |
| Convention (must-hold) | `invariants/` (with `claims:`) | direct |
| Gotcha / tool quirk / failure lesson | `gotchas/` | direct |
| Code fact ("how it works" changed) | component page / `architecture/` (anchors) | direct + re-pin |
| Domain term | `glossary/` | direct |
| Process bug (governs *process*, not code) | the governing skill / `_shared` rule | follow-up ticket — **not** a vault home |

Then **finalize the change's research drafts**: confirm the citations, add links to
what the change shipped (component pages, gotchas, decisions), flip the hub
`draft → stable`. End the load pass with **one** `projectwiki generate` (the closing
synthesis run) — never one per page.

## Core Operation C — Query + File Back

① `projectwiki query "<terms>"` — stage-1 ranked links + snippets (links-first).
② Open the few pages that matter (`projectwiki read` / the file). Don't assume the
   snippet is the page.
③ Synthesize the answer **citing the pages**: "based on `[[syntheses/…]]` and
   `[[concepts/…]]`".
④ **File valuable answers back** — a substantial comparison, deep dive, or novel
   synthesis becomes a `syntheses/` page (type `Synthesis` is the durable-query-answer
   home — llm-wiki's `queries/` has no separate folder here). Don't file trivial
   lookups.
⑤ Same-edit `projectwiki index`.

## Lint / Health (after any load pass)

The engine does what llm-wiki's 13 manual checks do:

| llm-wiki check | Ours |
|---|---|
| Orphan pages (no inbound links) | `projectwiki lint` (in-degree ≤ 1, alias-resolved) |
| Broken wikilinks | `projectwiki lint` |
| Index completeness | derived by construction — `projectwiki index` (fix: `reindex`) |
| Frontmatter validation | `projectwiki lint` (missing title, empty `sources:` on a grounded page, duplicate alias) |
| Stale content | `stale_after` flag (lint) + `projectwiki validate` pin-checks (grounded pages) |
| Contradictions | the `⚠️ Contradiction` blocks + `deprecated` pairs (013 §6) — review manually |
| Source drift (raw sha256) | **build_fingerprint drift on `sources/` pages** — a drifted fingerprint flags exactly one source page (WA-INV-3); re-verify or deprecate it |
| Page size (>200 lines) | advisory here only — split the page; `lint` does not enforce size |
| Tag audit | n/a — the type folders are the taxonomy |
| Log rotation | n/a — no engine-written log exists; `status --json` + `last-update.json` are the state view |

Real lint codes: `missing_title`, `stale_index`, `orphan`, `weak_cross_reference`,
`broken_wikilink`, `duplicate_alias`, `missing_sources`, `ungrounded_grounded`,
`weak_pin`, `past_stale_after`, `contradiction`, `broken_citation`, `coverage_gap`.
`--fix` repairs empty titles only.

Target: **no new findings** + `projectwiki validate` no drift (grounded pages). `lint`
without `--strict` always exits 0, and a legacy vault carries a warn baseline (233
findings at 2026-11-12) — never rewrite pages just to make it print `clean`.

## Working Notes

- **Search:** `projectwiki query` (default) → `projectwiki search` (raw FTS power
  path) → `projectwiki status --json` (cheap orient; `by_type` + `orphans`). Never
  `grep` the vault when the engine is up.
- **Bulk load:** batch per Operation A step ⑥ — one existence pass, one write pass,
  one index.
- **Archiving:** there is no `_archive/` move. Superseded pages get
  `status: deprecated` (unsearchable by query, still readable, still linked) — page
  files and ADR rows are never deleted (IRON RULE).
- **Obsidian:** the vault is Obsidian-compatible out of the box (`[[type/slug]]`
  renders, graph view works, frontmatter is queryable). `.obsidian/` is the user's
  per-view state — the agent never edits it. No headless-sync daemon: the vault is
  in-repo, git is the sync.
- **Secrets:** never load keys, tokens, or machine paths into pages — a citation is
  name + version + URL, never the fetched secret.

## Field Map (llm-wiki → ProjectWiki)

| llm-wiki | ProjectWiki |
|---|---|
| `WIKI_PATH` / `~/wiki` | `.skillgrid/projectwiki/` (sentinel `config.json`) |
| `raw/` immutable stored sources | **no raw layer** — `external:` citation + `build_fingerprint` (KB-8) + `web_cache` |
| `SCHEMA.md` (agent-written) | `PROJECTWIKI_SCHEMA.md` (scaffold — user-owned) |
| `index.md` / `log.md` (hand-maintained) | **no file counterpart** — the derived index is the `projectwiki_*` DB tables (`projectwiki index`); `index/` holds only `last-update.json` |
| `entities/` | `glossary/` (terms, tools, products — cited) |
| `concepts/` | `concepts/` (013) |
| `comparisons/` + `queries/` | `syntheses/` (013 — the durable query answer) |
| tags + taxonomy | the type folders (12 types; 001 §4) |
| `confidence:` / `contested:` | `status: draft\|stable\|deprecated` + `stale_after` + the `⚠️ Contradiction` block |
| raw `sha256:` | `build_fingerprint` on the `external:` citation |
| `[[Note Name]]` | `[[type/slug]]` (folder-qualified, vault-root-relative) |
| `OBSIDIAN_VAULT_PATH` | the vault root; `.obsidian/` is user-owned |
| hermes `web_extract` / `search_files` | `web_cache` + `read` for fetch; `projectwiki query/search` for retrieval |

## Pitfalls

- **Always orient first** — schema + `status --json` + `query` before any load. Skipping
  causes duplicates and missed cross-references.
- **Never hand-edit `index/`** — page + `projectwiki index` in the same edit, or the
  change is incomplete (PW-INV-3).
- **Never copy source content** — summarize + cite (KB-8); the fingerprint is the
  only copy.
- **Don't page passing mentions** — the page thresholds exist for a reason.
- **No page without 2+ outgoing links** — an isolated page is invisible to stage-1
  recall and orphan-linted anyway.
- **A contradiction is never silently overwritten** — link both, deprecate the
  loser; otherwise weak claims harden into unreviewed vault fact.
- **Ask before a mass update** — a load touching 10+ existing pages: confirm scope
  first.
- **One `projectwiki generate` per load pass** — not per page.
- **Engine down** → degrade to filesystem reads and declare `Degraded:
  projectwiki`; never invent a silent substitute retrieval path.

## Verification

- [ ] Orientation done (schema + catalog + `projectwiki query`) before the first write.
- [ ] Every loaded source has a `sources/` page with the full `external:` citation (name, version, URL, commit, fingerprint) — a page missing any field is malformed.
- [ ] Every research result has a `syntheses/` hub linking to all its source + concept pages; every concept page links back.
- [ ] Reflect results triaged per the table — ADR candidates proposed (not minted), preferences cited to `memory:`, process bugs ticketed (not filed).
- [ ] `projectwiki index` ran in the same edit as every page write; `projectwiki lint` has no **new** findings; `projectwiki validate` no drift on grounded pages; `status --json` shows `stale: 0`.
- [ ] Every new page has 2+ outgoing wikilinks — `status --json` `orphans` did not grow.
- [ ] No raw content copied in; no secrets in pages; `.obsidian/` untouched.
