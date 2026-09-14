---
name: research
# based on mattpocock-skills:research + bmad:bmad-deep-recon (epistemics, type packs)
description: Investigate a question against primary sources and capture the findings as a cited Markdown file. Use when the design or plan depends on a fact not in the codebase — a library's current API, a version's behavior, a domain constraint, a competitor's offering. The lightweight, single-pass research; for wide or high-stakes questions use skillgrid:deep-research.
---

# Research

Answer a question by investigating **primary sources** and writing the findings
to a cited Markdown file. One pass, done inline — no subagents, no rounds. For a
question that is wide (multiple independent dimensions) or high-stakes, use
`skillgrid:deep-research` instead, which fans out parallel researchers and adds a
verify + red-team layer.

**Announce at start:** "I'm using the skillgrid:research skill to investigate
this before the design."

## Config

Read `.skillgrid/config.yaml` before starting. Use `conventions.specs_root`
(default `.skillgrid/specs`) for the findings location. If the `mnemonic` block
is enabled, check `web_cache_lookup` for the query before fetching and
`web_cache_save` after, so repeated research is deduplicated.

## Epistemics (two standing rules)

1. **Never conclude from training data alone.** What you already know proposes
   hypotheses, queries, and structure — it is not evidence. Conclusions require
   evidence retrieved *this run*. A claim you cannot evidence is stated as
   `unverified` or dropped, never asserted.

2. **The research firewall.** Project context — briefs, specs, code, memory,
   the glossary — shapes *what to ask*, never *what is true*. It is inadmissible
   as evidence: every claim in the findings file traces to a source you fetched.
   The firewall is why a fact your codebase *assumes* still gets re-verified
   against the source that owns it.

## Sourcing rules

- **Primary sources only.** Official documentation, source code, specs,
  first-party APIs, filings, original papers. Follow every claim back to the
  source that owns it — not a secondary write-up of that source.
- **Tooling:** `context7` for library/framework API docs (current, versioned);
  `exa` and `webfetch` for everything else. Prefer the source's own docs over a
  blog about them.
- **Red flags that downgrade confidence:** speculative language ("could",
  "may", projections presented as findings), marketing register, cherry-picked
  or unsourced numbers, aggregators recycling a single upstream report. Answer
  engines (Perplexity, Grok, and kin) are aggregators too — chase their
  citations and cite *those*, never the engine.
- **A claim is a sentence with a source** — publisher, publication date, access
  date. No naked numbers.
- **Report what is real.** Thin public data is reported as thin; absence of
  evidence is a finding; freshness is part of truth. A version's behavior from
  two releases ago is history, not fact.

## The Process

### Step 1: Frame the question and its decision

Take the question and name the **decision it serves** (adopt this library, scope
this feature, position against that competitor). The decision prunes the research
to what matters. If the decision is unclear, ask once, then proceed.

### Step 2: Pick the research type

Infer the type from the ask and load the matching type pack for its dimensions,
source craft, and freshness bars:

| Type | Pack | Use when |
|------|------|----------|
| technical | [types/technical.md](types/technical.md) | adopt a tech, design an integration, ground an architecture in current practice, assess feasibility |
| competitive | [types/competitive.md](types/competitive.md) | position against named competitors, build a battlecard, anticipate their next move |
| domain | [types/domain.md](types/domain.md) | learn a domain's rules, constraints, vocabulary before designing in it |

Prune the pack's dimensions to the decision — research only the dimensions that
answer it, not the whole pack.

### Step 3: Research inline

Work the dimensions in priority order. **Query craft:** short, wide queries
first to map what exists (roughly five words or fewer), then narrow as the shape
emerges — long hyper-specific queries return nothing. Broaden when results are
sparse, narrow when abundant. After every result, pause and evaluate: what did
this add, what gap remains, what's the best next query? Never repeat an identical
query on the same tool.

Apply the type pack's **source craft** (the non-obvious: read retrospectives not
launch threads, changelogs are roadmap truth, 1-3★ reviews are the wedge, etc.)
and its **freshness bars** (reject a source older than the bar for that claim
class; report staleness when nothing current exists).

### Step 4: Write the findings

Fill the scaffold at [templates/research.md](templates/research.md) and save it
to `{specs_root}/YYYY-MM-DD-<topic>/research.md`. Every load-bearing claim is
cited inline `[n]` and resolves in the source appendix. Flag confidence per
claim: **high** (verified, fresh, credible publisher), **medium** (single credible
source, fresh), **low** (stale, weak publisher, or disputed), or `unverified`.
Sections with nothing behind them collapse to a line rather than pad.

Commit the findings (`git add` + `git commit`) — the artifact is a checkpoint the
blueprint will cite.

### Step 5: Hand off

Report: the decision, the 2-3 findings that drive it, the biggest caveat, and the
path to `research.md`. The design (in `skillgrid:brainstorming`) or the blueprint
(`skillgrid:writing-blueprints`) reads the file; it does not reprocess the web.

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "I know the answer from training data" | That's a hypothesis, not evidence. Retrieve the source that owns the fact; cite it or mark it `unverified`. |
| "The codebase assumes X, so X is true" | The firewall: the codebase shapes the question, not the answer. Re-verify X against the source. |
| "This blog is detailed enough" | A blog is a secondary source. Follow it to the docs, source, or spec it's describing; cite the owner. |
| "The question is small, I'll skip the file" | The file is what the blueprint cites. Two findings still get the scaffold — the decision-relevant truth, in a place that survives the session. |
| "Wide question, I'll just do more inline" | When the question has multiple independent dimensions or is high-stakes, stop — that's `skillgrid:deep-research`, which parallelizes and verifies. |

## Red Flags

**Never:**
- Assert a claim you didn't retrieve this run
- Let project context stand in for evidence
- Cite an answer engine instead of its citation
- Ship a stale source when a current one exists without flagging the staleness
- Present thin data as if it were thick
