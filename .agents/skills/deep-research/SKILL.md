---
name: deep-research
description: >
  Structured web research using search MCPs and REST fallbacks (Exa, Tavily, Firecrawl).
  Trigger: SDD explore/brainstorm, technology choices, competitor patterns, current docs, or any task needing external context before local code reading.
  Use as the **first search** step — run before codebase investigation unless the topic is purely internal.
license: Apache-2.0
metadata:
  author: devopstales
  version: "1.0"
---

# Deep Research

Orchestrates **external** web research so SDD phases start with current, cited context — not stale training data.

## First search rule (non-negotiable)

When this skill is loaded (especially from `sdd-explore` or `sdd-brainstorm`):

1. **Run external research first** — before reading application source, running repo-wide code search, or proposing approaches grounded only in the codebase.
2. **Skip only when** the topic is strictly internal (rename refactor, bug in known file, “where is X defined”) with no library, API, security, or product landscape angle. State `first_search: skipped` and why.
3. **Then** investigate the codebase and merge local + external findings in the exploration artifact.

## When to activate

- New feature, integration, or stack choice
- Unfamiliar library, protocol, or cloud service
- Security, compliance, or “current best practice” questions
- Competitor / market / UX pattern research
- User says explore, brainstorm, research, compare approaches, or what’s the latest

## Provider stack (use in order)

Try each tier until you have enough cited evidence (typically 3–8 quality sources). Do not repeat the same query across providers unless the first returned thin results.

### 1. Exa MCP (preferred)

**Server:** `user-exa-http` (repo documents this as the Exa MCP integration).

Before calling, read tool schemas under the MCP descriptors folder for `user-exa-http`.

| Tool | Use for |
|------|---------|
| `web_search_exa` | Broad discovery, news, companies, “latest X”, comparisons |
| `web_fetch_exa` | Full page/markdown for 1–3 URLs from search when snippets are insufficient |
| `get_code_context_exa` | API usage, code examples, framework patterns (if exposed on your server) |

Example flow:

```text
web_search_exa(query: "<rich natural language query>", numResults: 5–8)
→ optional web_fetch_exa(urls: [top 2–3 URLs])
```

See also: `exa-search` skill for query tips and parameters.

### 2. Tavily (REST fallback)

Use when Exa is unavailable, rate-limited, or returned too few results.

- Load `tavily` skill — requires `TAVILY_API_KEY`
- Prefer `POST /search` with `include_answer: true`, `search_depth: advanced` for planning research
- Use `POST /extract` for specific official docs URLs

### 3. Firecrawl CLI (optional)

When Exa/Tavily lack depth on documentation sites:

- Load `firecrawl-search` / `firecrawl` skills if installed (`firecrawl search "…" --scrape`)
- Requires `FIRECRAWL_API_KEY` and CLI on PATH

### 4. Built-in web search (last resort)

Use the host’s `WebSearch` tool only when no MCP/CLI provider is configured. Note `providers_used: [websearch_builtin]` in output.

## Research procedure

### A. Frame queries (1–3 minutes)

From the exploration topic, derive:

- **Primary query** — decision-oriented (“how do teams implement X in 2025”, not just “X”)
- **Docs query** — official docs / RFC / spec (`site:docs.vendor.com …` or Exa code context)
- **Risk query** — security, deprecation, breaking changes (optional)

Record queries in output under `queries_run`.

### B. Execute first search

1. Run tier 1 (Exa MCP). If weak: tier 2 (Tavily). If still weak: tier 3/4.
2. Prefer **primary sources**: official docs, specs, maintainer blogs, reputable engineering posts.
3. Capture **URL, title, date if known, one-line takeaway** per source.
4. Do not treat a single blog post as consensus; note disagreements.

### C. Synthesize (before codebase)

Produce a short block for the parent phase:

```markdown
## External research (first search)

**Providers:** exa | tavily | firecrawl | websearch_builtin  
**Skipped:** no / yes — <reason>

### Queries
- …

### Findings
1. …
2. …

### Implications for this change
- …

### Sources
| Source | URL | Note |
|--------|-----|------|
| … | … | … |

### Open questions (web could not answer)
- …
```

### D. Handoff to codebase work

After this block exists, proceed to repo investigation (`sdd-explore` Step 4+) and explicitly **reconcile** external findings with what the code actually does.

## Parallel research

For multiple independent questions (e.g. auth provider A vs B vs C), use `parallel-delegate` with one **read-only** child per question, each child following this skill’s first-search rule. Merge in the coordinator; write long dumps to `.skillgrid/tasks/research/<change-id>/web-<slug>.md`.

## Persistence (Skillgrid)

When tied to a change:

- Include the **External research** section in `exploration.md` or engram `sdd/{change}/explore`
- Optional spill file: `.skillgrid/tasks/research/<change-id>/first-search.md`

## Output contract

Return to the orchestrator:

- The markdown block above (required when first search ran)
- `providers_used`: array of provider ids actually invoked
- `first_search`: `completed` | `skipped`

When embedded in `sdd-explore`, the parent phase still returns the standard SDD envelope; put this material in `detailed_report` and reference paths in `artifacts`.

## Rules

- **MUST** run first search before codebase investigation when external context can change the recommendation.
- **MUST** cite URLs for non-obvious factual claims from the web.
- **MUST NOT** skip search because codebase search is faster.
- **MUST NOT** treat MCP snippets as full docs — fetch when implementing or when snippets conflict.
- **SHOULD** prefer Exa MCP when `user-exa-http` is available.
- **SHOULD** keep first-search synthesis under ~40 lines; link spill files for long notes.

## Related skills

- `exa-search` — Exa tool reference and query patterns
- `tavily` — REST API when MCP is unavailable
- `parallel-delegate` — parallel research lanes
- `sdd-explore` — consumes first search in planning explore phase
- `context7` — library docs **after** first search when a specific package is chosen (not a substitute for landscape search)
