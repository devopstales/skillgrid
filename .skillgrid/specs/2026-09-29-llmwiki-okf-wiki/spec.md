# Spec: llmwiki-okf-wiki — OKF v0.2 `.wiki/` Compiler (Pillar 1)

> **STATUS:** `draft` (2026-09-29)
>
> **Spec type:** New capability `wiki` (full spec).
> **Briefing:** `.skillgrid/specs/2026-09-29-llmwiki-okf-wiki/briefing.md`
> **Threat matrix:** see `design.md` §Threats (carried into the scenarios below).

Keywords MUST / MUST NOT / SHOULD / MAY are per RFC 2119.

---

## Capability: OKF v0.2 + llmwiki Emitter

### R1: The emitter MUST produce valid OKF v0.2 frontmatter

The emitter (`wiki.Emit`) MUST serialize a `Concept` to a UTF-8 markdown document with a YAML frontmatter block delimited by `---` and a markdown body. The frontmatter MUST include `type` (the only always-required OKF key) and MUST use absolute UTC ISO 8601 (`…T…Z`) for every timestamp-valued key (`generated.at`, `verified[].at`, `sources[].last_modified`, `stale_after`).

**Scenario 1.1 — Minimal conformant page (happy path)**
- Given a `Concept` with only `type: ADR`, `title`, and a non-empty body
- When `Emit(c)` is called
- Then the output begins with `---\n`, contains `type: ADR`, and `Conform` on the output returns no errors
- And the body appears verbatim after the closing `---`

**Scenario 1.2 — `stale_after` MUST be an absolute instant (edge)**
- Given a `Concept` with `generated.at = 2026-09-29T14:00:00Z` and a 90-day stale horizon
- When the compiler computes `stale_after`
- Then `stale_after` equals `2026-12-28T14:00:00Z` (an absolute instant, NOT a relative TTL string like `90d`)

**Scenario 1.3 — Absent optional keys MUST NOT be serialized as null**
- Given a `Concept` with no `tags`, no `verified`, and no `resource`
- When `Emit(c)` is called
- Then the frontmatter MUST NOT contain `tags: null`, `verified: null`, or `resource: null` keys (absence is meaningful per OKF §5)

### R2: The conformance gate MUST validate OKF v0.2 fields

`wiki.Conform(page)` MUST return a list of errors (empty when conformant) and MUST check, at minimum: (a) `type` is present and non-empty; (b) every `sources[]` entry has a non-empty `resource`; (c) every timestamp parses as absolute UTC ISO 8601; (d) `status` (if present) is one of `draft|stable|deprecated`. `wiki lint` MUST exit non-zero if any page fails.

**Scenario 2.1 — Missing `type` is a conformance failure**
- Given a page whose frontmatter has no `type`
- When `Conform(page)` is called
- Then it returns ≥1 error naming the missing `type`
- And `wiki lint` over a bundle containing that page exits non-zero

**Scenario 2.2 — `sources[].resource` required within an entry**
- Given a page with `sources: [{ author: team:x }]` (no `resource`)
- When `Conform(page)` is called
- Then it returns an error identifying the source entry missing `resource`

**Scenario 2.3 — All emitted pages MUST pass lint (happy path)**
- Given a full `.wiki/` produced by `wiki compile`
- When `wiki lint` is run against it
- Then every page under `wiki/` returns no errors and exit code is 0

### R3: The emitter MUST carry the llmwiki/Obsidian extension keys

For every page whose source is a local file, the emitter MUST set `source_path` (bundle-relative path into `raw/` for raw-indexed pages, or the `.skillgrid/` path for pillar-1 pages). `source_path` is a producer-defined extension key permitted by OKF §4.1 and is the llmwiki backlink target.

**Scenario 3.1 — raw-indexed page points at `raw/`**
- Given a user-dropped `raw/web/my-clipper-note.md`
- When the compiler emits its page
- Then the page frontmatter contains `source_path: raw/web/my-clipper-note.md`

**Scenario 3.2 — pillar-1 page points at the `.skillgrid/` path**
- Given ADR-0006 in `.skillgrid/ASSUMPTIONS.md`
- When the compiler emits its page
- Then the page frontmatter contains `source_path` referencing the `ASSUMPTIONS.md` path (not a `raw/` path)

---

## Capability: Pillar-1 `.skillgrid/` Source Adapter

### R4: The adapter MUST parse `.skillgrid/` pillar-1 sources into concepts

