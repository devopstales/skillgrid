---
name: llm-wiki
description: "Curate a persistent interlinked markdown wiki (Karpathy pattern): ingest sources into raw/, maintain entity/concept pages with [[wikilinks]], query compiled knowledge, lint health. Use for ~/wiki, ~/.skillgrid/wiki, or as read context for project .wiki/."
license: MIT
metadata:
  author: Hermes Agent (ported; see ADR-0015)
  version: "1.0"
  part-of: skillgrid
---

# LLM Wiki — Curated Markdown Knowledge Base

**Announce at start:** "I'm using the skillgrid:llm-wiki skill to work this wiki."

## Overview

Build and maintain a persistent, compounding knowledge base as interlinked markdown files (Karpathy's LLM Wiki pattern). Unlike per-query RAG, knowledge compiles once and stays current: cross-references already exist, contradictions are flagged, synthesis reflects everything ingested. The human curates sources and directs analysis; the agent summarizes, cross-references, files, and maintains consistency.

Vault mechanics (read/search/create/edit) live in `skillgrid:obsidian` — this skill owns curation judgment. Outside facts synthesized into pages cite per `skillgrid:grounded-citations`.

**Three targets, one skill.** Per ADR-0015 the Go `wiki` compiler is just the export function — it alone writes project `.wiki/`:

| Target | Path | Role |
|---|---|---|
| Personal vault | `~/wiki` (`WIKI_PATH`, default) | user-directed curation, any domain |
| Machine store | `~/.skillgrid/wiki` (`SKILLGRID_WIKI_PATH`, default) | agent-kept project memory projection |
| Project export | `<project>/.wiki/` | **read-only context** — never write here; the compiler owns it |

Select (or confirm) the target before any operation. All paths below use `$WIKI` for the active target.

Ported from the Hermes `llm-wiki` skill under MIT (see ADR-0015). Behavioral deltas: Hermes tool names mapped to skillgrid tools; retrieval via Exa/WebFetch/mnemonic webcache; the `obsidian-headless` + Sync-subscription tail dropped (paid account, machine-level concern — the vault stays Obsidian-compatible without it).

## When to Use

- Create, ingest into, query, lint, or audit a wiki / knowledge base / notes corpus
- A question arrives and a wiki exists at the configured path (orient first, then answer)
- Filing a substantial synthesis (comparison, deep dive) back into `queries/` or `comparisons/`

**When NOT to use:** Project decisions/specs/ADRs — those live in `.skillgrid/` (repo source of truth), never the wiki. Writing project `.wiki/` pages directly — compiler-owned. Trivial lookups — answer, don't file. One-off research with no compounding value — `skillgrid:research` instead.

## Layout (three layers)

```
$WIKI/
├── SCHEMA.md    # Layer 3: conventions, structure rules, tag taxonomy (agent reads, human approves changes)
├── index.md     # sectioned catalog, one line per page
├── log.md       # append-only action log, rotated yearly
├── raw/         # Layer 1: IMMUTABLE sources (articles/, papers/, transcripts/, assets/)
├── entities/    # Layer 2: people, orgs, products, models
├── concepts/    # Layer 2: topics
├── comparisons/ # Layer 2: side-by-side analyses
└── queries/     # Layer 2: filed answers worth keeping
```

**Layer 1 raw/ is immutable** — the agent reads, never modifies. Corrections go in wiki pages. See `references/schema-templates.md` for SCHEMA/index/log/frontmatter templates.

## The Process

### 0. Orient (every session, before anything)

Read `SCHEMA.md` (domain, taxonomy) → `index.md` (what exists) → last 20–30 `log.md` entries (recent activity). For 100+ pages also Grep the topic first. Skipping orientation causes duplicates, missed cross-refs, and contradicted conventions.

### 1. Initialize (new wiki)

Confirm target path → create structure → ask the domain → write domain-customized `SCHEMA.md` → seed `index.md` + `log.md` → suggest first sources.

### 2. Operations

Full procedures live in `references/wiki-operations.md`: **ingest** (capture raw + frontmatter/sha256 → discuss takeaways → dedupe-check → write/update pages → provenance/confidence → update index+log → report), **query** (index → read → synthesize citing `[[pages]]` → file if painful-to-rederive → log), **lint** (13 checks: orphans, broken links, index completeness, frontmatter, staleness, contradictions, quality signals, source drift, page size, tags, log rotation → severity-grouped report → log), **bulk ingest**, **archiving** (`_archive/`, delink, log).

### Standing rules

- Page thresholds: create at 2+ source mentions or central-to-one-source; split past ~200 lines; index sections split past 50 entries; rotate log past 500 entries.
- Every page: YAML frontmatter (title/created/updated/type/tags/sources + optional confidence/contested/contradictions), tags from taxonomy only (add to SCHEMA first), ≥ 2 outbound `[[wikilinks]]`, `updated` bumped on edit.
- Contradictions: note both claims with dates, mark frontmatter, surface in lint — never silently overwrite.
- Provenance: 3+ source syntheses carry `^[raw/...]` paragraph markers; single-source pages ride on `sources:` frontmatter.
- Mass updates touching 10+ pages: confirm scope with the user first.
- Obsidian-compat kept (wikilinks, frontmatter, `raw/assets/` attachments); pair with `skillgrid:obsidian` for vault mechanics.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I'll skip orientation, I know this wiki" | Orientation is the difference between a growing wiki and a pile of duplicates. SCHEMA + index + log, every session. |
| "This mention deserves a page" | Thresholds exist to stop sprawl: 2+ sources or central-to-one. Passing mentions don't qualify. |
| "Index/log update is bookkeeping, later" | The index IS navigation and the log IS memory. Pages without them degrade the wiki on arrival. |
| "One source is enough for high confidence" | `confidence: high` requires multi-source support. Single-source claims say `medium` or `low`. |
| "I'll fix the contradiction by overwriting" | Both claims, both dates, frontmatter flag, lint surfaces it. Overwriting destroys the trail. |
| "The compiler will handle project knowledge" | The compiler exports `.wiki/` from `.skillgrid/` + webcache. Curation judgment here, in the skill — different jobs. |

## Red Flags

- Writing or editing anything under `raw/` (immutable by design)
- Creating pages without cross-references (isolated pages are invisible)
- Freeform tags outside the SCHEMA taxonomy
- Filing trivial lookups into `queries/` (only painful-to-rederive answers)
- Any write to project `.wiki/` (compiler-owned; ADR-0014)
- Mass ingest/update without user-confirmed scope

## Verification

- [ ] Target (`~/wiki` / `~/.skillgrid/wiki` / `.wiki`-read-only) confirmed before operating
- [ ] Oriented: SCHEMA + index + recent log read this session
- [ ] New pages meet thresholds, carry frontmatter + taxonomy tags + ≥ 2 wikilinks, appear in `index.md`
- [ ] `updated` bumped on edits; contradictions flagged, not overwritten
- [ ] `log.md` appended for every action; raw/ untouched; nothing written to project `.wiki/`
