---
type: Concept
title: "Degradation floor (never silent, never blocking)"
status: stable
sources:
  - ".skillgrid/change-specs/2026-11-23-mnemonic-core/briefing.md R3 + ADR-9 (2026-11-23)"
  - "local: ../skillgrid/mnemonic/internal/intelligence/degrade/degrade.go (predecessor)"
---

# Degradation floor (never silent, never blocking)

Every capability in [[entities/mnemonic]] has a **deterministic,
dependency-free path** (the "floor") that keeps the feature working with zero
network, zero model files, and zero optional extensions. Degradation is a
reported state, not a failure and not a secret.

## The two rules

1. **Never silent** — when the full path is unavailable, the result carries
   `degraded: true` with a reason. Recall without the vector leg still runs
   (FTS5 BM25 only) and says so. An agent reading `degraded: true` knows the
   result may miss semantically-related hits.
2. **Never blocking** — an embedder that errors gets one bounded retry, then
   the floor takes over; a save or a search is **never** blocked on
   embeddings, an LLM, or an extension load.

## Floors in the v1 design

| Capability | Full path | Floor |
|---|---|---|
| Embeddings | local ONNX (default) / API (opt-in) | hash-64 (deterministic token hash, 64-dim) |
| Vector recall | vec0 KNN (build tag, spike-gated) | Go brute-force cosine over BLOBs |
| ONNX runtime | `knights-analytics/hugot` Go backend (GoMLX, pure Go — per findings 2026-11-23; the brainstorm's `unum-cloud/onnx` 404s) | hash floor in the lean build |
| Concurrency | WAL multi-connection | `busy_timeout=5000` — no leases, no queue |

The floor is what makes the opt-ins (model file, ONNX profile, vec extension)
safe to be optional: every one of them failing produces a slower or less
accurate mnemonic, never a dead one. The same discipline applies to the
deferred LLM curator — every LLM path degrades to the deterministic
equivalent on error.

## Related

- Supersedes: none.
- Consumed by: [[syntheses/mnemonic-core]], [[architecture/mnemonic-memory-core]].
- Depends on: [[decisions/0011-degradation-contract]] (the locked
  decision), [[concepts/in-file-vector-layer]] (the vector leg it governs),
  [[preferences/build-preferences]] (why the floors are pure-Go).
