# ADR-0035: local Ollama judgment model (default `tev1:0.8b`), fail-open to pre-filter

---
status: "accepted"
supersedes: none
date: 2026-10-08
---

## Context

The CTX plan (v1.11) specifies a "Judgment Layer" (curator agent): a small,
fast, structured-output model that makes keep / truncate / compress-tier
decisions on context units. It NEVER deletes and NEVER rewrites prose
(deletion is a code-level decision; PrefixGuard). The plan named it
"Jev-class" without pinning the model or the endpoint. The local Ollama catalog
(verified, `.skillgrid/archive/2026-10-03-local-ollama-models/findings.md`)
offers `tev1:0.8b` (decision, 797MB), `clef-flash` (9B), `llama3.2:1b`, and
`embeddinggemma:300m`.

## Decision

The judgment model is a **local Ollama** model, default **`tev1:0.8b`** at
`http://127.0.0.1:11434/v1/systemone` (OpenAI-compatible `/v1`). Config:
`ctx.yaml` `judgment.model` (default `tev1:0.8b`) and `judgment.endpoint`
(default the local Ollama `/v1/systemone` URL). `clef-flash` is an opt-in
alternative (larger, more capable, slower). The judgment call is a fast
per-unit structured-output decision; results are **fingerprint-cached** and
**truncate-only** (threshold default 0.22, range 0.20–0.25). When the endpoint
is unavailable, the judgment layer **fails open to pre-filter-only** (the
deterministic pre-filter still runs; no judgment, no block).

## Consequences

- **Positive:** Fully local — no network dependency for the judgment path,
  consistent with the local-first guarantee. Reuses the existing local Ollama
  the index embedder already uses. The `tev1:0.8b` decision model is fast
  enough for per-unit calls; the fingerprint cache makes repeat units free.
- **Negative:** A 0.8B model is the weak judgment model available locally;
  `clef-flash` (9B) is the escape hatch when a higher-quality judgment is
  wanted and the latency is acceptable.
- **Migration:** none — additive `judgment:` config block; the fail-open floor
  means an absent/unreachable Ollama changes nothing about the deterministic
  floor.
