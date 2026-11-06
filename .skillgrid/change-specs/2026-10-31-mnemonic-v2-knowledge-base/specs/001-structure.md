# 001 — Structure (the on-disk layout, page format, taxonomy)

**Status:** DRAFT · **Owner:** knowledge-base change
**Depends on:** `000-overview`, the memory change's `001-storage` (shared store),
`002-schema` (shared DB).
**Reference:** pi-llm-wiki vault restructure (sentinel dot-dir, page-type folders,
derived index), openwiki `.openwikiignore`, repocodex anchored concepts, OKF v0.2.

---

## 1. Capability

This spec is the **keystone for the on-disk shape**: where the vault lives, what is
source of truth, what is derived, the page taxonomy, the OKF page format, and the read
boundary. Every other spec cites the layout and the page format here.

## 2. The layout

### 2.1 The `.skillgrid/` root (reorg, 2026-11-06)

```
.skillgrid/
├── config.yaml            # skillgrid project sentinel + config (read before any skill)
├── change-specs/          # SDD change folders (active) — formerly `specs/`
├── change-archive/        # archived changes — formerly `xarchive/`
├── knowledge/             # the KB vault — §2.2
└── meta/                  # engine state — machine-written, never hand-edited
    ├── state.yaml         # dynamic project state (phase, current change)
    ├── .lock
    ├── okf-metrics.jsonl
    ├── .cache/            # mnemonic indexing tmp
    ├── .prototype/        # prototype tmp
    └── .sdd/              # agent handover tmp (gitignored)
```

Three tiers are visible from the tree (the pi-llm-wiki lesson): **`knowledge/`** is
what the agent + user own (this spec), **`change-specs/` + `change-archive/`** are SDD
process artifacts, **`meta/`** is engine state. `config.yaml` stays at the root as the
project sentinel — the same one-file-detect pattern as the vault's `config.json`, one
level up. The renames (`specs/` → `change-specs/`, `xarchive/` → `change-archive/`)
follow the domain vocabulary: "change" is the SDD unit (`current change` in
`state.yaml`, the `change:` field of the `[skillgrid-context]` commit block).

### 2.2 The vault

```
.skillgrid/knowledge/            the Obsidian vault — .obsidian/ lives at its root
├── .obsidian/                  Obsidian vault config (per-user view state)
├── config.json                 sentinel + vault config (page types, anchor engine, ignore rules)
├── KNOWLEDGE_SCHEMA.md         operating manual (human + agent)
├── templates/                  page templates (decision, invariant, workflow, …)
# ── project-level docs (curated at the vault root; out of index scope — §7) ──
├── ARCHITECTURE.md             live system architecture
├── ASSUMPTIONS.md              ADR index (§ LOCKED); ADR bodies live in decisions/
├── WINDOWS.md                  platform ops notes
# ── the knowledge pages (SOURCE OF TRUTH) ──
#    page-type folders live DIRECTLY under .skillgrid/knowledge/ (no kb/ wrapper, no raw/ layer)
├── decisions/                  why a construct exists (from memory + code)
├── invariants/                 must-hold contracts (pinned, claims)
├── workflows/                  cross-module flows (one anchor per site)
├── architecture/               module map, data flow, boundaries
├── gotchas/                    discovered issues / non-obvious behavior
├── glossary/                   domain terms
# ── input packets (engine-owned, immutable, git-tracked — 012) ──
#    present only when config.json "trajectories" is true (created at bootstrap,
#    or on first capture)
├── trajectories/               replayable task packets (TRJ-YYYY-MM-DD-NNN/; 012)
                                # NOT the raw/ layer (KB-8 is about *external* sources);
                                # a packet is a KB-generated input, cited as trajectory:TRJ-*
# ── derived (rebuilt, never hand-edited) ──
├── index/                      derived file projections — the shared DB is
│   │                           # authoritative; these are its human/browsable mirror
│   ├── last-update.json        # the FTS index is NOT a file here — it is the kb_*
│   │                           # tables in the project's shared DB (004-index).
│   ├── .page-manifest.json     # baseline manifest projection (003 §5.2)
│   ├── catalog.md              # generated one-line-per-page catalog (type, status,
│   │                           #   updated, in-degree) — the agent's cheap orient view
│   ├── log.md                  # generated activity log (the kb_events prose projection)
│   ├── reverse-index.md        # generated backlink map (who links to what)
│   └── events.jsonl            # append-only machine trail — backup-safe: a copy of
│                               #   the vault alone loses the DB, this file does not
└── .knowledgeignore            read boundary (openwiki .openwikiignore)
```

