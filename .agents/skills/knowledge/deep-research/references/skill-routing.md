# Skill Routing (deep-research)

deep-research is an **orchestrator**, not an executor. The agent is a research
project manager; it delegates execution to the skillgrid skill that owns the
task. Read the relevant SKILL.md before dispatching.

## Route to skillgrid skills

| Research activity | Route to |
|-------------------|----------|
| Need a fact not in the codebase (one-off lookup) | `skillgrid:research` |
| Wide/high-stakes informational question (parallel fan-out + verify) | `skillgrid:code-research` |
| Throwaway feasibility experiment (will this approach work?) | `skillgrid:prototype` |
| Debugging a bug or unexpected result in the experiment | `skillgrid:structured-debugging` |
| Implementing the experiment code (test-first) | `skillgrid:test-driven-development` |
| Executing a written plan for the experiment | `skillgrid:subagent-execution` or `skillgrid:simple-execution` |
| Quality gate before trusting a result | `skillgrid:qa` |
| Reviewing the experiment code before merging | `skillgrid:requesting-code-review` or `skillgrid:parallel-code-review` |
| Recording architectural decisions from the research | `skillgrid:architectural-decision-records` |
| Resuming after compaction/session restart | `skillgrid:resume` |

## Literature tooling by role

Match the tool to the job — do not stop at one source:

| Role | Tool |
|------|------|
| Broad discovery, finding relevant papers/sources | `exa_web_search_exa` |
| Library/framework API docs (current version) | `context7` |
| Understanding a GitHub repo's structure | `deepwiki` |
| Reading a specific URL / paper / page | `webfetch` / `exa_web_fetch_exa` / `agent-browser` |
| Primary sources (specs, filings, source code) | `webfetch`, `read`, `grep` |
| ML/AI papers with citation graphs | Semantic Scholar API (`pip install semanticscholar`) |
| Recent preprints | arXiv API (`pip install arxiv`) |
| DOI lookup + BibTeX | CrossRef |

**Answer engines (Perplexity, Grok) are aggregators** — chase their citations
and cite *those*, never the engine.

## Source-quality card (paste into researcher/subagent briefs)

- Primary sources only: official docs, source code, specs, first-party APIs,
  filings, original papers. Follow every claim to the source that owns it.
- A claim is a sentence with a source: publisher, pub date, access date.
- Red flags that downgrade confidence: speculative language, marketing
  register, cherry-picked/unsourced numbers, aggregators recycling one
  upstream report.
- Never conclude from training data alone. What you know proposes queries;
  evidence retrieved this run is what you report.
