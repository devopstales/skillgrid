# C/CGo-free is the target build (pure-Go ONNX, spike-gated)

---
status: accepted
date: 2026-11-23
supersedes: none
---

## Context and Problem Statement

[[entities/mnemonic]] wants ONNX embedding as the default tier. The official
`onnxruntime` Go bindings wrap a C/C++ core (CGo): a single static binary,
but a C toolchain at build time. The programmer asked (2026-11-23): "will
ONNX be pure go so c and cgo free?" A pure-Go ONNX runtime exists
(`unum-cloud/onnx`) with a **partial opset** and no SIMD matmul — viability
is an empirical question (opset coverage + CPU latency of the chosen model).

## Considered Options

- **CGo `onnxruntime` as default**: full opset, SIMD, ~50–100 ms/sentence;
  build needs a C toolchain; the "pure Go, zero deps" story is false.
- **Pure-Go `unum-cloud/onnx` as default**: fully C/CGo-free static binary;
  expected ~0.5–2 s/sentence (no SIMD); the chosen model must fit its opset.
- **No ONNX; hash floor only**: simplest, but the default path loses real
  semantic recall — against the programmer's ONNX-floor decision.

## Decision Outcome

Chosen option: **C/CGo-free is the target default** —
`modernc.org/sqlite` + `unum-cloud/onnx`, one static binary, no C
toolchain. The v0.2 spike is the gate: load the chosen model, verify opset
coverage, bench latency. **Escape hatch:** if the spike fails (missing op or
>~3 s/sentence), the default flips to CGo `onnxruntime`; a lean pure-Go
profile stays functional via the hash-64 floor either way. CI covers
whichever profiles exist.

### Consequences

- Good: the default binary is portable and buildable anywhere Go builds;
  no `.so` hunting, no CGO_ENABLED footguns.
- Good: the escape hatch is pre-decided, so the spike outcome changes a
  build default, not the design.
- Bad: pure-Go matmul is ~10× the C core — per-save embed ~0.5–2 s
  (tolerable); bulk re-embed of 10k (~1–2 h) is a background job by design.
- Bad: a pure-Go ONNX runtime is a less battle-tested dependency than
  onnxruntime — the spike's opset check is the risk owner.

## Not adopted

- **CGo default** — the C toolchain requirement and the false zero-deps story
  cost more than the latency gain at v1 scale; it remains the pre-decided
  fallback.
- **Ollama as the default local tier** — fast and C-for-the-binary, but a
  daemon dependency for the default path; it stays an opt-in `api` tier
  (Ollama-compatible). See [[preferences/build-preferences]].

## Related

- Supersedes: none.
- Depends on: [[concepts/degradation-floor]] (the hash floor that keeps the
  lean profile functional), [[decisions/0006-local-onnx-default-embedder]]
  (what the runtime runs).
- Consumed by: [[syntheses/mnemonic-core]], [[entities/mnemonic]].