The adapter MUST read (read-only) and parse: ADRs from `ASSUMPTIONS.md` (in-force set + superseded, each → a page), Terms from `artifacts/01-business-terms.md` and `02-technical-terms.md`, project `State` from `state.yaml`, `Spec` concepts from `specs/*/` (per change dir), `Spike` concepts from `spikes/*/`, `Constraint` concepts from the `### Locked constraints` section, and `Architecture` from `ARCHITECTURE.md`. Parsing MUST be tolerant: a missing file or section yields no concepts (not an error), except where a section is structurally required (see R4 scenarios).

**Scenario 4.1 — ADRs in-force and superseded (happy path)**
- Given `.skillgrid/ASSUMPTIONS.md` with 12 ADRs where ADR-0009 supersedes ADR-0004
- When the adapter parses it
- Then it emits 12 ADR concepts, ADR-0004 with `status: deprecated`, ADR-0009 with `status: stable`
- And ADR-0009's body contains a `[[wikilink]]` to ADR-0004

**Scenario 4.2 — Superseded ADR is `deprecated` with a back-link (edge)**
- Given ADR-0004 superseded by ADR-0009
- When the adapter parses
- Then ADR-0004's frontmatter has `status: deprecated`
- And ADR-0004's body links to ADR-0009 (the superseder)

**Scenario 4.3 — Locked constraints → `Constraint` concepts (happy path)**
- Given `.skillgrid/ASSUMPTIONS.md` `### Locked constraints` with 3 bullet items
- When the adapter parses
- Then it emits 3 `Constraint` concepts, each with `status: stable` and NO `stale_after` (constraints are timeless)

**Scenario 4.4 — Missing ARCHITECTURE.md is existence-gated (failure/edge)**
- Given the repo has no `.skillgrid/ARCHITECTURE.md`
- When the adapter parses
- Then it emits NO `Architecture` concept and returns no error
- And a subsequent compile with the file present emits exactly one `Architecture` concept

**Scenario 4.5 — Malformed section does not fail the compile (failure)**
- Given `state.yaml` with an unexpected extra top-level key
- When the adapter parses
- Then it still emits the `State` concept from the known keys and surfaces the unknown key as a lint warning (not a hard error)

---

## Capability: web_cache Indexer (hybrid, option C)

### R5: The indexer MUST read fresh web_cache rows directly from SQLite

The indexer MUST read `web_cache` rows scoped to the project where `source ∈ {context7, exa, deepwiki, fetch}` and the row is fresh (`expires_at IS NULL OR expires_at > now`). It MUST NOT write any file into `raw/` for these rows. It MUST derive each `Finding` page's `sources[].resource` from the row's `url`, `verified` from `{ by: <source tool actor>, at: <fetched_at> }`, and `stale_after` = `fetched_at + 90d`.

**Scenario 5.1 — Fresh cited row → `Finding` (happy path)**
- Given a fresh `exa` row (url U, fetched_at F) whose url U is cited in the `sources` of an emitted ADR page
- When the compiler runs
- Then it emits a `Finding` page for U under `wiki/` with `sources: [{resource: U, author: process:exa}]`, `verified: [{by: process:exa, at: F}]`
- And NO file is created under `raw/` for U

**Scenario 5.2 — Fresh uncited row → `wiki/research/` draft (edge)**
- Given a fresh `context7` row whose url is cited by NO emitted page
- When the compiler runs
- Then it emits the row as a `Finding` under `wiki/research/` with `status: draft`
- And it is listed in `wiki/index.md` under a "Research (uncited)" grouping

**Scenario 5.3 — Expired row is skipped (edge)**
- Given a `fetch` row with `expires_at` in the past
- When the compiler runs
- Then it emits NO page for that row

**Scenario 5.4 — Row with no url degrades gracefully (failure)**
- Given a fresh `manual`-excluded row set, and a `fetch` row with an empty `url`
- When the compiler runs
- Then it emits the row using its `title` (or a slug) as the page title and `sources[].resource` falls back to a stable internal descriptor; no panic

---

## Capability: `raw/` Indexer (user-dropped markdown)

### R6: The indexer MUST index user-dropped `raw/**/*.md` and never write to `raw/`

The indexer MUST walk `raw/` recursively (any depth) and, for each `.md` file, emit a concept. Files with valid OKF frontmatter are indexed with their `type` (defaulting to `Finding` when absent) and `source_path: raw/<path>`. Files WITHOUT frontmatter (or with no `type`) become bare `Source` pages (filename → title, `verified` absent). The compiler MUST NOT create, modify, or delete any file inside `raw/`.

**Scenario 6.1 — OKF-frontmatter clipper note (happy path)**
- Given `raw/web/notes/sigma-graph.md` with frontmatter `type: Finding`, `title`, `sources: [{resource: https://…}]`
- When the compiler runs
- Then it emits a `Finding` page with `source_path: raw/web/notes/sigma-graph.md` and the body preserved
- And the bytes of `raw/web/notes/sigma-graph.md` are unchanged after the compile