**`index/` is the human mirror of DB state** (the reference's `meta/` pattern — its
`registry.json`/`index.md`/`log.md`/`backlinks.json` projections over the append-only
`events.jsonl`). The shared DB stays authoritative; the files are rebuildable
projections rewritten by `kb_index` / `kb_reindex` / `kb_rebuild_meta` (008 §3). Two
guardrails (the reference's `qmd/` note): treat `index/` as a **unit** — never edit a
single file inside it (restore the dir or re-run the engine), and keep `events.jsonl`
in vault backups — the prose `log.md` cannot reconstruct the structured trail. The
agent's orient step **reads** `index/catalog.md` + the tail of `index/log.md` (009); it
never hand-maintains them (KB-INV-3).

**Why one dot-dir** (pi-llm-wiki, adopted): all vault content lives under a single
sentinel dir (`.skillgrid/knowledge/`, sentinel = `config.json`) instead of scattering
page folders and `index/` across the repo root. Cleaner gitignore, zero
directory-name collisions with the codebase. The sentinel `config.json` is how
`kb_bootstrap` / vault resolution detects "this is a knowledge base" — a project without
`config.json` has no KB, and `kb_*` tools report that cleanly (they do not error).

**Flat, no wrapper.** The page-type folders sit **directly** under
`.skillgrid/knowledge/` — there is no `kb/` subdirectory (KB-1). No `raw/` layer (KB-8),
no `outputs/` layer (KB-9).

## 3. The ownership model

| Path | Owned by | Editable by agent? | Editable by user? |
|---|---|---|---|
| `decisions/ invariants/ workflows/ architecture/ gotchas/ glossary/` | the agent (generated) + the user (reviewed) | ✅ | ✅ |
| `skills/ cases/` (opt-in, 012) | the agent (distilled) + the user (reviewed) | ✅ | ✅ |
| `trajectories/` (packets, 012) | the engine (captured) | ❌ (immutable — re-capture, never edit) | ❌ |
| `index/` (derived projections) | the engine | ❌ (rebuilt by `kb_index`/`kb_reindex`/`kb_rebuild_meta`) | ❌ |
| the `kb_*` tables (shared DB) | the engine | ❌ | ❌ |
| `config.json`, `templates/`, `KNOWLEDGE_SCHEMA.md`, `.knowledgeignore` | the user (scaffold) | ❌ (guardrail) | ✅ |
| `ARCHITECTURE.md` `ASSUMPTIONS.md` `WINDOWS.md` (vault-root project docs) | the user + the agent (ADR skill, architecture upkeep) | ✅ | ✅ |
| `.obsidian/` | the user (Obsidian) | ❌ | ✅ (per-user view state) |

**Guardrails** (KB-INV-3): direct agent edits to `index/`, the `kb_*` tables, or the
scaffold files are blocked. The agent's write path into the **page folders** is
sanctioned two ways: (a) `kb_generate` (the event-driven / proactive synthesis path), or
(b) **direct markdown authoring** (the "file if novel" step, `009-skill` §5) — the agent
may create/edit a page's `.md` directly; it is still the agent's page, not a derived
artifact. What is *never* agent-writable is the **derived layer** (`index/`, the `kb_*`
tables) — those are always rebuilt from the markdown, never the reverse. A direct page
edit must be followed by `kb_index` (re-index) +, if grounded, `kb_validate` (re-pin).

## 4. The page taxonomy (code-first types)

Six core page types + two opt-in working-memory types (012). The first three are
**grounded** (they pin code and carry `anchors:`); the rest are **contextual** (they
synthesize but do not pin a specific line of code).

| Type | Folder | Grounded? | What it captures |
|---|---|---|---|
| `decision` | `decisions/` | ✅ | why a construct exists / a choice was made (from memory + code) |
| `invariant` | `invariants/` | ✅ | a must-hold contract (pinned literals, `claims:`) |
| `workflow` | `workflows/` | ✅ | a cross-module flow (one anchor per site in the flow) |
| `architecture` | `architecture/` | ⬜ | the module map, data flow, boundaries (synthesized) |
| `gotcha` | `gotchas/` | ⬜ | a discovered issue / non-obvious behavior (cited, not pinned) |
| `glossary` | `glossary/` | ⬜ | a domain term (cited) |
| `skill` (opt-in, 012) | `skills/` | ⬜ | a generalized "how I do X", distilled from one or more trajectories (`sources:` cites `trajectory:TRJ-*`) |
| `case` (opt-in, 012) | `cases/` | ⬜ | one concrete past run (cites its one `trajectory:TRJ-*`; agent-authored `outcome: success \| failure \| partial`) |

**One file per thing** (golden rule): each decision / invariant / workflow gets its own
`.md`. A page is not a directory of sub-pages; sub-topics are separate pages linked by
wikilinks.

**Coverage is soft-bounded** (KB-O5): `kb_status` reports how many modules have a page;
there is no hard cap in v2. The graduate-rule (`003-generate`) decides *which* facts
become pages, not a counter.

## 5. The OKF page format

Every page is OKF-flavored markdown: a YAML frontmatter block + a body of OKF sections
with **wikilinks** and **citations**. Grounded pages (decision/invariant/workflow) carry
`anchors:` + `claims:` (repocodex + openwiki); all pages carry `sources:` (provenance,
INV-1).

```markdown
---
title: Enterprise capture is a generator, not a list
type: InvariantContract          # one of the eight page types (canonical form; 012)
project: skillgrid
aliases: []                       # optional, for wikilink resolution (lint dedups these)
anchors:                          # repocodex pin targets (grounded pages only)
  - path: internal/memory/service.go
    all_of: ["capturePayment", "ENTERPRISE"]
  - path: internal/memory/service.go
    line: 482                     # openwiki versioned evidence (a specific line)
claims:                           # must-hold assertions (grounded pages only)
  - subject: enterprise_grace_attempts
    literal: "3"
sources:                          # provenance (INV-1) — CITED, not stored (KB-8)
  - memory:obs-1f3a…              # the memory fact this was generated from
  - code:internal/memory/service.go#L480-L500
  - external:somelib
    version: v1.4.2
    source_url: https://pkg.go.dev/somelib#Capture
    source_commit: 9f2c1ab        # the upstream revision the version was built from
    build_fingerprint: sha256:ab…  # staleness token (neuledge model) — see §5.4
# ── OKF v0.2 control fields (engine-owned, the agent must not author these) ──
status: stable                    # draft | stable | deprecated
stale_after: 2026-10-31           # freshness horizon; a page past this is "stale" to lint
generated:                        # last body change, stamped by the engine
  by: mnemonic/2.0
  at: 2026-10-31T08:10:39.081Z
verified:                         # machine-verification events (pin-check passed)
  - by: mnemonic/2.0
    at: 2026-10-31T08:10:39.081Z
---
## Why
Enterprise capture is a generator so retries stay backpressure-aware.
Do not replace with an in-memory list. [[invariants/enterprise-grace]]

## Where
[[architecture/memory-store]] — the capture path in `internal/memory/service.go`.
```

**Page path** is `invariants/enterprise-grace.md` directly under `.skillgrid/knowledge/`.
The wikilink `[[invariants/enterprise-grace]]` is **relative to the vault root**.

### 5.1 Frontmatter fields

| Field | Required | Notes |
|---|---|---|
| `title` | ✅ | human-readable; becomes the page's canonical label |
| `type` | ✅ | one of the eight (the six core + the opt-in `Skill`/`Case`, 012); canonical case (`Decision`, `InvariantContract`, `BusinessWorkflow`, `Architecture`, `Gotcha`, `GlossaryTerm`, `Skill`, `Case`) |
| `project` | ✅ | the owning project (matches the store's project) |
| `aliases` | ⬜ | alternate wikilink targets; lint flags duplicate aliases across pages |
| `anchors` | grounded only | the pin targets (§5.2) |
| `claims` | grounded only | the must-hold assertions (§5.3) |
| `sources` | ✅ | the citation list (§5.4); empty is a lint finding on a grounded page |
| `status` | ✅ | OKF lifecycle: `draft` \| `stable` \| `deprecated` (engine sets; a `deprecated` page is unsearchable by `kb_query`) |
| `stale_after` | ⬜ | OKF freshness horizon (a date); a page past it is flagged `stale` by `kb_lint` (the *content-may-be-outdated* signal, distinct from a pin-drift) |
| `generated` | engine-owned | `{ by: mnemonic/<ver>, at: <ts> }` — stamped by `kb_generate` on any body change. **The agent must not author it.** A body change (even whitespace) advances it; frontmatter-only changes do not. |
| `verified` | engine-owned | a **history** (append-only) of `{ by, at }` machine-verification events. `kb_validate` **appends** a new event when the pin-check passes (it does not truncate — the list is the verification log, not a single latest stamp). Only `mnemonic/…`-actor events are reconciled (removed) by the engine on a body change; human/other-process events are preserved. A page is "verified" iff its *latest* `mnemonic/…` event postdates its latest `generated` stamp. |

> **OKF v0.2 control fields** (`status`, `stale_after`, `generated`, `verified`) are the
> engine's provenance/lifecycle layer (pi-llm-wiki's OKF integration). The agent authors
> `title`/`type`/`project`/`aliases`/`anchors`/`claims`/`sources`; the engine owns the
> four control fields. This keeps the vault a conformant OKF bundle *and* separates
> "what the agent asserted" from "what the engine verified."

### 5.2 `anchors:` (pin targets)

An anchor is a (path, selector) pair the pin-check (`006-validate`) resolves against
**live code**. Three selector forms:

- `all_of: [token, token, …]` — every listed token/symbol must be present in the file
  (token-level, ripgrep). Used for "this construct still has these identifiers."
- `line: N` + `version: <token>` — a line range pinned by a **relocating version token**
  (openwiki's model; see `006-validate` §2). The token carries the span's content hash +
  span length + 6 relocation anchors, so the pin-check distinguishes *moved* (no work) from *changed*
  (`stale`) from *ambiguous* (`unresolved`). Drift = the token no longer resolves.
- `symbol: Name` — a symbol-level anchor, resolved through the **code-index graph**
  (`02-code-index`). Used for "this function still exists / is still called from X."

The **engine** (KB-O1, default): `all_of` → ripgrep (token-level); `line`+`version` →
ripgrep + the version-token resolver (no graph needed); `symbol` → the code-index graph
(symbol-level, call-graph aware). Both engines are available; a page may mix them.

### 5.3 `claims:` (openwiki grounded assertions)

A claim is a (subject, literal) pair — a concrete, checkable assertion. `kb_validate`
re-reads the live code and checks that the `literal` still holds at the anchor site.
A claim that no longer holds is a **drift** (the page is stale). This is the "must-hold
contract" repocodex enforces: the *why* (the page) is only as good as the *what still
holds* (the claim).

### 5.4 `sources:` (the citation format, provenance INV-1)

`sources:` entries are **citations, not stored packets** (KB-8). Four forms:

- `memory:<sync_id>` — a memory fact (resolved live from the memory store).
- `code:<path>#L<a>-<b>` — a code span (read live from the working tree / graph).
- `trajectory:TRJ-YYYY-MM-DD-NNN` — an agent **trajectory packet** (012, opt-in): the
  replayable tool-call sequence of a completed task. Resolves to
  `trajectories/TRJ-…/extracted.md` **in the vault** (a packet is a KB-generated
  input, not an external source — KB-8's cite-don't-store rule is about *external*
  content). A missing packet is a `broken_citation` lint finding.
- `external:<name>` + `version:` + `source_url:` + `source_commit:` +
  `build_fingerprint:` — an external source (a lib doc, a spec), cited in the
  **neuledge/context identity form**: `<name>` is the library (e.g. `somelib`),
  `version` is the pinned version, `source_url` is where the docs live, `source_commit`
  is the upstream revision the version was built from, and `build_fingerprint` is the
  staleness token. The KB cites it; the agent re-fetches the **section-granular** body
  on demand (see `002-sources` §2.4). Nothing is copied into the vault.

The citation is **stable provenance**: it is the link from a synthesized page back to the
raw fact. If the raw fact changes or disappears, `kb_lint` / `kb_validate` flag the
orphaned citation (the page may be stale).

## 6. Wikilinks

- **Form:** `[[type/slug]]` — folder-qualified (the type folder is the first path
  segment), relative to the vault root. Obsidian-compatible.
- **Resolution:** `[[decisions/auth-model]]` → `decisions/auth-model.md`. Links are
  **always folder-qualified** (`[[type/slug]]`); an *unqualified* `[[slug]]` is not a
  legal link. An optional alias form `[[type/slug|display text]]` is allowed (display
  text only — the target is still `type/slug`). The `aliases:` frontmatter field provides
  *alternate targets* (e.g. a page reachable as either `decisions/auth-model` or
  `decisions/authentication-model`); a link resolves if its `type/slug` matches the page's
  path **or any of its aliases** (lint + `kb_read` backlinks both use this).
- **Backlinks** are **computed on the fly** from the markdown (grep for `[[…]]`, with
  alias resolution), never stored. `kb_query` and `kb_lint` compute them when needed (the
  *count* for `kb_query`'s ranking is cached in `kb_pages.in_degree`, `004-index` §3 —
  a count, not the graph). This keeps the index rebuild cheap and the backlink graph
  always consistent with the files.
- **Cross-reference everything** (golden rule): every page needs ≥2 **outgoing** links
  (wikilinks). The **orphan** check (`007-lint`) is the *incoming* side: a page with
  in-degree ≤ 1 (alias-resolved) is an orphan candidate. Outgoing (golden rule) and
  incoming (lint) are distinct — a page can satisfy the golden rule and still be an
  orphan if nothing links *to* it.

## 7. `.knowledgeignore` (the read boundary)

An openwiki-style `.openwikiignore` adapted: patterns that exclude paths from
**indexing, generation, and linting**. By default the vault only reads its own page
folders; the ignore file controls (a) which external/cited paths are scanned for
anchors, and (b) which generated/scratch files are skipped. It does **not** control the
shared DB (that is the store's concern). Syntactically gitignore-like.

**Index scope is a whitelist, not a blacklist.** `kb_index` / `kb_reindex` (004 §4)
scan exactly the page folders — `decisions/ invariants/ workflows/ architecture/
gotchas/ glossary/` (+ opt-in `skills/ cases/`, 012) — and nothing else. Everything
else under the vault root is out of index scope by construction: `index/` (derived
projections), `templates/`, `.obsidian/` and other dot-dirs, `KNOWLEDGE_SCHEMA.md`,
and the vault-root project docs (`ARCHITECTURE.md` / `ASSUMPTIONS.md` / `WINDOWS.md`).
Generated files and dot-dirs can therefore never become pages. (The shipped
`collectPages` is a bare `WalkDir` over the whole vault and does not honor
`.knowledgeignore` — it must be brought to this scope; KB-T16.) `.knowledgeignore`
layers **on top of** the whitelist — per-folder / per-page exclusions *within* scope —
it never widens it.

## 8. `config.json` (the sentinel)

The sentinel that marks a valid KB. Minimal fields:

```json
{
  "schema_version": 1,
  "project": "skillgrid",
  "page_types": ["decision","invariant","workflow","architecture","gotcha","glossary"],
  "anchor_engine": "auto",            // "auto" | "ripgrep" | "graph" (KB-O1)
  "strict": false,                     // pin-check fails the turn when true (KB-O3)
  "trajectories": false,               // 012: working-memory (skill/case) opt-in; the
                                       //   trajectory tools exist only when true (the
                                       //   skills/ cases/ trajectories/ dirs are created
                                       //   at bootstrap / first capture)
  "recall": {                          // 005 §2.3: the stage-1 inline carve-out
    "inline_types": ["invariant","gotcha","skill","case"],
    "inline_max": 1600,                // chars; a page body longer is links-only
    "links_threshold": 50              // page count below → previews inline for all types
  }
}
```

`kb_bootstrap` writes the scaffold (`config.json`, `KNOWLEDGE_SCHEMA.md`, `templates/`,
the empty page folders, `.knowledgeignore`) when a project has no KB yet. It is
**idempotent** — a second `kb_bootstrap` on an existing vault is a no-op.

## 9. Invariants

KB-INV-1 (files are truth), KB-INV-2 (grounded pages are grounded), KB-INV-3 (index
never hand-edited), INV-1 (provenance via `sources:`), INV-4 (local truth — the markdown
is the truth, the index is derived).

## 10. Open items

None new — the layout, taxonomy, and format are fixed. KB-O1 (anchor engine default) and
KB-O3 (strict mode) are carried to `006-validate` / `008-transport`.
