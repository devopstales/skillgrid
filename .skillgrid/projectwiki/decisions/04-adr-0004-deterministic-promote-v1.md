# v1 generate is the deterministic promote floor; LLM Synthesizer deferred

---
status: "accepted"
date: 2026-11-24
supersedes: ADR-0003
---

## Context and Problem Statement

ADR-0003 committed v1's `projectwiki generate` to the LLM Synthesizer (user answer "1+3"). When the provider-scope options were examined (OpenAI-compatible only vs GitNexus-style matrix vs local agent CLI), the user dropped the Synthesizer first: v1 keeps v2's deterministic promote floor as the only generate path, with a page-protection rule added.

## Considered Options

- Keep ADR-0003: ship the LLM Synthesizer in v1 under one of the provider scopes
- Drop the Synthesizer first: v1 generate = deterministic promote floor + page protection; Synthesizer returns in a later phase with its provider scope decided then
- LLM-only generate with degraded no-op (v2's floor removed)

## Decision Outcome

Chosen option: "Drop the Synthesizer first" — v1 `generate` runs the deterministic floor (promote qualifying observations → cited-entry drafts; importance ≥ 0.5 + type map) under page protection: generate only writes pages flagged `origin=auto/draft`; curated pages are never touched (fixes v2's generate-pollutes-curation gotcha). The LLM Synthesizer and its provider scope become an explicit later-phase decision.

### Consequences

- Good, because v1 stays zero-network by default — no chat HTTP client or model config in the first phase
- Good, because the floor is v2-proven and the protection rule fixes the documented gotcha
- Bad, because generate output is cited-entry drafts, not prose synthesis, until the Synthesizer lands
