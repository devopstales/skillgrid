# ADR Review Manifest — 2026-11-23-mnemonic-code

- **Change:** `2026-11-23-mnemonic-code`
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
- `.skillgrid/projectwiki/decisions/0003-per-project-sqlite-file.md` —
  the code domain's tables live in the same per-project `.sqlite`;
  migration range 100-199.
- `.skillgrid/projectwiki/decisions/0004-vectors-in-file.md` — chunk + symbol
  vectors are in-file BLOBs with the vec layer; the brute-force BLOB floor
  stays the KNN equivalence oracle.
- `.skillgrid/projectwiki/decisions/0005-c-cgo-free-target-build.md` —
  constrains the AST choice (ADR-0013: pure-Go binding, no CGo build tag).
- `.skillgrid/projectwiki/decisions/0006-local-onnx-default-embedder.md` —
  the code domain reuses the memory domain's default embedder (hugot GoMLX,
  MiniLM 384-dim); ADR-0014 keeps them in one space.
- `.skillgrid/projectwiki/decisions/0007-single-handler-two-adapters.md` —
  all 7 `code_*` tools served by the K6 handler; CLI + MCP byte-identical
  envelopes (R11 parity).
- `.skillgrid/projectwiki/decisions/0009-wal-only-concurrency.md` — watch
  re-indexes and read-only MCP connections share WAL + `busy_timeout` only.
- `.skillgrid/projectwiki/decisions/0010-multi-tool-binary.md` —
  `mnemonic code <verb>` + the per-tool MCP server.
- `.skillgrid/projectwiki/decisions/0011-degradation-contract.md` —
  the code domain's degradation carrier (R6/R8): below-rich tier notes,
  computed coverage, never silent.
- `.skillgrid/projectwiki/decisions/04-adr-0015-cgo-free-stack-selection.md`
  (from `2026-11-23-mnemonic-core`, spike-amended 2026-11-23) — the verified
  stack this change inherits: hugot v0.8.6 GoMLX (MiniLM 384-dim),
  ncruces/go-sqlite3 v0.38.0 + `ext/vec1` + `ext/fts5`, go ≥ 1.27; the
  asg017 vec0 binding is not adopted.
- `.skillgrid/projectwiki/decisions/04-adr-0012-go-module-path.md` (from
  `2026-11-23-mnemonic-core`) — module path; the code domain adds no new
  module.

## New Durable ADRs Created

- `.skillgrid/projectwiki/decisions/0013-pure-go-treesitter.md` — pure-Go
  `odvcencio/gotreesitter` for all AST extraction; no CGo build tag; v1
  links only supported grammars; copyleft grammars excluded.
- `.skillgrid/projectwiki/decisions/0014-unified-embedding-space.md` — one
  embedding space per project file (same model + dim as memory); chunks and
  symbols share it (two BLOB tables, one space); no cross-domain fusion in
  v1.

## Supersessions

- None.
