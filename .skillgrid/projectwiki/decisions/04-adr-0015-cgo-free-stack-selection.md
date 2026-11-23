# CGO-free stack selection: ncruces driver, vec ncruces binding, hugot GoMLX

---
status: accepted
date: 2026-11-23
supersedes: none
---

## Context and Problem Statement

[[decisions/0004-vectors-in-file]] and
[[decisions/0005-c-cgo-free-target-build]] fix the *shape* of the vector
layer (in-file, CGO-free, brute-force floor) but name no modules. The
briefing's named modules were partly wrong (`unum-cloud/onnx` 404s;
modernc cannot host vec0). Research
(`2026-11-23-mnemonic-core/findings.md`) verified the real candidates, and
the programmer's gate (2026-11-23, re-presented after a "what did the old
systems use?" check — both old generations ran **brute-force cosine over
BLOBs in a pure-Go driver**) locked the selection. This ADR records it. It
**refines** ADR-0004/0005/0006; it supersedes none. Both the `memory` tool
(`2026-11-23-mnemonic-core`) and the `code` tool
(`2026-11-23-mnemonic-code`) consume this decision.

## Considered Options

- **A: `ncruces/go-sqlite3` + in-ncruces vec extension + hugot GoMLX** —
  the vec0 candidate was the sqlite-vec project's official ncruces binding
  (8277★ project, Mozilla-sponsored, C core compiled to WASM); FTS5 verified
  in the driver; hugot's Go backend (pure-Go GoMLX) is explicitly designed
  for all-MiniLM-L6-v2. *(Spike 2026-11-23: the binding proved unpairable —
  see Decision Outcome; the working in-ncruces vec is the official
  `ext/vec1`.)*
- **B: modernc.org/sqlite + `viant/sqlite-vec`** — pure-Go virtual tables on
  the briefing's original driver; but 12★, built around
  dataset/shadow-table + MySQL SCN replication the product doesn't need,
  quickstart forces `SetMaxOpenConns(1)`, cover tree "experimental".
- **C: modernc.org/sqlite + plain BLOB tables + Go brute force** — exactly
  what both old generations shipped (v1: BLOB + `SearchByVector` cosine;
  v2: plain `vec_*` tables + in-memory HNSW). Proven, zero vec dependency,
  no ANN.
- **D: `glebarez/go-sqlite` first (887★, cznic lineage)** — virtual-table
  and FTS5 support unverified.

## Decision Outcome

Chosen option: **A** — the driver is `ncruces/go-sqlite3`
(`github.com/ncruces/go-sqlite3`, v0.38.0: WASM-built SQLite, CGO-free,
FTS5, virtual tables; the wasm binary lives in the companion module
`github.com/ncruces/go-sqlite3-wasm/v6`); the vector seam is ncruces'
official **`ext/vec1`** (SQLite's own ANN extension — IVFADC+OPQ,
L2/cosine, portable C, v0.8) behind build tag `sqlitevec`, registered
per-connection next to `ext/fts5`; the local embedder tier is
`github.com/knights-analytics/hugot` (665★, Apache-2.0) on its **GoMLX
pure-Go** backend behind build tag `onnx`, with `-tags ORT` (native
`libonnxruntime.so`) as the pre-decided speed escape and
`GOEXPERIMENT=simd` for GoMLX. Option C is retained as the **guaranteed
floor**: if the `sqlitevec` tag's build fails, brute force owns the vector
leg (hypothesis falsifier (b)).

### Consequences

- Good: real in-file ANN (vec1) is available in a CGO-free build from the
  driver's own official extension set — no third-party binding.
- Good: one bounded dependency set for two tools (memory + code) — the
  code tool's ADR-0013/0014 stack this on top unchanged.
- Bad: WASM per-connection memory overhead in ncruces (caveat documented in
  findings [4]); the store is per-project single-process in v1, so it is
  bounded — the `code` tool (larger files) re-checks it.
- Bad: **spike 2026-11-23 showed the sqlite-vec binding
  (`asg017/sqlite-vec-go-bindings/ncruces`) unpairable** — its embedded wasm
  matches no released ncruces host table (tried v0.17.1/v0.15.0/v0.38.0);
  hence the switch to the official `ext/vec1`. vec1 is v0.8 with an open
  roadmap; the brute-force floor + version pin contain the risk.
- Spike record (2026-11-23, both PASS, see `findings.md` § Spike Results):
  (1) MiniLM on GoMLX — dim 384, cross-backend cosine(GoMLX, ORT) =
  1.000000, p50 81 ms/sentence non-simd (target ≤ 3 s), semantic sanity
  0.584 vs 0.051; (2) `ext/vec1` + `ext/fts5` verified in one ncruces
  connection. Falsifiers (a)/(b) of the core blueprint not triggered;
  hugot v0.8.6 requires go ≥ 1.27 (toolchain auto-switch).

## Not adopted

- **B (viant)** — youngest of the real candidates; the sync machinery and
  single-connection limit would be live constraints in a two-writer
  (CLI + MCP) store.
- **C (plain BLOB only)** — proven, and it stays the floor; but at the
  `code` tool's 10⁴–10⁵ chunk scale brute force alone forfeits the speed
  target (T7), and the programmer chose the forward investment at the gate.
- **D (glebarez)** — the two capabilities this store requires (virtual
  tables, FTS5) are unverified in its lineage.
- **The asg017 vec0 binding (original candidate for option A)** — refuted
  by spike 2 (2026-11-23): unpairable with any released ncruces; would
  require pinning an untagged ncruces commit or waiting on a binding
  release. Revisit if sqlite-vec's vec0 features (e.g. `index=auto`
  selection) ever outweigh vec1's.

## Related

- Refines: [[decisions/0004-vectors-in-file]] (the vec0 mechanism),
  [[decisions/0005-c-cgo-free-target-build]] (the CGO-free driver/runtime),
  [[decisions/0006-local-onnx-default-embedder]] (the local-model runtime).
- Supersedes: none.
- Consumed by: [[syntheses/mnemonic-core]] (blueprint Key Decisions,
  Task 2/8/9), the `2026-11-23-mnemonic-code` briefing (inherited stack),
  [[concepts/in-file-vector-layer]].
