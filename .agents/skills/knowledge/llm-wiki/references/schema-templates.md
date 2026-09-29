# llm-wiki Schema Templates: SCHEMA, frontmatter, index, log

Copy shapes for `skillgrid:llm-wiki`. Adapt domain, tags, and thresholds;
keep the field names (lint validates them).

## SCHEMA.md

```markdown
# Wiki Schema

## Domain
[What this wiki covers — e.g. "AI/ML research", "project memory"]

## Conventions
- File names: lowercase, hyphens, no spaces (e.g. `transformer-architecture.md`)
- Every wiki page starts with YAML frontmatter (below)
- Use `[[wikilinks]]` (minimum 2 outbound links per page)
- Bump `updated` on every edit; new pages go into `index.md` under type
- Every action appended to `log.md`
- Provenance markers: on 3+ source syntheses, append `^[raw/articles/x.md]`
  to paragraphs tracing to a specific source

## Frontmatter
 title / created / updated / type (entity|concept|comparison|query|summary)
 / tags (taxonomy only) / sources (raw paths)
 Optional: confidence (high|medium|low), contested (true),
 contradictions ([other-slug])

## Tag Taxonomy
[10–20 top-level tags. Add here BEFORE using.]
Example: model, architecture, benchmark, person, company, technique,
  comparison, timeline, controversy, prediction

## Page Thresholds
- Create: 2+ source mentions, or central to one source
- Extend: mention of something already covered
- Never: passing mentions, minor details, out-of-domain
- Split past ~200 lines; archive when fully superseded

## Entity / Concept / Comparison pages
Entity: overview, key facts+dates, relationships ([[links]]), sources.
Concept: definition, current state, open questions/debates, related.
Comparison: what+why, dimensions table, verdict/synthesis, sources.

## Update Policy
Newer sources generally supersede; genuine conflicts keep both positions
with dates + sources, `contradictions:` frontmatter, user-review flag.
```

## raw/ frontmatter (drift detection)

```yaml
---
source_url: https://example.com/article
ingested: YYYY-MM-DD
sha256: <hex of body only, after the closing --->
---
```

## index.md

```markdown
# Wiki Index

> Content catalog: every page under its type, one-line summary each.
> Last updated: YYYY-MM-DD | Total pages: N

## Entities
## Concepts
## Comparisons
## Queries
```

Scaling: section past 50 entries → sub-sections; index past 200 →
`_meta/topic-map.md`.

## log.md

```markdown
# Wiki Log

> Append-only. `## [YYYY-MM-DD] action | subject`
> Actions: ingest, update, query, lint, create, archive, delete
> Past 500 entries → rotate to log-YYYY.md, start fresh.

## [YYYY-MM-DD] create | Wiki initialized
- Domain: [domain]
- Structure created with SCHEMA.md, index.md, log.md
```
