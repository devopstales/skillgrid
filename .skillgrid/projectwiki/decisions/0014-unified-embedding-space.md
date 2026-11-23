# One embedding space across tools (same model + dim per project)

---
status: accepted
date: 2026-11-23
supersedes: none
---

## Context and Problem Statement

The `code` tool embeds chunks and symbols; the `memory` tool embeds
observations. Both use the same per-project embedder tiers (ADR-0006: local
ONNX default, API opt-in, hash-64 floor) and the same per-project `.sqlite`
file (ADR-0003). The open question: do the tools share one vector space —
one active model + dim per project — or embed independently (each tool could
choose its own model/dim)?

## Considered Options

- **Unified space**: one active embedder per project; every vector row in the
  file is stamped with the same model + dim. Cross-domain similarity (a code
  chunk vs an observation) is well-defined, and cross-domain fusion becomes a
  later additive feature.
- **Independent spaces**: each tool could swap models/dims without touching
  the other; the cost is per-table dim locks + re-embed coordination inside
  one file, and cross-tool similarity is undefined.

## Decision Outcome

Chosen option: **unified space** — the project's active embedder (model +
dim) is single, so code and memory vectors are mutually comparable. A
model/dim swap is a whole-file re-embed (the dim-strict guard already
enforces it per table; now it is explicitly per file). v1 keeps the two
tools' search contracts **independent** — cross-domain fusion (memory hits
inside a code search, or code hits inside `mem_search`) is a later additive
feature, not a v1 behavior.

### Consequences

- Good: cross-domain recall becomes possible without re-architecture — the
  "search the repo and its recorded decisions together" story is one RRF
  fusion away.
- Good: one embedder config, one `doctor` report, one re-embed job per model
  swap.
- Bad: a model swap re-embeds observations as well as code (already true
  within memory; now explicit for code) — the re-embed is a background job,
  accepted.
- Bad: v1 readers may expect cross-tool hits; the no-fusion-in-v1 contract
  is stated in the change briefing.

## Not adopted

- **Independent spaces** — per-table model/dim freedom buys no v1 feature
  and costs the cross-domain story.

## Related

- Supersedes: none.
- Depends on: `decisions/0003-per-project-sqlite-file.md` (one file per
  project), `decisions/0006-local-onnx-default-embedder.md` (the tiers),
  `decisions/0011-degradation-contract.md` (floors never block).
- Consumed by: the code tool's embedding layer; any future cross-domain
  fusion.
