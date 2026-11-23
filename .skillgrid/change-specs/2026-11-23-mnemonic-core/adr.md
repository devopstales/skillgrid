# ADR Review Manifest — 2026-11-23-mnemonic-core

- **Change:** `2026-11-23-mnemonic-core`
- **Status:** completed
- **Review date:** 2026-11-23

## In-Force ADRs Reviewed

- `.skillgrid/projectwiki/decisions/0001-adopt-intent-driven-template-patterns.md` —
  the `.agents/` shared layout + ADR records contract; this change's ADRs are
  authored from its `madr-minimal` template (per `.skillgrid/config.yaml`
  `adr_style`).
- `.skillgrid/projectwiki/decisions/0002-git-discipline-as-separate-skill.md` —
  phase-transition git gates; constrains how this change moves through
  phases (no design impact).

## New Durable ADRs Created

- `.skillgrid/projectwiki/decisions/0003-per-project-sqlite-file.md` —
  one portable `.sqlite` per project under `~/.mnemonic/`.
- `.skillgrid/projectwiki/decisions/0004-vectors-in-file.md` —
  vectors inside the SQLite file: BLOB source of truth + vec0 (pure-Go,
  spike-gated) + brute-force floor; no external vector DB.
- `.skillgrid/projectwiki/decisions/0005-c-cgo-free-target-build.md` —
  C/CGo-free target build (pure-Go ONNX, spike-gated); CGo `onnxruntime` is
  the pre-decided fallback.
- `.skillgrid/projectwiki/decisions/0006-local-onnx-default-embedder.md` —
  local ONNX CPU model is the default embedder tier; API (Ollama-compatible)
  opt-in; hash-64 floor.
- `.skillgrid/projectwiki/decisions/0007-single-handler-two-adapters.md` —
  CLI + MCP over one Handler; byte-identical envelopes (K6 parity).
- `.skillgrid/projectwiki/decisions/0008-memory-core-only-v1.md` —
  v1 = memory core only; the deferred list is named.
- `.skillgrid/projectwiki/decisions/0009-wal-only-concurrency.md` —
  WAL + `busy_timeout` only; no leases in v1.
- `.skillgrid/projectwiki/decisions/0010-multi-tool-binary.md` —
  `mnemonic <tool> <verb>`; per-tool MCP servers; top-level `init`/`doctor`.
- `.skillgrid/projectwiki/decisions/0011-degradation-contract.md` —
  `degraded: true` never silent; floors never block save/search.

## Supersessions

- None.
