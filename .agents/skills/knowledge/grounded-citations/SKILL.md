---
name: grounded-citations
description: "Ground answers and documents in cited, verifiable sources: a ledger script owns the url-to-id mapping, prose cites ids, verify gates delivery. Use when writing research, comparisons, or any deliverable resting on fetched facts."
license: MIT
metadata:
  author: Hermes Agent + Teknium (ported; see ADR-0015)
  version: "1.0"
  part-of: skillgrid
---

# Grounded Citations

**Announce at start:** "I'm using the skillgrid:grounded-citations skill to keep this work cited."

## Overview

Every claim taken from an outside source gets an inline numbered citation plus a `Sources:` list, Perplexity-style. A ledger script (`scripts/sources.py`, stdlib-only) owns the `url → [n]` mapping so numbers and URLs come from retrieval, never from memory — the model only ever emits small integers it was handed. For high-stakes work the same ledger doubles as a fact-checking chain: verbatim quotes attached per source (rejected unless literally present in the fetched text), model-knowledge claims flagged `[unverified]`, `verify --evidence` failing drafts whose sources carry no evidence.

Covers chat answers, markdown documents, research reports, and wiki pages (`skillgrid:llm-wiki` output cites this way). Feed for `skillgrid:research` / `skillgrid:code-research` / `skillgrid:deep-research` findings — the `## Research:` sections those skills append are where this discipline pays off.

Ported from the Hermes `grounded-citations` skill under MIT (see ADR-0015). Behavioral deltas from upstream: ledger default rehomed to `.skillgrid/sdd/citations/ledger.json` (gitignored scratch; override with `--ledger` or `SKILLGRID_CITATION_LEDGER`); the multi-platform sweep table remapped from Hermes-only skills to skillgrid retrieval surfaces (below); parallel-subagent ledger sharing kept via `--ledger`.

## When to Use

- Research, comparisons, "current state of X", news summaries
- Any deliverable written to disk that quotes, paraphrases, or reports outside facts
- Fact-finding the user will want to check; multi-source synthesis with conflicting sources

**When NOT to use:** Incidental retrieval mid-coding (a syntax/version lookup) — mention a URL only if the user would plausibly want it. Creative writing, casual conversation. Source comments inside generated code (citations belong in prose deliverables and doc headers, not code).

## Prerequisites

Python 3, stdlib only. Script:

```bash
S=.agents/skills/knowledge/grounded-citations/scripts/sources.py
```

Ledger location: `.skillgrid/sdd/citations/ledger.json` (default; gitignored). Override per task with `--ledger <path>` or `SKILLGRID_CITATION_LEDGER`. Parallel subagents merging into one deliverable share one ledger via `--ledger` or the env var — otherwise their ids collide.

## How to Run

```bash
python "$S" reset                                  # start a clean ledger
python "$S" add https://example.com/a --title "A"  # prints: [1]
python "$S" add https://example.com/b --title "B"  # prints: [2]
python "$S" list                                   # ledger table
python "$S" render                                 # Sources: block
python "$S" verify draft.md                        # catch bad citations
```

`add` is idempotent and URL-normalized (fragment + trailing slash stripped; query kept): the same page always returns the same id, stable across many retrieval rounds.

## Procedure

1. **Reset** at task start (skip when continuing work whose ids live in a draft — reuse keeps numbering stable).
2. **Register every source at retrieval time**, before writing prose — from tool output, never reconstructed from the draft.
3. **Cite while drafting:** bracketed id(s) tight after each supported sentence (`…water.[1][2]`), max 3 ids per sentence, only ledger-issued ids, own knowledge uncited. Conflicting sources: both readings, each with its id.
4. **Append Sources** via `render --cited-in <draft>` (mechanical mapping, never retyped). Non-markdown placement per `references/citation-formats.md`.
5. **Verify before delivering:** `verify <draft>` fails on unknown ids, ledger-disagreeing Sources blocks, or (with `--min-coverage`) thinly cited prose. Fix and re-run.
6. **Chat answers:** same steps with the draft in the reply; short answers may `render --only <ids>`.

## Retrieval sweep (skillgrid surfaces)

"Research X across the web" is not one search. Fan out, register every URL as it arrives, then synthesize with each claim attributed to the platform it came from:

| Source type | Route | What it adds |
|---|---|---|
| Docs / papers / articles | Context7 query-docs, Exa search + fetch | official docs, announcements |
| Web research cache | mnemonic `webcache` (fresh rows) | previously vetted findings |
| Code | `skillgrid search`, `gh search repos/issues` | implementations, open bugs |
| Video | `skillgrid:youtube-transcript` | walkthroughs, demos, talks |
| Scholarly | arXiv via Exa/fetch | papers, prior art |

Pair opinion with primary sources (a thread reporting X is evidence users *report* X, not that X is true); report coverage gaps instead of silently narrowing to what worked.

## Fact-checking mode (high-stakes)

Attach a verbatim quote per source (`quote <id> --text "…" --from page.txt` — rejected unless verbatim modulo whitespace/case/markup; copy-paste, never retype). Flag unsourceable load-bearing claims `[unverified]` (rare; a deliverable dominated by them needed more retrieval). Cross-check disputed facts against a second independent source. Gate with `verify --evidence --min-coverage 0.5` and `render --style evidence --replace-in report.md`.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I'll register sources after drafting" | Reconstruction from memory reintroduces hallucinated-URL risk — the exact failure this skill removes. Register at retrieval. |
| "I'll type the Sources block by hand" | Always `render`. A hand-typed URL is an unverified claim. |
| "The snippet supports the claim" | A search description supports only what it literally says. Extract the page, then cite the body. |
| "One citation at paragraph end covers it" | Cite per sentence, max 3 ids. End-dumps hide which source carries the load. |
| "Mostly [unverified] is fine" | It marks the rare unsourceable claim. A marker-dominated draft needed more retrieval. |

## Red Flags

- Draft cites an id never issued by the ledger (invented id)
- Hand-edited renumbering mid-task (ids are ledger identities — `reset` only between tasks)
- Evidence quotes from search snippets instead of extracted page text
- Parallel subagents on separate ledgers merged into one deliverable (colliding ids)
- Retrieval narrowed silently to the one source type that worked

## Verification

- [ ] Ledger reset at task start (or deliberately reused with reason)
- [ ] Every outside-fact sentence carries ledger-issued id(s) (≤ 3) or `[unverified]`
- [ ] Sources block rendered mechanically (`render --cited-in`), never typed
- [ ] `verify [--strict] [--min-coverage N] [--evidence]` exits 0 — read warnings even on green
- [ ] Ledger + draft share one identity per task (shared `--ledger` across subagents when merged)
