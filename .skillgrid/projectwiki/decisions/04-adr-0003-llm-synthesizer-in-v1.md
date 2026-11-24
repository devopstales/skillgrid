# v1 generation ships the LLM Synthesizer

---
status: "accepted"
date: 2026-11-24
supersedes: none
---

## Context and Problem Statement

Scope of `projectwiki generate` in v1. v2's Synthesizer seam is wired nil: generate runs only the deterministic cited-entry floor, with a known gotcha that running generate on a curated vault pollutes it. v1 options: (1) index/serve/validate/cite only; (1+2) plus the deterministic promote floor (v2 behavior); (1+2+3) plus the LLM Synthesizer.

## Considered Options

- (1) Index/serve/validate/cite only — no generate
- (1+2) Plus deterministic promote floor
- (1+2+3) Plus LLM Synthesizer

## Decision Outcome

Chosen option: "1+3" (user-confirmed): the full index/serve/validate/cite surface, and `generate` with the LLM Synthesizer. The deterministic promote path's role as an LLM-absent fallback is a separate open decision (round 2).

### Consequences

- Good, because it closes v2's known gap (nil Synthesizer) — generate becomes genuinely useful
- Good, because local-first stays: the LLM is optional and external; the engine degrades without it
- Bad, because v1 gains provider plumbing (chat HTTP client, model config) in an otherwise zero-network-by-default engine
