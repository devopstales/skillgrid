---
type: TechnicalDecision
status: in-force (conditional)
supersedes: none
date: 2026-11-02
---

# ADR-0045 — Code-index conditional ONNX dependency (768-dim embedder)


Change: `2026-10-31-mnemonic-v2-code-index` (spec zone: `adr.md` § ADR-0045)

## Context

D-CI8 makes the semantic store model-agnostic; the **default** embedder is the
deterministic hash-64 floor (no dep, cgo-free, always works). A real 768-dim embedder
(ONNX nomic-embed-code, or an OpenAI-compat provider) is an **opt-in** that activates
HNSW/vec0 when `dim == 768`. v1's ONNX embedder (`onnxer` v0.7.0) is not actually wired
(it falls through to hash); wiring it is the v1 follow-up.

## Decision

Add `github.com/benedoc-inc/onnxer v0.7.0` to `mnemonic-v2/go.mod` **only if** the ONNX
768-dim embedder is wired in this change (TICKET-08a's 768-dim path). If the 768-dim
path is instead served by an OpenAI-compat provider over the memory change's
`intelligence/embedder` seam (which needs no new dep), this ADR is recorded as
*not taken* and `onnxer` is not added.

## Alternatives

- **OpenAI-compat provider only (no `onnxer`)** — viable; the hash-64 default + an
  OpenAI-compat 768-dim embedder needs no new dep.
- **ONNX local (nomic, cgo-free via `onnxer`)** — adds the dep but is local-first (no
  network at index time).

## Consequences

If taken, the 768-dim embedder activates HNSW/vec0 (D-CI8); if not taken, the semantic
leg is hash-64 brute-force by default + an opt-in OpenAI-compat 768-dim provider.
Either way the store is model-agnostic (the `embed_meta` model-swap guard, M9-dim-aware,
is lifted from v1). The dep lands at TICKET-08a's 768-dim path, gated by this ADR;
the decision is recorded in TICKET-08a's report.
