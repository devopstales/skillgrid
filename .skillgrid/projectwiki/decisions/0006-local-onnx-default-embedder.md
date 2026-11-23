# Local ONNX CPU model is the default embedder tier

---
status: accepted
date: 2026-11-23
supersedes: none
---

## Context and Problem Statement

The predecessor ([[sources/skillgrid-mnemonic-v2]]) ships an opt-in 768-dim
API embedder with a hash-64 floor — the semantic leg is off unless a remote
API is configured. The programmer locked (2026-11-23): the **local ONNX
model is the floor/default**, must run on CPU, with the API opt-in. A local
Ollama deployment already exists in this ecosystem
(`.skillgrid/config.yaml`: `embeddinggemma:300m`, 768-dim,
`localhost:11434`).

## Considered Options

- **hash-64 only**: zero model files, but "semantic" recall degrades to a
  token-hash bag — the default path is the weak path.
- **Local ONNX model as default**: real semantic recall offline; a ~90 MB
  model file (download-on-init + cache); a runtime question
  ([[decisions/0005-c-cgo-free-target-build]]).
- **Ollama daemon as default**: fast, model already pulled, but the default
  path gains a daemon dependency.
- **Remote API as default**: network on the default path — violates
  local-first.

## Decision Outcome

Chosen option: **three tiers, local ONNX on top** — (1) `local`: a small
CPU ONNX sentence-embedding model, default; working candidate
`all-MiniLM-L6-v2` (384-dim), final pick by the v0.2 spike (CPU latency +
retrieval eval); model downloaded once at `init`, cached in
`~/.mnemonic/models/`. (2) `api` (opt-in): any OpenAI-compatible
`/embeddings` — explicitly Ollama-compatible
(`MNEMONIC_API_BASE=http://127.0.0.1:11434/v1`), else the floor takes over.
(3) `hash` (floor): 64-dim deterministic, auto-selected when the ONNX
runtime or model file is absent, and on `MNEMONIC_EMBEDDER=hash`.
Vectors are model-stamped and dim-strict; a save/search is never blocked on
embedding (one bounded retry → floor).

### Consequences

- Good: the default path is offline, local, and semantically real — the
  predecessor's inversion (API default → local default) is done.
- Good: the Ollama tier reuses the programmer's existing daemon/model for
  speed without making it required.
- Bad: first-run needs a model download (network once) — `--no-models` +
  hash floor cover offline first-run.
- Bad: a 384-dim model is a capability choice the spike must validate on
  *retrieval* quality, not just latency.

## Not adopted

- **hash-64 default** — a floor, not a default: it keeps every build
  functional but makes the default path the weak one.
- **Ollama default** — daemon dependency; kept as the opt-in fast path.

## Related

- Supersedes: none.
- Depends on: [[decisions/0005-c-cgo-free-target-build]] (the runtime),
  [[concepts/in-file-vector-layer]] (dim-strict, model-stamped storage),
  [[preferences/build-preferences]] (local model over daemon).
- Consumed by: [[syntheses/mnemonic-core]], [[entities/mnemonic]].
