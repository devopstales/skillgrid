# Degradation contract: `degraded: true` never silent, floors never block

---
status: accepted
date: 2026-11-23
supersedes: none
---

## Context and Problem Statement

[[entities/mnemonic]]'s default path depends on optional pieces: a model
file, a pure-Go ONNX runtime, the vec extension, (later) an LLM. Each can be
absent or failing at any moment. The ecosystem already runs on this
discipline — the installed `skillgrid:mnemonic` skill's degradation contract
("do NOT fail silently … emit a degradation line"), the projectwiki engine's
`degraded: true` runs, the predecessor's `degrade` package (one bounded
retry → floor). The new design must carry it as a machine-readable
contract, not a prose hope.

## Considered Options

- **Fail loudly** (error when the full path is unavailable): honest, but a
  missing model file would make *search* unavailable — the wrong failure
  surface for a memory tool.
- **Fail silently** (drop the weak leg, return what works): fast, but the
  consumer cannot tell that semantic recall is off.
- **Report the state**: the result always carries what ran — the full path
  or the floor — and every floor is a *reported* one.

## Decision Outcome

Chosen option: **the degradation contract**, two rules:

1. **Never silent** — any result produced without its full path carries
   `degraded: true` + a reason in the shared envelope (CLI and MCP, via the
   K6 seam — [[decisions/0007-single-handler-two-adapters]]).
   Recall without the vector leg is FTS-only and says so.
2. **Never blocking** — an embedder that errors gets one bounded retry,
   then the floor takes over; a save or search is never blocked on
   embeddings, an LLM, or an extension load. Floors: hash-64 vectors,
   brute-force scan, pure-Go build, WAL + busy_timeout
   ([[concepts/degradation-floor]]).

### Consequences

- Good: every optional piece (model, runtime, vec, LLM) is safe to be
  optional — its absence produces a slower/weaker result, never a dead
  tool, and the consumer always knows which.
- Good: `degraded` is testable — a parity/contract test asserts the flag
  iff and only if the leg was absent (briefing testing decisions).
- Good: the deferred LLM curator inherits the same contract — an LLM
  timeout never blocks a save or a search.
- Bad: consumers that ignore `degraded` get the silent-failure disease back
  — the agent-facing docs (and the future skill flip) must teach the flag.

## Not adopted

- **Per-leg error codes instead of a flag** — richer, but the flag + reason
  string is the ecosystem's existing vocabulary; inventing a second one is
  churn.
- **Automatic repair** (background model re-download on degraded) — a v1.0+
  nicety; v1 degrades and reports, repair is the user's `mnemonic init`.

## Related

- Supersedes: none.
- Depends on: [[concepts/degradation-floor]] (the floors the contract names),
  [[concepts/in-file-vector-layer]] (the leg that degrades),
  [[decisions/0007-single-handler-two-adapters]] (one envelope for
  both surfaces).
- Consumed by: [[syntheses/mnemonic-core]], [[architecture/mnemonic-
  memory-core]] (recall pipeline provenance).