**Scenario 6.2 — Non-OKF clipper note → bare `Source` (edge)**
- Given `raw/loose-note.md` with no frontmatter (plain markdown)
- When the compiler runs
- Then it emits a `Source` page titled `loose-note` with `verified` absent and `source_path: raw/loose-note.md`
- And `wiki lint` still passes for it (a `Source` with no `sources` is conformant)

**Scenario 6.3 — `raw/` is never written (failure)**
- Given a `raw/` tree with known contents
- When `wiki compile` runs
- Then a before/after hash of every file under `raw/` is identical

**Scenario 6.4 — Empty `raw/` is fine (edge)**
- Given a repo with no `raw/` directory
- When `wiki compile` runs
- Then it succeeds, emits no raw-derived pages, and creates no `raw/` directory

---

## Capability: Compiler (assembly, determinism, output)

### R7: The compiler MUST write a complete, well-formed `.wiki/` tree

`wiki compile` MUST (re)write `wiki/` (all subdirs, `index.md`, `log.md`) and `AGENTS.md` (the schema layer), MUST leave `raw/` untouched, and MUST generate `wiki/index.md` as a catalog of all pages grouped by `type` (progressive disclosure for llmwiki) and append a single line to `wiki/log.md` per compile that changed content.

**Scenario 7.1 — Full tree on first compile (happy path)**
- Given a repo with `.skillgrid/` + `raw/` + a seeded `web_cache`
- When `wiki compile` runs on an empty `.wiki/`
- Then `.wiki/AGENTS.md`, `.wiki/wiki/index.md`, `.wiki/wiki/log.md`, and per-type subdirs exist
- And `wiki/index.md` lists every emitted page

**Scenario 7.2 — `AGENTS.md` documents the schema (happy path)**
- Given any compile
- When `AGENTS.md` is written
- Then it contains the `type` taxonomy (ADR, Term, Spec, Spike, State, Constraint, Architecture, Finding, Source), the slug rule, and the lint rule

### R8: The compile MUST be deterministic and no-churn

The compiler MUST be a pure function of (`.skillgrid/` contents, fresh `web_cache` rows, `raw/` contents, accepted inferred edges, compile clock). It MUST sort concepts and edges stably (by type, then id), MUST preserve the prior `generated.at` for any concept whose content is unchanged, and MUST only rewrite files whose content changed (content-hash gate). Two consecutive no-op compiles MUST produce byte-identical `.wiki/`.

**Scenario 8.1 — No-op recompile is byte-identical (happy path / determinism)**
- Given a `.wiki/` produced by a compile
- When `wiki compile` is run a second time with unchanged sources and the same clock
- Then the full byte-set of `.wiki/` is identical (a `git diff` of `.wiki/` is empty)

**Scenario 8.2 — Unchanged concept keeps its `generated.at` (edge)**
- Given ADR-0006 compiled at time T with `generated.at: T`
- When ADR-0003's source changes (ADR-0006 unchanged) and a recompile runs at time T+1
- Then ADR-0006's `generated.at` is still `T` (not `T+1`)
- And only ADR-0003's page file is rewritten

**Scenario 8.3 — Deterministic ordering (happy path)**
- Given two compiles over the same inputs
- When the page list and `index.md` ordering are compared
- Then the order is identical (stable sort by (type, id)) on both runs

### R9: The CLI MUST expose `wiki compile` and `wiki lint`

The `skillgrid` binary MUST gain a `wiki` command group with `compile` (writes `.wiki/`, flags `--project`, `--dir`, `--out`, `--no-llm` [no-op in Pillar 1], `--json`) and `lint` (validates `.wiki/`, non-zero exit on failure). Project + data dir resolution MUST reuse the existing `openMemService` pattern (CWD-resolved project, `~/.skillgrid/mnemonic` default data dir).

**Scenario 9.1 — `wiki compile` resolves project from CWD (happy path)**
- Given the CWD is the skillgrid repo and no `--project` flag
- When `skillgrid wiki compile` runs
- Then it compiles for the CWD-resolved project and writes `.wiki/` at the repo root

**Scenario 9.2 — `wiki lint` fails on a bad page (failure)**
- Given a hand-edited `.wiki/wiki/adr/x.md` missing `type`
- When `skillgrid wiki lint` runs
- Then it prints the conformance error and exits non-zero

**Scenario 9.3 — `wiki` with no subcommand prints usage (edge)**
- Given `skillgrid wiki` (no subcommand)
- When it runs
- Then it prints available subcommands (`compile`, `lint`) and exits 0
