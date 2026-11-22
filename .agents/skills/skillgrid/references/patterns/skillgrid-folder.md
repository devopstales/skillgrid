# The `.skillgrid` folder

The layout this skill writes in the programmer's project root. `projectwiki/` follows the LLM-wiki pattern: raw sources are immutable, the agent writes and maintains the wiki, and the wiki is the durable record. The existing `ASSUMPTIONS.md`, `ARCHITECTURE.md` and `glossary/` are kept and extended.

When to load: Phase 0, and whenever a path is in doubt. Role: orchestrator.

```text
.worktree/                      # git worktree root, gitignored
.skillgrid/
  config.yaml                   # project settings (ticketing, testing, gates, tiers)
  state.yaml                    # dynamic state: phase, current change, genesis block
  work/<YYYY-MM-DD-topic>/      # in-flight execution scratch + ledger, gitignored
    progress.md                   # SDD ledger (the recovery map)
    parallel-ledger.md            # fan-out ledger (one row per agent)
    briefs/<unit>.md
    attempts/<unit>/attempt-1.md
    attempts/<unit>/revision-rK.md
    packets/
  prototype/<YYYY-MM-DD-topic>/ # feasibility probes, permanently retained
  change-specs/<YYYY-MM-DD-topic>/
    briefing.md
    blueprint.md
    acceptance.feature
    tasks.md
    report.md
    review.md
    findings.md
    ledger.md
    evaluations/
  change-archive/<YYYY-MM-DD-topic>/   # closed changes, immutable
  projectwiki/
    ASSUMPTIONS.md              # durable project understanding: VERIFIED / INFERRED / LOCKED / Open Questions
    ARCHITECTURE.md             # live repo/program structure, 17-section skeleton
    index.md                    # wiki catalog: every page, one-line summary, by category
    log.md                      # append-only timeline: ## [YYYY-MM-DD] <op> | <title>
    backlog/
    glossary/business-terms.md
    glossary/technical-terms.md
    decisions/NNNN-slug.md
    raw/                        # immutable source docs: research findings, ADR context, prior-art
      assets/
    sources/                    # one summary page per raw doc, with citations
    entities/                   # things: services, APIs, users, data stores, external systems
    concepts/                   # ideas: invariants, patterns, error model, config precedence
    syntheses/                  # cross-entity analyses, comparisons, overviews
  cache/                        # optional, gitignored
```

## The wiki

`projectwiki/` is the LLM-wiki. The agent writes and maintains it; the programmer reads, sources and directs. The existing `ASSUMPTIONS.md` (project understanding), `ARCHITECTURE.md` (structure) and `glossary/` (vocabulary) are kept as the entry points a reader opens first. The wiki directories (`raw/`, `sources/`, `entities/`, `concepts/`, `syntheses/`) are the agent's working layer where new knowledge is ingested, cross-referenced and kept current.

| Path | Writer | Content |
|---|---|---|
| `ASSUMPTIONS.md` | agent + programmer | the four tiers: VERIFIED facts, INFERRED hypotheses, LOCKED ADR index, Open Questions. The first file a reader opens. |
| `ARCHITECTURE.md` | agent (scaffolded by `skillgrid init`, filled by onboarding) | 17-section skeleton: live repo/program structure with `<detect>` markers replaced by detected facts. |
| `glossary/*.md` | agent | one meaning per term. Seeded from `references/patterns/glossary.md`; maintained by the ADR skill. |
| `invariants/locked-constraints.md` | agent | testable claims the code must never break. |
| `raw/` | programmer | immutable source docs: research findings, ADR context, prior-art, sketches. Never edited by the agent. |
| `sources/` | agent | one summary page per raw doc, with citations back to `raw/`. |
| `entities/` | agent | pages for the things the program touches: services, APIs, data stores, external systems, user roles. |
| `concepts/` | agent | pages for the ideas: invariants, patterns, error model, config precedence, one-way doors. |
| `decisions/NNNN-slug.md` | agent | one ADR per file. The pointer row lives in `ASSUMPTIONS.md` § LOCKED and `index.md`. |
| `syntheses/` | agent | cross-entity analyses that don't belong to a single page: comparisons, overviews, risk maps. |
| `index.md` | agent | the catalog. Every wiki page listed with a link and a one-line summary, grouped by directory. Updated on every ingest. |
| `log.md` | agent | append-only timeline. Each entry: `## [YYYY-MM-DD] <ingest|query|lint|adr> | <title>`. Newest last. |

**Ingest.** A new raw source (a research finding, an ADR, a prototype result) is processed: the agent writes a `sources/` page, updates or creates the relevant `entities/` and `concepts/` pages, updates `ASSUMPTIONS.md` if a tier changes, updates `index.md`, and appends to `log.md`. A single ingest may touch 5–15 pages.

**Query.** The agent reads `index.md` first, drills into the relevant pages, and synthesizes an answer. A good answer that doesn't already exist gets filed back as a `syntheses/` page.

**Lint.** Before Phase 4 and at the end of every wave, the agent health-checks the wiki: contradictions between pages, stale claims superseded by newer sources, orphan pages with no inbound links, concepts mentioned but lacking their own page.

**Extend, don't create.** Before creating a page, check `index.md` and the existing pages. Near-duplicate pages are the default failure mode.

## Paths that are easy to get wrong

| Use | Not |
|---|---|
| `.skillgrid/state.yaml` | `.skillgrid/meta/state.yaml` |
| `.skillgrid/work/<YYYY-MM-DD-topic>/` | `.skillgrid/meta/.sdd/<plan>/` (legacy — merged into `work/`) |
| `.skillgrid/prototype/<YYYY-MM-DD-topic>/` | `.skillgrid/meta/.prototype/` |
| `.skillgrid/change-specs/<topic>/` | `.skillgrid/specs/<topic>/` |
| `.skillgrid/change-archive/<topic>/` | `.skillgrid/archive/<topic>/` |
| `.skillgrid/projectwiki/ASSUMPTIONS.md` | a sidecar `assumptions.yaml` |
| `.skillgrid/projectwiki/index.md` | a separate `glossary/` index |

## Rules

- A change folder is created at Phase 0 and never renamed mid-run. Its slug is `YYYY-MM-DD-topic`.
- `change-archive/` is immutable. A defect found after `ship` is a new change.
- `work/` is gitignored and ephemeral. `prototype/` is permanently retained — it is evidence, not scratch.
- `raw/` is immutable. The agent reads from it, never edits it.
- `ASSUMPTIONS.md` is the first file a reader opens. `index.md` is the first file the agent opens.
- `index.md` is updated on every ingest. A wiki page without an index entry is an orphan.
- The durable record is `projectwiki/` plus the change's `report.md`. Anything only in `work/` is lost when the run is cleaned.
- This skill never creates a `.skillgrid/` folder inside its own bundle. The template files in `references/patterns/` are copied into the programmer's project, not kept live here.
