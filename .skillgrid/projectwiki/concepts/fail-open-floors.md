---
type: Concept
title: "Fail-open floors"
status: stable
sources:
  - "code:mnemonic/internal/intelligence/degrade/degrade.go"
---

# Fail-open floors

Every LLM, vector, and policy seam in the engine has a deterministic fallback
that keeps the request succeeding when the seam is absent or errors. An agent must
never hard-crash because a model or an indexer is down.

## Where it shows up

- LLM seams (`dedup.llm`, `extraction.llm`, `mem_ask` LLM mode) fall back to
  deterministic floors on error/timeout.
- Vector leg degrades to BM25-only when no embedder is active (`degraded: true`).
- Pre-tool policy fails open when the server is unreachable.
- `projectwiki generate` falls back to the append-only floor when no
  `Synthesizer` is wired.

## Rule of thumb

A seam is fail-open when its absence only degrades quality, never blocks the
operation. If a feature cannot tolerate degradation, it is not a seam — it is a
hard dependency and needs its own ADR.

## Related

- [[decisions/04-adr-0016-second-brain-capability-layer]] — fail-open floors on mem_ask/consolidate
- [[decisions/04-adr-0021-pre-tool-policy-fail-open]] — policy fails open on server outage
- [[architecture/01-module-map]] — where the seams live
